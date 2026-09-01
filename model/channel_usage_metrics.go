package model

import "time"

type ChannelTokenUsage struct {
	DailyTokens int64 `json:"daily_tokens"`
	TotalTokens int64 `json:"total_tokens"`
	DailyQuota  int64 `json:"-"`
	TotalQuota  int64 `json:"-"`
}

// GetChannelTokenUsageAt returns today's and lifetime usage recorded in consume
// logs. Day boundaries use the local calendar day represented by now.
func GetChannelTokenUsageAt(channelIDs []int, now time.Time) (map[int]ChannelTokenUsage, error) {
	usageByChannel := make(map[int]ChannelTokenUsage, len(channelIDs))
	if len(channelIDs) == 0 {
		return usageByChannel, nil
	}

	localNow := now.In(time.Local)
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.Local)
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	type usageRow struct {
		ChannelID   int   `gorm:"column:channel_id"`
		DailyTokens int64 `gorm:"column:daily_tokens"`
		TotalTokens int64 `gorm:"column:total_tokens"`
		DailyQuota  int64 `gorm:"column:daily_quota"`
		TotalQuota  int64 `gorm:"column:total_quota"`
	}

	var rows []usageRow
	if err := LOG_DB.Table("logs").
		Select(`channel_id,
			COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS daily_tokens,
			COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint), 0) AS total_tokens,
			COALESCE(SUM(quota::bigint) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS daily_quota,
			COALESCE(SUM(quota::bigint), 0) AS total_quota`, todayStart.Unix(), tomorrowStart.Unix(), todayStart.Unix(), tomorrowStart.Unix()).
		Where("channel_id IN ?", channelIDs).
		Where("type = ?", LogTypeConsume).
		Group("channel_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		usageByChannel[row.ChannelID] = ChannelTokenUsage{
			DailyTokens: row.DailyTokens,
			TotalTokens: row.TotalTokens,
			DailyQuota:  row.DailyQuota,
			TotalQuota:  row.TotalQuota,
		}
	}
	return usageByChannel, nil
}
