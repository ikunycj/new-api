package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The cache hit rate rides the same ring as RPM / TPM, so these tests reuse the
// synthetic clock and registry helpers from realtime_metrics_test.go.

func TestRealtimeCacheHitRateIsNilWithoutCacheReportingRequests(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)
	*current = realtimeTestNow

	// Traffic exists, but no upstream said anything about caching. The rate
	// must stay unknown: reporting 0% here would tell the user their cache is
	// failing when in truth nothing was measured.
	RecordRealtimeCacheUsage(41, 1000, 0, 900, false)

	byWindow := windowsByLength(GetRealtimeSnapshot(41))
	require.Equal(t, 1, byWindow[60].Requests, "throughput must still be counted")
	require.Equal(t, 1000, byWindow[60].Tokens)
	require.Zero(t, byWindow[60].CacheReadTokens, "an ineligible request must not add a numerator")
	require.Zero(t, byWindow[60].InputTokensTotal, "nor a denominator")
	require.Nil(t, byWindow[60].CacheHitRate, "no sample means no rate, not 0%")
}

func TestRealtimeCacheHitRateDistinguishesRealMissFromNoData(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)
	*current = realtimeTestNow

	// The upstream did report cache metadata and it read nothing from cache.
	// That is a genuine 0% observation and must be reported as such, which is
	// exactly what the nil case above must not be confused with.
	RecordRealtimeCacheUsage(42, 1000, 0, 900, true)

	window := windowsByLength(GetRealtimeSnapshot(42))[60]
	require.Equal(t, 900, window.InputTokensTotal)
	require.NotNil(t, window.CacheHitRate, "a reported miss is data, not absence of data")
	require.Zero(t, *window.CacheHitRate)
}

func TestRealtimeCacheHitRateIsTokenWeightedAcrossRequests(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)
	*current = realtimeTestNow

	// Two requests of very different sizes. The rate must be the ratio of
	// summed tokens (900/1000), not the mean of the two per-request rates
	// (which would be 0.9 vs 0.0 -> 0.45 and would let a tiny request distort
	// the figure as much as a huge one.)
	RecordRealtimeCacheUsage(43, 500, 900, 900, true)
	RecordRealtimeCacheUsage(43, 500, 0, 100, true)

	window := windowsByLength(GetRealtimeSnapshot(43))[60]
	require.Equal(t, 900, window.CacheReadTokens)
	require.Equal(t, 1000, window.InputTokensTotal)
	require.NotNil(t, window.CacheHitRate)
	require.InDelta(t, 0.9, *window.CacheHitRate, 1e-9)
}

func TestRealtimeCacheCountersMixEligibleAndIneligibleRequests(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)
	*current = realtimeTestNow

	// A cache-silent request alongside a cache-reporting one. Only the latter
	// may shape the ratio, while both count toward throughput.
	RecordRealtimeCacheUsage(44, 100, 0, 5_000, false)
	RecordRealtimeCacheUsage(44, 100, 250, 1_000, true)

	window := windowsByLength(GetRealtimeSnapshot(44))[60]
	require.Equal(t, 2, window.Requests)
	require.Equal(t, 200, window.Tokens)
	require.Equal(t, 1_000, window.InputTokensTotal,
		"the cache-silent request's 5000 input tokens must stay out of the denominator")
	require.NotNil(t, window.CacheHitRate)
	require.InDelta(t, 0.25, *window.CacheHitRate, 1e-9)
}

func TestRealtimeCacheCountersAppearInSeriesBuckets(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)
	*current = realtimeTestNow

	RecordRealtimeCacheUsage(45, 100, 300, 400, true)

	snapshot := GetRealtimeSnapshot(45)

	var cacheRead, inputTotal int
	for _, bucket := range snapshot.Series {
		cacheRead += bucket.CacheReadTokens
		inputTotal += bucket.InputTokensTotal
	}

	// The series must reconcile with the 1h card; if it did not, the chart and
	// the card above it would disagree about the same hour.
	hour := windowsByLength(snapshot)[3600]
	require.Equal(t, hour.CacheReadTokens, cacheRead)
	require.Equal(t, hour.InputTokensTotal, inputTotal)
	require.Equal(t, 300, cacheRead)
	require.Equal(t, 400, inputTotal)
}

func TestRealtimeCacheCountersResetWhenSlotIsReused(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	*current = realtimeTestNow
	RecordRealtimeCacheUsage(46, 10, 900, 1000, true)

	// One full lap later the same slot index is reused. Its stale cache
	// counters must be cleared with the rest of the slot, or the ratio would
	// carry six-hour-old data forward forever.
	*current = realtimeTestNow + realtimeRetentionSeconds
	RecordRealtimeCacheUsage(46, 10, 0, 50, true)

	window := windowsByLength(GetRealtimeSnapshot(46))[60]
	require.Equal(t, 0, window.CacheReadTokens)
	require.Equal(t, 50, window.InputTokensTotal)
	require.NotNil(t, window.CacheHitRate)
	require.Zero(t, *window.CacheHitRate)
}

func TestRecordRealtimeUsageLeavesCacheCountersUntouched(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)
	*current = realtimeTestNow

	// The throughput-only entry point must not fabricate a cache sample.
	RecordRealtimeUsage(47, 1000)

	window := windowsByLength(GetRealtimeSnapshot(47))[60]
	require.Equal(t, 1, window.Requests)
	require.Zero(t, window.InputTokensTotal)
	require.Nil(t, window.CacheHitRate)
}
