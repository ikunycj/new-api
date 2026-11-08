package traj_record

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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

// testClock is a settable clock for day rotation.
type testClock struct{ t time.Time }

func (c *testClock) now() time.Time { return c.t }

func newClockedRecorder(t *testing.T, dir string, clock *testClock, cfg Config) *recorder {
	t.Helper()
	cfg.Enabled, cfg.Dir, cfg.Instance = true, dir, "test"
	cfg.normalize()
	fw, err := newFileWriterAt(cfg, clock.now)
	require.NoError(t, err)
	r := &recorder{cfg: cfg, fw: fw}
	r.initStats()
	r.resetFileState()
	return r
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	if !strings.HasSuffix(path, ".zst") {
		return data
	}
	zr, err := zstd.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer zr.Close()
	out, err := io.ReadAll(zr)
	require.NoError(t, err)
	return out
}

func lineTypes(data []byte) []string {
	var types []string
	for line := range bytes.Lines(data) {
		var l struct {
			Type string `json:"type"`
		}
		_ = common.Unmarshal(line, &l)
		types = append(types, l.Type)
	}
	return types
}

var day1 = time.Date(2026, 11, 4, 23, 59, 0, 0, time.UTC)

func TestRotationWithinDayContinuesSegment(t *testing.T) {
	dir := t.TempDir()
	clock := &testClock{day1}
	r := newClockedRecorder(t, dir, clock, Config{ZstdWindowMB: 1})
	u1 := `{"role":"user","content":"a"}`
	a1 := `{"role":"assistant","content":"b"}`
	u2 := `{"role":"user","content":"c"}`
	process(r, "r1", claudeBody("S", u1))
	r.fw.maxBytes = 1 // rotate before the next record
	process(r, "r2", claudeBody("S", u1, a1, u2))
	require.NoError(t, r.fw.close())

	old := readFile(t, filepath.Join(dir, "traj-20261104-000.jsonl.zst"))
	cur := readFile(t, filepath.Join(dir, activeFileName))
	assert.Equal(t, "file_start", lineTypes(cur)[0], "a full file continues the day's segment")
	bodies, calls := reconstruct(t, append(old, cur...))
	assert.Equal(t, string(claudeBody("S", u1, a1, u2)), string(bodies["r2"]))
	assert.Equal(t, "r1", calls["r2"].ParentRid, "references reach into the previous file of the day")
	assert.Equal(t, 0, bytes.Count(cur, []byte(`"type":"blob"`)), "blobs are not repeated within a day")
	assert.Contains(t, lineTypes(old), "stats", "every file ends with a stats line")
	assert.NoFileExists(t, filepath.Join(dir, manifestName("20261104")), "the day is not over")
}

