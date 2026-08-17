package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type onboardingResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Required bool `json:"onboarding_required"`
		Version  *int `json:"onboarding_version"`
	} `json:"data"`
}

func TestSetupLoginIncludesOnboardingStatus(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}))

	user := &model.User{
		Username:          "onboarding-login-user",
		Password:          "password123",
		Role:              common.RoleCommonUser,
		Status:            common.UserStatusEnabled,
		Group:             "default",
		AffCode:           "onboarding-login-user",
		OnboardingVersion: model.NewUserOnboardingVersion(),
	}
	require.NoError(t, db.Create(user).Error)

	router := gin.New()
	store := cookie.NewStore([]byte("test-session-secret"))
	router.Use(sessions.Sessions("session", store))
	router.GET("/", func(c *gin.Context) {
		setupLogin(user, c)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response onboardingResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.True(t, response.Data.Required)
	require.NotNil(t, response.Data.Version)
	assert.Equal(t, model.OnboardingPendingVersion, *response.Data.Version)
}

func TestCompleteOnboardingIsIdempotent(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	user := &model.User{
		Username:          "onboarding-complete-user",
		Password:          "password123",
		Role:              common.RoleCommonUser,
		Status:            common.UserStatusEnabled,
		AffCode:           "onboarding-complete-user",
		OnboardingVersion: model.NewUserOnboardingVersion(),
	}
	require.NoError(t, db.Create(user).Error)

	router := gin.New()
	router.PUT("/", func(c *gin.Context) {
		c.Set("id", user.Id)
		CompleteOnboarding(c)
	})

	for range 2 {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/", nil)
		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		var response onboardingResponse
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.True(t, response.Success)
		assert.False(t, response.Data.Required)
		require.NotNil(t, response.Data.Version)
		assert.Equal(t, model.CurrentOnboardingVersion, *response.Data.Version)
	}

	var storedUser model.User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	require.NotNil(t, storedUser.OnboardingVersion)
	assert.Equal(t, model.CurrentOnboardingVersion, *storedUser.OnboardingVersion)
}
