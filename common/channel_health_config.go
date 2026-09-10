package common

import (
	"strings"
	"sync/atomic"
)

// Option keys persisted in the options table. Channel health scoring is opt-in
// so existing deployments keep the legacy routing behaviour after an upgrade.
const (
	ChannelHealthEnabledOptionKey         = "ChannelHealthEnabled"
	ChannelHealthModeOptionKey            = "ChannelHealthMode"
	ChannelHealthHalfLifeSecondsKey       = "ChannelHealthHalfLifeSeconds"
	ChannelHealthMinSamplesOptionKey      = "ChannelHealthMinSamples"
	ChannelHealthLatencyHalfLifeKey       = "ChannelHealthLatencyHalfLifeSeconds"
	ChannelHealthStateTTLSecondsOptionKey = "ChannelHealthStateTTLSeconds"
	ChannelHealthProbeEnabledOptionKey    = "ChannelHealthProbeEnabled"
	ChannelHealthProbeIntervalOptionKey   = "ChannelHealthProbeIntervalSeconds"
	ChannelHealthProbeIdleGraceOptionKey  = "ChannelHealthProbeIdleGraceSeconds"

	// History persistence is separate from scoring: it costs disk instead of
	// upstream quota, and an operator may reasonably want the live signal
	// without keeping a long trend.
	ChannelHealthHistoryEnabledOptionKey       = "ChannelHealthHistoryEnabled"
	ChannelHealthHistoryBucketSecondsOptionKey = "ChannelHealthHistoryBucketSeconds"
	ChannelHealthHistoryRetentionDaysOptionKey = "ChannelHealthHistoryRetentionDays"
)

// Scoring modes. Observe records and exports scores without ever influencing
// channel selection; active additionally allows routing to consume them.
const (
	ChannelHealthModeObserve = "observe"
	ChannelHealthModeActive  = "active"
)

// Defaults tuned for the current production traffic profile (~1.7 real samples
// per channel per minute): a 5 minute half-life keeps roughly 8 samples in the
// effective window, which is responsive without thrashing on sparse traffic.
const (
	DefaultChannelHealthHalfLifeSeconds        = 300
	DefaultChannelHealthMinSamples             = 5
	DefaultChannelHealthLatencyHalfLifeSeconds = 600
	DefaultChannelHealthStateTTLSeconds        = 3600

	// Probing only covers channels that real traffic has not exercised
	// recently, so the idle grace window is deliberately aligned with the
	// availability half-life: once a channel's samples have decayed by half it
	// is worth refreshing them synthetically.
	DefaultChannelHealthProbeIntervalSeconds  = 60
	DefaultChannelHealthProbeIdleGraceSeconds = 300

	// A 60s bucket matches the probe cadence, so a probe-only channel
	// contributes roughly one observation per bucket rather than being averaged
	// away. Seven days at that resolution is a few tens of MB for a deployment
	// of this size.
	DefaultChannelHealthHistoryBucketSeconds = 60
	DefaultChannelHealthHistoryRetentionDays = 7
)

const (
	minChannelHealthProbeIntervalSeconds  = 10
	maxChannelHealthProbeIntervalSeconds  = 86400
	minChannelHealthProbeIdleGraceSeconds = 0
	maxChannelHealthProbeIdleGraceSeconds = 86400
)

const (
	minChannelHealthHalfLifeSeconds = 5
	maxChannelHealthHalfLifeSeconds = 86400
	maxChannelHealthMinSamples      = 1000
	minChannelHealthStateTTLSeconds = 60
	maxChannelHealthStateTTLSeconds = 604800

	// A bucket shorter than 10s would write more rows than the probe produces
	// observations; longer than an hour stops being a trend.
	minChannelHealthHistoryBucketSeconds = 10
	maxChannelHealthHistoryBucketSeconds = 3600
	// Retention 0 is rejected rather than treated as "keep forever": unbounded
	// growth on a table written every bucket is not a safe default.
	minChannelHealthHistoryRetentionDays = 1
	maxChannelHealthHistoryRetentionDays = 365
)

