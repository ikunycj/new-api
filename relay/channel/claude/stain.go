// Package-level response staining: append the gateway request id to Claude
// message ids returned downstream, so a downstream reseller forwarding our
// responses verbatim carries a fingerprint attributable to a specific request.
package claude

import (
	"bytes"

	"github.com/QuantumNous/new-api/common"
)

// stainingEnabled is read once at startup. Default off; enable with
// RELAY_STAINING_ENABLED=true. When off, StainClaudeMessageID is a no-op and
// bodies pass through byte-identical to the disabled baseline.
var stainingEnabled = common.GetEnvOrDefaultBool("RELAY_STAINING_ENABLED", false)

// stainRIDTailLen caps how much of the request id is appended. The id is the
// gateway's own `20260930HHMMSS...` string; 12 chars keep the fingerprint
// unique per request while the whole rid still lives in the X-Oneapi-Request-Id
// response header and the logs table for full attribution.
const stainRIDTailLen = 12

// stainClaudeMessageID rewrites the `"id":"msg_..."` value inside an upstream
// Claude response body by appending the gateway request id tail. The rewrite
// keeps the value alphanumeric and prefixed with msg_, so downstream SDKs and
// Claude Code keep parsing it unchanged.
//
// The scan is a bounded byte search (single Index/IndexByte pair), so the hot
// path pays microseconds per response regardless of body size; events without
// a msg_ id (everything except message_start) return unchanged.
func StainClaudeMessageID(data []byte, rid string) []byte {
	if !stainingEnabled || len(data) == 0 || len(rid) == 0 {
		return data
	}
	key := []byte(`"id":"`)
	start := bytes.Index(data, key)
	if start < 0 {
		return data
	}
	start += len(key)
	end := bytes.IndexByte(data[start:], '"')
	if end <= 0 {
		return data
	}
	end += start
	if !bytes.HasPrefix(data[start:end], []byte("msg_")) {
		return data
	}
	tail := rid
	if len(tail) > stainRIDTailLen {
		tail = tail[len(tail)-stainRIDTailLen:]
	}
	out := make([]byte, 0, len(data)+len(tail))
	out = append(out, data[:end]...)
	out = append(out, tail...)
	out = append(out, data[end:]...)
	return out
}
