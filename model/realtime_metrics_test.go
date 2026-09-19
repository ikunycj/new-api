package model

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// The ring works in ten-second slots, so these tests drive it with a synthetic
// clock. Every timestamp below is chosen to be a multiple of the slot size
// unless the test is deliberately probing misalignment.
const realtimeTestNow = int64(1_700_000_000)

// freezeRealtimeClock installs a synthetic clock for the duration of a test and
// restores the real one afterwards.
func freezeRealtimeClock(t *testing.T, start int64) *int64 {
	t.Helper()
	current := new(int64)
	*current = start
	original := realtimeNow
	realtimeNow = func() int64 { return *current }
	t.Cleanup(func() { realtimeNow = original })
	return current
}

// resetRealtimeRegistry clears the global registry so tests do not observe each
// other's rings, and restores whatever was there on the way out. It also
// switches the feature on, which main.go does at startup.
func resetRealtimeRegistry(t *testing.T) {
	t.Helper()
	originalEnabled := realtimeEnabled
	realtimeEnabled = true
	realtimeRegistry.mu.Lock()
	original := realtimeRegistry.users
	realtimeRegistry.users = make(map[int]*realtimeRing)
	realtimeRegistry.mu.Unlock()
	t.Cleanup(func() {
		realtimeEnabled = originalEnabled
		realtimeRegistry.mu.Lock()
		realtimeRegistry.users = original
		realtimeRegistry.mu.Unlock()
	})
}

// windowsByLength indexes a snapshot's windows for readable assertions.
func windowsByLength(snapshot RealtimeSnapshot) map[int]RealtimeWindow {
	byWindow := make(map[int]RealtimeWindow, len(snapshot.Windows))
	for _, window := range snapshot.Windows {
		byWindow[window.WindowSeconds] = window
	}
	return byWindow
}

// recordAtSlots records one request in each of count consecutive slots ending
// at the slot containing now, so a window spanning exactly count slots sees
// exactly count requests no matter where inside a slot the caller stands.
func recordAtSlots(current *int64, now int64, count int, userId int, tokens int) {
	base := now - now%realtimeSlotSeconds
	for i := 0; i < count; i++ {
		*current = base - int64(count-1-i)*realtimeSlotSeconds
		RecordRealtimeUsage(userId, tokens)
	}
}

func TestRealtimeSnapshotSumsTrailingWindows(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	// One request per ten-second slot for a full hour: 360 requests.
	recordAtSlots(current, realtimeTestNow, 360, 42, 100)
	*current = realtimeTestNow

	snapshot := GetRealtimeSnapshot(42)
	require.Len(t, snapshot.Windows, 3)
	byWindow := windowsByLength(snapshot)

	// Each window covers exactly window/realtimeSlotSeconds slots, and the
	// per-slot rate is constant, so RPM and TPM are identical across all three.
	for _, windowSeconds := range []int{60, 300, 3600} {
		window, ok := byWindow[windowSeconds]
		require.True(t, ok, "missing %ds window", windowSeconds)
		expectedRequests := windowSeconds / realtimeSlotSeconds
		require.Equal(t, expectedRequests, window.Requests, "%ds requests", windowSeconds)
		require.Equal(t, expectedRequests*100, window.Tokens, "%ds tokens", windowSeconds)
		require.InDelta(t, float64(expectedRequests)/(float64(windowSeconds)/60), window.RPM, 0.001)
		require.InDelta(t, float64(expectedRequests*100)/(float64(windowSeconds)/60), window.TPM, 0.001)
	}

	// Sanity-check one concrete pair so the loop above cannot pass vacuously.
	require.Equal(t, 6, byWindow[60].Requests)
	require.InDelta(t, 6.0, byWindow[60].RPM, 0.001)
	require.InDelta(t, 600.0, byWindow[60].TPM, 0.001)
}

func TestRealtimeWindowCoverageDoesNotDependOnSecondWithinSlot(t *testing.T) {
	// A window start that is not snapped to a slot boundary would cover seven
	// slots at some seconds and six at others, making the reported rate jitter
	// by ~17% purely from when the browser happens to poll.
	resetRealtimeRegistry(t)

	current := freezeRealtimeClock(t, realtimeTestNow)
	recordAtSlots(current, realtimeTestNow, 360, 77, 10)

	for offset := int64(0); offset < realtimeSlotSeconds; offset++ {
		*current = realtimeTestNow + offset
		window := windowsByLength(GetRealtimeSnapshot(77))[60]
		require.Equal(t, 6, window.Requests,
			"a 60s window must always cover exactly 6 slots (offset %d)", offset)
	}
}

