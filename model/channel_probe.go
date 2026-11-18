package model

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const channelProbeHistoryRetention = 31 * 24 * time.Hour

// ChannelProbeState stores the latest probe result and the per-channel lease.
// Scheduling is driven by minute-based period groups in the system task.
type ChannelProbeState struct {
	ChannelID     int   `json:"channel_id" gorm:"primaryKey"`
	LastProbeAt   int64 `json:"last_probe_at" gorm:"bigint"`
	LeaseUntil    int64 `json:"-" gorm:"bigint;index"`
	LastIQProbeAt int64 `json:"last_iq_probe_at" gorm:"type:bigint;default:0"`
	IQLeaseUntil  int64 `json:"-" gorm:"type:bigint;default:0;index"`
	LastSuccess   bool  `json:"last_success"`
	LastLatencyMs int64 `json:"last_latency_ms"`
}

type ChannelProbeHistory struct {
	ID           int64  `json:"id" gorm:"primaryKey"`
	ChannelID    int    `json:"channel_id" gorm:"index:idx_channel_probe_history,priority:1;not null"`
	Success      bool   `json:"success"`
	LatencyMs    int64  `json:"latency_ms"`
	StatusCode   int    `json:"status_code"`
	ErrorMessage string `json:"error_message,omitempty" gorm:"type:text"`
	CheckedAt    int64  `json:"checked_at" gorm:"bigint;index:idx_channel_probe_history,priority:2;index"`
}

// ChannelIQProbeAnalysis carries the top detector prediction and calibrated
// probabilities used by channel display and dynamic routing.
type ChannelIQProbeAnalysis struct {
	Prediction       string
	ModelConfidences map[string]float64
}

func setChannelIQProbeAnalysis(channel *Channel, testModel string, analysis *ChannelIQProbeAnalysis, checkedAt int64) error {
	if analysis == nil {
		return errors.New("IQ probe requires a calibrated result")
	}
	prediction := strings.TrimSpace(analysis.Prediction)
	confidences := make(map[string]float64, len(analysis.ModelConfidences))
	for modelName, confidence := range analysis.ModelConfidences {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" || confidence < 0 || confidence > 1 || math.IsNaN(confidence) || math.IsInf(confidence, 0) {
			continue
		}
		confidences[modelName] = confidence
	}
	if prediction == "" || len(confidences) == 0 {
		return errors.New("invalid IQ probe analysis")
	}

	info, err := channelMetadataForUpdate(channel.OtherInfo)
	if err != nil {
		return err
	}
	info["iq_probe_engine"] = DefaultChannelIQProbeEngine
	info["iq_probe_test_model"] = strings.TrimSpace(testModel)
	info["iq_probe_at"] = checkedAt
	info["iq_probe_prediction"] = prediction
	info["iq_probe_degraded"] = !strings.EqualFold(prediction, strings.TrimSpace(testModel))
	if confidence, ok := confidences[prediction]; ok {
		info["iq_probe_confidence"] = confidence
	} else {
		delete(info, "iq_probe_confidence")
	}
	info["iq_probe_model_confidences"] = confidences
	return channel.SetOtherInfo(info)
}

