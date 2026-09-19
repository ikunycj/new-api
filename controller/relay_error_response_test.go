package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func newTestContext(t *testing.T, written int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	if written > 0 {
		// Simulate a stream that already emitted bytes: gin only reports a
		// positive writer size once something was written.
		_, _ = ctx.Writer.Write([]byte(strings.Repeat("x", written)))
	}
	return ctx, recorder
}

func withNormalizedMode(t *testing.T) {
	t.Helper()
	previous := common.PublicErrorMode()
	common.SetPublicErrorMode(common.PublicErrorModeNormalized)
	t.Cleanup(func() { common.SetPublicErrorMode(previous) })
}

func withPassthroughMode(t *testing.T) {
	t.Helper()
	previous := common.PublicErrorMode()
	common.SetPublicErrorMode(common.PublicErrorModePassthrough)
	t.Cleanup(func() { common.SetPublicErrorMode(previous) })
}

// TestWriteRelayErrorBeforeStreamStart verifies the ordinary path: nothing has
// been written yet, so the error is a normal JSON response.
func TestWriteRelayErrorBeforeStreamStart(t *testing.T) {
	withNormalizedMode(t)
	ctx, recorder := newTestContext(t, 0)

	apiErr := types.NewOpenAIError(errors.New("Insufficient account balance"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)
	apiErr.SetChannelLocation(38, "Claude Pro")

	writeRelayError(ctx, types.RelayFormatOpenAI, nil, apiErr, "req_test")

	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v (%q)", err, recorder.Body.String())
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %s", recorder.Body.String())
	}
	message, _ := errObj["message"].(string)
	if strings.Contains(strings.ToLower(message), "insufficient") {
		t.Errorf("upstream wording leaked: %q", message)
	}
	if strings.Contains(message, "CH38") || strings.Contains(message, "Claude Pro") {
		t.Errorf("channel identity leaked: %q", message)
	}
}

// TestWriteRelayErrorAfterStreamStartIsInBand is the regression test for the
// transport bug: an error raised after the first chunk must be delivered as an
// SSE frame, not appended as a bare JSON object.
func TestWriteRelayErrorAfterStreamStartIsInBand(t *testing.T) {
	withNormalizedMode(t)

	for _, tc := range []struct {
		name       string
		format     types.RelayFormat
		wantPrefix string
		wantAbsent string
	}{
		{
			name:       "openai stream",
			format:     types.RelayFormatOpenAI,
			wantPrefix: "xdata: ",
			wantAbsent: "x{",
		},
		{
			name:       "claude stream",
			format:     types.RelayFormatClaude,
			wantPrefix: "xevent: error\n",
			wantAbsent: "x{",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, recorder := newTestContext(t, 1)
			apiErr := types.NewOpenAIError(errors.New("Insufficient account balance"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)
			apiErr.SetChannelLocation(38, "Claude Pro")

			writeRelayError(ctx, tc.format, nil, apiErr, "req_test")
			got := recorder.Body.String()

			if !strings.HasPrefix(got, tc.wantPrefix) {
				t.Errorf("stream error was not framed: %q", got)
			}
			// The raw status must not be re-written onto a committed response.
			if recorder.Code != http.StatusOK {
				t.Errorf("status rewritten after stream start: %d", recorder.Code)
			}
			if strings.Contains(strings.ToLower(got), "insufficient") {
				t.Errorf("upstream wording leaked into stream: %q", got)
			}
			if strings.Contains(got, "CH38") {
				t.Errorf("channel identity leaked into stream: %q", got)
			}
		})
	}
}

// TestInternalErrorHeadersFollowAuthentication pins that the internal
// classification headers survive only for authenticated internal callers.
func TestInternalErrorHeadersFollowAuthentication(t *testing.T) {
	withNormalizedMode(t)

	ctx, _ := newTestContext(t, 0)
	if internalErrorDetailAllowed(ctx) {
		t.Error("a normal caller must not receive internal error headers")
	}

	ctx.Set(contextKeyInternalErrorDetail, true)
	if !internalErrorDetailAllowed(ctx) {
		t.Error("an authenticated internal caller must keep internal error headers")
	}

	// A client-supplied header cannot claim the flag: only the verified
	// load-test path sets it.
	ctx2, _ := newTestContext(t, 0)
	ctx2.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx2.Request.Header.Set("X-Alltoken-Mock-Load-Test", "true")
	if internalErrorDetailAllowed(ctx2) {
		t.Error("a forged marker must not unlock internal error headers")
	}
}

// TestInternalErrorHeadersUnchangedInPassthrough pins the rollback behaviour.
func TestInternalErrorHeadersUnchangedInPassthrough(t *testing.T) {
	withPassthroughMode(t)

	ctx, _ := newTestContext(t, 0)
	if !internalErrorDetailAllowed(ctx) {
		t.Error("passthrough mode must not change header emission")
	}
}