var (
	channelHealthEnabled            atomic.Bool
	channelHealthActive             atomic.Bool
	channelHealthHalfLifeSeconds    atomic.Int64
	channelHealthMinSamples         atomic.Int64
	channelHealthLatencyHalfLifeSec atomic.Int64
	channelHealthStateTTLSeconds    atomic.Int64
	channelHealthProbeEnabled       atomic.Bool
	channelHealthProbeInterval      atomic.Int64
	channelHealthProbeIdleGrace     atomic.Int64
	channelHealthHistoryEnabled     atomic.Bool
	channelHealthHistoryBucketSec   atomic.Int64
	channelHealthHistoryRetention   atomic.Int64
	channelHealthConfigVersion      atomic.Uint64
)

func init() {
	channelHealthHalfLifeSeconds.Store(DefaultChannelHealthHalfLifeSeconds)
	channelHealthMinSamples.Store(DefaultChannelHealthMinSamples)
	channelHealthLatencyHalfLifeSec.Store(DefaultChannelHealthLatencyHalfLifeSeconds)
	channelHealthStateTTLSeconds.Store(DefaultChannelHealthStateTTLSeconds)
	channelHealthProbeInterval.Store(DefaultChannelHealthProbeIntervalSeconds)
	channelHealthProbeIdleGrace.Store(DefaultChannelHealthProbeIdleGraceSeconds)
	channelHealthHistoryBucketSec.Store(DefaultChannelHealthHistoryBucketSeconds)
	channelHealthHistoryRetention.Store(DefaultChannelHealthHistoryRetentionDays)
}

// SetChannelHealthProbeEnabled toggles synthetic probing. It is separate from
// the scoring switch because probing is the only part of this subsystem that
// spends money on upstream calls, so it must be possible to run scoring on
// real traffic alone.
func SetChannelHealthProbeEnabled(enabled bool) {
	if channelHealthProbeEnabled.Swap(enabled) == enabled {
		return
	}
	channelHealthConfigVersion.Add(1)
}

func SetChannelHealthProbeIntervalSeconds(seconds int) {
	setBoundedChannelHealthInt(&channelHealthProbeInterval, seconds,
		minChannelHealthProbeIntervalSeconds, maxChannelHealthProbeIntervalSeconds)
}

// SetChannelHealthProbeIdleGraceSeconds controls how long a channel must go
// without real traffic before it becomes eligible for a synthetic probe. Zero
// probes every channel on every tick, which is intentionally allowed but
// costly.
func SetChannelHealthProbeIdleGraceSeconds(seconds int) {
	setBoundedChannelHealthInt(&channelHealthProbeIdleGrace, seconds,
		minChannelHealthProbeIdleGraceSeconds, maxChannelHealthProbeIdleGraceSeconds)
}

// IsChannelHealthProbeEnabled requires the master switch as well: probing
// without scoring would spend upstream quota for data nobody reads.
func IsChannelHealthProbeEnabled() bool {
	return channelHealthEnabled.Load() && channelHealthProbeEnabled.Load()
}

func ChannelHealthProbeIntervalSeconds() int {
	return int(channelHealthProbeInterval.Load())
}

func ChannelHealthProbeIdleGraceSeconds() int {
	return int(channelHealthProbeIdleGrace.Load())
}

// SetChannelHealthEnabled toggles the whole subsystem. Disabling it makes every
// recording call a no-op and restores the pre-existing routing behaviour.
func SetChannelHealthEnabled(enabled bool) {
	if channelHealthEnabled.Swap(enabled) == enabled {
		return
	}
	channelHealthConfigVersion.Add(1)
}

// SetChannelHealthMode stores the scoring mode. Any value other than "active"
// is treated as observe, so a malformed or unknown option can never silently
// start steering production traffic.
func SetChannelHealthMode(mode string) {
	active := strings.EqualFold(strings.TrimSpace(mode), ChannelHealthModeActive)
	if channelHealthActive.Swap(active) == active {
		return
	}
	channelHealthConfigVersion.Add(1)
}

// setBoundedChannelHealthInt ignores out-of-range values so a bad option cannot
// disable decay, remove the confidence floor, or pin state forever.
func setBoundedChannelHealthInt(target *atomic.Int64, value, min, max int) {
	if value < min || value > max {
		return
	}
	if target.Swap(int64(value)) == int64(value) {
		return
	}
	channelHealthConfigVersion.Add(1)
}