func TestDayRotationSealsDay(t *testing.T) {
	dir := t.TempDir()
	clock := &testClock{day1}
	r := newClockedRecorder(t, dir, clock, Config{ZstdWindowMB: 1})
	u1 := `{"role":"user","content":"a"}`
	process(r, "r1", claudeBody("S", u1))
	clock.t = day1.Add(2 * time.Minute) // 11/05 00:01 UTC
	process(r, "r2", claudeBody("S", u1, `{"role":"assistant","content":"b"}`))
	require.NoError(t, r.fw.close())

	var m manifest
	require.NoError(t, common.Unmarshal(readFile(t, filepath.Join(dir, manifestName("20261104"))), &m))
	require.Len(t, m.Files, 1)
	assert.Equal(t, "traj-20261104-000.jsonl.zst", m.Files[0].Name)
	assert.Equal(t, sha(mustRead(t, filepath.Join(dir, m.Files[0].Name))), m.Files[0].Sha256)
	assert.Equal(t, "20261104", m.Day)

	cur := readFile(t, filepath.Join(dir, activeFileName))
	assert.Equal(t, "segment_start", lineTypes(cur)[0], "a new day starts self-contained")
	assert.Contains(t, string(cur), `"day":"20261105"`)
	_, calls := reconstruct(t, cur)
	assert.Empty(t, calls["r2"].ParentRid)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestStartupRecovery(t *testing.T) {
	dir := t.TempDir()
	// The previous process wrote 11/03 and crashed in the middle of a 11/04
	// line, and while compressing an 11/03 file.
	ts := strconv.FormatInt(day1.UnixMilli(), 10)
	torn := `{"type":"call","rid":"x","ap`
	active := `{"type":"segment_start","ts":` + ts + `}` + "\n" + `{"type":"stats","ts":` + ts + `}` + "\n" + torn
	require.NoError(t, os.WriteFile(filepath.Join(dir, activeFileName), []byte(active), 0o640))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "traj-20261103-000.jsonl"), []byte(`{"type":"segment_start"}`+"\n"), 0o640))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "traj-20261103-000.jsonl.zst.tmp"), []byte("partial"), 0o640))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "traj-20261017-155408.956.jsonl.zst"), []byte("v3"), 0o640))

	clock := &testClock{day1.Add(time.Hour)} // 11/05
	r := newClockedRecorder(t, dir, clock, Config{ZstdWindowMB: 1})
	require.NoError(t, r.fw.close())

	assert.NoFileExists(t, filepath.Join(dir, "traj-20261103-000.jsonl.zst.tmp"))
	assert.NoFileExists(t, filepath.Join(dir, "traj-20261103-000.jsonl"))
	assert.Equal(t, `{"type":"segment_start"}`+"\n", string(readFile(t, filepath.Join(dir, "traj-20261103-000.jsonl.zst"))))
	sealed := readFile(t, filepath.Join(dir, "traj-20261104-000.jsonl.zst"))
	assert.Equal(t, []string{"segment_start", "stats"}, lineTypes(sealed), "torn line cut, file sealed into its own day")
	for _, day := range []string{"20261103", "20261104"} {
		assert.FileExists(t, filepath.Join(dir, manifestName(day)))
	}
	assert.NoFileExists(t, filepath.Join(dir, manifestName("20261017")), "v3 files are left alone")
	cur := readFile(t, filepath.Join(dir, activeFileName))
	assert.Contains(t, string(cur), fmt.Sprintf(`"recovered_bytes":%d`, len(torn)))
	assert.Contains(t, string(cur), `"day":"20261105"`)
}

func TestStartupMovesV3FileAside(t *testing.T) {
	dir := t.TempDir()
	ts := strconv.FormatInt(day1.Add(-6*24*time.Hour).UnixMilli(), 10)
	v3 := `{"type":"segment_start","v":3,"ts":` + ts + `}` + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, activeFileName), []byte(v3), 0o640))
	clock := &testClock{day1}
	r := newClockedRecorder(t, dir, clock, Config{ZstdWindowMB: 1})
	require.NoError(t, r.fw.close())
	matches, _ := filepath.Glob(filepath.Join(dir, "traj-*"))
	require.Len(t, matches, 1, "no 录制日 file, no manifest")
	assert.Regexp(t, `traj-\d{8}-\d{6}\.\d{3}\.jsonl\.zst$`, matches[0])
	assert.Equal(t, v3, string(readFile(t, matches[0])))
	assert.Contains(t, string(readFile(t, filepath.Join(dir, activeFileName))), `"day":"20261104"`)
}

func TestFilterHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer sk-1")
	h.Set("X-Api-Key", "sk-2")
	h.Set("X-Codex-Turn-State", "state")
	h.Set("Idempotency-Key", "abc")
	h.Set("Anthropic-Beta", "beta")
	kept, dropped := filterHeaders(h)
	assert.Equal(t, []string{"Authorization", "X-Api-Key"}, dropped)
	assert.Equal(t, "beta", kept["Anthropic-Beta"])
	assert.Equal(t, "state", kept["X-Codex-Turn-State"])
	assert.True(t, strings.HasPrefix(kept["Idempotency-Key"], "sha256:"))
	assert.Len(t, kept["Idempotency-Key"], len("sha256:")+16)
}

