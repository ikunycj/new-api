package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/bytedance/gopkg/util/gopool"
)

// The synthetic probe scheduler is the missing piece of the pre-existing probe
// infrastructure in model/channel_probe.go: the lease, history and retention
// helpers were all written but never driven by anything, so the probe tables
// stayed empty and Channel.PreviousDayProbeSuccessRate always fell back to a
// hard-coded 100.
//
// Two properties matter more than probe coverage here:
//
//   - Probes cost real upstream money. A channel that real traffic already
//     exercises within the idle-grace window is skipped entirely, so probing
//     concentrates on genuinely idle channels.
//   - Probing must never fight the circuit breaker. A channel whose circuit is
//     open is left alone: the breaker's own half-open request is the designated
//     recovery probe, and duplicating it would both waste calls and distort the
//     breaker's failure accounting.
const (
	// probeLeaseSeconds bounds how long a crashed node can block a channel's
	// probe. It is deliberately larger than any realistic probe duration.
	probeLeaseSeconds = 120

	// probeConcurrency caps parallel upstream probes so a large channel list
	// cannot turn one tick into a traffic spike.
	probeConcurrency = 4

	// probeTimeout stops a hanging upstream from holding a lease for the whole
	// lease window.
	probeTimeout = 30 * time.Second
)

// ChannelProbeExecutor performs one synthetic request against a channel and
// reports success plus observed latency. It is injected from the controller
// package so this scheduler can reuse the existing, battle-tested testChannel
// implementation instead of duplicating provider-specific request building.
type ChannelProbeExecutor func(ctx context.Context, channel *model.Channel) (bool, time.Duration, error)

var (
	channelProbeExecutor   atomic.Pointer[ChannelProbeExecutor]
	channelProbeTaskOnce   sync.Once
	channelProbeTickActive atomic.Bool
	channelProbeStats      struct {
		sync.Mutex
		lastRunAt   time.Time
		lastProbed  int
		lastSkipped int
		lastFailed  int
	}
)

// SetChannelProbeExecutor registers the probe implementation. Without it the
// scheduler stays dormant rather than inventing its own request format.
func SetChannelProbeExecutor(executor ChannelProbeExecutor) {
	if executor == nil {
		channelProbeExecutor.Store(nil)
		return
	}
	channelProbeExecutor.Store(&executor)
}

func loadChannelProbeExecutor() ChannelProbeExecutor {
	if pointer := channelProbeExecutor.Load(); pointer != nil {
		return *pointer
	}
	return nil
}

// StartChannelHealthProbeTask launches the probe loop on the master node only.
// Followers would otherwise duplicate every probe; the database lease is a
// second line of defence rather than the primary one.
func StartChannelHealthProbeTask() {
	channelProbeTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), "channel health probe task started")
			// The tick is fixed at the minimum supported interval and each
			// channel's own next_probe_at decides eligibility, so an operator
			// can lengthen the interval at runtime without a restart.
			ticker := time.NewTicker(time.Duration(minChannelProbeTickSeconds) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				RunChannelHealthProbeOnce()
			}
		})
	})
}

const minChannelProbeTickSeconds = 10

// RunChannelHealthProbeOnce probes every eligible channel once. It is exported
// so tests and an operator-triggered refresh can drive it directly.
func RunChannelHealthProbeOnce() {
	if !common.IsChannelHealthProbeEnabled() {
		return
	}
	executor := loadChannelProbeExecutor()
	if executor == nil {
		return
	}
	// Overlap guard: a slow upstream must not cause ticks to pile up.
	if !channelProbeTickActive.CompareAndSwap(false, true) {
		return
	}
	defer channelProbeTickActive.Store(false)

	channels, err := model.GetChannelsForHealthProbe()
	if err != nil {
		common.SysError(fmt.Sprintf("channel health probe: list channels failed: %v", err))
		return
	}

	now := time.Now()
	interval := time.Duration(common.ChannelHealthProbeIntervalSeconds()) * time.Second
	idleGrace := time.Duration(common.ChannelHealthProbeIdleGraceSeconds()) * time.Second

	var (
		wg      sync.WaitGroup
		slots   = make(chan struct{}, probeConcurrency)
		probed  atomic.Int64
		skipped atomic.Int64
		failed  atomic.Int64
	)

	for index := range channels {
		channel := channels[index]
		if channel.Id <= 0 {
			continue
		}
		if !channelProbeDue(channel.Id, now, idleGrace) {
			skipped.Add(1)
			continue
		}

		// The lease is claimed before the upstream call so a concurrent node
		// cannot probe the same channel in the same window.
		leaseUntil := now.Unix() + probeLeaseSeconds
		claimed, claimErr := model.ClaimChannelProbe(channel.Id, now.Unix(), probeLeaseSeconds)
		if claimErr != nil {
			common.SysError(fmt.Sprintf("channel health probe: claim channel %d failed: %v", channel.Id, claimErr))
			continue
		}
		if !claimed {
			skipped.Add(1)
			continue
		}

		wg.Add(1)
		slots <- struct{}{}
		gopool.Go(func() {
			defer func() {
				<-slots
				wg.Done()
			}()
			success := runSingleChannelProbe(executor, channel, interval, leaseUntil)
			probed.Add(1)
			if !success {
				failed.Add(1)
			}
		})
	}
	wg.Wait()

	channelProbeStats.Lock()
	channelProbeStats.lastRunAt = now
	channelProbeStats.lastProbed = int(probed.Load())
	channelProbeStats.lastSkipped = int(skipped.Load())
	channelProbeStats.lastFailed = int(failed.Load())
	channelProbeStats.Unlock()
}

