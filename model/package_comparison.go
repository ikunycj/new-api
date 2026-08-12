package model

import (
	"errors"
	"time"
)

const maxPackageComparisonPlans = 10

type PackageComparisonStat struct {
	PlanId           int     `json:"plan_id"`
	PlanTitle        string  `json:"plan_title"`
	PlanPrice        float64 `json:"plan_price"`
	Currency         string  `json:"currency"`
	PlanQuota        int64   `json:"plan_quota"`
	Requests         int64   `json:"requests"`
	SuccessRequests  int64   `json:"success_requests"`
	ErrorRequests    int64   `json:"error_requests"`
	SuccessRate      float64 `json:"success_rate"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	Quota            int64   `json:"quota"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	ChannelHitRate   float64 `json:"channel_hit_rate"`
}

type packageComparisonRow struct {
	PlanId           int64   `gorm:"column:plan_id"`
	Requests         int64   `gorm:"column:requests"`
	SuccessRequests  int64   `gorm:"column:success_requests"`
	ErrorRequests    int64   `gorm:"column:error_requests"`
	PromptTokens     int64   `gorm:"column:prompt_tokens"`
	CompletionTokens int64   `gorm:"column:completion_tokens"`
	Quota            int64   `gorm:"column:quota"`
	AverageUseTime   float64 `gorm:"column:average_use_time"`
}

// GetPackageComparisonStats aggregates request logs by the subscription plan
// attached at request time. It intentionally uses portable SQL expressions so
// the endpoint works with SQLite, MySQL, PostgreSQL, and the SQL log backends.
func GetPackageComparisonStats(planIDs []int, startTimestamp, endTimestamp int64, modelName, group string) ([]PackageComparisonStat, error) {
	if len(planIDs) == 0 || len(planIDs) > maxPackageComparisonPlans {
		return nil, errors.New("plan_ids must contain between 1 and 10 plans")
	}
	if startTimestamp <= 0 {
		startTimestamp = time.Now().Add(-24 * time.Hour).Unix()
	}
	if endTimestamp <= 0 {
		endTimestamp = time.Now().Unix()
	}
	if endTimestamp <= startTimestamp || endTimestamp-startTimestamp > 90*24*60*60 {
		return nil, errors.New("time range must be positive and no longer than 90 days")
	}

	type planSummary struct {
		title    string
		price    float64
		currency string
		quota    int64
	}
	plans := make(map[int]planSummary, len(planIDs))
	var planRows []SubscriptionPlan
	if err := DB.Where("id IN ?", planIDs).Find(&planRows).Error; err != nil {
		return nil, err
	}
	for _, plan := range planRows {
		plans[plan.Id] = planSummary{title: plan.Title, price: plan.PriceAmount, currency: plan.Currency, quota: plan.TotalAmount}
	}
	if len(plans) != len(planIDs) {
		return nil, errors.New("one or more subscription plans were not found")
	}

	query := LOG_DB.Table("logs").Select(
		"subscription_plan_id AS plan_id, "+
			"COUNT(*) AS requests, "+
			"SUM(CASE WHEN type = ? THEN 1 ELSE 0 END) AS success_requests, "+
			"SUM(CASE WHEN type = ? THEN 1 ELSE 0 END) AS error_requests, "+
			"COALESCE(SUM(CASE WHEN type = ? THEN prompt_tokens ELSE 0 END), 0) AS prompt_tokens, "+
			"COALESCE(SUM(CASE WHEN type = ? THEN completion_tokens ELSE 0 END), 0) AS completion_tokens, "+
			"COALESCE(SUM(CASE WHEN type = ? THEN quota ELSE 0 END), 0) AS quota, "+
			"COALESCE(AVG(CASE WHEN type = ? THEN use_time * 1000.0 ELSE NULL END), 0) AS average_use_time",
		LogTypeConsume, LogTypeError, LogTypeConsume, LogTypeConsume, LogTypeConsume, LogTypeConsume,
	).Where("subscription_plan_id IN ? AND type IN ? AND created_at >= ? AND created_at <= ?", planIDs, []int{LogTypeConsume, LogTypeError}, startTimestamp, endTimestamp).
		Group("subscription_plan_id")
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}
	if group != "" {
		query = query.Where(""+logGroupCol+" = ?", group)
	}
	var rows []packageComparisonRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]PackageComparisonStat, 0, len(planIDs))
	for _, planID := range planIDs {
		plan := plans[planID]
		stat := PackageComparisonStat{PlanId: planID, PlanTitle: plan.title, PlanPrice: plan.price, Currency: plan.currency, PlanQuota: plan.quota}
		for _, row := range rows {
			if int(row.PlanId) != planID {
				continue
			}
			stat.Requests = row.Requests
			stat.SuccessRequests = row.SuccessRequests
			stat.ErrorRequests = row.ErrorRequests
			stat.PromptTokens = row.PromptTokens
			stat.CompletionTokens = row.CompletionTokens
			stat.TotalTokens = row.PromptTokens + row.CompletionTokens
			stat.Quota = row.Quota
			stat.AverageLatencyMs = row.AverageUseTime
			if stat.Requests > 0 {
				stat.SuccessRate = float64(stat.SuccessRequests) / float64(stat.Requests)
				stat.ChannelHitRate = stat.SuccessRate
			}
			break
		}
		result = append(result, stat)
	}
	return result, nil
}
