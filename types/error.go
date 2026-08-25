package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type OpenAIError struct {
	Message      string          `json:"message"`
	Type         string          `json:"type"`
	Param        string          `json:"param"`
	Code         any             `json:"code"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	Source       ErrorSource     `json:"source,omitempty"`
	SourceCode   string          `json:"source_code,omitempty"`
	Retryable    *bool           `json:"retryable,omitempty"`
	RequestID    string          `json:"request_id,omitempty"`
	AttemptCount int             `json:"attempt_count,omitempty"`
	StableCode   int             `json:"stable_code,omitempty"`
	ErrorRef     string          `json:"error_ref,omitempty"`
	Category     string          `json:"category,omitempty"`
	ChannelID    int             `json:"channel_id,omitempty"`
	ChannelName  string          `json:"channel_name,omitempty"`
	FailureScope string          `json:"failure_scope,omitempty"`
	Action       string          `json:"action,omitempty"`
	Cause        *ErrorCause     `json:"cause,omitempty"`
}

type ErrorSource string

const (
	ErrorSourceUnknown ErrorSource = ""
	ErrorSourceOpenAI  ErrorSource = "openai"
	ErrorSourceChannel ErrorSource = "channel"
)

type ErrorCause struct {
	Source      ErrorSource `json:"source,omitempty"`
	Code        string      `json:"code"`
	RawCode     string      `json:"raw_code,omitempty"`
	StatusCode  int         `json:"status_code,omitempty"`
	StableCode  int         `json:"stable_code,omitempty"`
	ErrorRef    string      `json:"error_ref,omitempty"`
	ChannelID   int         `json:"channel_id,omitempty"`
	ChannelName string      `json:"channel_name,omitempty"`
}

type ClaudeError struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

type ErrorType string

const (
	ErrorTypeNewAPIError     ErrorType = "new_api_error"
	ErrorTypeOpenAIError     ErrorType = "openai_error"
	ErrorTypeClaudeError     ErrorType = "claude_error"
	ErrorTypeMidjourneyError ErrorType = "midjourney_error"
	ErrorTypeGeminiError     ErrorType = "gemini_error"
	ErrorTypeRerankError     ErrorType = "rerank_error"
	ErrorTypeUpstreamError   ErrorType = "upstream_error"
)

type ErrorCode string

const (
	ErrorCodeInvalidRequest         ErrorCode = "invalid_request"
	ErrorCodeSensitiveWordsDetected ErrorCode = "sensitive_words_detected"
	ErrorCodeViolationFeeGrokCSAM   ErrorCode = "violation_fee.grok.csam"

	// new api error
	ErrorCodeCountTokenFailed   ErrorCode = "count_token_failed"
	ErrorCodeModelPriceError    ErrorCode = "model_price_error"
	ErrorCodeInvalidApiType     ErrorCode = "invalid_api_type"
	ErrorCodeJsonMarshalFailed  ErrorCode = "json_marshal_failed"
	ErrorCodeDoRequestFailed    ErrorCode = "do_request_failed"
	ErrorCodeGetChannelFailed   ErrorCode = "get_channel_failed"
	ErrorCodeGenRelayInfoFailed ErrorCode = "gen_relay_info_failed"

	// channel error
	ErrorCodeChannelNoAvailableKey        ErrorCode = "channel:no_available_key"
	ErrorCodeChannelParamOverrideInvalid  ErrorCode = "channel:param_override_invalid"
	ErrorCodeChannelHeaderOverrideInvalid ErrorCode = "channel:header_override_invalid"
	ErrorCodeChannelModelMappedError      ErrorCode = "channel:model_mapped_error"
	ErrorCodeChannelAwsClientError        ErrorCode = "channel:aws_client_error"
	ErrorCodeChannelInvalidKey            ErrorCode = "channel:invalid_key"
	ErrorCodeChannelResponseTimeExceeded  ErrorCode = "channel:response_time_exceeded"

	// client request error
	ErrorCodeReadRequestBodyFailed ErrorCode = "read_request_body_failed"
	ErrorCodeConvertRequestFailed  ErrorCode = "convert_request_failed"
	ErrorCodeAccessDenied          ErrorCode = "access_denied"

	// request error
	ErrorCodeBadRequestBody ErrorCode = "bad_request_body"

	// response error
	ErrorCodeReadResponseBodyFailed ErrorCode = "read_response_body_failed"
	ErrorCodeBadResponseStatusCode  ErrorCode = "bad_response_status_code"
	ErrorCodeBadResponse            ErrorCode = "bad_response"
	ErrorCodeBadResponseBody        ErrorCode = "bad_response_body"
	ErrorCodeEmptyResponse          ErrorCode = "empty_response"
	ErrorCodeAwsInvokeError         ErrorCode = "aws_invoke_error"
	ErrorCodeModelNotFound          ErrorCode = "model_not_found"
	ErrorCodePromptBlocked          ErrorCode = "prompt_blocked"
	ErrorCodeUpstreamExhausted      ErrorCode = "upstream_exhausted"

	// sql error
	ErrorCodeQueryDataError  ErrorCode = "query_data_error"
	ErrorCodeUpdateDataError ErrorCode = "update_data_error"

	// quota error
	ErrorCodeInsufficientUserQuota      ErrorCode = "insufficient_user_quota"
	ErrorCodePreConsumeTokenQuotaFailed ErrorCode = "pre_consume_token_quota_failed"
)

type NewAPIError struct {
	Err            error
	RelayError     any
	skipRetry      bool
	recordErrorLog *bool
	errorType      ErrorType
	errorCode      ErrorCode
	StatusCode     int
	Metadata       json.RawMessage
	errorSource    ErrorSource
	retryable      *bool
	requestID      string
	attemptCount   int
	cause          *ErrorCause
	channelID      int
	channelName    string
	classification *errorDefinition
}

// Unwrap enables errors.Is / errors.As to work with NewAPIError by exposing the underlying error.
func (e *NewAPIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *NewAPIError) GetErrorCode() ErrorCode {
	if e == nil {
		return ""
	}
	return e.errorCode
}

func (e *NewAPIError) GetErrorType() ErrorType {
	if e == nil {
		return ""
	}
	return e.errorType
}

func ParseErrorSource(source string) ErrorSource {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case string(ErrorSourceOpenAI):
		return ErrorSourceOpenAI
	case string(ErrorSourceChannel):
		return ErrorSourceChannel
	default:
		return ErrorSourceUnknown
	}
}

// ResolveErrorSource applies the explicit channel setting first, then uses a
// conservative default for OpenAI-compatible endpoints. Direct api.openai.com
// traffic is attributed to OpenAI; other compatible upstreams are attributed
// to the configured channel layer. Locally generated errors do not have an
// upstream source and never call this function.
func ResolveErrorSource(configured, baseURL string) ErrorSource {
	if source := ParseErrorSource(configured); source == ErrorSourceOpenAI || source == ErrorSourceChannel {
		return source
	}
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err == nil && strings.EqualFold(parsed.Hostname(), "api.openai.com") {
		return ErrorSourceOpenAI
	}
	return ErrorSourceChannel
}

func (e *NewAPIError) GetErrorSource() ErrorSource {
	if e == nil {
		return ErrorSourceUnknown
	}
	return e.errorSource
}

func (e *NewAPIError) SetErrorSource(source ErrorSource) {
	if e == nil || !isSupportedErrorSource(source) {
		return
	}
	e.errorSource = source
}

func isSupportedErrorSource(source ErrorSource) bool {
	return source == ErrorSourceOpenAI || source == ErrorSourceChannel
}

func (e *NewAPIError) EnsureErrorSource(source ErrorSource) {
	if e == nil || e.errorSource != ErrorSourceUnknown {
		return
	}
	e.SetErrorSource(source)
}

func (e *NewAPIError) SourceCode() string {
	if e == nil {
		return ""
	}
	code := strings.TrimSpace(string(e.errorCode))
	if code == "" {
		code = "unknown_error"
	}
	code = strings.ReplaceAll(code, ":", ".")
	if e.errorSource == ErrorSourceUnknown {
		return code
	}
	prefix := string(e.errorSource) + "."
	if strings.HasPrefix(strings.ToLower(code), strings.ToLower(prefix)) {
		return code
	}
	return prefix + code
}

func (e *NewAPIError) SetRetryable(retryable bool) {
	if e == nil {
		return
	}
	e.retryable = common.GetPointer(retryable)
}

func (e *NewAPIError) IsRetryable() bool {
	return e != nil && e.retryable != nil && *e.retryable
}

func (e *NewAPIError) HasRetryable() bool {
	return e != nil && e.retryable != nil
}

func (e *NewAPIError) SetRequestID(requestID string) {
	if e != nil {
		e.requestID = requestID
	}
}

func (e *NewAPIError) SetAttemptCount(attemptCount int) {
	if e != nil && attemptCount > 0 {
		e.attemptCount = attemptCount
	}
}

func (e *NewAPIError) SetChannelLocation(channelID int, channelName string) {
	if e == nil {
		return
	}
	if channelID > 0 {
		e.channelID = channelID
	}
	e.channelName = strings.TrimSpace(channelName)
}

func (e *NewAPIError) SetClassification(stableCode int, category string, failureScope string, action string, retryable bool) {
	if e == nil || stableCode < 100000 || stableCode > 999999 {
		return
	}
	e.classification = &errorDefinition{
		Code:         stableCode,
		Category:     strings.TrimSpace(category),
		FailureScope: strings.TrimSpace(failureScope),
		Action:       strings.TrimSpace(action),
	}
	e.SetRetryable(retryable)
}

func (e *NewAPIError) StableCode() int {
	return classifyError(e).Code
}

func (e *NewAPIError) ErrorRef() string {
	return buildErrorRef(e.StableCode(), e.channelID)
}

func (e *NewAPIError) ChannelID() int {
	if e == nil {
		return 0
	}
	return e.channelID
}

func (e *NewAPIError) ChannelName() string {
	if e == nil {
		return ""
	}
	return e.channelName
}

func (e *NewAPIError) ErrorCategory() string {
	return classifyError(e).Category
}

func (e *NewAPIError) FailureScope() string {
	return classifyError(e).FailureScope
}

func (e *NewAPIError) ErrorAction() string {
	return classifyError(e).Action
}

func (e *NewAPIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		// fallback message when underlying error is missing
		return string(e.errorCode)
	}
	return e.Err.Error()
}

func (e *NewAPIError) ErrorWithStatusCode() string {
	if e == nil {
		return ""
	}
	msg := e.Error()
	if e.StatusCode == 0 {
		return msg
	}
	if msg == "" {
		return fmt.Sprintf("status_code=%d", e.StatusCode)
	}
	return fmt.Sprintf("status_code=%d, %s", e.StatusCode, msg)
}

func (e *NewAPIError) MaskSensitiveError() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return string(e.errorCode)
	}
	errStr := e.Err.Error()
	if e.errorCode == ErrorCodeCountTokenFailed {
		return errStr
	}
	return common.MaskSensitiveInfo(errStr)
}

func (e *NewAPIError) MaskSensitiveErrorWithStatusCode() string {
	if e == nil {
		return ""
	}
	msg := e.MaskSensitiveError()
	if e.StatusCode == 0 {
		return msg
	}
	if msg == "" {
		return fmt.Sprintf("status_code=%d", e.StatusCode)
	}
	return fmt.Sprintf("status_code=%d, %s", e.StatusCode, msg)
}

func (e *NewAPIError) SetMessage(message string) {
	e.Err = errors.New(message)
}

func (e *NewAPIError) ToOpenAIError() OpenAIError {
	var result OpenAIError
	switch e.errorType {
	case ErrorTypeOpenAIError:
		if openAIError, ok := e.RelayError.(OpenAIError); ok {
			result = openAIError
		}
	case ErrorTypeClaudeError:
		if claudeError, ok := e.RelayError.(ClaudeError); ok {
			result = OpenAIError{
				Message: e.Error(),
				Type:    claudeError.Type,
				Param:   "",
				Code:    e.errorCode,
			}
		}
	default:
		result = OpenAIError{
			Message: e.Error(),
			Type:    string(e.errorType),
			Param:   "",
			Code:    e.errorCode,
		}
	}
	if e.errorCode != ErrorCodeCountTokenFailed {
		result.Message = common.MaskSensitiveInfo(result.Message)
	}
	if result.Message == "" {
		result.Message = string(e.errorType)
	}
	result.Source = e.errorSource
	result.SourceCode = e.SourceCode()
	result.Retryable = e.retryable
	result.RequestID = e.requestID
	result.AttemptCount = e.attemptCount
	definition := classifyError(e)
	result.StableCode = definition.Code
	result.ErrorRef = buildErrorRef(definition.Code, e.channelID)
	result.Category = definition.Category
	result.ChannelID = e.channelID
	result.ChannelName = e.channelName
	result.FailureScope = definition.FailureScope
	result.Action = definition.Action
	result.Cause = e.cause
	return result
}

func (e *NewAPIError) ToClaudeError() ClaudeError {
	var result ClaudeError
	switch e.errorType {
	case ErrorTypeOpenAIError:
		if openAIError, ok := e.RelayError.(OpenAIError); ok {
			result = ClaudeError{
				Message: e.Error(),
				Type:    fmt.Sprintf("%v", openAIError.Code),
			}
		}
	case ErrorTypeClaudeError:
		if claudeError, ok := e.RelayError.(ClaudeError); ok {
			result = claudeError
		}
	default:
		result = ClaudeError{
			Message: e.Error(),
			Type:    string(e.errorType),
		}
	}
	if e.errorCode != ErrorCodeCountTokenFailed {
		result.Message = common.MaskSensitiveInfo(result.Message)
	}
	if result.Message == "" {
		result.Message = string(e.errorType)
	}
	return result
}

type NewAPIErrorOptions func(*NewAPIError)

func NewError(err error, errorCode ErrorCode, ops ...NewAPIErrorOptions) *NewAPIError {
	var newErr *NewAPIError
	// 保留深层传递的 new err
	if errors.As(err, &newErr) {
		for _, op := range ops {
			op(newErr)
		}
		return newErr
	}
	e := &NewAPIError{
		Err:        err,
		RelayError: nil,
		errorType:  ErrorTypeNewAPIError,
		StatusCode: http.StatusInternalServerError,
		errorCode:  errorCode,
	}
	for _, op := range ops {
		op(e)
	}
	return e
}

func NewOpenAIError(err error, errorCode ErrorCode, statusCode int, ops ...NewAPIErrorOptions) *NewAPIError {
	var newErr *NewAPIError
	// 保留深层传递的 new err
	if errors.As(err, &newErr) {
		if newErr.RelayError == nil {
			openaiError := OpenAIError{
				Message: newErr.Error(),
				Type:    string(errorCode),
				Code:    errorCode,
			}
			newErr.RelayError = openaiError
		}
		for _, op := range ops {
			op(newErr)
		}
		return newErr
	}
	message := string(errorCode)
	if err != nil {
		message = err.Error()
	}
	openaiError := OpenAIError{
		Message: message,
		Type:    string(errorCode),
		Code:    errorCode,
	}
	return WithOpenAIError(openaiError, statusCode, ops...)
}

func InitOpenAIError(errorCode ErrorCode, statusCode int, ops ...NewAPIErrorOptions) *NewAPIError {
	openaiError := OpenAIError{
		Type: string(errorCode),
		Code: errorCode,
	}
	return WithOpenAIError(openaiError, statusCode, ops...)
}

func NewErrorWithStatusCode(err error, errorCode ErrorCode, statusCode int, ops ...NewAPIErrorOptions) *NewAPIError {
	e := &NewAPIError{
		Err: err,
		RelayError: OpenAIError{
			Message: err.Error(),
			Type:    string(errorCode),
		},
		errorType:  ErrorTypeNewAPIError,
		StatusCode: statusCode,
		errorCode:  errorCode,
	}
	for _, op := range ops {
		op(e)
	}

	return e
}

func WithOpenAIError(openAIError OpenAIError, statusCode int, ops ...NewAPIErrorOptions) *NewAPIError {
	code, ok := openAIError.Code.(string)
	if !ok {
		if openAIError.Code != nil {
			code = fmt.Sprintf("%v", openAIError.Code)
		} else {
			code = "unknown_error"
		}
	}
	if openAIError.Type == "" {
		openAIError.Type = "upstream_error"
	}
	source := ParseErrorSource(string(openAIError.Source))
	e := &NewAPIError{
		RelayError:  openAIError,
		errorType:   ErrorTypeOpenAIError,
		StatusCode:  statusCode,
		Err:         errors.New(openAIError.Message),
		errorCode:   ErrorCode(code),
		errorSource: source,
	}
	// OpenRouter
	if len(openAIError.Metadata) > 0 {
		openAIError.Message = fmt.Sprintf("%s (%s)", openAIError.Message, openAIError.Metadata)
		e.Metadata = openAIError.Metadata
		e.RelayError = openAIError
		e.Err = errors.New(openAIError.Message)
	}
	for _, op := range ops {
		op(e)
	}
	return e
}

func WithClaudeError(claudeError ClaudeError, statusCode int, ops ...NewAPIErrorOptions) *NewAPIError {
	if claudeError.Type == "" {
		claudeError.Type = "upstream_error"
	}
	e := &NewAPIError{
		RelayError: claudeError,
		errorType:  ErrorTypeClaudeError,
		StatusCode: statusCode,
		Err:        errors.New(claudeError.Message),
		errorCode:  ErrorCode(claudeError.Type),
	}
	for _, op := range ops {
		op(e)
	}
	return e
}

func IsChannelError(err *NewAPIError) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(string(err.errorCode), "channel:")
}

func IsSkipRetryError(err *NewAPIError) bool {
	if err == nil {
		return false
	}

	return err.skipRetry
}

func ErrOptionWithSkipRetry() NewAPIErrorOptions {
	return func(e *NewAPIError) {
		e.skipRetry = true
	}
}

func ErrOptionWithNoRecordErrorLog() NewAPIErrorOptions {
	return func(e *NewAPIError) {
		e.recordErrorLog = common.GetPointer(false)
	}
}

func ErrOptionWithStatusCode(statusCode int) NewAPIErrorOptions {
	return func(e *NewAPIError) {
		e.StatusCode = statusCode
	}
}

func ErrOptionWithErrorSource(source ErrorSource) NewAPIErrorOptions {
	return func(e *NewAPIError) {
		e.SetErrorSource(source)
	}
}

func NewUpstreamExhaustedError(lastErr *NewAPIError, attemptCount int) *NewAPIError {
	if lastErr == nil {
		return NewError(errors.New("all upstream routes failed"), ErrorCodeUpstreamExhausted)
	}
	cause := &ErrorCause{
		Source:      lastErr.GetErrorSource(),
		Code:        lastErr.SourceCode(),
		RawCode:     string(lastErr.GetErrorCode()),
		StatusCode:  lastErr.StatusCode,
		StableCode:  lastErr.StableCode(),
		ErrorRef:    lastErr.ErrorRef(),
		ChannelID:   lastErr.channelID,
		ChannelName: lastErr.channelName,
	}
	e := NewErrorWithStatusCode(
		fmt.Errorf("all upstream routes failed: %w", lastErr),
		ErrorCodeUpstreamExhausted,
		lastErr.StatusCode,
	)
	e.errorType = ErrorTypeUpstreamError
	e.errorSource = ErrorSourceUnknown
	e.attemptCount = attemptCount
	e.cause = cause
	e.channelID = lastErr.channelID
	e.channelName = lastErr.channelName
	e.SetRetryable(true)
	return e
}

func ErrOptionWithHideErrMsg(replaceStr string) NewAPIErrorOptions {
	return func(e *NewAPIError) {
		if common.DebugEnabled {
			fmt.Printf("ErrOptionWithHideErrMsg: %s, origin error: %s", replaceStr, e.Err)
		}
		e.Err = errors.New(replaceStr)
	}
}

func IsRecordErrorLog(e *NewAPIError) bool {
	if e == nil {
		return false
	}
	if e.recordErrorLog == nil {
		// default to true if not set
		return true
	}
	return *e.recordErrorLog
}
