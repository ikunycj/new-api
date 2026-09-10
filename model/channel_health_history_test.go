package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearChannelHealthHistory(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Where("1 = 1").Delete(&ChannelHealthHistory{}).Error)
}

// The upsert must maintain a running mean, not a sum. A health score is a gauge:
// two observations of 1.0 mean "still healthy", not "twice as healthy". This is
// the core behaviour that separates this table from perf_metrics.
func TestUpsertChannelHealthHistoryKeepsRunningMean(t *testing.T) {
	clearChannelHealthHistory(t)

	base := ChannelHealthHistory{
		ChannelId: 71, Route: "__probe__", Family: "claude", BucketTs: 1000,
		ObservationCount: 1,
	}

	first := base
	first.Score = 1.0
	first.Availability = 1.0
	first.LatencyMs = 100
	require.NoError(t, UpsertChannelHealthHistory(&first))

	second := base
	second.Score = 0.0
	second.Availability = 0.0
	second.LatencyMs = 300
	require.NoError(t, UpsertChannelHealthHistory(&second))

	rows, err := GetChannelHealthHistory(0, 2000)
	require.NoError(t, err)
	require.Len(t, rows, 1, "same key must fold into one bucket row")

	row := rows[0]
	assert.InDelta(t, 0.5, row.Score, 0.0001, "score must be the mean of 1.0 and 0.0")
	assert.InDelta(t, 0.5, row.Availability, 0.0001)
	assert.InDelta(t, 200, row.LatencyMs, 0.0001, "latency must average, not sum")
	assert.Equal(t, int64(2), row.ObservationCount)
}

// A third observation must be weighted correctly against the accumulated mean
// rather than simply averaged with it: (1.0 + 1.0 + 0.0)/3 == 0.6667, not
// ((1.0+1.0)/2 + 0.0)/2 == 0.5. This is what makes the incremental formula
// worth its complexity.
func TestUpsertChannelHealthHistoryWeightsByObservationCount(t *testing.T) {
	clearChannelHealthHistory(t)

	base := ChannelHealthHistory{
		ChannelId: 72, Route: "r", Family: "claude", BucketTs: 2000,
		ObservationCount: 1,
	}
	for _, score := range []float64{1.0, 1.0, 0.0} {
		entry := base
		entry.Score = score
		require.NoError(t, UpsertChannelHealthHistory(&entry))
	}

	rows, err := GetChannelHealthHistory(0, 3000)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.InDelta(t, 2.0/3.0, rows[0].Score, 0.0001,
		"mean must be weighted by observation count, got %v", rows[0].Score)
	assert.Equal(t, int64(3), rows[0].ObservationCount)
}

// confident_count accumulates while the gauges average, so the UI can tell a
// trusted low score apart from one that is merely under-sampled.
func TestUpsertChannelHealthHistoryAccumulatesConfidentCount(t *testing.T) {
	clearChannelHealthHistory(t)

	base := ChannelHealthHistory{
		ChannelId: 73, Route: "r", Family: "claude", BucketTs: 3000,
		ObservationCount: 1, Score: 0.5,
	}
	confident := base
	confident.ConfidentCount = 1
	require.NoError(t, UpsertChannelHealthHistory(&confident))
	require.NoError(t, UpsertChannelHealthHistory(&confident))
	notConfident := base
	notConfident.ConfidentCount = 0
	require.NoError(t, UpsertChannelHealthHistory(&notConfident))

	rows, err := GetChannelHealthHistory(0, 4000)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(2), rows[0].ConfidentCount)
	assert.Equal(t, int64(3), rows[0].ObservationCount)
}

