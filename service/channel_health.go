package service

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// Channel health scoring keeps a time-decayed view of how well a channel is
// serving one route. It exists because the legacy availability input,
// Channel.PreviousDayProbeSuccessRate, is derived from the previous calendar
// day and therefore cannot react to a channel failing right now.
//
// Design notes:
//
//   - Decay is time-based, not sample-based. A channel that stops receiving
//     traffic must drift back toward the neutral prior instead of freezing at
//     whatever value it last had.
//   - Confidence is tracked separately from the score. A channel with two
//
// samples must not outrank a channel with two hundred just because both
//
//	  happen to be at 100%, so the exported score is blended with the prior
//	  until minSamples is reached.
//	- Latency is scored against the channel's own decayed baseline rather than
//	  a global constant, because a slow-but-stable provider is not unhealthy.
//	- State is held in memory and read off the hot path. Cross-node
//	  aggregation is intentionally left to a later phase so that a Redis
//	  outage can never block channel selection.
const (
	neutralHealthScore = 1.0
	minHealthScore     = 0.0
	maxHealthScore     = 1.0

	// latencyScoreFloor stops a single very slow response from driving the
	// combined score to zero; availability remains the dominant signal.
	latencyScoreFloor = 0.2

	// latencySensitivity dampens the latency ratio so ordinary jitter does not
	// look like degradation.
	latencySensitivity = 0.5

	// minSampleWeight is the floor on how much a single observation counts,
	// regardless of how little time has passed since the previous one.
	//
	// Pure time-based decay is too slow on sparse traffic: with a 300s
	// half-life, ten failures one second apart would only shift the average by
	// ~2%, so an outage would take minutes to register. Giving every sample a
	// minimum weight (here 1/8, i.e. an effective window of ~8 samples) keeps
	// the score responsive to consecutive failures while the time component
	// still handles idle decay.
	minSampleWeight = 0.125

	// sampleCountHalfLifeFactor stretches the half-life used to age the effective
	// sample count relative to the one used for the availability signal.
	//
	// Evidence and verdict decay at different rates. "Is this channel healthy
	// right now" must react within one half-life, but "have I observed it enough
	// to trust that verdict" should survive a few quiet windows -- otherwise
	// confidence collapses on any channel whose traffic is sparse, and every
	// score is permanently blended back toward the neutral prior.
	//
	// This is not hypothetical. The probe scheduler suppresses probing for
	// probe_idle_grace_seconds after each observation, so on a channel with no
	// real traffic the effective sampling interval is the idle grace, not the
	// probe interval. With both grace and half-life at their defaults of 300s
	// the retention per sample was exactly 0.5, capping the accumulated count at
	// 1/(1-0.5) = 2 -- unreachably below a min_samples of 5, so `confident` could
	// never become true no matter how long the system ran.
	sampleCountHalfLifeFactor = 4
)

// sampleCountRetention returns the retention multiplier for the effective sample
// counter. Unlike availability, the counter must not be sensitive to how far
// apart two observations happened to fall: a channel sampled every 300s has not
// seen less evidence than one sampled every 60s, it has just seen it slower. The
// floor therefore applies unconditionally, giving a stable effective window of
// 1/minSampleWeight observations, while snapshotLocked applies the time-based
// ageing separately.
func sampleCountRetention() float64 {
	return 1 - minSampleWeight
}

// sampleRetention returns the EWMA retention multiplier for one observation.
// It takes the stronger of time-based decay and the per-sample floor so that
// both "many events in a short burst" and "few events over a long gap" move
// the score meaningfully.
func sampleRetention(elapsed time.Duration, halfLifeSeconds int) float64 {
	timeRetain := decayFactor(elapsed, halfLifeSeconds)
	if timeRetain > 1-minSampleWeight {
		return 1 - minSampleWeight
	}
	return timeRetain
}

