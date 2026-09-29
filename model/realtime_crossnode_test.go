package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// publishedSummary round-trips slots through the exact bytes a node writes to
// Redis, so the tests exercise the wire format rather than the Go struct.
func publishedSummary(t *testing.T, slots []realtimeSlot, now int64) realtimeNodeSummary {
	t.Helper()
	b, err := marshalRealtimeSummary(slots, now)
	require.NoError(t, err)
	var summary realtimeNodeSummary
	require.NoError(t, json.Unmarshal(b, &summary))
	return summary
}

// slotsAt builds one slot per timestamp, each carrying the given request count
// and a token count derived from it so a misplaced bucket shows up in both.
func slotsAt(requests int32, timestamps ...int64) []realtimeSlot {
	slots := make([]realtimeSlot, 0, len(timestamps))
	for _, ts := range timestamps {
		slots = append(slots, realtimeSlot{
			Timestamp:        ts,
			Requests:         requests,
			Tokens:           requests * 100,
			CacheReadTokens:  requests * 10,
			InputTokensTotal: requests * 50,
		})
	}
	return slots
}

func sumSeriesRequests(series []realtimeBucket) int {
	total := 0
	for _, bucket := range series {
		total += bucket.Requests
	}
	return total
}

func TestRealtimeSummarySeriesIsSparseAndMinuteAligned(t *testing.T) {
	now := realtimeTestNow
	slot := now - now%realtimeSlotSeconds
	// Two slots in the current minute, one slot twenty minutes back.
	slots := slotsAt(2, slot, slot-10, slot-20*60)

	summary := publishedSummary(t, slots, now)

	require.Len(t, summary.Series, 2, "only non-empty buckets are published")
	for _, bucket := range summary.Series {
		require.Zero(t, bucket[0]%realtimeSeriesBucketSeconds, "bucket timestamps are absolute minutes")
	}
	require.Less(t, summary.Series[0][0], summary.Series[1][0], "buckets are oldest first")

	total := int64(0)
	for _, bucket := range summary.Series {
		total += bucket[1]
	}
	require.EqualValues(t, summary.Windows[3600].Requests, total,
		"the published series must reconcile with the published 1h total")
}

// The core invariant: merging a remote node's published summary into the
// local series gives the same chart as if one node had served all the
// traffic, even though the two nodes built their views at different `now`s.
func TestMergeRemoteSeriesMatchesSingleNodeChart(t *testing.T) {
	readerNow := realtimeTestNow
	// The remote published a few seconds earlier, as it does in production
	// (publish interval plus clock skew). Pick an offset that lands the remote
	// in the previous minute so an index-based merge would visibly shift.
	publisherNow := readerNow - 25
	require.NotEqual(t, readerNow/60, publisherNow/60, "test needs the two nows in different minutes")

	readerSlot := readerNow - readerNow%realtimeSlotSeconds
	local := slotsAt(1, readerSlot, readerSlot-60, readerSlot-5*60)
	remote := slotsAt(3, readerSlot-60, readerSlot-2*60, readerSlot-30*60)

	longest := realtimeWindows[len(realtimeWindows)-1]
	series := buildSeries(local, readerNow, longest)
	mergeRemoteSeries(series, []realtimeNodeSummary{publishedSummary(t, remote, publisherNow)})

	all := append(append([]realtimeSlot{}, local...), remote...)
	require.Equal(t, buildSeries(all, readerNow, longest), series)
}

func TestMergeRemoteSeriesIntoEmptyLocalSeries(t *testing.T) {
	now := realtimeTestNow
	slot := now - now%realtimeSlotSeconds
	remote := slotsAt(4, slot, slot-3*60)

	longest := realtimeWindows[len(realtimeWindows)-1]
	series := emptySeries(now, longest)
	mergeRemoteSeries(series, []realtimeNodeSummary{publishedSummary(t, remote, now)})

	require.Equal(t, buildSeries(remote, now, longest), series,
		"a node with no local traffic must still draw the other node's curve")
}

func TestMergeRemoteSeriesDropsBucketsOutsideLocalSpan(t *testing.T) {
	now := realtimeTestNow
	longest := realtimeWindows[len(realtimeWindows)-1]
	series := emptySeries(now, longest)
	start := series[0].Timestamp
	end := series[len(series)-1].Timestamp

	remotes := []realtimeNodeSummary{{Series: []realtimeSummaryBucket{
		{start - realtimeSeriesBucketSeconds, 7, 0, 0, 0}, // rolled off locally
		{end + realtimeSeriesBucketSeconds, 7, 0, 0, 0},   // from a clock running ahead
		{start + 30, 7, 0, 0, 0},                          // not minute-aligned
		{start, 1, 0, 0, 0},
	}}}
	mergeRemoteSeries(series, remotes)

	require.Equal(t, 1, sumSeriesRequests(series))
	require.Equal(t, 1, series[0].Requests)
}

// A node still running a build from before the series was published sends
// only windows. Its cards must still merge; its chart simply contributes
// nothing, which is the pre-existing behaviour, so a rolling deploy is safe.
func TestMergeToleratesSummaryWithoutSeries(t *testing.T) {
	var legacy realtimeNodeSummary
	require.NoError(t, json.Unmarshal([]byte(`{"w":{"60":{"Requests":5},"300":{"Requests":9},"3600":{"Requests":20}}}`), &legacy))
	require.Nil(t, legacy.Series)

	now := realtimeTestNow
	slot := now - now%realtimeSlotSeconds
	local := slotsAt(1, slot)
	longest := realtimeWindows[len(realtimeWindows)-1]

	series := buildSeries(local, now, longest)
	mergeRemoteSeries(series, []realtimeNodeSummary{legacy})
	require.Equal(t, buildSeries(local, now, longest), series)

	merged := mergeRemoteWindow(sumSlots(local, now, 3600), 3600, []realtimeNodeSummary{legacy})
	require.Equal(t, 21, merged.Requests)
}

func TestMergeRemoteWindowSumsEveryRemote(t *testing.T) {
	now := realtimeTestNow
	slot := now - now%realtimeSlotSeconds
	remotes := []realtimeNodeSummary{
		publishedSummary(t, slotsAt(2, slot), now),
		publishedSummary(t, slotsAt(3, slot, slot-10), now),
	}

	local := realtimeWindowResult{Requests: 1, Tokens: 100}
	merged := mergeRemoteWindow(local, 60, remotes)

	require.Equal(t, 1+2+3*2, merged.Requests)
	require.Equal(t, 100+200+300*2, merged.Tokens)
	require.Equal(t, 20+30*2, merged.CacheReadTokens)
	require.Equal(t, 100+150*2, merged.InputTokensTotal)
	require.Equal(t, local, mergeRemoteWindow(local, 60, nil), "no remotes leaves local untouched")
}
