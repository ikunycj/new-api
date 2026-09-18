package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/relaytrace"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
)

// traceResponseWriter mirrors response bytes into a bounded buffer while passing
// them through to the client unchanged.
//
// Correctness requirements for the relay path:
//   - Flush must reach the real writer, otherwise SSE stops streaming
//     incrementally and clients receive the whole answer at the end.
//   - Unwrap must be provided, otherwise http.NewResponseController cannot find
//     the deadline-capable writer and ExtendWriteDeadline in
//     relay/helper/stream_scanner.go silently stops working.
//   - Hijack must reach the real writer for the websocket realtime route.
//
// Embedding gin.ResponseWriter forwards Flush/Hijack/CloseNotify automatically;
// Unwrap is declared explicitly because it is not part of the interface.
type traceResponseWriter struct {
	gin.ResponseWriter
	body      *bytes.Buffer
	maxSize   int
	total     int
	truncated bool
}

func (w *traceResponseWriter) Write(b []byte) (int, error) {
	w.capture(b)
	return w.ResponseWriter.Write(b)
}

func (w *traceResponseWriter) WriteString(s string) (int, error) {
	w.capture([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

// capture copies up to maxSize bytes, tracking the true total so the record can
// report the real size even when the stored body is clipped.
func (w *traceResponseWriter) capture(b []byte) {
	w.total += len(b)
	remain := w.maxSize - w.body.Len()
	if remain <= 0 {
		w.truncated = true
		return
	}
	if len(b) > remain {
		w.body.Write(b[:remain])
		w.truncated = true
		return
	}
	w.body.Write(b)
}

// Unwrap exposes the underlying writer to http.ResponseController so write
// deadlines keep working through this wrapper.
func (w *traceResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// RelayTrace records the user-facing request and response of relay calls into a
// local JSONL file.
//
// Mount after TokenAuth (so user/token ids exist) and before Distribute (so the
// wrapper is in place before any handler writes). When tracing is disabled the
// middleware returns immediately and allocates nothing.
func RelayTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !relaytrace.Enabled() {
			c.Next()
			return
		}

		cfg := relaytrace.ActiveConfig()
		maxBody := cfg.MaxBodyBytes()
		start := time.Now()

		// Read the body up front. GetBodyStorage caches it on the context, and
		// Bytes() does not disturb the read offset, so downstream handlers still
		// see a complete body. This is the single most important invariant here.
		reqBody, reqSize, reqTruncated := readRequestBody(c, maxBody)

		writer := &traceResponseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
			maxSize:        maxBody,
		}
		c.Writer = writer

		c.Next()

		// Restore the original writer so later middleware in the chain does not
		// keep writing through a wrapper that outlived its purpose.
		c.Writer = writer.ResponseWriter

		status := writer.Status()
		userId := common.GetContextKeyInt(c, constant.ContextKeyUserId)
		model := common.GetContextKeyString(c, constant.ContextKeyOriginalModel)

		if !relaytrace.ShouldRecord(status, userId, model) {
			return
		}

		respBytes := writer.body.Bytes()
		rec := &relaytrace.Record{
			Rid:           c.GetString(common.RequestIdKey),
			Ts:            start.UnixMilli(),
			Status:        status,
			DurationMs:    time.Since(start).Milliseconds(),
			Method:        c.Request.Method,
			Path:          c.Request.URL.Path,
			Model:         model,
			Stream:        common.GetContextKeyBool(c, constant.ContextKeyIsStream),
			ReqHeaders:    relaytrace.FilterHeaders(c.GetHeader),
			ReqSize:       reqSize,
			RespSize:      writer.total,
			ReqTruncated:  reqTruncated,
			RespTruncated: writer.truncated,
			Resp:          string(respBytes),
		}

		// Inline valid JSON verbatim; fall back to a plain string for multipart
		// uploads and malformed payloads so the line stays parseable either way.
		if json.Valid(reqBody) {
			rec.Req = json.RawMessage(reqBody)
		} else if len(reqBody) > 0 {
			rec.ReqRaw = string(reqBody)
		}

		// Derived flag so "which channel strips thinking signatures" is a single
		// jq filter rather than a re-parse of every stored SSE stream. Only set
		// when the key actually appears, because its absence is normal for
		// models that never emit signatures and must not look like a stripped one.
		if cfg.DetectSignature {
			if hasSig, present := relaytrace.DetectSignature(respBytes); present {
				rec.HasSignature = &hasSig
			}
		}

		relaytrace.Submit(rec)
	}
}

// readRequestBody returns a copy of the request body bounded by maxBody.
//
// Returns the (possibly clipped) bytes, the true size, and whether clipping
// occurred. Failures are non-fatal: tracing degrades to an empty body rather
// than breaking the request.
func readRequestBody(c *gin.Context, maxBody int) ([]byte, int, bool) {
	if c.Request == nil || c.Request.Body == nil || c.Request.Body == http.NoBody {
		return nil, 0, false
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, 0, false
	}
	full, err := storage.Bytes()
	if err != nil {
		return nil, 0, false
	}
	// GetBodyStorage drains c.Request.Body into its own buffer. Relay handlers
	// go through UnmarshalBodyReusable and read the storage, but any handler
	// that touches c.Request.Body directly would otherwise see an empty body.
	// Restore it so this middleware is transparent to every caller.
	c.Request.Body = io.NopCloser(bytes.NewReader(full))
	// Rewind the storage as well, so sequential readers start from byte zero.
	if _, err := storage.Seek(0, io.SeekStart); err != nil {
		return nil, 0, false
	}
	size := len(full)
	if size > maxBody {
		clipped := make([]byte, maxBody)
		copy(clipped, full[:maxBody])
		return clipped, size, true
	}
	out := make([]byte, size)
	copy(out, full)
	return out, size, false
}
