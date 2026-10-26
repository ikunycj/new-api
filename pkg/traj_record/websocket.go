package traj_record

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

// WebSocketCapture records application messages, not TCP frames. A connection
// is a transport identity, never a conversation identity. Capture all messages
// on selected connections, including controls and malformed payloads.
type WebSocketCapture struct {
	mu              sync.Mutex
	recorder        *recorder
	id              string
	seq             int64
	userID, tokenID int
}

type wsRecord struct {
	Instance     string      `json:"instance"`
	V            int         `json:"v"`
	Type         string      `json:"type"`
	APIFamily    string      `json:"api_family"`
	Transport    string      `json:"transport"`
	ConnectionID string      `json:"connection_id"`
	Seq          int64       `json:"connection_seq"`
	Ts           int64       `json:"ts"`
	UserID       int         `json:"user_id"`
	TokenID      int         `json:"token_id"`
	Direction    string      `json:"direction,omitempty"`
	MessageType  int         `json:"message_type,omitempty"`
	Payload      string      `json:"payload,omitempty"`
	PayloadB64   []byte      `json:"payload_b64,omitempty"`
	Size         int         `json:"size,omitempty"`
	Truncated    bool        `json:"truncated,omitempty"`
	IOFailed     bool        `json:"io_failed,omitempty"`
	Call         *callRecord `json:"call,omitempty"`
}

func BeginWebSocket(c *gin.Context) *WebSocketCapture {
	r := current()
	if r == nil {
		return nil
	}
	rt, ok := r.routes["/v1/responses"]
	if !ok || !matchesUA(rt, c.Request.UserAgent()) {
		return nil
	}
	user := common.GetContextKeyInt(c, constant.ContextKeyUserId)
	if len(r.cfg.Users) > 0 && !r.cfg.Users[user] {
		return nil
	}
	// A connection may contain many sessions or no session metadata at all.
	// Like unknown HTTP sessions it is retained in full unless rate is zero.
	if r.cfg.SampleRate != nil && *r.cfg.SampleRate <= 0 {
		return nil
	}
	w := &WebSocketCapture{recorder: r, id: common.NewRequestId(), userID: user, tokenID: common.GetContextKeyInt(c, constant.ContextKeyTokenId)}
	w.emit(&wsRecord{Type: "ws_open"})
	return w
}

func (w *WebSocketCapture) emit(ev *wsRecord) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.seq++
	ev.V, ev.APIFamily, ev.Transport = recordVersion, "responses", "websocket"
	ev.Instance = w.recorder.cfg.Instance
	ev.ConnectionID, ev.Seq, ev.Ts = w.id, w.seq, time.Now().UnixMilli()
	ev.UserID, ev.TokenID = w.userID, w.tokenID
	w.recorder.submit(&job{ws: ev})
}

func (w *WebSocketCapture) Message(direction string, kind int, body []byte, err error) {
	if w == nil {
		return
	}
	ev := &wsRecord{Type: "ws_message", Direction: direction, MessageType: kind, Size: len(body), IOFailed: err != nil}
	limit := w.recorder.cfg.MaxRespMB << 20
	if len(body) > limit {
		body = body[:limit]
		ev.Truncated = true
	}
	if utf8.Valid(body) {
		ev.Payload = string(body)
	} else {
		ev.PayloadB64 = bytes.Clone(body)
	}
	w.emit(ev)
}

// Call attaches authenticated per-request/channel metadata to the original
// response.create envelope. Response messages remain in the ordered event log.
func (w *WebSocketCapture) Call(c *gin.Context, body []byte, started time.Time) {
	if w == nil {
		return
	}
	j := newJob(c, codexClient, &captureWriter{ResponseWriter: c.Writer}, nil, started)
	j.rec.Transport, j.rec.APIFamily, j.rec.Type = "websocket", "responses", "call_metadata"
	j.rec.V, j.rec.Instance, j.rec.Client = recordVersion, w.recorder.cfg.Instance, "codex"
	// This is not an HTTP response; errors/status live in the native events.
	j.rec.Status = 0
	j.rec.ReqSize = len(body)
	sum := sha256.Sum256(body)
	j.rec.ReqSha256 = hex.EncodeToString(sum[:])
	ev := &wsRecord{Type: "ws_call", Call: &j.rec, Size: len(body)}
	limit := w.recorder.cfg.MaxRespMB << 20
	if len(body) > limit {
		body = body[:limit]
		ev.Truncated = true
	}
	if utf8.Valid(body) {
		ev.Payload = string(body)
	} else {
		ev.PayloadB64 = bytes.Clone(body)
	}
	w.emit(ev)
}

func (w *WebSocketCapture) Close() {
	if w != nil {
		w.emit(&wsRecord{Type: "ws_close"})
	}
}
