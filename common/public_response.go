package common

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Public channel identities are display placeholders, never routing identifiers.
const (
	PublicChannelID   = 1
	PublicChannelName = "auto"
	PublicErrorCode   = "service_error"
)

func DefaultPublicChannelErrorMessage() string {
	message := "服务暂异常，大概率是临时错误，请重试，如多次重试无效，请联系管理员"
	OptionMapRWMutex.RLock()
	contact := strings.TrimSpace(OptionMap["CustomerService"])
	OptionMapRWMutex.RUnlock()
	if contact != "" {
		message += "\n" + contact
	}
	return message
}

// PublicSunoData replaces provider media URLs with authenticated gateway
// resource paths. The original task.Data remains unchanged in storage.
func PublicSunoData(taskID string, data json.RawMessage) (json.RawMessage, error) {
	projected, err := PublicTaskJSON(data)
	if err != nil || len(projected) == 0 {
		return projected, err
	}
	var value any
	if err := Unmarshal(projected, &value); err != nil {
		return nil, err
	}
	var rewrite func(any)
	rewrite = func(node any) {
		switch current := node.(type) {
		case []any:
			for _, child := range current {
				rewrite(child)
			}
		case map[string]any:
			clipID, _ := current["id"].(string)
			if clipID != "" && taskID != "" {
				for kind, field := range map[string]string{
					"audio": "audio_url", "video": "video_url", "image": "image_url", "image_large": "image_large_url",
				} {
					if raw, ok := current[field].(string); ok && strings.TrimSpace(raw) != "" {
						current[field] = "/suno/media/" + url.PathEscape(taskID) + "/" + url.PathEscape(clipID) + "/" + kind
					}
				}
			}
			for _, child := range current {
				rewrite(child)
			}
		}
	}
	rewrite(value)
	return Marshal(value)
}

// PublicVideoData removes provider video URLs from task payloads while keeping
// the existing public task resource endpoint usable.
func PublicVideoData(taskID string, data json.RawMessage) (json.RawMessage, error) {
	projected, err := PublicTaskJSON(data)
	if err != nil || len(projected) == 0 {
		return projected, err
	}
	var value any
	if err := Unmarshal(projected, &value); err != nil {
		return nil, err
	}
	var rewrite func(any)
	rewrite = func(node any) {
		switch current := node.(type) {
		case []any:
			for _, child := range current {
				rewrite(child)
			}
		case map[string]any:
			for key, child := range current {
				switch key {
				case "video_url", "videoUrl":
					if raw, ok := child.(string); ok && strings.TrimSpace(raw) != "" {
						current[key] = "/v1/videos/" + url.PathEscape(taskID) + "/content"
					}
				default:
					rewrite(child)
				}
			}
		}
	}
	rewrite(value)
	return Marshal(value)
}

// PublicTaskJSON projects provider task metadata, without altering stored bytes.
// Prompt/text/input subtrees belong to the user and are not channel metadata.
// RawMessage preserves numeric and opaque protocol identifiers exactly.
func PublicTaskJSON(data json.RawMessage) (json.RawMessage, error) {
	if len(data) == 0 {
		return nil, nil
	}
	switch GetJsonType(data) {
	case "object":
		var fields map[string]json.RawMessage
		if err := Unmarshal(data, &fields); err != nil {
			return nil, fmt.Errorf("decode task metadata: %w", err)
		}
		status := strings.ToLower(JsonRawMessageToString(fields["status"]))
		statusCode := publicStatusCode(fields)
		failed := status == "failed" || status == "failure" || status == "error"
		for key, value := range fields {
			switch key {
			case "channel_id", "channel":
				fields[key] = json.RawMessage("1")
			case "channel_name":
				fields[key] = json.RawMessage(`"auto"`)
			case "channel_type", "base_url", "proxy", "private_data", "upstream_model_name",
				"upstream_request_id", "upstream_task_id", "error_ref", "cause",
				"source_code", "stable_code", "error_source", "failure_scope", "attempt_count",
				"admin_info", "audit_info", "use_channel", "is_multi_key", "multi_key_index",
				"discordInstanceId", "discord_instance_id", "account_id", "instance_id", "provider", "vendor":
				delete(fields, key)
			case "error":
				if string(value) != "null" && string(value) != `""` {
					var errorFields map[string]json.RawMessage
					if GetJsonType(value) == "object" {
						if err := Unmarshal(value, &errorFields); err != nil {
							return nil, err
						}
					}
					code := JsonRawMessageToString(errorFields["code"])
					errorType := JsonRawMessageToString(errorFields["type"])
					message := JsonRawMessageToString(errorFields["message"])
					if GetJsonType(value) == "string" {
						message = JsonRawMessageToString(value)
					}
					errorStatusCode := publicStatusCode(errorFields)
					if errorStatusCode == 0 {
						errorStatusCode = statusCode
					}
					publicFields, _ := PublicChannelErrorFields(errorStatusCode, code, errorType, message)
					public := map[string]any{
						"code":    publicFields.Code,
						"type":    publicFields.Type,
						"message": publicFields.Message,
					}
					encoded, err := Marshal(public)
					if err != nil {
						return nil, err
					}
					fields[key] = encoded
				}
			case "fail_reason", "failReason", "error_message", "error_msg", "errorMessage":
				if string(value) != "null" && string(value) != `""` {
					_, message := PublicChannelError("", JsonRawMessageToString(value))
					encoded, err := Marshal(message)
					if err != nil {
						return nil, err
					}
					fields[key] = encoded
				}
			case "error_code", "error_type":
				if string(value) != "null" && string(value) != `""` && string(value) != "0" {
					fields[key] = json.RawMessage(`"service_error"`)
				}
			case "message", "description", "reason":
				if failed && string(value) != "null" && string(value) != `""` {
					_, message := PublicChannelError(JsonRawMessageToString(fields["code"]), JsonRawMessageToString(value))
					encoded, err := Marshal(message)
					if err != nil {
						return nil, err
					}
					fields[key] = encoded
				}
			case "input", "prompt", "prompt_en", "promptEn", "text", "lyrics", "content", "state":
				// User content, not a gateway diagnostic object.
			default:
				projected, err := PublicTaskJSON(value)
				if err != nil {
					return nil, err
				}
				fields[key] = projected
			}
		}
		return Marshal(fields)
	case "array":
		var items []json.RawMessage
		if err := Unmarshal(data, &items); err != nil {
			return nil, fmt.Errorf("decode task metadata array: %w", err)
		}
		for i, item := range items {
			projected, err := PublicTaskJSON(item)
			if err != nil {
				return nil, err
			}
			items[i] = projected
		}
		return Marshal(items)
	default:
		var scalar json.RawMessage
		if err := Unmarshal(data, &scalar); err != nil {
			return nil, fmt.Errorf("decode task metadata value: %w", err)
		}
		return append(json.RawMessage(nil), data...), nil
	}
}

func publicStatusCode(fields map[string]json.RawMessage) int {
	for _, key := range []string{"status_code", "status"} {
		value := strings.TrimSpace(JsonRawMessageToString(fields[key]))
		if value == "" {
			continue
		}
		statusCode, err := strconv.Atoi(value)
		if err == nil {
			return statusCode
		}
	}
	return 0
}
