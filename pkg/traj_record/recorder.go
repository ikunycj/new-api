package traj_record

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

// recordVersion 2 added client and history_key, and replaced system_ref and
// tools_ref with refs.
const recordVersion = 2

// job is one finished request handed from the request goroutine to the writer.
// Full parsing happens on the writer side; partial sampling reads session IDs
// on the request path before copying bytes.
type job struct {
	client *client
	rec    callRecord
	body   []byte
	resp   []byte
}

// callRecord is the "call" line. Fields filled in the request goroutine are
// clues for offline linking; the rest is derived by the writer. The raw
// request parts ("params", "append") are spliced in after marshaling, see
// write.
type callRecord struct {
	V        int    `json:"v"`
	Type     string `json:"type"`
	Rid      string `json:"rid"`
	Instance string `json:"instance"`
	Seq      int64  `json:"seq"`
	Client   string `json:"client"`

	TsStart     int64 `json:"ts_start"`
	TsFirstByte int64 `json:"ts_first_byte,omitempty"`
	TsEnd       int64 `json:"ts_end"`

	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Status   int               `json:"status"`
	Stream   bool              `json:"stream"`
	ClientIP string            `json:"client_ip,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`

	UserId       int          `json:"user_id"`
	Username     string       `json:"username,omitempty"`
	UserGroup    string       `json:"user_group,omitempty"`
	TokenId      int          `json:"token_id"`
	TokenName    string       `json:"token_name,omitempty"`
	Group        string       `json:"group,omitempty"`
	ChannelId    int          `json:"channel_id,omitempty"`
	ChannelName  string       `json:"channel_name,omitempty"`
	ChannelType  int          `json:"channel_type,omitempty"`
	ChannelTries []string     `json:"channel_tries,omitempty"`
	Upstream     upstreamInfo `json:"upstream"`
	Model        string       `json:"model,omitempty"`

	SessionId   string `json:"session_id,omitempty"`
	DeviceId    string `json:"device_id,omitempty"`
	AccountUuid string `json:"account_uuid,omitempty"`
	// Codex only: the thread within the session (subagents get their own),
	// the thread that spawned it, and the user turn the call belongs to.
	ThreadId       string `json:"thread_id,omitempty"`
	ParentThreadId string `json:"parent_thread_id,omitempty"`
	TurnId         string `json:"turn_id,omitempty"`

	ReqSize      int    `json:"req_size"`
	ReqSha256    string `json:"req_sha256"`
	ReqCompacted bool   `json:"req_compacted,omitempty"`
	// HistoryKey is the request key whose array is stored as parent prefix
	// plus "append": messages for Claude, input for Codex.
	HistoryKey string `json:"history_key"`
	NHistory   int    `json:"n_history"`
	ParentRid  string `json:"parent_rid,omitempty"`
	PrefixLen  int    `json:"prefix_len"`
	// Refs maps each blob key present in the request (system, tools,
	// instructions) to the hash of its blob line.
	Refs map[string]string `json:"refs,omitempty"`
	// KeyOrder lists the top-level keys in body order. Present only when
	// reassembling the stored parts in this order reproduces the (compacted)
	// body byte for byte, so offline code can verify it against ReqSha256.
	KeyOrder []string `json:"key_order,omitempty"`
	// Set only when the body is not a JSON object with a history array.
	ReqRaw    string `json:"req_raw,omitempty"`
	ReqRawB64 []byte `json:"req_raw_b64,omitempty"`

	RespContentType string `json:"resp_content_type,omitempty"`
	RespSize        int    `json:"resp_size"`
	RespTruncated   bool   `json:"resp_truncated,omitempty"`
	RespComplete    bool   `json:"resp_complete"`
	// ClientCanceled is set when the client had gone away by the time the
	// handler returned, telling an aborted call (the agent interrupted or
	// restarted the turn) apart from an upstream that stopped mid-stream.
	ClientCanceled bool            `json:"client_canceled,omitempty"`
	RespId         string          `json:"resp_id,omitempty"`
	RespModel      string          `json:"resp_model,omitempty"`
	StopReason     string          `json:"stop_reason,omitempty"`
	UsageStart     json.RawMessage `json:"usage_start,omitempty"`
	Usage          json.RawMessage `json:"usage,omitempty"`
	// ToolUses lists tool_use ids (Claude) or call_ids (Codex) with names.
	ToolUses []toolUse `json:"resp_tool_uses,omitempty"`
	// Claude: thinking blocks, and signatures on any block.
	ThinkingBlocks int `json:"thinking_blocks,omitempty"`
	Signatures     int `json:"signatures,omitempty"`
	// Codex: reasoning items, and items carrying encrypted_content.
	ReasoningItems int    `json:"reasoning_items,omitempty"`
	EncryptedItems int    `json:"encrypted_items,omitempty"`
	Resp           string `json:"resp,omitempty"`
	RespB64        []byte `json:"resp_b64,omitempty"`
}

