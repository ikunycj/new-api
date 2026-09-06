package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalAPIRateLimitUsesUpdatedSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousRedisEnabled := common.RedisEnabled
	previousEnabled := common.GlobalApiRateLimitEnable
	previousNum := common.GlobalApiRateLimitNum
	previousDuration := common.GlobalApiRateLimitDuration
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		common.GlobalApiRateLimitEnable = previousEnabled
		common.GlobalApiRateLimitNum = previousNum
		common.GlobalApiRateLimitDuration = previousDuration
	})

	common.RedisEnabled = false
	common.GlobalApiRateLimitEnable = true
	common.GlobalApiRateLimitNum = 1
	common.GlobalApiRateLimitDuration = 60
	handler := GlobalAPIRateLimit()
	require.NotNil(t, handler)

	first, _ := gin.CreateTestContext(httptest.NewRecorder())
	first.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	first.Request.RemoteAddr = "198.51.100.254:12345"
	handler(first)
	assert.False(t, first.IsAborted())

	secondRecorder := httptest.NewRecorder()
	second, _ := gin.CreateTestContext(secondRecorder)
	second.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	second.Request.RemoteAddr = "198.51.100.254:12345"
	handler(second)
	assert.True(t, second.IsAborted())
	assert.Equal(t, http.StatusTooManyRequests, second.Writer.Status())

	common.GlobalApiRateLimitEnable = false
	third, _ := gin.CreateTestContext(httptest.NewRecorder())
	third.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	third.Request.RemoteAddr = "198.51.100.254:12345"
	handler(third)
	assert.False(t, third.IsAborted())
}
