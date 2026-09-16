package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

const (
	channelProbeWorkerCount  = 8
	channelProbeTimeout      = 240 * time.Second
	channelProbeLeaseSeconds = int64(300)
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
	}
	var lastReport time.Time
	lastChecked := 0
	scheduler := channelProbeScheduler{
		loadDue: model.GetDueChannelProbes,
		probe:   executor.run,
		enabled: h.Enabled,
		report: func(summary channelProbeSummary, active int) error {
			if summary.Checked != lastChecked {
				service.PublishChannelProbeRefresh()
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
	ticker := time.NewTicker(time.Second)
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

// The dispatcher owns all counters and active IDs. Workers only send results,
// so a slow channel cannot block another channel's next due probe.
type channelProbeScheduler struct {
	loadDue func(context.Context, int64) ([]*model.Channel, error)
	probe   func(context.Context, int) channelProbeResult
	enabled func() bool
	report  func(channelProbeSummary, int) error
}

func (scheduler channelProbeScheduler) run(parent context.Context, ticks <-chan time.Time) (summary channelProbeSummary, err error) {
	ctx, cancel := context.WithCancel(parent)
	results := make(chan channelProbeResult, channelProbeWorkerCount)
	active := make(map[int]bool)
	var workers sync.WaitGroup
	defer func() {
		cancel()
		workers.Wait()
	}()
	scan := true
	for {
		if ctx.Err() != nil {
			return summary, ctx.Err()
		}
		if scan {
			if !scheduler.enabled() {
				return summary, nil
			}
			if len(active) < channelProbeWorkerCount {
				channels, loadErr := scheduler.loadDue(ctx, common.GetTimestamp())
				if loadErr != nil {
					return summary, loadErr
				}
				for _, channel := range channels {
					if ctx.Err() != nil || len(active) == channelProbeWorkerCount {
						break
					}
					if !shouldRunChannelProbe(channel) || active[channel.Id] {
						continue
					}
					// Busy relay channels must not repeatedly consume the first worker
					// slots and starve later recovery probes. The worker still acquires
					// capacity atomically to cover races with new relay requests.
					if service.CurrentChannelConcurrency(channel.Id) >= channel.GetMaxConcurrency() {
						continue
					}
					active[channel.Id] = true
					workers.Add(1)
					go func(id int) {
						defer workers.Done()
						result := scheduler.probe(ctx, id)
						result.channelID = id
						results <- result
					}(channel.Id)
				}
			}
			if scheduler.report != nil {
				if reportErr := scheduler.report(summary, len(active)); reportErr != nil {
					return summary, reportErr
				}
			}
			scan = false
		}
		select {
		case <-ctx.Done():
			return summary, ctx.Err()
		case <-ticks:
			scan = true
		case result := <-results:
			delete(active, result.channelID)
			summary.Checked += result.summary.Checked
			summary.Succeeded += result.summary.Succeeded
			summary.Failed += result.summary.Failed
			summary.Disabled += result.summary.Disabled
			summary.Enabled += result.summary.Enabled
			if result.err != nil {
				// A single channel lease or persistence error must not stop the
				// continuous dispatcher for every other channel. The worker has
				// already released its active slot; the channel can be retried on
				// a later scan after its lease expires.
				common.SysError(fmt.Sprintf("channel probe worker failed: channel=%d error=%v", result.channelID, result.err))
			}
		}
	}
}

type channelProbeExecutor struct {
	testUserID int
	request    func(context.Context, *model.Channel, int) testResult
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
	// Re-read after claiming: status/configuration may change while awaiting a slot.
	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return channelProbeResult{err: err}
	}
	if !shouldRunChannelProbe(channel) || ctx.Err() != nil {
		return channelProbeResult{}
	}

	// Bound the whole probe, including retries, below the channel lease lifetime.
	probeCtx, cancel := context.WithTimeout(ctx, channelProbeTimeout)
	defer cancel()
	started := time.Now()
	var result testResult
	for attempt := 0; attempt <= channel.GetUpstreamMaxRetries(); attempt++ {
		if probeCtx.Err() != nil {
			break
		}
		if !service.TryAcquireChannelConcurrency(channelID, channel.GetMaxConcurrency()) {
			// Capacity exhaustion is not an upstream failure.
			return channelProbeResult{}
		}
		result = func() testResult {
			defer service.ReleaseChannelConcurrency(channelID)
			return executor.request(probeCtx, channel, executor.testUserID)
		}()
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
