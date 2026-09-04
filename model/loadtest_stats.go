package model

import (
	"sort"
)

// LoadTestChannelStats is the usage summary for one channel in a load-test run.
// It is deliberately based on consume logs, so failed attempts are not counted
// as successful upstream token usage.
type LoadTestChannelStats struct {
	ChannelID        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	BillingGroup     string  `json:"billing_group"`
	CostFactor       float64 `json:"cost_factor"`
	Requests         int64   `json:"requests"`
	InputTokens      int64   `json:"input_tokens"`
	InputTokensTotal int64   `json:"input_tokens_total"`
	OutputTokens     int64   `json:"output_tokens"`
	CacheReadTokens  int64   `json:"cache_read_tokens"`
	CacheWriteTokens int64   `json:"cache_write_tokens"`
}

// GetLoadTestChannelStatsByRunID returns consume-log usage correlated with a
// load-test run. The run ID is written into the consume log by RecordConsumeLog
// from the validated client trace header.
func GetLoadTestChannelStatsByRunID(userID int, runID string) ([]LoadTestChannelStats, error) {
	if userID <= 0 || runID == "" {
		return []LoadTestChannelStats{}, nil
	}
	var rows []loadTestChannelStatsRow
	err := LOG_DB.Table("logs").
		Select(loadTestChannelStatsSelect()).
		Where("user_id = ? AND type = ? AND other LIKE ?", userID, LogTypeConsume, "%\"load_test_run_id\":\""+runID+"\"%").
		Group("channel_id, " + logGroupCol).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return buildLoadTestChannelStats(rows)
}

func GetLoadTestChannelStats(userID int, requestIDs []string) ([]LoadTestChannelStats, error) {
	if userID <= 0 || len(requestIDs) == 0 {
		return []LoadTestChannelStats{}, nil
	}

	var rows []loadTestChannelStatsRow
	err := LOG_DB.Table("logs").
		Select(loadTestChannelStatsSelect()).
		Where("user_id = ? AND type = ? AND (request_id IN ? OR upstream_request_id IN ?)", userID, LogTypeConsume, requestIDs, requestIDs).
		Group("channel_id, " + logGroupCol).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return buildLoadTestChannelStats(rows)
}

type loadTestChannelStatsRow struct {
	ChannelID        int
	BillingGroup     string
	Requests         int64
	InputTokens      int64
	InputTokensTotal int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
}

type LoadTestTokenStats struct {
	Requests         int64 `json:"requests"`
	InputTokens      int64 `json:"input_tokens"`
	InputTokensTotal int64 `json:"input_tokens_total"`
	OutputTokens     int64 `json:"output_tokens"`
	CacheReadTokens  int64 `json:"cache_read_tokens"`
	CacheWriteTokens int64 `json:"cache_write_tokens"`
}

func GetLoadTestTokenStatsByRunID(userID int, runID string) (LoadTestTokenStats, error) {
	stats, err := GetLoadTestChannelStatsByRunID(userID, runID)
	if err != nil {
		return LoadTestTokenStats{}, err
	}
	result := LoadTestTokenStats{}
	for _, item := range stats {
		result.Requests += item.Requests
		result.InputTokens += item.InputTokens
		result.InputTokensTotal += item.InputTokensTotal
		result.OutputTokens += item.OutputTokens
		result.CacheReadTokens += item.CacheReadTokens
		result.CacheWriteTokens += item.CacheWriteTokens
	}
	return result, nil
}

func loadTestChannelStatsSelect() string {
	return "channel_id, " + logGroupCol + " AS billing_group, COUNT(*) AS requests, COALESCE(SUM(prompt_tokens), 0) AS input_tokens, COALESCE(SUM(input_tokens_total), 0) AS input_tokens_total, COALESCE(SUM(completion_tokens), 0) AS output_tokens, COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens, COALESCE(SUM(cache_write_tokens), 0) AS cache_write_tokens"
}

func buildLoadTestChannelStats(rows []loadTestChannelStatsRow) ([]LoadTestChannelStats, error) {

	channelIDs := make([]int, 0, len(rows))
	for _, item := range rows {
		if item.ChannelID > 0 {
			channelIDs = append(channelIDs, item.ChannelID)
		}
	}
	type channelMetadata struct {
		Id   int
		Name string
	}
	var channels []channelMetadata
	if len(channelIDs) > 0 {
		if err := DB.Model(&Channel{}).Select("id, name").Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
			return nil, err
		}
	}
	metadataByID := make(map[int]channelMetadata, len(channels))
	for _, channel := range channels {
		metadataByID[channel.Id] = channel
	}

	stats := make([]LoadTestChannelStats, 0, len(rows))
	for _, item := range rows {
		channel := metadataByID[item.ChannelID]
		costFactor := ResolveChannelCostFactor(item.BillingGroup, item.ChannelID)
		stats = append(stats, LoadTestChannelStats{
			ChannelID:        item.ChannelID,
			ChannelName:      channel.Name,
			BillingGroup:     item.BillingGroup,
			CostFactor:       costFactor,
			Requests:         item.Requests,
			InputTokens:      item.InputTokens,
			InputTokensTotal: item.InputTokensTotal,
			OutputTokens:     item.OutputTokens,
			CacheReadTokens:  item.CacheReadTokens,
			CacheWriteTokens: item.CacheWriteTokens,
		})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Requests != stats[j].Requests {
			return stats[i].Requests > stats[j].Requests
		}
		return stats[i].ChannelID < stats[j].ChannelID
	})
	return stats, nil
}
