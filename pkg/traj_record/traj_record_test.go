package traj_record

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const testSSE = "event: message_start\n" +
	`data: {"type":"message_start","message":{"id":"msg_1","model":"claude-test","usage":{"input_tokens":10}}}` + "\n\n" +
	"event: content_block_start\n" +
	`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"","signature":""}}` + "\n\n" +
	"event: content_block_delta\n" +
	`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"hmm"}}` + "\n\n" +
	"event: content_block_delta\n" +
	`data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig"}}` + "\n\n" +
	"event: content_block_start\n" +
	`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"Bash","input":{}}}` + "\n\n" +
	"event: message_delta\n" +
	`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":5}}` + "\n\n" +
	"event: message_stop\n" +
	`data: {"type":"message_stop"}` + "\n\n"

func newTestRecorder(t *testing.T, cfg Config) *recorder {
	t.Helper()
	cfg.Enabled = true
	cfg.Dir = t.TempDir()
	cfg.Instance = "test"
	cfg.normalize()
	fw, err := newFileWriter(cfg)
	require.NoError(t, err)
	r := &recorder{cfg: cfg, fw: fw}
	r.resetFileState()
	t.Cleanup(func() { _ = fw.close() })
	return r
}

// claudeBody builds a compact Claude Code style body. Messages are passed as
// raw JSON so tests control their exact bytes.
func claudeBody(session string, msgs ...string) []byte {
	userId, _ := common.Marshal(fmt.Sprintf(`{"device_id":"dev","account_uuid":"acc","session_id":"%s"}`, session))
	return fmt.Appendf(nil,
		`{"model":"claude-test","messages":[%s],"system":[{"type":"text","text":"You are Claude Code."}],"tools":[{"name":"Bash","input_schema":{"type":"object"}}],"metadata":{"user_id":%s},"max_tokens":32000,"stream":true}`,
		strings.Join(msgs, ","), userId)
}

func process(r *recorder, rid string, body []byte) {
	r.process(&job{
		client: claudeClient,
		rec:    callRecord{Rid: rid, TokenId: 1, RespContentType: "text/event-stream", Status: 200},
		body:   body,
		resp:   []byte(testSSE),
	})
}

type storedLine struct {
	ReqSize       int                        `json:"req_size"`
	ReqCompacted  bool                       `json:"req_compacted"`
	RespComplete  bool                       `json:"resp_complete"`
	RespTruncated bool                       `json:"resp_truncated"`
	Type          string                     `json:"type"`
	Rid           string                     `json:"rid"`
	Hash          string                     `json:"hash"`
	Data          json.RawMessage            `json:"data"`
	ParentRid     string                     `json:"parent_rid"`
	PrefixLen     int                        `json:"prefix_len"`
	HistoryKey    string                     `json:"history_key"`
	Refs          map[string]string          `json:"refs"`
	KeyOrder      []string                   `json:"key_order"`
	ReqSha256     string                     `json:"req_sha256"`
	Params        map[string]json.RawMessage `json:"params"`
	Append        []json.RawMessage          `json:"append"`
	Resp          string                     `json:"resp"`
	ReqRaw        string                     `json:"req_raw"`
}

