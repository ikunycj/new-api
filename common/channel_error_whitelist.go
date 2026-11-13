package common

import (
	"fmt"
	"strconv"
	"strings"
)

const ChannelErrorWhitelistOptionKey = "console_setting.channel_error_whitelist"

type ChannelErrorRule struct {
	MatchType     string `json:"match_type"`
	MatchValue    string `json:"match_value"`
	ResponseMode  string `json:"response_mode"`
	CustomCode    string `json:"custom_code,omitempty"`
	CustomType    string `json:"custom_type,omitempty"`
	CustomMessage string `json:"custom_message,omitempty"`
}

type PublicErrorFields struct {
	Code    string
	Type    string
	Message string
}

// PublicChannelErrorFields projects a channel error only after an explicit
// whitelist rule matches. Status rules match the final HTTP status exactly.
// Error rules use case-insensitive substring matching against any of the
// upstream error object's code, type, or message fields.
func PublicChannelErrorFields(status int, code, errorType, message string) (fields PublicErrorFields, matched bool) {
	fields = PublicErrorFields{
		Code:    PublicErrorCode,
		Type:    "server_error",
		Message: DefaultPublicChannelErrorMessage(),
	}

	rules := loadChannelErrorRules()
	for _, rule := range rules {
		if !channelErrorRuleMatches(rule, status, code, errorType, message) {
			continue
		}
		matched = true
		switch rule.ResponseMode {
		case "passthrough":
			fields = PublicErrorFields{
				Code:    firstNonEmpty(code, PublicErrorCode),
				Type:    firstNonEmpty(errorType, "server_error"),
				Message: firstNonEmpty(message, DefaultPublicChannelErrorMessage()),
			}
		case "custom":
			fields = PublicErrorFields{
				Code:    firstNonEmpty(rule.CustomCode, PublicErrorCode),
				Type:    firstNonEmpty(rule.CustomType, "server_error"),
				Message: firstNonEmpty(rule.CustomMessage, DefaultPublicChannelErrorMessage()),
			}
		}
		return fields, matched
	}
	return fields, false
}

// PublicChannelError keeps the legacy two-field projection API for task and
// log paths that do not carry an HTTP status or error type.
func PublicChannelError(code, message string) (publicCode, publicMessage string) {
	fields, _ := PublicChannelErrorFields(0, code, "", message)
	return fields.Code, fields.Message
}

func loadChannelErrorRules() []ChannelErrorRule {
	OptionMapRWMutex.RLock()
	raw := OptionMap[ChannelErrorWhitelistOptionKey]
	OptionMapRWMutex.RUnlock()
	if raw == "" {
		return nil
	}
	var rules []ChannelErrorRule
	if err := UnmarshalJsonStr(raw, &rules); err != nil {
		SysError(fmt.Sprintf("invalid channel error whitelist: %v", err))
		return nil
	}
	return rules
}

func channelErrorRuleMatches(rule ChannelErrorRule, status int, code, errorType, message string) bool {
	switch rule.MatchType {
	case "status":
		configuredStatus, err := strconv.Atoi(strings.TrimSpace(rule.MatchValue))
		return err == nil && configuredStatus == status
	case "error":
		needle := strings.ToLower(strings.TrimSpace(rule.MatchValue))
		if needle == "" {
			return false
		}
		return strings.Contains(strings.ToLower(code), needle) ||
			strings.Contains(strings.ToLower(errorType), needle) ||
			strings.Contains(strings.ToLower(message), needle)
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
