package model

import (
	"math"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// ChannelUsage separates model base charges from customer pricing. The caller
// applies the channel's current upstream multiplier to these historical bases.
type ChannelUsage struct {
	DailyTokens         int64
	TotalTokens         int64
	DailyBaseCostUSD    float64
	TotalBaseCostUSD    float64
	DailyBaseCostCNY    float64
	TotalBaseCostCNY    float64
	DailyCostIncomplete bool
	TotalCostIncomplete bool
}

// GetChannelUsageAt includes consume logs (including probes) still retained in
// the log database. Day boundaries follow the application's local calendar day.
func GetChannelUsageAt(channelIDs []int, now time.Time) (map[int]ChannelUsage, error) {
	usageByChannel := make(map[int]ChannelUsage, len(channelIDs))
	if len(channelIDs) == 0 {
		return usageByChannel, nil
	}
	localNow := now.In(time.Local)
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.Local)
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	type usageRow struct {
		ChannelID   int
		GroupRatio  string
		BillingRate string
		DailyCount  int64
		DailyTokens int64
		TotalTokens int64
		DailyQuota  int64
		TotalQuota  int64
	}
	var rows []usageRow
	// Group by frozen pricing so historical group/rate changes and different
	// probe pricing remain independent, without loading every log into memory.
	if err := LOG_DB.Table("logs").
		Select(`channel_id,
            NULLIF(BTRIM(other), '')::jsonb->>'group_ratio' AS group_ratio,
            NULLIF(BTRIM(other), '')::jsonb->>'billing_usd_to_cny_rate' AS billing_rate,
            COUNT(*) FILTER (WHERE created_at >= ? AND created_at < ?) AS daily_count,
            COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS daily_tokens,
            COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint), 0) AS total_tokens,
            COALESCE(SUM(quota::bigint) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS daily_quota,
            COALESCE(SUM(quota::bigint), 0) AS total_quota`,
			todayStart.Unix(), tomorrowStart.Unix(),
			todayStart.Unix(), tomorrowStart.Unix(),
			todayStart.Unix(), tomorrowStart.Unix()).
		Where("channel_id IN ?", channelIDs).
		Where("type = ?", LogTypeConsume).
		Group("channel_id, group_ratio, billing_rate").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		usage := usageByChannel[row.ChannelID]
		usage.DailyTokens += row.DailyTokens
		usage.TotalTokens += row.TotalTokens
		groupRatio, groupErr := strconv.ParseFloat(row.GroupRatio, 64)
		// Logs predating the billing currency setting used a conversion of 1.
		billingRate := 1.0
		var rateErr error
		if row.BillingRate != "" {
			billingRate, rateErr = strconv.ParseFloat(row.BillingRate, 64)
		}
		if groupErr != nil || rateErr != nil ||
			groupRatio <= 0 || math.IsNaN(groupRatio) || math.IsInf(groupRatio, 0) ||
			billingRate <= 0 || math.IsNaN(billingRate) || math.IsInf(billingRate, 0) ||
			common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) ||
			row.DailyQuota < 0 || row.TotalQuota < 0 {
			// Zero customer pricing cannot reveal upstream cost. Missing history
			// must not silently appear as a free upstream request.
			usage.DailyCostIncomplete = usage.DailyCostIncomplete || row.DailyCount > 0
			usage.TotalCostIncomplete = true
		} else {
			dailyCNY := float64(row.DailyQuota) / common.QuotaPerUnit / groupRatio
			totalCNY := float64(row.TotalQuota) / common.QuotaPerUnit / groupRatio
			usage.DailyBaseCostCNY += dailyCNY
			usage.TotalBaseCostCNY += totalCNY
			usage.DailyBaseCostUSD += dailyCNY / billingRate
			usage.TotalBaseCostUSD += totalCNY / billingRate
		}
		usageByChannel[row.ChannelID] = usage
	}
	return usageByChannel, nil
}
