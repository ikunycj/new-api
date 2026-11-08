package traj_record

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var chatClient = &client{
	name: "chat", apiFamily: "chat_completions",
	paths:      []string{"/v1/chat/completions"},
	historyKey: "messages", blobKeys: []string{"tools", "functions"},
	requestIds: func(_ *callRecord, _ map[string]json.RawMessage) string { return "" },
	summarize:  summarizeChatResponse,
}

// This is a searchable summary, not a replacement for the original response.
// finish_reason describes one choice; [DONE] describes stream termination.
func summarizeChatResponse(rec *callRecord, resp []byte) {
	type choice struct {
		Index        int     `json:"index"`
		FinishReason *string `json:"finish_reason"`
	}
	type chunk struct {
		ID      string          `json:"id"`
		Model   string          `json:"model"`
		Usage   json.RawMessage `json:"usage"`
		Choices []choice        `json:"choices"`
		Error   json.RawMessage `json:"error"`
	}
	finished := map[int]bool{}
	malformed := false
	consume := func(raw []byte) {
		var c chunk
		if common.Unmarshal(raw, &c) != nil {
			malformed = true
			return
		}
		if len(c.Error) > 0 && string(c.Error) != "null" {
			malformed = true
		}
		if c.ID != "" {
			rec.RespId = c.ID
		}
		if c.Model != "" {
			rec.RespModel = c.Model
		}
		if len(c.Usage) > 0 && string(c.Usage) != "null" {
			rec.Usage = c.Usage
		}
		for _, v := range c.Choices {
			if _, ok := finished[v.Index]; !ok {
				finished[v.Index] = false
			}
			if v.FinishReason != nil && *v.FinishReason != "" {
				finished[v.Index] = true
				if v.Index == 0 {
					rec.StopReason = *v.FinishReason
				}
			}
		}
	}
	ended := false
	if strings.HasPrefix(rec.RespContentType, "text/event-stream") {
		for payload := range ssePayloads(resp) {
			if bytes.Equal(payload, []byte("[DONE]")) {
				ended = true
			} else {
				consume(payload)
			}
		}
	} else {
		consume(resp)
		ended = true
	}
	rec.RespComplete = ended && !malformed && len(finished) > 0 && !rec.RespTruncated
	for _, done := range finished {
		rec.RespComplete = rec.RespComplete && done
	}
}
