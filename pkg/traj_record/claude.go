package traj_record

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// claudeClient is Claude Code on the Anthropic Messages API.
var claudeClient = &client{
	name:       "claude",
	paths:      []string{"/v1/messages"},
	uaPrefixes: []string{"claude-cli/"},
	historyKey: "messages",
	blobKeys:   []string{"system", "tools"},
	requestIds: claudeRequestIds,
	summarize:  summarizeClaudeResponse,
}

// claudeRequestIds reads the ids Claude Code packs into metadata.user_id.
func claudeRequestIds(rec *callRecord, top map[string]json.RawMessage) string {
	md := top["metadata"]
	if md == nil {
		return ""
	}
	var meta struct {
		UserId string `json:"user_id"`
	}
	_ = common.Unmarshal(md, &meta)
	var ids struct {
		SessionId   string `json:"session_id"`
		DeviceId    string `json:"device_id"`
		AccountUuid string `json:"account_uuid"`
	}
	if common.UnmarshalJsonStr(meta.UserId, &ids) == nil {
		rec.SessionId, rec.DeviceId, rec.AccountUuid = ids.SessionId, ids.DeviceId, ids.AccountUuid
	} else if rest, ok := strings.CutPrefix(meta.UserId, "user_"); ok {
		// Older Claude Code: user_<device>_account_<uuid>_session_<uuid>
		rest, rec.SessionId, _ = strings.Cut(rest, "_session_")
		rec.DeviceId, rec.AccountUuid, _ = strings.Cut(rest, "_account_")
	}
	return rec.SessionId
}

type contentBlock struct {
	Type      string `json:"type"`
	Id        string `json:"id"`
	Name      string `json:"name"`
	Signature string `json:"signature"`
}

// summarizeClaudeResponse derives searchable fields from a Messages response:
// ids to link turns (message id, tool_use ids), the model that actually
// answered, usage, and whether thinking came back signed.
func summarizeClaudeResponse(rec *callRecord, resp []byte) {
	addBlock := func(b *contentBlock) {
		if strings.HasSuffix(b.Type, "tool_use") {
			rec.ToolUses = append(rec.ToolUses, toolUse{Id: b.Id, Name: b.Name})
		}
		if b.Type == "thinking" {
			rec.ThinkingBlocks++
		}
		if b.Signature != "" {
			rec.Signatures++
		}
	}

	if !strings.HasPrefix(rec.RespContentType, "text/event-stream") {
		var msg struct {
			Id         string          `json:"id"`
			Model      string          `json:"model"`
			StopReason string          `json:"stop_reason"`
			Usage      json.RawMessage `json:"usage"`
			Content    []contentBlock  `json:"content"`
		}
		if common.Unmarshal(resp, &msg) != nil {
			return
		}
		rec.RespId, rec.RespModel, rec.StopReason, rec.Usage = msg.Id, msg.Model, msg.StopReason, msg.Usage
		rec.RespComplete = msg.StopReason != ""
		for i := range msg.Content {
			addBlock(&msg.Content[i])
		}
		return
	}

	for payload := range ssePayloads(resp) {
		// Deltas are the bulk of a stream and carry nothing to summarize
		// except signatures, so skip decoding the rest of them.
		if bytes.Contains(payload, []byte(`"content_block_delta"`)) && !bytes.Contains(payload, []byte(`"signature_delta"`)) {
			continue
		}
		var ev struct {
			Type    string `json:"type"`
			Message *struct {
				Id    string          `json:"id"`
				Model string          `json:"model"`
				Usage json.RawMessage `json:"usage"`
			} `json:"message"`
			ContentBlock *contentBlock `json:"content_block"`
			Delta        *struct {
				StopReason string `json:"stop_reason"`
				Signature  string `json:"signature"`
			} `json:"delta"`
			Usage json.RawMessage `json:"usage"`
		}
		if common.Unmarshal(payload, &ev) != nil {
			continue
		}
		switch ev.Type {
		case "message_start":
			if ev.Message != nil {
				rec.RespId, rec.RespModel, rec.UsageStart = ev.Message.Id, ev.Message.Model, ev.Message.Usage
			}
		case "content_block_start":
			if ev.ContentBlock != nil {
				addBlock(ev.ContentBlock)
			}
		case "content_block_delta":
			if ev.Delta != nil && ev.Delta.Signature != "" {
				rec.Signatures++
			}
		case "message_delta":
			if ev.Delta != nil && ev.Delta.StopReason != "" {
				rec.StopReason = ev.Delta.StopReason
			}
			rec.Usage = ev.Usage
		case "message_stop":
			rec.RespComplete = true
		}
	}
}
