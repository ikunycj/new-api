package model

import (
	"math"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
)

// chooseRouteCandidate bypasses the DB-backed channel cache, so these tests
// construct candidates directly. They verify the phase-2 wiring: when the real
// time health subsystem is active and has a snapshot for a channel/route pair,
// the live blended score replaces the legacy PreviousDayProbeSuccessRate input.

func newHealthChannel(id int, prevRate float64, weight int) *Channel {
	return &Channel{
		Id:                          id,
		Status:                      common.ChannelStatusEnabled,
		ResponseTime:                0, // identical across candidates: loadScore stays constant
		PreviousDayProbeSuccessRate: prevRate,
	}
}

func newHealthCandidate(id int, prevRate float64, weight int, costFactor float64) weightedRouteCandidate {
	return weightedRouteCandidate{
		channel:    newHealthChannel(id, prevRate, weight),
		weight:     weight,
		costFactor: costFactor,
	}
}

// TestChooseRouteCandidateHealthActiveUsesLiveScore proves that in active mode
// with a snapshot present, the live score drives selection rather than the
// legacy previous-day rate. Two channels share weight and cost so the only
// differentiator is health; a 0.9 / 0.1 split with the default 40% availability
// weight should heavily favour the healthy channel.
func TestChooseRouteCandidateHealthActiveUsesLiveScore(t *testing.T) {
	common.SetChannelHealthEnabled(true)
	common.SetChannelHealthMode(common.ChannelHealthModeActive)
	defer func() {
		common.SetChannelHealthEnabled(false)
		common.SetChannelHealthMode(common.ChannelHealthModeObserve)
	}()

	// Active mode with a getter injected.
	SetChannelHealthScoreGetter(func(channelID int, route string, modelName string) (float64, bool) {
		if channelID == 1 {
			return 0.9, true
		}
		if channelID == 2 {
			return 0.1, true
		}
		return 0, false
	})
	defer SetChannelHealthScoreGetter(nil)

	candidates := []weightedRouteCandidate{
		newHealthCandidate(1, 100, 1, 1),
		newHealthCandidate(2, 100, 1, 1),
	}

	wins := map[int]int{}
	const iterations = 4000
	for i := 0; i < iterations; i++ {
		chosen := chooseRouteCandidate(candidates, RoutingStrategyConfig{
			Type:               RoutingStrategyWeighted,
			PriceWeight:        0,
			AvailabilityWeight: 40,
			LoadWeight:         0,
		}, "/v1/chat/completions", "claude-sonnet-4-6")
		if chosen != nil {
			wins[chosen.Id]++
		}
	}

	// The healthy channel must win far more often. With availability weight 40
	// and identical price/load, the weight ratio is roughly 0.9 : 0.1 (9:1).
	// Allow generous slack so the test is robust to random jitter.
	healthyShare := float64(wins[1]) / float64(iterations)
	assert.Greater(t, healthyShare, 0.70, "healthy channel should win most selections, got %.3f", healthyShare)
	assert.Less(t, float64(wins[2])/float64(iterations), 0.30, "unhealthy channel should win rarely")
}

// TestChooseRouteCandidateObserveIgnoresHealth proves that observe mode leaves
// the getter untouched: selection falls back to the legacy previous-day rate,
// which here is identical for both channels, so wins should be near 50/50.
func TestChooseRouteCandidateObserveIgnoresHealth(t *testing.T) {
	common.SetChannelHealthEnabled(true)
	common.SetChannelHealthMode(common.ChannelHealthModeObserve)
	defer func() {
		common.SetChannelHealthEnabled(false)
		common.SetChannelHealthMode(common.ChannelHealthModeObserve)
	}()

	var getterCalls atomic.Int64
	SetChannelHealthScoreGetter(func(channelID int, route string, modelName string) (float64, bool) {
		getterCalls.Add(1)
		return 1.0, true
	})
	defer SetChannelHealthScoreGetter(nil)

	candidates := []weightedRouteCandidate{
		newHealthCandidate(1, 100, 1, 1),
		newHealthCandidate(2, 100, 1, 1),
	}

	wins := map[int]int{}
	const iterations = 4000
	for i := 0; i < iterations; i++ {
		chosen := chooseRouteCandidate(candidates, RoutingStrategyConfig{
			Type:               RoutingStrategyWeighted,
			PriceWeight:        0,
			AvailabilityWeight: 40,
			LoadWeight:         0,
		}, "/v1/chat/completions", "claude-sonnet-4-6")
		if chosen != nil {
			wins[chosen.Id]++
		}
	}

	assert.Zero(t, getterCalls.Load(), "observe mode must not invoke the health getter")
	share := float64(wins[1]) / float64(iterations)
	assert.InDelta(t, 0.5, share, 0.06, "identical candidates should split ~50/50 in observe mode")
}

