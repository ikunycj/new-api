package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const channelProbeHistoryRetention = 31 * 24 * time.Hour

// ChannelProbeState is the request-independent scheduler state for one
// channel. The lease prevents two master nodes from probing the same channel
// concurrently; the system-task lease remains the coarse-grained scheduler
// guard.
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

func SaveChannelProbeResult(channelID int, result ChannelProbeHistory, nextProbeAt int64) error {
	if result.CheckedAt == 0 {
		result.CheckedAt = common.GetTimestamp()
	}
	if nextProbeAt < result.CheckedAt {
		nextProbeAt = result.CheckedAt
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&result).Error; err != nil {
			return err
		}
		if err := tx.Model(&ChannelProbeState{}).
			Where("channel_id = ?", channelID).
			Updates(map[string]any{
				"last_probe_at":   result.CheckedAt,
				"next_probe_at":   nextProbeAt,
				"lease_until":     0,
				"last_success":    result.Success,
				"last_latency_ms": result.LatencyMs,
			}).Error; err != nil {
			return err
		}
		cutoff := result.CheckedAt - int64(channelProbeHistoryRetention/time.Second)
		return tx.Where("channel_id = ? AND checked_at < ?", channelID, cutoff).
			Delete(&ChannelProbeHistory{}).Error
	})
}

func PreviousNaturalDayBounds(now time.Time) (int64, int64) {
	localNow := now.In(time.Local)
	year, month, day := localNow.Date()
	start := time.Date(year, month, day-1, 0, 0, 0, 0, time.Local)
	return start.Unix(), start.Add(24 * time.Hour).Unix()
}

// GetPreviousDayChannelProbeSuccessRates returns percentages for the previous
// local calendar day. Channels without a record are deliberately reported as
// 100%, which keeps a newly created channel from being penalized before its
// first completed day.
func GetPreviousDayChannelProbeSuccessRates(channelIDs []int, now time.Time) (map[int]float64, error) {
	rates := make(map[int]float64, len(channelIDs))
	for _, channelID := range channelIDs {
		if channelID > 0 {
			rates[channelID] = 100
		}
	}
	if len(rates) == 0 {
		return rates, nil
	}
	if DB == nil || !DB.Migrator().HasTable(&ChannelProbeHistory{}) {
		return rates, nil
	}
	start, end := PreviousNaturalDayBounds(now)
	var history []ChannelProbeHistory
	if err := DB.Where("channel_id IN ? AND checked_at >= ? AND checked_at < ?", channelIDs, start, end).
		Find(&history).Error; err != nil {
		return nil, err
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
		if count > 0 {
			rates[channelID] = float64(successes[channelID]) / float64(count) * 100
		}
	}
	return rates, nil
}

func GetPreviousDayChannelProbeSuccessRate(channelID int, now time.Time) (float64, error) {
	rates, err := GetPreviousDayChannelProbeSuccessRates([]int{channelID}, now)
	if err != nil {
		return 0, err
	}
	return rates[channelID], nil
}
