package traj_record

import (
	"bytes"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"

	"github.com/gin-gonic/gin"
)

// captureWriter mirrors the full response while passing it through unchanged.
//
// Embedding gin.ResponseWriter forwards Flush/Hijack/CloseNotify, so SSE keeps
// streaming incrementally. Unwrap is declared explicitly so
// http.ResponseController still finds the deadline-capable writer used by
// relay/helper/stream_scanner.go.
type captureWriter struct {
	gin.ResponseWriter
	body      bytes.Buffer
	maxSize   int
	total     int
	truncated bool
	firstByte time.Time
}

func (w *captureWriter) Write(b []byte) (int, error) {
	w.capture(b)
	return w.ResponseWriter.Write(b)
}

func (w *captureWriter) WriteString(s string) (int, error) {
	w.capture([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

func (w *captureWriter) capture(b []byte) {
	if w.firstByte.IsZero() && len(b) > 0 {
		w.firstByte = time.Now()
	}
	w.total += len(b)
	remain := w.maxSize - w.body.Len()
	if len(b) > remain {
		b = b[:max(remain, 0)]
		w.truncated = true
	}
	w.body.Write(b)
}

func (w *captureWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// headerDenySubstrings keeps credentials out of the files. Matching on
// substrings rather than an exact list also covers custom auth headers.
var headerDenySubstrings = []string{"auth", "key", "token", "secret", "cookie", "password"}

// Middleware records the calls of the enabled clients (see clients.go).
//
// Mount after TokenAuth (user and token ids exist) and before Distribute (the
// writer is wrapped before any handler writes). Requests from other clients
// pass through without any allocation.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		r := current()
		if r == nil || c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		rt, ok := r.routes[c.Request.URL.Path]
		ua := c.Request.UserAgent()
		if !ok || !matchesUA(rt, ua) {
			c.Next()
			return
		}
		userId := common.GetContextKeyInt(c, constant.ContextKeyUserId)
		if len(r.cfg.Users) > 0 && !r.cfg.Users[userId] {
			c.Next()
			return
		}

		start := time.Now()
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			c.Next()
			return
		}
		full, err := storage.Bytes()
		if err != nil {
			c.Next()
			return
		}
		// GetBodyStorage drained c.Request.Body; restore it and rewind the
		// storage so every downstream reader still sees the complete body.
		c.Request.Body = io.NopCloser(bytes.NewReader(full))
		if _, err := storage.Seek(0, io.SeekStart); err != nil {
			c.Next()
			return
		}

		if !r.sampleRequest(rt.client, userId, c.Request.Header, full) {
			c.Next()
			return
		}
		// Copy only selected requests; storage is released after the handler.
		body := bytes.Clone(full)

		writer := &captureWriter{ResponseWriter: c.Writer, maxSize: r.cfg.MaxRespMB << 20}
		c.Writer = writer

		// Deferred so the record survives a panicking handler.
		defer func() {
			c.Writer = writer.ResponseWriter
			r.submit(newJob(c, rt.client, writer, body, start))
		}()
		c.Next()
	}
}

// newJob collects every cheap clue available on the gin context. Anything
// that needs parsing is left to the writer goroutine.
func newJob(c *gin.Context, cl *client, w *captureWriter, body []byte, start time.Time) *job {
	end := time.Now()
	headers := map[string]string{}
	for name, values := range c.Request.Header {
		lower := strings.ToLower(name)
		if !slices.ContainsFunc(headerDenySubstrings, func(sub string) bool { return strings.Contains(lower, sub) }) {
			headers[name] = strings.Join(values, ", ")
		}
	}
	channelSetting, _ := common.GetContextKeyType[dto.ChannelSettings](c, constant.ContextKeyChannelSetting)
	modelMapping := common.GetContextKeyString(c, constant.ContextKeyChannelModelMapping)
	if modelMapping == "{}" {
		modelMapping = ""
	}
	overrideOps, bodyOverride := summarizeParamOverride(common.GetContextKeyStringMap(c, constant.ContextKeyChannelParamOverride))
	headerOverrideKeys := slices.Sorted(maps.Keys(common.GetContextKeyStringMap(c, constant.ContextKeyChannelHeaderOverride)))
	rec := callRecord{
		Rid:          c.GetString(common.RequestIdKey),
		TsStart:      start.UnixMilli(),
		TsEnd:        end.UnixMilli(),
		Method:       c.Request.Method,
		Path:         c.Request.URL.Path,
		Status:       w.Status(),
		Stream:       common.GetContextKeyBool(c, constant.ContextKeyIsStream),
		ClientIP:     c.ClientIP(),
		Headers:      headers,
		UserId:       common.GetContextKeyInt(c, constant.ContextKeyUserId),
		Username:     common.GetContextKeyString(c, constant.ContextKeyUserName),
		UserGroup:    common.GetContextKeyString(c, constant.ContextKeyUserGroup),
		TokenId:      common.GetContextKeyInt(c, constant.ContextKeyTokenId),
		TokenName:    c.GetString("token_name"),
		Group:        common.GetContextKeyString(c, constant.ContextKeyUsingGroup),
		ChannelId:    common.GetContextKeyInt(c, constant.ContextKeyChannelId),
		ChannelName:  common.GetContextKeyString(c, constant.ContextKeyChannelName),
		ChannelType:  common.GetContextKeyInt(c, constant.ContextKeyChannelType),
		ChannelTries: c.GetStringSlice("use_channel"),
		Upstream: upstreamInfo{
			PassThrough:          model_setting.GetGlobalSettings().PassThroughRequestEnabled || channelSetting.PassThroughBodyEnabled,
			SystemPrompt:         channelSetting.SystemPrompt != "",
			SystemPromptOverride: common.GetContextKeyBool(c, constant.ContextKeySystemPromptOverride),
			ParamOverrideOps:     overrideOps,
			BodyOverride:         bodyOverride,
			HeaderOverrideKeys:   headerOverrideKeys,
			ModelMapping:         modelMapping,
		},
		Model:           common.GetContextKeyString(c, constant.ContextKeyOriginalModel),
		RespContentType: w.Header().Get("Content-Type"),
		RespSize:        w.total,
		RespTruncated:   w.truncated,
		ClientCanceled:  c.Request.Context().Err() != nil,
	}
	if !w.firstByte.IsZero() {
		rec.TsFirstByte = w.firstByte.UnixMilli()
	}
	return &job{client: cl, rec: rec, body: body, resp: w.body.Bytes()}
}

// headerOnlyOps are param override modes that never touch the request body.
var headerOnlyOps = []string{"set_header", "delete_header", "copy_header", "move_header", "pass_headers"}

// summarizeParamOverride lists the operations relay/common.ApplyParamOverride
// would run: legacy top-level keys (each a body set) first, then the
// "operations" array.
func summarizeParamOverride(override map[string]any) (ops []overrideOp, bodyOverride bool) {
	for _, key := range slices.Sorted(maps.Keys(override)) {
		if !strings.EqualFold(strings.TrimSpace(key), "operations") {
			ops = append(ops, overrideOp{Mode: "set", Path: key, Value: override[key]})
		}
	}
	var rawOps []map[string]any
	switch v := override["operations"].(type) {
	case []map[string]any:
		rawOps = v
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				rawOps = append(rawOps, m)
			}
		}
	}
	for _, m := range rawOps {
		op := overrideOp{Value: m["value"], Conditional: m["conditions"] != nil}
		op.Mode, _ = m["mode"].(string)
		op.Path, _ = m["path"].(string)
		op.From, _ = m["from"].(string)
		op.To, _ = m["to"].(string)
		if op.Mode == "set_header" {
			op.Value = nil
		}
		ops = append(ops, op)
	}
	for _, op := range ops {
		if !slices.Contains(headerOnlyOps, op.Mode) {
			bodyOverride = true
		}
	}
	return ops, bodyOverride
}