// upstreamInfo flags gateway-side rewrites that make the upstream request
// differ from what the agent sent, so offline analysis can tell when the
// recorded request is not exactly what the model saw.
type upstreamInfo struct {
	PassThrough          bool `json:"pass_through,omitempty"`
	SystemPrompt         bool `json:"system_prompt,omitempty"`
	SystemPromptOverride bool `json:"system_prompt_override,omitempty"`
	// ParamOverrideOps is the effective param override (channel setting merged
	// with any channel-affinity template), one entry per operation in apply
	// order. BodyOverride is set when any of them can touch the request body
	// rather than only headers.
	ParamOverrideOps []overrideOp `json:"param_override_ops,omitempty"`
	BodyOverride     bool         `json:"body_override,omitempty"`
	// HeaderOverrideKeys names the headers set by the channel header override;
	// values are dropped because they usually hold upstream credentials.
	HeaderOverrideKeys []string `json:"header_override_keys,omitempty"`
	ModelMapping       string   `json:"model_mapping,omitempty"`
}

// overrideOp mirrors relay/common.ParamOperation. Conditions are reduced to a
// flag, and set_header values are dropped since they may be credentials.
type overrideOp struct {
	Mode        string `json:"mode"`
	Path        string `json:"path,omitempty"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	Value       any    `json:"value,omitempty"`
	Conditional bool   `json:"conditional,omitempty"`
}

type toolUse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type recorder struct {
	cfg    Config
	routes map[string]route
	ch     chan *job
	done   chan struct{}
	wg     sync.WaitGroup

	// Owned by the writer goroutine.
	fw    *fileWriter
	idx   *prefixIndex
	blobs map[string]bool // blob hashes already in the active file
	seq   int64

	written    atomic.Int64
	dropped    atomic.Int64
	failed     atomic.Int64
	diskPaused atomic.Int64
	lastWarn   atomic.Int64
}

var (
	mu     sync.RWMutex
	active *recorder
)

// Init starts capture. Safe to call when disabled. An error must not prevent
// server startup; the caller logs and continues.
func Init(cfg Config) error {
	cfg.normalize()
	mu.Lock()
	defer mu.Unlock()
	if active != nil {
		active.stop()
		active = nil
	}
	if !cfg.Enabled {
		return nil
	}
	fw, err := newFileWriter(cfg)
	if err != nil {
		return err
	}
	r := &recorder{
		cfg:    cfg,
		routes: buildRoutes(cfg),
		ch:     make(chan *job, cfg.QueueSize),
		done:   make(chan struct{}),
		fw:     fw,
	}
	r.resetFileState()
	r.wg.Go(r.run)
	active = r
	common.SysLog("traj record enabled, dir: " + cfg.Dir)
	return nil
}

// Close drains the queue and flushes. Called during graceful shutdown.
func Close() {
	mu.Lock()
	r := active
	active = nil
	mu.Unlock()
	if r != nil {
		r.stop()
	}
}

// Stats returns cumulative counters: records written, dropped because the
// queue stayed full, failed to write, and skipped by the free-disk guard.
func Stats() (written, dropped, failed, diskPaused int64) {
	r := current()
	if r == nil {
		return 0, 0, 0, 0
	}
	return r.written.Load(), r.dropped.Load(), r.failed.Load(), r.diskPaused.Load()
}

func current() *recorder {
	mu.RLock()
	defer mu.RUnlock()
	return active
}

func (r *recorder) stop() {
	close(r.done)
	r.wg.Wait()
}

// submit waits briefly for queue space instead of dropping at once: a missing
// record breaks nothing (the next one becomes a keyframe), but trajectories
// are the point of this feature, so a short stall is preferred over a loss.
func (r *recorder) submit(j *job) {
	select {
	case r.ch <- j:
		return
	default:
	}
	if r.cfg.EnqueueTimeoutMs > 0 {
		timer := time.NewTimer(time.Duration(r.cfg.EnqueueTimeoutMs) * time.Millisecond)
		defer timer.Stop()
		select {
		case r.ch <- j:
			return
		case <-timer.C:
		}
	}
	n := r.dropped.Add(1)
	r.warnf("traj record queue full, dropped rid=%s (total dropped %d)", j.rec.Rid, n)
}

// warnf logs at most once per 10s so a sustained failure cannot flood the log.
func (r *recorder) warnf(format string, args ...any) {
	now := time.Now().Unix()
	last := r.lastWarn.Load()
	if now-last < 10 || !r.lastWarn.CompareAndSwap(last, now) {
		return
	}
	common.SysError(fmt.Sprintf(format, args...))
}

func (r *recorder) run() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	lastSweep := time.Now()
	for {
		select {
		case j := <-r.ch:
			r.process(j)
		case now := <-ticker.C:
			if err := r.fw.flush(); err != nil {
				r.failed.Add(1)
				r.warnf("traj record flush failed: %v", err)
			}
			if now.Sub(lastSweep) > time.Minute {
				r.idx.evict(now)
				lastSweep = now
			}
		case <-r.done:
			for {
				select {
				case j := <-r.ch:
					r.process(j)
				default:
					_ = r.fw.close()
					return
				}
			}
		}
	}
}

// resetFileState starts a new self-contained segment: the index and blob set
// are cleared so no record in the file references anything outside it.
func (r *recorder) resetFileState() {
	r.idx = newPrefixIndex(time.Duration(r.cfg.IndexTTLMin)*time.Minute, r.cfg.IndexMaxEntries)
	r.blobs = map[string]bool{}
	line, _ := common.Marshal(map[string]any{
		"v": recordVersion, "type": "segment_start", "instance": r.cfg.Instance,
		"pid": os.Getpid(), "ts": time.Now().UnixMilli(),
	})
	if err := r.fw.writeLines(line); err != nil {
		r.warnf("traj record write segment marker failed: %v", err)
	}
}

func (r *recorder) process(j *job) {
	now := time.Now()
	if r.fw.full() {
		if err := r.fw.rotate(); err != nil {
			r.warnf("traj record rotate failed: %v", err)
		} else {
			r.resetFileState()
		}
	}
	r.seq++
	rec := &j.rec
	cl := j.client
	rec.V, rec.Type, rec.Instance, rec.Seq = recordVersion, "call", r.cfg.Instance, r.seq
	rec.Client, rec.HistoryKey = cl.name, cl.historyKey
	storeResponse(rec, j.resp)
	cl.summarize(rec, j.resp)

	body := j.body
	rec.ReqSize = len(body)
	sum := sha256.Sum256(body)
	rec.ReqSha256 = hex.EncodeToString(sum[:])
	// Raw parts are spliced into the line verbatim, so a pretty-printed body
	// would break the one-record-per-line format. Agents send compact JSON;
	// anything else is compacted and flagged.
	if bytes.ContainsAny(body, "\r\n") {
		var buf bytes.Buffer
		if json.Compact(&buf, body) == nil {
			body = buf.Bytes()
			rec.ReqCompacted = true
		}
	}

	var top map[string]json.RawMessage
	var history []json.RawMessage
	if common.Unmarshal(body, &top) != nil || top[cl.historyKey] == nil || common.Unmarshal(top[cl.historyKey], &history) != nil {
		if utf8.Valid(j.body) {
			rec.ReqRaw = string(j.body)
		} else {
			rec.ReqRawB64 = j.body
		}
		r.write(rec, nil, nil, nil, nil)
		return
	}
	rec.NHistory = len(history)
	rec.KeyOrder = keyOrder(body, top, cl.historyKey, history)

	// Scope the index by session so identical openings in unrelated sessions
	// are never linked. Without a session id, fall back to the API token.
	scope := fmt.Sprintf("%s:token:%d", cl.name, rec.TokenId)
	if id := cl.requestIds(rec, top); id != "" {
		scope = cl.name + ":session:" + id
	}
	hashes := prefixHashes(scope, history)
	rec.ParentRid, rec.PrefixLen = r.idx.match(hashes)

	var blobLines [][]byte
	var newBlobs []string
	for _, kind := range cl.blobKeys {
		raw := top[kind]
		if raw == nil {
			continue
		}
		h := sha256.Sum256(raw)
		ref := hex.EncodeToString(h[:16])
		if rec.Refs == nil {
			rec.Refs = map[string]string{}
		}
		rec.Refs[kind] = ref
		if r.blobs[ref] || slices.Contains(newBlobs, ref) {
			continue
		}
		head, _ := common.Marshal(map[string]any{"v": recordVersion, "type": "blob", "kind": kind, "hash": ref})
		blobLines = append(blobLines, spliceRaw(head, "data", raw))
		newBlobs = append(newBlobs, ref)
	}

	params := map[string]json.RawMessage{}
	for k, v := range top {
		if k != cl.historyKey && !slices.Contains(cl.blobKeys, k) {
			params[k] = v
		}
	}
	if r.write(rec, blobLines, params, history[rec.PrefixLen:], newBlobs) {
		r.idx.insert(scope, rec.Rid, hashes, rec.PrefixLen, now)
	}
}

// write appends the blobs and the call line, and only then marks blobs as
// present, so a failed write never leaves a reference to a missing blob.
func (r *recorder) write(rec *callRecord, blobLines [][]byte, params map[string]json.RawMessage, appendMsgs []json.RawMessage, newBlobs []string) bool {
	meta, err := common.Marshal(rec)
	if err != nil {
		r.failed.Add(1)
		r.warnf("traj record marshal failed rid=%s: %v", rec.Rid, err)
		return false
	}
	line := meta
	if params != nil {
		var obj bytes.Buffer
		obj.WriteByte('{')
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for i, k := range keys {
			if i > 0 {
				obj.WriteByte(',')
			}
			kb, _ := common.Marshal(k)
			obj.Write(kb)
			obj.WriteByte(':')
			obj.Write(params[k])
		}
		obj.WriteByte('}')
		line = spliceRaw(line, "params", obj.Bytes())
		arr := []byte{'['}
		for i, m := range appendMsgs {
			if i > 0 {
				arr = append(arr, ',')
			}
			arr = append(arr, m...)
		}
		line = spliceRaw(line, "append", append(arr, ']'))
	}
	if err := r.fw.writeLines(append(blobLines, line)...); err != nil {
		if errors.Is(err, errDiskLow) {
			r.diskPaused.Add(1)
			return false
		}
		r.failed.Add(1)
		r.warnf("traj record write failed rid=%s: %v", rec.Rid, err)
		return false
	}
	for _, b := range newBlobs {
		r.blobs[b] = true
	}
	r.written.Add(1)
	return true
}

// keyOrder recovers the top-level key order by walking body as the exact
// concatenation `{"k":v,...}` of the decoded parts, with the history re-joined
// from its elements the way offline reassembly will do it. Any deviation
// (whitespace, escaped keys, duplicate keys) returns nil: the request is then
// still stored losslessly per part, only not reproducible byte for byte.
func keyOrder(body []byte, top map[string]json.RawMessage, historyKey string, msgs []json.RawMessage) []string {
	joined := []byte{'['}
	for i, m := range msgs {
		if i > 0 {
			joined = append(joined, ',')
		}
		joined = append(joined, m...)
	}
	joined = append(joined, ']')

	order := make([]string, 0, len(top))
	rest, ok := bytes.CutPrefix(body, []byte{'{'})
	for ok && len(order) < len(top) {
		if len(order) > 0 {
			if rest, ok = bytes.CutPrefix(rest, []byte{','}); !ok {
				break
			}
		}
		ok = false
		for k, v := range top {
			if k == historyKey {
				v = joined
			}
			kb, _ := common.Marshal(k)
			after, found := bytes.CutPrefix(rest, kb)
			if !found || len(after) == 0 || after[0] != ':' || !bytes.HasPrefix(after[1:], v) || slices.Contains(order, k) {
				continue
			}
			rest, ok = after[1+len(v):], true
			order = append(order, k)
			break
		}
	}
	if !ok || !bytes.Equal(rest, []byte{'}'}) {
		return nil
	}
	return order
}

// spliceRaw appends "key":raw to a marshaled JSON object without re-encoding
// raw. Going through json.Marshal would compact it and escape <, >, & as
// <..., so the stored bytes would no longer hash to what the client sent.
func spliceRaw(obj []byte, key string, raw []byte) []byte {
	out := make([]byte, 0, len(obj)+len(key)+len(raw)+4)
	out = append(out, obj[:len(obj)-1]...)
	out = append(out, ",\""...)
	out = append(out, key...)
	out = append(out, "\":"...)
	out = append(out, raw...)
	return append(out, '}')
}
