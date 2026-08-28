package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRequestConfigUsesValidatedHeaders(t *testing.T) {
	header := make(http.Header)
	header.Set(mockFailureRateHeader, "0.25")
	header.Set(mockFailureStatusHeader, "502")
	header.Set(mockLatencyMSHeader, "750")

	got, err := loadRequestConfig(header, config{errorRate: 0.1}, 1)
	require.NoError(t, err)
	assert.Equal(t, 0.25, got.errorRate)
	assert.Equal(t, http.StatusBadGateway, got.errorStatus)
	assert.Equal(t, 750*time.Millisecond, got.latency)
}

func TestLoadRequestConfigAcceptsMixedFailureStatus(t *testing.T) {
	header := make(http.Header)
	header.Set(mockFailureStatusHeader, "0")

	got, err := loadRequestConfig(header, config{}, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, got.errorStatus)
}

func TestResolveFailureStatus(t *testing.T) {
	for index, want := range mockFailureStatuses {
		assert.Equal(t, want, resolveFailureStatus(0, index))
	}
	assert.Equal(t, http.StatusBadGateway, resolveFailureStatus(http.StatusBadGateway, 0))
	assert.Equal(t, http.StatusServiceUnavailable, resolveFailureStatus(0, -1))
	assert.Equal(t, http.StatusServiceUnavailable, resolveFailureStatus(0, len(mockFailureStatuses)))
}

func TestLoadRequestConfigRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
	}{
		{name: "rate", header: mockFailureRateHeader, value: "1.1"},
		{name: "status", header: mockFailureStatusHeader, value: "418"},
		{name: "latency", header: mockLatencyMSHeader, value: "120001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := make(http.Header)
			header.Set(tt.header, tt.value)
			_, err := loadRequestConfig(header, config{}, 1)
			assert.Error(t, err)
		})
	}
}

func TestLoadRequestConfigSelectsChannelProfile(t *testing.T) {
	header := make(http.Header)
	header.Set(mockChannelsHeader, `[
  {"slot":1,"max_rps":10,"failure_rate":0.1,"failure_status":503,"latency_ms":50},
  {"slot":2,"max_rps":20,"failure_rate":0.2,"failure_status":0,"latency_ms":100},
  {"slot":3,"max_rps":30,"failure_rate":0,"failure_status":429,"latency_ms":0}
]`)

	got, err := loadRequestConfig(header, config{}, 2)
	require.NoError(t, err)
	assert.Equal(t, 20, got.maxRPS)
	assert.Equal(t, 0.2, got.errorRate)
	assert.Equal(t, 0, got.errorStatus)
	assert.Equal(t, 100*time.Millisecond, got.latency)
}

func TestChannelCapacityLimitsRequestsPerWindow(t *testing.T) {
	state := &channelState{}
	now := time.Unix(100, 0)
	assert.True(t, state.allowRequest(2, now))
	assert.True(t, state.allowRequest(2, now.Add(100*time.Millisecond)))
	assert.False(t, state.allowRequest(2, now.Add(200*time.Millisecond)))
	assert.True(t, state.allowRequest(2, now.Add(time.Second)))
}

func TestHandleChatRejectsRequestsAboveConfiguredChannelCapacity(t *testing.T) {
	profiles := `[{"slot":1,"max_rps":1,"failure_rate":0,"failure_status":503,"latency_ms":0},{"slot":2,"max_rps":1,"failure_rate":0,"failure_status":503,"latency_ms":0},{"slot":3,"max_rps":1,"failure_rate":0,"failure_status":503,"latency_ms":0}]`
	state := &channelState{id: 1, name: "mock-a", remaining: 1000}
	firstRequest := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-test","stream":false}`))
	firstRequest.Header.Set(mockChannelsHeader, profiles)
	firstRecorder := httptest.NewRecorder()
	handleChat(firstRecorder, firstRequest, config{}, state)
	require.Equal(t, http.StatusOK, firstRecorder.Code)

	secondRequest := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-test","stream":false}`))
	secondRequest.Header.Set(mockChannelsHeader, profiles)
	secondRecorder := httptest.NewRecorder()
	handleChat(secondRecorder, secondRequest, config{}, state)

	assert.Equal(t, http.StatusTooManyRequests, secondRecorder.Code)
	assert.Contains(t, secondRecorder.Body.String(), `"code":"mock_capacity_exceeded"`)
}

func TestShouldInjectFailureHonorsBoundary(t *testing.T) {
	assert.True(t, shouldInjectFailure(0.25, 0.249))
	assert.False(t, shouldInjectFailure(0.25, 0.25))
	assert.False(t, shouldInjectFailure(0, 0))
}

func TestHandleChatInjectsConfiguredHTTPFailure(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-test","stream":false}`))
	request.Header.Set(mockFailureRateHeader, "1")
	request.Header.Set(mockFailureStatusHeader, "502")
	recorder := httptest.NewRecorder()
	state := &channelState{id: 7, name: "mock", remaining: 1000}

	handleChat(recorder, request, config{}, state)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"code":"mock_error"`)
	assert.Equal(t, "7", recorder.Header().Get("X-Mock-Channel"))
}

func TestHandleChatReturnsClaudeResponseForMessages(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"model":"claude-test","stream":false}`))
	recorder := httptest.NewRecorder()
	state := &channelState{id: 1, name: "mock", remaining: 1000}

	handleChat(recorder, request, config{}, state)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"type":"message"`)
	assert.Contains(t, recorder.Body.String(), `"input_tokens":10`)
}