// fakeChannel is a model provider: it records what it got and answers with
// the given SSE body and status.
func fakeChannel(t *testing.T, status int, resp string) (*httptest.Server, *[]byte) {
	var got []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		got, _ = io.ReadAll(req.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Request-Id", "req_upstream")
		w.Header().Set("Set-Cookie", "session=secret")
		w.Header().Set("Chatgpt-Account-Id", "acct-plaintext")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, resp)
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

// relayHandler mimics new-api's relay: it rewrites the body, tries each
// channel in turn through the hooks, and streams the last answer to the
// client with message_delta usage patched in.
func relayHandler(t *testing.T, channels []*httptest.Server, rewrite func(map[string]any)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]any
		raw, _ := io.ReadAll(c.Request.Body)
		require.NoError(t, json.Unmarshal(raw, &body))
		rewrite(body)
		out, _ := json.Marshal(body)
		var resp *http.Response
		for i, srv := range channels {
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/messages?key=sk-query", bytes.NewReader(out))
			req.Header.Set("X-Api-Key", "sk-upstream")
			req.Header.Set("X-Sign", "sk-override")
			// The relay passes the effective header override's keys, which cover
			// the runtime override param_override's set_header writes, and the
			// headers pass_headers copied from the client unchanged.
			req.Header.Set("User-Agent", c.GetHeader("User-Agent"))
			req.Header.Set("X-Override-Me", "channel-value") // the client sent "client-value"
			attempt := ChannelRequest(c, req, i, 10+i, 14, "claude-upstream", []string{"x-sign", "user-agent", "x-override-me"})
			r, err := http.DefaultClient.Do(req)
			attempt.Response(r, err)
			require.NoError(t, err)
			resp = r
			if r.StatusCode < 400 {
				break
			}
			_, _ = io.ReadAll(r.Body)
			_ = r.Body.Close()
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		c.Header("Content-Type", "text/event-stream")
		c.Status(resp.StatusCode)
		_, _ = c.Writer.Write(bytes.Replace(data, []byte(`"usage":{"output_tokens":5}`), []byte(`"usage":{"output_tokens":5,"input_tokens":10}`), 1))
	}
}

func runRelay(t *testing.T, channels []*httptest.Server, rewrite func(map[string]any), body []byte) []callRecord {
	dir := t.TempDir()
	require.NoError(t, Init(Config{Enabled: true, Dir: dir, Instance: "test", Users: map[int]bool{7: true}}))
	t.Cleanup(Close)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(common.RequestIdKey, c.GetHeader("X-Rid"))
		common.SetContextKey(c, constant.ContextKeyUserId, 7)
		c.Next()
	})
	engine.Use(Middleware())
	engine.POST("/v1/messages", relayHandler(t, channels, rewrite))
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("User-Agent", "claude-cli/2.0.0")
	req.Header.Set("X-Claude-Code-Agent-Id", "agent-1")
	req.Header.Set("X-Override-Me", "client-value")
	req.Header.Set("X-Rid", "r1")
	engine.ServeHTTP(httptest.NewRecorder(), req)
	Close()
	var recs []callRecord
	for line := range bytes.Lines(mustRead(t, filepath.Join(dir, activeFileName))) {
		var rec callRecord
		require.NoError(t, common.Unmarshal(line, &rec))
		if rec.Type == "call" {
			recs = append(recs, rec)
		}
	}
	return recs
}