// ChannelHealthSample is one observation of a channel serving a route.
type ChannelHealthSample struct {
	ChannelID int
	Route     string
	// ModelName is the model that served this observation. It is used only to
	// derive the family bucket, so callers pass the request's model and do not
	// need to know how families are defined.
	ModelName string
	Success   bool
	Latency   time.Duration
	// Observed marks when the sample happened. Zero means "now"; it is
	// injectable so tests do not depend on wall-clock sleeps.
	Observed time.Time
}

// ChannelHealthSnapshot is the exported, read-only view of one channel/route
// pair. Score is the blended value routing would consume; RawScore is the
// unblended EWMA, kept separate so observe-mode dashboards can show how much
// of the score is still prior.
type ChannelHealthSnapshot struct {
	ChannelID       int       `json:"channel_id"`
	Route           string    `json:"route"`
	Family          string    `json:"family"`
	Score           float64   `json:"score"`
	RawScore        float64   `json:"raw_score"`
	Availability    float64   `json:"availability"`
	LatencyScore    float64   `json:"latency_score"`
	LatencyBaseline float64   `json:"latency_baseline_ms"`
	LastLatencyMs   float64   `json:"last_latency_ms"`
	Samples         float64   `json:"samples"`
	Successes       int64     `json:"successes"`
	Failures        int64     `json:"failures"`
	Confident       bool      `json:"confident"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type channelHealthState struct {
	availability    float64
	latencyBaseline float64
	lastLatencyMs   float64
	// samples is itself decayed, so it represents "effective recent sample
	// count" rather than a lifetime total.
	samples   float64
	successes int64
	failures  int64
	updatedAt time.Time
}

type channelHealthKey struct {
	channelID int
	route     string
	// family buckets scores by model family (see common.ChannelModelFamily).
	// Without it, a claude channel probed with claude-sonnet-4-6 and a gpt
	// channel probed with gpt-5.4-mini would be compared on the same axis even
	// though their latency and error profiles are not comparable. Bucketing per
	// family is what makes "claude competes with claude" true rather than
	// aspirational.
	family string
}

var channelHealth = struct {
	sync.RWMutex
	states  map[channelHealthKey]*channelHealthState
	version uint64
	inited  bool
}{states: make(map[channelHealthKey]*channelHealthState)}

// decayFactor converts an elapsed duration into an EWMA retention multiplier
// using the configured half-life: after halfLife seconds, weight is 0.5.
func decayFactor(elapsed time.Duration, halfLifeSeconds int) float64 {
	if halfLifeSeconds <= 0 || elapsed <= 0 {
		return 1
	}
	return math.Exp2(-elapsed.Seconds() / float64(halfLifeSeconds))
}

// syncChannelHealthConfig drops all state when the operator changes tuning
// parameters, mirroring how the circuit breaker handles reconfiguration. Old
// samples were accumulated under different decay settings and would otherwise
// linger with the wrong weight.
func syncChannelHealthConfig() bool {
	enabled := common.IsChannelHealthEnabled()
	version := common.ChannelHealthConfigVersion()

	channelHealth.Lock()
	changed := channelHealth.inited && channelHealth.version != version
	channelHealth.inited = true
	channelHealth.version = version
	if changed {
		channelHealth.states = make(map[channelHealthKey]*channelHealthState)
	}
	channelHealth.Unlock()
	return enabled
}

// RecordChannelHealthSample folds one observation into the channel/route EWMA.
// It is a no-op while the subsystem is disabled, and it never influences
// routing on its own: consumers decide whether to read the score.
func RecordChannelHealthSample(sample ChannelHealthSample) {
	if !syncChannelHealthConfig() {
		return
	}
	if sample.ChannelID <= 0 {
		return
	}
	now := sample.Observed
	if now.IsZero() {
		now = time.Now()
	}

	halfLife := common.ChannelHealthHalfLifeSeconds()
	latencyHalfLife := common.ChannelHealthLatencyHalfLifeSeconds()
	key := channelHealthKey{
		channelID: sample.ChannelID,
		route:     sample.Route,
		family:    common.ChannelModelFamily(sample.ModelName),
	}

	channelHealth.Lock()
	defer channelHealth.Unlock()

	state := channelHealth.states[key]
	if state == nil {
		state = &channelHealthState{
			availability: neutralHealthScore,
			updatedAt:    now,
		}
		channelHealth.states[key] = state
	}

	// Guard against clock skew between nodes: a sample older than the stored
	// state must not resurrect stale weight.
	elapsed := now.Sub(state.updatedAt)
	if elapsed < 0 {
		elapsed = 0
	}

	retain := sampleRetention(elapsed, halfLife)
	observation := 0.0
	if sample.Success {
		observation = 1.0
	}

	// Time-decayed EWMA with a per-sample weight floor: a long gap makes one
	// sample dominate, and a rapid burst still moves the average.
	state.availability = state.availability*retain + observation*(1-retain)
	// The sample counter uses its own retention so a slow sampling cadence does
	// not permanently cap it below min_samples (see sampleCountRetention).
	state.samples = state.samples*sampleCountRetention() + 1

	if sample.Success {
		state.successes++
	} else {
		state.failures++
	}

	latencyMs := float64(sample.Latency.Milliseconds())
	if latencyMs > 0 {
		state.lastLatencyMs = latencyMs
		if state.latencyBaseline <= 0 {
			state.latencyBaseline = latencyMs
		} else {
			latencyRetain := decayFactor(elapsed, latencyHalfLife)
			state.latencyBaseline = state.latencyBaseline*latencyRetain + latencyMs*(1-latencyRetain)
		}
	}

	state.updatedAt = now
}

// latencyScore rates the most recent latency against the channel's own decayed
// baseline. Faster than baseline is capped at 1: being quick does not earn a
// bonus, it just avoids a penalty.
func latencyScore(state *channelHealthState) float64 {
	if state.latencyBaseline <= 0 || state.lastLatencyMs <= 0 {
		return neutralHealthScore
	}
	ratio := state.latencyBaseline / state.lastLatencyMs
	if ratio >= 1 {
		return neutralHealthScore
	}
	// Soften the penalty so routine jitter is not treated as a fault.
	score := 1 - (1-ratio)*latencySensitivity
	if score < latencyScoreFloor {
		return latencyScoreFloor
	}
	return score
}

// snapshotLocked projects the stored state to "now", applying decay that has
// accrued since the last sample. Callers must hold at least a read lock.
func snapshotLocked(key channelHealthKey, state *channelHealthState, now time.Time,
	halfLife, minSamples int) ChannelHealthSnapshot {

	elapsed := now.Sub(state.updatedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	retain := decayFactor(elapsed, halfLife)

	// Idle channels drift back toward neutral instead of holding a stale
	// verdict; effective sample count decays with them, but more slowly, so a
	// channel does not lose its accumulated confidence during one quiet window.
	availability := state.availability*retain + neutralHealthScore*(1-retain)
	samples := state.samples * decayFactor(elapsed, halfLife*sampleCountHalfLifeFactor)

	raw := clampHealthScore(availability * latencyScore(state))

	// Confidence blending: with few samples the score stays close to the
	// neutral prior, which keeps a brand-new channel from being either
	// blindly trusted or unfairly condemned.
	score := raw
	confident := true
	if minSamples > 0 && samples < float64(minSamples) {
		confident = false
		weight := samples / float64(minSamples)
		score = raw*weight + neutralHealthScore*(1-weight)
	}

	return ChannelHealthSnapshot{
		ChannelID:       key.channelID,
		Route:           key.route,
		Family:          key.family,
		Score:           clampHealthScore(score),
		RawScore:        raw,
		Availability:    clampHealthScore(availability),
		LatencyScore:    latencyScore(state),
		LatencyBaseline: state.latencyBaseline,
		LastLatencyMs:   state.lastLatencyMs,
		Samples:         samples,
		Successes:       state.successes,
		Failures:        state.failures,
		Confident:       confident,
		UpdatedAt:       state.updatedAt,
	}
}

func clampHealthScore(score float64) float64 {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return neutralHealthScore
	}
	if score < minHealthScore {
		return minHealthScore
	}
	if score > maxHealthScore {
		return maxHealthScore
	}
	return score
}

// GetChannelHealthScore returns the blended score for one channel/route/model
// triple. Unknown channels and a disabled subsystem both yield the neutral
// score, so callers can use the result unconditionally without special-casing.
func GetChannelHealthScore(channelID int, route string, modelName string) float64 {
	snapshot, ok := GetChannelHealthSnapshot(channelID, route, modelName)
	if !ok {
		return neutralHealthScore
	}
	return snapshot.Score
}

func GetChannelHealthSnapshot(channelID int, route string, modelName string) (ChannelHealthSnapshot, bool) {
	if !syncChannelHealthConfig() || channelID <= 0 {
		return ChannelHealthSnapshot{}, false
	}
	key := channelHealthKey{
		channelID: channelID,
		route:     route,
		family:    common.ChannelModelFamily(modelName),
	}
	now := time.Now()
	halfLife := common.ChannelHealthHalfLifeSeconds()
	minSamples := common.ChannelHealthMinSamples()

	channelHealth.RLock()
	state := channelHealth.states[key]
	if state == nil {
		channelHealth.RUnlock()
		return ChannelHealthSnapshot{}, false
	}
	snapshot := snapshotLocked(key, state, now, halfLife, minSamples)
	channelHealth.RUnlock()
	return snapshot, true
}

// ListChannelHealthSnapshots returns every tracked pair, newest activity first.
// It also evicts entries that have been idle past the configured TTL so an
// removed channel does not leak memory forever.
func ListChannelHealthSnapshots() []ChannelHealthSnapshot {
	if !syncChannelHealthConfig() {
		return nil
	}
	now := time.Now()
	halfLife := common.ChannelHealthHalfLifeSeconds()
	minSamples := common.ChannelHealthMinSamples()
	ttl := time.Duration(common.ChannelHealthStateTTLSeconds()) * time.Second

	channelHealth.Lock()
	snapshots := make([]ChannelHealthSnapshot, 0, len(channelHealth.states))
	for key, state := range channelHealth.states {
		if ttl > 0 && now.Sub(state.updatedAt) > ttl {
			delete(channelHealth.states, key)
			continue
		}
		snapshots = append(snapshots, snapshotLocked(key, state, now, halfLife, minSamples))
	}
	channelHealth.Unlock()

	sort.Slice(snapshots, func(i, j int) bool {
		if !snapshots[i].UpdatedAt.Equal(snapshots[j].UpdatedAt) {
			return snapshots[i].UpdatedAt.After(snapshots[j].UpdatedAt)
		}
		if snapshots[i].ChannelID != snapshots[j].ChannelID {
			return snapshots[i].ChannelID < snapshots[j].ChannelID
		}
		if snapshots[i].Route != snapshots[j].Route {
			return snapshots[i].Route < snapshots[j].Route
		}
		return snapshots[i].Family < snapshots[j].Family
	})
	return snapshots
}

// ResetChannelHealthState clears all tracked state. Used by tests and by the
// admin surface when an operator wants to discard a skewed history.
func ResetChannelHealthState() {
	channelHealth.Lock()
	channelHealth.states = make(map[channelHealthKey]*channelHealthState)
	channelHealth.Unlock()
}

// ChannelLastObservedAt reports when the channel was last observed on any
// route, plus whether any observation exists at all. The probe scheduler uses
// this to skip channels that real traffic is already exercising, which is both
// cheaper and more faithful than a synthetic call.
func ChannelLastObservedAt(channelID int) (time.Time, bool) {
	if channelID <= 0 {
		return time.Time{}, false
	}
	channelHealth.RLock()
	defer channelHealth.RUnlock()
	var latest time.Time
	for key, state := range channelHealth.states {
		if key.channelID != channelID {
			continue
		}
		if state.updatedAt.After(latest) {
			latest = state.updatedAt
		}
	}
	return latest, !latest.IsZero()
}
