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

	got, err := loadRequestConfig(header, config{errorRate: 0.1})
	require.NoError(t, err)
	assert.Equal(t, 0.25, got.errorRate)
	assert.Equal(t, http.StatusBadGateway, got.errorStatus)
	assert.Equal(t, 750*time.Millisecond, got.latency)
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
			_, err := loadRequestConfig(header, config{})
			assert.Error(t, err)
		})
	}
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