// EnrichChannelIQProbeMetrics exposes the persisted detector result and gives
// channels without a calibrated result the neutral 100% confidence score.
func EnrichChannelIQProbeMetrics(channels []*Channel) {
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		neutralConfidence := float64(1)
		channel.IQTestConfidence = &neutralConfidence
		channel.IQProbePrediction = ""
		channel.IQProbeConfidence = nil
		channel.IQProbeModelConfidences = make(map[string]float64)
		channel.IQProbeAt = 0

		info := channel.GetOtherInfo()
		if checkedAt, ok := info["iq_probe_at"].(float64); ok && checkedAt > 0 {
			channel.IQProbeAt = int64(checkedAt)
		}
		if prediction, ok := info["iq_probe_prediction"].(string); ok {
			channel.IQProbePrediction = strings.TrimSpace(prediction)
		}
		if rawConfidences, ok := info["iq_probe_model_confidences"].(map[string]interface{}); ok {
			for modelName, rawConfidence := range rawConfidences {
				confidence, ok := rawConfidence.(float64)
				if !ok || confidence < 0 || confidence > 1 || math.IsNaN(confidence) || math.IsInf(confidence, 0) {
					continue
				}
				channel.IQProbeModelConfidences[modelName] = confidence
			}
		}
		if confidence, ok := info["iq_probe_confidence"].(float64); ok && confidence >= 0 && confidence <= 1 && !math.IsNaN(confidence) && !math.IsInf(confidence, 0) {
			channel.IQProbeConfidence = &confidence
		}
		if channel.IQProbeAt > 0 {
			confidence := float64(0)
			if measuredConfidence, ok := channel.IQProbeModelConfidences[channel.GetTestModel()]; ok {
				confidence = measuredConfidence
			}
			channel.IQTestConfidence = &confidence
		}
	}
}

// CompleteManualChannelIQTest only stores a calibrated result for the current
// test model. Recovery uses that very result under the same row lock.
func CompleteManualChannelIQTest(ctx context.Context, channelID int, testModel, usingKey string, analysis *ChannelIQProbeAnalysis) (*Channel, bool, error) {
	channelStatusLock.Lock()
	defer channelStatusLock.Unlock()
	var channel Channel
	var saved, changed bool
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ?", channelID).First(&channel).Error; err != nil {
			return err
		}
		if testModel != channel.GetTestModel() {
			return nil
		}
		if err := setChannelIQProbeAnalysis(&channel, testModel, analysis, common.GetTimestamp()); err != nil {
			return err
		}
		saved = true
		if channel.Status == common.ChannelStatusAutoDisabled && !channel.IQProbeDegraded() &&
			(channel.ShouldAutoProbe() || channel.GetOtherInfo()["status_reason"] == channelIQDegradationReason) {
			var err error
			changed, err = applyChannelProbeStatusWithAvailability(tx, &channel, true, usingKey, "", true, true, true)
			if changed || err != nil {
				return err
			}
		}
		return tx.Model(&Channel{}).Where("id = ?", channelID).Update("other_info", channel.OtherInfo).Error
	})
	if err != nil {
		return nil, false, err
	}
	if saved && common.MemoryCacheEnabled {
		SyncChannelCacheEntry(&channel)
	}
	return &channel, changed, nil
}

// GetChannelProbeCandidates loads every enabled or automatically disabled
// channel that participates in automatic probing. Credentials are loaded only
// after a worker acquires the per-channel lease.
func GetChannelProbeCandidates(ctx context.Context) ([]*Channel, error) {
	var channels []*Channel
	err := DB.WithContext(ctx).Model(&Channel{}).Omit("key").
		Where("auto_probe_enabled = ? AND status IN ?", true, []int{common.ChannelStatusEnabled, common.ChannelStatusAutoDisabled}).
		Order("probe_period_minutes ASC, id ASC").
		Find(&channels).Error
	return channels, err
}

// GetChannelIQProbeCandidates loads channels configured for scheduled IQ probes.
func GetChannelIQProbeCandidates(ctx context.Context) ([]*Channel, error) {
	var channels []*Channel
	err := DB.WithContext(ctx).Model(&Channel{}).Omit("key").
		Where("iq_probe_enabled = ? AND status IN ?", true, []int{common.ChannelStatusEnabled, common.ChannelStatusAutoDisabled}).
		Order("iq_probe_period_minutes ASC, id ASC").
		Find(&channels).Error
	return channels, err
}

