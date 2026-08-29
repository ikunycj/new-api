package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyMockLoadTestRequestBindsConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	channels := `[{"slot":1,"failure_rate":0.5,"failure_status":503,"latency_ms":10}]`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set(constant.MockLoadTestHeader, "true")
	req.Header.Set(constant.MockLoadTestRunHeader, "run-1")
	req.Header.Set("X-Alltoken-Mock-Channels", channels)
	req.Header.Set(constant.MockLoadTestTokenHeader, MockLoadTestSignature("run-1", 7, channels, 0, 0, 0))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Set("token_id", 7)
	require.NoError(t, VerifyMockLoadTestRequest(c, channels, 0, 0, 0))
	assert.Error(t, VerifyMockLoadTestRequest(c, channels+" ", 0, 0, 0))
}

func TestVerifyMockLoadTestRequestRequiresMarkerAndTokenContext(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	assert.NoError(t, VerifyMockLoadTestRequest(c, "", 0, 0, 0))
	c.Request.Header.Set(constant.MockLoadTestHeader, "true")
	assert.Error(t, VerifyMockLoadTestRequest(c, "", 0, 0, 0))
}
