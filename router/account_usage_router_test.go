package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAccountUsageCredentialPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run account usage authorization tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, pool.Close()) })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { assert.NoError(t, tx.Rollback().Error) })
	schema := fmt.Sprintf("account_usage_test_%d", time.Now().UnixNano())
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	require.NoError(t, tx.AutoMigrate(&model.User{}, &model.Token{}))

	previousDB := model.DB
	previousRedis := common.RedisEnabled
	previousDisplay := *operation_setting.GetGeneralSetting()
	previousRate := operation_setting.USDExchangeRate
	model.DB = tx
	common.RedisEnabled = false
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.USDExchangeRate = 6.5
	t.Cleanup(func() {
		model.DB = previousDB
		common.RedisEnabled = previousRedis
		*operation_setting.GetGeneralSetting() = previousDisplay
		operation_setting.USDExchangeRate = previousRate
	})

	accountToken := "test-dashboard-access-token"
	owner := model.User{Username: "owner", AffCode: "owner", Quota: int(common.QuotaPerUnit) * 3 / 2, UsedQuota: int(common.QuotaPerUnit) / 2, AccessToken: &accountToken}
	otherUser := model.User{Username: "other", AffCode: "other"}
	require.NoError(t, tx.Create(&owner).Error)
	require.NoError(t, tx.Create(&otherUser).Error)
	key := model.Token{UserId: owner.Id, Name: "provider key", Key: "test-provider-key", UnlimitedQuota: true, RemainQuota: 50, UsedQuota: 100}
	require.NoError(t, tx.Create(&key).Error)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("account-usage-test"))))
	engine.GET("/test/sign-in/:id", func(c *gin.Context) {
		user := owner
		if c.Param("id") == strconv.Itoa(otherUser.Id) {
			user = otherUser
		}
		session := sessions.Default(c)
		session.Set("username", user.Username)
		session.Set("role", user.Role)
		session.Set("id", user.Id)
		session.Set("status", user.Status)
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(engine)

	request := func(method, path string, cookie *http.Cookie, bearer string, userID int) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		if userID > 0 {
			r.Header.Set("New-Api-User", strconv.Itoa(userID))
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, r)
		return w
	}
	ownerCookies := request(http.MethodGet, fmt.Sprintf("/test/sign-in/%d", owner.Id), nil, "", 0).Result().Cookies()
	otherCookies := request(http.MethodGet, fmt.Sprintf("/test/sign-in/%d", otherUser.Id), nil, "", 0).Result().Cookies()
	require.NotEmpty(t, ownerCookies)
	require.NotEmpty(t, otherCookies)
	issuePath := fmt.Sprintf("/api/token/%d/usage-authorization", key.Id)
	const usagePath = "/api/usage/account/"
	issue := request(http.MethodPost, issuePath, ownerCookies[0], "", owner.Id)
	require.Equal(t, http.StatusOK, issue.Code, issue.Body.String())
	assert.Contains(t, issue.Header().Get("Cache-Control"), "no-store")
	var issued struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
			UserID      int    `json:"user_id"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(issue.Body.Bytes(), &issued))
	require.True(t, issued.Success)
	assert.Equal(t, owner.Id, issued.Data.UserID)
	credential := issued.Data.AccessToken
	require.NotEmpty(t, credential)
	assert.NotContains(t, credential, key.Key)
	assert.NotContains(t, credential, accountToken)
	assert.Equal(t, http.StatusNotFound, request(http.MethodPost, issuePath, otherCookies[0], "", otherUser.Id).Code)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodPost, issuePath, nil, accountToken, owner.Id).Code)

	for _, tt := range []struct {
		display   string
		unit      string
		remaining float64
		used      float64
	}{
		{operation_setting.QuotaDisplayTypeUSD, "USD", 1.5, 0.5},
		{operation_setting.QuotaDisplayTypeCNY, "CNY", 9.75, 3.25},
		{operation_setting.QuotaDisplayTypeTokens, "Tokens", float64(owner.Quota), float64(owner.UsedQuota)},
		{operation_setting.QuotaDisplayTypeCustom, "¤", 3.0, 1.0},
	} {
		t.Run(tt.display, func(t *testing.T) {
			setting := operation_setting.GetGeneralSetting()
			setting.QuotaDisplayType = tt.display
			setting.CustomCurrencyExchangeRate = 2
			setting.CustomCurrencySymbol = "¤"
			w := request(http.MethodGet, usagePath, nil, credential, owner.Id)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Contains(t, w.Header().Get("Cache-Control"), "no-store")
			var response struct {
				Success bool `json:"success"`
				Data    struct {
					Remaining float64 `json:"remaining"`
					Used      float64 `json:"used"`
					Unit      string  `json:"unit"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
			require.True(t, response.Success)
			assert.Equal(t, tt.unit, response.Data.Unit)
			assert.InDelta(t, tt.remaining, response.Data.Remaining, 1e-8)
			assert.InDelta(t, tt.used, response.Data.Used, 1e-8)
		})
	}

	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, usagePath, nil, credential, otherUser.Id).Code)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, usagePath, nil, credential, 0).Code)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, usagePath, nil, "sk-"+key.Key, owner.Id).Code)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, usagePath, nil, accountToken, owner.Id).Code)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, usagePath, nil, credential+"x", owner.Id).Code)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, "/api/usage/token/", nil, credential, owner.Id).Code)
	dashboard := request(http.MethodGet, "/api/token/", nil, credential, owner.Id)
	var accessCheck struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(dashboard.Body.Bytes(), &accessCheck))
	assert.False(t, accessCheck.Success)

	require.NoError(t, tx.Model(&key).Updates(map[string]any{"status": common.TokenStatusExpired, "expired_time": 1}).Error)
	assert.Equal(t, http.StatusOK, request(http.MethodGet, usagePath, nil, credential, owner.Id).Code)
	require.NoError(t, tx.Model(&key).Update("status", common.TokenStatusDisabled).Error)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, usagePath, nil, credential, owner.Id).Code)
	require.NoError(t, tx.Model(&key).Update("status", common.TokenStatusEnabled).Error)
	require.NoError(t, tx.Model(&owner).Update("status", common.UserStatusDisabled).Error)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, usagePath, nil, credential, owner.Id).Code)
	require.NoError(t, tx.Model(&owner).Update("status", common.UserStatusEnabled).Error)
	require.NoError(t, tx.Model(&key).Update("key", "rotated-provider-key").Error)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, usagePath, nil, credential, owner.Id).Code)
	rotatedIssue := request(http.MethodPost, issuePath, ownerCookies[0], "", owner.Id)
	require.Equal(t, http.StatusOK, rotatedIssue.Code, rotatedIssue.Body.String())
	require.NoError(t, common.Unmarshal(rotatedIssue.Body.Bytes(), &issued))
	require.True(t, issued.Success)
	require.NotEmpty(t, issued.Data.AccessToken)
	require.NotEqual(t, credential, issued.Data.AccessToken)
	require.NoError(t, tx.Delete(&key).Error)
	assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, usagePath, nil, issued.Data.AccessToken, owner.Id).Code)
}
