package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	channelMonitorRunnerInterval          = time.Second
	channelMonitorLeaseSeconds            = int64(180)
	channelMonitorClaimLimit              = 16
	channelMonitorMaxConcurrency          = 8
	channelMonitorMaxErrorLength          = 500
	channelMonitorMinInterval             = 1
	channelMonitorMaxInterval             = 86400
	channelMonitorMinTimeout              = 1
	channelMonitorMaxTimeout              = 120
	ChannelMonitorUserTestCooldownSeconds = int64(10)
)

var ErrChannelMonitorUserTestUnavailable = errors.New("channel monitor is not available for user tests")

type ChannelMonitorUserTestCooldownError struct {
	NextTestAt int64
}

func (err *ChannelMonitorUserTestCooldownError) Error() string {
	return "channel monitor user test is cooling down"
}

type ChannelMonitorInput struct {
	PricingGroup             string
	TestModel                string
	IntervalSeconds          int
	TimeoutSeconds           int
	Enabled                  bool
	Visible                  bool
	AvailabilityBoostPercent float64
	CreatedBy                int
}

type ChannelMonitorProbe func(context.Context, *model.Channel, *model.ChannelMonitor) (statusCode int, latencyMs int, err error)

type ChannelMonitorView struct {
	Monitor            *model.ChannelMonitor
	Status             string
	Latest             *model.ChannelMonitorHistory
	RawAvailability24h *float64
	RawAvailability7d  *float64
	RawAvailability30d *float64
	Availability24h    *float64
	Availability7d     *float64
	Availability30d    *float64
	RecentResults      []*model.ChannelMonitorHistory
}

type ChannelMonitorUserTestResult struct {
	Success    bool
	LatencyMs  int
	CheckedAt  int64
	NextTestAt int64
}

func validateChannelMonitorAvailabilityBoost(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
		return errors.New("availability boost must be between 0 and 100")
	}
	return nil
}

func applyChannelMonitorAvailabilityBoost(raw *float64, boost float64) *float64 {
	if raw == nil {
		return nil
	}
	value := *raw + (100-*raw)*boost/100
	value = math.Max(0, math.Min(100, value))
	value = math.Round(value*100) / 100
	return &value
}

func normalizeChannelMonitorInput(input ChannelMonitorInput) (ChannelMonitorInput, error) {
	input.PricingGroup = strings.TrimSpace(input.PricingGroup)
	input.TestModel = strings.TrimSpace(input.TestModel)

	if input.PricingGroup == "" || len(input.PricingGroup) > 100 {
		return input, errors.New("pricing group must contain 1 to 100 characters")
	}
	if !ratio_setting.ContainsGroupRatio(input.PricingGroup) {
		return input, errors.New("pricing group does not exist")
	}
	if input.TestModel == "" || len(input.TestModel) > 200 {
		return input, errors.New("test model must contain 1 to 200 characters")
	}
	if input.IntervalSeconds < channelMonitorMinInterval || input.IntervalSeconds > channelMonitorMaxInterval {
		return input, fmt.Errorf("test interval must be between %d and %d seconds", channelMonitorMinInterval, channelMonitorMaxInterval)
	}
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 15
	}
	if input.TimeoutSeconds < channelMonitorMinTimeout || input.TimeoutSeconds > channelMonitorMaxTimeout {
		return input, fmt.Errorf("request timeout must be between %d and %d seconds", channelMonitorMinTimeout, channelMonitorMaxTimeout)
	}
	if err := validateChannelMonitorAvailabilityBoost(input.AvailabilityBoostPercent); err != nil {
		return input, err
	}
	return input, nil
}

func CreateChannelMonitor(input ChannelMonitorInput) (*model.ChannelMonitor, error) {
	normalized, err := normalizeChannelMonitorInput(input)
	if err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	monitor := &model.ChannelMonitor{
		PricingGroup:             normalized.PricingGroup,
		TestModel:                normalized.TestModel,
		IntervalSeconds:          normalized.IntervalSeconds,
		TimeoutSeconds:           normalized.TimeoutSeconds,
		Enabled:                  normalized.Enabled,
		Visible:                  normalized.Visible,
		AvailabilityBoostPercent: normalized.AvailabilityBoostPercent,
		CreatedBy:                normalized.CreatedBy,
	}
	if monitor.Enabled {
		monitor.NextCheckAt = &now
	}
	if err := model.CreateChannelMonitor(monitor); err != nil {
		return nil, err
	}
	return monitor, nil
}

