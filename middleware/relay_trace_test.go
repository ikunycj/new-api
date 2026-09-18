package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/relaytrace"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// enableTrace starts tracing into a temp dir and tears it down after the test.
func enableTrace(t *testing.T, tweak func(*relaytrace.Config)) string {
	t.Helper()
	dir := t.TempDir()
	cfg := relaytrace.Config{
		Enabled:         true,
		Dir:             dir,
		SampleRate:      100,
		ErrorAlways:     true,
		MaxBodyKB:       256,
		QueueSize:       1024,
		MaxFileMB:       512,
		MaxBackups:      5,
		DetectSignature: true,
	}
	if tweak != nil {
		tweak(&cfg)
	}
	if err := relaytrace.Init(cfg); err != nil {
		t.Fatalf("relaytrace.Init: %v", err)
	}
	t.Cleanup(relaytrace.Close)
	return dir
}

// readRecords flushes the tracer and returns the persisted records.
func readRecords(t *testing.T, dir string) []relaytrace.Record {
	t.Helper()
	relaytrace.Close() // drain + flush so assertions see everything

	f, err := os.Open(filepath.Join(dir, "trace.jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("open trace file: %v", err)
	}
	defer f.Close()

	var out []relaytrace.Record
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r relaytrace.Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("invalid JSONL line: %v", err)
		}
		out = append(out, r)
	}
	return out
}

// newTraceRouter builds a router with RelayTrace mounted ahead of handler.
func newTraceRouter(handler gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(common.RequestIdKey, "test-request-id")
		c.Next()
	})
	r.Use(RelayTrace())
	r.POST("/v1/chat/completions", handler)
	r.POST("/v1/messages", handler)
	return r
}

// TestBodyRestoredForDownstream is the most important test in this file: if the
// middleware consumed the body, every relay request would break.
func TestBodyRestoredForDownstream(t *testing.T) {
	dir := enableTrace(t, nil)

	const payload = `{"model":"gpt-4o","messages":[{"role":"user","content":"hello world"}]}`
	var seenByHandler string

	router := newTraceRouter(func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Errorf("handler could not read body: %v", err)
		}
		seenByHandler = string(body)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if seenByHandler != payload {
		t.Fatalf("downstream handler got a damaged body.\nwant: %s\ngot:  %s", payload, seenByHandler)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	records := readRecords(t, dir)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if string(records[0].Req) != payload {
		t.Fatalf("recorded request body mismatch: %s", string(records[0].Req))
	}
}

// TestUnwrapChainIntact guards ExtendWriteDeadline in the stream scanner: if the
// wrapper hides the underlying writer, write deadlines silently stop applying.
func TestUnwrapChainIntact(t *testing.T) {
	enableTrace(t, nil)

	var deadlineErr error
	router := newTraceRouter(func(c *gin.Context) {
		deadlineErr = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(time.Minute))
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))
	router.ServeHTTP(newDeadlineRecorder(), req)

	if deadlineErr != nil {
		t.Fatalf("SetWriteDeadline failed through the trace wrapper: %v", deadlineErr)
	}
}

// deadlineRecorder is an httptest.ResponseRecorder that supports write deadlines,
// mimicking a real net/http connection.
type deadlineRecorder struct {
	*httptest.ResponseRecorder
}

