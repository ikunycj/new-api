package types

// PublicChannelID and PublicChannelName are response-only placeholders. They
// must never be written back to routing context, logs, billing or persistence.
const (
	PublicChannelID   = 1
	PublicChannelName = "auto"
	PublicErrorCode   = "service_error"
)

type PublicErrorOptions struct {
	Message string
	Code    string
	Type    string
	Param   string
}

// PublicOpenAIError deliberately has no slots for internal diagnostics.
type PublicOpenAIError struct {
	Message     string `json:"message"`
	Type        string `json:"type"`
	Param       string `json:"param"`
	Code        string `json:"code"`
	RequestID   string `json:"request_id,omitempty"`
	ChannelID   int    `json:"channel_id,omitempty"`
	ChannelName string `json:"channel_name,omitempty"`
}

// ToPublicOpenAIError builds an independent response object. The internal
// OpenAIError remains unchanged for logging, retry decisions and billing.
func (e *NewAPIError) ToPublicOpenAIError(options PublicErrorOptions) PublicOpenAIError {
	if e == nil {
		return PublicOpenAIError{}
	}
	result := PublicOpenAIError{
		Message:   options.Message,
		Type:      options.Type,
		Param:     options.Param,
		Code:      options.Code,
		RequestID: e.requestID,
	}
	if e.channelID > 0 {
		result.ChannelID = PublicChannelID
		result.ChannelName = PublicChannelName
	}
	return result
}

func (e *NewAPIError) ToPublicClaudeError(options PublicErrorOptions) ClaudeError {
	if e == nil {
		return ClaudeError{}
	}
	return ClaudeError{Type: options.Type, Message: options.Message}
}
