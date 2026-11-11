package model

import (
	"sync"
	"time"
)

type ChannelTokenUsage struct {
	DailyTokens   int64 `json:"daily_tokens"`
	MonthlyTokens int64 `json:"monthly_tokens"`
}

// channelTokenUsageCacheTTL bounds how often the channel list recomputes the
// month-to-date rollup. quota_data itself is only flushed every few minutes,
// so a short cache costs no real freshness.
const channelTokenUsageCacheTTL = time.Minute

var channelTokenUsageCache struct {
	mu        sync.Mutex
	monthKey  int64
	dayKey    int64
	expiresAt time.Time
	usage     map[int]ChannelTokenUsage
}

// GetChannelTokenUsageAt returns today's and this month's token usage per
// channel. It reads the hourly quota_data rollup instead of the raw logs
// table: summing a month of logs on every channel-list request saturated the
// disk and stalled the whole site. Hour buckets align with local midnight for
// whole-hour timezone offsets, so day/month totals stay exact.
func GetChannelTokenUsageAt(channelIDs []int, now time.Time) (map[int]ChannelTokenUsage, error) {
	usageByChannel := make(map[int]ChannelTokenUsage, len(channelIDs))
	if len(channelIDs) == 0 {
		return usageByChannel, nil
	}

	all, err := loadAllChannelTokenUsage(now)
	if err != nil {
		return nil, err
	}
	for _, id := range channelIDs {
		if usage, ok := all[id]; ok {
			usageByChannel[id] = usage
		}
	}
	return usageByChannel, nil
}

// loadAllChannelTokenUsage computes usage for every channel at once so list,
// search and tag views share one cached query. The mutex also collapses
// concurrent refreshes into a single database round trip.
func loadAllChannelTokenUsage(now time.Time) (map[int]ChannelTokenUsage, error) {
	localNow := now.In(time.Local)
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.Local)
	monthStart := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, time.Local)

	cache := &channelTokenUsageCache
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.usage != nil && cache.dayKey == todayStart.Unix() && cache.monthKey == monthStart.Unix() && now.Before(cache.expiresAt) {
		return cache.usage, nil
	}

	type usageRow struct {
		ChannelID     int   `gorm:"column:channel_id"`
		DailyTokens   int64 `gorm:"column:daily_tokens"`
		MonthlyTokens int64 `gorm:"column:monthly_tokens"`
	}

	var rows []usageRow
	if err := DB.Table("quota_data").
		Select(`channel_id,
			COALESCE(SUM(CASE WHEN created_at >= ? THEN token_used ELSE 0 END), 0) AS daily_tokens,
			COALESCE(SUM(token_used), 0) AS monthly_tokens`, todayStart.Unix()).
		Where("created_at >= ?", monthStart.Unix()).
		Where("channel_id > 0").
		Group("channel_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	usage := make(map[int]ChannelTokenUsage, len(rows))
	for _, row := range rows {
		usage[row.ChannelID] = ChannelTokenUsage{
			DailyTokens:   row.DailyTokens,
			MonthlyTokens: row.MonthlyTokens,
		}
	}
	cache.usage = usage
	cache.dayKey = todayStart.Unix()
	cache.monthKey = monthStart.Unix()
	cache.expiresAt = now.Add(channelTokenUsageCacheTTL)
	return usage, nil
}

func resetChannelTokenUsageCache() {
	channelTokenUsageCache.mu.Lock()
	channelTokenUsageCache.usage = nil
	channelTokenUsageCache.mu.Unlock()
}