func newDeadlineRecorder() *deadlineRecorder {
	return &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (d *deadlineRecorder) SetWriteDeadline(time.Time) error { return nil }
func (d *deadlineRecorder) SetReadDeadline(time.Time) error  { return nil }

// TestStreamingFlushPassthrough verifies SSE chunks reach the client as they are
// produced rather than being withheld until the handler returns.
func TestStreamingFlushPassthrough(t *testing.T) {
	dir := enableTrace(t, nil)

	chunks := []string{
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"He\"}}\n\n",
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"llo\"}}\n\n",
		"data: [DONE]\n\n",
	}

	router := newTraceRouter(func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyIsStream, true)
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		for _, chunk := range chunks {
			if _, err := c.Writer.WriteString(chunk); err != nil {
				t.Errorf("write chunk: %v", err)
			}
			c.Writer.Flush()
		}
	})

	rec := &flushCountingRecorder{ResponseRecorder: httptest.NewRecorder()}
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"stream":true}`))
	router.ServeHTTP(rec, req)

	if rec.flushes < len(chunks) {
		t.Fatalf("expected at least %d flushes to reach the client, got %d (SSE would appear frozen)", len(chunks), rec.flushes)
	}

	body := rec.Body.String()
	for _, chunk := range chunks {
		if !strings.Contains(body, chunk) {
			t.Fatalf("client did not receive chunk %q", chunk)
		}
	}

	records := readRecords(t, dir)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if !records[0].Stream {
		t.Error("record should be marked as stream")
	}
	// The stored response must be the raw SSE stream, not a reassembled body.
	if !strings.Contains(records[0].Resp, "[DONE]") {
		t.Errorf("recorded response missing SSE terminator: %q", records[0].Resp)
	}
}

type flushCountingRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (f *flushCountingRecorder) Flush() {
	f.flushes++
	f.ResponseRecorder.Flush()
}

// TestSensitiveHeadersRedacted ensures credentials never reach disk.
func TestSensitiveHeadersRedacted(t *testing.T) {
	dir := enableTrace(t, nil)

	router := newTraceRouter(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer sk-super-secret")
	req.Header.Set("X-Api-Key", "sk-ant-super-secret")
	req.Header.Set("Cookie", "session=super-secret")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), req)

	raw, err := os.ReadFile(filepath.Join(dir, "trace.jsonl"))
	relaytrace.Close()
	if err != nil {
		raw, err = os.ReadFile(filepath.Join(dir, "trace.jsonl"))
		if err != nil {
			t.Fatalf("read trace file: %v", err)
		}
	}
	// Scan the raw file: a leak anywhere in the line matters, not just in
	// the headers map.
	for _, secret := range []string{"sk-super-secret", "sk-ant-super-secret", "session=super-secret"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("credential %q leaked into the trace file", secret)
		}
	}
}

// TestErrorAlwaysSampling checks the recommended production setting: record
// nothing on success, everything on failure.
func TestErrorAlwaysSampling(t *testing.T) {
	dir := enableTrace(t, func(c *relaytrace.Config) {
		c.SampleRate = 0
		c.ErrorAlways = true
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(common.RequestIdKey, "rid")
		c.Next()
	})
	router.Use(RelayTrace())
	router.POST("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	router.POST("/fail", func(c *gin.Context) { c.JSON(http.StatusInternalServerError, gin.H{"error": "boom"}) })

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/ok", strings.NewReader(`{}`)))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/fail", strings.NewReader(`{}`)))

	records := readRecords(t, dir)
	if len(records) != 1 {
		t.Fatalf("expected only the failing request to be recorded, got %d records", len(records))
	}
	if records[0].Status != http.StatusInternalServerError {
		t.Fatalf("expected the 500 to be recorded, got status %d", records[0].Status)
	}
}

// TestContextFieldsCaptured verifies the metadata that makes traces joinable
// back to the logs table.
func TestContextFieldsCaptured(t *testing.T) {
	dir := enableTrace(t, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(common.RequestIdKey, "rid-12345")
		common.SetContextKey(c, constant.ContextKeyUserId, 38)
		common.SetContextKey(c, constant.ContextKeyOriginalModel, "claude-opus-4-8")
		c.Next()
	})
	router.Use(RelayTrace())
	router.POST("/v1/messages", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`)))

	records := readRecords(t, dir)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.Rid != "rid-12345" {
		t.Errorf("rid = %q, want rid-12345 (needed to join the logs table)", r.Rid)
	}
	if r.Model != "claude-opus-4-8" {
		t.Errorf("model = %q, want claude-opus-4-8", r.Model)
	}
	if r.Path != "/v1/messages" {
		t.Errorf("path = %q, want /v1/messages", r.Path)
	}
	if r.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", r.Method)
	}
}

