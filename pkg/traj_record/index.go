package traj_record

import (
	"crypto/sha256"
	"encoding/json"
	"slices"
	"time"
)

// prefixIndex remembers, per session scope, which record first stored each
// message-history prefix. A new request looks up its longest stored prefix and
// references that record instead of storing the shared history again.
//
// Keys are cumulative hashes over the raw bytes of messages, seeded with the
// scope, so a hit guarantees the referenced record reconstructs to exactly the
// same bytes for that many messages. No normalization is needed: when Claude
// Code rewrites the tail of the history (moving cache_control, turning the
// trailing role:system message into a plain string), the longest raw prefix
// simply ends one message earlier and the rewritten message is stored again.
//
// Only the single writer goroutine touches the index, so it needs no lock.
type prefixIndex struct {
	ttl        time.Duration
	maxEntries int
	entries    map[[32]byte]string // cumulative prefix hash -> rid
	scopes     map[string]*scopeState
}

type scopeState struct {
	keys     [][32]byte
	lastUsed time.Time
}

func newPrefixIndex(ttl time.Duration, maxEntries int) *prefixIndex {
	return &prefixIndex{
		ttl:        ttl,
		maxEntries: maxEntries,
		entries:    map[[32]byte]string{},
		scopes:     map[string]*scopeState{},
	}
}

// prefixHashes returns h where h[i] identifies scope + msgs[:i+1].
func prefixHashes(scope string, msgs []json.RawMessage) [][32]byte {
	out := make([][32]byte, len(msgs))
	prev := sha256.Sum256([]byte(scope))
	var pair [64]byte
	for i, m := range msgs {
		mh := sha256.Sum256(m)
		copy(pair[:32], prev[:])
		copy(pair[32:], mh[:])
		prev = sha256.Sum256(pair[:])
		out[i] = prev
	}
	return out
}

// match returns the record holding the longest stored prefix and the number of
// messages it covers, or ("", 0) when nothing matches.
func (x *prefixIndex) match(hashes [][32]byte) (string, int) {
	for i := len(hashes) - 1; i >= 0; i-- {
		if rid, ok := x.entries[hashes[i]]; ok {
			return rid, i + 1
		}
	}
	return "", 0
}

// insert registers the prefixes this record stored itself (from index `from`
// on). Called only after the record was written, so the index never points at
// a record that is missing from the file.
func (x *prefixIndex) insert(scope, rid string, hashes [][32]byte, from int, now time.Time) {
	st := x.scopes[scope]
	if st == nil {
		st = &scopeState{}
		x.scopes[scope] = st
	}
	st.lastUsed = now
	for _, h := range hashes[from:] {
		if _, ok := x.entries[h]; ok {
			continue
		}
		x.entries[h] = rid
		st.keys = append(st.keys, h)
	}
	if len(x.entries) > x.maxEntries {
		x.evict(now)
	}
}

// evict drops idle scopes, then the least recently used ones until the index is
// back under 90% of its budget. Evicting only costs file size: the next request
// of an evicted session is written as a keyframe.
func (x *prefixIndex) evict(now time.Time) {
	for scope, st := range x.scopes {
		if now.Sub(st.lastUsed) > x.ttl {
			x.dropScope(scope, st)
		}
	}
	if len(x.entries) <= x.maxEntries {
		return
	}
	type aged struct {
		scope string
		st    *scopeState
	}
	order := make([]aged, 0, len(x.scopes))
	for scope, st := range x.scopes {
		order = append(order, aged{scope, st})
	}
	slices.SortFunc(order, func(a, b aged) int { return a.st.lastUsed.Compare(b.st.lastUsed) })
	for _, a := range order {
		if len(x.entries) <= x.maxEntries*9/10 {
			break
		}
		x.dropScope(a.scope, a.st)
	}
}

func (x *prefixIndex) dropScope(scope string, st *scopeState) {
	for _, h := range st.keys {
		delete(x.entries, h)
	}
	delete(x.scopes, scope)
}
