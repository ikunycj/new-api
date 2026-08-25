package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
)

// RegisterScheduledSystemTasks wires the periodic channel test, upstream model
// update, and async task polling (Midjourney / Suno / video) jobs into the
// system task framework so a DB lease dedups execution across multiple master
// instances and each run is recorded as one task row. Call this before
// service.StartSystemTaskRunner.
func RegisterScheduledSystemTasks() {
	service.RegisterSystemTaskHandler(channelTestHandler{})
	service.RegisterSystemTaskHandler(channelProbeHandler{})
	service.RegisterSystemTaskHandler(modelUpdateHandler{})
	service.RegisterSystemTaskHandler(midjourneyPollHandler{})
	service.RegisterSystemTaskHandler(asyncTaskPollHandler{})
}

// channelProbeHandler runs the per-channel probe scheduler. ChannelProbeState
// carries each channel's due time and lease independently from the system task.
type channelProbeHandler struct{}

func (channelProbeHandler) Type() string { return model.SystemTaskTypeChannelProbe }

func (channelProbeHandler) Enabled() bool {
	return common.GetEnvOrDefaultBool("CHANNEL_PROBE_TASK_ENABLED", true)
}

func (channelProbeHandler) Interval() time.Duration { return time.Minute }

func (channelProbeHandler) NewPayload() any { return nil }

