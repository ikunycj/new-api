package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRecommendedModelLeaderboardPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run PostgreSQL recommendation settings test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, pool.Close()) })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { assert.NoError(t, tx.Rollback().Error) })
	schema := fmt.Sprintf("recommended_models_test_%d", time.Now().UnixNano())
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	previousDB := model.DB
	common.OptionMapRWMutex.Lock()
	previousOptions := common.OptionMap
	common.OptionMap = nil
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		model.DB = previousDB
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptions
		common.OptionMapRWMutex.Unlock()
		model.InvalidatePricingCache()
	})
	model.DB = tx
	model.InvalidatePricingCache()
	require.NoError(t, tx.AutoMigrate(&model.Option{}, &model.Ability{}, &model.Channel{}, &model.Model{}, &model.Vendor{}))
	require.NoError(t, tx.Create(&model.Option{
		Key:   "ModelLeaderboardSettings",
		Value: `{"enabled":false,"default":"coding"}`,
	}).Error)
	require.NoError(t, tx.Create(&model.Channel{Id: 1, Name: "Test channel", Key: "test-key", Type: constant.ChannelTypeOpenAI}).Error)
	require.NoError(t, tx.Create(&[]model.Ability{
		{Model: "alpha-model", Group: "default", ChannelId: 1, Enabled: true},
		{Model: "zeta-model", Group: "default", ChannelId: 1, Enabled: true},
	}).Error)

	const adminPath = "/api/option/model_leaderboards"
	const publicPath = "/api/model_leaderboards"
	const statusPath = "/api/status"
	router := gin.New()
	router.GET(publicPath, GetPublicModelLeaderboards)
	router.GET(adminPath, GetAdminModelLeaderboards)
	router.PUT(adminPath, UpdateModelLeaderboardSettings)
	router.GET(statusPath, GetStatus)
	type responseData struct {
		model.ModelLeaderboardSettings
		DefaultModelOrder []string `json:"default_model_order"`
	}
	request := func(method, path, body string) responseData {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		require.Equal(t, http.StatusOK, response.Code)
		var result struct {
			Success bool         `json:"success"`
			Message string       `json:"message"`
			Data    responseData `json:"data"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
		require.True(t, result.Success, result.Message)
		if path == adminPath || path == publicPath {
			assert.NotContains(t, response.Body.String(), `"default":`)
		}
		return result.Data
	}

	assert.Equal(t, []string{}, request(http.MethodGet, publicPath, "").RecommendedModels)
	assert.Equal(t, []string{}, request(http.MethodGet, adminPath, "").RecommendedModels)

	saved := request(http.MethodPut, adminPath, `{"enabled":true,"recommended_models":[" zeta-model ","future-model","", "Alpha-model", "alpha-model","zeta-model"]}`).ModelLeaderboardSettings
	assert.Equal(t, model.ModelLeaderboardSettings{
		Enabled:           true,
		RecommendedModels: []string{"zeta-model", "future-model", "Alpha-model", "alpha-model"},
	}, saved)
	assert.Equal(t, saved, request(http.MethodGet, adminPath, "").ModelLeaderboardSettings)
	assert.Equal(t, []string{"zeta-model", "alpha-model"}, request(http.MethodGet, publicPath, "").RecommendedModels)
	assert.Equal(t, []string{"zeta-model", "alpha-model"}, request(http.MethodGet, statusPath, "").DefaultModelOrder)

	var option model.Option
	require.NoError(t, tx.Take(&option, &model.Option{Key: "ModelLeaderboardSettings"}).Error)
	assert.NotContains(t, option.Value, `"default":`)

	assert.Equal(t, []string{}, request(http.MethodPut, adminPath, `{"enabled":true,"recommended_models":[]}`).RecommendedModels)
	assert.Equal(t, []string{}, request(http.MethodGet, publicPath, "").RecommendedModels)
	assert.Equal(t, []string{}, request(http.MethodGet, statusPath, "").DefaultModelOrder)
}

func TestModelLeaderboardSyncDueAtFourBeijingTime(t *testing.T) {
	scheduled := time.Date(2026, 11, 22, 20, 0, 0, 0, time.UTC)
	handler := modelLeaderboardSyncHandler{}
	previous := &model.SystemTask{CreatedAt: scheduled.Add(-time.Minute).Unix()}
	today := &model.SystemTask{CreatedAt: scheduled.Unix()}

	assert.False(t, handler.Due(scheduled.Add(-time.Second), nil))
	assert.True(t, handler.Due(scheduled, nil))
	assert.True(t, handler.Due(scheduled.Add(time.Minute), previous))
	assert.False(t, handler.Due(scheduled.Add(time.Minute), today))
	assert.True(t, handler.Due(scheduled.Add(24*time.Hour), today))
}
