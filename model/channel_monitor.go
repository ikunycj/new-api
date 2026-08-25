package model

import (
	"math"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const channelMonitorHistoryRetention = 31 * 24 * time.Hour

type ChannelMonitor struct {
	Id                       int                     `json:"id"`
	PricingGroup             string                  `json:"pricing_group" gorm:"size:100;not null;uniqueIndex"`
	TestModel                string                  `json:"test_model" gorm:"size:200;not null"`
	IntervalSeconds          int                     `json:"interval_seconds" gorm:"not null"`
	TimeoutSeconds           int                     `json:"timeout_seconds" gorm:"not null"`
	Enabled                  bool                    `json:"enabled" gorm:"index:idx_channel_monitor_due,priority:1;not null"`
	Visible                  bool                    `json:"visible" gorm:"index;not null"`
	AvailabilityBoostPercent float64                 `json:"availability_boost_percent" gorm:"column:availability_boost_percent"`
	UserTestAvailableAt      *int64                  `json:"-" gorm:"column:user_test_available_at;bigint"`
	LastCheckedAt            *int64                  `json:"last_checked_at" gorm:"bigint"`
	NextCheckAt              *int64                  `json:"next_check_at" gorm:"bigint;index:idx_channel_monitor_due,priority:2"`
	LeaseExpiresAt           *int64                  `json:"-" gorm:"bigint;index:idx_channel_monitor_due,priority:3"`
	CreatedBy                int                     `json:"created_by" gorm:"index"`
	CreatedAt                int64                   `json:"created_at" gorm:"bigint"`
	UpdatedAt                int64                   `json:"updated_at" gorm:"bigint"`
	Histories                []ChannelMonitorHistory `json:"-" gorm:"foreignKey:MonitorId;references:Id;constraint:OnDelete:CASCADE"`
}

type ChannelMonitorHistory struct {
	Id           int64  `json:"id"`
	MonitorId    int    `json:"monitor_id" gorm:"index:idx_channel_monitor_history,priority:1;not null"`
	Success      bool   `json:"success" gorm:"not null"`
	LatencyMs    int    `json:"latency_ms" gorm:"not null"`
	StatusCode   int    `json:"status_code" gorm:"not null"`
	ErrorMessage string `json:"error_message,omitempty" gorm:"type:text"`
	CheckedAt    int64  `json:"checked_at" gorm:"bigint;index:idx_channel_monitor_history,priority:2;index;not null"`
}

func (monitor *ChannelMonitor) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if monitor.CreatedAt == 0 {
		monitor.CreatedAt = now
	}
	monitor.UpdatedAt = now
	return nil
}

func (monitor *ChannelMonitor) BeforeUpdate(_ *gorm.DB) error {
	monitor.UpdatedAt = common.GetTimestamp()
	return nil
}

func CreateChannelMonitor(monitor *ChannelMonitor) error {
	return DB.Create(monitor).Error
}

func UpdateChannelMonitor(monitor *ChannelMonitor) error {
	monitor.UpdatedAt = common.GetTimestamp()
	return DB.Model(&ChannelMonitor{}).
		Where("id = ?", monitor.Id).
		Updates(map[string]any{
			"test_model":                 monitor.TestModel,
			"interval_seconds":           monitor.IntervalSeconds,
			"timeout_seconds":            monitor.TimeoutSeconds,
			"enabled":                    monitor.Enabled,
			"visible":                    monitor.Visible,
			"availability_boost_percent": monitor.AvailabilityBoostPercent,
			"next_check_at":              monitor.NextCheckAt,
			"lease_expires_at":           nil,
			"updated_at":                 monitor.UpdatedAt,
		}).Error
}