func TestChannelDelta(t *testing.T) {
	failing, _ := fakeChannel(t, http.StatusInternalServerError, `{"error":"overloaded"}`)
	ok, got := fakeChannel(t, http.StatusOK, testSSE)
	bigSystem := strings.Repeat("injected ", 200)
	rewrite := func(b map[string]any) {
		b["thinking"] = map[string]any{"type": "enabled", "budget_tokens": 1024, "display": "summarized"}
		b["system"] = bigSystem
		delete(b, "metadata")
	}
	body := claudeBody("S", `{"role":"user","content":"hi <b>"}`)
	recs := runRelay(t, []*httptest.Server{failing, ok}, rewrite, body)
	require.Len(t, recs, 1)
	rec := recs[0]
	assert.Equal(t, "complete", rec.EndReason)
	assert.Equal(t, "agent-1", rec.AgentId)
	require.Len(t, rec.Channel, 2)

	first := rec.Channel[0]
	assert.Equal(t, 500, first.Status)
	assert.Equal(t, `{"error":"overloaded"}`, *first.Resp)
	assert.False(t, first.RespComplete)

	a := rec.Channel[1]
	assert.Equal(t, 1, a.Attempt)
	assert.Equal(t, 11, a.ChannelId)
	assert.Equal(t, "claude-upstream", a.Model)
	assert.Equal(t, ok.URL+"/v1/messages", a.URL, "query left out")
	assert.Equal(t, []string{"X-Api-Key", "X-Override-Me", "X-Sign"}, a.HeadersDropped, "header override values are channel credentials")
	assert.NotContains(t, a.Headers, "X-Sign")
	assert.Equal(t, "claude-cli/2.0.0", a.Headers["User-Agent"], "a client header passed through is not a channel credential")
	assert.NotContains(t, a.Headers, "X-Override-Me", "the override replaced the client's value")
	assert.NotContains(t, a.RespHeaders, "Chatgpt-Account-Id")
	assert.Equal(t, []string{"Chatgpt-Account-Id", "Set-Cookie"}, a.RespHeadersDropped)
	assert.Equal(t, "req_upstream", a.RespHeaders["Request-Id"])
	assert.Equal(t, sha(*got), a.ReqSha256)
	assert.Equal(t, sha(canonical(*got)), a.ReqCanonicalSha256)
	require.NotNil(t, a.ReqDelta)
	assert.Equal(t, []string{"metadata"}, a.ReqDelta.Del)
	assert.Contains(t, a.ReqDelta.Refs, "system", "large values go to blobs")
	assert.JSONEq(t, `{"type":"enabled","budget_tokens":1024,"display":"summarized"}`, string(a.ReqDelta.Set["thinking"]))
	assert.Nil(t, a.ReqDelta.History, "history unchanged")
	assert.True(t, a.RespComplete)
	require.Len(t, a.RespLines, 1, "only the patched message_delta differs")
	assert.Equal(t, `{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":5}}`, a.RespLines[0].Data)

	// Applying the delta to the client body gives the channel body.
	var client, channel map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &client))
	require.NoError(t, json.Unmarshal(*got, &channel))
	for _, k := range a.ReqDelta.Del {
		delete(client, k)
	}
	for k, v := range a.ReqDelta.Set {
		client[k] = v
	}
	client["system"] = canonical([]byte(strconv.Quote(bigSystem)))
	assert.Equal(t, len(channel), len(client))
	for k := range channel {
		assert.Equal(t, string(canonical(channel[k])), string(canonical(client[k])), k)
	}
}

func TestChannelCutAndSameRequest(t *testing.T) {
	cut := testSSE[:strings.Index(testSSE, "event: message_delta")]
	srv, _ := fakeChannel(t, http.StatusOK, cut)
	recs := runRelay(t, []*httptest.Server{srv}, func(map[string]any) {}, claudeBody("S", `{"role":"user","content":"hi"}`))
	require.Len(t, recs, 1)
	assert.Equal(t, "channel_cut", recs[0].EndReason)
	a := recs[0].Channel[0]
	assert.True(t, a.ReqSame, "reserialized but equal as canonical JSON")
	assert.NotEmpty(t, a.ReqCanonicalSha256)
	assert.Nil(t, a.ReqDelta)
	assert.True(t, a.RespSame)
	assert.False(t, a.RespComplete)
}

func TestDiffRequestHistory(t *testing.T) {
	top := func(s string) map[string]json.RawMessage {
		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(s), &m))
		return m
	}
	noBlob := func(string, []byte) string { t.Fatal("no blob expected"); return "" }
	client := top(`{"model":"m","messages":[{"role":"user","content":"a"},{"role":"user","content":"b"}]}`)
	assert.Nil(t, diffRequest("messages", client, top(`{"messages":[{"content":"a","role":"user"}, {"content":"b","role":"user"}],"model":"m"}`), noBlob),
		"reserialized history is unchanged")
	d := diffRequest("messages", client, top(`{"model":"m","messages":[{"role":"user","content":"a"},{"role":"user","content":"B"},{"role":"user","content":"c"}]}`), noBlob)
	require.NotNil(t, d)
	require.NotNil(t, d.History)
	assert.Equal(t, 3, d.History.Len)
	assert.Equal(t, []string{"1", "2"}, slices.Sorted(maps.Keys(d.History.Set)))
	assert.Nil(t, d.Set)
	d = diffRequest("messages", client, top(`{"model":"m","messages":[{"role":"user","content":"a"}]}`), noBlob)
	require.NotNil(t, d.History, "a shorter history is a change")
	assert.Equal(t, 1, d.History.Len)
	assert.Nil(t, d.History.Set)
}

