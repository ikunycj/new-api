package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/relaytrace"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
)

// sseChunk mimics one Anthropic content_block_delta event.
const sseChunk = `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello world this is a token"}}` + "\n\n"

// benchRouter builds a router that streams n chunks, optionally through RelayTrace.
func benchRouter(traced bool, chunks int) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(common.RequestIdKey, "bench-rid")
		common.SetContextKey(c, constant.ContextKeyIsStream, true)
		common.SetContextKey(c, constant.ContextKeyOriginalModel, "claude-opus-4-8")
		c.Next()
	})
	if traced {
		r.Use(RelayTrace())
	}
	r.POST("/v1/messages", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		for i := 0; i < chunks; i++ {
			_, _ = c.Writer.WriteString(sseChunk)
			c.Writer.Flush()
		}
	})
	return r
}

const benchPayload = `{"model":"claude-opus-4-8","max_tokens":4096,"messages":[{"role":"user","content":"explain quantum entanglement in detail with examples and analogies"}]}`

// BenchmarkStreamingTraceOff is the baseline: no middleware at all.
func BenchmarkStreamingTraceOff(b *testing.B) {
	relaytrace.Close()
	runStreamBench(b, benchRouter(false, 200))
}

// BenchmarkStreamingTraceDisabled measures the cost of the early-return path.
func BenchmarkStreamingTraceDisabled(b *testing.B) {
	if err := relaytrace.Init(relaytrace.Config{Enabled: false, Dir: b.TempDir()}); err != nil {
		b.Fatal(err)
	}
	defer relaytrace.Close()
	runStreamBench(b, benchRouter(true, 200))
}

// BenchmarkStreamingTraceOn measures full capture + submit under real disk IO.
func BenchmarkStreamingTraceOn(b *testing.B) {
	cfg := relaytrace.Config{
		Enabled: true, Dir: b.TempDir(), SampleRate: 100, ErrorAlways: true,
		MaxBodyKB: 256, QueueSize: 8192, MaxFileMB: 512, MaxBackups: 3,
		DetectSignature: true,
	}
	if err := relaytrace.Init(cfg); err != nil {
		b.Fatal(err)
	}
	defer relaytrace.Close()
	runStreamBench(b, benchRouter(true, 200))
}

func runStreamBench(b *testing.B, router *gin.Engine) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(benchPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer sk-bench")
		router.ServeHTTP(httptest.NewRecorder(), req)
	}
}

// BenchmarkCaptureOnly isolates the per-Write cost that sits on the first-token
// path, independent of queueing and disk IO.
func BenchmarkCaptureOnly(b *testing.B) {
	for _, size := range []int{128, 1024, 8192} {
		b.Run(fmt.Sprintf("chunk-%dB", size), func(b *testing.B) {
			// capture never touches the embedded writer, so leaving it nil keeps
			// this measurement free of recorder overhead. Do not call Write here.
			w := &traceResponseWriter{
				body:    bytes.NewBuffer(nil),
				maxSize: 256 << 10,
			}
			chunk := make([]byte, size)
			b.SetBytes(int64(size))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w.capture(chunk)
				if w.body.Len() > 200<<10 {
					w.body.Reset()
				}
			}
		})
	}
}