type channelProbeSummary struct {
	Checked   int `json:"checked"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Disabled  int `json:"disabled"`
	Enabled   int `json:"enabled"`
}

func shouldRunChannelProbe(channel *model.Channel) bool {
	if channel == nil || channel.Status == common.ChannelStatusManuallyDisabled || !supportsChannelTest(channel.Type) || channel.GetTestModel() == "" {
		return false
	}
	// The normal relay path deliberately refuses every disabled key. Until the
	// probe has an explicit key-only recovery path, probing an auto-disabled
	// multi-key channel cannot recover it and must not claim otherwise.
	return !channel.ChannelInfo.IsMultiKey || channel.Status != common.ChannelStatusAutoDisabled
}

func channelProbeIntervalSeconds(channel *model.Channel, resultingStatus int) int {
	if resultingStatus == common.ChannelStatusAutoDisabled {
		return channel.GetAutoDisabledProbeIntervalSeconds()
	}
	return channel.GetProbeIntervalSeconds()
}

func (channelProbeHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	summary, err := runChannelProbeTask(ctx, service.NewSystemTaskProgressReporter(task, runnerID))
	if summary.Checked > 0 {
		service.PublishChannelProbeRefresh()
	}
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, summary, nil)
}

func runChannelProbeTask(ctx context.Context, report func(processed, total int)) (channelProbeSummary, error) {
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		return channelProbeSummary{}, err
	}
	testUserID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return channelProbeSummary{}, err
	}

	var summary channelProbeSummary
	total := len(channels)
	for index, channel := range channels {
		if ctx != nil && ctx.Err() != nil {
			break
		}
		if report != nil {
			report(index, total)
		}
		if !shouldRunChannelProbe(channel) {
			continue
		}

		now := common.GetTimestamp()
		const probeLeaseSeconds = int64(300)
		leaseUntil := now + probeLeaseSeconds
		claimed, err := model.ClaimChannelProbe(channel.Id, now, probeLeaseSeconds)
		if err != nil {
			return summary, err
		}
		if !claimed {
			continue
		}

		leaseReleased := false
		releaseLease := func() {
			if leaseReleased {
				return
			}
			leaseReleased = true
			if releaseErr := model.ReleaseChannelProbe(channel.Id, leaseUntil); releaseErr != nil {
				common.SysError(fmt.Sprintf("release channel probe lease failed: channel=%d error=%v", channel.Id, releaseErr))
			}
		}

		started := time.Now()
		result := testResult{}
		cancelled := false
		for attempt := 0; attempt <= channel.GetUpstreamMaxRetries(); attempt++ {
			if ctx != nil && ctx.Err() != nil {
				cancelled = true
				break
			}
			result = testChannelWithTokenName(ctx, channel, testUserID, channel.GetTestModel(), "", shouldUseStreamForAutomaticChannelTest(channel), channelProbeTokenName)
			if result.localErr == nil && result.newAPIError == nil {
				break
			}
		}
		if cancelled || (ctx != nil && ctx.Err() != nil) {
			releaseLease()
			break
		}

		latencyMs := time.Since(started).Milliseconds()
		if latencyMs < 0 {
			latencyMs = 0
		}
		success := result.localErr == nil && result.newAPIError == nil
		statusCode := 0
		errorMessage := ""
		if result.newAPIError != nil {
			statusCode = result.newAPIError.StatusCode
			errorMessage = result.newAPIError.ErrorWithStatusCode()
		} else if result.localErr != nil {
			errorMessage = result.localErr.Error()
		}

		resultingStatus := channel.Status
		if !success && !channel.ChannelInfo.IsMultiKey && channel.Status == common.ChannelStatusEnabled && channel.ShouldProbeFailureAutoBan() && common.AutomaticDisableChannelEnabled {
			usingKey := ""
			if result.context != nil {
				usingKey = common.GetContextKeyString(result.context, constant.ContextKeyChannelKey)
			}
			service.DisableChannel(*types.NewChannelError(
				channel.Id,
				channel.Type,
				channel.Name,
				channel.ChannelInfo.IsMultiKey,
				usingKey,
				true,
			), errorMessage)
			resultingStatus = common.ChannelStatusAutoDisabled
			summary.Disabled++
		}
		if success && !channel.ChannelInfo.IsMultiKey && channel.Status == common.ChannelStatusAutoDisabled && channel.ShouldProbeSuccessAutoEnable() && common.AutomaticEnableChannelEnabled {
			usingKey := ""
			if result.context != nil {
				usingKey = common.GetContextKeyString(result.context, constant.ContextKeyChannelKey)
			}
			service.EnableChannel(channel.Id, usingKey, channel.Name)
			resultingStatus = common.ChannelStatusEnabled
			summary.Enabled++
		}

		if success {
			summary.Succeeded++
		} else {
			summary.Failed++
		}
		summary.Checked++
		checkedAt := common.GetTimestamp()
		if err := model.SaveChannelProbeResultWithLease(channel.Id, model.ChannelProbeHistory{
			ChannelID:    channel.Id,
			Success:      success,
			LatencyMs:    latencyMs,
			StatusCode:   statusCode,
			ErrorMessage: errorMessage,
			CheckedAt:    checkedAt,
		}, checkedAt+int64(channelProbeIntervalSeconds(channel, resultingStatus)), leaseUntil); err != nil {
			releaseLease()
			return summary, err
		}
		leaseReleased = true
	}
	if report != nil && (ctx == nil || ctx.Err() == nil) {
		report(total, total)
	}
	return summary, nil
}

// channelTestHandler runs the scheduled "test all channels" job. Enablement and
// cadence still come from the monitor settings; only the execution path moved
// into the system task runner.
type channelTestHandler struct{}

func (channelTestHandler) Type() string { return model.SystemTaskTypeChannelTest }

func (channelTestHandler) Enabled() bool {
	return operation_setting.GetMonitorSetting().AutoTestChannelEnabled
}

func (channelTestHandler) Interval() time.Duration {
	minutes := operation_setting.GetMonitorSetting().AutoTestChannelMinutes
	if minutes <= 0 {
		minutes = 10
	}
	return time.Duration(minutes * float64(time.Minute))
}

func (channelTestHandler) NewPayload() any { return nil }

// channelTestTaskPayload controls one channel_test run. A nil/empty payload is a
// scheduled run, which uses the configured monitor ChannelTestMode and does not
// notify. A manual "test all channels" trigger sets Mode=scheduled_all and
// Notify=true to reproduce the legacy manual behavior (test every channel and
// notify root on completion).
type channelTestTaskPayload struct {
	Mode   string `json:"mode,omitempty"`
	Notify bool   `json:"notify,omitempty"`
}

func (channelTestHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	payload := channelTestTaskPayload{}
	if err := task.DecodePayload(&payload); err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	summary, err := runChannelTestTask(ctx, payload.Mode, payload.Notify, service.NewSystemTaskProgressReporter(task, runnerID))
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, summary, nil)
}

// modelUpdateHandler runs the scheduled upstream model update detection job.
type modelUpdateHandler struct{}

func (modelUpdateHandler) Type() string { return model.SystemTaskTypeModelUpdate }

func (modelUpdateHandler) Enabled() bool {
	return common.GetEnvOrDefaultBool("CHANNEL_UPSTREAM_MODEL_UPDATE_TASK_ENABLED", true)
}

func (modelUpdateHandler) Interval() time.Duration {
	intervalMinutes := common.GetEnvOrDefault(
		"CHANNEL_UPSTREAM_MODEL_UPDATE_TASK_INTERVAL_MINUTES",
		channelUpstreamModelUpdateTaskDefaultIntervalMinutes,
	)
	if intervalMinutes < 1 {
		intervalMinutes = channelUpstreamModelUpdateTaskDefaultIntervalMinutes
	}
	return time.Duration(intervalMinutes) * time.Minute
}

func (modelUpdateHandler) NewPayload() any { return nil }

// modelUpdateTaskPayload controls one model_update run. A scheduled run
// (Manual=false) respects the per-channel minimum check interval and may
// auto-apply detected models when a channel has auto-sync enabled. A manual
// "detect all" trigger sets Manual=true to reproduce the legacy detect-all
// semantics: force a re-check regardless of the interval and never auto-apply,
// so the admin reviews and applies changes explicitly.
type modelUpdateTaskPayload struct {
	Manual bool `json:"manual,omitempty"`
}

func (modelUpdateHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	payload := modelUpdateTaskPayload{}
	if err := task.DecodePayload(&payload); err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	summary := runChannelUpstreamModelUpdateTaskOnce(ctx, payload.Manual, !payload.Manual, service.NewSystemTaskProgressReporter(task, runnerID))
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, summary, nil)
}

// midjourneyPollHandler runs one Midjourney polling pass per scheduled run.
// Enabled() folds the "are there unfinished tasks?" check into enablement so the
// scheduler creates no row when the system is idle; only when at least one
// Midjourney task is in progress does a row get scheduled.
type midjourneyPollHandler struct{}

func (midjourneyPollHandler) Type() string { return model.SystemTaskTypeMidjourneyPoll }

func (midjourneyPollHandler) Enabled() bool {
	return constant.UpdateTask && model.HasUnfinishedMidjourneyTasks()
}

func (midjourneyPollHandler) Interval() time.Duration { return 15 * time.Second }

func (midjourneyPollHandler) NewPayload() any { return nil }

func (midjourneyPollHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	summary := runMidjourneyTaskUpdateOnce(ctx, service.NewSystemTaskProgressReporter(task, runnerID))
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, summary, nil)
}

// asyncTaskPollHandler runs one async-task (Suno/video) polling pass per
// scheduled run. Like midjourneyPollHandler, Enabled() folds in the unfinished
// task existence check so an idle system schedules no rows.
type asyncTaskPollHandler struct{}

func (asyncTaskPollHandler) Type() string { return model.SystemTaskTypeAsyncTaskPoll }

func (asyncTaskPollHandler) Enabled() bool {
	return constant.UpdateTask && model.HasUnfinishedSyncTasks()
}

func (asyncTaskPollHandler) Interval() time.Duration { return 15 * time.Second }

func (asyncTaskPollHandler) NewPayload() any { return nil }

func (asyncTaskPollHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	summary := service.RunTaskPollingOnce(ctx, service.NewSystemTaskProgressReporter(task, runnerID))
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, summary, nil)
}

func finishSystemTaskHandler(task *model.SystemTask, runnerID string, status model.SystemTaskStatus, result any, runErr error) {
	errorMessage := ""
	if runErr != nil {
		errorMessage = runErr.Error()
	}
	if err := model.FinishSystemTask(task.TaskID, runnerID, status, result, errorMessage); err != nil {
		common.SysLog(fmt.Sprintf("system task %s failed to persist result: %v", task.TaskID, err))
	}
}
