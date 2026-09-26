package model

// realtime_crossnode.go — cross-node ring summary merging via Redis.
//
// In a multi-instance deployment the in-process rings only see traffic served
// by the local node. This file adds a background writer that periodically
// publishes each live user's ring summary into Redis, and a reader that
// merges all nodes' summaries so the dashboard shows combined throughput.
//
// Design constraints:
//
//  1. Zero hot-path impact. The write is done by a background timer that
//     batches all live users in one pass. The relay path is untouched.
//  2. Graceful degradation. If Redis is unavailable the feature silently
//     falls back to single-node data. No error is surfaced to users.
//  3. Bounded memory and key space. Each node publishes one Redis hash whose
//     fields are user IDs. Entries expire automatically after the retention
//     TTL so a node that dies does not leave stale keys forever.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	// realtimeCrossnodeInterval is how often the background writer pushes local
	// ring summaries to Redis. Short enough that pollers see fresh cross-node
	// data without hammering Redis on every request.
	realtimeCrossnodeInterval = 5 * time.Second

	// realtimeCrossnodeKeyTTL is the Redis key TTL. It is longer than the ring
	// retention so a node that stops publishing (crash, restart) expires on its
	// own within one retention window rather than never.
	realtimeCrossnodeKeyTTL = (realtimeRetentionSeconds + 60) * time.Second

	// realtimeCrossnodeHashPrefix is the Redis key for a node's summary hash.
	// Each node gets one key; the hash fields are stringified user IDs.
	realtimeCrossnodeHashPrefix = "realtime_node:"
)

// realtimeNodeSummary is the per-user record a node publishes to Redis.
// It carries pre-aggregated slot data for each of the three window sizes so
// the merge step is a simple addition with no slot-level work.
type realtimeNodeSummary struct {
	// Windows maps window_seconds -> aggregated result for that window.
	Windows map[int]realtimeWindowResult `json:"w"`
}

// crossnodeEnabled is true when Redis is available and the cross-node loop has
// been started. It is checked before every Redis call so no-Redis deployments
// incur no overhead.
var crossnodeEnabled bool

// initRealtimeCrossnode starts the background publisher if Redis is available.
// Called from InitRealtimeMetrics so the goroutine is never spawned from the
// relay hot path.
func initRealtimeCrossnode() {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	crossnodeEnabled = true
	go crossnodeSyncLoop()
}

// crossnodeSyncLoop pushes local ring summaries to Redis every interval.
func crossnodeSyncLoop() {
	ticker := time.NewTicker(realtimeCrossnodeInterval)
	defer ticker.Stop()
	for range ticker.C {
		publishLocalSummaries()
	}
}

// publishLocalSummaries snapshots all live user rings and writes them into the
// node's Redis hash in a single pipeline.
func publishLocalSummaries() {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}

	now := realtimeNow()

	// Snapshot the registry under a read lock, then release before doing IO.
	realtimeRegistry.mu.RLock()
	type ringCopy struct {
		slots []realtimeSlot
	}
	snaps := make(map[int]ringCopy, len(realtimeRegistry.users))
	for uid, ring := range realtimeRegistry.users {
		ring.mu.Lock()
		// snapshot uses the longest window so all shorter windows are covered
		longest := realtimeWindows[len(realtimeWindows)-1]
		slots := ring.snapshot(now, longest)
		ring.mu.Unlock()
		if len(slots) > 0 {
			snaps[uid] = ringCopy{slots: slots}
		}
	}
	realtimeRegistry.mu.RUnlock()

	if len(snaps) == 0 {
		return
	}

	nodeKey := realtimeCrossnodeHashPrefix + common.NodeName
	ctx := context.Background()

	// Build the hash fields: one JSON value per user.
	fields := make(map[string]interface{}, len(snaps))
	for uid, snap := range snaps {
		summary := realtimeNodeSummary{
			Windows: make(map[int]realtimeWindowResult, len(realtimeWindows)),
		}
		for _, window := range realtimeWindows {
			summary.Windows[int(window)] = sumSlots(snap.slots, now, window)
		}
		b, err := json.Marshal(summary)
		if err != nil {
			continue
		}
		fields[fmt.Sprintf("%d", uid)] = b
	}

	// Use a pipeline: HSET all fields then reset the TTL.
	pipe := common.RDB.Pipeline()
	pipe.HSet(ctx, nodeKey, fields)
	pipe.Expire(ctx, nodeKey, realtimeCrossnodeKeyTTL)
	_, _ = pipe.Exec(ctx)
}

// mergeFromRedis fetches all nodes' summaries for a user and returns a merged
// realtimeWindowResult for the requested window. The local node's data is
// already in localResult; this adds contributions from other nodes.
//
// If Redis is unavailable or returns no cross-node data the function returns
// localResult unchanged so the caller degrades gracefully to local-only data.
func mergeFromRedis(userId int, window int64, localResult realtimeWindowResult) realtimeWindowResult {
	if !crossnodeEnabled || !common.RedisEnabled || common.RDB == nil {
		return localResult
	}

	ctx := context.Background()
	pattern := realtimeCrossnodeHashPrefix + "*"

	// Scan all node keys. In production there are typically 2-4 nodes so this
	// is a handful of round trips, all on the read path polled every 5s.
	var cursor uint64
	merged := localResult
	localNodeKey := realtimeCrossnodeHashPrefix + common.NodeName

	for {
		keys, nextCursor, err := common.RDB.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		for _, key := range keys {
			if key == localNodeKey {
				// Skip local node: already counted in localResult.
				continue
			}
			field := fmt.Sprintf("%d", userId)
			val, err := common.RDB.HGet(ctx, key, field).Result()
			if err != nil {
				// Key not found for this user on that node — normal, skip it.
				continue
			}
			var summary realtimeNodeSummary
			if err := json.Unmarshal([]byte(val), &summary); err != nil {
				continue
			}
			remote, ok := summary.Windows[int(window)]
			if !ok {
				continue
			}
			merged.Requests += remote.Requests
			merged.Tokens += remote.Tokens
			merged.CacheReadTokens += remote.CacheReadTokens
			merged.InputTokensTotal += remote.InputTokensTotal
			merged.BusySeconds += remote.BusySeconds
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return merged
}