// reconstruct replays one file the way offline tooling would and returns every
// request body, keyed by rid, together with the parsed call lines. It fails the
// test if a record references something outside the file.
func reconstruct(t *testing.T, data []byte) (map[string][]byte, map[string]storedLine) {
	t.Helper()
	blobs := map[string]json.RawMessage{}
	history := map[string][]json.RawMessage{}
	bodies := map[string][]byte{}
	calls := map[string]storedLine{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(nil, 256<<20)
	for sc.Scan() {
		var l storedLine
		require.NoError(t, common.Unmarshal(sc.Bytes(), &l))
		switch l.Type {
		case "segment_start":
			blobs, history = map[string]json.RawMessage{}, map[string][]json.RawMessage{}
		case "blob":
			blobs[l.Hash] = l.Data
		case "call":
			var msgs []json.RawMessage
			if l.ParentRid != "" {
				parent, ok := history[l.ParentRid]
				require.True(t, ok, "rid %s references %s outside this segment", l.Rid, l.ParentRid)
				msgs = append(msgs, parent[:l.PrefixLen]...)
			}
			msgs = append(msgs, l.Append...)
			history[l.Rid] = msgs
			calls[l.Rid] = l
			if l.KeyOrder == nil {
				bodies[l.Rid] = []byte(l.ReqRaw)
				continue
			}
			body := []byte{'{'}
			for i, k := range l.KeyOrder {
				if i > 0 {
					body = append(body, ',')
				}
				kb, _ := common.Marshal(k)
				body = append(append(body, kb...), ':')
				if k == l.HistoryKey {
					body = append(body, '[')
					for j, m := range msgs {
						if j > 0 {
							body = append(body, ',')
						}
						body = append(body, m...)
					}
					body = append(body, ']')
				} else if ref, ok := l.Refs[k]; ok {
					require.Contains(t, blobs, ref, "rid %s references a blob outside this segment", l.Rid)
					body = append(body, blobs[ref]...)
				} else {
					body = append(body, l.Params[k]...)
				}
			}
			bodies[l.Rid] = append(body, '}')
		}
	}
	require.NoError(t, sc.Err())
	return bodies, calls
}

func readActive(t *testing.T, r *recorder) []byte {
	t.Helper()
	require.NoError(t, r.fw.flush())
	data, err := os.ReadFile(r.fw.activePath())
	require.NoError(t, err)
	return data
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestRoundTripDedupe(t *testing.T) {
	r := newTestRecorder(t, Config{})
	u1 := `{"role":"user","content":"fix <b>&amp;</b> < bug ✓"}`
	a1 := `{"role":"assistant","content":[{"type":"thinking","thinking":"t","signature":"s1"},{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"ls"}}]}`
	u2 := `{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}`
	a2 := `{"role":"assistant","content":[{"type":"text","text":"done"}]}`
	u3c := `{"role":"user","content":[{"type":"text","text":"next","cache_control":{"type":"ephemeral"}}]}`
	u3 := `{"role":"user","content":[{"type":"text","text":"next"}]}`
	a3 := `{"role":"assistant","content":[{"type":"text","text":"ok"}]}`
	u4 := `{"role":"user","content":[{"type":"text","text":"more","cache_control":{"type":"ephemeral"}}]}`
	sysTail := `{"role":"system","content":"<total_tokens>1000</total_tokens>"}`
	branch := `{"role":"assistant","content":[{"type":"text","text":"other"}]}`
	summary := `{"role":"user","content":"This session is being continued from a previous conversation."}`

	bodies := map[string][]byte{
		"r1": claudeBody("S1", u1),
		"r2": claudeBody("S1", u1, a1, u2),
		"r3": claudeBody("S1", u1, a1, u2, a2, u3c, sysTail),
		// cache_control moved off u3 and the trailing system message changed.
		"r4": claudeBody("S1", u1, a1, u2, a2, u3, a3, u4),
		// Regenerated reply: branches off r2.
		"r5": claudeBody("S1", u1, a1, u2, branch),
		// Compaction: same session, history replaced by a summary.
		"r6": claudeBody("S1", summary),
		// Subagent / other session with an identical opening.
		"r7": claudeBody("S2", u1),
	}
	for _, rid := range []string{"r1", "r2", "r3", "r4", "r5", "r6", "r7"} {
		process(r, rid, bodies[rid])
	}
	data := readActive(t, r)
	got, calls := reconstruct(t, data)

	for rid, want := range bodies {
		require.Contains(t, got, rid)
		assert.Equal(t, string(want), string(got[rid]), rid)
		assert.Equal(t, sha(want), calls[rid].ReqSha256, rid)
	}

	link := func(rid string) [3]any {
		c := calls[rid]
		return [3]any{c.ParentRid, c.PrefixLen, len(c.Append)}
	}
	assert.Equal(t, [3]any{"", 0, 1}, link("r1"))
	assert.Equal(t, [3]any{"r1", 1, 2}, link("r2"))
	assert.Equal(t, [3]any{"r2", 3, 3}, link("r3"))
	assert.Equal(t, [3]any{"r3", 4, 3}, link("r4"))
	assert.Equal(t, [3]any{"r2", 3, 1}, link("r5"))
	assert.Equal(t, [3]any{"", 0, 1}, link("r6"))
	assert.Equal(t, [3]any{"", 0, 1}, link("r7"))

	// system and tools are identical across calls: one blob line each.
	assert.Equal(t, 2, bytes.Count(data, []byte(`"type":"blob"`)))
	// Raw bytes survive: HTML characters are not re-escaped.
	assert.Contains(t, string(data), `fix <b>&amp;</b> < bug ✓`)

	var rec callRecord
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(nil, 16<<20)
	for sc.Scan() {
		if bytes.Contains(sc.Bytes(), []byte(`"rid":"r1"`)) {
			require.NoError(t, common.Unmarshal(sc.Bytes(), &rec))
		}
	}
	assert.Equal(t, "S1", rec.SessionId)
	assert.Equal(t, "dev", rec.DeviceId)
	assert.Equal(t, "acc", rec.AccountUuid)
	assert.Equal(t, "msg_1", rec.RespId)
	assert.Equal(t, "tool_use", rec.StopReason)
	assert.True(t, rec.RespComplete)
	assert.Equal(t, 1, rec.ThinkingBlocks)
	assert.Equal(t, 1, rec.Signatures)
	assert.Equal(t, []toolUse{{Id: "toolu_1", Name: "Bash"}}, rec.ToolUses)
	assert.Equal(t, testSSE, rec.Resp)
	assert.EqualValues(t, 1, rec.Seq)
}

func TestNonCanonicalBodies(t *testing.T) {
	r := newTestRecorder(t, Config{})
	spaced := []byte(`{"model": "claude-test", "messages": [{"role":"user","content":"hi"}]}`)
	pretty := []byte("{\n  \"model\": \"claude-test\",\n  \"messages\": [\n    {\"role\": \"user\", \"content\": \"hi\"}\n  ]\n}")
	legacy := []byte(`{"messages":[{"role":"user","content":"hi"}],"metadata":{"user_id":"user_dev_account_acc_session_S9"}}`)
	process(r, "spaced", spaced)
	process(r, "pretty", pretty)
	process(r, "notjson", []byte("not json"))
	process(r, "legacy", legacy)

	data := readActive(t, r)
	bodies, calls := reconstruct(t, data)

	assert.Nil(t, calls["spaced"].KeyOrder, "whitespace cannot be reproduced")
	assert.Empty(t, calls["spaced"].Append)
	assert.Equal(t, string(spaced), string(bodies["spaced"]))

	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, pretty))
	assert.Equal(t, string(pretty), string(bodies["pretty"]))
	assert.Equal(t, sha(pretty), calls["pretty"].ReqSha256, "hash covers the bytes the client sent")
	assert.NotContains(t, string(data), `"req_compacted":true`)

	assert.Contains(t, string(data), `"req_raw":"not json"`)

	assert.Equal(t, string(legacy), string(bodies["legacy"]))
	assert.Contains(t, string(data), `"session_id":"S9","device_id":"dev","account_uuid":"acc"`)
}

