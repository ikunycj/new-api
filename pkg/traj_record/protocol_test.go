package traj_record

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestProtocolCaptureUnknownAgents(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Init(Config{Enabled: true, Dir: dir}))
	t.Cleanup(Close)
	engine := gin.New()
	engine.Use(Middleware())
	for _, path := range []string{"/v1/messages", "/v1/responses", "/v1/chat/completions"} {
		engine.POST(path, func(c *gin.Context) {
			c.Data(200, "application/json", []byte(`{"choices":[{"index":0,"finish_reason":"stop"}]}`))
		})
	}
	for _, ua := range []string{"opencode/test", "pi/test", ""} {
		for _, path := range []string{"/v1/messages", "/v1/responses", "/v1/chat/completions"} {
			req := httptest.NewRequest("POST", path, strings.NewReader(`{"messages":[],"input":"hello"}`))
			req.Header.Set("User-Agent", ua)
			engine.ServeHTTP(httptest.NewRecorder(), req)
		}
	}
	Close()
	data, err := os.ReadFile(filepath.Join(dir, activeFileName))
	require.NoError(t, err)
	require.Equal(t, 9, bytes.Count(data, []byte(`"type":"call"`)))
	require.Equal(t, 3, bytes.Count(data, []byte(`"api_family":"chat_completions"`)))
	require.NotContains(t, string(data), `"agent":"codex"`)
}

func TestProtocolAllowlistDoesNotFailOpen(t *testing.T) {
	t.Setenv(EnvAPIFamilies, "unknown")
	require.Empty(t, buildRoutes(ConfigFromEnv()))
	t.Setenv(EnvAPIFamilies, "chat_completions")
	routes := buildRoutes(ConfigFromEnv())
	require.Len(t, routes, 1)
	require.True(t, matchesUA(routes["/v1/chat/completions"], "anything"))
}

func TestExactFallbackAndIdentity(t *testing.T) {
	r := newTestRecorder(t, Config{})
	inputs := []string{`{"input":"hi","prompt_cache_key":"same"}`, `{"input":[],"model":"a","model":"b"}`, "{\n\"input\":[]\n}", `{"input":[],"previous_response_id":"resp_1"}`}
	for i, body := range inputs {
		r.process(&job{client: codexClient, rec: callRecord{Rid: strings.Repeat("r", i+1)}, body: []byte(body)})
	}
	bodies, calls := reconstruct(t, readActive(t, r))
	for i, body := range inputs {
		rid := strings.Repeat("r", i+1)
		require.Equal(t, body, string(bodies[rid]))
		require.Equal(t, sha([]byte(body)), calls[rid].ReqSha256)
	}
	require.NotContains(t, string(readActive(t, r)), `"thread_id":"same"`)
}

func TestChatCompletionTransport(t *testing.T) {
	prefix := "data: {\"id\":\"c\",\"choices\":[{\"index\":0,\"finish_reason\":\"tool_calls\"},{\"index\":1,\"finish_reason\":\"stop\"}]}\n\n"
	usage := "data: {\"choices\":[],\"usage\":{\"total_tokens\":7}}\n\n"
	for _, done := range []bool{false, true} {
		body := prefix + usage
		if done {
			body += "data: [DONE]\n\n"
		}
		rec := callRecord{RespContentType: "text/event-stream"}
		summarizeChatResponse(&rec, []byte(body))
		require.Equal(t, done, rec.RespComplete)
		require.JSONEq(t, `{"total_tokens":7}`, string(rec.Usage))
		require.Equal(t, "tool_calls", rec.StopReason)
	}
}

func TestWebSocketNativeEvents(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Init(Config{Enabled: true, Dir: dir, MaxRespMB: 1}))
	t.Cleanup(Close)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/responses", nil)
	c.Request.Header.Set("Authorization", "Bearer secret")
	c.Set(common.RequestIdKey, "request-1")
	capture := BeginWebSocket(c)
	require.NotNil(t, capture)
	capture.Message("in", 1, []byte(`{"type":"response.create","input":[]}`), nil)
	capture.Message("in", 1, []byte(`{"type":"response.cancel"}`), nil)
	capture.Call(c, []byte(`{"type":"response.create","input":[]}`), time.Now())
	capture.Message("out", 1, []byte(`{"type":"response.completed"}`), nil)
	capture.Message("out", 2, []byte{0xff}, nil)
	capture.Message("out", 1, bytes.Repeat([]byte("a"), (1<<20)+1), nil)
	capture.Close()
	Close()
	data, err := os.ReadFile(filepath.Join(dir, activeFileName))
	require.NoError(t, err)
	require.NotContains(t, string(data), "Bearer secret")
	var seq int64
	for line := range bytes.Lines(data) {
		var ev wsRecord
		require.NoError(t, json.Unmarshal(line, &ev))
		if !strings.HasPrefix(ev.Type, "ws_") {
			continue
		}
		seq++
		require.Equal(t, seq, ev.Seq)
		if ev.Size > (1 << 20) {
			require.True(t, ev.Truncated)
			require.Len(t, ev.Payload, 1<<20)
		}
	}
	require.EqualValues(t, 8, seq)
}

func TestUnknownSessionDedupIsStorageOnlyAndUserIsolated(t *testing.T) {
	r := newTestRecorder(t, Config{})
	body := []byte(`{"input":[{"role":"user","content":"same"}],"prompt_cache_key":"shared-cache"}`)
	for i, user := range []int{1, 1, 2} {
		r.process(&job{client: codexClient, rec: callRecord{Rid: strings.Repeat("r", i+1), TokenId: 1, UserId: user}, body: body})
	}
	bodies, calls := reconstruct(t, readActive(t, r))
	require.Equal(t, "r", calls["rr"].ParentRid)
	require.Empty(t, calls["rrr"].ParentRid)
	for _, got := range bodies {
		require.Equal(t, body, got)
	}
	require.NotContains(t, string(readActive(t, r)), `"session_id"`)
	require.NotContains(t, string(readActive(t, r)), `"thread_id"`)
}

func TestRawEmptyAndNonUTF8Requests(t *testing.T) {
	r := newTestRecorder(t, Config{})
	binary := []byte{'{', '"', 'i', 'n', 'p', 'u', 't', '"', ':', '[', '"', 0xff, '"', ']', '}'}
	r.process(&job{client: codexClient, rec: callRecord{Rid: "binary"}, body: binary})
	r.process(&job{client: codexClient, rec: callRecord{Rid: "empty"}, body: []byte{}})
	for line := range bytes.Lines(readActive(t, r)) {
		var rec callRecord
		require.NoError(t, common.Unmarshal(line, &rec))
		if rec.Rid == "binary" {
			require.Equal(t, binary, rec.ReqRawB64)
			require.Nil(t, rec.ReqRaw)
		}
		if rec.Rid == "empty" {
			require.NotNil(t, rec.ReqRaw)
			require.Empty(t, *rec.ReqRaw)
		}
	}
}
