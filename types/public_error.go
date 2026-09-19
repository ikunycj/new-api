package types

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// Public error projection.
//
// The gateway classifies every upstream failure into a stable six-digit
// alltoken_code (see error_catalog.go). That classification, the channel that
// produced it, and the raw upstream message are all useful internally and are
// kept untouched: logs, traces, circuit breaking and observability still see
// the full record.
//
// None of it is useful to the caller, and some of it is actively harmful. An
// upstream saying "insufficient balance" or "no available accounts" describes
// our supplier's operational state, and an error_ref such as 204001-CH38
// enumerates the channel pool. This file projects the internal error onto a
// small set of client-facing shapes.
//
// Which problems survive the projection is the important part of the design:
//
//   - Failures caused by the caller (malformed request, their own quota,
//     content policy, unsupported model) are reported truthfully. Blurring
//     them only converts a self-service fix into a support ticket.
//   - Failures caused by the upstream (exhausted balance, empty pool,
//     credential failure, 5xx, timeout) collapse into a generic 5xx. The
//     caller cannot act on the difference, and we would rather they did not
//     learn which one it was.
//   - Rate limiting keeps its 429. Clients need the backoff signal, and the
//     status is already visible in the legacy contract.

// PublicErrorOptions carries per-response context into the projection.
type PublicErrorOptions struct {
	// MessageSuffix is appended to the client-facing message, typically the
	// request id so a support request can be traced without exposing anything
	// about the failure itself.
	MessageSuffix string
}

// PublicErrorCategories is the closed set of failure shapes a client can
// observe.
type PublicErrorCategory string

const (
	PublicErrorCategoryInvalidRequest PublicErrorCategory = "invalid_request"
	PublicErrorCategoryAuthentication PublicErrorCategory = "authentication"
	PublicErrorCategoryQuota          PublicErrorCategory = "quota"
	PublicErrorCategoryRateLimit      PublicErrorCategory = "rate_limit"
	PublicErrorCategoryPolicy         PublicErrorCategory = "policy"
	PublicErrorCategoryModel          PublicErrorCategory = "model"
	PublicErrorCategoryServer         PublicErrorCategory = "server"
)

// Generic messages. These are constants on purpose: a message that varied with
// the underlying failure would leak the classification the projection exists to
// hide.
const (
	publicMessageServiceUnavailable = "Service temporarily unavailable, please retry later"
	publicMessageRateLimited        = "Rate limit exceeded, please retry later"
	publicMessageModelUnavailable   = "The requested model is not available"
)

// publicExposedCodes are the classification codes whose message may be shown to
// the caller verbatim, because the failure is theirs to fix.
//
// Everything else — including the whole upstream range — is replaced by a
// generic message. The six-digit code itself is never echoed, so this set also
// decides which "code" values survive.
var publicExposedCodes = map[ErrorCode]struct{}{
	// The caller's request was malformed, unsupported, or outside their access.
	ErrorCodeInvalidRequest:         {},
	ErrorCodeBadRequestBody:         {},
	ErrorCodeInvalidApiType:         {},
	ErrorCodeReadRequestBodyFailed:  {},
	ErrorCodeConvertRequestFailed:   {},
	ErrorCodeSensitiveWordsDetected: {},
	ErrorCodeViolationFeeGrokCSAM:   {},
	ErrorCodePromptBlocked:          {},
	ErrorCodeModelNotFound:          {},
	ErrorCodeAccessDenied:           {},
	// The caller's own quota, as opposed to the upstream account's balance.
	ErrorCodeInsufficientUserQuota:      {},
	ErrorCodePreConsumeTokenQuotaFailed: {},
}

// publicProjection is the mode-independent decision for one error: what the
// caller is told, and with which HTTP status.
type publicProjection struct {
	StatusCode int
	Category   PublicErrorCategory
	Type       string
	Message    string
	Code       any
}