func TestRotationKeepsFilesSelfContained(t *testing.T) {
	r := newTestRecorder(t, Config{ZstdWindowMB: 1})
	u1 := `{"role":"user","content":"a"}`
	a1 := `{"role":"assistant","content":"b"}`
	u2 := `{"role":"user","content":"c"}`
	process(r, "r1", claudeBody("S", u1))
	r.fw.maxBytes = 1 // rotate before the next record
	process(r, "r2", claudeBody("S", u1, a1, u2))
	require.NoError(t, r.fw.close())

	matches, err := filepath.Glob(filepath.Join(r.cfg.Dir, rotatePrefix+"*.jsonl.zst"))
	require.NoError(t, err)
	require.Len(t, matches, 1)
	f, err := os.Open(matches[0])
	require.NoError(t, err)
	defer f.Close()
	zr, err := zstd.NewReader(f)
	require.NoError(t, err)
	defer zr.Close()
	old, err := io.ReadAll(zr)
	require.NoError(t, err)
	oldBodies, _ := reconstruct(t, old)
	assert.Contains(t, oldBodies, "r1")

	cur, err := os.ReadFile(r.fw.activePath())
	require.NoError(t, err)
	bodies, calls := reconstruct(t, cur)
	assert.Equal(t, string(claudeBody("S", u1, a1, u2)), string(bodies["r2"]))
	assert.Empty(t, calls["r2"].ParentRid, "new file starts with a keyframe")
	assert.Equal(t, 2, bytes.Count(cur, []byte(`"type":"blob"`)), "blobs are repeated in the new file")
}