func TestDiffDataLinesInvalidUTF8(t *testing.T) {
	client := []byte("data: {\"a\":1}\n\ndata: {\"b\":2}\n\ndata: {\"c\":3}\n\n")
	channel := []byte("data: {\"a\":1}\n\ndata: {\"b\":\"\xff\"}\n\ndata: {\"c\":3}\n\n")
	assert.Nil(t, diffDataLines(client, channel), "a Go string would turn 0xff into U+FFFD")
}

func TestStatsLines(t *testing.T) {
	dir := t.TempDir()
	rate := 0.0
	require.NoError(t, Init(Config{Enabled: true, Dir: dir, Instance: "test", Users: map[int]bool{7: true}, SampleRate: &rate}))
	t.Cleanup(Close)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		user, _ := strconv.Atoi(c.GetHeader("X-User"))
		common.SetContextKey(c, constant.ContextKeyUserId, user)
		c.Next()
	})
	engine.Use(Middleware())
	engine.POST("/v1/messages", func(c *gin.Context) { c.Status(http.StatusOK) })
	for _, user := range []string{"7", "8", "8"} {
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader("{}"))
		req.Header.Set("X-User", user)
		engine.ServeHTTP(httptest.NewRecorder(), req)
	}
	Close()
	var stats []statsRecord
	for line := range bytes.Lines(mustRead(t, filepath.Join(dir, activeFileName))) {
		var s statsRecord
		require.NoError(t, common.Unmarshal(line, &s))
		if s.Type == "stats" {
			stats = append(stats, s)
		}
	}
	require.Len(t, stats, 1)
	assert.Equal(t, statsCounts{Seen: 3, Filtered: 2, SampledOut: 1}, stats[0].Families["messages"])
	assert.NotContains(t, stats[0].Families, "responses", "families without activity are left out")
}

// TestWriteFixture writes a small 录制日 for the Python reader's tests
// (test/python/atk_traj/traj_record/data/recording_v4/). Skipped unless
// TRAJ_RECORD_FIXTURE_DIR is set.
func TestWriteFixture(t *testing.T) {
	out := os.Getenv("TRAJ_RECORD_FIXTURE_DIR")
	if out == "" {
		t.Skip("TRAJ_RECORD_FIXTURE_DIR not set")
	}
	clock := &testClock{day1}
	r := newClockedRecorder(t, out, clock, Config{ZstdWindowMB: 1})
	u1 := `{"role":"user","content":"hi <b>"}`
	a1 := `{"role":"assistant","content":"hello"}`
	// Escapes and number spellings that Go and Python canonical JSON must agree on.
	u2 := `{"role":"user","content":"more\u2028\b\u001f\ud800 é","x":1.50,"y":1e2}`
	call := func(rid string, body []byte, channel []byte, resp string) {
		r.process(&job{
			client: claudeClient,
			rec: callRecord{Rid: rid, UserId: 7, TokenId: 1, ChannelId: 1, Model: "claude-test",
				Path: "/v1/messages", RespContentType: "text/event-stream", Status: 200, RespSize: len(testSSE),
				Headers: map[string]string{"User-Agent": "claude-cli/2.0.0", "X-Claude-Code-Agent-Id": "agent-1"}},
			body: body,
			resp: []byte(testSSE),
			channel: []rawAttempt{{
				rec:  channelAttempt{ChannelId: 1, ChannelType: 14, Status: 200, RespContentType: "text/event-stream"},
				body: channel,
				resp: []byte(resp),
			}},
		})
	}
	// history: 0 leaves it alone, >0 keeps that many items and rewrites the
	// first one, so the fixture covers an element change and a shorter history
	// while the later items (which carry the tricky escapes) stay comparable.
	rewrite := func(body []byte, system string, history int) []byte {
		var b map[string]any
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.UseNumber()
		require.NoError(t, dec.Decode(&b))
		b["thinking"] = map[string]any{"type": "enabled", "budget_tokens": 1024, "display": "summarized"}
		b["temperature"] = json.Number("0.10")
		b["system"] = system
		delete(b, "metadata")
		if history > 0 {
			msgs := b["messages"].([]any)
			msgs = msgs[:min(history, len(msgs))]
			msgs[0] = map[string]any{"role": "user", "content": "rewritten by the channel"}
			b["messages"] = msgs
		}
		out, _ := json.Marshal(b)
		return out
	}
	system := strings.Repeat("injected ", 200)
	b1 := claudeBody("S", u1)
	call("r1", b1, rewrite(b1, system, 0), testSSE)
	r.fw.maxBytes = 1 // the next call goes to the day's second file
	b2 := claudeBody("S", u1, a1, u2)
	call("r2", b2, rewrite(b2, system, 3), strings.Replace(testSSE, `"output_tokens":5`, `"output_tokens":6`, 1))
	// A third file of the same 录制日, where the channel keeps only part of the
	// history: the Python reader must rebuild a shortened history too.
	b2b := claudeBody("S", u1, a1, u2, a1)
	call("r2b", b2b, rewrite(b2b, system, 2), testSSE)
	r.fw.maxBytes = 1 << 30
	clock.t = day1.Add(2 * time.Minute)
	b3 := claudeBody("S", u1, a1, u2, a1)
	call("r3", b3, rewrite(b3, system, 2), testSSE[:100])
	require.NoError(t, r.fw.close())
}