// Distinct channel / route / family / bucket combinations must never collide,
// otherwise a probe series would overwrite the real-traffic series for the same
// channel.
func TestUpsertChannelHealthHistorySeparatesKeys(t *testing.T) {
	clearChannelHealthHistory(t)

	variants := []ChannelHealthHistory{
		{ChannelId: 80, Route: "__probe__", Family: "claude", BucketTs: 5000, Score: 0.1, ObservationCount: 1},
		{ChannelId: 80, Route: "real", Family: "claude", BucketTs: 5000, Score: 0.2, ObservationCount: 1},
		{ChannelId: 80, Route: "__probe__", Family: "gpt", BucketTs: 5000, Score: 0.3, ObservationCount: 1},
		{ChannelId: 80, Route: "__probe__", Family: "claude", BucketTs: 5060, Score: 0.4, ObservationCount: 1},
		{ChannelId: 81, Route: "__probe__", Family: "claude", BucketTs: 5000, Score: 0.5, ObservationCount: 1},
	}
	for i := range variants {
		require.NoError(t, UpsertChannelHealthHistory(&variants[i]))
	}

	rows, err := GetChannelHealthHistory(0, 6000)
	require.NoError(t, err)
	assert.Len(t, rows, len(variants), "each distinct key must get its own row")
}

func TestUpsertChannelHealthHistoryRejectsInvalidInput(t *testing.T) {
	clearChannelHealthHistory(t)

	// No observations: nothing to record, and dividing by zero must be avoided.
	assert.NoError(t, UpsertChannelHealthHistory(&ChannelHealthHistory{
		ChannelId: 90, Route: "r", Family: "claude", BucketTs: 7000, ObservationCount: 0,
	}))
	// Missing channel id would produce an unattributable row.
	assert.Error(t, UpsertChannelHealthHistory(&ChannelHealthHistory{
		ChannelId: 0, Route: "r", Family: "claude", BucketTs: 7000, ObservationCount: 1,
	}))
	assert.NoError(t, UpsertChannelHealthHistory(nil))

	rows, err := GetChannelHealthHistory(0, 8000)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestDeleteChannelHealthHistoryBefore(t *testing.T) {
	clearChannelHealthHistory(t)

	for _, ts := range []int64{1000, 2000, 3000} {
		entry := ChannelHealthHistory{
			ChannelId: 95, Route: "r", Family: "claude", BucketTs: ts,
			Score: 1, ObservationCount: 1,
		}
		require.NoError(t, UpsertChannelHealthHistory(&entry))
	}

	require.NoError(t, DeleteChannelHealthHistoryBefore(2500))

	rows, err := GetChannelHealthHistory(0, 9000)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(3000), rows[0].BucketTs)

	// A non-positive cutoff must be a no-op rather than truncating the table.
	require.NoError(t, DeleteChannelHealthHistoryBefore(0))
	rows, err = GetChannelHealthHistory(0, 9000)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestChannelHealthHistoryBucketAlignment(t *testing.T) {
	assert.Equal(t, int64(1200), ChannelHealthHistoryBucket(1234, 60))
	assert.Equal(t, int64(1200), ChannelHealthHistoryBucket(1200, 60))
	assert.Equal(t, int64(1230), ChannelHealthHistoryBucket(1234, 30))
	// A zero or negative bucket width must fall back to the 60s default instead
	// of dividing by zero.
	assert.Equal(t, int64(1200), ChannelHealthHistoryBucket(1234, 0))
	assert.Equal(t, int64(1200), ChannelHealthHistoryBucket(1234, -5))
}

// The query must return buckets oldest-first so the chart can plot them without
// re-sorting, and must respect the window bounds.
func TestGetChannelHealthHistoryOrdersAndFilters(t *testing.T) {
	clearChannelHealthHistory(t)

	for _, ts := range []int64{3000, 1000, 2000} {
		entry := ChannelHealthHistory{
			ChannelId: 96, Route: "r", Family: "claude", BucketTs: ts,
			Score: 1, ObservationCount: 1,
		}
		require.NoError(t, UpsertChannelHealthHistory(&entry))
	}

	rows, err := GetChannelHealthHistory(1500, 2500)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(2000), rows[0].BucketTs)

	rows, err = GetChannelHealthHistory(0, 9000)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, int64(1000), rows[0].BucketTs)
	assert.Equal(t, int64(2000), rows[1].BucketTs)
	assert.Equal(t, int64(3000), rows[2].BucketTs)
}
