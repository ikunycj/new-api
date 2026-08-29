package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	projectcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayMockBranchDoesNotContactConfiguredUpstream(t *testing.T) {
	upstreamHits := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamHits++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()
	gin.SetMode(gin.TestMode)
	payload := map[string]any{"model": "mock-model", "messages": []map[string]string{{"role": "user", "content": "ping"}}}
	body, err := projectcommon.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constant.MockLoadTestHeader, "true")
	req.Header.Set(constant.MockLoadTestRunHeader, "run-relay")
	channels := `[{"slot":1,"failure_rate":0,"failure_status":0,"latency_ms":0}]`
	req.Header.Set("X-Alltoken-Mock-Channels", channels)
	req.Header.Set(constant.MockLoadTestTokenHeader, service.MockLoadTestSignature("run-relay", 7, channels, 0, 0, 0))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Set("id", 1)
	c.Set("token_id", 7)
	c.Set("token_key", "test-key")
	projectcommon.SetContextKey(c, constant.ContextKeyTokenGroup, "test")
	projectcommon.SetContextKey(c, constant.ContextKeyUsingGroup, "test")
	projectcommon.SetContextKey(c, constant.ContextKeyUserGroup, "default")
	projectcommon.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
	c.Set(projectcommon.RequestIdKey, "relay-request")
	projectcommon.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())

	Relay(c, types.RelayFormatOpenAI)
	assert.Equal(t, 0, upstreamHits)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "true", recorder.Header().Get("X-Alltoken-Mock-Executed"))
}

func newMockContext(t *testing.T, channelsJSON string, failureRate float64, failureStatus, latencyMS int) *gin.Context {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set(constant.MockLoadTestHeader, "true")
	req.Header.Set(constant.MockLoadTestRunHeader, "run-1")
	req.Header.Set(constant.MockLoadTestTokenHeader, service.MockLoadTestSignature("run-1", 7, channelsJSON, failureRate, failureStatus, latencyMS))
	if channelsJSON != "" {
		req.Header.Set("X-Alltoken-Mock-Channels", channelsJSON)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Set("token_id", 7)
	c.Set(projectcommon.RequestIdKey, "request-1")
	return c
}

func TestInternalMockNeverNeedsChannelAndRetriesSlotsInOrder(t *testing.T) {
	channels := `[{
  "slot": 1, "failure_rate": 1, "failure_status": 503, "latency_ms": 0
}, {
  "slot": 2, "failure_rate": 0, "failure_status": 0, "latency_ms": 0
}, {
  "slot": 3, "failure_rate": 1, "failure_status": 429, "latency_ms": 0
}]`
	c := newMockContext(t, channels, 0, 0, 0)
	info := &relayInfoForMockTest
	result := executeInternalMockLoadTest(c, info, types.RelayFormatOpenAI, 12)
	assert.Nil(t, result, "mock executor should advance to the second slot")
	assert.Equal(t, "true", c.Writer.Header().Get("X-Alltoken-Mock-Executed"))
	assert.Equal(t, "2", c.Writer.Header().Get("X-Alltoken-Mock-Slot"))
	assert.Equal(t, http.StatusOK, c.Writer.Status())
}

func TestInternalMockRejectsUnsignedMarkerBeforeExecution(t *testing.T) {
	c := newMockContext(t, "", 0, 0, 0)
	c.Request.Header.Set(constant.MockLoadTestTokenHeader, "invalid")
	result := executeInternalMockLoadTest(c, &relayInfoForMockTest, types.RelayFormatOpenAI, 1)
	require.Error(t, result)
	assert.Equal(t, types.ErrorCodeAccessDenied, result.GetErrorCode())
	assert.Equal(t, http.StatusForbidden, result.StatusCode)
}

func TestInternalMockReturnsLastConfiguredFailure(t *testing.T) {
	channels := `[{"slot":3,"failure_rate":1,"failure_status":429,"latency_ms":0},{"slot":1,"failure_rate":1,"failure_status":500,"latency_ms":0}]`
	c := newMockContext(t, channels, 0, 0, 0)
	result := executeInternalMockLoadTest(c, &relayInfoForMockTest, types.RelayFormatOpenAI, 1)
	require.Error(t, result)
	assert.Equal(t, http.StatusTooManyRequests, result.StatusCode)
}

var relayInfoForMockTest = relaycommon.RelayInfo{OriginModelName: "mock-model"}
