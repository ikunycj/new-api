package types

import (
	"net/http"
	"strings"
)

type PublicErrorOptions struct {
	MessageSuffix string
}

type publicErrorKind string

const (
	publicErrorInvalidRequest publicErrorKind = "invalid_request"
	publicErrorAuthentication publicErrorKind = "authentication"
	publicErrorQuota          publicErrorKind = "quota"
	publicErrorRateLimit      publicErrorKind = "rate_limit"
	publicErrorPolicy         publicErrorKind = "policy"
	publicErrorModel          publicErrorKind = "model"
	publicErrorServer         publicErrorKind = "server"
)

func (e *NewAPIError) ToPublicOpenAIError(options ...PublicErrorOptions) OpenAIError {
	if e == nil {
		return OpenAIError{}
	}
	kind := e.publicKind()
	message, code, errType := publicErrorDetails(kind)
	if e.errorType == ErrorTypeNewAPIError && e.errorSource == ErrorSourceUnknown && publicCallerError(e.errorCode) {
		code = string(e.errorCode)
	}
	if len(options) > 0 && strings.TrimSpace(options[0].MessageSuffix) != "" {
		message += " (" + strings.TrimSpace(options[0].MessageSuffix) + ")"
	}
	return OpenAIError{Message: message, Type: errType, Code: code}
}

func (e *NewAPIError) ToPublicClaudeError(options ...PublicErrorOptions) ClaudeError {
	openAI := e.ToPublicOpenAIError(options...)
	return ClaudeError{Type: openAI.Type, Message: openAI.Message}
}

func (e *NewAPIError) PublicStatusCode() int {
	if e == nil {
		return http.StatusServiceUnavailable
	}
	kind := e.publicKind()
	if e.errorType == ErrorTypeNewAPIError && e.errorSource == ErrorSourceUnknown && publicCallerError(e.errorCode) && e.StatusCode >= 400 && e.StatusCode < 500 {
		return e.StatusCode
	}
	switch kind {
	case publicErrorInvalidRequest, publicErrorPolicy:
		return http.StatusBadRequest
	case publicErrorAuthentication:
		return http.StatusUnauthorized
	case publicErrorQuota:
		return http.StatusForbidden
	case publicErrorRateLimit:
		return http.StatusTooManyRequests
	case publicErrorModel:
		return http.StatusNotFound
	default:
		return http.StatusServiceUnavailable
	}
}

func (e *NewAPIError) publicKind() publicErrorKind {
	if e == nil {
		return publicErrorServer
	}
	if e.StatusCode == http.StatusTooManyRequests || e.ErrorCategory() == "rate_limit" {
		return publicErrorRateLimit
	}
	if e.errorCode == ErrorCodeUpstreamExhausted && e.cause != nil && e.cause.StatusCode == http.StatusTooManyRequests {
		return publicErrorRateLimit
	}
	if e.errorSource != ErrorSourceUnknown {
		return publicErrorServer
	}
	switch e.errorCode {
	case ErrorCodeSensitiveWordsDetected, ErrorCodeViolationFeeGrokCSAM, ErrorCodePromptBlocked:
		return publicErrorPolicy
	case ErrorCodeAccessDenied:
		return publicErrorAuthentication
	case ErrorCodeInsufficientUserQuota, ErrorCodePreConsumeTokenQuotaFailed:
		return publicErrorQuota
	case ErrorCodeModelNotFound:
		return publicErrorModel
	case ErrorCodeInvalidRequest, ErrorCodeBadRequestBody, ErrorCodeInvalidApiType,
		ErrorCodeReadRequestBodyFailed, ErrorCodeConvertRequestFailed:
		return publicErrorInvalidRequest
	}
	return publicErrorServer
}

func publicCallerError(code ErrorCode) bool {
	switch code {
	case ErrorCodeInvalidRequest, ErrorCodeBadRequestBody, ErrorCodeInvalidApiType,
		ErrorCodeReadRequestBodyFailed, ErrorCodeConvertRequestFailed,
		ErrorCodeSensitiveWordsDetected, ErrorCodeViolationFeeGrokCSAM,
		ErrorCodePromptBlocked, ErrorCodeAccessDenied,
		ErrorCodeInsufficientUserQuota, ErrorCodePreConsumeTokenQuotaFailed:
		return true
	default:
		return false
	}
}

func publicErrorDetails(kind publicErrorKind) (message, code, errType string) {
	switch kind {
	case publicErrorInvalidRequest:
		return "Invalid request", "invalid_request", "invalid_request_error"
	case publicErrorAuthentication:
		return "Authentication failed", "authentication_failed", "authentication_error"
	case publicErrorQuota:
		return "Insufficient quota", "insufficient_quota", "insufficient_quota"
	case publicErrorRateLimit:
		return "Rate limit exceeded, please retry later", "rate_limit_exceeded", "rate_limit_error"
	case publicErrorPolicy:
		return "Request blocked by content policy", "content_policy_violation", "invalid_request_error"
	case publicErrorModel:
		return "The requested model is not available", "model_not_found", "invalid_request_error"
	default:
		return "Service temporarily unavailable, please retry later", "service_unavailable", "server_error"
	}
}