// channelProbeDue reports whether a channel needs a synthetic probe. Recent
// real traffic is treated as a better signal than a probe, so it suppresses
// probing for the whole idle-grace window.
func channelProbeDue(channelID int, now time.Time, idleGrace time.Duration) bool {
	if idleGrace <= 0 {
		return true
	}
	lastObserved, ok := ChannelLastObservedAt(channelID)
	if !ok {
		// Never observed: probing is the only way to learn anything about it.
		return true
	}
	return now.Sub(lastObserved) >= idleGrace
}

// runSingleChannelProbe executes one probe, feeds the result into the health
// score and persists it through the existing probe history helpers.
func runSingleChannelProbe(executor ChannelProbeExecutor, channel *model.Channel,
	interval time.Duration, leaseUntil int64) bool {

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	startedAt := time.Now()
	success, latency, probeErr := executor(ctx, channel)
	if latency <= 0 {
		latency = time.Since(startedAt)
	}

	message := ""
	if probeErr != nil {
		message = probeErr.Error()
	}

	// Probe results are attributed to the synthetic probe route so they never
	// overwrite the per-route scores earned by real traffic.
	RecordChannelHealthSample(ChannelHealthSample{
		ChannelID: channel.Id,
		Route:     ChannelHealthProbeRoute,
		Success:   success,
		Latency:   latency,
		Observed:  startedAt,
	})

	nextProbeAt := startedAt.Add(interval).Unix()
	history := model.ChannelProbeHistory{
		Success:      success,
		LatencyMs:    latency.Milliseconds(),
		ErrorMessage: message,
		CheckedAt:    startedAt.Unix(),
	}
	if err := model.SaveChannelProbeResultWithLease(channel.Id, history, nextProbeAt, leaseUntil); err != nil {
		common.SysError(fmt.Sprintf("channel health probe: persist channel %d failed: %v", channel.Id, err))
		// The lease is released explicitly so a persistence failure cannot
		// block this channel until the lease expires.
		if releaseErr := model.ReleaseChannelProbe(channel.Id, leaseUntil); releaseErr != nil {
			common.SysError(fmt.Sprintf("channel health probe: release channel %d failed: %v", channel.Id, releaseErr))
		}
	}
	return success
}

// ChannelHealthProbeRoute is the pseudo-route recorded for synthetic probes.
// Keeping probe samples on their own route means an idle channel builds a
// health history without diluting the scores measured from real traffic.
const ChannelHealthProbeRoute = "__probe__"

// ChannelProbeRunStats is the read-only summary of the most recent probe tick.
type ChannelProbeRunStats struct {
	LastRunAt   time.Time `json:"last_run_at"`
	LastProbed  int       `json:"last_probed"`
	LastSkipped int       `json:"last_skipped"`
	LastFailed  int       `json:"last_failed"`
}

func GetChannelProbeRunStats() ChannelProbeRunStats {
	channelProbeStats.Lock()
	defer channelProbeStats.Unlock()
	return ChannelProbeRunStats{
		LastRunAt:   channelProbeStats.lastRunAt,
		LastProbed:  channelProbeStats.lastProbed,
		LastSkipped: channelProbeStats.lastSkipped,
		LastFailed:  channelProbeStats.lastFailed,
	}
}
