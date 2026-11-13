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
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRelayInvalidCallerRequestReturnsBadRequest(t *testing.T) {
	for _, tc := range []struct {
		name, body, message string
	}{
		{"missing model", `{}`, "model is required"},
		{"missing messages", `{"model":"test-model","messages":[]}`, "field messages is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set(common.RequestIdKey, "req_invalid_caller")
			Relay(c, types.RelayFormatOpenAI)
			require.Equal(t, http.StatusBadRequest, w.Code)
			var response struct {
				Error types.PublicOpenAIError `json:"error"`
			}
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
			assert.Equal(t, "invalid_request", response.Error.Code)
			assert.Equal(t, tc.message, response.Error.Message)
			assert.Equal(t, "req_invalid_caller", response.Error.RequestID)
			assert.NotContains(t, w.Body.String(), "channel_id")
		})
	}
}

func TestUserQuotaResponseProjectsChannelWithoutChangingAdminDataPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run PostgreSQL public quota response tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	previousDB, previousCache := model.DB, common.MemoryCacheEnabled
	t.Cleanup(func() {
		model.DB, common.MemoryCacheEnabled = previousDB, previousCache
		assert.NoError(t, tx.Rollback().Error)
		assert.NoError(t, sqlDB.Close())
	})
	schema := fmt.Sprintf("public_quota_test_%d", time.Now().UnixNano())
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	model.DB, common.MemoryCacheEnabled = tx, false
	require.NoError(t, tx.AutoMigrate(&model.QuotaData{}, &model.Channel{}))
	require.NoError(t, tx.Create(&model.Channel{Id: 923, Name: "private-supplier", Key: "test-key"}).Error)
	require.NoError(t, tx.Create(&model.QuotaData{UserID: 7, ChannelID: 923, ModelName: "public-model", CreatedAt: 1100, Quota: 50, TokenUsed: 10, Count: 1}).Error)
	require.NoError(t, tx.Create(&model.QuotaData{UserID: 8, ChannelID: 923, ModelName: "other-model", CreatedAt: 1100, Quota: 90, TokenUsed: 20, Count: 2}).Error)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", 7)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/data/self?start_timestamp=1000&end_timestamp=2000", nil)
	GetUserQuotaDates(c)
	var response struct {
		Success bool              `json:"success"`
		Data    []model.QuotaData `json:"data"`
	}
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data, 1)
	assert.Equal(t, 1, response.Data[0].ChannelID)
	assert.Equal(t, "auto", response.Data[0].ChannelName)
	assert.Equal(t, 50, response.Data[0].Quota)
	assert.Equal(t, 10, response.Data[0].TokenUsed)
	adminData, err := model.GetAllQuotaDates(1000, 2000, "")
	require.NoError(t, err)
	require.Len(t, adminData, 2)
	for _, row := range adminData {
		assert.Equal(t, 923, row.ChannelID)
		assert.Equal(t, "private-supplier #923", row.ChannelName)
	}
	var stored model.QuotaData
	require.NoError(t, tx.Where("user_id = ?", 7).First(&stored).Error)
	assert.Equal(t, 923, stored.ChannelID)
}
