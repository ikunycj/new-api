package service

import (
	"math"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// enableChannelHealthForTest turns the subsystem on with deterministic tuning
// and restores the previous configuration afterwards.
func enableChannelHealthForTest(t *testing.T, halfLifeSeconds, minSamples int) {
	t.Helper()
	common.SetChannelHealthEnabled(true)
	common.SetChannelHealthHalfLifeSeconds(halfLifeSeconds)
	common.SetChannelHealthMinSamples(minSamples)
	common.SetChannelHealthLatencyHalfLifeSeconds(600)
	common.SetChannelHealthStateTTLSeconds(3600)
	ResetChannelHealthState()
	t.Cleanup(func() {
		common.SetChannelHealthEnabled(false)
		common.SetChannelHealthHalfLifeSeconds(common.DefaultChannelHealthHalfLifeSeconds)
		common.SetChannelHealthMinSamples(common.DefaultChannelHealthMinSamples)
		ResetChannelHealthState()
	})
}

// A disabled subsystem must not allocate state, so an upgrade that ships this
// code cannot change behaviour until an operator opts in.
func TestChannelHealthDisabledRecordsNothing(t *testing.T) {
	common.SetChannelHealthEnabled(false)
	ResetChannelHealthState()

	RecordChannelHealthSample(ChannelHealthSample{ChannelID: 1, Route: "/v1/messages", Success: false})

	if _, ok := GetChannelHealthSnapshot(1, "/v1/messages", ""); ok {
		t.Fatal("expected no snapshot while channel health is disabled")
	}
	if score := GetChannelHealthScore(1, "/v1/messages", ""); score != neutralHealthScore {
		t.Fatalf("expected neutral score while disabled, got %v", score)
	}
}

// An unknown channel must read as neutral rather than zero, otherwise routing
// would treat every new channel as broken.
func TestChannelHealthUnknownChannelIsNeutral(t *testing.T) {
	enableChannelHealthForTest(t, 300, 5)

	if score := GetChannelHealthScore(4242, "/v1/messages", ""); score != neutralHealthScore {
		t.Fatalf("expected neutral score for unknown channel, got %v", score)
	}
}

func TestChannelHealthFailuresLowerScore(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)

	base := time.Now()
	for i := 0; i < 10; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 7,
			Route:     "/v1/messages",
			Success:   false,
			Observed:  base.Add(time.Duration(i) * time.Second),
		})
	}

	snapshot, ok := GetChannelHealthSnapshot(7, "/v1/messages", "")
	if !ok {
		t.Fatal("expected snapshot after recording failures")
	}
	if snapshot.Failures != 10 {
		t.Fatalf("expected 10 failures, got %d", snapshot.Failures)
	}
	if snapshot.Availability >= 0.9 {
		t.Fatalf("expected availability to drop after repeated failures, got %v", snapshot.Availability)
	}
}

// Successes must pull the score back up: recovery has to be observable or the
// channel could never return to service.
func TestChannelHealthRecoversAfterSuccesses(t *testing.T) {
	enableChannelHealthForTest(t, 60, 0)

	base := time.Now()
	for i := 0; i < 10; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 8, Route: "r", Success: false,
			Observed: base.Add(time.Duration(i) * time.Second),
		})
	}
	degraded, _ := GetChannelHealthSnapshot(8, "r", "")

	for i := 10; i < 40; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 8, Route: "r", Success: true,
			Observed: base.Add(time.Duration(i) * time.Second),
		})
	}
	recovered, _ := GetChannelHealthSnapshot(8, "r", "")

	if recovered.Availability <= degraded.Availability {
		t.Fatalf("expected recovery, degraded=%v recovered=%v",
			degraded.Availability, recovered.Availability)
	}
}

// The confidence blend is what stops a single sample from being treated as
// authoritative.
func TestChannelHealthLowSampleCountStaysNearPrior(t *testing.T) {
	enableChannelHealthForTest(t, 300, 10)

	RecordChannelHealthSample(ChannelHealthSample{ChannelID: 9, Route: "r", Success: false})

	snapshot, ok := GetChannelHealthSnapshot(9, "r", "")
	if !ok {
		t.Fatal("expected snapshot")
	}
	if snapshot.Confident {
		t.Fatal("expected snapshot to be flagged as low confidence")
	}
	if snapshot.Score <= snapshot.RawScore {
		t.Fatalf("expected blended score above raw score, score=%v raw=%v",
			snapshot.Score, snapshot.RawScore)
	}
	if snapshot.Score < 0.8 {
		t.Fatalf("one failure should not collapse the score, got %v", snapshot.Score)
	}
}

