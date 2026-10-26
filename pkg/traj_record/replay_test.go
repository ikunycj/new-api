package traj_record

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Opt-in replay uses real request histories with one long-lived key and no
// session scope. It never contacts a model or prints request content.
func TestReplaySingleKeyIndex(t *testing.T) {
	path := os.Getenv("TRAJ_INDEX_SAMPLES")
	if path == "" {
		t.Skip("TRAJ_INDEX_SAMPLES not set")
	}
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	idx := newPrefixIndex(6*time.Hour, defaultIndexMaxEntries)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	var total, saved, calls, peak int
	started := time.Now()
	scan := bufio.NewScanner(f)
	scan.Buffer(nil, 128<<20)
	for scan.Scan() {
		var row struct {
			Req json.RawMessage `json:"req"`
		}
		require.NoError(t, json.Unmarshal(scan.Bytes(), &row))
		var req map[string]json.RawMessage
		if json.Unmarshal(row.Req, &req) != nil {
			continue
		}
		var msgs []json.RawMessage
		family := "messages"
		raw := req["messages"]
		if raw == nil {
			raw = req["input"]
			family = "responses"
		}
		if json.Unmarshal(raw, &msgs) != nil {
			continue
		}
		scope := family + ":user:1:token:1"
		hashes := prefixHashes(scope, msgs)
		_, n := idx.match(hashes)
		for i, m := range msgs {
			total += len(m)
			if i < n {
				saved += len(m)
			}
		}
		idx.insert(scope, fmt.Sprint(calls), hashes, n, time.Now())
		peak = max(peak, len(idx.entries))
		calls++
	}
	require.NoError(t, scan.Err())
	elapsed := time.Since(started)
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(idx)
	t.Logf("calls=%d history_bytes=%d saved_bytes=%d peak_entries=%d retained_heap_delta=%d elapsed=%s", calls, total, saved, peak, int64(after.HeapAlloc)-int64(before.HeapAlloc), elapsed)
	require.Positive(t, calls)
}