const testCodexSSE = "event: response.created\n" +
	`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-test","status":"in_progress","output":[]}}` + "\n\n" +
	"event: response.output_item.done\n" +
	`data: {"type":"response.output_item.done","output_index":0,"item":{"type":"reasoning","id":"rs_1","summary":[],"encrypted_content":"gAAAA-enc"}}` + "\n\n" +
	"event: response.custom_tool_call_input.delta\n" +
	`data: {"type":"response.custom_tool_call_input.delta","output_index":1,"delta":"ls"}` + "\n\n" +
	"event: response.output_item.done\n" +
	`data: {"type":"response.output_item.done","output_index":1,"item":{"type":"custom_tool_call","call_id":"call_1","name":"exec","input":"ls"}}` + "\n\n" +
	"event: response.completed\n" +
	`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-test","status":"completed","usage":{"input_tokens":10,"output_tokens":5},"output":[{"type":"reasoning","encrypted_content":"gAAAA-enc"},{"type":"custom_tool_call","call_id":"call_1","name":"exec"}]}}` + "\n\n"

// codexBody builds a compact Codex style body. Like Codex 0.14x it carries the
// tool list as an input item; instructions are included to exercise blobs.
func codexBody(session, thread, parent string, items ...string) []byte {
	meta := fmt.Sprintf(`{"session_id":"%s","thread_id":"%s","turn_id":"turn-1","x-codex-installation-id":"inst"`, session, thread)
	if parent != "" {
		meta += fmt.Sprintf(`,"x-codex-parent-thread-id":"%s","x-openai-subagent":"collab_spawn"`, parent)
	}
	return fmt.Appendf(nil,
		`{"model":"gpt-test","instructions":"You are Codex.","input":[%s],"tool_choice":"auto","reasoning":{"effort":"high"},"store":false,"stream":true,"include":["reasoning.encrypted_content"],"prompt_cache_key":"%s","client_metadata":%s}}`,
		strings.Join(items, ","), thread, meta)
}

func processCodex(r *recorder, rid string, body []byte, contentType, resp string) {
	r.process(&job{
		client: codexClient,
		rec:    callRecord{Rid: rid, TokenId: 1, RespContentType: contentType, Status: 200},
		body:   body,
		resp:   []byte(resp),
	})
}