func UpdateChannelMonitor(id int, input ChannelMonitorInput) (*model.ChannelMonitor, error) {
	monitor, err := model.GetChannelMonitorByID(id)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeChannelMonitorInput(input)
	if err != nil {
		return nil, err
	}
	if normalized.PricingGroup != monitor.PricingGroup {
		return nil, errors.New("pricing group cannot be changed on a monitor")
	}
	monitor.TestModel = normalized.TestModel
	monitor.IntervalSeconds = normalized.IntervalSeconds
	monitor.TimeoutSeconds = normalized.TimeoutSeconds
	monitor.Enabled = normalized.Enabled
	monitor.Visible = normalized.Visible
	monitor.AvailabilityBoostPercent = normalized.AvailabilityBoostPercent
	now := common.GetTimestamp()
	if monitor.Enabled {
		monitor.NextCheckAt = &now
	} else {
		monitor.NextCheckAt = nil
	}
	if err := model.UpdateChannelMonitor(monitor); err != nil {
		return nil, err
	}
	return model.GetChannelMonitorByID(id)
}

func ListChannelMonitorViews(visibleOnly bool) ([]*ChannelMonitorView, error) {
	monitors, err := model.ListChannelMonitors(visibleOnly)
	if err != nil {
		return nil, err
	}
	views := make([]*ChannelMonitorView, 0, len(monitors))
	for _, monitor := range monitors {
		view, err := buildChannelMonitorView(monitor)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func GetChannelMonitorView(id int) (*ChannelMonitorView, error) {
	monitor, err := model.GetChannelMonitorByID(id)
	if err != nil {
		return nil, err
	}
	return buildChannelMonitorView(monitor)
}

func buildChannelMonitorView(monitor *model.ChannelMonitor) (*ChannelMonitorView, error) {
	now := common.GetTimestamp()
	view := &ChannelMonitorView{Monitor: monitor, Status: "unknown"}
	latest, err := model.GetLatestChannelMonitorHistory(monitor.Id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		view.Latest = latest
		if latest.Success {
			view.Status = "success"
		} else {
			view.Status = "failed"
		}
	}
	view.RawAvailability24h, err = model.GetChannelMonitorAvailability(monitor.Id, now-24*60*60)
	if err != nil {
		return nil, err
	}
	view.RawAvailability7d, err = model.GetChannelMonitorAvailability(monitor.Id, now-7*24*60*60)
	if err != nil {
		return nil, err
	}
	view.RawAvailability30d, err = model.GetChannelMonitorAvailability(monitor.Id, now-30*24*60*60)
	if err != nil {
		return nil, err
	}
	view.Availability24h = applyChannelMonitorAvailabilityBoost(view.RawAvailability24h, monitor.AvailabilityBoostPercent)
	view.Availability7d = applyChannelMonitorAvailabilityBoost(view.RawAvailability7d, monitor.AvailabilityBoostPercent)
	view.Availability30d = applyChannelMonitorAvailabilityBoost(view.RawAvailability30d, monitor.AvailabilityBoostPercent)
	view.RecentResults, err = model.ListChannelMonitorHistory(monitor.Id, 30)
	if err != nil {
		return nil, err
	}
	for left, right := 0, len(view.RecentResults)-1; left < right; left, right = left+1, right-1 {
		view.RecentResults[left], view.RecentResults[right] = view.RecentResults[right], view.RecentResults[left]
	}
	return view, nil
}

func ListChannelMonitorHistory(id int, limit int) ([]*model.ChannelMonitorHistory, error) {
	if _, err := model.GetChannelMonitorByID(id); err != nil {
		return nil, err
	}
	return model.ListChannelMonitorHistory(id, limit)
}

func RunChannelMonitorCheck(ctx context.Context, id int, probe ChannelMonitorProbe) (*model.ChannelMonitorHistory, error) {
	monitor, err := model.GetChannelMonitorByID(id)
	if err != nil {
		return nil, err
	}
	return runChannelMonitorCheck(ctx, monitor, probe)
}

func RunUserChannelMonitorTest(ctx context.Context, id int, probe ChannelMonitorProbe) (*ChannelMonitorUserTestResult, error) {
	monitor, err := model.GetChannelMonitorByID(id)
	if err != nil {
		return nil, err
	}
	if !monitor.Enabled || !monitor.Visible {
		return nil, ErrChannelMonitorUserTestUnavailable
	}

	now := common.GetTimestamp()
	leaseUntil := now + int64(monitor.TimeoutSeconds) + ChannelMonitorUserTestCooldownSeconds
	claimed, err := model.ClaimChannelMonitorUserTest(monitor.Id, now, leaseUntil)
	if err != nil {
		return nil, err
	}
	if !claimed {
		current, getErr := model.GetChannelMonitorByID(id)
		if getErr != nil {
			return nil, getErr
		}
		nextTestAt := now + ChannelMonitorUserTestCooldownSeconds
		if current.UserTestAvailableAt != nil && *current.UserTestAvailableAt > now {
			nextTestAt = *current.UserTestAvailableAt
		}
		return nil, &ChannelMonitorUserTestCooldownError{NextTestAt: nextTestAt}
	}

	history, err := runChannelMonitorCheck(ctx, monitor, probe)
	if err != nil {
		return nil, err
	}
	result := &ChannelMonitorUserTestResult{
		Success:   history.Success,
		LatencyMs: history.LatencyMs,
		CheckedAt: history.CheckedAt,
	}
	result.NextTestAt = common.GetTimestamp() + ChannelMonitorUserTestCooldownSeconds
	if completeErr := model.CompleteChannelMonitorUserTest(monitor.Id, result.NextTestAt); completeErr != nil {
		return nil, completeErr
	}
	return result, nil
}

func runChannelMonitorCheck(parent context.Context, monitor *model.ChannelMonitor, probe ChannelMonitorProbe) (*model.ChannelMonitorHistory, error) {
	checkedAt := common.GetTimestamp()
	result := &model.ChannelMonitorHistory{
		MonitorId: monitor.Id,
		CheckedAt: checkedAt,
	}
	if probe == nil {
		result.ErrorMessage = "pricing group monitor probe is not configured"
	} else {
		if parent == nil {
			parent = context.Background()
		}
		requestPath := "/v1/chat/completions"
		modelName := strings.ToLower(monitor.TestModel)
		switch {
		case strings.Contains(modelName, "embedding") || strings.HasPrefix(modelName, "m3e") || strings.Contains(modelName, "bge-"):
			requestPath = "/v1/embeddings"
		case strings.Contains(modelName, "rerank"):
			requestPath = "/v1/rerank"
		case strings.Contains(modelName, "codex") || strings.HasSuffix(modelName, ratio_setting.CompactModelSuffix):
			requestPath = "/v1/responses"
		}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		retryParam := &RetryParam{
			Ctx:         c,
			TokenGroup:  monitor.PricingGroup,
			ModelName:   monitor.TestModel,
			RequestPath: requestPath,
		}
		ctx, cancel := context.WithTimeout(parent, time.Duration(monitor.TimeoutSeconds)*time.Second)
		var lastErr error
		attemptLimit, configured := retryParam.groupRetryLimit(monitor.PricingGroup)
		if !configured {
			attemptLimit = 1
		}
		for attempt := 0; attempt < attemptLimit; attempt++ {
			if err := ctx.Err(); err != nil {
				lastErr = err
				break
			}
			channel, _, err := CacheGetRandomSatisfiedChannel(retryParam)
			if err != nil {
				lastErr = err
				break
			}
			if channel == nil {
				lastErr = fmt.Errorf("no enabled channel for pricing group %q and model %q", monitor.PricingGroup, monitor.TestModel)
				break
			}
			result.StatusCode, result.LatencyMs, lastErr = probe(ctx, channel, monitor)
			retryParam.MarkChannelAttempted(channel.Id)
			if lastErr == nil {
				result.Success = true
				break
			}
			retryParam.ExcludeChannel(channel.Id)
		}
		cancel()
		if lastErr != nil {
			result.ErrorMessage = truncateChannelMonitorError(lastErr.Error())
		}
	}
	if err := model.SaveChannelMonitorResult(monitor, result); err != nil {
		nextCheckAt := checkedAt + int64(monitor.IntervalSeconds)
		_ = model.ReleaseChannelMonitorLease(monitor.Id, nextCheckAt)
		return nil, err
	}
	return result, nil
}

func truncateChannelMonitorError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) <= channelMonitorMaxErrorLength {
		return message
	}
	return message[:channelMonitorMaxErrorLength]
}

var channelMonitorRunnerOnce sync.Once

func StartChannelMonitorRunner(probe ChannelMonitorProbe) {
	if !common.IsMasterNode {
		return
	}
	if probe == nil {
		common.SysError("channel monitor scheduler requires a pricing group probe")
		return
	}
	channelMonitorRunnerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(channelMonitorRunnerInterval)
			defer ticker.Stop()
			semaphore := make(chan struct{}, channelMonitorMaxConcurrency)
			for range ticker.C {
				now := common.GetTimestamp()
				monitors, err := model.ClaimDueChannelMonitors(now, channelMonitorLeaseSeconds, channelMonitorClaimLimit)
				if err != nil {
					common.SysError("channel monitor scheduler failed: " + err.Error())
					continue
				}
				for _, monitor := range monitors {
					semaphore <- struct{}{}
					go func(claimed *model.ChannelMonitor) {
						defer func() { <-semaphore }()
						if _, err := runChannelMonitorCheck(context.Background(), claimed, probe); err != nil {
							common.SysError(fmt.Sprintf("channel monitor %d failed to persist result: %v", claimed.Id, err))
						}
					}(monitor)
				}
			}
		}()
		common.SysLog("channel monitor scheduler started")
	})
}