func TestRealtimeSlotsOutsideWindowAreExcluded(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	*current = realtimeTestNow - 1800
	RecordRealtimeUsage(7, 500)
	*current = realtimeTestNow

	snapshot := GetRealtimeSnapshot(7)
	byWindow := windowsByLength(snapshot)

	require.Equal(t, 0, byWindow[60].Requests, "60s window must not see a half-hour-old request")
	require.Zero(t, byWindow[60].Tokens)
	require.Zero(t, byWindow[60].RPM)

	// The 1h window still covers it, and RPM for an hour is requests/60.
	require.Equal(t, 1, byWindow[3600].Requests)
	require.Equal(t, 500, byWindow[3600].Tokens)
	require.InDelta(t, 500.0/60, byWindow[3600].TPM, 0.001)
}

func TestRealtimeRingOverwritesAfterAFullLap(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	// Fill the slot the ring will reuse first, then come back one full lap
	// later. The reused slot must be reset rather than accumulated onto.
	*current = realtimeTestNow
	RecordRealtimeUsage(9, 111)
	RecordRealtimeUsage(9, 111)

	*current = realtimeTestNow + realtimeRetentionSeconds
	RecordRealtimeUsage(9, 7)

	window := windowsByLength(GetRealtimeSnapshot(9))[60]
	require.Equal(t, 1, window.Requests, "data one full lap old must not be counted")
	require.Equal(t, 7, window.Tokens, "the reused slot must start from zero")
}

func TestRealtimeSeriesIsBoundedAndOrdered(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	recordAtSlots(current, realtimeTestNow, 360, 11, 10)
	*current = realtimeTestNow

	snapshot := GetRealtimeSnapshot(11)

	lastBucket := snapshot.Now - snapshot.Now%realtimeSeriesBucketSeconds

	// The series covers the longest window at one-minute resolution, plus the
	// alignment bucket at the start.
	require.Len(t, snapshot.Series, int(realtimeWindows[2]/realtimeSeriesBucketSeconds)+1)
	require.Equal(t, lastBucket-realtimeWindows[2], snapshot.Series[0].Timestamp,
		"the series must start exactly one window before the current minute")

	for i := 1; i < len(snapshot.Series); i++ {
		require.Greater(t, snapshot.Series[i].Timestamp, snapshot.Series[i-1].Timestamp,
			"series must be strictly increasing in time")
	}

	// The series is minute-aligned and so ends with the current minute; nothing
	// may be timestamped past it.
	for _, bucket := range snapshot.Series {
		require.LessOrEqual(t, bucket.Timestamp, lastBucket,
			"a bucket must never be timestamped in the future")
		require.Zero(t, bucket.Timestamp%realtimeSeriesBucketSeconds,
			"bucket timestamps must align to the bucket size")
	}

	totalRequests := 0
	totalTokens := 0
	for _, bucket := range snapshot.Series {
		totalRequests += bucket.Requests
		totalTokens += bucket.Tokens
	}
	// recordAtSlots wrote one request per slot for the last 3600 seconds, so
	// every one of them falls inside the returned span regardless of how the
	// minute alignment shifted it — and nothing outside it.
	require.Equal(t, 360, totalRequests, "every request inside the window must land in a bucket")
	require.Equal(t, 3600, totalTokens)
}

func TestRealtimeSeriesBucketsAlignToOneMinute(t *testing.T) {
	// now sits mid-minute so the six slots ending at its slot all fall inside
	// the same minute-aligned bucket (the one starting at now-30).
	const now = int64(1_700_000_035)
	require.Equal(t, int64(50), (now-now%realtimeSlotSeconds)%realtimeSeriesBucketSeconds)

	current := freezeRealtimeClock(t, now)
	resetRealtimeRegistry(t)
	for i := int64(0); i < 6; i++ {
		*current = now - (5-i)*realtimeSlotSeconds
		RecordRealtimeUsage(13, 5)
	}
	*current = now

	snapshot := GetRealtimeSnapshot(13)
	nonEmpty := 0
	for _, bucket := range snapshot.Series {
		if bucket.Requests == 0 {
			continue
		}
		nonEmpty++
		require.Equal(t, 6, bucket.Requests)
		require.Equal(t, 30, bucket.Tokens)
		require.Zero(t, bucket.Timestamp%realtimeSeriesBucketSeconds,
			"bucket timestamps must align to the bucket size")
	}
	require.Equal(t, 1, nonEmpty, "six slots in one minute must yield exactly one non-empty bucket")
}

