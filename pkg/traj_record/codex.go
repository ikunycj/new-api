package traj_record

import (
	"cmp"
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// codexClient is the Codex CLI (codex_cli_rs, codex_exec, IDE extensions) on
// the OpenAI Responses API. Codex runs stateless (store=false): every call
// resends the whole thread as "input", including each reasoning item's
// encrypted_content, so the history dedups by prefix just like Claude Code's
// messages. The tool list travels inside input as an additional_tools item.
var codexClient = &client{
	name:       "codex",
	apiFamily:  "responses",
	paths:      []string{"/v1/responses", "/v1/responses/compact"},
	uaPrefixes: []string{"codex_"},
	historyKey: "input",
	blobKeys:   []string{"instructions", "tools"},
	requestIds: codexRequestIds,
	summarize:  summarizeCodexResponse,
}

// codexRequestIds reads client_metadata. The session id is the scope: a
// subagent spawned by forking its parent thread shares the session, so its
// copied history dedups against the parent.
func codexRequestIds(rec *callRecord, top map[string]json.RawMessage) string {
	var meta struct {
		SessionId      string `json:"session_id"`
		ThreadId       string `json:"thread_id"`
		TurnId         string `json:"turn_id"`
		ParentThreadId string `json:"x-codex-parent-thread-id"`
		InstallationId string `json:"x-codex-installation-id"`
	}
	if md := top["client_metadata"]; md != nil {
		_ = common.Unmarshal(md, &meta)
	}
	rec.SessionId, rec.ThreadId, rec.TurnId = meta.SessionId, meta.ThreadId, meta.TurnId
	rec.ParentThreadId, rec.DeviceId = meta.ParentThreadId, meta.InstallationId
	// The ids are also sent as headers: Session_id by older Codex,
	// Session-Id and Thread-Id since 0.15x.
	rec.SessionId = cmp.Or(rec.SessionId, rec.Headers["Session-Id"], rec.Headers["Session_id"])
	rec.ThreadId = cmp.Or(rec.ThreadId, rec.Headers["Thread-Id"])
	return cmp.Or(rec.SessionId, rec.ThreadId)
}

type responseItem struct {
	Type             string `json:"type"`
	CallId           string `json:"call_id"`
	Name             string `json:"name"`
	EncryptedContent string `json:"encrypted_content"`
}

type responseObject struct {
	Id                string `json:"id"`
	Model             string `json:"model"`
	Status            string `json:"status"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Usage  json.RawMessage `json:"usage"`
	Output []responseItem  `json:"output"`
}

// summarizeCodexResponse derives searchable fields from a Responses API
// response: the response id and model, final status and usage, tool call ids
// to link the next request's outputs, and how many reasoning items came back
// with encrypted_content (the full reasoning, only decryptable by OpenAI).
func summarizeCodexResponse(rec *callRecord, resp []byte) {
	addItem := func(it *responseItem) {
		if it.Type == "reasoning" {
			rec.ReasoningItems++
		}
		if it.EncryptedContent != "" {
			rec.EncryptedItems++
		}
		if it.CallId != "" {
			rec.ToolUses = append(rec.ToolUses, toolUse{Id: it.CallId, Name: it.Name})
		}
	}
	// final applies a terminal response object. Its output repeats the items
	// already seen as output_item.done events, so those are only counted when
	// the stream had none (non-streaming and compact responses).
	sawItems := false
	final := func(r *responseObject) {
		rec.RespId, rec.RespModel, rec.Usage = cmp.Or(r.Id, rec.RespId), cmp.Or(r.Model, rec.RespModel), r.Usage
		rec.StopReason = r.Status
		if r.IncompleteDetails != nil && r.IncompleteDetails.Reason != "" {
			rec.StopReason += ":" + r.IncompleteDetails.Reason
		}
		if !sawItems {
			for i := range r.Output {
				addItem(&r.Output[i])
			}
		}
	}

	if !strings.HasPrefix(rec.RespContentType, "text/event-stream") {
		var r responseObject
		if common.Unmarshal(resp, &r) != nil || (r.Id == "" && r.Output == nil) {
			return
		}
		final(&r)
		rec.RespComplete = r.Status != "failed"
		return
	}

	for payload := range ssePayloads(resp) {
		var ev struct {
			Type     string          `json:"type"`
			Response *responseObject `json:"response"`
			Item     *responseItem   `json:"item"`
		}
		if common.Unmarshal(payload, &ev) != nil {
			continue
		}
		switch ev.Type {
		case "response.created":
			if ev.Response != nil {
				rec.RespId, rec.RespModel = ev.Response.Id, ev.Response.Model
			}
		case "response.output_item.done":
			if ev.Item != nil {
				sawItems = true
				addItem(ev.Item)
			}
		case "response.completed", "response.incomplete":
			if ev.Response != nil {
				final(ev.Response)
			}
			rec.RespComplete = true
		case "response.failed":
			if ev.Response != nil {
				final(ev.Response)
			}
		}
	}
}
