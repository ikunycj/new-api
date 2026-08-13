package model

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	MaxChannelReconciliationRange  = 90 * 24 * time.Hour
	MaxChannelCostEntryRange       = 366 * 24 * time.Hour
	channelReconciliationBatchSize = 2000
)

var (
	ErrChannelCostEntryOverlap  = errors.New("channel cost period overlaps an existing entry")
	ErrChannelCostEntryNotFound = errors.New("channel cost entry not found")
)

type ChannelCostEntry struct {
	Id        int     `json:"id"`
	ChannelId int     `json:"channel_id" gorm:"index:idx_channel_cost_period,priority:1;not null"`
	StartAt   int64   `json:"start_at" gorm:"bigint;index:idx_channel_cost_period,priority:2;not null"`
	EndAt     int64   `json:"end_at" gorm:"bigint;index:idx_channel_cost_period,priority:3;not null"`
	AmountUSD float64 `json:"amount_usd" gorm:"type:decimal(18,6);not null"`
	Currency  string  `json:"currency" gorm:"type:varchar(8);not null"`
	Source    string  `json:"source" gorm:"type:varchar(32);not null"`
	Note      string  `json:"note" gorm:"type:varchar(500)"`
	CreatedBy int     `json:"created_by" gorm:"index;not null"`
	CreatedAt int64   `json:"created_at" gorm:"bigint;not null"`
	UpdatedAt int64   `json:"updated_at" gorm:"bigint;not null"`
}

func (entry *ChannelCostEntry) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if entry.CreatedAt == 0 {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now
	return nil
}

type ChannelReconciliationSummary struct {
	Requests            int64   `json:"requests"`
	PromptTokens        int64   `json:"prompt_tokens"`
	CompletionTokens    int64   `json:"completion_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheWriteTokens    int64   `json:"cache_write_tokens"`
	UserChargeUSD       float64 `json:"user_charge_usd"`
	EstimatedCostUSD    float64 `json:"estimated_cost_usd"`
	ActualCostUSD       float64 `json:"actual_cost_usd"`
	GrossMarginUSD      float64 `json:"gross_margin_usd"`
	EstimateVarianceUSD float64 `json:"estimate_variance_usd"`
	AverageLatencyMS    float64 `json:"average_latency_ms"`
	ActiveDays          int     `json:"active_days"`
	CostSource          string  `json:"cost_source"`
}

type ChannelReconciliationBucket struct {
	Name             string  `json:"name"`
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	CacheReadTokens  int64   `json:"cache_read_tokens"`
	CacheWriteTokens int64   `json:"cache_write_tokens"`
	UserChargeUSD    float64 `json:"user_charge_usd"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	ActualCostUSD    float64 `json:"actual_cost_usd"`
	LatencyTotalMS   float64 `json:"-"`
}

