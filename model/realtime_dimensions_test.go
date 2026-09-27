package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests cover the (user, key, model) breakdown rings. They reuse the
// synthetic clock and registry helpers from realtime_metrics_test.go.

func setupDimensionTest(t *testing.T) *int64 {
	t.Helper()
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)
	resetRealtimeDimensionsForTest()
	t.Cleanup(resetRealtimeDimensionsForTest)
	*current = realtimeTestNow
	return current
}

func TestRealtimeFilterSeparatesKeysAndModels(t *testing.T) {
	setupDimensionTest(t)

	RecordRealtimeRequest(1, 10, "gpt-4o", 100, 40, 80, true)
	RecordRealtimeRequest(1, 10, "claude-sonnet-4", 200, 0, 50, true)
	RecordRealtimeRequest(1, 20, "gpt-4o", 400, 10, 20, true)

	// The account total must be the sum of every combination, because the
	// user-level ring is written unconditionally.
	all := windowsByLength(GetRealtimeSnapshot(1))[60]
	require.Equal(t, 3, all.Requests)
	require.Equal(t, 700, all.Tokens)

	// One key, all of its models.
	byKey := windowsByLength(GetRealtimeSnapshotFiltered(1, RealtimeFilter{TokenID: 10}))[60]
	require.Equal(t, 2, byKey.Requests)
	require.Equal(t, 300, byKey.Tokens)

	// One model, across every key.
	byModel := windowsByLength(GetRealtimeSnapshotFiltered(1, RealtimeFilter{Model: "gpt-4o"}))[60]
	require.Equal(t, 2, byModel.Requests)
	require.Equal(t, 500, byModel.Tokens)

	// One key and one model.
	both := windowsByLength(GetRealtimeSnapshotFiltered(1, RealtimeFilter{TokenID: 10, Model: "gpt-4o"}))[60]
	require.Equal(t, 1, both.Requests)
	require.Equal(t, 100, both.Tokens)
	require.NotNil(t, both.CacheHitRate)
	require.InDelta(t, 0.5, *both.CacheHitRate, 1e-9)
}

func TestRealtimeFilterCannotReachAnotherUsersTraffic(t *testing.T) {
	setupDimensionTest(t)

	// User 2 owns key 99. User 1 asking for key 99 must get nothing: the filter
	// narrows within the caller's own rings, so a forged key id cannot widen the
	// result to another account. This is the property the /self endpoint relies
	// on instead of an ownership lookup.
	RecordRealtimeRequest(2, 99, "gpt-4o", 500_000, 0, 0, false)
	RecordRealtimeRequest(1, 10, "gpt-4o", 100, 0, 0, false)

	leaked := windowsByLength(GetRealtimeSnapshotFiltered(1, RealtimeFilter{TokenID: 99}))[60]
	require.Zero(t, leaked.Requests, "another user's key must not match")
	require.Zero(t, leaked.Tokens, "another user's tokens must never be reported")

	victim := windowsByLength(GetRealtimeSnapshotFiltered(2, RealtimeFilter{TokenID: 99}))[60]
	require.Equal(t, 1, victim.Requests, "the owner still sees their own traffic")
}

func TestRealtimeFilterZeroValueMatchesAccountTotal(t *testing.T) {
	setupDimensionTest(t)

	RecordRealtimeRequest(1, 10, "gpt-4o", 100, 30, 60, true)
	RecordRealtimeRequest(1, 20, "o3", 300, 10, 40, true)

	// An empty filter must read the account ring and agree with the breakdown
	// summed over every combination; if it did not, the headline card and the
	// filtered view would contradict each other.
	unfiltered := windowsByLength(GetRealtimeSnapshotFiltered(1, RealtimeFilter{}))[60]
	total := windowsByLength(GetRealtimeSnapshot(1))[60]
	require.Equal(t, total, unfiltered)
	require.Equal(t, 2, unfiltered.Requests)
	require.Equal(t, 400, unfiltered.Tokens)
	require.Equal(t, 40, unfiltered.CacheReadTokens)
	require.Equal(t, 100, unfiltered.InputTokensTotal)
}

