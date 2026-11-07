package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newPublicProjectionTestError() *types.NewAPIError {
	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "provider detail: https://provider.example/request/secret",
		Type:    "provider_error",
		Code:    "provider_internal",
		Source:  types.ErrorSourceChannel,
	}, http.StatusBadGateway)
	apiErr.SetChannelLocation(17, "private-provider")
	return apiErr
}

func TestWriteRelayErrorProjectsJSONAndClaudeResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		format types.RelayFormat
		check  func(t *testing.T, body string)
	}{
		{
			name:   "openai",
			format: types.RelayFormatOpenAI,
			check: func(t *testing.T, body string) {
				require.Contains(t, body, "Service temporarily unavailable")
				require.Contains(t, body, "request id: req-local")
			},
		},
		{
			name:   "claude",
			format: types.RelayFormatClaude,
			check: func(t *testing.T, body string) {
				require.Contains(t, body, `"type":"server_error"`)
				require.Contains(t, body, "request id: req-local")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Set(common.RequestIdKey, "req-local")

			writeRelayError(ctx, tt.format, nil, newPublicProjectionTestError(), "req-local")

			body := recorder.Body.String()
			require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
			require.NotContains(t, body, "provider detail")
			require.NotContains(t, body, "provider_internal")
			require.NotContains(t, body, "private-provider")
			tt.check(t, body)
		})
	}
}

func TestWriteRelayErrorProjectsSSEAfterResponseStarted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(common.RequestIdKey, "req-local")
	_, err := ctx.Writer.Write([]byte("data: partial\n\n"))
	require.NoError(t, err)

	writeRelayError(ctx, types.RelayFormatOpenAI, nil, newPublicProjectionTestError(), "req-local")

	body := recorder.Body.String()
	require.True(t, strings.Contains(body, "data: "))
	require.NotContains(t, body, "provider detail")
	require.NotContains(t, body, "private-provider")
	require.Contains(t, body, "request id: req-local")
}

func TestPublicProjectionDoesNotUseUnderlyingErrorText(t *testing.T) {
	apiErr := types.NewErrorWithStatusCode(
		errors.New("internal provider response"),
		types.ErrorCodeBadResponse,
		http.StatusBadGateway,
	)
	public := apiErr.ToPublicOpenAIError()
	require.NotContains(t, public.Message, "internal provider response")
}
