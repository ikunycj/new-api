package model

import (
	"context"
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

// GetChannelUsageAt performs one snapshot query for the page request.
func GetChannelUsageAt(channelIDs []int, now time.Time) (map[int]ChannelUsage, error) {
	return queryChannelUsageAt(channelIDs, now)
}

// queryChannelUsageAt includes consume logs (including probes) still retained
// in the log database. Day boundaries follow the application's local calendar day.
func queryChannelUsageAt(channelIDs []int, now time.Time) (map[int]ChannelUsage, error) {
	if !logRollupEnabled() {
		return queryChannelUsageFromLogs(channelIDs, now)
	}
	state, err := GetLogRollupState(context.Background())
	if err != nil {
		common.SysLog("failed to read log rollup state, using source logs: " + err.Error())
		return queryChannelUsageFromLogs(channelIDs, now)
	}
	historical, err := queryChannelUsageFromRollup(channelIDs, state, now)
	if err != nil {
		common.SysLog("failed to read channel usage rollup, using source logs: " + err.Error())
		return queryChannelUsageFromLogs(channelIDs, now)
	}
	// Only the uncompleted tail is read from logs. Dates before an explicit
	// aggregate cleanup boundary are intentionally omitted from the snapshot.
	rawStart := state.ReadyBefore
	if state.ExcludedBefore > rawStart {
		rawStart = state.ExcludedBefore
	}
	if todayStart := localDayStart(now); rawStart > todayStart {
		rawStart = todayStart
	}
	tail, err := queryChannelUsageFromLogsSince(channelIDs, now, rawStart)
	if err != nil {
		return nil, err
	}
	return mergeChannelUsageMaps(historical, tail), nil
}

func mergeChannelUsageMaps(left, right map[int]ChannelUsage) map[int]ChannelUsage {
	merged := make(map[int]ChannelUsage, len(left)+len(right))
	for id, usage := range left {
		merged[id] = usage
	}
	for id, usage := range right {
		current := merged[id]
		current.DailyTokens += usage.DailyTokens
		current.TotalTokens += usage.TotalTokens
		current.DailyBaseCostUSD += usage.DailyBaseCostUSD
		current.TotalBaseCostUSD += usage.TotalBaseCostUSD
		current.DailyBaseCostCNY += usage.DailyBaseCostCNY
		current.TotalBaseCostCNY += usage.TotalBaseCostCNY
		current.DailyCostIncomplete = current.DailyCostIncomplete || usage.DailyCostIncomplete
		current.TotalCostIncomplete = current.TotalCostIncomplete || usage.TotalCostIncomplete
		merged[id] = current
	}
	return merged
}

func queryChannelUsageFromLogs(channelIDs []int, now time.Time) (map[int]ChannelUsage, error) {
	return queryChannelUsageFromLogsSince(channelIDs, now, 0)
}

func queryChannelUsageFromLogsSince(channelIDs []int, now time.Time, startTimestamp int64) (map[int]ChannelUsage, error) {
	usageByChannel := make(map[int]ChannelUsage, len(channelIDs))
	if len(channelIDs) == 0 {
		return usageByChannel, nil
	}
	localNow := now.In(time.Local)
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.Local)
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	type usageRow struct {
		ChannelID   int
		Other       string
		DailyCount  int64
		DailyTokens int64
		TotalTokens int64
		DailyQuota  int64
		TotalQuota  int64
	}
	var rows []usageRow
	// Group by frozen pricing so historical group/rate changes and different
	// probe pricing remain independent, without loading every log into memory.
	query := LOG_DB.Table("logs").
		Where("channel_id IN ?", channelIDs).
		Where("type = ?", LogTypeConsume)
	if startTimestamp > 0 {
		query = query.Where("created_at >= ? AND created_at < ?", startTimestamp, tomorrowStart.Unix())
	}
	if err := query.
		Select(`channel_id,
			other,
			COUNT(*) FILTER (WHERE created_at >= ? AND created_at < ?) AS daily_count,
			COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS daily_tokens,
            COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint), 0) AS total_tokens,
            COALESCE(SUM(quota::bigint) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS daily_quota,
            COALESCE(SUM(quota::bigint), 0) AS total_quota`,
			todayStart.Unix(), tomorrowStart.Unix(),
			todayStart.Unix(), tomorrowStart.Unix(),
			todayStart.Unix(), tomorrowStart.Unix()).
		Group("channel_id, other").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		usage := usageByChannel[row.ChannelID]
		usage.DailyTokens += row.DailyTokens
		usage.TotalTokens += row.TotalTokens
		groupRatio, billingRate, incomplete := parseFrozenPricing(row.Other)
		if incomplete || row.DailyQuota < 0 || row.TotalQuota < 0 {
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