func TestRealtimeSnapshotForUnknownUserIsEmptyNotError(t *testing.T) {
	freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	snapshot := GetRealtimeSnapshot(999)
	require.Equal(t, 999, snapshot.UserID)
	require.Len(t, snapshot.Windows, 3)
	for _, window := range snapshot.Windows {
		require.Zero(t, window.Requests)
		require.Zero(t, window.Tokens)
		require.Zero(t, window.RPM)
		require.Zero(t, window.TPM)
	}
	require.NotEmpty(t, snapshot.Series, "the chart needs a series even with no traffic")
	for _, bucket := range snapshot.Series {
		require.Zero(t, bucket.Requests)
	}
}

func TestRealtimeIsInertUntilInitialized(t *testing.T) {
	freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	originalEnabled := realtimeEnabled
	realtimeEnabled = false
	t.Cleanup(func() { realtimeEnabled = originalEnabled })

	RecordRealtimeUsage(7, 1000)

	realtimeRegistry.mu.RLock()
	defer realtimeRegistry.mu.RUnlock()
	require.Empty(t, realtimeRegistry.users,
		"a disabled feature must not allocate anything on the relay path")
}

func TestRealtimeIgnoresNonPositiveUserIds(t *testing.T) {
	freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	RecordRealtimeUsage(0, 100)
	RecordRealtimeUsage(-1, 100)

	realtimeRegistry.mu.RLock()
	defer realtimeRegistry.mu.RUnlock()
	require.Empty(t, realtimeRegistry.users,
		"an unauthenticated request must not allocate a ring")
}

func TestRealtimeUserSummariesSortByRequestsAndHideIdleUsers(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	for i := 0; i < 5; i++ {
		RecordRealtimeUsage(1, 10)
	}
	for i := 0; i < 20; i++ {
		RecordRealtimeUsage(2, 10)
	}
	// User 3 only has traffic far outside the requested window.
	*current = realtimeTestNow - 1800
	for i := 0; i < 50; i++ {
		RecordRealtimeUsage(3, 10)
	}
	*current = realtimeTestNow

	summaries := GetRealtimeUserSummaries(60)
	require.Len(t, summaries, 2, "a user with no traffic in the window is omitted")
	require.Equal(t, 2, summaries[0].UserID)
	require.Equal(t, 20, summaries[0].Requests)
	require.InDelta(t, 20.0, summaries[0].RPM, 0.001)
	require.Equal(t, 1, summaries[1].UserID)
	require.Equal(t, 5, summaries[1].Requests)
	require.Equal(t, realtimeTestNow, summaries[1].LastSeen)
}

func TestRealtimeConcurrentRecordingDoesNotLoseUpdates(t *testing.T) {
	freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	const writers = 8
	const perWriter = 200

	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				RecordRealtimeUsage(5, 3)
			}
		}()
	}
	wg.Wait()

	window := windowsByLength(GetRealtimeSnapshot(5))[60]
	require.Equal(t, writers*perWriter, window.Requests)
	require.Equal(t, writers*perWriter*3, window.Tokens)
}

func TestRealtimeConcurrentUsersShareNoState(t *testing.T) {
	freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	var wg sync.WaitGroup
	for user := 1; user <= 16; user++ {
		wg.Add(1)
		go func(userId int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				RecordRealtimeUsage(userId, userId)
			}
		}(user)
	}
	wg.Wait()

	for user := 1; user <= 16; user++ {
		window := windowsByLength(GetRealtimeSnapshot(user))[60]
		require.Equal(t, 50, window.Requests, "user %d", user)
		require.Equal(t, 50*user, window.Tokens, "user %d", user)
	}
}