// Enough consistent samples must eventually overcome the prior, otherwise a
// genuinely broken channel would stay artificially healthy.
func TestChannelHealthBecomesConfidentWithSamples(t *testing.T) {
	enableChannelHealthForTest(t, 3600, 5)

	base := time.Now()
	for i := 0; i < 20; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 10, Route: "r", Success: false,
			Observed: base.Add(time.Duration(i) * time.Millisecond),
		})
	}

	snapshot, _ := GetChannelHealthSnapshot(10, "r", "")
	if !snapshot.Confident {
		t.Fatalf("expected confidence after 20 samples, samples=%v", snapshot.Samples)
	}
	if snapshot.Score > 0.2 {
		t.Fatalf("expected low score for consistently failing channel, got %v", snapshot.Score)
	}
}

// Idle channels must drift back toward neutral instead of holding a stale
// verdict forever.
func TestChannelHealthDecaysTowardNeutralWhenIdle(t *testing.T) {
	enableChannelHealthForTest(t, 60, 0)

	past := time.Now().Add(-30 * time.Minute)
	for i := 0; i < 20; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 11, Route: "r", Success: false,
			Observed: past.Add(time.Duration(i) * time.Second),
		})
	}

	snapshot, ok := GetChannelHealthSnapshot(11, "r", "")
	if !ok {
		t.Fatal("expected snapshot")
	}
	if snapshot.Availability < 0.95 {
		t.Fatalf("expected idle decay back toward neutral, got %v", snapshot.Availability)
	}
}

// Samples arriving out of order (clock skew between nodes) must not corrupt
// the state or produce a negative elapsed time.
func TestChannelHealthHandlesOutOfOrderSamples(t *testing.T) {
	enableChannelHealthForTest(t, 60, 0)

	now := time.Now()
	RecordChannelHealthSample(ChannelHealthSample{ChannelID: 12, Route: "r", Success: true, Observed: now})
	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: 12, Route: "r", Success: false,
		Observed: now.Add(-10 * time.Minute),
	})

	snapshot, ok := GetChannelHealthSnapshot(12, "r", "")
	if !ok {
		t.Fatal("expected snapshot")
	}
	if math.IsNaN(snapshot.Score) || snapshot.Score < 0 || snapshot.Score > 1 {
		t.Fatalf("score out of range after out-of-order sample: %v", snapshot.Score)
	}
}

// Health is tracked per channel/route pair; one route failing must not taint
// the same channel on another route.
func TestChannelHealthIsolatesRoutes(t *testing.T) {
	enableChannelHealthForTest(t, 3600, 0)

	base := time.Now()
	for i := 0; i < 10; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 13, Route: "/v1/messages", Success: false,
			Observed: base.Add(time.Duration(i) * time.Millisecond),
		})
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 13, Route: "/v1/chat", Success: true,
			Observed: base.Add(time.Duration(i) * time.Millisecond),
		})
	}

	failing, _ := GetChannelHealthSnapshot(13, "/v1/messages", "")
	healthy, _ := GetChannelHealthSnapshot(13, "/v1/chat", "")
	if failing.Availability >= healthy.Availability {
		t.Fatalf("expected route isolation, failing=%v healthy=%v",
			failing.Availability, healthy.Availability)
	}
}

// A latency spike well above the channel's own baseline should reduce the
// score even while every request still succeeds.
func TestChannelHealthLatencySpikeReducesScore(t *testing.T) {
	enableChannelHealthForTest(t, 3600, 0)

	base := time.Now()
	for i := 0; i < 30; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: 14, Route: "r", Success: true,
			Latency:  100 * time.Millisecond,
			Observed: base.Add(time.Duration(i) * time.Second),
		})
	}
	stable, _ := GetChannelHealthSnapshot(14, "r", "")

	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: 14, Route: "r", Success: true,
		Latency:  5 * time.Second,
		Observed: base.Add(31 * time.Second),
	})
	spiked, _ := GetChannelHealthSnapshot(14, "r", "")

	if spiked.Score >= stable.Score {
		t.Fatalf("expected latency spike to lower score, stable=%v spiked=%v",
			stable.Score, spiked.Score)
	}
	if spiked.Score < latencyScoreFloor-0.01 {
		t.Fatalf("latency penalty must stay bounded, got %v", spiked.Score)
	}
}

