package types

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalErrorUsesNeutralSourceCode(t *testing.T) {
	apiErr := NewErrorWithStatusCode(
		errors.New("invalid model parameter"),
		ErrorCodeInvalidRequest,
		http.StatusBadRequest,
		ErrOptionWithSkipRetry(),
	)
	apiErr.SetRetryable(false)
	apiErr.SetRequestID("req-local")

	response := apiErr.ToOpenAIError()

	assert.Equal(t, ErrorCodeInvalidRequest, response.Code)
	assert.Equal(t, ErrorSourceUnknown, response.Source)
	assert.Equal(t, "invalid_request", response.SourceCode)
	require.NotNil(t, response.Retryable)
	assert.False(t, *response.Retryable)
	assert.Equal(t, "req-local", response.RequestID)
	assert.Equal(t, 301001, response.StableCode)
	assert.Equal(t, "301001", response.ErrorRef)
	assert.Equal(t, "request", response.FailureScope)
}

func TestOpenAIErrorPreservesUpstreamCodeAndSource(t *testing.T) {
	apiErr := WithOpenAIError(OpenAIError{
		Message: "rate limited",
		Type:    "rate_limit_error",
		Code:    "rate_limit_exceeded",
		Source:  ErrorSourceOpenAI,
	}, http.StatusTooManyRequests)
	apiErr.SetRetryable(true)
	apiErr.SetChannelLocation(23, "Claude Pro")

	response := apiErr.ToOpenAIError()

	assert.Equal(t, "rate_limit_exceeded", response.Code)
	assert.Equal(t, ErrorSourceOpenAI, response.Source)
	assert.Equal(t, "openai.rate_limit_exceeded", response.SourceCode)
	require.NotNil(t, response.Retryable)
	assert.True(t, *response.Retryable)
	assert.Equal(t, 104001, response.StableCode)
	assert.Equal(t, "104001-CH23", response.ErrorRef)
	assert.Equal(t, 23, response.ChannelID)
	assert.Equal(t, "Claude Pro", response.ChannelName)
	assert.Equal(t, "channel", response.FailureScope)
}

func TestChannelRateLimitIsChannelScoped(t *testing.T) {
	apiErr := WithOpenAIError(OpenAIError{
		Message: "channel rate limited",
		Code:    "rate_limit_exceeded",
		Source:  ErrorSourceChannel,
	}, http.StatusTooManyRequests)
	apiErr.SetChannelLocation(23, "Claude Pro")

	response := apiErr.ToOpenAIError()

	assert.Equal(t, 204001, response.StableCode)
	assert.Equal(t, "channel", response.FailureScope)
	assert.Equal(t, "204001-CH23", response.ErrorRef)
}

func TestChannelGenericServerErrorsAreChannelScoped(t *testing.T) {
	for _, statusCode := range []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable} {
		apiErr := WithOpenAIError(OpenAIError{
			Message: "upstream server error",
			Code:    "unknown_error",
			Source:  ErrorSourceChannel,
		}, statusCode)
		apiErr.SetChannelLocation(4, "Official")

		response := apiErr.ToOpenAIError()

		assert.Equal(t, 205002, response.StableCode)
		assert.Equal(t, "channel", response.FailureScope)
		assert.Equal(t, "205002-CH4", response.ErrorRef)
	}
}

func TestNewUpstreamExhaustedErrorKeepsStructuredCause(t *testing.T) {
	lastErr := WithOpenAIError(OpenAIError{
		Message: "channel unavailable",
		Type:    "server_error",
		Code:    "server_error",
		Source:  ErrorSourceChannel,
	}, http.StatusBadGateway)

	exhausted := NewUpstreamExhaustedError(lastErr, 3)
	response := exhausted.ToOpenAIError()

	assert.Equal(t, ErrorCodeUpstreamExhausted, response.Code)
	assert.Equal(t, ErrorSourceUnknown, response.Source)
	assert.Equal(t, "upstream_exhausted", response.SourceCode)
	assert.Equal(t, 3, response.AttemptCount)
	require.NotNil(t, response.Cause)
	assert.Equal(t, ErrorSourceChannel, response.Cause.Source)
	assert.Equal(t, "channel.server_error", response.Cause.Code)
	assert.Equal(t, "server_error", response.Cause.RawCode)
	assert.Equal(t, http.StatusBadGateway, response.Cause.StatusCode)
}

