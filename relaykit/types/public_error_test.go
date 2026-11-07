package types

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicErrorHidesUpstreamDetailsAndChannelLocation(t *testing.T) {
	apiErr := WithOpenAIError(OpenAIError{
		Message: "provider secret: https://upstream.example/request/abc",
		Type:    "provider_error",
		Code:    "provider_internal_code",
		Source:  ErrorSourceChannel,
	}, http.StatusBadGateway)
	apiErr.SetChannelLocation(42, "private-provider")

	public := apiErr.ToPublicOpenAIError(PublicErrorOptions{MessageSuffix: "request id: req_local"})
	payload, err := kitutil.Marshal(public)
	require.NoError(t, err)

	assert.Equal(t, "Service temporarily unavailable, please retry later (request id: req_local)", public.Message)
	assert.Equal(t, "server_error", public.Type)
	assert.Equal(t, "service_unavailable", public.Code)
	assert.NotContains(t, string(payload), "provider secret")
	assert.NotContains(t, string(payload), "provider_internal_code")
	assert.NotContains(t, string(payload), "upstream.example")
	assert.NotContains(t, string(payload), "private-provider")
	assert.NotContains(t, string(payload), "channel_id")
	assert.NotContains(t, string(payload), "channel_name")
}

func TestPublicErrorKeepsLocalClassificationWithoutOriginalMessage(t *testing.T) {
	apiErr := NewErrorWithStatusCode(
		errors.New("internal validation details"),
		ErrorCodeInvalidRequest,
		http.StatusBadRequest,
	)

	public := apiErr.ToPublicClaudeError()
	assert.Equal(t, "invalid_request_error", public.Type)
	assert.Equal(t, "Invalid request", public.Message)
	assert.NotContains(t, public.Message, "internal validation details")
}

func TestPublicErrorMapsUpstreamRateLimitToRetryableResponse(t *testing.T) {
	apiErr := NewErrorWithStatusCode(
		errors.New("provider rate limit details"),
		ErrorCodeUpstreamExhausted,
		http.StatusTooManyRequests,
	)

	public := apiErr.ToPublicOpenAIError()
	assert.Equal(t, "rate_limit_error", public.Type)
	assert.Equal(t, "rate_limit_exceeded", public.Code)
	assert.Equal(t, "Rate limit exceeded, please retry later", public.Message)
}
