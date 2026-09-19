package types

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

// upstreamLeakKeywords are the words an upstream uses to describe our own
// supply. None of them may survive the projection.
var upstreamLeakKeywords = []string{
	"insufficient",
	"balance",
	"quota exceeded",
	"no available",
	"pool",
	"account",
	"gptstore",
	"ikun",
	"api.",
	"http://",
	"https://",
}

// internalFieldNames are the JSON keys of the internal classification. They
// must never appear in a projected response body.
var internalFieldNames = []string{
	"channel_id",
	"channel_name",
	"cause",
	"failure_scope",
	"action",
	"error_ref",
	"source_code",
	"attempt_count",
	"retryable",
	"alltoken_code",
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

// TestPublicProjectionHidesUpstreamOperationalState is the core contract: a
// failure of our own supply must not describe that supply to the caller.
func TestPublicProjectionHidesUpstreamOperationalState(t *testing.T) {
	withNormalizedMode(t)

	cases := []struct {
		name       string
		message    string
		errorCode  ErrorCode
		statusCode int
		wantStatus int
		wantType   string
	}{
		{
			name:       "upstream balance exhausted",
			message:    "Insufficient account balance for channel Claude Pro",
			errorCode:  ErrorCodeBadResponseStatusCode,
			statusCode: http.StatusTooManyRequests,
			wantStatus: http.StatusServiceUnavailable,
			wantType:   "server_error",
		},
		{
			name:       "empty account pool",
			message:    "no available accounts in pool vip.gptstore.club",
			errorCode:  ErrorCodeChannelNoAvailableKey,
			statusCode: http.StatusInternalServerError,
			wantStatus: http.StatusServiceUnavailable,
			wantType:   "server_error",
		},
		{
			name:       "upstream credential rejected",
			message:    "invalid api key for https://api.example.com/v1",
			errorCode:  ErrorCodeChannelInvalidKey,
			statusCode: http.StatusUnauthorized,
			wantStatus: http.StatusServiceUnavailable,
			wantType:   "server_error",
		},
		{
			name:       "upstream 502",
			message:    "bad gateway from upstream",
			errorCode:  ErrorCodeBadResponseStatusCode,
			statusCode: http.StatusBadGateway,
			wantStatus: http.StatusServiceUnavailable,
			wantType:   "server_error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := NewOpenAIError(errors.New(tc.message), tc.errorCode, tc.statusCode)
			apiErr.SetChannelLocation(38, "Claude Pro")
			apiErr.SetRequestID("req_abc")

			body := apiErr.ToPublicOpenAIError()

			if got := apiErr.PublicStatusCode(); got != tc.wantStatus {
				t.Errorf("status = %d, want %d", got, tc.wantStatus)
			}
			if body.Type != tc.wantType {
				t.Errorf("type = %q, want %q", body.Type, tc.wantType)
			}
			assertNoInternalLeak(t, body.Message)
			if body.Code != nil && strings.TrimSpace(toStr(body.Code)) != "" {
				t.Errorf("code leaked: %v", body.Code)
			}
		})
	}
}

// TestPublicProjectionReportsCallerFaults guards the other half of the design:
// an error the caller caused must stay actionable.
func TestPublicProjectionReportsCallerFaults(t *testing.T) {
	withNormalizedMode(t)

	cases := []struct {
		name       string
		message    string
		errorCode  ErrorCode
		statusCode int
		wantStatus int
	}{
		{
			name:       "caller quota exhausted",
			message:    "用户额度不足, 剩余额度: $0.00",
			errorCode:  ErrorCodeInsufficientUserQuota,
			statusCode: http.StatusForbidden,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "malformed request",
			message:    "invalid request body: missing model",
			errorCode:  ErrorCodeInvalidRequest,
			statusCode: http.StatusBadRequest,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "model not supported",
			message:    "model claude-opus-4-8 is not available on this channel",
			errorCode:  ErrorCodeModelNotFound,
			statusCode: http.StatusNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := NewOpenAIError(errors.New(tc.message), tc.errorCode, tc.statusCode)
			apiErr.SetChannelLocation(38, "Claude Pro")

			if got := apiErr.PublicStatusCode(); got != tc.wantStatus {
				t.Errorf("status = %d, want %d", got, tc.wantStatus)
			}
			body := apiErr.ToPublicOpenAIError()
			if !strings.Contains(body.Message, strings.SplitN(tc.message, ",", 2)[0]) {
				t.Errorf("caller-facing message was hidden: %q", body.Message)
			}
			// Even when the message is exposed, the channel must not be.
			assertNoInternalLeak(t, body.Message)
			if strings.Contains(body.Message, "38") || strings.Contains(body.Message, "Claude Pro") {
				t.Errorf("channel identity leaked into exposed message: %q", body.Message)
			}
		})
	}
}

