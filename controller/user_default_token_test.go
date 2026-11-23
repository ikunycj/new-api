package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRegisterUsesConfiguredDefaultAPITokenGroupPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run PostgreSQL default API token registration tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { assert.NoError(t, tx.Rollback().Error) })
	schema := fmt.Sprintf("default_api_token_group_%d", time.Now().UnixNano())
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	require.NoError(t, tx.AutoMigrate(&model.User{}, &model.Token{}, &model.Option{}, &model.Log{}, &model.AffiliateQuotaAccount{}))

	previousDB, previousLogDB, previousRedis := model.DB, model.LOG_DB, common.RedisEnabled
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	previousGenerate := constant.GenerateDefaultToken
	previousRegister, previousPasswordRegister := common.RegisterEnabled, common.PasswordRegisterEnabled
	previousEmailVerification, previousQuota := common.EmailVerificationEnabled, common.QuotaForNewUser
	previousPermissions := setting.UserGroupPricingGroups2JSONString()
	previousPricing, err := ratio_setting.ParsePersistedPricingGroupConfiguration(
		ratio_setting.GroupRatio2JSONString(), ratio_setting.PricingGroupEnabled2JSONString(),
		ratio_setting.PricingGroupOrder2JSONString(), ratio_setting.PricingGroupRetryPolicy2JSONString(),
		ratio_setting.PricingGroupRoutingStrategy2JSONString(), ratio_setting.PricingGroupRemark2JSONString(),
		ratio_setting.PricingGroupDisplayName2JSONString(), ratio_setting.PricingGroupModelAllowlist2JSONString(),
	)
	require.NoError(t, err)
	common.OptionMapRWMutex.Lock()
	previousOptions := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.RedisEnabled = previousDB, previousLogDB, previousRedis
		common.SetDatabaseTypes(previousMain, previousLog)
		constant.GenerateDefaultToken = previousGenerate
		common.RegisterEnabled, common.PasswordRegisterEnabled = previousRegister, previousPasswordRegister
		common.EmailVerificationEnabled, common.QuotaForNewUser = previousEmailVerification, previousQuota
		ratio_setting.ApplyPricingGroupConfiguration(previousPricing)
		require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(previousPermissions))
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptions
		common.OptionMapRWMutex.Unlock()
	})

	model.DB, model.LOG_DB, common.RedisEnabled = tx, tx, false
	common.SetDatabaseTypes(common.DatabaseTypePostgreSQL, common.DatabaseTypePostgreSQL)
	constant.GenerateDefaultToken = true
	common.RegisterEnabled, common.PasswordRegisterEnabled = true, true
	common.EmailVerificationEnabled, common.QuotaForNewUser = false, 0
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"starter":0.5,"disabled":1,"restricted":2,"[starter]":1}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(`{"default":true,"starter":true,"disabled":false,"restricted":true,"[starter]":true}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupOrderByJSONString(`["starter","restricted","disabled","default","[starter]"]`))
	require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(`{"default":["default","starter","disabled","[starter]"]}`))

	router := gin.New()
	router.GET("/api/group/pricing-groups", GetPricingGroups)
	router.PUT("/api/option/", UpdateOption)
	router.POST("/api/user/register", Register)

	catalogResponse := httptest.NewRecorder()
	router.ServeHTTP(catalogResponse, httptest.NewRequest(http.MethodGet, "/api/group/pricing-groups", nil))
	require.Equal(t, http.StatusOK, catalogResponse.Code)
	var catalog struct {
		Success             bool     `json:"success"`
		AvailableToNewUsers []string `json:"available_to_new_users"`
	}
	require.NoError(t, common.Unmarshal(catalogResponse.Body.Bytes(), &catalog))
	require.True(t, catalog.Success)
	assert.Equal(t, []string{"starter", "default", "[starter]"}, catalog.AvailableToNewUsers)

	setDefaultGroup := func(value any, wantSuccess bool) {
		t.Helper()
		body, err := common.Marshal(OptionUpdateRequest{Key: common.DefaultAPITokenGroupOptionKey, Value: value})
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/option/", bytes.NewReader(body)))
		require.Equal(t, http.StatusOK, w.Code)
		var result struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		require.NoError(t, common.Unmarshal(w.Body.Bytes(), &result))
		require.Equal(t, wantSuccess, result.Success, result.Message)
	}
	registerUser := func(username, expectedGroup string) model.Token {
		t.Helper()
		body, err := common.Marshal(map[string]string{"username": username, "password": "password123"})
		require.NoError(t, err)
		w := httptest.NewRecorder()
		// Register from a non-affiliate host so this fixture exercises ordinary signup.
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/user/register", bytes.NewReader(body)))
		require.Equal(t, http.StatusOK, w.Code)
		var result struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		require.NoError(t, common.Unmarshal(w.Body.Bytes(), &result))
		require.True(t, result.Success, result.Message)

		var user model.User
		require.NoError(t, tx.Where("username = ?", username).First(&user).Error)
		assert.Equal(t, model.DefaultUserGroup, user.Group)
		var token model.Token
		require.NoError(t, tx.Where("user_id = ?", user.Id).First(&token).Error)
		assert.Equal(t, expectedGroup, token.Group)
		return token
	}

	setDefaultGroup("starter", true)
	setDefaultGroup("restricted", false)
	setDefaultGroup("disabled", false)
	setDefaultGroup(123, false)
	setDefaultGroup(`["default","restricted"]`, false)
	setDefaultGroup(`["default","starter","starter"]`, false)
	setDefaultGroup(`["default","disabled"]`, false)
	setDefaultGroup(`["default","auto"]`, false)
	setDefaultGroup(`["default","*"]`, false)
	setDefaultGroup(`["default",7]`, false)
	setDefaultGroup(`["default"`, false)
	var option model.Option
	require.NoError(t, tx.First(&option, &model.Option{Key: common.DefaultAPITokenGroupOptionKey}).Error)
	assert.Equal(t, "starter", option.Value)
	configuredToken := registerUser("starter-user", "starter")
	assert.Empty(t, configuredToken.GroupCandidates)
	assert.False(t, configuredToken.CrossGroupRetry)
	setDefaultGroup("[starter]", true)
	registerUser("literal-group-user", "[starter]")
	setDefaultGroup(`["starter"]`, true)
	singleToken := registerUser("single-group-user", "starter")
	assert.Empty(t, singleToken.GroupCandidates)
	assert.False(t, singleToken.CrossGroupRetry)
	setDefaultGroup(`["default","starter"]`, true)
	multipleToken := registerUser("multiple-groups-user", "auto")
	groups, err := multipleToken.GetGroupCandidates()
	require.NoError(t, err)
	assert.Equal(t, []string{"default", "starter"}, groups)
	assert.True(t, multipleToken.CrossGroupRetry)

	require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(`{"default":true,"starter":false,"disabled":false,"restricted":true,"[starter]":true}`))
	registerUser("fallback-user", "")
	setDefaultGroup("", true)
	registerUser("inherited-user", "")
	setDefaultGroup(`[]`, true)
	registerUser("empty-array-user", "")

	require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(`{"default":["*"]}`))
	setDefaultGroup(`["restricted","default"]`, true)
	wildcardToken := registerUser("wildcard-group-user", "auto")
	groups, err = wildcardToken.GetGroupCandidates()
	require.NoError(t, err)
	assert.Equal(t, []string{"restricted", "default"}, groups)

	require.NoError(t, tx.First(&configuredToken, configuredToken.Id).Error)
	assert.Equal(t, "starter", configuredToken.Group)
}