// GetChannelProbeState returns the current scheduling state. A channel may
// have no row before its first probe; that is equivalent to an empty state and
// makes the first scheduled probe due immediately.
func GetChannelProbeState(channelID int) (*ChannelProbeState, error) {
	if channelID <= 0 {
		return nil, errors.New("channel id is required")
	}
	var state ChannelProbeState
	err := DB.Where("channel_id = ?", channelID).First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &ChannelProbeState{ChannelID: channelID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// CompleteChannelProbe atomically commits the history and optional status
// transition. It uses current settings and never overrides manual disable or
// accepts a result from an expired/replaced lease.
func CompleteChannelProbe(ctx context.Context, channelID int, result ChannelProbeHistory, leaseUntil int64, usingKey string) (*Channel, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	channelStatusLock.Lock()
	defer channelStatusLock.Unlock()
	var channel Channel
	changed := false
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ?", channelID).First(&channel).Error; err != nil {
			return err
		}
		var state ChannelProbeState
		if err := lockForUpdate(tx).Where("channel_id = ?", channelID).First(&state).Error; err != nil {
			return err
		}
		now := common.GetTimestamp()
		if leaseUntil <= now || state.LeaseUntil != leaseUntil {
			return errors.New("probe lease is no longer owned")
		}
		var err error
		changed, err = applyChannelProbeStatus(tx, &channel, result.Success, usingKey, result.ErrorMessage)
		if err != nil {
			return err
		}
		result.ChannelID = channelID
		result.CheckedAt = now
		return persistChannelProbeResult(tx, channelID, result, leaseUntil)
	})
	if err != nil {
		return nil, false, err
	}
	if changed && common.MemoryCacheEnabled {
		SyncChannelCacheEntry(&channel)
	}
	return &channel, changed, nil
}

// A nil availability result consumes only the IQ schedule slot. It must not
// become an availability observation or overwrite a previous calibrated result.
func CompleteChannelIQProbe(ctx context.Context, channelID int, result *ChannelProbeHistory, leaseUntil int64, usingKey, testModel string, analysis *ChannelIQProbeAnalysis) (*Channel, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	channelStatusLock.Lock()
	defer channelStatusLock.Unlock()
	var channel Channel
	changed := false
	analysisUpdated := false
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ?", channelID).First(&channel).Error; err != nil {
			return err
		}
		var state ChannelProbeState
		if err := lockForUpdate(tx).Where("channel_id = ?", channelID).First(&state).Error; err != nil {
			return err
		}
		now := common.GetTimestamp()
		if leaseUntil <= now || state.IQLeaseUntil != leaseUntil {
			return errors.New("IQ probe lease is no longer owned")
		}

		stateValues := map[string]any{"last_iq_probe_at": now, "iq_lease_until": 0}
		if result != nil && channel.ShouldIQProbe() && channel.GetTestModel() == testModel {
			if result.Success {
				if err := setChannelIQProbeAnalysis(&channel, testModel, analysis, now); err != nil {
					return err
				}
				analysisUpdated = true
				if err := tx.Model(&channel).Update("other_info", channel.OtherInfo).Error; err != nil {
					return err
				}
			}
			var err error
			changed, err = applyChannelProbeStatusForIQProbe(tx, &channel, result.Success, usingKey, result.ErrorMessage, analysisUpdated)
			if err != nil {
				return err
			}
			result.ChannelID = channelID
			result.CheckedAt = now
			if err := tx.Create(result).Error; err != nil {
				return err
			}
			stateValues["last_probe_at"] = now
			stateValues["last_success"] = result.Success
			stateValues["last_latency_ms"] = result.LatencyMs
			cutoff := now - int64(channelProbeHistoryRetention/time.Second)
			if err := tx.Where("checked_at < ?", cutoff).Delete(&ChannelProbeHistory{}).Error; err != nil {
				return err
			}
		}
		stateUpdate := tx.Model(&ChannelProbeState{}).
			Where("channel_id = ? AND iq_lease_until = ?", channelID, leaseUntil).
			Updates(stateValues)
		if stateUpdate.Error != nil {
			return stateUpdate.Error
		}
		if stateUpdate.RowsAffected != 1 {
			return errors.New("IQ probe lease is no longer owned")
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if (analysisUpdated || changed) && common.MemoryCacheEnabled {
		SyncChannelCacheEntry(&channel)
	}
	return &channel, changed, nil
}