// Changing tuning parameters must discard state accumulated under the old
// configuration rather than reinterpreting it with new weights.
func TestChannelHealthConfigChangeResetsState(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)

	RecordChannelHealthSample(ChannelHealthSample{ChannelID: 15, Route: "r", Success: false})
	if _, ok := GetChannelHealthSnapshot(15, "r", ""); !ok {
		t.Fatal("expected snapshot before config change")
	}

	common.SetChannelHealthHalfLifeSeconds(120)

	if _, ok := GetChannelHealthSnapshot(15, "r", ""); ok {
		t.Fatal("expected state reset after configuration change")
	}
}

// Entries idle beyond the TTL must be evicted so removed channels cannot leak.
func TestChannelHealthListEvictsExpiredEntries(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)
	common.SetChannelHealthStateTTLSeconds(60)

	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: 16, Route: "r", Success: true,
		Observed: time.Now().Add(-2 * time.Hour),
	})
	RecordChannelHealthSample(ChannelHealthSample{ChannelID: 17, Route: "r", Success: true})

	snapshots := ListChannelHealthSnapshots()
	for _, snapshot := range snapshots {
		if snapshot.ChannelID == 16 {
			t.Fatal("expected expired entry to be evicted")
		}
	}
	if _, ok := GetChannelHealthSnapshot(17, "r", ""); !ok {
		t.Fatal("expected live entry to survive eviction")
	}
}

// Observe mode is the safety property that makes phase 1 deployable: recording
// happens, but nothing is allowed to consume it for routing.
func TestChannelHealthObserveModeIsNotActive(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)
	common.SetChannelHealthMode(common.ChannelHealthModeObserve)

	if common.IsChannelHealthActive() {
		t.Fatal("observe mode must not report active")
	}
	if !common.IsChannelHealthEnabled() {
		t.Fatal("observe mode must still record samples")
	}

	common.SetChannelHealthMode(common.ChannelHealthModeActive)
	if !common.IsChannelHealthActive() {
		t.Fatal("active mode should report active")
	}

	// An unrecognised mode must fall back to observe rather than defaulting to
	// steering traffic.
	common.SetChannelHealthMode("bogus")
	if common.IsChannelHealthActive() {
		t.Fatal("unknown mode must fall back to observe")
	}

	t.Cleanup(func() { common.SetChannelHealthMode(common.ChannelHealthModeObserve) })
}

// The master switch must dominate the mode: active mode with the subsystem
// disabled still has to report inactive.
func TestChannelHealthDisabledOverridesActiveMode(t *testing.T) {
	common.SetChannelHealthEnabled(false)
	common.SetChannelHealthMode(common.ChannelHealthModeActive)
	t.Cleanup(func() { common.SetChannelHealthMode(common.ChannelHealthModeObserve) })

	if common.IsChannelHealthActive() {
		t.Fatal("disabled subsystem must never report active")
	}
}

func TestChannelHealthRejectsOutOfRangeConfig(t *testing.T) {
	common.SetChannelHealthHalfLifeSeconds(common.DefaultChannelHealthHalfLifeSeconds)
	original := common.ChannelHealthHalfLifeSeconds()

	common.SetChannelHealthHalfLifeSeconds(0)
	common.SetChannelHealthHalfLifeSeconds(-5)
	common.SetChannelHealthHalfLifeSeconds(999999)

	if common.ChannelHealthHalfLifeSeconds() != original {
		t.Fatalf("out-of-range half-life must be ignored, got %d",
			common.ChannelHealthHalfLifeSeconds())
	}
}

