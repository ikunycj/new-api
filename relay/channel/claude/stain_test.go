package claude

import (
	"encoding/json"
	"strings"
	"testing"
)

// unset the env-dependent switch; the tests toggle it directly.
func setStaining(t *testing.T, on bool) {
	t.Helper()
	old := stainingEnabled
	stainingEnabled = on
	t.Cleanup(func() { stainingEnabled = old })
}

const fullStart = `{"type":"message_start","message":{"id":"msg_011CfRO916Mol4q4Oj5ZF5YT","type":"message","role":"assistant","content":[],"usage":{"input_tokens":10}}}`

func TestStainDisabledIsByteIdentical(t *testing.T) {
	setStaining(t, false)
	got := StainClaudeMessageID([]byte(fullStart), "2026093012345612345678987654321Dt0AbWuf")
	if string(got) != fullStart {
		t.Fatalf("disabled staining must not touch the body")
	}
}

func TestStainAppendsRIDTail(t *testing.T) {
	setStaining(t, true)
	rid := "2026093012345612345678987654321Dt0AbWuf"
	got := StainClaudeMessageID([]byte(fullStart), rid)
	var resp map[string]any
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("stained body is not valid JSON: %v", err)
	}
	msg := resp["message"].(map[string]any)
	id := msg["id"].(string)
	if !strings.HasPrefix(id, "msg_011CfRO916Mol4q4Oj5ZF5YT") {
		t.Fatalf("original id must be preserved as prefix, got %s", id)
	}
	if !strings.HasSuffix(id, rid[len(rid)-12:]) {
		t.Fatalf("id must end with the 12-char rid tail, got %s", id)
	}
}

func TestStainNonStreamBody(t *testing.T) {
	setStaining(t, true)
	body := `{"id":"msg_011Cabc","model":"claude-opus-4-8","content":[{"type":"text","text":"hi"}]}`
	got := StainClaudeMessageID([]byte(body), "202609301234567890123456789012349AbCdEfG")
	var resp map[string]any
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("stained body is not valid JSON: %v", err)
	}
	if !strings.HasSuffix(resp["id"].(string), "12349AbCdEfG") {
		t.Fatalf("non-stream id tail mismatch: %s", resp["id"])
	}
}

func TestStainLeavesNonMsgIDAlone(t *testing.T) {
	setStaining(t, true)
	body := `{"id":"chatcmpl-xyz","object":"chat.completion"}`
	got := StainClaudeMessageID([]byte(body), "2026093012345612345678987654321Dt0AbWuf")
	if string(got) != body {
		t.Fatalf("non msg_ ids must not be rewritten")
	}
}

func TestStainNoIdReturnsUnchanged(t *testing.T) {
	setStaining(t, true)
	body := `{"type":"message_delta","delta":{"stop_reason":"end_turn"}}`
	got := StainClaudeMessageID([]byte(body), "2026093012345612345678987654321Dt0AbWuf")
	if string(got) != body {
		t.Fatalf("bodies without id must be unchanged")
	}
}

func TestStainRidEmptyReturnsUnchanged(t *testing.T) {
	setStaining(t, true)
	got := StainClaudeMessageID([]byte(fullStart), "")
	if string(got) != fullStart {
		t.Fatalf("empty rid must leave the body unchanged")
	}
}
