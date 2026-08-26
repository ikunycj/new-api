package model

import "time"

type ChannelTokenUsage struct {
	DailyTokens   int64 `json:"daily_tokens"`
	MonthlyTokens int64 `json:"monthly_tokens"`
}

func GetChannelTokenUsageAt(channelIDs []int, now time.Time) (map[int]ChannelTokenUsage, error) {
	usageByChannel := make(map[int]ChannelTokenUsage, len(channelIDs))
	if len(channelIDs) == 0 {
		return usageByChannel, nil
	}

	localNow := now.In(time.Local)
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.Local)
	monthStart := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, time.Local)
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	type usageRow struct {
		ChannelID     int   `gorm:"column:channel_id"`
		DailyTokens   int64 `gorm:"column:daily_tokens"`
		MonthlyTokens int64 `gorm:"column:monthly_tokens"`
	}

	var rows []usageRow
	if err := LOG_DB.Table("logs").
		Select(`channel_id,
			COALESCE(SUM(prompt_tokens + completion_tokens) FILTER (WHERE created_at >= ?), 0) AS daily_tokens,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS monthly_tokens`, todayStart.Unix()).
		Where("channel_id IN ?", channelIDs).
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? AND created_at < ?", monthStart.Unix(), tomorrowStart.Unix()).
		Group("channel_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		usageByChannel[row.ChannelID] = ChannelTokenUsage{
			DailyTokens:   row.DailyTokens,
			MonthlyTokens: row.MonthlyTokens,
		}
	}
	return usageByChannel, nil
}
