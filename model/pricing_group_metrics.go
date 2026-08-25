package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
)

type PricingGroupUsagePeriod struct {
	Tokens int64 `json:"tokens"`
	Quota  int64 `json:"quota"`
}

type PricingGroupUsage struct {
	Today     PricingGroupUsagePeriod `json:"today"`
	Yesterday PricingGroupUsagePeriod `json:"yesterday"`
	Total     PricingGroupUsagePeriod `json:"total"`
}

type PricingGroupChannelCount struct {
	Available int `json:"available"`
	Total     int `json:"total"`
}

func GetPricingGroupUsageAt(now time.Time) (map[string]PricingGroupUsage, error) {
	localNow := now.In(time.Local)
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.Local)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	type usageRow struct {
		PricingGroup   string `gorm:"column:pricing_group"`
		TokensToday    int64  `gorm:"column:tokens_today"`
		QuotaToday     int64  `gorm:"column:quota_today"`
		TokensYesterday int64 `gorm:"column:tokens_yesterday"`
		QuotaYesterday int64  `gorm:"column:quota_yesterday"`
		TokensTotal    int64  `gorm:"column:tokens_total"`
		QuotaTotal     int64  `gorm:"column:quota_total"`
	}

	var rows []usageRow
	selectClause := logGroupCol + ` AS pricing_group,
		COALESCE(SUM(prompt_tokens + completion_tokens) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS tokens_today,
		COALESCE(SUM(quota) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS quota_today,
		COALESCE(SUM(prompt_tokens + completion_tokens) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS tokens_yesterday,
		COALESCE(SUM(quota) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS quota_yesterday,
		COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS tokens_total,
		COALESCE(SUM(quota), 0) AS quota_total`
	if err := LOG_DB.Table("logs").
		Select(
			selectClause,
			todayStart.Unix(), tomorrowStart.Unix(),
			todayStart.Unix(), tomorrowStart.Unix(),
			yesterdayStart.Unix(), todayStart.Unix(),
			yesterdayStart.Unix(), todayStart.Unix(),
		).
		Where("type = ?", LogTypeConsume).
		Where(logGroupCol + " <> ''").
		Group(logGroupCol).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	usageByGroup := make(map[string]PricingGroupUsage, len(rows))
	for _, row := range rows {
		usageByGroup[row.PricingGroup] = PricingGroupUsage{
			Today: PricingGroupUsagePeriod{
				Tokens: row.TokensToday,
				Quota:  row.QuotaToday,
			},
			Yesterday: PricingGroupUsagePeriod{
				Tokens: row.TokensYesterday,
				Quota:  row.QuotaYesterday,
			},
			Total: PricingGroupUsagePeriod{
				Tokens: row.TokensTotal,
				Quota:  row.QuotaTotal,
			},
		}
	}
	return usageByGroup, nil
}

func GetPricingGroupChannelCounts() (map[string]PricingGroupChannelCount, error) {
	var channels []Channel
	if err := DB.Model(&Channel{}).
		Select("id", "status", commonGroupCol).
		Find(&channels).Error; err != nil {
		return nil, err
	}

	counts := make(map[string]PricingGroupChannelCount)
	for _, channel := range channels {
		for _, group := range channel.GetGroups() {
			count := counts[group]
			count.Total++
			if channel.Status == common.ChannelStatusEnabled {
				count.Available++
			}
			counts[group] = count
		}
	}
	return counts, nil
}
