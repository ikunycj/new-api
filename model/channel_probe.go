package model

import (
	"context"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const channelProbeHistoryRetention = 31 * 24 * time.Hour

// ChannelProbeState stores request-independent scheduling state for one
// channel. The lease prevents multiple master nodes probing it concurrently.
type ChannelProbeState struct {
	ChannelID     int   `json:"channel_id" gorm:"primaryKey"`
	LastProbeAt   int64 `json:"last_probe_at" gorm:"bigint"`
	NextProbeAt   int64 `json:"next_probe_at" gorm:"bigint;index"`
	LeaseUntil    int64 `json:"-" gorm:"bigint;index"`
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

// GetDueChannelProbes loads only due work, oldest first. Credentials are loaded
// by the worker after it acquires a lease, not by every scheduler tick.
func GetDueChannelProbes(ctx context.Context, now int64) ([]*Channel, error) {
	var channels []*Channel
	err := DB.WithContext(ctx).Model(&Channel{}).Omit("key").
		Joins("LEFT JOIN channel_probe_states ON channel_probe_states.channel_id = channels.id").
		Where("channels.auto_probe_enabled = ? AND channels.status IN ?", true, []int{common.ChannelStatusEnabled, common.ChannelStatusAutoDisabled}).
		Where("COALESCE(channel_probe_states.next_probe_at, 0) <= ? AND COALESCE(channel_probe_states.lease_until, 0) <= ?", now, now).
		Order("COALESCE(channel_probe_states.next_probe_at, 0) ASC, channels.id ASC").
		Find(&channels).Error
	return channels, err
}

// CompleteChannelProbe atomically commits the history, schedule and optional
// status transition. It uses current settings and never overrides manual disable
// or accepts a result from an expired/replaced lease.
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
		interval := channel.GetProbeIntervalSeconds()
		if channel.Status == common.ChannelStatusAutoDisabled {
			interval = channel.GetAutoDisabledProbeIntervalSeconds()
		}
		result.ChannelID = channelID
		result.CheckedAt = now
		return persistChannelProbeResult(tx, channelID, result, now+int64(interval), leaseUntil)
	})
	if err != nil {
		return nil, false, err
	}
	if changed && common.MemoryCacheEnabled {
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

// applyChannelProbeStatus 只能传入当前事务中加行锁后读取的渠道。
// 探测闭环独立于业务请求的 auto_ban 和全局自动禁用配置。
func applyChannelProbeStatus(tx *gorm.DB, channel *Channel, success bool, usingKey, reason string) (bool, error) {
	if !channel.ShouldAutoProbe() || channel.Status == common.ChannelStatusManuallyDisabled {
		return false, nil
	}
	beforeStatus := channel.Status
	if success {
		if channel.Status != common.ChannelStatusAutoDisabled {
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
	} else {
		if channel.Status != common.ChannelStatusEnabled {
			return false, nil
		}
		channel.Status = common.ChannelStatusAutoDisabled
	}
	info := channel.GetOtherInfo()
	info["status_time"] = common.GetTimestamp()
	info["status_reason"] = reason
	channel.SetOtherInfo(info)
	if err := tx.Model(&Channel{}).Where("id = ? AND status = ? AND auto_probe_enabled = ?", channel.Id, beforeStatus, true).
		Updates(map[string]any{"status": channel.Status, "other_info": channel.OtherInfo}).Error; err != nil {
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

// ClaimChannelProbe atomically claims a due channel probe. A zero NextProbeAt
// means the channel has never been probed and is immediately due.
func ClaimChannelProbe(channelID int, now int64, leaseSeconds int64) (bool, error) {
	if _, err := ensureChannelProbeState(channelID); err != nil {
		return false, err
	}
	if leaseSeconds <= 0 {
		leaseSeconds = 300
	}
	result := DB.Model(&ChannelProbeState{}).
		Where("channel_id = ? AND next_probe_at <= ? AND lease_until <= ?", channelID, now, now).
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

func SaveChannelProbeResult(channelID int, result ChannelProbeHistory, nextProbeAt int64) error {
	return saveChannelProbeResult(channelID, result, nextProbeAt, 0)
}

// SaveChannelProbeResultWithLease persists the outcome and clears the lease in
// one transaction. The lease token prevents a stale worker overwriting a newer
// worker's result.
func SaveChannelProbeResultWithLease(channelID int, result ChannelProbeHistory, nextProbeAt int64, leaseUntil int64) error {
	return saveChannelProbeResult(channelID, result, nextProbeAt, leaseUntil)
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

func saveChannelProbeResult(channelID int, result ChannelProbeHistory, nextProbeAt int64, leaseUntil int64) error {
	if channelID <= 0 {
		return errors.New("channel id is required")
	}
	result.ChannelID = channelID
	if result.CheckedAt == 0 {
		result.CheckedAt = common.GetTimestamp()
	}
	if nextProbeAt < result.CheckedAt {
		nextProbeAt = result.CheckedAt
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return persistChannelProbeResult(tx, channelID, result, nextProbeAt, leaseUntil)
	})
}

func persistChannelProbeResult(tx *gorm.DB, channelID int, result ChannelProbeHistory, nextProbeAt, leaseUntil int64) error {
	if err := tx.Create(&result).Error; err != nil {
		return err
	}
	stateQuery := tx.Model(&ChannelProbeState{}).Where("channel_id = ?", channelID)
	if leaseUntil > 0 {
		stateQuery = stateQuery.Where("lease_until = ?", leaseUntil)
	}
	stateUpdate := stateQuery.Updates(map[string]any{
		"last_probe_at":   result.CheckedAt,
		"next_probe_at":   nextProbeAt,
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
