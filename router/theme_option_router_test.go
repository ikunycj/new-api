package router

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestThemeOptionRouteRequiresRoot(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run theme option route tests")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, pool.Close()) })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { assert.NoError(t, tx.Rollback().Error) })

	schema := fmt.Sprintf("theme_option_route_test_%d", time.Now().UnixNano())
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	require.NoError(t, tx.AutoMigrate(&model.Option{}, &model.User{}, &model.Log{}))

	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousRedis := common.RedisEnabled
	previousAPIRateLimit := common.GlobalApiRateLimitEnable
	previousTheme := model.GetSiteThemeCustomization()
	previousThemeJSON, err := common.Marshal(previousTheme)
	require.NoError(t, err)
	key := system_setting.SiteThemeCustomizationOptionKey
	model.DB, model.LOG_DB = tx, tx
	common.SetDatabaseTypes(common.DatabaseTypePostgreSQL, common.DatabaseTypePostgreSQL)
	common.OptionMapRWMutex.Lock()
	previousOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	common.RedisEnabled = false
	common.GlobalApiRateLimitEnable = false

	t.Cleanup(func() {
		assert.NoError(t, model.UpdateOption(key, string(previousThemeJSON)))
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
		common.RedisEnabled = previousRedis
		common.GlobalApiRateLimitEnable = previousAPIRateLimit
	})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("theme-option-route-test"))))
	engine.GET("/test/sign-in/:role", func(c *gin.Context) {
		role, err := strconv.Atoi(c.Param("role"))
		require.NoError(t, err)
		session := sessions.Default(c)
		session.Set("username", "theme-test-user")
		session.Set("role", role)
		session.Set("id", 42)
		session.Set("status", common.UserStatusEnabled)
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(engine)

	const themeJSON = `{"preset":"ikun-love","mode":"dark","sidebar":"floating","layout":"offcanvas","direction":"rtl"}`
	body, err := common.Marshal(map[string]string{"key": key, "value": themeJSON})
	require.NoError(t, err)
	request := func(t *testing.T, role int) (int, bool) {
		t.Helper()
		login := httptest.NewRecorder()
		engine.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/test/sign-in/"+strconv.Itoa(role), nil))
		require.Equal(t, http.StatusNoContent, login.Code)
		cookies := login.Result().Cookies()
		require.Len(t, cookies, 1)

		req := httptest.NewRequest(http.MethodPut, "/api/option/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("New-Api-User", "42")
		req.AddCookie(cookies[0])
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)
		var payload struct {
			Success *bool `json:"success"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
		require.NotNil(t, payload.Success)
		return response.Code, *payload.Success
	}

	for _, test := range []struct {
		name string
		role int
	}{
		{name: "common user", role: common.RoleCommonUser},
		{name: "administrator", role: common.RoleAdminUser},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, success := request(t, test.role)
			assert.Equal(t, http.StatusOK, status)
			assert.False(t, success)
		})
	}
	var count int64
	require.NoError(t, tx.Model(&model.Option{}).Where(&model.Option{Key: key}).Count(&count).Error)
	require.Zero(t, count)

	status, success := request(t, common.RoleRootUser)
	require.Equal(t, http.StatusOK, status)
	require.True(t, success)
	var option model.Option
	require.NoError(t, tx.Where(&model.Option{Key: key}).First(&option).Error)
	assert.JSONEq(t, themeJSON, option.Value)

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	require.Equal(t, http.StatusOK, response.Code)
	var publicStatus struct {
		Data struct {
			Theme system_setting.SiteThemeCustomization `json:"site_theme_customization"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &publicStatus))
	expectedTheme := system_setting.DefaultSiteThemeCustomization()
	expectedTheme.Preset = "ikun-love"
	expectedTheme.Mode = "dark"
	expectedTheme.Sidebar = "floating"
	expectedTheme.Layout = "offcanvas"
	expectedTheme.Direction = "rtl"
	assert.Equal(t, expectedTheme, publicStatus.Data.Theme)
}