func TestRealtimeDimensionRingCapPreservesAccountTotals(t *testing.T) {
	setupDimensionTest(t)

	// Fill the breakdown to its cap, then send one more distinct combination.
	for i := 0; i < realtimeMaxDimensionRings; i++ {
		RecordRealtimeRequest(1, i+1, "gpt-4o", 1, 0, 0, false)
	}
	RecordRealtimeRequest(1, realtimeMaxDimensionRings+1, "gpt-4o", 1000, 0, 0, false)

	realtimeDimensionRegistry.mu.RLock()
	ringCount := len(realtimeDimensionRegistry.rings)
	capped := realtimeDimensionRegistry.capped
	realtimeDimensionRegistry.mu.RUnlock()
	require.Equal(t, realtimeMaxDimensionRings, ringCount, "the cap must hold")
	require.True(t, capped, "hitting the cap must be recorded so the reader can be told")

	// The refused combination still counted toward the account, which is the
	// whole point of writing the user-level ring first: the breakdown may be
	// incomplete but the totals never are.
	total := windowsByLength(GetRealtimeSnapshot(1))[60]
	require.Equal(t, realtimeMaxDimensionRings+1, total.Requests)
	require.Equal(t, realtimeMaxDimensionRings+1000, total.Tokens)

	// The refused combination has no breakdown row.
	refused := windowsByLength(GetRealtimeSnapshotFiltered(1, RealtimeFilter{TokenID: realtimeMaxDimensionRings + 1}))[60]
	require.Zero(t, refused.Requests)
}

func TestRealtimeDimensionSkipsRequestsWithoutTokenId(t *testing.T) {
	setupDimensionTest(t)

	// Some internal paths bill without a key. Such a request belongs in the
	// account total but cannot be attributed, so it must not create a ring under
	// a bogus key id.
	RecordRealtimeRequest(1, 0, "gpt-4o", 100, 0, 0, false)

	require.Equal(t, 1, windowsByLength(GetRealtimeSnapshot(1))[60].Requests)
	realtimeDimensionRegistry.mu.RLock()
	count := len(realtimeDimensionRegistry.rings)
	realtimeDimensionRegistry.mu.RUnlock()
	require.Zero(t, count, "a request with no key must not allocate a breakdown ring")
}

func TestRealtimeDimensionTruncatesLongModelNames(t *testing.T) {
	setupDimensionTest(t)

	// Model names come from the request body. Without truncation a caller could
	// mint an unbounded number of rings by varying a long suffix.
	long := ""
	for i := 0; i < 500; i++ {
		long += "x"
	}
	RecordRealtimeRequest(1, 10, long+"aaa", 100, 0, 0, false)
	RecordRealtimeRequest(1, 10, long+"bbb", 100, 0, 0, false)

	realtimeDimensionRegistry.mu.RLock()
	count := len(realtimeDimensionRegistry.rings)
	realtimeDimensionRegistry.mu.RUnlock()
	require.Equal(t, 1, count, "names differing past the bound must share one ring")

	truncated := long[:realtimeMaxModelNameLen]
	got := windowsByLength(GetRealtimeSnapshotFiltered(1, RealtimeFilter{Model: truncated}))[60]
	require.Equal(t, 2, got.Requests)
}

func TestRealtimeDimensionSweepReclaimsIdleRings(t *testing.T) {
	current := setupDimensionTest(t)

	RecordRealtimeRequest(1, 10, "gpt-4o", 100, 0, 0, false)
	realtimeDimensionRegistry.mu.RLock()
	require.Equal(t, 1, len(realtimeDimensionRegistry.rings))
	realtimeDimensionRegistry.mu.RUnlock()

	// Past the idle TTL the ring can no longer contribute to any window a caller
	// can ask for, so holding it would only pin memory.
	*current = realtimeTestNow + realtimeRetentionSeconds + 1
	sweepRealtimeDimensionRegistry()

	realtimeDimensionRegistry.mu.RLock()
	count := len(realtimeDimensionRegistry.rings)
	realtimeDimensionRegistry.mu.RUnlock()
	require.Zero(t, count, "an idle breakdown ring must be freed")
}

