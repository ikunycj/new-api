package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func TestChannelProbeDueSkipsChannelsWithRecentTraffic(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)

	now := time.Now()
	idleGrace := 5 * time.Minute

	// Real traffic 10 seconds ago: probing would waste an upstream call for a
	// channel we already have fresh evidence about.
	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: 501, Route: "/v1/messages", Success: true,
		Observed: now.Add(-10 * time.Second),
	})
	if channelProbeDue(501, now, idleGrace) {
		t.Fatal("channel with recent real traffic must not be probed")
	}

	// Traffic older than the grace window: the score is going stale, so a
	// synthetic probe is warranted.
	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: 502, Route: "/v1/messages", Success: true,
		Observed: now.Add(-10 * time.Minute),
	})
	if !channelProbeDue(502, now, idleGrace) {
		t.Fatal("channel idle beyond the grace window must be probed")
	}
}

// A channel nobody has ever exercised is exactly the case probing exists for.
func TestChannelProbeDueProbesUnknownChannel(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)

	if !channelProbeDue(9999, time.Now(), 5*time.Minute) {
		t.Fatal("never-observed channel must be probed")
	}
}

// A zero grace window is allowed but means "probe everything every tick".
func TestChannelProbeDueZeroGraceAlwaysProbes(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)

	RecordChannelHealthSample(ChannelHealthSample{ChannelID: 503, Route: "r", Success: true})
	if !channelProbeDue(503, time.Now(), 0) {
		t.Fatal("zero idle grace must probe unconditionally")
	}
}

// Traffic on any route counts as activity for the channel, otherwise a busy
// channel would still be probed because one specific route was quiet.
func TestChannelLastObservedAtUsesLatestRoute(t *testing.T) {
	enableChannelHealthForTest(t, 3600, 0)

	base := time.Now()
	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: 504, Route: "/v1/a", Success: true, Observed: base.Add(-time.Hour),
	})
	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: 504, Route: "/v1/b", Success: true, Observed: base,
	})

	observed, ok := ChannelLastObservedAt(504)
	if !ok {
		t.Fatal("expected an observation timestamp")
	}
	if observed.Before(base) {
		t.Fatalf("expected latest route timestamp, got %v want >= %v", observed, base)
	}
}

func TestChannelLastObservedAtUnknownChannel(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)

	if _, ok := ChannelLastObservedAt(8888); ok {
		t.Fatal("unknown channel must not report an observation")
	}
}

// The probe loop must stay dormant unless both switches are on, because it is
// the only part of the subsystem that spends upstream quota.
func TestChannelProbeRequiresBothSwitches(t *testing.T) {
	t.Cleanup(func() {
		common.SetChannelHealthEnabled(false)
		common.SetChannelHealthProbeEnabled(false)
	})

	common.SetChannelHealthEnabled(false)
	common.SetChannelHealthProbeEnabled(true)
	if common.IsChannelHealthProbeEnabled() {
		t.Fatal("probing must stay off while channel health is disabled")
	}

	common.SetChannelHealthEnabled(true)
	common.SetChannelHealthProbeEnabled(false)
	if common.IsChannelHealthProbeEnabled() {
		t.Fatal("probing must stay off while the probe switch is disabled")
	}

	common.SetChannelHealthProbeEnabled(true)
	if !common.IsChannelHealthProbeEnabled() {
		t.Fatal("probing should be enabled once both switches are on")
	}
}

// Without a registered executor the scheduler must not attempt to invent its
// own upstream request format.
func TestRunChannelHealthProbeOnceRequiresExecutor(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)
	common.SetChannelHealthProbeEnabled(true)
	SetChannelProbeExecutor(nil)
	t.Cleanup(func() {
		common.SetChannelHealthProbeEnabled(false)
		SetChannelProbeExecutor(nil)
	})

	// Must return without panicking even though the database is unavailable in
	// this unit test: the executor guard is checked first.
	RunChannelHealthProbeOnce()
}

// A disabled subsystem must never call the executor.
func TestRunChannelHealthProbeOnceDisabledDoesNotExecute(t *testing.T) {
	common.SetChannelHealthEnabled(false)
	common.SetChannelHealthProbeEnabled(false)

	var calls atomic.Int64
	SetChannelProbeExecutor(func(context.Context, *model.Channel) (ChannelProbeResult, error) {
		calls.Add(1)
		return ChannelProbeResult{Success: true, Latency: time.Millisecond}, nil
	})
	t.Cleanup(func() { SetChannelProbeExecutor(nil) })

	RunChannelHealthProbeOnce()

	if calls.Load() != 0 {
		t.Fatalf("expected no probe calls while disabled, got %d", calls.Load())
	}
}

func TestSetChannelProbeExecutorRoundTrip(t *testing.T) {
	t.Cleanup(func() { SetChannelProbeExecutor(nil) })

	if loadChannelProbeExecutor() != nil {
		SetChannelProbeExecutor(nil)
		if loadChannelProbeExecutor() != nil {
			t.Fatal("expected executor to be cleared")
		}
	}

	sentinel := errors.New("probe failed")
	SetChannelProbeExecutor(func(context.Context, *model.Channel) (ChannelProbeResult, error) {
		return ChannelProbeResult{Latency: 5 * time.Millisecond}, sentinel
	})

	executor := loadChannelProbeExecutor()
	if executor == nil {
		t.Fatal("expected executor to be registered")
	}
	result, err := executor(context.Background(), &model.Channel{Id: 1})
	if result.Success {
		t.Fatal("expected failure result from stub executor")
	}
	if result.Latency != 5*time.Millisecond {
		t.Fatalf("unexpected latency %v", result.Latency)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("unexpected error %v", err)
	}
}

// Probe samples are recorded on a dedicated pseudo-route so that synthetic
// results cannot dilute the scores earned from real traffic on real routes.
func TestProbeSamplesUseDedicatedRoute(t *testing.T) {
	enableChannelHealthForTest(t, 3600, 0)

	base := time.Now()
	for i := 0; i < 10; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 505, Route: "/v1/messages", Success: true,
			Observed: base.Add(time.Duration(i) * time.Millisecond),
		})
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 505, Route: ChannelHealthProbeRoute, Success: false,
			Observed: base.Add(time.Duration(i) * time.Millisecond),
		})
	}

	liveRoute, ok := GetChannelHealthSnapshot(505, "/v1/messages", "")
	if !ok {
		t.Fatal("expected snapshot for the real route")
	}
	probeRoute, ok := GetChannelHealthSnapshot(505, ChannelHealthProbeRoute, "")
	if !ok {
		t.Fatal("expected snapshot for the probe route")
	}

	if liveRoute.Availability <= probeRoute.Availability {
		t.Fatalf("probe failures must not drag down the real route: real=%v probe=%v",
			liveRoute.Availability, probeRoute.Availability)
	}
	if liveRoute.Availability < 0.9 {
		t.Fatalf("successful real traffic should stay healthy, got %v", liveRoute.Availability)
	}
}

func TestGetChannelProbeRunStatsIsReadable(t *testing.T) {
	stats := GetChannelProbeRunStats()
	if stats.LastProbed < 0 || stats.LastSkipped < 0 || stats.LastFailed < 0 {
		t.Fatalf("probe stats must never be negative: %+v", stats)
	}
}