type ChannelReconciliationResult struct {
	Channel struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"channel"`
	Range struct {
		StartTimestamp int64 `json:"start_timestamp"`
		EndTimestamp   int64 `json:"end_timestamp"`
	} `json:"range"`
	Summary           ChannelReconciliationSummary  `json:"summary"`
	Daily             []ChannelReconciliationBucket `json:"daily"`
	Models            []ChannelReconciliationBucket `json:"models"`
	InboundEndpoints  []ChannelReconciliationBucket `json:"inbound_endpoints"`
	UpstreamEndpoints []ChannelReconciliationBucket `json:"upstream_endpoints"`
	CostEntries       []ChannelCostEntry            `json:"cost_entries"`
}

type channelReconciliationLogOther struct {
	CacheTokens           float64  `json:"cache_tokens"`
	CacheWriteTokens      float64  `json:"cache_write_tokens"`
	CacheCreationTokens   float64  `json:"cache_creation_tokens"`
	CacheCreationTokens5m float64  `json:"cache_creation_tokens_5m"`
	CacheCreationTokens1h float64  `json:"cache_creation_tokens_1h"`
	BillingUSDToCNYRate   float64  `json:"billing_usd_to_cny_rate"`
	GroupRatio            float64  `json:"group_ratio"`
	UserGroupRatio        float64  `json:"user_group_ratio"`
	RequestPath           string   `json:"request_path"`
	RequestConversion     []string `json:"request_conversion"`
	UpstreamRequestPath   string   `json:"upstream_request_path"`
}

func ValidateChannelReconciliationRange(startAt, endAt int64) error {
	if startAt <= 0 || endAt <= startAt {
		return errors.New("invalid reconciliation time range")
	}
	if endAt-startAt > int64(MaxChannelReconciliationRange/time.Second) {
		return errors.New("reconciliation time range cannot exceed 90 days")
	}
	return nil
}

func ValidateChannelCostEntry(entry *ChannelCostEntry) error {
	if entry == nil || entry.ChannelId <= 0 || entry.StartAt <= 0 || entry.EndAt <= entry.StartAt {
		return errors.New("invalid channel cost entry")
	}
	if entry.EndAt-entry.StartAt > int64(MaxChannelCostEntryRange/time.Second) {
		return errors.New("cost period cannot exceed 366 days")
	}
	if entry.AmountUSD < 0 || math.IsNaN(entry.AmountUSD) || math.IsInf(entry.AmountUSD, 0) {
		return errors.New("cost amount must be a finite non-negative number")
	}
	entry.Currency = strings.ToUpper(strings.TrimSpace(entry.Currency))
	if entry.Currency == "" {
		entry.Currency = "USD"
	}
	if entry.Currency != "USD" {
		return errors.New("only USD cost entries are supported")
	}
	entry.Source = strings.ToLower(strings.TrimSpace(entry.Source))
	if entry.Source == "" {
		entry.Source = "manual"
	}
	if entry.Source != "manual" && entry.Source != "imported" && entry.Source != "provider_api" {
		return errors.New("unsupported cost source")
	}
	entry.Note = strings.TrimSpace(entry.Note)
	if len(entry.Note) > 500 {
		return errors.New("cost note cannot exceed 500 characters")
	}
	return nil
}

func CreateChannelCostEntry(entry *ChannelCostEntry) error {
	if err := ValidateChannelCostEntry(entry); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var channel Channel
		if err := lockForUpdate(tx).Select("id").Where("id = ?", entry.ChannelId).First(&channel).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&ChannelCostEntry{}).
			Where("channel_id = ? AND start_at < ? AND end_at > ?", entry.ChannelId, entry.EndAt, entry.StartAt).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrChannelCostEntryOverlap
		}
		return tx.Create(entry).Error
	})
}

func DeleteChannelCostEntry(channelId, entryId int) error {
	result := DB.Where("id = ? AND channel_id = ?", entryId, channelId).Delete(&ChannelCostEntry{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrChannelCostEntryNotFound
	}
	return nil
}

func GetChannelReconciliation(channelId int, startAt, endAt int64) (*ChannelReconciliationResult, error) {
	if err := ValidateChannelReconciliationRange(startAt, endAt); err != nil {
		return nil, err
	}
	channel, err := GetChannelById(channelId, false)
	if err != nil {
		return nil, err
	}
	result := &ChannelReconciliationResult{
		Daily:             make([]ChannelReconciliationBucket, 0),
		Models:            make([]ChannelReconciliationBucket, 0),
		InboundEndpoints:  make([]ChannelReconciliationBucket, 0),
		UpstreamEndpoints: make([]ChannelReconciliationBucket, 0),
		CostEntries:       make([]ChannelCostEntry, 0),
	}
	result.Channel.Id = channel.Id
	result.Channel.Name = channel.Name
	result.Range.StartTimestamp = startAt
	result.Range.EndTimestamp = endAt

	if err := DB.Where("channel_id = ? AND start_at < ? AND end_at > ?", channelId, endAt, startAt).
		Order("start_at asc, id asc").Find(&result.CostEntries).Error; err != nil {
		return nil, err
	}

	daily := map[string]*ChannelReconciliationBucket{}
	models := map[string]*ChannelReconciliationBucket{}
	inbound := map[string]*ChannelReconciliationBucket{}
	upstream := map[string]*ChannelReconciliationBucket{}
	for offset := 0; ; offset += channelReconciliationBatchSize {
		logs := make([]Log, 0, channelReconciliationBatchSize)
		query := LOG_DB.Select("created_at", "request_id", "prompt_tokens", "completion_tokens", "quota", "use_time", "model_name", "other").
			Where("channel_id = ? AND type = ? AND created_at >= ? AND created_at < ?", channelId, LogTypeConsume, startAt, endAt).
			Order("created_at asc, request_id asc").Limit(channelReconciliationBatchSize).Offset(offset)
		if err := query.Find(&logs).Error; err != nil {
			return nil, err
		}
		for i := range logs {
			aggregateChannelReconciliationLog(&logs[i], &result.Summary, daily, models, inbound, upstream)
		}
		if len(logs) < channelReconciliationBatchSize {
			break
		}
	}

	if result.Summary.Requests > 0 {
		result.Summary.AverageLatencyMS /= float64(result.Summary.Requests)
	}
	result.Summary.ActiveDays = len(daily)
	result.Summary.CostSource = "none"
	for _, entry := range result.CostEntries {
		actual := proratedCost(entry, startAt, endAt)
		result.Summary.ActualCostUSD += actual
		allocateCostToDaily(actual, maxInt64(startAt, entry.StartAt), minInt64(endAt, entry.EndAt), daily)
	}
	if len(result.CostEntries) > 0 {
		result.Summary.CostSource = "manual"
	}
	result.Summary.GrossMarginUSD = result.Summary.UserChargeUSD - result.Summary.ActualCostUSD
	result.Summary.EstimateVarianceUSD = result.Summary.ActualCostUSD - result.Summary.EstimatedCostUSD
	allocateActualCost(result.Summary.ActualCostUSD, models)
	allocateActualCost(result.Summary.ActualCostUSD, inbound)
	allocateActualCost(result.Summary.ActualCostUSD, upstream)

	result.Daily = sortedBuckets(daily, false)
	result.Models = sortedBuckets(models, true)
	result.InboundEndpoints = sortedBuckets(inbound, true)
	result.UpstreamEndpoints = sortedBuckets(upstream, true)
	return result, nil
}

func aggregateChannelReconciliationLog(log *Log, summary *ChannelReconciliationSummary, bucketMaps ...map[string]*ChannelReconciliationBucket) {
	var other channelReconciliationLogOther
	_ = common.UnmarshalJsonStr(log.Other, &other)
	cacheWrite := other.CacheWriteTokens
	if cacheWrite <= 0 {
		if other.CacheCreationTokens5m > 0 || other.CacheCreationTokens1h > 0 {
			cacheWrite = other.CacheCreationTokens5m + other.CacheCreationTokens1h
		} else {
			cacheWrite = other.CacheCreationTokens
		}
	}
	rate := other.BillingUSDToCNYRate
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		rate = 1
	}
	userCharge := 0.0
	if common.QuotaPerUnit > 0 {
		userCharge = float64(log.Quota) / common.QuotaPerUnit / rate
	}
	groupRatio := other.UserGroupRatio
	if groupRatio <= 0 || groupRatio == -1 || math.IsNaN(groupRatio) || math.IsInf(groupRatio, 0) {
		groupRatio = other.GroupRatio
	}
	if groupRatio <= 0 || math.IsNaN(groupRatio) || math.IsInf(groupRatio, 0) {
		groupRatio = 1
	}
	estimatedCost := userCharge / groupRatio
	latencyMS := float64(log.UseTime) * 1000

	summary.Requests++
	summary.PromptTokens += int64(log.PromptTokens)
	summary.CompletionTokens += int64(log.CompletionTokens)
	summary.CacheReadTokens += int64(other.CacheTokens)
	summary.CacheWriteTokens += int64(cacheWrite)
	summary.UserChargeUSD += userCharge
	summary.EstimatedCostUSD += estimatedCost
	summary.AverageLatencyMS += latencyMS

	day := time.Unix(log.CreatedAt, 0).UTC().Format("2006-01-02")
	modelName := strings.TrimSpace(log.ModelName)
	if modelName == "" {
		modelName = "Unknown"
	}
	inboundPath := strings.TrimSpace(other.RequestPath)
	if inboundPath == "" {
		inboundPath = "Unknown"
	}
	upstreamPath := strings.TrimSpace(other.UpstreamRequestPath)
	if upstreamPath == "" && len(other.RequestConversion) > 0 {
		upstreamPath = other.RequestConversion[len(other.RequestConversion)-1]
	}
	if upstreamPath == "" {
		upstreamPath = inboundPath
	}
	keys := []string{day, modelName, inboundPath, upstreamPath}
	for i, buckets := range bucketMaps {
		bucket := buckets[keys[i]]
		if bucket == nil {
			bucket = &ChannelReconciliationBucket{Name: keys[i]}
			buckets[keys[i]] = bucket
		}
		bucket.Requests++
		bucket.PromptTokens += int64(log.PromptTokens)
		bucket.CompletionTokens += int64(log.CompletionTokens)
		bucket.CacheReadTokens += int64(other.CacheTokens)
		bucket.CacheWriteTokens += int64(cacheWrite)
		bucket.UserChargeUSD += userCharge
		bucket.EstimatedCostUSD += estimatedCost
		bucket.LatencyTotalMS += latencyMS
	}
}

func proratedCost(entry ChannelCostEntry, startAt, endAt int64) float64 {
	overlapStart := maxInt64(entry.StartAt, startAt)
	overlapEnd := minInt64(entry.EndAt, endAt)
	if overlapEnd <= overlapStart || entry.EndAt <= entry.StartAt {
		return 0
	}
	return entry.AmountUSD * float64(overlapEnd-overlapStart) / float64(entry.EndAt-entry.StartAt)
}

func allocateCostToDaily(amount float64, startAt, endAt int64, daily map[string]*ChannelReconciliationBucket) {
	if amount <= 0 || endAt <= startAt {
		return
	}
	type dailyCandidate struct {
		bucket         *ChannelReconciliationBucket
		overlapSeconds int64
	}
	candidates := make([]dailyCandidate, 0)
	totalEstimate := 0.0
	for dayStart := time.Unix(startAt, 0).UTC().Truncate(24 * time.Hour); dayStart.Unix() < endAt; dayStart = dayStart.Add(24 * time.Hour) {
		dayEnd := dayStart.Add(24 * time.Hour)
		overlapStart := maxInt64(startAt, dayStart.Unix())
		overlapEnd := minInt64(endAt, dayEnd.Unix())
		if overlapEnd <= overlapStart {
			continue
		}
		name := dayStart.Format("2006-01-02")
		bucket := daily[name]
		if bucket == nil {
			bucket = &ChannelReconciliationBucket{Name: name}
			daily[name] = bucket
		}
		candidates = append(candidates, dailyCandidate{bucket: bucket, overlapSeconds: overlapEnd - overlapStart})
		totalEstimate += bucket.EstimatedCostUSD
	}
	if len(candidates) == 0 {
		return
	}
	if totalEstimate > 0 {
		for _, candidate := range candidates {
			candidate.bucket.ActualCostUSD += amount * candidate.bucket.EstimatedCostUSD / totalEstimate
		}
		return
	}
	totalSeconds := int64(0)
	for _, candidate := range candidates {
		totalSeconds += candidate.overlapSeconds
	}
	if totalSeconds > 0 {
		for _, candidate := range candidates {
			candidate.bucket.ActualCostUSD += amount * float64(candidate.overlapSeconds) / float64(totalSeconds)
		}
	}
}

func allocateActualCost(amount float64, buckets map[string]*ChannelReconciliationBucket) {
	if amount <= 0 || len(buckets) == 0 {
		return
	}
	totalEstimate := 0.0
	for _, bucket := range buckets {
		totalEstimate += bucket.EstimatedCostUSD
	}
	if totalEstimate > 0 {
		for _, bucket := range buckets {
			bucket.ActualCostUSD = amount * bucket.EstimatedCostUSD / totalEstimate
		}
		return
	}
	totalRequests := int64(0)
	for _, bucket := range buckets {
		totalRequests += bucket.Requests
	}
	if totalRequests == 0 {
		return
	}
	for _, bucket := range buckets {
		bucket.ActualCostUSD = amount * float64(bucket.Requests) / float64(totalRequests)
	}
}

func sortedBuckets(buckets map[string]*ChannelReconciliationBucket, byUsage bool) []ChannelReconciliationBucket {
	result := make([]ChannelReconciliationBucket, 0, len(buckets))
	for _, bucket := range buckets {
		result = append(result, *bucket)
	}
	sort.Slice(result, func(i, j int) bool {
		if byUsage && result[i].Requests != result[j].Requests {
			return result[i].Requests > result[j].Requests
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