// SetChannelHealthHalfLifeSeconds controls how fast the availability EWMA
// forgets old samples.
func SetChannelHealthHalfLifeSeconds(seconds int) {
	setBoundedChannelHealthInt(&channelHealthHalfLifeSeconds, seconds,
		minChannelHealthHalfLifeSeconds, maxChannelHealthHalfLifeSeconds)
}

// SetChannelHealthMinSamples sets the confidence threshold below which a score
// stays blended with the neutral prior instead of trusting a tiny sample.
func SetChannelHealthMinSamples(samples int) {
	setBoundedChannelHealthInt(&channelHealthMinSamples, samples, 0, maxChannelHealthMinSamples)
}

// SetChannelHealthLatencyHalfLifeSeconds controls the latency baseline decay.
// The baseline intentionally decays slower than availability so a transient
// slowdown does not immediately become the new "normal".
func SetChannelHealthLatencyHalfLifeSeconds(seconds int) {
	setBoundedChannelHealthInt(&channelHealthLatencyHalfLifeSec, seconds,
		minChannelHealthHalfLifeSeconds, maxChannelHealthHalfLifeSeconds)
}

// SetChannelHealthStateTTLSeconds bounds how long an idle channel keeps its
// recorded state before it is discarded and treated as unknown again.
func SetChannelHealthStateTTLSeconds(seconds int) {
	setBoundedChannelHealthInt(&channelHealthStateTTLSeconds, seconds,
		minChannelHealthStateTTLSeconds, maxChannelHealthStateTTLSeconds)
}

func NotifyChannelHealthConfigChanged() {
	channelHealthConfigVersion.Add(1)
}

func IsChannelHealthEnabled() bool {
	return channelHealthEnabled.Load()
}

// IsChannelHealthActive reports whether scores may influence routing. Observe
// mode keeps recording while leaving channel selection untouched.
func IsChannelHealthActive() bool {
	return channelHealthEnabled.Load() && channelHealthActive.Load()
}

func ChannelHealthMode() string {
	if channelHealthActive.Load() {
		return ChannelHealthModeActive
	}
	return ChannelHealthModeObserve
}

func ChannelHealthHalfLifeSeconds() int {
	return int(channelHealthHalfLifeSeconds.Load())
}

func ChannelHealthMinSamples() int {
	return int(channelHealthMinSamples.Load())
}

func ChannelHealthLatencyHalfLifeSeconds() int {
	return int(channelHealthLatencyHalfLifeSec.Load())
}

func ChannelHealthStateTTLSeconds() int {
	return int(channelHealthStateTTLSeconds.Load())
}

func ChannelHealthConfigVersion() uint64 {
	return channelHealthConfigVersion.Load()
}

// SetChannelHealthHistoryEnabled toggles persistence of the score time series.
// It deliberately does not bump the config version: unlike the tuning knobs,
// turning history on or off does not invalidate accumulated in-memory state, and
// discarding live scores just to start writing rows would be a bad trade.
func SetChannelHealthHistoryEnabled(enabled bool) {
	channelHealthHistoryEnabled.Store(enabled)
}

// IsChannelHealthHistoryEnabled requires the master switch too: there is nothing
// to persist while scoring is off.
func IsChannelHealthHistoryEnabled() bool {
	return channelHealthEnabled.Load() && channelHealthHistoryEnabled.Load()
}

func SetChannelHealthHistoryBucketSeconds(seconds int) {
	if seconds < minChannelHealthHistoryBucketSeconds || seconds > maxChannelHealthHistoryBucketSeconds {
		return
	}
	channelHealthHistoryBucketSec.Store(int64(seconds))
}

func ChannelHealthHistoryBucketSeconds() int {
	return int(channelHealthHistoryBucketSec.Load())
}

func SetChannelHealthHistoryRetentionDays(days int) {
	if days < minChannelHealthHistoryRetentionDays || days > maxChannelHealthHistoryRetentionDays {
		return
	}
	channelHealthHistoryRetention.Store(int64(days))
}

func ChannelHealthHistoryRetentionDays() int {
	return int(channelHealthHistoryRetention.Load())
}
