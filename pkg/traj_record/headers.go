package traj_record

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"slices"
	"strings"
)

// headerDrop are headers that only carry credentials. They are never written;
// their names are listed in headers_dropped.
var headerDrop = []string{
	"authorization", "proxy-authorization", "x-api-key", "api-key", "x-goog-api-key",
	"mj-api-secret", "cookie", "set-cookie",
	// Channel account identifiers whose names have no telltale substring. The
	// baidu_v2 appid is half of the channel key (relay/channel/baidu_v2,
	// codex, openai and vertex adaptors set the others).
	"chatgpt-account-id", "appid", "openai-organization", "x-goog-user-project",
}

// headerFingerprintSubstrings catch custom credential headers. Their values
// are replaced by a fingerprint instead of being dropped, so a harmless header
// that happens to match (Idempotency-Key, x-codex-turn-state) is still visible
// and comparable across calls.
var headerFingerprintSubstrings = []string{"auth", "key", "token", "secret", "cookie", "password"}

// filterHeaders flattens h, dropping credential headers and fingerprinting
// suspicious ones ("sha256:" + the first 16 hex digits of the value's hash).
// extraDrop names more headers to drop (compared case-insensitively).
func filterHeaders(h http.Header, extraDrop ...string) (kept map[string]string, dropped []string) {
	kept = map[string]string{}
	for name, values := range h {
		lower := strings.ToLower(name)
		if slices.Contains(headerDrop, lower) || slices.ContainsFunc(extraDrop, func(d string) bool { return strings.EqualFold(d, name) }) {
			dropped = append(dropped, name)
			continue
		}
		value := strings.Join(values, ", ")
		if slices.ContainsFunc(headerFingerprintSubstrings, func(sub string) bool { return strings.Contains(lower, sub) }) {
			sum := sha256.Sum256([]byte(value))
			value = "sha256:" + hex.EncodeToString(sum[:8])
		}
		kept[name] = value
	}
	slices.Sort(dropped)
	return kept, dropped
}

// headerClues copies linking clues that clients only send as headers. The
// headers stay in rec.Headers; these fields only make them easy to query.
func headerClues(rec *callRecord) {
	get := func(name string) string { return rec.Headers[http.CanonicalHeaderKey(name)] }
	rec.AgentId = get("X-Claude-Code-Agent-Id")
	rec.ParentAgentId = get("X-Claude-Code-Parent-Agent-Id")
	rec.WindowId = get("X-Codex-Window-Id")
	rec.Subagent = get("X-Openai-Subagent")
}