func TestCountLoneSurrogates(t *testing.T) {
	for raw, want := range map[string]int{
		`{"a":"😀"}`:             0,
		`{"a":"\ud83d"}`:        1,
		`{"a":"\ude00x\ud83d"}`: 2,
		`{"a":"\ud83d😀"}`:       1,
		`{"a":"\\ud83d"}`:       0, // an escaped backslash, then text
		`{"a":"\ud83d\n"}`:      1,
		`{"a":"é é 𐀀"}`:         0,
	} {
		assert.Equal(t, want, countLoneSurrogates([]byte(raw)), raw)
	}
}

func TestCodexTurnMetadata(t *testing.T) {
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(`{"client_metadata":{"session_id":"s","x-codex-turn-metadata":"{\"request_kind\":\"turn\",\"thread_source\":\"guardian_review\",\"subagent_kind\":\"guardian\"}"}}`), &top))
	rec := callRecord{Headers: map[string]string{}}
	codexRequestIds(&rec, top)
	assert.Equal(t, "turn", rec.RequestKind)
	assert.Equal(t, "guardian_review", rec.ThreadSource)
}

func TestClaudeSessionHeaderFallback(t *testing.T) {
	rec := callRecord{Headers: map[string]string{"X-Claude-Code-Session-Id": "hs"}}
	assert.Equal(t, "hs", claudeRequestIds(&rec, map[string]json.RawMessage{}))
	assert.Equal(t, "hs", rec.SessionId)
	rec = callRecord{Headers: map[string]string{"X-Claude-Code-Session-Id": "hs"}}
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(claudeBody("S", `{"role":"user","content":"hi"}`), &top))
	assert.Equal(t, "S", claudeRequestIds(&rec, top), "metadata.user_id comes first")
}

func TestGenericSessionHeaderFallback(t *testing.T) {
	for _, cl := range clients {
		rec := callRecord{Headers: map[string]string{"X-Session-Id": "ses_x"}}
		requestIds(cl, &rec, map[string]json.RawMessage{})
		assert.Equal(t, "ses_x", rec.SessionId, cl.name)
	}
	rec := callRecord{Headers: map[string]string{"X-Session-Id": "ses_x"}}
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(claudeBody("S", `{"role":"user","content":"hi"}`), &top))
	requestIds(claudeClient, &rec, top)
	assert.Equal(t, "S", rec.SessionId, "the client's own id comes first")
}

