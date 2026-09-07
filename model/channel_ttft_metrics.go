package model

import (
	"math"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	ChannelTestTokenName  = "模型测试"
	ChannelProbeTokenName = "渠道探测"
)

// GetLatestChannelTestTTFTs returns the most recent valid TTFT recorded by a
// manual model test or an automatic channel probe for each requested channel.
// Test logs are ordered newest-first so only the first valid sample per channel
// is retained. Channels without a valid sample retain the zero-value result.
func GetLatestChannelTestTTFTs(channelIDs []int) (map[int]float64, error) {
	ttfts := make(map[int]float64, len(channelIDs))
	if len(channelIDs) == 0 || LOG_DB == nil || !LOG_DB.Migrator().HasTable(&Log{}) {
		return ttfts, nil
	}

	var logs []struct {
		ChannelID int    `gorm:"column:channel_id"`
		TokenName string `gorm:"column:token_name"`
		Other     string `gorm:"column:other"`
	}
	if err := LOG_DB.Table("logs").
		Select("channel_id, token_name, other").
		Where("channel_id IN ?", channelIDs).
		Where("type = ?", LogTypeConsume).
		Where("token_name IN ?", []string{ChannelTestTokenName, ChannelProbeTokenName}).
		Order("created_at DESC, id DESC").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	for _, log := range logs {
		if _, ok := ttfts[log.ChannelID]; ok {
			continue
		}
		var other struct {
			FRT float64 `json:"frt"`
		}
		if err := common.UnmarshalJsonStr(log.Other, &other); err != nil {
			continue
		}
		if other.FRT <= 0 || math.IsNaN(other.FRT) || math.IsInf(other.FRT, 0) {
			continue
		}
		ttfts[log.ChannelID] = other.FRT
	}
	return ttfts, nil
}

// GetPreviousDayChannelAverageTTFTs returns the average time to first token in
// milliseconds for each channel during the previous local calendar day.
// Only streaming consume logs with a valid positive frt value are included.
// Channels without a valid sample retain the zero-value result.
func GetPreviousDayChannelAverageTTFTs(channelIDs []int, now time.Time) (map[int]float64, error) {
	averages := make(map[int]float64, len(channelIDs))
	if len(channelIDs) == 0 || LOG_DB == nil || !LOG_DB.Migrator().HasTable(&Log{}) {
		return averages, nil
	}

	start, end := PreviousNaturalDayBounds(now)
	var logs []struct {
		ChannelID int    `gorm:"column:channel_id"`
		Other     string `gorm:"column:other"`
	}
	if err := LOG_DB.Table("logs").
		Select("channel_id, other").
		Where("channel_id IN ?", channelIDs).
		Where("type = ?", LogTypeConsume).
		Where("is_stream = ?", true).
		Where("created_at >= ? AND created_at < ?", start, end).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	totals := make(map[int]float64, len(logs))
	counts := make(map[int]int, len(logs))
	for _, log := range logs {
		var other struct {
			FRT float64 `json:"frt"`
		}
		if err := common.UnmarshalJsonStr(log.Other, &other); err != nil {
			continue
		}
		if other.FRT <= 0 || math.IsNaN(other.FRT) || math.IsInf(other.FRT, 0) {
			continue
		}
		totals[log.ChannelID] += other.FRT
		counts[log.ChannelID]++
	}

	for channelID, count := range counts {
		if count > 0 {
			averages[channelID] = totals[channelID] / float64(count)
		}
	}
	return averages, nil
}
