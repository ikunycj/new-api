package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// PublicRelayError only creates a response value; all retry/billing/logging
// consumers keep the original error and its original channel identity.
func PublicRelayError(c *gin.Context, apiErr *types.NewAPIError) types.PublicOpenAIError {
	if apiErr == nil {
		return types.PublicOpenAIError{}
	}
	options := types.PublicErrorOptions{
		Message: common.DefaultPublicChannelErrorMessage(),
		Code:    common.PublicErrorCode,
		Type:    "server_error",
	}
	if apiErr.GetErrorSource() == types.ErrorSourceUnknown && apiErr.ChannelID() == 0 {
		internal := apiErr.ToOpenAIError()
		options.Param = internal.Param
		switch apiErr.GetErrorCode() {
		case types.ErrorCodeInvalidRequest, types.ErrorCodeBadRequestBody, types.ErrorCodeReadRequestBodyFailed,
			types.ErrorCodeAccessDenied, types.ErrorCodeInsufficientUserQuota, types.ErrorCodePreConsumeTokenQuotaFailed:
			options.Message = apiErr.MaskSensitiveError()
			options.Code = string(apiErr.GetErrorCode())
			options.Type = "invalid_request_error"
		}
	} else {
		status, code, errorType, message := originalChannelFailure(apiErr)
		fields, _ := common.PublicChannelErrorFields(status, code, errorType, message)
		options.Code, options.Type, options.Message = fields.Code, fields.Type, fields.Message
	}
	view := apiErr.ToPublicOpenAIError(options)
	if c != nil {
		view.RequestID = c.GetString(common.RequestIdKey)
	}
	return view
}

func originalChannelFailure(apiErr *types.NewAPIError) (status int, code, errorType, message string) {
	internal := apiErr.ToOpenAIError()
	if original, ok := apiErr.OriginalUpstreamFailure(); ok {
		return original.StatusCode, original.Code, internal.Type, original.Message
	}
	return apiErr.StatusCode, string(apiErr.GetErrorCode()), internal.Type, apiErr.Error()
}

func PublicTaskError(taskErr *taskdto.TaskError) *taskdto.TaskError {
	if taskErr == nil {
		return nil
	}
	view := taskdto.TaskError{StatusCode: taskErr.StatusCode}
	if taskErr.LocalError && isPublicTaskLocalError(taskErr.Code) {
		view.Code, view.Message = taskErr.Code, taskErr.Message
		return &view
	}
	fields, _ := common.PublicChannelErrorFields(taskErr.StatusCode, taskErr.Code, "", taskErr.Message)
	view.Code, view.Message = fields.Code, fields.Message
	return &view
}

func isPublicTaskLocalError(code string) bool {
	switch code {
	case "invalid_request", "task_not_exist", "access_denied", "insufficient_user_quota", "pre_consume_token_quota_failed":
		return true
	default:
		return false
	}
}

// WritePublicRelayError handles errors both before and after response commitment.
// It never appends JSON to binary output or tries to change a committed status.
func WritePublicRelayError(c *gin.Context, format types.RelayFormat, ws *websocket.Conn, apiErr *types.NewAPIError) {
	if apiErr == nil || c.Request.Context().Err() != nil || c.GetBool("public_relay_error_written") {
		return
	}
	view := PublicRelayError(c, apiErr)
	status := apiErr.StatusCode
	if status < 100 || status > 599 {
		status = http.StatusBadGateway
	}
	var body any = gin.H{"error": view}
	claude := types.ClaudeError{Type: view.Type, Message: view.Message}
	if format == types.RelayFormatClaude {
		if claude.Type == "server_error" {
			claude.Type = "api_error"
		}
		body = gin.H{"type": "error", "error": claude}
	} else if format == types.RelayFormatGemini {
		body = gin.H{"error": gin.H{"code": status, "message": view.Message, "status": geminiPublicErrorStatus(status)}}
	}
	if ws != nil {
		body = gin.H{"type": "error", "error": view}
		encoded, err := common.Marshal(body)
		if err != nil {
			logger.LogError(c, fmt.Sprintf("marshal public websocket error: %v", err))
			return
		}
		if err := ws.WriteMessage(websocket.TextMessage, encoded); err != nil {
			logger.LogError(c, fmt.Sprintf("write public websocket error: %v", err))
		}
		c.Set("public_relay_error_written", true)
		return
	}
	if !c.Writer.Written() {
		c.Writer.Header().Del("Content-Length")
		c.Writer.Header().Del("Content-Encoding")
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(status, body)
		c.Set("public_relay_error_written", true)
		return
	}
	if !strings.HasPrefix(c.Writer.Header().Get("Content-Type"), "text/event-stream") {
		logger.LogError(c, "relay failed after non-SSE response started")
		return
	}
	if format == types.RelayFormatOpenAIResponses || format == types.RelayFormatOpenAIResponsesCompaction {
		body = gin.H{"type": "error", "code": view.Code, "message": view.Message, "param": view.Param, "request_id": view.RequestID}
	}
	encoded, err := common.Marshal(body)
	if err != nil {
		logger.LogError(c, fmt.Sprintf("marshal public stream error: %v", err))
		return
	}
	event := ""
	if format == types.RelayFormatClaude || format == types.RelayFormatOpenAIResponses || format == types.RelayFormatOpenAIResponsesCompaction {
		event = "event: error\n"
	}
	if _, err = c.Writer.Write([]byte(event + "data: " + string(encoded) + "\n\n")); err != nil {
		logger.LogError(c, fmt.Sprintf("write public stream error: %v", err))
		return
	}
	c.Writer.Flush()
	c.Set("public_relay_error_written", true)
}

func geminiPublicErrorStatus(status int) string {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return "INVALID_ARGUMENT"
	case http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case http.StatusForbidden:
		return "PERMISSION_DENIED"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusTooManyRequests:
		return "RESOURCE_EXHAUSTED"
	case http.StatusServiceUnavailable:
		return "UNAVAILABLE"
	default:
		return "INTERNAL"
	}
}