func TestUpstreamExhaustedResponseOmitsProductNamespace(t *testing.T) {
	lastErr := WithOpenAIError(OpenAIError{
		Message: "channel unavailable",
		Code:    "server_error",
		Source:  ErrorSourceChannel,
	}, http.StatusBadGateway)

	response := NewUpstreamExhaustedError(lastErr, 2).ToOpenAIError()
	payload, err := common.Marshal(response)
	require.NoError(t, err)

	assert.NotContains(t, string(payload), "alltoken")
	assert.NotContains(t, string(payload), "AllToken")
	assert.Contains(t, string(payload), `"source_code":"upstream_exhausted"`)
	assert.NotContains(t, string(payload), `"source":""`)
}

func TestLocalCauseOmitsUnknownSource(t *testing.T) {
	response := NewUpstreamExhaustedError(
		NewError(errors.New("local upstream failure"), ErrorCodeBadResponse),
		1,
	).ToOpenAIError()
	payload, err := common.Marshal(response)
	require.NoError(t, err)

	assert.NotContains(t, string(payload), `"source":""`)
}

func TestParseErrorSourceRejectsProductNames(t *testing.T) {
	assert.Equal(t, ErrorSourceUnknown, ParseErrorSource("alltoken"))
	assert.Equal(t, ErrorSourceUnknown, ParseErrorSource("ikun"))
}

func TestResolveErrorSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured string
		baseURL    string
		want       ErrorSource
	}{
		{name: "explicit channel", configured: "channel", baseURL: "https://api.openai.com/v1", want: ErrorSourceChannel},
		{name: "official OpenAI", baseURL: "https://api.openai.com/v1", want: ErrorSourceOpenAI},
		{name: "generic compatible endpoint", baseURL: "https://api.example.com/v1", want: ErrorSourceChannel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ResolveErrorSource(tt.configured, tt.baseURL))
		})
	}
}

func TestChannelExhaustionUsesStableLocationReference(t *testing.T) {
	apiErr := WithOpenAIError(OpenAIError{
		Message: "all pools exhausted",
		Code:    "all_pools_exhausted",
		Source:  ErrorSourceChannel,
	}, http.StatusServiceUnavailable)
	apiErr.SetChannelLocation(17, "Claude Plus")

	response := apiErr.ToOpenAIError()

	assert.Equal(t, 205003, response.StableCode)
	assert.Equal(t, "205003-CH17", response.ErrorRef)
	assert.Equal(t, "channel", response.FailureScope)
	assert.Equal(t, "switch_channel", response.Action)
}

func TestUnknownUpstreamSourceIsIgnored(t *testing.T) {
	apiErr := WithOpenAIError(OpenAIError{
		Message: "spoofed source",
		Code:    "server_error",
		Source:  ErrorSource("product_name"),
	}, http.StatusBadGateway)

	assert.Equal(t, ErrorSourceUnknown, apiErr.GetErrorSource())
	apiErr.EnsureErrorSource(ErrorSourceChannel)
	assert.Equal(t, ErrorSourceChannel, apiErr.GetErrorSource())
}

func TestConfiguredClassificationOverridesBuiltInCatalog(t *testing.T) {
	apiErr := WithOpenAIError(OpenAIError{
		Message: "pool depleted",
		Code:    "vendor_pool_empty",
		Source:  ErrorSourceChannel,
	}, http.StatusServiceUnavailable)
	apiErr.SetChannelLocation(23, "Claude Pro")
	apiErr.SetClassification(205004, "upstream", "channel", "switch_channel", true)

	response := apiErr.ToOpenAIError()

	assert.Equal(t, 205004, response.StableCode)
	assert.Equal(t, "205004-CH23", response.ErrorRef)
	assert.Equal(t, "channel", response.FailureScope)
	require.NotNil(t, response.Retryable)
	assert.True(t, *response.Retryable)
}
