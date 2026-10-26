package traj_record

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"
)

// client is a legacy-named protocol adapter; name is a storage dialect alias. The recorder itself knows
// nothing about any API: it stores a request as a history array deduplicated
// by raw prefix, a few blobs stored once per file, and the remaining top-level
// keys as params. A client only says where those parts live, and how to pull
// linking clues out of its requests and responses.
type client struct {
	name      string // legacy storage dialect, not agent identity
	apiFamily string
	// paths are the gateway routes the client calls.
	paths []string
	// uaPrefixes identify the client by User-Agent, unless overridden by
	// TRAJ_RECORD_<NAME>_UA_PREFIXES.
	uaPrefixes []string
	// historyKey is the top-level request key holding the conversation, a
	// JSON array that grows by appending across calls of a session.
	historyKey string
	// blobKeys are large top-level values that rarely change within a
	// session; each distinct value is stored once per file.
	blobKeys []string
	// requestIds extracts optional client identity clues from the request.
	// These clues do not decide whether an unknown client is captured.
	requestIds func(rec *callRecord, top map[string]json.RawMessage) string
	// summarize derives searchable fields from the raw response. The response
	// itself is always stored verbatim.
	summarize func(rec *callRecord, resp []byte)
}

var clients = []*client{claudeClient, codexClient, chatClient}

func clientByName(name string) *client {
	for _, cl := range clients {
		if cl.name == name || cl.apiFamily == name {
			return cl
		}
	}
	return nil
}

// route is an enabled client together with its effective User-Agent prefixes.
type route struct {
	client     *client
	uaPrefixes []string
}

// buildRoutes indexes the enabled clients by request path.
func buildRoutes(cfg Config) map[string]route {
	routes := map[string]route{}
	for _, cl := range clients {
		prefixes, ok := cfg.Clients[cl.name]
		if !ok {
			continue
		}
		for _, p := range cl.paths {
			routes[p] = route{client: cl, uaPrefixes: prefixes}
		}
	}
	return routes
}

// ssePayloads yields the trimmed data of every "data:" line of an SSE body.
func ssePayloads(resp []byte) func(yield func([]byte) bool) {
	return func(yield func([]byte) bool) {
		for line := range bytes.Lines(resp) {
			payload, ok := bytes.CutPrefix(bytes.TrimRight(line, "\r\n"), []byte("data:"))
			if ok && !yield(bytes.TrimSpace(payload)) {
				return
			}
		}
	}
}

// storeResponse keeps the raw response on the record, as text when possible.
func storeResponse(rec *callRecord, resp []byte) {
	rec.RespSize = max(rec.RespSize, len(resp))
	if utf8.Valid(resp) {
		rec.Resp = string(resp)
	} else {
		rec.RespB64 = resp
	}
}

func matchesUA(rt route, ua string) bool {
	return slices.ContainsFunc(rt.uaPrefixes, func(p string) bool { return strings.HasPrefix(ua, p) })
}