func TestRealtimeDimensionsListsOnlyActiveCombinations(t *testing.T) {
	setupDimensionTest(t)

	RecordRealtimeRequest(1, 10, "gpt-4o", 100, 0, 0, false)
	RecordRealtimeRequest(1, 10, "gpt-4o", 100, 0, 0, false)
	RecordRealtimeRequest(1, 20, "o3", 100, 0, 0, false)
	RecordRealtimeRequest(2, 30, "gpt-4o", 100, 0, 0, false)

	dims := GetRealtimeDimensions(1, 60)

	// Only this user's combinations, busiest first.
	require.Len(t, dims.Tokens, 2)
	require.Equal(t, 10, dims.Tokens[0].TokenID)
	require.Equal(t, 2, dims.Tokens[0].Requests)
	require.Equal(t, 20, dims.Tokens[1].TokenID)

	require.Len(t, dims.Models, 2)
	require.Equal(t, "gpt-4o", dims.Models[0].Model)
	require.Equal(t, 2, dims.Models[0].Requests)

	for _, option := range dims.Tokens {
		require.NotEqual(t, 30, option.TokenID, "another user's key must not be offered")
	}
}

func TestRealtimeDimensionsRecordsDisabledIsNoop(t *testing.T) {
	setupDimensionTest(t)
	realtimeEnabled = false

	RecordRealtimeRequest(1, 10, "gpt-4o", 100, 0, 0, false)

	realtimeDimensionRegistry.mu.RLock()
	count := len(realtimeDimensionRegistry.rings)
	realtimeDimensionRegistry.mu.RUnlock()
	require.Zero(t, count, "the feature flag must gate the breakdown too")
}

// BenchmarkRecordRealtimeRequest measures the relay hot path with the breakdown
// included, so its cost can be read against BenchmarkRecordRealtimeCacheUsage
// rather than assumed.
func BenchmarkRecordRealtimeRequest(b *testing.B) {
	originalEnabled := realtimeEnabled
	realtimeEnabled = true
	b.Cleanup(func() { realtimeEnabled = originalEnabled })
	b.Cleanup(resetRealtimeDimensionsForTest)

	RecordRealtimeRequest(1, 7, "gpt-4o", 1000, 400, 800, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RecordRealtimeRequest(1, 7, "gpt-4o", 1000, 400, 800, true)
	}
}

// BenchmarkRecordRealtimeRequestSpread uses a realistic spread of keys and
// models so the map is not a single hot entry.
func BenchmarkRecordRealtimeRequestSpread(b *testing.B) {
	originalEnabled := realtimeEnabled
	realtimeEnabled = true
	b.Cleanup(func() { realtimeEnabled = originalEnabled })
	b.Cleanup(resetRealtimeDimensionsForTest)

	models := make([]string, 10)
	for i := range models {
		models[i] = fmt.Sprintf("model-%d", i)
	}
	for k := 1; k <= 50; k++ {
		for _, m := range models {
			RecordRealtimeRequest(1, k, m, 10, 0, 0, false)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RecordRealtimeRequest(1, i%50+1, models[i%10], 1000, 400, 800, true)
	}
}

// BenchmarkRecordRealtimeCacheUsageSerial is the pre-breakdown path measured
// serially, so it is directly comparable to BenchmarkRecordRealtimeRequest.
// The existing parallel benchmarks include lock contention and cannot be
// compared against a serial one.
func BenchmarkRecordRealtimeCacheUsageSerial(b *testing.B) {
	originalEnabled := realtimeEnabled
	realtimeEnabled = true
	b.Cleanup(func() { realtimeEnabled = originalEnabled })

	RecordRealtimeCacheUsage(1, 1000, 400, 800, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RecordRealtimeCacheUsage(1, 1000, 400, 800, true)
	}
}