// TestPublicProjectionKeepsRateLimitDistinct pins the decision that a 429 stays
// a 429: clients need the backoff signal.
func TestPublicProjectionKeepsRateLimitDistinct(t *testing.T) {
	withNormalizedMode(t)

	for _, code := range []ErrorCode{ErrorCodeBadResponseStatusCode} {
		apiErr := NewOpenAIError(errors.New("rate limit exceeded, too many requests"), code, http.StatusTooManyRequests)
		apiErr.SetClassification(204001, "rate_limit", "channel", "switch_channel", true)

		if got := apiErr.PublicStatusCode(); got != http.StatusTooManyRequests {
			t.Errorf("status = %d, want 429", got)
		}
		body := apiErr.ToPublicOpenAIError()
		if body.Type != "rate_limit_error" {
			t.Errorf("type = %q, want rate_limit_error", body.Type)
		}
		assertNoInternalLeak(t, body.Message)
	}
}

// TestPublicProjectionClaudeFormat covers the Anthropic shape, which carries the
// classification in a different field.
func TestPublicProjectionClaudeFormat(t *testing.T) {
	withNormalizedMode(t)

	apiErr := NewOpenAIError(errors.New("Insufficient account balance"), ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)
	apiErr.SetChannelLocation(56, "vip.gptstore.club")

	claudeErr := apiErr.ToPublicClaudeError()
	if claudeErr.Type != "server_error" {
		t.Errorf("type = %q, want server_error", claudeErr.Type)
	}
	assertNoInternalLeak(t, claudeErr.Message)
	assertNoInternalLeak(t, claudeErr.Type)
}

// TestPublicProjectionExhaustedErrorHasNoCauseChain ensures the nested cause
// record — which embeds channel id, channel name and error_ref — is dropped.
func TestPublicProjectionExhaustedErrorHasNoCauseChain(t *testing.T) {
	withNormalizedMode(t)

	lastErr := NewOpenAIError(errors.New("no healthy account"), ErrorCodeBadResponseStatusCode, http.StatusInternalServerError)
	lastErr.SetChannelLocation(17, "ikun cc代理")
	exhausted := NewUpstreamExhaustedError(lastErr, 3)

	body := exhausted.ToPublicOpenAIError()
	serialized, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, field := range internalFieldNames {
		if strings.Contains(string(serialized), field) {
			t.Errorf("internal field %q leaked into %s", field, serialized)
		}
	}
	assertNoInternalLeak(t, string(serialized))
}

// TestPublicProjectionIgnoresChannelStatusCodeMapping pins that a per-channel
// status rewrite stays internal.
func TestPublicProjectionIgnoresChannelStatusCodeMapping(t *testing.T) {
	withNormalizedMode(t)

	// ResetStatusCode rewrote the status to something a channel config asked
	// for; the projection must not inherit it.
	apiErr := NewOpenAIError(errors.New("Insufficient account balance"), ErrorCodeBadResponseStatusCode, http.StatusInternalServerError)
	apiErr.StatusCode = http.StatusTeapot

	if got := apiErr.PublicStatusCode(); got != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 (channel mapping must not be client-visible)", got)
	}
}

// TestPassthroughModeIsUnchanged pins the rollback path: with the projection
// off, the response must be byte-for-byte the legacy one.
func TestPassthroughModeIsUnchanged(t *testing.T) {
	withPassthroughMode(t)

	apiErr := NewOpenAIError(errors.New("Insufficient account balance"), ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)
	apiErr.SetChannelLocation(38, "Claude Pro")

	legacy := apiErr.ToOpenAIError()
	public := apiErr.ToPublicOpenAIError()

	if legacy.ChannelID != public.ChannelID || legacy.ChannelName != public.ChannelName {
		t.Errorf("passthrough changed channel fields: %+v vs %+v", legacy, public)
	}
	if legacy.ErrorRef != public.ErrorRef {
		t.Errorf("passthrough changed error_ref: %q vs %q", legacy.ErrorRef, public.ErrorRef)
	}
	if legacy.Message != public.Message {
		t.Errorf("passthrough changed message: %q vs %q", legacy.Message, public.Message)
	}
	if got := apiErr.PublicStatusCode(); got != http.StatusTooManyRequests {
		t.Errorf("passthrough changed status: %d", got)
	}

	claudeLegacy := apiErr.ToClaudeError()
	claudePublic := apiErr.ToPublicClaudeError()
	if claudeLegacy.Type != claudePublic.Type || claudeLegacy.Message != claudePublic.Message {
		t.Errorf("passthrough changed claude error: %+v vs %+v", claudeLegacy, claudePublic)
	}
}