// RecoverChannelAfterTest 使用与自动探测相同的事务条件，但不改动探测历史、租约或调度。
func RecoverChannelAfterTest(ctx context.Context, channelID int, usingKey string) (*Channel, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	channelStatusLock.Lock()
	defer channelStatusLock.Unlock()
	var channel Channel
	changed := false
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ?", channelID).First(&channel).Error; err != nil {
			return err
		}
		var err error
		changed, err = applyChannelProbeStatus(tx, &channel, true, usingKey, "")
		return err
	})
	if err != nil {
		return nil, false, err
	}
	if changed && common.MemoryCacheEnabled {
		SyncChannelCacheEntry(&channel)
	}
	return &channel, changed, nil
}

const channelIQDegradationReason = "IQ probe detected model degradation"

// applyChannelProbeStatus 只能传入当前事务中加行锁后读取的渠道。
// 探测闭环独立于业务请求的 auto_ban 和全局自动禁用配置。
func applyChannelProbeStatus(tx *gorm.DB, channel *Channel, success bool, usingKey, reason string) (bool, error) {
	return applyChannelProbeStatusWithAvailability(tx, channel, success, usingKey, reason, false, false, false)
}

func applyChannelProbeStatusForIQProbe(tx *gorm.DB, channel *Channel, success bool, usingKey, reason string, analysisUpdated bool) (bool, error) {
	return applyChannelProbeStatusWithAvailability(tx, channel, success, usingKey, reason, true, analysisUpdated, false)
}

func applyChannelProbeStatusWithAvailability(tx *gorm.DB, channel *Channel, success bool, usingKey, reason string, iqProbe, analysisUpdated, manualIQTest bool) (bool, error) {
	probeEnabled := channel.ShouldAutoProbe()
	if iqProbe {
		probeEnabled = probeEnabled || channel.ShouldIQProbe()
	}
	if (!probeEnabled && !manualIQTest) || channel.Status == common.ChannelStatusManuallyDisabled {
		return false, nil
	}
	beforeStatus := channel.Status
	if success {
		degradedAutoDisable := iqProbe && analysisUpdated && channel.IQAutoDisableDegraded && channel.IQProbeDegraded()
		if degradedAutoDisable {
			if channel.Status != common.ChannelStatusEnabled {
				return false, nil
			}
			channel.Status = common.ChannelStatusAutoDisabled
			reason = channelIQDegradationReason
		} else {
			if channel.Status != common.ChannelStatusAutoDisabled {
				return false, nil
			}
			if channel.IQAutoDisableDegraded && channel.IQProbeDegraded() && (!iqProbe || !analysisUpdated) {
				return false, nil
			}
			if channel.ChannelInfo.IsMultiKey {
				// 只恢复渠道，不解除任何 Key 隔离；成功的 Key 必须仍存在且当前可用。
				available := false
				for index, key := range channel.GetKeys() {
					status, explicit := channel.ChannelInfo.MultiKeyStatusList[index]
					if usingKey != "" && key == usingKey && (!explicit || status == common.ChannelStatusEnabled) {
						available = true
						break
					}
				}
				if !available {
					return false, nil
				}
			}
			channel.Status = common.ChannelStatusEnabled
		}
	} else {
		if channel.Status != common.ChannelStatusEnabled {
			return false, nil
		}
		channel.Status = common.ChannelStatusAutoDisabled
	}
	info, err := channelMetadataForUpdate(channel.OtherInfo)
	if err != nil {
		return false, err
	}
	info["status_time"] = common.GetTimestamp()
	info["status_reason"] = reason
	if err := channel.SetOtherInfo(info); err != nil {
		return false, err
	}
	statusQuery := tx.Model(&Channel{}).Where("id = ? AND status = ?", channel.Id, beforeStatus)
	if !manualIQTest {
		if iqProbe {
			statusQuery = statusQuery.Where("(auto_probe_enabled = ? OR iq_probe_enabled = ?)", true, true)
		} else {
			statusQuery = statusQuery.Where("auto_probe_enabled = ?", true)
		}
	}
	if err := statusQuery.Updates(map[string]any{"status": channel.Status, "other_info": channel.OtherInfo}).Error; err != nil {
		return false, err
	}
	if err := tx.Model(&Ability{}).Where("channel_id = ?", channel.Id).
		Update("enabled", channel.Status == common.ChannelStatusEnabled).Error; err != nil {
		return false, err
	}
	return true, nil
}

