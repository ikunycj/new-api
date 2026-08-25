package service

import (
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
)

const pricingGroupBaseMetricsCacheTTL = 10 * time.Second

type PricingGroupMetrics struct {
	PricingGroup string                         `json:"pricing_group"`
	Usage        model.PricingGroupUsage        `json:"usage"`
	Channels     model.PricingGroupChannelCount `json:"channels"`
	Activity     PricingGroupActivity            `json:"activity"`
	UpdatedAt    int64                           `json:"updated_at"`
}

type pricingGroupBaseMetrics struct {
	usageByGroup   map[string]model.PricingGroupUsage
	channelByGroup map[string]model.PricingGroupChannelCount
}

var pricingGroupBaseMetricsCache = struct {
	sync.Mutex
	groupKey  string
	expiresAt time.Time
	metrics   pricingGroupBaseMetrics
}{}

func getPricingGroupBaseMetrics(groups []string, now time.Time) (pricingGroupBaseMetrics, error) {
	groupKey := strings.Join(groups, "\x00")
	pricingGroupBaseMetricsCache.Lock()
	defer pricingGroupBaseMetricsCache.Unlock()
	if pricingGroupBaseMetricsCache.groupKey == groupKey &&
		now.Before(pricingGroupBaseMetricsCache.expiresAt) {
		return pricingGroupBaseMetricsCache.metrics, nil
	}

	usageByGroup, err := model.GetPricingGroupUsageAt(now)
	if err != nil {
		return pricingGroupBaseMetrics{}, err
	}
	channelByGroup, err := model.GetPricingGroupChannelCounts()
	if err != nil {
		return pricingGroupBaseMetrics{}, err
	}
	metrics := pricingGroupBaseMetrics{
		usageByGroup:   usageByGroup,
		channelByGroup: channelByGroup,
	}
	pricingGroupBaseMetricsCache.groupKey = groupKey
	pricingGroupBaseMetricsCache.expiresAt = now.Add(pricingGroupBaseMetricsCacheTTL)
	pricingGroupBaseMetricsCache.metrics = metrics
	return metrics, nil
}

func GetPricingGroupMetrics(groups []string) ([]PricingGroupMetrics, error) {
	now := time.Now()
	baseMetrics, err := getPricingGroupBaseMetrics(groups, now)
	if err != nil {
		return nil, err
	}
	activityByGroup := GetPricingGroupActivity(groups)
	items := make([]PricingGroupMetrics, 0, len(groups))
	for _, group := range groups {
		items = append(items, PricingGroupMetrics{
			PricingGroup: group,
			Usage:        baseMetrics.usageByGroup[group],
			Channels:     baseMetrics.channelByGroup[group],
			Activity:     activityByGroup[group],
			UpdatedAt:    now.Unix(),
		})
	}
	return items, nil
}
