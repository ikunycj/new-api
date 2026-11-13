package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicRelayErrorHidesChannelDiagnosticsAndUsesPlaceholders(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	previous := common.OptionMap
	common.OptionMap = map[string]string{common.ChannelErrorWhitelistOptionKey: "[]"}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previous
		common.OptionMapRWMutex.Unlock()
	})

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "private supplier account exhausted",
		Code:    "pool_exhausted",
		Source:  types.ErrorSourceChannel,
	}, http.StatusServiceUnavailable)
	apiErr.SetChannelLocation(23, "private-supplier")
	apiErr.SetRequestID("req-public")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set(common.RequestIdKey, "req-public")
	view := PublicRelayError(ctx, apiErr)

	assert.Equal(t, common.PublicChannelID, view.ChannelID)
	assert.Equal(t, common.PublicChannelName, view.ChannelName)
	assert.Equal(t, common.PublicErrorCode, view.Code)
	assert.Equal(t, "req-public", view.RequestID)
	assert.Contains(t, view.Message, "服务暂异常")
	assert.NotContains(t, view.Message, "private supplier")
}

func TestShouldCopyUpstreamHeaderOnlyAllowsApprovedProtocolHeaders(t *testing.T) {
	assert.True(t, ShouldCopyUpstreamHeader(nil, "Content-Type", []string{"application/json"}))
	assert.True(t, ShouldCopyUpstreamHeader(nil, "X-Codex-Turn-State", []string{"done"}))
	assert.False(t, ShouldCopyUpstreamHeader(nil, "X-Error-Ref", []string{"205001-CH23"}))
	assert.False(t, ShouldCopyUpstreamHeader(nil, "X-Upstream-Request-Id", []string{"upstream-1"}))
	assert.True(t, ShouldCopyUpstreamHeader(nil, "Retry-After", []string{"60"}))
	assert.False(t, ShouldCopyUpstreamHeader(nil, "Retry-After", []string{"account pool exhausted"}))
	assert.True(t, ShouldCopyUpstreamHeader(nil, "Last-Modified", []string{http.TimeFormat}))
}

func TestPublicRelayErrorWhitelistSupportsStatusAndCustomErrorFields(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	previous := common.OptionMap
	common.OptionMap = map[string]string{common.ChannelErrorWhitelistOptionKey: `[{"match_type":"status","match_value":"503","response_mode":"custom","custom_code":"service_unavailable","custom_type":"server_error","custom_message":"请稍后再试"}]`}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previous
		common.OptionMapRWMutex.Unlock()
	})

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "private supplier account exhausted",
		Code:    "pool_exhausted",
		Source:  types.ErrorSourceChannel,
	}, http.StatusServiceUnavailable)
	apiErr.SetChannelLocation(23, "private-supplier")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	view := PublicRelayError(ctx, apiErr)
	assert.Equal(t, "请稍后再试", view.Message)
	assert.Equal(t, "service_unavailable", view.Code)
	assert.Equal(t, "server_error", view.Type)
	assert.Equal(t, common.PublicChannelID, view.ChannelID)
	assert.NotContains(t, view.Message, "private supplier")

	local := types.NewErrorWithStatusCode(errors.New("invalid model"), types.ErrorCodeInvalidRequest, http.StatusBadRequest)
	localView := PublicRelayError(ctx, local)
	require.Equal(t, string(types.ErrorCodeInvalidRequest), localView.Code)
	assert.Contains(t, localView.Message, "invalid model")
}

func TestProjectPublicStreamDataOnlyProjectsErrorEvents(t *testing.T) {
	normal, projected := ProjectPublicStreamData(`{"type":"response.output_text.delta","delta":"supplier word"}`)
	assert.False(t, projected)
	assert.Contains(t, normal, "supplier word")

	errorData, projected := ProjectPublicStreamData(`{"type":"error","error":{"code":"pool_exhausted","message":"private supplier","channel_id":23}}`)
	assert.True(t, projected)
	assert.Contains(t, errorData, common.DefaultPublicChannelErrorMessage())
	assert.NotContains(t, errorData, "private supplier")
	assert.NotContains(t, errorData, "23")
	topLevel, projected := ProjectPublicStreamData(`{"type":"response.failed","code":"pool_exhausted","message":"private supplier","response":{"error":{"message":"private supplier"}}}`)
	assert.True(t, projected)
	assert.NotContains(t, topLevel, "private supplier")
}

func TestPublicRelayErrorWhitelistFuzzyMatchesErrorFields(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	previous := common.OptionMap
	common.OptionMap = map[string]string{common.ChannelErrorWhitelistOptionKey: `[{"match_type":"error","match_value":"rate limit","response_mode":"custom","custom_code":"rate_limited","custom_type":"rate_limit_error","custom_message":"请降低请求频率"}]`}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previous
		common.OptionMapRWMutex.Unlock()
	})

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "provider rate limit exceeded",
		Type:    "upstream_rate_limit",
		Code:    "provider_rate_limit",
		Source:  types.ErrorSourceChannel,
	}, http.StatusTooManyRequests)
	apiErr.SetChannelLocation(23, "private-supplier")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	view := PublicRelayError(ctx, apiErr)
	assert.Equal(t, "rate_limited", view.Code)
	assert.Equal(t, "rate_limit_error", view.Type)
	assert.Equal(t, "请降低请求频率", view.Message)
}
