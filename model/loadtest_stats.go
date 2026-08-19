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
	ClusterID        int     `json:"cluster_id"`
	PoolName         string  `json:"pool_name"`
	CostFactor       float64 `json:"cost_factor"`
	Requests         int64   `json:"requests"`
	InputTokens      int64   `json:"input_tokens"`
	InputTokensTotal int64   `json:"input_tokens_total"`
	OutputTokens     int64   `json:"output_tokens"`
	CacheReadTokens  int64   `json:"cache_read_tokens"`
	CacheWriteTokens int64   `json:"cache_write_tokens"`
}

func GetLoadTestChannelStats(userID int, requestIDs []string) ([]LoadTestChannelStats, error) {
	if userID <= 0 || len(requestIDs) == 0 {
		return []LoadTestChannelStats{}, nil
	}

	type row struct {
		ChannelID        int
		Requests         int64
		InputTokens      int64
		InputTokensTotal int64
		OutputTokens     int64
		CacheReadTokens  int64
		CacheWriteTokens int64
	}
	var rows []row
	err := LOG_DB.Table("logs").
		Select("channel_id, COUNT(*) AS requests, COALESCE(SUM(prompt_tokens), 0) AS input_tokens, COALESCE(SUM(input_tokens_total), 0) AS input_tokens_total, COALESCE(SUM(completion_tokens), 0) AS output_tokens, COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens, COALESCE(SUM(cache_write_tokens), 0) AS cache_write_tokens").
		Where("user_id = ? AND type = ? AND request_id IN ?", userID, LogTypeConsume, requestIDs).
		Group("channel_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	channelIDs := make([]int, 0, len(rows))
	for _, item := range rows {
		if item.ChannelID > 0 {
			channelIDs = append(channelIDs, item.ChannelID)
		}
	}
	type channelMetadata struct {
		Id            int
		Name          string
		ClusterId     int
		ClusterPoolId int
	}
	var channels []channelMetadata
	if len(channelIDs) > 0 {
		if err := DB.Model(&Channel{}).Select("id, name, cluster_id, cluster_pool_id").Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
			return nil, err
		}
	}
	metadataByID := make(map[int]channelMetadata, len(channels))
	poolIDs := make([]int, 0, len(channels))
	for _, channel := range channels {
		metadataByID[channel.Id] = channel
		if channel.ClusterPoolId > 0 {
			poolIDs = append(poolIDs, channel.ClusterPoolId)
		}
	}
	type poolMetadata struct {
		Id         int
		Name       string
		CostFactor float64
	}
	var pools []poolMetadata
	if len(poolIDs) > 0 {
		if err := DB.Model(&ClusterPool{}).Select("id, name, cost_factor").Where("id IN ?", poolIDs).Find(&pools).Error; err != nil {
			return nil, err
		}
	}
	poolByID := make(map[int]poolMetadata, len(pools))
	for _, pool := range pools {
		poolByID[pool.Id] = pool
	}

	stats := make([]LoadTestChannelStats, 0, len(rows))
	for _, item := range rows {
		channel := metadataByID[item.ChannelID]
		pool := poolByID[channel.ClusterPoolId]
		costFactor := pool.CostFactor
		if pool.Id == 0 {
			costFactor = 1
		}
		stats = append(stats, LoadTestChannelStats{
			ChannelID:        item.ChannelID,
			ChannelName:      channel.Name,
			ClusterID:        channel.ClusterId,
			PoolName:         pool.Name,
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