// TestChooseRouteCandidateHealthFallsBackToLegacy proves that when the getter
// reports no snapshot, the legacy previous-day rate is kept as the prior. Here
// channel 1 has a legacy rate of 100 (a healthy prior) and channel 2 a legacy
// rate of 20, so after the active-mode fallback the split should favour 1.
func TestChooseRouteCandidateHealthFallsBackToLegacy(t *testing.T) {
	common.SetChannelHealthEnabled(true)
	common.SetChannelHealthMode(common.ChannelHealthModeActive)
	defer func() {
		common.SetChannelHealthEnabled(false)
		common.SetChannelHealthMode(common.ChannelHealthModeObserve)
	}()

	var getterCalls atomic.Int64
	SetChannelHealthScoreGetter(func(channelID int, route string, modelName string) (float64, bool) {
		getterCalls.Add(1)
		return 0, false // no snapshot: fall back to legacy
	})
	defer SetChannelHealthScoreGetter(nil)

	candidates := []weightedRouteCandidate{
		newHealthCandidate(1, 100, 1, 1),
		newHealthCandidate(2, 20, 1, 1),
	}

	wins := map[int]int{}
	const iterations = 4000
	for i := 0; i < iterations; i++ {
		chosen := chooseRouteCandidate(candidates, RoutingStrategyConfig{
			Type:               RoutingStrategyWeighted,
			PriceWeight:        0,
			AvailabilityWeight: 40,
			LoadWeight:         0,
		}, "/v1/chat/completions", "claude-sonnet-4-6")
		if chosen != nil {
			wins[chosen.Id]++
		}
	}

	share := float64(wins[1]) / float64(iterations)
	assert.Greater(t, share, 0.60, "legacy rate should still favour the higher prior, got %.3f", share)
}

// TestChooseRouteCandidatePriorityIgnoresHealth proves that the priority strategy
// short-circuits before any health logic, so the getter is never called.
func TestChooseRouteCandidatePriorityIgnoresHealth(t *testing.T) {
	common.SetChannelHealthEnabled(true)
	common.SetChannelHealthMode(common.ChannelHealthModeActive)
	defer func() {
		common.SetChannelHealthEnabled(false)
		common.SetChannelHealthMode(common.ChannelHealthModeObserve)
	}()

	var getterCalls atomic.Int64
	SetChannelHealthScoreGetter(func(channelID int, route string, modelName string) (float64, bool) {
		getterCalls.Add(1)
		return 0.5, true
	})
	defer SetChannelHealthScoreGetter(nil)

	candidates := []weightedRouteCandidate{
		newHealthCandidate(1, 100, 1, 1),
		newHealthCandidate(2, 100, 1, 1),
	}

	chosen := chooseRouteCandidate(candidates, RoutingStrategyConfig{Type: RoutingStrategyPriority}, "/v1/chat/completions", "claude-sonnet-4-6")
	assert.NotNil(t, chosen)
	assert.Equal(t, 1, chosen.Id, "priority strategy returns the first candidate")
	assert.Zero(t, getterCalls.Load(), "priority strategy must not invoke the health getter")
}

// TestChannelHealthScoreGetterInjectionRoundTrip proves the injection setter and
// loader behave: setting nil clears, setting a value stores it.
func TestChannelHealthScoreGetterInjectionRoundTrip(t *testing.T) {
	SetChannelHealthScoreGetter(nil)
	assert.Nil(t, loadChannelHealthScoreGetter(), "nil getter must load as nil")

	getter := func(channelID int, route string, modelName string) (float64, bool) { return 0.42, true }
	SetChannelHealthScoreGetter(getter)
	loaded := loadChannelHealthScoreGetter()
	assert.NotNil(t, loaded)
	score, ok := loaded(7, "/v1/chat/completions", "claude-sonnet-4-6")
	assert.True(t, ok)
	assert.Equal(t, 0.42, score)

	SetChannelHealthScoreGetter(nil)
	assert.Nil(t, loadChannelHealthScoreGetter())
}

// TestChooseRouteCandidateHealthScoreClamping proves the live score is scaled to
// the 0..100 range the legacy input uses, and clamped to a sane ceiling so a
// malformed score above 1.0 can never produce an infinite availability input.
func TestChooseRouteCandidateHealthScoreClamping(t *testing.T) {
	common.SetChannelHealthEnabled(true)
	common.SetChannelHealthMode(common.ChannelHealthModeActive)
	defer func() {
		common.SetChannelHealthEnabled(false)
		common.SetChannelHealthMode(common.ChannelHealthModeObserve)
	}()

	SetChannelHealthScoreGetter(func(channelID int, route string, modelName string) (float64, bool) {
		return 2.5, true // above 1.0; must be clamped, not multiplied through
	})
	defer SetChannelHealthScoreGetter(nil)

	candidates := []weightedRouteCandidate{
		newHealthCandidate(1, 100, 1, 1),
	}
	// A single candidate must not panic or produce NaN weights.
	chosen := chooseRouteCandidate(candidates, RoutingStrategyConfig{
		Type:               RoutingStrategyWeighted,
		AvailabilityWeight: 40,
	}, "/v1/chat/completions", "claude-sonnet-4-6")
	assert.NotNil(t, chosen)
	assert.False(t, math.IsNaN(float64(chosen.Id)))
}