func TestCodexRoundTrip(t *testing.T) {
	r := newTestRecorder(t, Config{})
	tools := `{"type":"additional_tools","role":"developer","tools":[{"type":"custom","name":"exec","description":"Run <js> & more"}]}`
	dev := `{"type":"message","role":"developer","content":[{"type":"input_text","text":"sandbox: none"}]}`
	u1 := `{"type":"message","role":"user","content":[{"type":"input_text","text":"list files ✓"}]}`
	rs := `{"type":"reasoning","summary":[],"encrypted_content":"gAAAA-enc","internal_chat_message_metadata_passthrough":{"turn_id":"turn-1"}}`
	call := `{"type":"custom_tool_call","call_id":"call_1","name":"exec","input":"ls"}`
	out := `{"type":"custom_tool_call_output","call_id":"call_1","output":"a.txt"}`
	spawn := `{"type":"message","role":"user","content":[{"type":"input_text","text":"subtask"}]}`

	bodies := map[string][]byte{
		"c1": codexBody("S1", "T1", "", tools, dev, u1),
		"c2": codexBody("S1", "T1", "", tools, dev, u1, rs, call, out),
		// Subagent forked from T1: same session, own thread, parent history.
		"c3": codexBody("S1", "T2", "T1", tools, dev, u1, rs, call, out, spawn),
		// Unrelated session with an identical opening.
		"c4": codexBody("S2", "T9", "", tools, dev, u1),
	}
	for _, rid := range []string{"c1", "c2", "c3", "c4"} {
		processCodex(r, rid, bodies[rid], "text/event-stream", testCodexSSE)
	}
	compact := []byte(`{"model":"gpt-test","input":[` + u1 + `],"instructions":"You are Codex."}`)
	processCodex(r, "c5", compact, "application/json",
		`{"id":"resp_c","object":"response.compaction","output":[{"type":"compaction","encrypted_content":"gAAAA-cmp"}],"usage":{"input_tokens":3}}`)
	processCodex(r, "c6", []byte(`{"model":"gpt-test","input":"plain string prompt"}`), "application/json", `{}`)

	data := readActive(t, r)
	got, calls := reconstruct(t, data)
	bodies["c5"] = compact
	for rid, want := range bodies {
		require.Contains(t, got, rid)
		assert.Equal(t, string(want), string(got[rid]), rid)
		assert.Equal(t, sha(want), calls[rid].ReqSha256, rid)
	}

	link := func(rid string) [3]any {
		c := calls[rid]
		return [3]any{c.ParentRid, c.PrefixLen, len(c.Append)}
	}
	assert.Equal(t, [3]any{"", 0, 3}, link("c1"))
	assert.Equal(t, [3]any{"c1", 3, 3}, link("c2"))
	assert.Equal(t, [3]any{"c2", 6, 1}, link("c3"), "fork dedups against its parent thread")
	assert.Equal(t, [3]any{"", 0, 3}, link("c4"))
	assert.Equal(t, "input", calls["c1"].HistoryKey)
	assert.Contains(t, calls["c1"].Refs, "instructions")
	assert.Contains(t, string(data), `"req_raw":"{\"model\":\"gpt-test\",\"input\":\"plain string prompt\"}"`)
	// Raw bytes survive: HTML characters are not re-escaped.
	assert.Contains(t, string(data), `Run <js> & more`)

	recs := map[string]callRecord{}
	for line := range bytes.Lines(data) {
		var rec callRecord
		require.NoError(t, common.Unmarshal(line, &rec))
		if rec.Type == "call" {
			recs[rec.Rid] = rec
		}
	}
	c3 := recs["c3"]
	assert.Equal(t, "codex", c3.Client)
	assert.Equal(t, [4]string{"S1", "T2", "T1", "inst"}, [4]string{c3.SessionId, c3.ThreadId, c3.ParentThreadId, c3.DeviceId})
	assert.Equal(t, "turn-1", c3.TurnId)
	assert.Equal(t, "resp_1", c3.RespId)
	assert.Equal(t, "gpt-test", c3.RespModel)
	assert.Equal(t, "completed", c3.StopReason)
	assert.JSONEq(t, `{"input_tokens":10,"output_tokens":5}`, string(c3.Usage))
	assert.True(t, c3.RespComplete)
	assert.Equal(t, 1, c3.ReasoningItems, "completed output does not double count")
	assert.Equal(t, 1, c3.EncryptedItems)
	assert.Equal(t, []toolUse{{Id: "call_1", Name: "exec"}}, c3.ToolUses)
	assert.Equal(t, testCodexSSE, c3.Resp)

	c5 := recs["c5"]
	assert.Equal(t, "resp_c", c5.RespId)
	assert.Equal(t, 1, c5.EncryptedItems, "compaction item is kept encrypted")
	assert.True(t, c5.RespComplete)
}

func TestCodexHeaderIds(t *testing.T) {
	r := newTestRecorder(t, Config{})
	body := []byte(`{"model":"gpt-test","input":[{"type":"message","role":"user","content":"hi"}],"prompt_cache_key":"PCK"}`)
	for rid, headers := range map[string]map[string]string{
		"new":  {"Session-Id": "S-new", "Thread-Id": "T-new"},
		"old":  {"Session_id": "S-old"},
		"none": {},
	} {
		r.process(&job{
			client: codexClient,
			rec:    callRecord{Rid: rid, TokenId: 1, Headers: headers},
			body:   body,
		})
	}
	recs := map[string]callRecord{}
	for line := range bytes.Lines(readActive(t, r)) {
		var rec callRecord
		require.NoError(t, common.Unmarshal(line, &rec))
		recs[rec.Rid] = rec
	}
	assert.Equal(t, [2]string{"S-new", "T-new"}, [2]string{recs["new"].SessionId, recs["new"].ThreadId})
	assert.Equal(t, [2]string{"S-old", ""}, [2]string{recs["old"].SessionId, recs["old"].ThreadId})
	assert.Equal(t, [2]string{"", ""}, [2]string{recs["none"].SessionId, recs["none"].ThreadId})
}