func TestChannelHealthConcurrentRecording(t *testing.T) {
	enableChannelHealthForTest(t, 300, 0)

	done := make(chan struct{})
	for worker := 0; worker < 8; worker++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			for i := 0; i < 200; i++ {
				RecordChannelHealthSample(ChannelHealthSample{
					ChannelID: 20 + id%3,
					Route:     "r",
					Success:   i%2 == 0,
					Latency:   time.Duration(i) * time.Millisecond,
				})
				GetChannelHealthScore(20+id%3, "r", "")
			}
		}(worker)
	}
	for worker := 0; worker < 8; worker++ {
		<-done
	}

	for channelID := 20; channelID < 23; channelID++ {
		snapshot, ok := GetChannelHealthSnapshot(channelID, "r", "")
		if !ok {
			t.Fatalf("expected snapshot for channel %d", channelID)
		}
		if math.IsNaN(snapshot.Score) || snapshot.Score < 0 || snapshot.Score > 1 {
			t.Fatalf("score out of range under concurrency: %v", snapshot.Score)
		}
	}
}

// decayFactor is the core of the time-weighting: one half-life must retain
// exactly half the previous weight.
func TestDecayFactorHalfLife(t *testing.T) {
	if got := decayFactor(60*time.Second, 60); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("expected 0.5 retention after one half-life, got %v", got)
	}
	if got := decayFactor(0, 60); got != 1 {
		t.Fatalf("expected full retention for zero elapsed, got %v", got)
	}
	if got := decayFactor(120*time.Second, 60); math.Abs(got-0.25) > 1e-9 {
		t.Fatalf("expected 0.25 retention after two half-lives, got %v", got)
	}
}

// Family isolation is the guarantee that makes cross-channel comparison fair:
// a claude channel failing on claude-sonnet-4-6 must not drag down the same
// channel's gpt score, and vice versa. Without bucketing per family, one
// family's outage would silently reprice the other family's routing.
func TestChannelHealthFamiliesAreIsolated(t *testing.T) {
	enableChannelHealthForTest(t, 300, 1)

	const channelID = 9100
	const route = "/v1/chat/completions"

	// Fail the claude family repeatedly.
	for i := 0; i < 10; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: channelID,
			Route:     route,
			ModelName: "claude-sonnet-4-6",
			Success:   false,
		})
	}
	// Succeed on the gpt family for the same channel and route.
	for i := 0; i < 10; i++ {
		RecordChannelHealthSample(ChannelHealthSample{
			ChannelID: channelID,
			Route:     route,
			ModelName: "gpt-5.4-mini",
			Success:   true,
		})
	}

	claudeSnapshot, ok := GetChannelHealthSnapshot(channelID, route, "claude-sonnet-4-6")
	if !ok {
		t.Fatal("expected a claude snapshot")
	}
	gptSnapshot, ok := GetChannelHealthSnapshot(channelID, route, "gpt-5.4-mini")
	if !ok {
		t.Fatal("expected a gpt snapshot")
	}

	if claudeSnapshot.Family != "claude" {
		t.Fatalf("expected claude family bucket, got %q", claudeSnapshot.Family)
	}
	if gptSnapshot.Family != "gpt" {
		t.Fatalf("expected gpt family bucket, got %q", gptSnapshot.Family)
	}
	if claudeSnapshot.Score >= gptSnapshot.Score {
		t.Fatalf("claude failures must not be averaged with gpt successes: claude=%v gpt=%v",
			claudeSnapshot.Score, gptSnapshot.Score)
	}
	if gptSnapshot.Failures != 0 {
		t.Fatalf("gpt bucket must not inherit claude failures, got %d", gptSnapshot.Failures)
	}
	if claudeSnapshot.Successes != 0 {
		t.Fatalf("claude bucket must not inherit gpt successes, got %d", claudeSnapshot.Successes)
	}
}

// Variants within one family must share a bucket: a dated or suffixed model is
// still the same upstream family, so splitting them would fragment the sample
// count and keep every bucket below the confidence threshold.
func TestChannelHealthFamilyVariantsShareBucket(t *testing.T) {
	enableChannelHealthForTest(t, 300, 1)

	const channelID = 9101
	const route = "/v1/messages"

	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: channelID, Route: route, ModelName: "claude-sonnet-4-6", Success: false,
	})
	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: channelID, Route: route, ModelName: "claude-opus-4-8", Success: false,
	})

	snapshot, ok := GetChannelHealthSnapshot(channelID, route, "claude-haiku-4-5-20251001")
	if !ok {
		t.Fatal("expected variants to resolve to the same claude bucket")
	}
	if snapshot.Failures != 2 {
		t.Fatalf("expected both claude variants in one bucket, got %d failures", snapshot.Failures)
	}
}