// ToPublicOpenAIError projects the error for OpenAI-compatible endpoints.
//
// In passthrough mode this is exactly ToOpenAIError(), so callers can route
// every response through this method without branching on the mode.
func (e *NewAPIError) ToPublicOpenAIError(opts ...PublicErrorOptions) OpenAIError {
	if e == nil {
		return OpenAIError{}
	}
	if !common.IsPublicErrorNormalized() {
		return e.ToOpenAIError()
	}
	projected := e.publicProjection()
	return OpenAIError{
		Message: applyPublicErrorSuffix(projected.Message, opts),
		Type:    projected.Type,
		Code:    projected.Code,
		Param:   "",
	}
}

// ToPublicClaudeError projects the error for the Anthropic message format.
func (e *NewAPIError) ToPublicClaudeError(opts ...PublicErrorOptions) ClaudeError {
	if e == nil {
		return ClaudeError{}
	}
	if !common.IsPublicErrorNormalized() {
		return e.ToClaudeError()
	}
	projected := e.publicProjection()
	return ClaudeError{
		Type:    projected.Type,
		Message: applyPublicErrorSuffix(projected.Message, opts),
	}
}

// PublicStatusCode returns the HTTP status the caller should see.
//
// A per-channel status_code_mapping rewrites the upstream status for internal
// tuning; that rewrite must not become a client-visible contract, so the
// projected status is recomputed from the classification instead.
func (e *NewAPIError) PublicStatusCode(opts ...PublicErrorOptions) int {
	if e == nil {
		return http.StatusInternalServerError
	}
	if !common.IsPublicErrorNormalized() {
		return e.StatusCode
	}
	return e.publicProjection().StatusCode
}

// publicProjection derives the client-facing shape from the internal
// classification. It never reads the upstream message.
func (e *NewAPIError) publicProjection() publicProjection {
	category := publicCategoryFor(e)
	statusCode, errType := publicStatusFor(category)

	// Failures the caller can act on keep their own message and code.
	if _, exposed := publicExposedCodes[e.errorCode]; exposed {
		return publicProjection{
			StatusCode: publicExposedStatus(e.StatusCode, category),
			Category:   category,
			Type:       publicTypeFor(e, category),
			Message:    common.MaskSensitiveInfo(e.Error()),
			Code:       string(e.errorCode),
		}
	}

	return publicProjection{
		StatusCode: statusCode,
		Category:   category,
		Type:       errType,
		Message:    publicMessageFor(category),
	}
}

// publicExposedStatus decides the status for an error the caller can act on.
//
// The original status is kept when it already describes a client-side failure.
// Anything else is replaced by the category default: a caller-facing error must
// never inherit a 5xx that the gateway happened to attach internally.
func publicExposedStatus(original int, category PublicErrorCategory) int {
	if original >= 400 && original < 500 {
		return original
	}
	statusCode, _ := publicStatusFor(category)
	return statusCode
}

func applyPublicErrorSuffix(message string, opts []PublicErrorOptions) string {
	if len(opts) == 0 || strings.TrimSpace(opts[0].MessageSuffix) == "" {
		return message
	}
	return message + " (" + strings.TrimSpace(opts[0].MessageSuffix) + ")"
}

// publicCategoryFor maps the internal classification onto the small public set.
//
// The mapping runs on the six-digit alltoken_code rather than the HTTP status,
// because the same status can arrive from different layers with very different
// meanings: a 401 from an upstream credential failure must not be shown as the
// caller's own authentication problem.
func publicCategoryFor(e *NewAPIError) PublicErrorCategory {
	// After failover gives up, the error handed to the caller is the
	// upstream_exhausted wrapper, whose own classification only says "our
	// supply ran out". The wrapper carries the last real upstream failure as
	// its cause, and that is the classification the caller must be judged on:
	// a rate limit that survived failover is still a rate limit, and must keep
	// its 429 rather than collapsing into the generic 503.
	if cause := e.cause; cause != nil && cause.AlltokenCode > 0 && e.AlltokenCode() == 305001 {
		code := cause.AlltokenCode
		return publicCategoryForCode(code, ErrorCode(cause.RawCode))
	}
	return publicCategoryForCode(e.AlltokenCode(), e.errorCode)
}