func ensureChannelProbeState(channelID int) (*ChannelProbeState, error) {
	if channelID <= 0 {
		return nil, errors.New("channel id is required")
	}
	var state ChannelProbeState
	err := DB.Where("channel_id = ?", channelID).First(&state).Error
	if err == nil {
		return &state, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	state = ChannelProbeState{ChannelID: channelID}
	if err := DB.Create(&state).Error; err != nil {
		// Another master may have created the row between First and Create.
		if loadErr := DB.Where("channel_id = ?", channelID).First(&state).Error; loadErr == nil {
			return &state, nil
		}
		return nil, err
	}
	return &state, nil
}

// ClaimChannelProbe atomically claims a channel whose previous lease is no
// longer active. Period-group scheduling decides when the channel is due.
func ClaimChannelProbe(channelID int, now int64, leaseSeconds int64) (bool, error) {
	if _, err := ensureChannelProbeState(channelID); err != nil {
		return false, err
	}
	if leaseSeconds <= 0 {
		leaseSeconds = 300
	}
	result := DB.Model(&ChannelProbeState{}).
		Where("channel_id = ? AND lease_until <= ?", channelID, now).
		Update("lease_until", now+leaseSeconds)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

// ReleaseChannelProbe clears a lease only while its timestamp still matches
// the caller's ownership token.
func ReleaseChannelProbe(channelID int, leaseUntil int64) error {
	if channelID <= 0 {
		return errors.New("channel id is required")
	}
	if leaseUntil <= 0 {
		return errors.New("probe lease is required")
	}
	return DB.Model(&ChannelProbeState{}).
		Where("channel_id = ? AND lease_until = ?", channelID, leaseUntil).
		Update("lease_until", 0).Error
}

func ClaimChannelIQProbe(channelID int, now int64, leaseSeconds int64) (bool, error) {
	if _, err := ensureChannelProbeState(channelID); err != nil {
		return false, err
	}
	if leaseSeconds <= 0 {
		leaseSeconds = 300
	}
	result := DB.Model(&ChannelProbeState{}).
		Where("channel_id = ? AND iq_lease_until <= ?", channelID, now).
		Update("iq_lease_until", now+leaseSeconds)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func ReleaseChannelIQProbe(channelID int, leaseUntil int64) error {
	if channelID <= 0 {
		return errors.New("channel id is required")
	}
	if leaseUntil <= 0 {
		return errors.New("IQ probe lease is required")
	}
	return DB.Model(&ChannelProbeState{}).
		Where("channel_id = ? AND iq_lease_until = ?", channelID, leaseUntil).
		Update("iq_lease_until", 0).Error
}

func SaveChannelProbeResult(channelID int, result ChannelProbeHistory) error {
	return saveChannelProbeResult(channelID, result, 0)
}

// SaveChannelProbeResultWithLease persists the outcome and clears the lease in
// one transaction. The lease token prevents a stale worker overwriting a newer
// worker's result.
func SaveChannelProbeResultWithLease(channelID int, result ChannelProbeHistory, leaseUntil int64) error {
	return saveChannelProbeResult(channelID, result, leaseUntil)
}

// GetLastChannelProbeTimes returns the latest automatic probe timestamp for
// each requested channel. Missing probe-state rows are intentionally omitted.
func GetLastChannelProbeTimes(channelIDs []int) (map[int]int64, error) {
	times := make(map[int]int64, len(channelIDs))
	if len(channelIDs) == 0 || DB == nil || !DB.Migrator().HasTable(&ChannelProbeState{}) {
		return times, nil
	}

	var states []ChannelProbeState
	if err := DB.Where("channel_id IN ?", channelIDs).Find(&states).Error; err != nil {
		return nil, err
	}
	for _, state := range states {
		if state.LastProbeAt > 0 {
			times[state.ChannelID] = state.LastProbeAt
		}
	}
	return times, nil
}

func saveChannelProbeResult(channelID int, result ChannelProbeHistory, leaseUntil int64) error {
	if channelID <= 0 {
		return errors.New("channel id is required")
	}
	result.ChannelID = channelID
	if result.CheckedAt == 0 {
		result.CheckedAt = common.GetTimestamp()
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		return persistChannelProbeResult(tx, channelID, result, leaseUntil)
	})
	return err
}

func persistChannelProbeResult(tx *gorm.DB, channelID int, result ChannelProbeHistory, leaseUntil int64) error {
	if err := tx.Create(&result).Error; err != nil {
		return err
	}
	stateQuery := tx.Model(&ChannelProbeState{}).Where("channel_id = ?", channelID)
	if leaseUntil > 0 {
		stateQuery = stateQuery.Where("lease_until = ?", leaseUntil)
	}
	stateUpdate := stateQuery.Updates(map[string]any{
		"last_probe_at":   result.CheckedAt,
		"lease_until":     0,
		"last_success":    result.Success,
		"last_latency_ms": result.LatencyMs,
	})
	if stateUpdate.Error != nil {
		return stateUpdate.Error
	}
	if leaseUntil > 0 && stateUpdate.RowsAffected != 1 {
		return errors.New("probe lease is no longer owned")
	}
	cutoff := result.CheckedAt - int64(channelProbeHistoryRetention/time.Second)
	return tx.Where("checked_at < ?", cutoff).
		Delete(&ChannelProbeHistory{}).Error
}

func PreviousNaturalDayBounds(now time.Time) (int64, int64) {
	localNow := now.In(time.Local)
	year, month, day := localNow.Date()
	todayStart := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	return todayStart.AddDate(0, 0, -1).Unix(), todayStart.Unix()
}

// GetPreviousDayChannelProbeSuccessRates returns percentages for the previous
// local calendar day. Channels without records default to 100 percent so new
// channels are not penalized before their first complete day.
func GetPreviousDayChannelProbeSuccessRates(channelIDs []int, now time.Time) (map[int]float64, error) {
	rates, _, err := GetPreviousDayChannelProbeStats(channelIDs, now)
	return rates, err
}

// GetPreviousDayChannelProbeStats returns the previous-day success percentage
// and sample count. Missing channels retain the neutral 100% compatibility
// value but have zero confidence for dynamic routing.
func GetPreviousDayChannelProbeStats(channelIDs []int, now time.Time) (map[int]float64, map[int]int, error) {
	rates := make(map[int]float64, len(channelIDs))
	samples := make(map[int]int, len(channelIDs))
	for _, channelID := range channelIDs {
		if channelID > 0 {
			rates[channelID] = 100
		}
	}
	if len(rates) == 0 || DB == nil || !DB.Migrator().HasTable(&ChannelProbeHistory{}) {
		return rates, samples, nil
	}

	start, end := PreviousNaturalDayBounds(now)
	var history []ChannelProbeHistory
	if err := DB.Where("channel_id IN ? AND checked_at >= ? AND checked_at < ?", channelIDs, start, end).
		Find(&history).Error; err != nil {
		return nil, nil, err
	}
	total := make(map[int]int)
	successes := make(map[int]int)
	for _, item := range history {
		total[item.ChannelID]++
		if item.Success {
			successes[item.ChannelID]++
		}
	}
	for channelID, count := range total {
		samples[channelID] = count
		if count > 0 {
			rates[channelID] = float64(successes[channelID]) / float64(count) * 100
		}
	}
	return rates, samples, nil
}

func GetPreviousDayChannelProbeSuccessRate(channelID int, now time.Time) (float64, error) {
	rates, err := GetPreviousDayChannelProbeSuccessRates([]int{channelID}, now)
	if err != nil {
		return 0, err
	}
	return rates[channelID], nil
}
