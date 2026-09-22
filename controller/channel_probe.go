package controller

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

const (
	channelProbeCycle             = time.Minute
	channelProbeRandomDelayWindow = time.Minute
	channelProbeTimeout           = 240 * time.Second
	channelProbeLeaseSeconds      = int64(360)
)

// One system-task lease owns this continuous dispatcher. Individual channel
// leases also protect probes during master failover.
type channelProbeHandler struct{}

func (channelProbeHandler) Type() string { return model.SystemTaskTypeChannelProbe }
func (channelProbeHandler) Enabled() bool {
	return common.GetEnvOrDefaultBool("CHANNEL_PROBE_TASK_ENABLED", true)
}

// Only used to restart a stopped/failed dispatcher, not between channel probes.
func (channelProbeHandler) Interval() time.Duration { return 10 * time.Second }
func (channelProbeHandler) NewPayload() any         { return nil }

type channelProbeSummary struct {
	Checked   int `json:"checked"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Disabled  int `json:"disabled"`
	Enabled   int `json:"enabled"`
}

func shouldRunChannelProbe(channel *model.Channel) bool {
	if channel == nil || !channel.ShouldAutoProbe() {
		return false
	}
	if (channel.Status != common.ChannelStatusEnabled && channel.Status != common.ChannelStatusAutoDisabled) ||
		!supportsChannelTest(channel.Type) || channel.GetTestModel() == "" {
		return false
	}
	// 多 Key 渠道仍通过正常选 Key 机制测试，不绕过已禁用 Key 的隔离。
	return true
}

func (h channelProbeHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	testUserID, err := resolveChannelTestUserID(nil)
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	executor := channelProbeExecutor{
		testUserID: testUserID,
		request: func(ctx context.Context, channel *model.Channel, userID int) testResult {
			return testChannelWithTokenName(ctx, channel, userID, channel.GetTestModel(), "", shouldUseStreamForAutomaticChannelTest(channel), channelProbeTokenName, "")
		},
		randomDelay: func() time.Duration {
			return time.Duration(rand.Int64N(int64(channelProbeRandomDelayWindow)))
		},
	}
	var lastReport time.Time
	lastChecked := 0
	scheduler := channelProbeScheduler{
		loadCandidates: model.GetChannelProbeCandidates,
		probe:          executor.run,
		enabled:        h.Enabled,
		report: func(summary channelProbeSummary, active int) error {
			if summary.Checked != lastChecked {
				lastChecked = summary.Checked
			}
			if !lastReport.IsZero() && time.Since(lastReport) < 15*time.Second {
				return nil
			}
			lastReport = time.Now()
			// A continuous task has counters and active work, not a percentage.
			return model.UpdateSystemTaskState(task.TaskID, runnerID, struct {
				channelProbeSummary
				Active int `json:"active"`
			}{summary, active})
		},
	}
	ticker := time.NewTicker(channelProbeCycle)
	defer ticker.Stop()
	summary, err := scheduler.run(ctx, ticker.C)
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, summary, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, summary, nil)
}

type channelProbeResult struct {
	channelID int
	summary   channelProbeSummary
	err       error
}

// channelProbeScheduler counts the fixed one-minute system cycles per probe
// period. When a period group becomes due, every eligible channel in that group
// is launched asynchronously without a scheduler-level concurrency limit.
type channelProbeScheduler struct {
	loadCandidates func(context.Context) ([]*model.Channel, error)
	probe          func(context.Context, int) channelProbeResult
	enabled        func() bool
	report         func(channelProbeSummary, int) error
}

func (scheduler channelProbeScheduler) run(parent context.Context, ticks <-chan time.Time) (summary channelProbeSummary, err error) {
	ctx, cancel := context.WithCancel(parent)
	results := make(chan channelProbeResult)
	active := make(map[int]bool)
	periodCounts := make(map[int]int)
	var workers sync.WaitGroup
	defer func() {
		cancel()
		workers.Wait()
	}()

	if !scheduler.enabled() {
		return summary, nil
	}
	for {
		select {
		case <-ctx.Done():
			return summary, ctx.Err()
		case <-ticks:
			if !scheduler.enabled() {
				return summary, nil
			}
			channels, loadErr := scheduler.loadCandidates(ctx)
			if loadErr != nil {
				return summary, loadErr
			}

			groups := make(map[int][]*model.Channel)
			for _, channel := range channels {
				if !shouldRunChannelProbe(channel) {
					continue
				}
				period := channel.GetProbePeriodMinutes()
				groups[period] = append(groups[period], channel)
			}
			for period := range periodCounts {
				if _, exists := groups[period]; !exists {
					delete(periodCounts, period)
				}
			}
			for period, group := range groups {
				periodCounts[period]++
				if periodCounts[period] < period {
					continue
				}
				periodCounts[period] = 0
				for _, channel := range group {
					if active[channel.Id] {
						continue
					}
					active[channel.Id] = true
					workers.Add(1)
					go func(channelID int) {
						defer workers.Done()
						result := scheduler.probe(ctx, channelID)
						result.channelID = channelID
						select {
						case results <- result:
						case <-ctx.Done():
						}
					}(channel.Id)
				}
			}
			if scheduler.report != nil {
				if reportErr := scheduler.report(summary, len(active)); reportErr != nil {
					return summary, reportErr
				}
			}
		case result := <-results:
			delete(active, result.channelID)
			summary.Checked += result.summary.Checked
			summary.Succeeded += result.summary.Succeeded
			summary.Failed += result.summary.Failed
			summary.Disabled += result.summary.Disabled
			summary.Enabled += result.summary.Enabled
			if result.err != nil {
				// A single channel lease or persistence error must not stop the
				// continuous dispatcher for every other channel.
				common.SysError(fmt.Sprintf("channel probe worker failed: channel=%d error=%v", result.channelID, result.err))
			}
			if scheduler.report != nil {
				if reportErr := scheduler.report(summary, len(active)); reportErr != nil {
					return summary, reportErr
				}
			}
		}
	}
}

type channelProbeExecutor struct {
	testUserID  int
	request     func(context.Context, *model.Channel, int) testResult
	randomDelay func() time.Duration
}

func (executor channelProbeExecutor) waitRandomDelay(ctx context.Context, enabled bool) bool {
	if !enabled || executor.randomDelay == nil {
		return ctx.Err() == nil
	}
	delay := executor.randomDelay()
	if delay <= 0 {
		return ctx.Err() == nil
	}
	if delay > channelProbeRandomDelayWindow {
		delay = channelProbeRandomDelayWindow
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (executor channelProbeExecutor) run(ctx context.Context, channelID int) channelProbeResult {
	if ctx.Err() != nil {
		return channelProbeResult{}
	}
	now := common.GetTimestamp()
	leaseUntil := now + channelProbeLeaseSeconds
	claimed, err := model.ClaimChannelProbe(channelID, now, channelProbeLeaseSeconds)
	if err != nil || !claimed {
		return channelProbeResult{err: err}
	}
	defer func() {
		if err := model.ReleaseChannelProbe(channelID, leaseUntil); err != nil {
			common.SysError(fmt.Sprintf("release channel probe lease failed: channel=%d error=%v", channelID, err))
		}
	}()

	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return channelProbeResult{err: err}
	}
	if !shouldRunChannelProbe(channel) || ctx.Err() != nil {
		return channelProbeResult{}
	}
	if !executor.waitRandomDelay(ctx, channel.ProbeRandomDelayEnabled) {
		return channelProbeResult{}
	}
	if channel.ProbeRandomDelayEnabled {
		channel, err = model.GetChannelById(channelID, true)
		if err != nil {
			return channelProbeResult{err: err}
		}
		if !shouldRunChannelProbe(channel) {
			return channelProbeResult{}
		}
	}

	// Bound the upstream probe and all retries below the remaining lease time.
	probeCtx, cancel := context.WithTimeout(ctx, channelProbeTimeout)
	defer cancel()
	started := time.Now()
	var result testResult
	for attempt := 0; attempt <= channel.GetUpstreamMaxRetries(); attempt++ {
		if probeCtx.Err() != nil {
			break
		}
		result = executor.request(probeCtx, channel, executor.testUserID)
		if result.localErr == nil && result.newAPIError == nil {
			break
		}
	}
	if ctx.Err() != nil {
		return channelProbeResult{}
	}
	history := model.ChannelProbeHistory{
		ChannelID: channelID,
		CheckedAt: common.GetTimestamp(),
		LatencyMs: time.Since(started).Milliseconds(),
		Success:   result.localErr == nil && result.newAPIError == nil && probeCtx.Err() == nil,
	}
	if probeCtx.Err() != nil {
		history.StatusCode = 504
		history.ErrorMessage = probeCtx.Err().Error()
	} else if result.newAPIError != nil {
		history.StatusCode = result.newAPIError.StatusCode
		history.ErrorMessage = result.newAPIError.ErrorWithStatusCode()
	} else if result.localErr != nil {
		history.ErrorMessage = result.localErr.Error()
	}
	usingKey := ""
	if result.context != nil {
		usingKey = common.GetContextKeyString(result.context, constant.ContextKeyChannelKey)
	}
	status, changed, err := service.CompleteChannelProbe(ctx, channelID, history, leaseUntil, usingKey)
	if err != nil {
		return channelProbeResult{err: err}
	}
	summary := channelProbeSummary{Checked: 1}
	if history.Success {
		summary.Succeeded = 1
	} else {
		summary.Failed = 1
	}
	if changed && status == common.ChannelStatusEnabled {
		summary.Enabled = 1
	} else if changed && status == common.ChannelStatusAutoDisabled {
		summary.Disabled = 1
	}
	return channelProbeResult{summary: summary}
}
