package relaytrace

import (
	"bytes"
	"encoding/json"
)

// Record is one JSONL line: the user-facing request and response for a single
// relay call.
//
// Metadata is deliberately minimal. Everything else about the request
// (user, token, channel, model mapping, quota, upstream request id) already
// lives in the `logs` table and is joined via Rid. Duplicating it here would
// double storage and risk the two sources disagreeing.
type Record struct {
	// Rid is the request id, also emitted as the X-Oneapi-Request-Id response
	// header. It joins this trace to its `logs` row and to nginx access logs.
	Rid string `json:"rid"`

	// Ts is the request start time in Unix milliseconds.
	Ts int64 `json:"ts"`

	// Status is the HTTP status written to the client.
	Status int `json:"status"`

	// DurationMs is the wall-clock time spent handling the request.
	DurationMs int64 `json:"duration_ms"`

	// Method and Path identify the endpoint. Model is kept despite living in
	// `logs` too, because it makes the file self-sufficient for the most common
	// filter (`jq 'select(.model=="...")'`) without a DB round trip.
	Method string `json:"method"`
	Path   string `json:"path"`
	Model  string `json:"model,omitempty"`

	// Stream reports whether the response was streamed (SSE).
	Stream bool `json:"stream,omitempty"`

	// ReqHeaders holds allow-listed request headers only. Credential headers
	// (Authorization, X-Api-Key, ...) are never recorded.
	ReqHeaders map[string]string `json:"req_headers,omitempty"`

	// Req holds the request body when it is valid JSON, inlined verbatim so no
	// escaping is applied and the file stays readable.
	Req json.RawMessage `json:"req,omitempty"`

	// ReqRaw holds the request body when it is not valid JSON (multipart audio
	// uploads, malformed payloads). Only one of Req/ReqRaw is ever set.
	ReqRaw string `json:"req_raw,omitempty"`

	// Resp holds the response body: a JSON document for unary calls, or the raw
	// SSE event stream for streaming calls. Stored as a string because SSE is
	// not a single JSON value.
	Resp string `json:"resp,omitempty"`

	// ReqSize and RespSize are the true byte sizes observed, before truncation.
	// They stay accurate even when the stored body is clipped.
	ReqSize  int `json:"req_size"`
	RespSize int `json:"resp_size"`

	// ReqTruncated and RespTruncated flag that the stored body was clipped at
	// MaxBodyKB, so absence of content is not mistaken for an empty body.
	ReqTruncated  bool `json:"req_truncated,omitempty"`
	RespTruncated bool `json:"resp_truncated,omitempty"`

	// HasSignature reports whether the response carried a non-empty Anthropic
	// thinking signature. Nil for non-Claude paths or when detection is off.
	//
	// This is a derived convenience field: it makes finding channels that strip
	// signatures a one-liner (`jq 'select(.has_signature == false)'`) instead of
	// re-parsing every SSE stream. Computed from the same bytes already in
	// memory, so it costs one scan and no extra storage.
	HasSignature *bool `json:"has_signature,omitempty"`

	// Dropped, when set, means the writer shed load before this point; used by
	// the stats line rather than per-request records.
	Err string `json:"err,omitempty"`
}

// sensitiveHeaders are never written to disk: they carry credentials that would
// let a reader of the trace file impersonate the caller.
var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"x-api-key":           true,
	"api-key":             true,
	"x-goog-api-key":      true,
	"cookie":              true,
	"set-cookie":          true,
	"proxy-authorization": true,
}

// allowedHeaders is the allow-list of request headers worth keeping for audit.
// An allow-list (rather than a deny-list) means a newly introduced credential
// header cannot leak by default.
var allowedHeaders = []string{
	"Content-Type",
	"Content-Encoding",
	"Accept",
	"User-Agent",
	"Anthropic-Version",
	"Anthropic-Beta",
	"X-Stainless-Lang",
}

// FilterHeaders returns only allow-listed headers, and never a sensitive one.
func FilterHeaders(get func(string) string) map[string]string {
	out := make(map[string]string, len(allowedHeaders))
	for _, name := range allowedHeaders {
		if sensitiveHeaders[lowerASCII(name)] {
			continue
		}
		if v := get(name); v != "" {
			out[name] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// lowerASCII lowercases an ASCII header name without allocating via strings.ToLower
// for the common already-lower case.
func lowerASCII(s string) string {
	hasUpper := false
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return s
	}
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// signatureKey matches the Anthropic thinking signature field in both unary
// responses ({"type":"thinking","signature":"..."}) and streaming deltas
// ({"type":"signature_delta","signature":"..."}).
var signatureKey = []byte(`"signature"`)

// DetectSignature reports whether body contains a non-empty `signature` value.
//
// Returns (false, true) when the key is present but every occurrence is empty —
// exactly the "channel stripped the signature" case worth alerting on — and
// (false, false) when the key never appears, which is normal for models and
// endpoints that do not emit thinking signatures at all. Distinguishing the two
// matters: an absent key is not evidence of a dirty channel.
func DetectSignature(body []byte) (hasNonEmpty bool, keyPresent bool) {
	rest := body
	for {
		idx := bytes.Index(rest, signatureKey)
		if idx < 0 {
			return false, keyPresent
		}
		keyPresent = true
		rest = rest[idx+len(signatureKey):]

		// Skip whitespace and the ':' separator.
		i := 0
		for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t' || rest[i] == ':') {
			i++
		}
		// A non-empty string value must open with a quote followed by content.
		if i+1 < len(rest) && rest[i] == '"' && rest[i+1] != '"' {
			return true, true
		}
		rest = rest[i:]
	}
}
