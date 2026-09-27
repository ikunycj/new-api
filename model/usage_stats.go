package model

import (
	"errors"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// Historical per-key / per-model usage.
//
// This is the long-range counterpart to the realtime rings. The rings answer
// "right now" at ten-second resolution but only retain six hours and die with
// the process; this reads quota_data, which is durable and unbounded in range
// but rolled up hourly.
//
// The split is deliberate rather than incidental: an hourly rollup cannot draw
// a one-hour chart (it would be a single point), and a six-hour in-memory ring
// cannot answer "last month". Each source is used where it is actually able to
// answer.
//
// Cost is included here and nowhere else. It is a billing figure, so it is
// wanted to the cent over a period rather than as an instantaneous rate, and
// putting it in the rings would have widened every slot on the relay hot path
// to serve a question no one asks per-second.

const (
	// usageStatsHourSeconds is the granularity of quota_data rows.
	usageStatsHourSeconds = 3600

	// usageStatsMaxRange caps how much history one request may scan. A year is
	// far beyond any dashboard use and exists to stop a single query walking the
	// whole table.
	usageStatsMaxRange = 366 * 24 * usageStatsHourSeconds

	// usageStatsMaxBuckets bounds the returned series. Combined with the range
	// cap it keeps the response a predictable size regardless of granularity.
	usageStatsMaxBuckets = 2000
)

// UsageStatsQuery selects the rows to aggregate.
//
// TokenID 0 and an empty Model mean "all", so the zero value is an unfiltered
// account-wide query.
type UsageStatsQuery struct {
	UserID    int
	TokenID   int
	Model     string
	StartTime int64
	EndTime   int64
	// BucketSeconds is the series resolution. It is rounded up to a whole
	// number of hours because quota_data cannot resolve finer.
	BucketSeconds int64
}

// UsageStatsBucket is one point of the historical series.
type UsageStatsBucket struct {
	Timestamp int64 `json:"timestamp"`
	Requests  int   `json:"requests"`
	Tokens    int   `json:"tokens"`
	// Quota is spend in the system's internal quota unit.
	Quota int `json:"quota"`
	// Raw cache terms rather than a ratio, so buckets can be re-aggregated by
	// the client without needing to weight per-bucket rates.
	CacheReadTokens  int `json:"cache_read_tokens"`
	InputTokensTotal int `json:"input_tokens_total"`
	// RPM and TPM are averages across the bucket, not peaks. An hourly bucket
	// smooths away bursts entirely, which is why the realtime panel and not
	// this series is what a user should read for instantaneous load.
	RPM float64 `json:"rpm"`
	TPM float64 `json:"tpm"`
}

// UsageStatsTotals is the aggregate across the whole requested range.
type UsageStatsTotals struct {
	Requests         int     `json:"requests"`
	Tokens           int     `json:"tokens"`
	Quota            int     `json:"quota"`
	CacheReadTokens  int     `json:"cache_read_tokens"`
	InputTokensTotal int     `json:"input_tokens_total"`
	RPM              float64 `json:"rpm"`
	TPM              float64 `json:"tpm"`
	// CacheHitRate is nil when no request in the range reported cache metadata,
	// which is not the same as a 0% rate. The realtime path makes the same
	// distinction, and for the same reason: rendering "unmeasured" as a
	// confident 0% would tell a user their cache is broken when nothing was
	// ever measured.
	CacheHitRate *float64 `json:"cache_hit_rate"`
}

// UsageStatsOption is one selectable key or model, with its usage so the client
// can order or label the choice.
type UsageStatsOption struct {
	TokenID   int    `json:"token_id,omitempty"`
	TokenName string `json:"token_name,omitempty"`
	Model     string `json:"model,omitempty"`
	Requests  int    `json:"requests"`
	Quota     int    `json:"quota"`
}

// UsageStatsResult is the full response.
type UsageStatsResult struct {
	StartTime     int64              `json:"start_time"`
	EndTime       int64              `json:"end_time"`
	BucketSeconds int64              `json:"bucket_seconds"`
	TokenID       int                `json:"token_id,omitempty"`
	Model         string             `json:"model,omitempty"`
	Totals        UsageStatsTotals   `json:"totals"`
	Series        []UsageStatsBucket `json:"series"`
	Tokens        []UsageStatsOption `json:"tokens"`
	Models        []UsageStatsOption `json:"models"`
}

// normalize validates the query and fills defaults, returning the effective
// values rather than mutating in place so the caller keeps what it asked for.
func (q UsageStatsQuery) normalize() (UsageStatsQuery, error) {
	if q.UserID <= 0 {
		return q, errors.New("invalid user id")
	}
	if q.EndTime <= 0 {
		q.EndTime = time.Now().Unix()
	}
	if q.StartTime <= 0 {
		q.StartTime = q.EndTime - 24*usageStatsHourSeconds
	}
	if q.StartTime >= q.EndTime {
		return q, errors.New("start_time must be before end_time")
	}
	if q.EndTime-q.StartTime > usageStatsMaxRange {
		return q, errors.New("time range too large")
	}

	if q.BucketSeconds <= 0 {
		q.BucketSeconds = usageStatsHourSeconds
	}
	// quota_data is an hourly rollup, so a finer bucket would produce empty
	// points between hours rather than more detail.
	if q.BucketSeconds < usageStatsHourSeconds {
		q.BucketSeconds = usageStatsHourSeconds
	}
	q.BucketSeconds -= q.BucketSeconds % usageStatsHourSeconds

	// Widen the bucket until the series fits rather than truncating it, so a
	// long range returns a coarser but complete picture instead of silently
	// stopping partway through.
	for (q.EndTime-q.StartTime)/q.BucketSeconds > usageStatsMaxBuckets {
		q.BucketSeconds += usageStatsHourSeconds
	}
	return q, nil
}

// usageStatsRow is one aggregated quota_data row.
type usageStatsRow struct {
	CreatedAt        int64
	TokenID          int
	ModelName        string
	Count            int
	Quota            int
	TokenUsed        int
	CacheReadTokens  int
	InputTokensTotal int
}

// GetUsageStats aggregates a user's historical usage, optionally narrowed to
// one key and/or model.
//
// The user id is always applied as a filter and is never taken from caller
// input at the HTTP layer, which is what keeps one account's history out of
// another's response.
func GetUsageStats(query UsageStatsQuery) (UsageStatsResult, error) {
	query, err := query.normalize()
	if err != nil {
		return UsageStatsResult{}, err
	}

	db := DB.Table("quota_data").
		Where("user_id = ? and created_at >= ? and created_at <= ?", query.UserID, query.StartTime, query.EndTime)
	if query.TokenID > 0 {
		db = db.Where("token_id = ?", query.TokenID)
	}
	if query.Model != "" {
		db = db.Where("model_name = ?", query.Model)
	}

	var rows []usageStatsRow
	// Group by the dimensions rather than reading raw rows: the option lists
	// need per-key and per-model totals, and doing the collapse in the database
	// keeps the transferred set proportional to the distinct combinations
	// instead of to the number of hours.
	err = db.Select(`created_at, token_id, model_name,
		sum(count) as count,
		sum(quota) as quota,
		sum(token_used) as token_used,
		sum(cache_read_tokens) as cache_read_tokens,
		sum(input_tokens_total) as input_tokens_total`).
		Group("created_at, token_id, model_name").
		Find(&rows).Error
	if err != nil {
		return UsageStatsResult{}, err
	}

	result := UsageStatsResult{
		StartTime:     query.StartTime,
		EndTime:       query.EndTime,
		BucketSeconds: query.BucketSeconds,
		TokenID:       query.TokenID,
		Model:         query.Model,
	}

	bucketCount := int((query.EndTime-query.StartTime)/query.BucketSeconds) + 1
	if bucketCount < 1 {
		bucketCount = 1
	}
	series := make([]UsageStatsBucket, bucketCount)
	for i := range series {
		series[i].Timestamp = query.StartTime + int64(i)*query.BucketSeconds
	}

	byToken := make(map[int]*UsageStatsOption)
	byModel := make(map[string]*UsageStatsOption)

	for _, row := range rows {
		index := int((row.CreatedAt - query.StartTime) / query.BucketSeconds)
		if index >= 0 && index < bucketCount {
			bucket := &series[index]
			bucket.Requests += row.Count
			bucket.Tokens += row.TokenUsed
			bucket.Quota += row.Quota
			bucket.CacheReadTokens += row.CacheReadTokens
			bucket.InputTokensTotal += row.InputTokensTotal
		}

		result.Totals.Requests += row.Count
		result.Totals.Tokens += row.TokenUsed
		result.Totals.Quota += row.Quota
		result.Totals.CacheReadTokens += row.CacheReadTokens
		result.Totals.InputTokensTotal += row.InputTokensTotal

		// The option lists describe what the user could switch to, so they are
		// built from the unfiltered dimensions of the rows that matched. When a
		// filter is active the list narrows with it, which is why the client
		// should load options unfiltered to populate its pickers.
		if row.TokenID > 0 {
			option, ok := byToken[row.TokenID]
			if !ok {
				option = &UsageStatsOption{TokenID: row.TokenID}
				byToken[row.TokenID] = option
			}
			option.Requests += row.Count
			option.Quota += row.Quota
		}
		if row.ModelName != "" {
			option, ok := byModel[row.ModelName]
			if !ok {
				option = &UsageStatsOption{Model: row.ModelName}
				byModel[row.ModelName] = option
			}
			option.Requests += row.Count
			option.Quota += row.Quota
		}
	}

	bucketMinutes := float64(query.BucketSeconds) / 60
	for i := range series {
		series[i].RPM = float64(series[i].Requests) / bucketMinutes
		series[i].TPM = float64(series[i].Tokens) / bucketMinutes
	}
	result.Series = series

	rangeMinutes := float64(query.EndTime-query.StartTime) / 60
	if rangeMinutes > 0 {
		result.Totals.RPM = float64(result.Totals.Requests) / rangeMinutes
		result.Totals.TPM = float64(result.Totals.Tokens) / rangeMinutes
	}
	result.Totals.CacheHitRate = usageStatsCacheHitRate(result.Totals)

	result.Tokens = make([]UsageStatsOption, 0, len(byToken))
	for _, option := range byToken {
		result.Tokens = append(result.Tokens, *option)
	}
	result.Models = make([]UsageStatsOption, 0, len(byModel))
	for _, option := range byModel {
		result.Models = append(result.Models, *option)
	}
	sort.Slice(result.Tokens, func(i, j int) bool {
		if result.Tokens[i].Requests != result.Tokens[j].Requests {
			return result.Tokens[i].Requests > result.Tokens[j].Requests
		}
		return result.Tokens[i].TokenID < result.Tokens[j].TokenID
	})
	sort.Slice(result.Models, func(i, j int) bool {
		if result.Models[i].Requests != result.Models[j].Requests {
			return result.Models[i].Requests > result.Models[j].Requests
		}
		return result.Models[i].Model < result.Models[j].Model
	})
	fillUsageStatsTokenNames(result.Tokens)

	return result, nil
}

// usageStatsCacheHitRate mirrors the realtime rule: no cache-reporting sample
// means no rate at all, rather than a rate of zero.
func usageStatsCacheHitRate(totals UsageStatsTotals) *float64 {
	if totals.InputTokensTotal <= 0 {
		return nil
	}
	rate := float64(totals.CacheReadTokens) / float64(totals.InputTokensTotal)
	return &rate
}

// fillUsageStatsTokenNames resolves key names in one query. A key deleted since
// it served traffic keeps an empty name rather than losing its row, because the
// spend it accounts for is real and still belongs in the breakdown.
func fillUsageStatsTokenNames(options []UsageStatsOption) {
	if len(options) == 0 {
		return
	}
	ids := make([]int, 0, len(options))
	for _, option := range options {
		ids = append(ids, option.TokenID)
	}
	var rows []struct {
		Id   int
		Name string
	}
	if err := DB.Model(&Token{}).Select("id, name").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		common.SysLog("usage stats: failed to resolve token names: " + err.Error())
		return
	}
	names := make(map[int]string, len(rows))
	for _, row := range rows {
		names[row.Id] = row.Name
	}
	for i := range options {
		options[i].TokenName = names[options[i].TokenID]
	}
}
