package model

import (
	"errors"
	"time"
)

// TokenUsageMetrics contains usage recorded for one API key. Quota values are
// kept in the same raw units as Log. The frontend converts them using the
// configured wallet display rules.
type TokenUsageMetrics struct {
	DailyTokens int64
	TotalTokens int64
	DailyQuota  int64
	TotalQuota  int64
}

// GetTokenUsageMetricsAt returns today's and lifetime usage for the requested
// API keys. Lifetime means all consume logs still present in the log database.
// Day boundaries use the local calendar day represented by now.
func GetTokenUsageMetricsAt(tokenIDs []int, now time.Time) (map[int]TokenUsageMetrics, error) {
	usageByToken := make(map[int]TokenUsageMetrics, len(tokenIDs))
	if len(tokenIDs) == 0 {
		return usageByToken, nil
	}
	if LOG_DB == nil {
		return nil, errors.New("log database is not initialized")
	}

	// Deduplicate IDs before constructing the IN list. This keeps the query
	// stable when a caller accidentally supplies the same key more than once.
	uniqueTokenIDs := make([]int, 0, len(tokenIDs))
	seen := make(map[int]struct{}, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		if tokenID <= 0 {
			continue
		}
		if _, exists := seen[tokenID]; exists {
			continue
		}
		seen[tokenID] = struct{}{}
		uniqueTokenIDs = append(uniqueTokenIDs, tokenID)
		usageByToken[tokenID] = TokenUsageMetrics{}
	}
	if len(uniqueTokenIDs) == 0 {
		return usageByToken, nil
	}

	localNow := now.In(time.Local)
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.Local)
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	type usageRow struct {
		TokenID     int   `gorm:"column:token_id"`
		DailyTokens int64 `gorm:"column:daily_tokens"`
		TotalTokens int64 `gorm:"column:total_tokens"`
		DailyQuota  int64 `gorm:"column:daily_quota"`
		TotalQuota  int64 `gorm:"column:total_quota"`
	}

	var rows []usageRow
	query := LOG_DB.Table("logs").
		Select(`token_id,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN CAST(prompt_tokens AS BIGINT) + CAST(completion_tokens AS BIGINT) ELSE 0 END), 0) AS daily_tokens,
			COALESCE(SUM(CAST(prompt_tokens AS BIGINT) + CAST(completion_tokens AS BIGINT)), 0) AS total_tokens,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN CAST(quota AS BIGINT) ELSE 0 END), 0) AS daily_quota,
			COALESCE(SUM(CAST(quota AS BIGINT)), 0) AS total_quota`, todayStart.Unix(), tomorrowStart.Unix(), todayStart.Unix(), tomorrowStart.Unix()).
		Where("token_id IN ?", uniqueTokenIDs).
		Where("type = ?", LogTypeConsume).
		Group("token_id")
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		usageByToken[row.TokenID] = TokenUsageMetrics{
			DailyTokens: row.DailyTokens,
			TotalTokens: row.TotalTokens,
			DailyQuota:  row.DailyQuota,
			TotalQuota:  row.TotalQuota,
		}
	}
	return usageByToken, nil
}
