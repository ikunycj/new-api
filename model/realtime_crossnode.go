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
//     batches all live users (and their active key/model breakdowns) in one
//     pass. The relay path is untouched.
//  2. Graceful degradation. If Redis is unavailable the feature silently
//     falls back to single-node data. No error is surfaced to users.
//  3. Bounded memory and key space. Each node publishes one Redis hash whose
//     fields are either "u:<user>" (account totals) or
//     "d:<user>:<token>:<model>" (one breakdown). The breakdown side is
//     bounded by the same realtimeMaxDimensionRings cap that already limits
//     the in-process dimension registry, so this does not add a second
//     unbounded surface. Entries expire automatically after the retention
//     TTL so a node that dies does not leave stale keys forever.

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

// realtimeNodeSummary is the per-user (or per-dimension) record a node
// publishes to Redis. It carries pre-aggregated slot data for each of the
// three window sizes so the merge step is a simple addition with no
// slot-level work.
type realtimeNodeSummary struct {
	// Windows maps window_seconds -> aggregated result for that window.
	Windows map[int]realtimeWindowResult `json:"w"`
}

// realtimeAccountFieldPrefix and realtimeDimensionFieldPrefix distinguish the
// two kinds of hash fields a node publishes under the same node key:
// account-level (unfiltered) totals and per (token, model) breakdowns.
//
// Using one hash per node for both, instead of a second hash, keeps the key
// space bounded by the same cap that already limits dimension rings
// (realtimeMaxDimensionRings) rather than adding a second unbounded surface.
const (
	realtimeAccountFieldPrefix   = "u:"
	realtimeDimensionFieldPrefix = "d:"
)

// realtimeAccountField and realtimeDimensionField build the hash field name
// for, respectively, a user's account-level summary and one of its (token,
// model) breakdowns. The model name travels as-is: hash field names have no
// delimiter restrictions, and realtimeMaxModelNameLen already bounds it.
func realtimeAccountField(userId int) string {
	return fmt.Sprintf("%s%d", realtimeAccountFieldPrefix, userId)
}

func realtimeDimensionField(userId int, tokenId int, model string) string {
	return fmt.Sprintf("%s%d:%d:%s", realtimeDimensionFieldPrefix, userId, tokenId, model)
}

// realtimeDimensionFieldMatches reports whether a hash field published by
// publishLocalSummaries's dimension loop belongs to userId and satisfies
// filter.
//
// This is needed because a filter can be partial (token_id only, or model
// only), in which case one exact field name is not enough: the field encodes
// one specific (token, model) pair, but the filter may match many of them.
// realtimeDimensionSlots (the local-ring equivalent) has the same problem and
// solves it the same way, by scanning every candidate and testing filter.matches.
func realtimeDimensionFieldMatches(field string, userId int, filter RealtimeFilter) bool {
	rest, ok := strings.CutPrefix(field, realtimeDimensionFieldPrefix)
	if !ok {
		return false
	}
	parts := strings.SplitN(rest, ":", 3)
	if len(parts) != 3 {
		return false
	}
	fieldUserId, err := strconv.Atoi(parts[0])
	if err != nil || fieldUserId != userId {
		return false
	}
	tokenId, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	return filter.matches(realtimeDimensionKey{UserID: fieldUserId, TokenID: tokenId, Model: parts[2]})
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

// publishLocalSummaries snapshots all live user rings — both the account-level
// ones and the (token, model) breakdown rings — and writes them into the
// node's Redis hash in a single pipeline.
//
// Publishing the breakdown alongside the account totals, under the same node
// key, is what lets a filtered request merge across nodes too: before this,
// only the unfiltered path merged, and a filtered poll silently fell back to
// whichever node happened to answer.
func publishLocalSummaries() {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}

	now := realtimeNow()
	longest := realtimeWindows[len(realtimeWindows)-1]

	fields := make(map[string]interface{})

	// Account-level rings.
	realtimeRegistry.mu.RLock()
	type ringCopy struct {
		slots []realtimeSlot
	}
	accountSnaps := make(map[int]ringCopy, len(realtimeRegistry.users))
	for uid, ring := range realtimeRegistry.users {
		ring.mu.Lock()
		slots := ring.snapshot(now, longest)
		ring.mu.Unlock()
		if len(slots) > 0 {
			accountSnaps[uid] = ringCopy{slots: slots}
		}
	}
	realtimeRegistry.mu.RUnlock()

	for uid, snap := range accountSnaps {
		b, err := marshalRealtimeSummary(snap.slots, now)
		if err != nil {
			continue
		}
		fields[realtimeAccountField(uid)] = b
	}

	// (token, model) breakdown rings.
	realtimeDimensionRegistry.mu.RLock()
	type dimSnap struct {
		key   realtimeDimensionKey
		slots []realtimeSlot
	}
	dimSnaps := make([]dimSnap, 0, len(realtimeDimensionRegistry.rings))
	for key, ring := range realtimeDimensionRegistry.rings {
		ring.mu.Lock()
		slots := ring.snapshot(now, longest)
		ring.mu.Unlock()
		if len(slots) > 0 {
			dimSnaps = append(dimSnaps, dimSnap{key: key, slots: slots})
		}
	}
	realtimeDimensionRegistry.mu.RUnlock()

	for _, snap := range dimSnaps {
		b, err := marshalRealtimeSummary(snap.slots, now)
		if err != nil {
			continue
		}
		fields[realtimeDimensionField(snap.key.UserID, snap.key.TokenID, snap.key.Model)] = b
	}

	if len(fields) == 0 {
		return
	}

	nodeKey := realtimeCrossnodeHashPrefix + common.NodeName
	ctx := context.Background()

	// Use a pipeline: HSET all fields then reset the TTL.
	pipe := common.RDB.Pipeline()
	pipe.HSet(ctx, nodeKey, fields)
	pipe.Expire(ctx, nodeKey, realtimeCrossnodeKeyTTL)
	_, _ = pipe.Exec(ctx)
}