func TestRealtimeSweepReclaimsIdleRings(t *testing.T) {
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	RecordRealtimeUsage(100, 10)
	RecordRealtimeUsage(200, 10)

	// Nothing is idle yet, so a sweep at this instant must be a no-op.
	sweepRealtimeRegistry()
	realtimeRegistry.mu.RLock()
	require.Contains(t, realtimeRegistry.users, 100, "a fresh ring must not be swept")
	require.Contains(t, realtimeRegistry.users, 200, "a fresh ring must not be swept")
	realtimeRegistry.mu.RUnlock()

	// Age both rings past the idle TTL, then touch one of them so it survives.
	*current = realtimeTestNow + int64(realtimeIdleTTL.Seconds()) + 60
	RecordRealtimeUsage(200, 10)

	sweepRealtimeRegistry()

	realtimeRegistry.mu.RLock()
	defer realtimeRegistry.mu.RUnlock()
	require.NotContains(t, realtimeRegistry.users, 100, "idle ring must be reclaimed")
	require.Contains(t, realtimeRegistry.users, 200, "recently active ring must survive")
}

func TestRealtimeSweepKeepsRingsAtTheIdleBoundary(t *testing.T) {
	// The comparison is lastSeen < cutoff, so a ring whose last activity is
	// exactly the TTL ago is still live. Pinning the boundary keeps a future
	// refactor from silently turning "<" into "<=" and reclaiming a ring a
	// second early.
	current := freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	RecordRealtimeUsage(300, 10)
	*current = realtimeTestNow + int64(realtimeIdleTTL.Seconds())

	sweepRealtimeRegistry()

	realtimeRegistry.mu.RLock()
	defer realtimeRegistry.mu.RUnlock()
	require.Contains(t, realtimeRegistry.users, 300)
}

func TestRealtimeRingRejectsNewUsersAtCapacity(t *testing.T) {
	freezeRealtimeClock(t, realtimeTestNow)
	resetRealtimeRegistry(t)

	realtimeRegistry.mu.Lock()
	for i := 0; i < realtimeMaxUsers; i++ {
		realtimeRegistry.users[i+1] = &realtimeRing{}
	}
	realtimeRegistry.mu.Unlock()

	RecordRealtimeUsage(realtimeMaxUsers+1, 10)

	realtimeRegistry.mu.RLock()
	defer realtimeRegistry.mu.RUnlock()
	require.Len(t, realtimeRegistry.users, realtimeMaxUsers,
		"a new user must not grow the map past the cap")
	require.NotContains(t, realtimeRegistry.users, realtimeMaxUsers+1)
}

func TestRealtimeRingMemoryStaysBounded(t *testing.T) {
	// Guard the constants that determine worst-case memory: a ring is
	// retention/resolution slots of two int32 counters, and the registry is
	// capped at realtimeMaxUsers rings.
	require.Equal(t, realtimeSlotCount, realtimeRetentionSeconds/realtimeSlotSeconds)
	require.Equal(t, 2160, realtimeSlotCount, "10s slots over 6h")
	require.Equal(t, 20000, realtimeMaxUsers)

	slotBytes := 16 // int64 timestamp + 2 int32 counters, no padding
	perRing := int64(realtimeSlotCount) * int64(slotBytes)
	worstCase := perRing * int64(realtimeMaxUsers)
	require.Less(t, worstCase, int64(800<<20),
		"worst-case ring memory must stay well under a gigabyte")
}

func TestRealtimeWindowStartAlignsToSlotBoundary(t *testing.T) {
	// Directly pin the alignment helper: a 60s window always spans six slots,
	// and its start is always a slot boundary.
	for offset := int64(0); offset < 100; offset++ {
		now := realtimeTestNow + offset
		start := realtimeWindowStart(now, 60)
		require.Zero(t, start%realtimeSlotSeconds,
			"window start must be aligned to %ds (now=%d)", realtimeSlotSeconds, now)
		require.Equal(t, int64(6), (now-now%realtimeSlotSeconds-start)/realtimeSlotSeconds+1,
			"a 60s window must cover six slots (now=%d)", now)
	}

	// A window shorter than one slot still covers the current slot.
	require.Equal(t, realtimeNow()-realtimeNow()%realtimeSlotSeconds, realtimeWindowStart(realtimeNow(), 1))
}