// TestPublicProjectionCoversEveryCatalogCode walks the whole classification
// table and asserts no category escapes the projection with an internal field or
// an upstream keyword.
func TestPublicProjectionCoversEveryCatalogCode(t *testing.T) {
	withNormalizedMode(t)

	for rawCode := range alltokenCatalogForTest() {
		apiErr := NewOpenAIError(
			errors.New("Insufficient account balance; no available accounts in pool at https://api.example.com"),
			ErrorCode(rawCode),
			http.StatusInternalServerError,
		)
		apiErr.SetChannelLocation(38, "Claude Pro")
		apiErr.SetErrorSource(ErrorSourceChannel)

		body := apiErr.ToPublicOpenAIError()
		serialized, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("%s: marshal: %v", rawCode, err)
		}
		if body.Message == "" {
			t.Errorf("%s: empty message", rawCode)
		}
		status := apiErr.PublicStatusCode()
		if status < 400 || status > 599 {
			t.Errorf("%s: status %d outside 4xx/5xx", rawCode, status)
		}
		for _, field := range internalFieldNames {
			if strings.Contains(string(serialized), field) {
				t.Errorf("%s: internal field %q leaked into %s", rawCode, field, serialized)
			}
		}
	}
}

// alltokenCatalogForTest exposes the code table keys without duplicating them.
func alltokenCatalogForTest() map[string]errorDefinition {
	out := make(map[string]errorDefinition)
	for rawCode, definition := range alltokenDefinitions {
		out[rawCode] = definition
	}
	return out
}

func assertNoInternalLeak(t *testing.T, text string) {
	t.Helper()
	lower := strings.ToLower(text)
	for _, keyword := range upstreamLeakKeywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			t.Errorf("upstream keyword %q leaked into client-facing text: %q", keyword, text)
		}
	}
}

func toStr(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return "non-empty"
	}
}

// channelUpstreamFailure builds the error the relay produces for a supplier
// response, i.e. one attributed to the channel that served the request. The
// classification of such an error is derived from its source and status, which
// is what makes the failover cause non-trivial.
func channelUpstreamFailure(message string, status int) *NewAPIError {
	err := NewErrorWithStatusCode(errors.New(message), ErrorCodeBadResponseStatusCode, status)
	err.SetErrorSource(ErrorSourceChannel)
	return err
}

// TestFailingOverPreservesTheUpstreamClassification pins the failover case.
//
// When every candidate channel fails, the error the caller finally sees is the
// upstream_exhausted wrapper (305001) rather than the last real failure. The
// wrapper must not erase what actually went wrong upstream: a rate limit that
// survived failover is still a rate limit, and a caller that backs off on 429
// would otherwise be handed a 503 and never learn to slow down.
func TestFailingOverPreservesTheUpstreamClassification(t *testing.T) {
	withNormalizedMode(t)

	cases := []struct {
		name         string
		lastErr      *NewAPIError
		wantStatus   int
		wantCategory PublicErrorCategory
	}{
		{
			name: "upstream rate limit keeps its 429 through failover",
			lastErr: channelUpstreamFailure("Rate limit reached for gpt-4o",
				http.StatusTooManyRequests),
			wantStatus:   http.StatusTooManyRequests,
			wantCategory: PublicErrorCategoryRateLimit,
		},
		{
			name: "upstream balance exhaustion becomes a 503",
			lastErr: channelUpstreamFailure("Insufficient balance for acct_9f31",
				http.StatusPaymentRequired),
			wantStatus:   http.StatusServiceUnavailable,
			wantCategory: PublicErrorCategoryServer,
		},
		{
			name: "upstream 5xx becomes a 503",
			lastErr: channelUpstreamFailure("upstream exploded at node-7",
				http.StatusInternalServerError),
			wantStatus:   http.StatusServiceUnavailable,
			wantCategory: PublicErrorCategoryServer,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := NewUpstreamExhaustedError(tc.lastErr, 2)

			if got := wrapped.AlltokenCode(); got != 305001 {
				t.Fatalf("wrapper classification = %d, want 305001", got)
			}
			if got := wrapped.PublicStatusCode(); got != tc.wantStatus {
				t.Errorf("PublicStatusCode() = %d, want %d", got, tc.wantStatus)
			}
			if got := publicCategoryFor(wrapped); got != tc.wantCategory {
				t.Errorf("publicCategoryFor() = %q, want %q", got, tc.wantCategory)
			}
		})
	}
}

// TestFailoverWrapperDoesNotLeakTheCause guards the other half: recovering the
// classification must not put upstream detail back into the client message.
func TestFailoverWrapperDoesNotLeakTheCause(t *testing.T) {
	withNormalizedMode(t)

	lastErr := NewErrorWithStatusCode(
		errors.New("Insufficient balance for account acct_9f31 at https://pay.gptstore.example/recharge"),
		ErrorCodeBadResponseStatusCode,
		http.StatusPaymentRequired,
	)
	lastErr.SetErrorSource(ErrorSourceChannel)

	wrapped := NewUpstreamExhaustedError(lastErr, 2)
	projected := wrapped.ToPublicOpenAIError()

	for _, needle := range []string{"acct_9f31", "gptstore", "Insufficient balance", "https://"} {
		if strings.Contains(projected.Message, needle) {
			t.Errorf("projected message leaks %q: %s", needle, projected.Message)
		}
	}
	if projected.Message != publicMessageServiceUnavailable {
		t.Errorf("projected message = %q, want the generic constant", projected.Message)
	}
	if wrapped.ErrorRef() == "" {
		t.Error("error ref should still be available internally")
	}
}