// marshalRealtimeSummary folds a slot snapshot into the three-window summary
// published to Redis.
func marshalRealtimeSummary(slots []realtimeSlot, now int64) ([]byte, error) {
	summary := realtimeNodeSummary{
		Windows: make(map[int]realtimeWindowResult, len(realtimeWindows)),
	}
	for _, window := range realtimeWindows {
		summary.Windows[int(window)] = sumSlots(slots, now, window)
	}
	return json.Marshal(summary)
}

// mergeFromRedis fetches all other nodes' summaries matching userId/filter
// and returns a merged realtimeWindowResult for the requested window. The
// local node's data is already in localResult; this adds contributions from
// other nodes.
//
// An unfiltered request (filter.IsZero()) merges the single account-level
// field. A filtered request may match more than one published field — the
// filter can be partial (only token_id, or only model) while each field
// encodes one specific (token, model) pair — so it walks every field on each
// remote node's hash and sums the ones realtimeDimensionFieldMatches accepts.
// That is more Redis round trips than the single HGET the unfiltered path
// uses, but it only runs on a filtered poll, not the common case.
//
// If Redis is unavailable or returns no cross-node data the function returns
// localResult unchanged so the caller degrades gracefully to local-only data.
func mergeFromRedis(userId int, filter RealtimeFilter, window int64, localResult realtimeWindowResult) realtimeWindowResult {
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

	fold := func(val string) {
		var summary realtimeNodeSummary
		if err := json.Unmarshal([]byte(val), &summary); err != nil {
			return
		}
		remote, ok := summary.Windows[int(window)]
		if !ok {
			return
		}
		merged.Requests += remote.Requests
		merged.Tokens += remote.Tokens
		merged.CacheReadTokens += remote.CacheReadTokens
		merged.InputTokensTotal += remote.InputTokensTotal
		merged.BusySeconds += remote.BusySeconds
	}

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
			if filter.IsZero() {
				val, err := common.RDB.HGet(ctx, key, realtimeAccountField(userId)).Result()
				if err != nil {
					// Field not found for this node — normal, skip it.
					continue
				}
				fold(val)
				continue
			}
			// Filtered: the field name is not known in advance, so walk the
			// remote node's whole hash and keep the fields that match.
			all, err := common.RDB.HGetAll(ctx, key).Result()
			if err != nil {
				continue
			}
			for field, val := range all {
				if !realtimeDimensionFieldMatches(field, userId, filter) {
					continue
				}
				fold(val)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return merged
}
