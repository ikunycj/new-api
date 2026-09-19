package model

import (
	"sort"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// This file implements the in-process counters behind the dashboard's realtime
// RPM / TPM cards.
//
// Design constraints, in priority order:
//
//  1. The relay hot path must not get slower. A request only takes a read lock
//     plus a handful of integer adds on memory that is already resident; there
//     is no allocation, no Redis round trip and no database write.
//  2. Nothing is persisted. The numbers describe the recent past only, so
//     losing them on restart is expected — the hourly quota_data rollup remains
//     the durable record and is what the dashboard's historical charts read.
//  3. Memory must be bounded. One fixed-size ring per user that has sent
//     traffic, reclaimed after a period of inactivity.
//
// The rings are per node. Each instance only sees the requests it served
// itself, so a multi-node deployment reports per-node throughput; the snapshot
// carries NodeName so a caller can tell which node answered.

const (
	// realtimeSlotSeconds is the resolution of the ring. One minute windows are
	// the shortest the dashboard asks for, so ten seconds leaves enough
	// granularity to react within a window without making the ring huge.
	realtimeSlotSeconds = 10

	// realtimeRetentionSeconds is how much history a ring holds. Six hours
	// covers the longest window plus a comfortable margin for the chart.
	realtimeRetentionSeconds = 6 * 60 * 60

	// realtimeSlotCount is the ring length. Each slot is 16 bytes, so one ring
	// costs ~34KB regardless of how much traffic the user generates.
	realtimeSlotCount = realtimeRetentionSeconds / realtimeSlotSeconds

	// realtimeIdleTTL is how long a user's ring survives without traffic before
	// the sweeper frees it. It matches the retention window on purpose: a ring
	// whose newest sample is older than that can no longer contribute to any
	// window the dashboard can ask for, so keeping it would only hold memory
	// for data that can never be read back.
	realtimeIdleTTL = realtimeRetentionSeconds * time.Second

	// realtimeSweepInterval is how often the sweeper runs.
	realtimeSweepInterval = time.Minute

	// realtimeMaxUsers caps the number of live rings. It exists so an attacker
	// cannot grow the map without bound by cycling through accounts; the value
	// is far above any realistic concurrent user count on a single node.
	realtimeMaxUsers = 20000

	// realtimeSeriesBucketSeconds is the resolution of the returned series. The
	// ring stores ten-second slots, but a chart with thousands of points is
	// wasted payload, so points are summed into one-minute buckets.
	realtimeSeriesBucketSeconds = 60

	// realtimeSlowLogInterval ratelimits the "ring limit reached" warning so a
	// flood of rejected users does not itself become a log flood.
	realtimeSlowLogInterval = 60 * time.Second
)

// realtimeNow is indirected so tests can drive the ring with a synthetic clock
// instead of sleeping through real ten-second slots.
var realtimeNow = func() int64 {
	return time.Now().Unix()
}

// realtimeSlot is one ten-second bucket of a user's traffic.
type realtimeSlot struct {
	// Timestamp is the slot's start, aligned to realtimeSlotSeconds. Zero marks
	// an unused slot.
	Timestamp int64
	// Requests counts relay requests that produced usage in this slot.
	Requests int32
	// Tokens counts prompt plus completion tokens, matching how the existing
	// dashboard defines token throughput so the two views agree.
	Tokens int32
}

// realtimeRing is a fixed-size circular buffer of slots for a single user.
// Slots are indexed by absolute slot number modulo the ring length, so an
// advancing clock overwrites the oldest data without any shifting.
type realtimeRing struct {
	mu       sync.Mutex
	slots    [realtimeSlotCount]realtimeSlot
	lastSeen int64
}

// add folds one request into the slot covering now.
func (r *realtimeRing) add(now int64, tokens int) {
	slotStart := now - (now % realtimeSlotSeconds)
	idx := int((slotStart / realtimeSlotSeconds) % realtimeSlotCount)
	slot := &r.slots[idx]
	if slot.Timestamp != slotStart {
		// Either an unused slot or one left over from a full lap of the ring.
		// Both cases mean its previous contents are outside the retention
		// window, so resetting it is the rollover.
		*slot = realtimeSlot{Timestamp: slotStart}
	}
	slot.Requests++
	slot.Tokens += int32(tokens)
	r.lastSeen = now
}

// realtimeWindowResult is the aggregate for one requested window.
type realtimeWindowResult struct {
	Requests int
	Tokens   int
}

// snapshot copies the slots covering the given window so the caller can
// aggregate without holding the ring lock.
func (r *realtimeRing) snapshot(now int64, window int64) []realtimeSlot {
	start := realtimeWindowStart(now, window)
	slots := make([]realtimeSlot, 0, window/realtimeSlotSeconds)
	for i := range r.slots {
		slot := r.slots[i]
		if slot.Timestamp == 0 || slot.Timestamp < start || slot.Timestamp > now {
			continue
		}
		slots = append(slots, slot)
	}
	return slots
}

// realtimeWindowStart returns the first slot included in a trailing window.
//
// The start is snapped down to a slot boundary so a window always spans exactly
// window/realtimeSlotSeconds slots. Without the snap the same nominal window
// would cover six slots at one instant and seven a few seconds later, making
// the reported rate jitter for no operational reason. The cost is that a window
// can reach up to realtimeSlotSeconds-1 further back than its name suggests,
// which is immaterial for a dashboard and is why the value is not documented as
// an exact cut-off.
func realtimeWindowStart(now int64, window int64) int64 {
	slotCount := window / realtimeSlotSeconds
	if slotCount < 1 {
		slotCount = 1
	}
	currentSlot := now - now%realtimeSlotSeconds
	return currentSlot - (slotCount-1)*realtimeSlotSeconds
}

// realtimeRegistry holds one ring per active user.
var realtimeRegistry = struct {
	mu       sync.RWMutex
	users    map[int]*realtimeRing
	lastWarn int64
}{
	users: make(map[int]*realtimeRing),
}

// EnableRealtimeMetricsForTest switches the counters on without starting the
// sweeper, and ResetRealtimeMetricsForTest clears the registry and switches
// them back off. Controller tests need both because main.go's startup path is
// not exercised under `go test ./controller/...`.
func EnableRealtimeMetricsForTest() {
	realtimeEnabled = true
}

// ResetRealtimeMetricsForTest drops every ring and disables the feature again.
func ResetRealtimeMetricsForTest() {
	realtimeEnabled = false
	realtimeRegistry.mu.Lock()
	realtimeRegistry.users = make(map[int]*realtimeRing)
	realtimeRegistry.mu.Unlock()
}

// RecordRealtimeUsage folds one billable request into the user's ring.
// It is called from the relay hot path and does no IO.
func RecordRealtimeUsage(userId int, tokens int) {
	if !realtimeEnabled || userId <= 0 {
		return
	}
	now := realtimeNow()

	realtimeRegistry.mu.RLock()
	ring, ok := realtimeRegistry.users[userId]
	realtimeRegistry.mu.RUnlock()

	if !ok {
		realtimeRegistry.mu.Lock()
		// Re-check under the write lock: a concurrent request for the same user
		// may have created the ring while this one was waiting.
		ring, ok = realtimeRegistry.users[userId]
		if !ok {
			if len(realtimeRegistry.users) >= realtimeMaxUsers {
				// Refuse rather than evict: dropping the newest arrival keeps
				// the ring set stable, and the warning is rate limited so a
				// flood of distinct users cannot turn into a log flood.
				if now-realtimeRegistry.lastWarn >= int64(realtimeSlowLogInterval/time.Second) {
					realtimeRegistry.lastWarn = now
					common.SysLog("realtime metrics: user ring limit reached, further users are not tracked")
				}
				realtimeRegistry.mu.Unlock()
				return
			}
			ring = &realtimeRing{}
			realtimeRegistry.users[userId] = ring
		}
		realtimeRegistry.mu.Unlock()
	}

	ring.mu.Lock()
	ring.add(now, tokens)
	ring.mu.Unlock()
}

// realtimeEnabled is set by InitRealtimeMetrics. Until it is true,
// RecordRealtimeUsage does nothing and no sweeper runs, so the feature costs
// exactly zero when switched off.
var realtimeEnabled bool

// InitRealtimeMetrics starts the idle reclamation loop. It is called once at
// startup rather than lazily from the first request, so the goroutine is not
// spawned from the relay hot path (which would also make it non-deterministic
// under test) and so a node that is later enabled does not need a restart.
func InitRealtimeMetrics() {
	realtimeEnabled = true
	go realtimeSweepLoop()
}

// realtimeSweepLoop periodically frees rings for users that stopped sending
// traffic. Without it the map would keep one ring per user forever.
func realtimeSweepLoop() {
	ticker := time.NewTicker(realtimeSweepInterval)
	defer ticker.Stop()
	for range ticker.C {
		sweepRealtimeRegistry()
	}
}

// sweepRealtimeRegistry drops every ring whose user has been idle past
// realtimeIdleTTL. It is a separate function from the loop so the reclamation
// rule can be tested directly rather than through a minute-long ticker.
func sweepRealtimeRegistry() {
	cutoff := realtimeNow() - int64(realtimeIdleTTL/time.Second)
	realtimeRegistry.mu.Lock()
	defer realtimeRegistry.mu.Unlock()
	for userId, ring := range realtimeRegistry.users {
		ring.mu.Lock()
		idle := ring.lastSeen < cutoff
		ring.mu.Unlock()
		if idle {
			delete(realtimeRegistry.users, userId)
		}
	}
}

// realtimeBucket is one point of the returned series.
type realtimeBucket struct {
	Timestamp int64 `json:"timestamp"`
	Requests  int   `json:"requests"`
	Tokens    int   `json:"tokens"`
}

// RealtimeWindow is throughput over one trailing window.
type RealtimeWindow struct {
	WindowSeconds int     `json:"window_seconds"`
	Requests      int     `json:"requests"`
	Tokens        int     `json:"tokens"`
	RPM           float64 `json:"rpm"`
	TPM           float64 `json:"tpm"`
}

// RealtimeSnapshot is the payload behind the realtime cards and chart.
type RealtimeSnapshot struct {
	UserID int   `json:"user_id"`
	Now    int64 `json:"now"`
	// NodeName identifies which instance produced this snapshot. The rings are
	// per process, so in a multi-node deployment the numbers describe only the
	// traffic this node served.
	NodeName string           `json:"node_name"`
	Windows  []RealtimeWindow `json:"windows"`
	Series   []realtimeBucket `json:"series"`
}

// realtimeWindows are the trailing windows the API reports, in seconds.
var realtimeWindows = []int64{60, 300, 3600}

// GetRealtimeSnapshot builds the snapshot for a user. A user that has never
// sent traffic yields zeroes rather than an error, so the dashboard does not
// need a special empty state.
func GetRealtimeSnapshot(userId int) RealtimeSnapshot {
	now := realtimeNow()

	realtimeRegistry.mu.RLock()
	ring := realtimeRegistry.users[userId]
	realtimeRegistry.mu.RUnlock()

	snapshot := RealtimeSnapshot{
		UserID:   userId,
		Now:      now,
		NodeName: common.NodeName,
		Windows:  make([]RealtimeWindow, 0, len(realtimeWindows)),
	}

	if ring == nil {
		for _, window := range realtimeWindows {
			snapshot.Windows = append(snapshot.Windows, buildWindow(window, realtimeWindowResult{}))
		}
		snapshot.Series = emptySeries(now, realtimeWindows[len(realtimeWindows)-1])
		return snapshot
	}

	ring.mu.Lock()
	// The longest window drives the series, and every shorter window is folded
	// from the same slot scan so a request reads the ring once.
	longest := realtimeWindows[len(realtimeWindows)-1]
	slots := ring.snapshot(now, longest)
	ring.mu.Unlock()

	for _, window := range realtimeWindows {
		snapshot.Windows = append(snapshot.Windows, buildWindow(window, sumSlots(slots, now, window)))
	}
	snapshot.Series = buildSeries(slots, now, longest)
	return snapshot
}

// RealtimeUserSummary is one row of the admin all-users view.
type RealtimeUserSummary struct {
	UserID   int     `json:"user_id"`
	Username string  `json:"username"`
	Requests int     `json:"requests"`
	Tokens   int     `json:"tokens"`
	RPM      float64 `json:"rpm"`
	TPM      float64 `json:"tpm"`
	LastSeen int64   `json:"last_seen"`
}

// GetRealtimeUserSummaries returns every tracked user's traffic over the window,
// busiest first. It is admin-only and reads only in-process state.
func GetRealtimeUserSummaries(window int64) []RealtimeUserSummary {
	now := realtimeNow()

	realtimeRegistry.mu.RLock()
	rings := make(map[int]*realtimeRing, len(realtimeRegistry.users))
	for userId, ring := range realtimeRegistry.users {
		rings[userId] = ring
	}
	realtimeRegistry.mu.RUnlock()

	summaries := make([]RealtimeUserSummary, 0, len(rings))
	for userId, ring := range rings {
		ring.mu.Lock()
		slots := ring.snapshot(now, window)
		lastSeen := ring.lastSeen
		ring.mu.Unlock()

		totals := sumSlots(slots, now, window)
		if totals.Requests == 0 && totals.Tokens == 0 {
			continue
		}
		windowMinutes := float64(window) / 60
		summaries = append(summaries, RealtimeUserSummary{
			UserID:   userId,
			Requests: totals.Requests,
			Tokens:   totals.Tokens,
			RPM:      float64(totals.Requests) / windowMinutes,
			TPM:      float64(totals.Tokens) / windowMinutes,
			LastSeen: lastSeen,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Requests != summaries[j].Requests {
			return summaries[i].Requests > summaries[j].Requests
		}
		return summaries[i].UserID < summaries[j].UserID
	})
	return summaries
}

// FillRealtimeUsernames resolves the username for each summary. The registry
// only stores user ids, and the admin table wants something readable, so the
// names are looked up in one query at read time rather than being cached on the
// hot path.
func FillRealtimeUsernames(summaries []RealtimeUserSummary) {
	if len(summaries) == 0 {
		return
	}
	ids := make([]int, 0, len(summaries))
	seen := make(map[int]bool, len(summaries))
	for _, summary := range summaries {
		if seen[summary.UserID] {
			continue
		}
		seen[summary.UserID] = true
		ids = append(ids, summary.UserID)
	}

	var users []User
	if err := DB.Select("id", "username").Where("id in (?)", ids).Find(&users).Error; err != nil {
		// A failed lookup only costs the display name; the throughput numbers
		// are already computed and are the point of the endpoint.
		common.SysLog("realtime metrics: failed to resolve usernames: " + err.Error())
		return
	}
	names := make(map[int]string, len(users))
	for _, user := range users {
		names[user.Id] = user.Username
	}
	for i := range summaries {
		if name, ok := names[summaries[i].UserID]; ok {
			summaries[i].Username = name
		}
	}
}

// realtimeSeriesStart returns the timestamp of the first returned bucket.
//
// Unlike the window start, this is snapped to a minute rather than to a slot:
// the series exists to be drawn, and a chart whose x-axis ticks all sit offset
// from the minute boundary reads badly. Snapping means the minute-aligned span
// can be up to a minute shorter than the requested window, so the series gets
// one extra bucket to compensate and always covers the whole window. The first
// and last buckets are therefore partial, which is normal for a live chart and
// keeps the series total equal to the corresponding card.
func realtimeSeriesStart(now int64, window int64) int64 {
	lastBucket := now - now%realtimeSeriesBucketSeconds
	return lastBucket - window
}

// realtimeSeriesBucketCount is how many points the series carries: the window
// at one-minute resolution, plus one for the alignment described above.
func realtimeSeriesBucketCount(window int64) int {
	count := int(window/realtimeSeriesBucketSeconds) + 1
	if count < 1 {
		count = 1
	}
	return count
}

// sumSlots totals the slots inside the trailing window. Slots are assumed to
// have been pre-filtered by realtimeRing.snapshot, which applies the same
// aligned start.
func sumSlots(slots []realtimeSlot, now int64, window int64) realtimeWindowResult {
	start := realtimeWindowStart(now, window)
	var result realtimeWindowResult
	for _, slot := range slots {
		if slot.Timestamp < start || slot.Timestamp > now {
			continue
		}
		result.Requests += int(slot.Requests)
		result.Tokens += int(slot.Tokens)
	}
	return result
}

// buildSeries folds the slots into one-minute buckets covering the window.
func buildSeries(slots []realtimeSlot, now int64, window int64) []realtimeBucket {
	start := realtimeSeriesStart(now, window)
	bucketCount := realtimeSeriesBucketCount(window)
	series := make([]realtimeBucket, bucketCount)
	for i := range series {
		series[i].Timestamp = start + int64(i)*realtimeSeriesBucketSeconds
	}
	for _, slot := range slots {
		index := int((slot.Timestamp - start) / realtimeSeriesBucketSeconds)
		if index < 0 || index >= bucketCount {
			continue
		}
		series[index].Requests += int(slot.Requests)
		series[index].Tokens += int(slot.Tokens)
	}
	return series
}

// emptySeries returns a zeroed series for a user with no recorded traffic, so
// the chart renders an empty axis instead of a single point.
func emptySeries(now int64, window int64) []realtimeBucket {
	start := realtimeSeriesStart(now, window)
	series := make([]realtimeBucket, realtimeSeriesBucketCount(window))
	for i := range series {
		series[i].Timestamp = start + int64(i)*realtimeSeriesBucketSeconds
	}
	return series
}

// buildWindow converts raw totals into a window result with derived rates.
func buildWindow(window int64, result realtimeWindowResult) RealtimeWindow {
	windowMinutes := float64(window) / 60
	return RealtimeWindow{
		WindowSeconds: int(window),
		Requests:      result.Requests,
		Tokens:        result.Tokens,
		RPM:           float64(result.Requests) / windowMinutes,
		TPM:           float64(result.Tokens) / windowMinutes,
	}
}