func TestClientCanceled(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Init(Config{Enabled: true, Dir: dir, Instance: "test"}))
	t.Cleanup(Close)

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(common.RequestIdKey, c.GetHeader("X-Rid"))
		c.Next()
	})
	engine.Use(Middleware())
	engine.POST("/v1/responses", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)
		_, _ = c.Writer.WriteString(testCodexSSE[:100])
	})
	body := codexBody("S", "T", "", `{"type":"message","role":"user","content":"hi"}`)
	for _, rid := range []string{"live", "gone"} {
		ctx, cancel := context.WithCancel(context.Background())
		if rid == "gone" {
			cancel()
		}
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/responses", bytes.NewReader(body))
		req.Header.Set("User-Agent", "codex_cli_rs/0.155.0")
		req.Header.Set("X-Rid", rid)
		engine.ServeHTTP(httptest.NewRecorder(), req)
		cancel()
	}

	Close()
	data, err := os.ReadFile(filepath.Join(dir, activeFileName))
	require.NoError(t, err)
	recs := map[string]callRecord{}
	for line := range bytes.Lines(data) {
		var rec callRecord
		require.NoError(t, common.Unmarshal(line, &rec))
		recs[rec.Rid] = rec
	}
	require.Contains(t, recs, "gone")
	assert.True(t, recs["gone"].ClientCanceled)
	assert.False(t, recs["live"].ClientCanceled)
	assert.False(t, recs["gone"].RespComplete, "stream cut before response.completed")
}

func TestMiddleware(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Init(Config{Enabled: true, Dir: dir, Instance: "test"}))
	t.Cleanup(Close)

	var seen []byte
	engine := gin.New()
	nreq := 0
	engine.Use(func(c *gin.Context) {
		nreq++
		c.Set(common.RequestIdKey, fmt.Sprintf("req-%d", nreq))
		common.SetContextKey(c, constant.ContextKeyUserId, 7)
		common.SetContextKey(c, constant.ContextKeyOriginalModel, "claude-test")
		common.SetContextKey(c, constant.ContextKeyChannelParamOverride, map[string]any{
			"operations": []any{map[string]any{"mode": "pass_headers", "value": []any{"User-Agent"}}},
		})
		common.SetContextKey(c, constant.ContextKeyChannelHeaderOverride, map[string]any{"X-Upstream-Key": "sk-secret"})
		c.Next()
	})
	engine.Use(Middleware())
	engine.POST("/v1/messages", func(c *gin.Context) {
		seen, _ = io.ReadAll(c.Request.Body)
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)
		_, _ = c.Writer.WriteString(testSSE)
		c.Writer.Flush()
	})

	engine.POST("/v1/responses", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)
		_, _ = c.Writer.WriteString(testCodexSSE)
	})

	body := claudeBody("S", `{"role":"user","content":"hi"}`)
	send := func(ua string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Authorization", "Bearer sk-secret")
		req.Header.Set("X-Api-Key", "sk-secret")
		req.Header.Set("Anthropic-Beta", "beta-1")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}

	w := send("claude-cli/2.0.0 (external, cli)")
	assert.Equal(t, testSSE, w.Body.String())
	assert.Equal(t, string(body), string(seen), "downstream handler sees the full body")
	assert.True(t, w.Flushed)

	send("curl/8.0")

	cbody := codexBody("S", "T", "", `{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}`)
	for _, ua := range []string{"codex_exec/0.144.6 (Debian 13.0.0; x86_64) unknown (codex_exec; 0.144.6)", "claude-cli/2.0.0"} {
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(cbody))
		req.Header.Set("User-Agent", ua)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, testCodexSSE, w.Body.String())
	}

	Close()
	data, err := os.ReadFile(filepath.Join(dir, activeFileName))
	require.NoError(t, err)
	assert.Equal(t, 4, bytes.Count(data, []byte(`"type":"call"`)), "protocol routes accept all clients")
	assert.NotContains(t, string(data), "sk-secret")

	bodies, calls := reconstruct(t, data)
	require.Len(t, calls, 4)
	recs := map[string]callRecord{}
	for line := range bytes.Lines(data) {
		var rec callRecord
		require.NoError(t, common.Unmarshal(line, &rec))
		if rec.Type == "call" {
			recs[rec.Client] = rec
		}
	}
	assert.Equal(t, string(cbody), string(bodies[recs["codex"].Rid]))
	assert.Equal(t, testCodexSSE, recs["codex"].Resp)
	rec := recs["claude"]
	assert.Equal(t, string(body), string(bodies[rec.Rid]))
	assert.Equal(t, testSSE, rec.Resp)
	assert.Equal(t, 7, rec.UserId)
	assert.Equal(t, "claude-test", rec.Model)
	assert.Equal(t, "beta-1", rec.Headers["Anthropic-Beta"])
	assert.Equal(t, []overrideOp{{Mode: "pass_headers", Value: []any{"User-Agent"}}}, rec.Upstream.ParamOverrideOps)
	assert.False(t, rec.Upstream.BodyOverride, "pass_headers leaves the body alone")
	assert.Equal(t, []string{"X-Upstream-Key"}, rec.Upstream.HeaderOverrideKeys)
	assert.NotZero(t, rec.TsFirstByte)
	assert.GreaterOrEqual(t, rec.TsEnd, rec.TsStart)
}