// publicCategoryForCode maps a classification onto the public set.
//
// rawCode carries the original classification code so the specific cases that
// are not decidable from the six-digit range alone (policy, the caller's own
// quota, model availability) still resolve correctly when the category is
// derived from a failover cause.
func publicCategoryForCode(code int, rawCode ErrorCode) PublicErrorCategory {

	// Content policy is checked first so a policy failure is never folded into
	// a generic request error.
	switch rawCode {
	case ErrorCodeSensitiveWordsDetected, ErrorCodeViolationFeeGrokCSAM, ErrorCodePromptBlocked:
		return PublicErrorCategoryPolicy
	case ErrorCodeInsufficientUserQuota, ErrorCodePreConsumeTokenQuotaFailed:
		return PublicErrorCategoryQuota
	case ErrorCodeAccessDenied:
		return PublicErrorCategoryAuthentication
	case ErrorCodeModelNotFound:
		return PublicErrorCategoryModel
	}

	switch {
	// Rate limiting, from either layer. Kept distinct because clients need the
	// backoff signal.
	case code == 104001, code == 204001:
		return PublicErrorCategoryRateLimit

	// Request-shape errors (301xxx) and protocol conversion errors (308xxx)
	// are the caller's to fix.
	case code >= 301000 && code < 302000:
		return PublicErrorCategoryInvalidRequest
	case code >= 308000 && code < 309000:
		return PublicErrorCategoryInvalidRequest

	// Gateway authentication (302xxx) and caller quota (303xxx).
	case code >= 302000 && code < 303000:
		return PublicErrorCategoryAuthentication
	case code >= 303000 && code < 304000:
		return PublicErrorCategoryQuota

	case code == 311001:
		return PublicErrorCategoryModel

	// Everything that describes our own supply: upstream quota exhaustion,
	// channel failures, exhausted pools, credential failures, upstream 5xx,
	// network errors and timeouts.
	default:
		return PublicErrorCategoryServer
	}
}

// publicStatusFor returns the status and error type for a category.
func publicStatusFor(category PublicErrorCategory) (int, string) {
	switch category {
	case PublicErrorCategoryInvalidRequest:
		return http.StatusBadRequest, "invalid_request_error"
	case PublicErrorCategoryAuthentication:
		return http.StatusUnauthorized, "authentication_error"
	case PublicErrorCategoryQuota:
		return http.StatusPaymentRequired, "insufficient_quota"
	case PublicErrorCategoryRateLimit:
		return http.StatusTooManyRequests, "rate_limit_error"
	case PublicErrorCategoryPolicy:
		return http.StatusBadRequest, "invalid_request_error"
	case PublicErrorCategoryModel:
		return http.StatusNotFound, "invalid_request_error"
	default:
		// Exhausted supply is a transient condition for the caller: the same
		// request may succeed later, once a channel recovers. 503 says that
		// more accurately than 502, which implies a broken upstream.
		return http.StatusServiceUnavailable, "server_error"
	}
}

// publicMessageFor returns the constant message for a projected failure.
//
// The upstream's own text is never included. It routinely describes our supply
// ("Insufficient account balance", "No available accounts") and occasionally
// echoes request content back.
func publicMessageFor(category PublicErrorCategory) string {
	switch category {
	case PublicErrorCategoryRateLimit:
		return publicMessageRateLimited
	case PublicErrorCategoryModel:
		return publicMessageModelUnavailable
	default:
		return publicMessageServiceUnavailable
	}
}

// publicTypeFor keeps the error type stable across the projection. For exposed
// errors the caller already sees the classification, so a more specific type is
// safe; otherwise the category default is used.
func publicTypeFor(e *NewAPIError, category PublicErrorCategory) string {
	if e.errorType == ErrorTypeClaudeError {
		if claudeError, ok := e.RelayError.(ClaudeError); ok && strings.TrimSpace(claudeError.Type) != "" {
			return claudeError.Type
		}
	}
	_, errType := publicStatusFor(category)
	return errType
}