// TestSignatureDetection covers the derived has_signature field across the three
// meaningful cases.
func TestSignatureDetection(t *testing.T) {
	cases := []struct {
		name     string
		respBody string
		wantSet  bool
		wantVal  bool
	}{
		{
			name:     "signature present and non-empty",
			respBody: `data: {"delta":{"type":"signature_delta","signature":"EqQBCkYIARgC"}}`,
			wantSet:  true,
			wantVal:  true,
		},
		{
			// The channel-is-dirty case worth alerting on.
			name:     "signature stripped to empty",
			respBody: `data: {"delta":{"type":"signature_delta","signature":""}}`,
			wantSet:  true,
			wantVal:  false,
		},
		{
			// Normal for non-thinking models: must stay nil so it is not
			// mistaken for a stripped signature.
			name:     "no signature key at all",
			respBody: `{"choices":[{"message":{"content":"hi"}}]}`,
			wantSet:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := enableTrace(t, nil)
			router := newTraceRouter(func(c *gin.Context) {
				c.String(http.StatusOK, tc.respBody)
			})
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`)))

			records := readRecords(t, dir)
			if len(records) != 1 {
				t.Fatalf("expected 1 record, got %d", len(records))
			}
			got := records[0].HasSignature
			if !tc.wantSet {
				if got != nil {
					t.Fatalf("has_signature should be absent when no signature key exists, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("has_signature should be set")
			}
			if *got != tc.wantVal {
				t.Fatalf("has_signature = %v, want %v", *got, tc.wantVal)
			}
		})
	}
}

// TestTruncation verifies oversized bodies are clipped, flagged, and that the
// true size is still reported.
func TestTruncation(t *testing.T) {
	dir := enableTrace(t, func(c *relaytrace.Config) {
		c.MaxBodyKB = 1 // 1KB cap
	})

	bigContent := strings.Repeat("A", 8<<10)
	payload := `{"model":"gpt-4o","content":"` + bigContent + `"}`

	router := newTraceRouter(func(c *gin.Context) {
		c.String(http.StatusOK, strings.Repeat("B", 8<<10))
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(payload)))

	records := readRecords(t, dir)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if !r.ReqTruncated || !r.RespTruncated {
		t.Fatalf("expected both bodies flagged truncated, got req=%v resp=%v", r.ReqTruncated, r.RespTruncated)
	}
	// Sizes must reflect reality, not the clipped copy.
	if r.ReqSize != len(payload) {
		t.Errorf("req_size = %d, want the true size %d", r.ReqSize, len(payload))
	}
	if r.RespSize != 8<<10 {
		t.Errorf("resp_size = %d, want the true size %d", r.RespSize, 8<<10)
	}
	if len(r.Resp) > 1<<10 {
		t.Errorf("stored response should be clipped to 1KB, got %d bytes", len(r.Resp))
	}
	// A truncated JSON body is no longer valid JSON, so it must degrade to the
	// raw field instead of producing a broken `req` value.
	if len(r.Req) > 0 && !json.Valid(r.Req) {
		t.Error("req must never contain invalid JSON")
	}
}

// TestDisabledLeavesRequestUntouched confirms the zero-cost disabled path.
func TestDisabledLeavesRequestUntouched(t *testing.T) {
	dir := t.TempDir()
	if err := relaytrace.Init(relaytrace.Config{Enabled: false, Dir: dir}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(relaytrace.Close)

	const payload = `{"model":"gpt-4o"}`
	var seen string
	router := newTraceRouter(func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		seen = string(body)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(payload)))

	if seen != payload {
		t.Fatalf("body altered while tracing was disabled: %q", seen)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if _, err := os.Stat(filepath.Join(dir, "trace.jsonl")); !os.IsNotExist(err) {
		t.Fatal("no trace file should be created while disabled")
	}
}

// TestNonJSONBodyFallback covers multipart audio uploads.
func TestNonJSONBodyFallback(t *testing.T) {
	dir := enableTrace(t, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(common.RequestIdKey, "rid"); c.Next() })
	router.Use(RelayTrace())
	router.POST("/v1/audio/transcriptions", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"text": "hi"}) })

	body := "--boundary\r\nContent-Disposition: form-data; name=\"file\"\r\n\r\nnot-json\r\n--boundary--"
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	router.ServeHTTP(httptest.NewRecorder(), req)

	records := readRecords(t, dir)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if len(records[0].Req) != 0 {
		t.Errorf("non-JSON body must not populate req, got %s", string(records[0].Req))
	}
	if !strings.Contains(records[0].ReqRaw, "not-json") {
		t.Errorf("non-JSON body should land in req_raw, got %q", records[0].ReqRaw)
	}
}

// TestHijackSupported keeps the websocket realtime route working through the
// wrapper.
func TestHijackSupported(t *testing.T) {
	enableTrace(t, nil)

	var hijackable bool
	router := newTraceRouter(func(c *gin.Context) {
		_, hijackable = c.Writer.(http.Hijacker)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`)))

	if !hijackable {
		t.Fatal("trace wrapper must remain an http.Hijacker for websocket upgrades")
	}
}