func TestSummarizeParamOverride(t *testing.T) {
	ops, body := summarizeParamOverride(map[string]any{
		"temperature": 0.5,
		"operations": []any{
			map[string]any{"mode": "set_header", "path": "X-Key", "value": "sk-secret"},
			map[string]any{"mode": "set", "path": "max_tokens", "value": 8192,
				"conditions": []any{map[string]any{"path": "model", "mode": "prefix", "value": "claude-"}}},
		},
	})
	assert.Equal(t, []overrideOp{
		{Mode: "set", Path: "temperature", Value: 0.5},
		{Mode: "set_header", Path: "X-Key"},
		{Mode: "set", Path: "max_tokens", Value: 8192, Conditional: true},
	}, ops)
	assert.True(t, body)

	ops, body = summarizeParamOverride(nil)
	assert.Empty(t, ops)
	assert.False(t, body)
}

// TestReplaySamples feeds a relaytrace capture (Claude or GPT) through the
// recorder and checks that every request of a recorded client reconstructs
// exactly. Opt-in because the samples hold real user data and are not part of
// the repository:
//
//	TRAJ_RECORD_SAMPLES=/path/claude-trace-samples.jsonl go test -run TestReplaySamples -v ./pkg/traj_record/
func TestReplaySamples(t *testing.T) {
	path := os.Getenv("TRAJ_RECORD_SAMPLES")
	if path == "" {
		t.Skip("TRAJ_RECORD_SAMPLES not set")
	}
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	type sample struct {
		Rid        string            `json:"rid"`
		Ts         int64             `json:"ts"`
		Path       string            `json:"path"`
		Req        json.RawMessage   `json:"req"`
		Resp       string            `json:"resp"`
		ReqHeaders map[string]string `json:"req_headers"`
		client     *client
	}
	cfg := Config{}
	cfg.normalize()
	routes := buildRoutes(cfg)
	var samples []sample
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 512<<20)
	for sc.Scan() {
		var s sample
		require.NoError(t, common.Unmarshal(sc.Bytes(), &s))
		rt, ok := routes[s.Path]
		ua := s.ReqHeaders["User-Agent"]
		if s.Req != nil && ok && slices.ContainsFunc(rt.uaPrefixes, func(p string) bool { return strings.HasPrefix(ua, p) }) {
			s.client = rt.client
			samples = append(samples, s)
		}
	}
	require.NoError(t, sc.Err())

	slices.SortStableFunc(samples, func(a, b sample) int { return cmp.Compare(a.Ts, b.Ts) })

	r := newTestRecorder(t, Config{})
	want := map[string][]byte{}
	var reqBytes, respBytes int
	start := time.Now()
	for _, s := range samples {
		want[s.Rid] = s.Req
		reqBytes += len(s.Req)
		respBytes += len(s.Resp)
		r.process(&job{
			client: s.client,
			rec:    callRecord{Rid: s.Rid, TsStart: s.Ts, RespContentType: "text/event-stream"},
			body:   s.Req,
			resp:   []byte(s.Resp),
		})
	}
	elapsed := time.Since(start)
	data := readActive(t, r)
	bodies, calls := reconstruct(t, data)

	keyframes := 0
	for rid, b := range want {
		require.Equal(t, sha(b), sha(bodies[rid]), rid)
		if calls[rid].ParentRid == "" {
			keyframes++
		}
	}
	t.Logf("%d calls, %d keyframes; requests %.1fMB + responses %.1fMB -> file %.1fMB; %v",
		len(samples), keyframes, float64(reqBytes)/1e6, float64(respBytes)/1e6, float64(len(data))/1e6, elapsed)
}
