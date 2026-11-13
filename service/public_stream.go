package service

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// ProjectPublicStreamData handles only protocol error events. Normal model
// output is returned byte-for-byte so user content and tool payloads are not
// rewritten by the channel privacy layer.
func ProjectPublicStreamData(data string) (string, bool) {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return data, false
	}
	var envelope map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(trimmed, &envelope); err != nil {
		return data, false
	}
	eventType := common.JsonRawMessageToString(envelope["type"])
	lowerType := strings.ToLower(strings.TrimSpace(eventType))
	errorEvent := lowerType == "error" || lowerType == "upstream_error" ||
		lowerType == "response.failed" || lowerType == "response.incomplete" ||
		(len(envelope["error"]) > 0 && string(envelope["error"]) != "null")
	var response map[string]json.RawMessage
	if raw := envelope["response"]; len(raw) > 0 && string(raw) != "null" {
		if err := common.Unmarshal(raw, &response); err == nil {
			var details map[string]json.RawMessage
			if nested := response["status_details"]; len(nested) > 0 {
				_ = common.Unmarshal(nested, &details)
			}
			errorEvent = errorEvent || (len(response["error"]) > 0 && string(response["error"]) != "null") ||
				(len(details["error"]) > 0 && string(details["error"]) != "null")
		}
	}
	if !errorEvent {
		return data, false
	}
	projected, err := common.PublicTaskJSON([]byte(trimmed))
	if err != nil {
		common.SysError("cannot project stream error: " + err.Error())
		return "", true
	}
	var projectedEnvelope map[string]json.RawMessage
	if err := common.Unmarshal(projected, &projectedEnvelope); err == nil {
		status := streamStatusCode(envelope)
		if rawError := envelope["error"]; len(rawError) > 0 && common.GetJsonType(rawError) == "object" {
			var errorFields map[string]json.RawMessage
			if err := common.Unmarshal(rawError, &errorFields); err == nil {
				code := common.JsonRawMessageToString(errorFields["code"])
				errorType := common.JsonRawMessageToString(errorFields["type"])
				message := common.JsonRawMessageToString(errorFields["message"])
				publicFields, _ := common.PublicChannelErrorFields(status, code, errorType, message)
				encoded, encodeErr := common.Marshal(map[string]string{
					"code":    publicFields.Code,
					"type":    publicFields.Type,
					"message": publicFields.Message,
				})
				if encodeErr == nil {
					projectedEnvelope["error"] = encoded
				}
			}
		}
		code := common.JsonRawMessageToString(envelope["code"])
		errorType := common.JsonRawMessageToString(envelope["type"])
		message := common.JsonRawMessageToString(envelope["message"])
		if code != "" || errorType != "" || message != "" {
			publicFields, _ := common.PublicChannelErrorFields(status, code, errorType, message)
			if encoded, encodeErr := common.Marshal(publicFields.Code); encodeErr == nil {
				projectedEnvelope["code"] = encoded
			}
			if encoded, encodeErr := common.Marshal(publicFields.Type); encodeErr == nil {
				projectedEnvelope["type"] = encoded
			}
			if encoded, encodeErr := common.Marshal(publicFields.Message); encodeErr == nil {
				projectedEnvelope["message"] = encoded
			}
		}
		projected, err = common.Marshal(projectedEnvelope)
		if err != nil {
			return "", true
		}
	}
	return string(projected), true
}

func streamStatusCode(envelope map[string]json.RawMessage) int {
	for _, key := range []string{"status_code", "status"} {
		value := strings.TrimSpace(common.JsonRawMessageToString(envelope[key]))
		status, err := strconv.Atoi(value)
		if err == nil {
			return status
		}
	}
	return 0
}