func ClaimChannelMonitorUserTest(id int, now int64, leaseUntil int64) (bool, error) {
	result := DB.Model(&ChannelMonitor{}).
		Where("id = ? AND enabled = ? AND visible = ? AND (user_test_available_at IS NULL OR user_test_available_at <= ?)", id, true, true, now).
		Update("user_test_available_at", leaseUntil)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func CompleteChannelMonitorUserTest(id int, availableAt int64) error {
	return DB.Model(&ChannelMonitor{}).
		Where("id = ?", id).
		Update("user_test_available_at", availableAt).Error
}

// DeleteChannelMonitorsOutsidePricingGroups keeps monitor ownership aligned
// with the current pricing-group catalog inside the caller's transaction.
func DeleteChannelMonitorsOutsidePricingGroups(tx *gorm.DB, pricingGroups []string) error {
	query := tx.Model(&ChannelMonitor{}).Select("id")
	if len(pricingGroups) > 0 {
		query = query.Where("pricing_group NOT IN ?", pricingGroups)
	}
	var monitorIDs []int
	if err := query.Pluck("id", &monitorIDs).Error; err != nil {
		return err
	}
	if len(monitorIDs) == 0 {
		return nil
	}
	if err := tx.Where("monitor_id IN ?", monitorIDs).Delete(&ChannelMonitorHistory{}).Error; err != nil {
		return err
	}
	return tx.Where("id IN ?", monitorIDs).Delete(&ChannelMonitor{}).Error
}

func GetChannelMonitorByID(id int) (*ChannelMonitor, error) {
	var monitor ChannelMonitor
	if err := DB.First(&monitor, id).Error; err != nil {
		return nil, err
	}
	return &monitor, nil
}

func ListChannelMonitors(visibleOnly bool) ([]*ChannelMonitor, error) {
	monitors := make([]*ChannelMonitor, 0)
	query := DB.Model(&ChannelMonitor{})
	if visibleOnly {
		query = query.Where("visible = ?", true)
	}
	if err := query.Order("id ASC").Find(&monitors).Error; err != nil {
		return nil, err
	}
	return monitors, nil
}

func ClaimDueChannelMonitors(now int64, leaseSeconds int64, limit int) ([]*ChannelMonitor, error) {
	if limit <= 0 {
		return []*ChannelMonitor{}, nil
	}

	candidates := make([]*ChannelMonitor, 0, limit)
	err := DB.Where("enabled = ? AND (next_check_at IS NULL OR next_check_at <= ?) AND (lease_expires_at IS NULL OR lease_expires_at <= ?)", true, now, now).
		Order("next_check_at ASC").
		Order("id ASC").
		Limit(limit).
		Find(&candidates).Error
	if err != nil {
		return nil, err
	}

	claimed := make([]*ChannelMonitor, 0, len(candidates))
	for _, monitor := range candidates {
		result := DB.Model(&ChannelMonitor{}).
			Where("id = ? AND enabled = ? AND (next_check_at IS NULL OR next_check_at <= ?) AND (lease_expires_at IS NULL OR lease_expires_at <= ?)", monitor.Id, true, now, now).
			Updates(map[string]any{
				"lease_expires_at": now + leaseSeconds,
				"updated_at":       now,
			})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 1 {
			leaseUntil := now + leaseSeconds
			monitor.LeaseExpiresAt = &leaseUntil
			claimed = append(claimed, monitor)
		}
	}
	return claimed, nil
}

func SaveChannelMonitorResult(monitor *ChannelMonitor, result *ChannelMonitorHistory) error {
	nextCheckAt := result.CheckedAt + int64(monitor.IntervalSeconds)
	return DB.Transaction(func(tx *gorm.DB) error {
		update := tx.Model(&ChannelMonitor{}).
			Where("id = ?", monitor.Id).
			Updates(map[string]any{
				"last_checked_at":  result.CheckedAt,
				"next_check_at":    nextCheckAt,
				"lease_expires_at": nil,
				"updated_at":       result.CheckedAt,
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Create(result).Error; err != nil {
			return err
		}
		cutoff := result.CheckedAt - int64(channelMonitorHistoryRetention/time.Second)
		return tx.Where("monitor_id = ? AND checked_at < ?", monitor.Id, cutoff).
			Delete(&ChannelMonitorHistory{}).Error
	})
}

func ReleaseChannelMonitorLease(id int, nextCheckAt int64) error {
	return DB.Model(&ChannelMonitor{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"lease_expires_at": nil,
			"next_check_at":    nextCheckAt,
			"updated_at":       common.GetTimestamp(),
		}).Error
}

func GetLatestChannelMonitorHistory(monitorId int) (*ChannelMonitorHistory, error) {
	var history ChannelMonitorHistory
	err := DB.Where("monitor_id = ?", monitorId).
		Order("checked_at DESC").
		Order("id DESC").
		First(&history).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

func ListChannelMonitorHistory(monitorId int, limit int) ([]*ChannelMonitorHistory, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	history := make([]*ChannelMonitorHistory, 0, limit)
	err := DB.Where("monitor_id = ?", monitorId).
		Order("checked_at DESC").
		Order("id DESC").
		Limit(limit).
		Find(&history).Error
	return history, err
}

func GetChannelMonitorAvailability(monitorId int, since int64) (*float64, error) {
	query := DB.Model(&ChannelMonitorHistory{}).
		Where("monitor_id = ? AND checked_at >= ?", monitorId, since)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, nil
	}
	var succeeded int64
	if err := query.Where("success = ?", true).Count(&succeeded).Error; err != nil {
		return nil, err
	}
	availability := math.Round(float64(succeeded)/float64(total)*10000) / 100
	return &availability, nil
}