// TestChannelResponseNotComparable covers the cases where canonical JSON
// comparison would swallow a real difference: the gateway's decoder stops at
// the first value and replaces invalid UTF-8 and lone surrogates.
func TestChannelResponseNotComparable(t *testing.T) {
	for name, channel := range map[string]string{
		"second document":  "{\"a\":1}\n{\"secret_second_doc\":true}",
		"trailing brace":   "{\"a\":1}}",
		"trailing bracket": "{\"a\":1}]",
		"invalid utf-8":    "{\"a\":\"\xff\"}",
		"lone surrogate":   `{"a":"\ud83d"}`,
	} {
		t.Run(name, func(t *testing.T) {
			client := []byte("{\"a\":1}")
			if name == "invalid utf-8" || name == "lone surrogate" {
				client = []byte("{\"a\":\"\ufffd\"}")
			}
			a := rawAttempt{rec: channelAttempt{RespContentType: "application/json"}, resp: []byte(channel)}
			rec := diffChannel(claudeClient, a, nil, nil, client, "application/json", nil)
			assert.False(t, rec.RespSame, "a real difference must not be dropped")
			require.True(t, rec.Resp != nil || rec.RespB64 != nil, "the whole response is stored")
			if rec.Resp != nil {
				assert.Equal(t, channel, *rec.Resp)
			} else {
				assert.Equal(t, channel, string(rec.RespB64))
			}
		})
	}
	// Reserialized but equal stays req_same/resp_same.
	a := rawAttempt{rec: channelAttempt{RespContentType: "application/json"}, resp: []byte(`{"b":2,"a":1}`)}
	rec := diffChannel(claudeClient, a, nil, nil, []byte(`{"a":1,"b":2}`), "application/json", nil)
	assert.True(t, rec.RespSame)
}

func TestManifestWaitsForCompression(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "traj-20261104-000.jsonl")
	require.NoError(t, os.WriteFile(plain, []byte("{}\n"), 0o640))
	err := writeManifest(dir, "test", "20261104", day1, true)
	require.Error(t, err, "an uncompressed file means compression failed; no manifest yet")
	assert.NoFileExists(t, filepath.Join(dir, manifestName("20261104")))
	require.NoError(t, os.Rename(plain, plain+".zst"))
	require.NoError(t, writeManifest(dir, "test", "20261104", day1, true))
	assert.FileExists(t, filepath.Join(dir, manifestName("20261104")))
}

// drainTasks waits for the background compression and manifest queue.
func drainTasks(w *fileWriter) {
	done := make(chan struct{})
	w.tasks <- func() { close(done) }
	<-done
}

func TestChannelRequestKeepsLoneSurrogate(t *testing.T) {
	client := []byte(`{"model":"m","messages":[{"role":"user","content":"x\ud83d"}]}`)
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(client, &top))
	// The channel changed the model but kept the lone surrogate.
	channel := []byte(`{"model":"m2","messages":[{"role":"user","content":"x\ud83d"}]}`)
	a := rawAttempt{rec: channelAttempt{}, body: channel}
	rec := diffChannel(claudeClient, a, client, top, nil, "", func(string, []byte) string { return "ref" })
	assert.False(t, rec.ReqSame)
	assert.Nil(t, rec.ReqDelta, "a delta would canonicalise the surrogate to U+FFFD")
	require.NotNil(t, rec.ReqRaw)
	assert.Equal(t, string(channel), *rec.ReqRaw)
	assert.Empty(t, rec.ReqCanonicalSha256, "nothing was canonicalised")
}

func TestSealRetriedAfterFailedCompression(t *testing.T) {
	dir := t.TempDir()
	clock := &testClock{day1}
	r := newClockedRecorder(t, dir, clock, Config{ZstdWindowMB: 1})
	process(r, "r1", claudeBody("S", `{"role":"user","content":"hi"}`))
	// Rotate into the next day with compression failing: the day gets no
	// manifest, rather than one naming a file the next startup recompresses.
	blocked := filepath.Join(dir, "traj-20261104-000.jsonl.zst.tmp")
	require.NoError(t, os.Mkdir(blocked, 0o500))
	clock.t = day1.Add(2 * time.Minute)
	process(r, "r2", claudeBody("T", `{"role":"user","content":"hi"}`))
	drainTasks(r.fw)
	assert.NoFileExists(t, filepath.Join(dir, manifestName("20261104")))
	assert.FileExists(t, filepath.Join(dir, "traj-20261104-000.jsonl"))

	// The next rotation retries it, without a restart.
	require.NoError(t, os.Remove(blocked))
	r.fw.maxBytes = 1
	process(r, "r3", claudeBody("T", `{"role":"user","content":"more"}`))
	drainTasks(r.fw)
	require.NoError(t, r.fw.close())
	m := map[string]any{}
	require.NoError(t, json.Unmarshal(readFile(t, filepath.Join(dir, manifestName("20261104"))), &m))
	files := m["files"].([]any)
	require.Len(t, files, 1)
	assert.Equal(t, "traj-20261104-000.jsonl.zst", files[0].(map[string]any)["name"])
}
