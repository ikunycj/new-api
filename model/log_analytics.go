package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const maxLogAnalyticsRange = 90 * 24 * time.Hour

type LogAnalyticsFilters struct {
	LogType        int
	StartTimestamp int64
	EndTimestamp   int64
	ModelName      string
	Keyword        string
	TokenName      string
	Channel        int
	Group          string
	Granularity    string
	TimezoneOffset int
	UserKeyword    string
	UserLimit      int
}

type LogAnalyticsSummary struct {
	TotalRequests    int64   `json:"total_requests"`
	SuccessRequests  int64   `json:"success_requests"`
	FailedRequests   int64   `json:"failed_requests"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	CacheReadTokens  int64   `json:"cache_read_tokens"`
	CacheWriteTokens int64   `json:"cache_write_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalQuota       int64   `json:"total_quota"`
	AverageUseTime   float64 `json:"average_use_time"`
	P90UseTime       float64 `json:"p90_use_time"`
	P99UseTime       float64 `json:"p99_use_time"`
}

type LogTokenTrendPoint struct {
	Timestamp        int64   `json:"timestamp"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	CacheReadTokens  int64   `json:"cache_read_tokens"`
	CacheWriteTokens int64   `json:"cache_write_tokens"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
}

type LogUserTrendPoint struct {
	Timestamp int64  `json:"timestamp"`
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Remark    string `json:"remark"`
	Tokens    int64  `json:"tokens"`
}

type LogDistributionItem struct {
	Name     string `json:"name"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
	Quota    int64  `json:"quota"`
}

type LogAnalytics struct {
	Summary           LogAnalyticsSummary   `json:"summary"`
	TokenTrend        []LogTokenTrendPoint  `json:"token_trend"`
	UserTrend         []LogUserTrendPoint   `json:"user_trend"`
	GroupDistribution []LogDistributionItem `json:"group_distribution"`
	ModelDistribution []LogDistributionItem `json:"model_distribution"`
}

type LogFilterChannel struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type LogFilterOptions struct {
	Groups   []string           `json:"groups"`
	Channels []LogFilterChannel `json:"channels"`
}

func validateLogAnalyticsFilters(filters *LogAnalyticsFilters) error {
	if filters.StartTimestamp <= 0 || filters.EndTimestamp <= 0 {
		return errors.New("start_timestamp and end_timestamp are required")
	}
	if filters.EndTimestamp <= filters.StartTimestamp {
		return errors.New("end_timestamp must be later than start_timestamp")
	}
	if filters.EndTimestamp-filters.StartTimestamp > int64(maxLogAnalyticsRange/time.Second) {
		return errors.New("log analytics range cannot exceed 90 days")
	}
	if filters.Granularity != "day" {
		filters.Granularity = "hour"
	}
	if filters.TimezoneOffset < -14*60 || filters.TimezoneOffset > 14*60 {
		return errors.New("timezone_offset must be between -840 and 840 minutes")
	}
	if filters.UserLimit != 20 {
		filters.UserLimit = 10
	}
	return nil
}

func applyLogAnalyticsFilters(tx *gorm.DB, filters LogAnalyticsFilters) (*gorm.DB, error) {
	var err error
	if filters.LogType != LogTypeUnknown {
		tx = tx.Where("logs.type = ?", filters.LogType)
	}
	if tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", filters.ModelName); err != nil {
		return nil, err
	}
	if tx, err = applyLogKeywordFilter(tx, filters.Keyword); err != nil {
		return nil, err
	}
	if filters.TokenName != "" {
		tx = tx.Where("logs.token_name = ?", filters.TokenName)
	}
	tx = tx.Where("logs.created_at >= ? AND logs.created_at <= ?", filters.StartTimestamp, filters.EndTimestamp)
	if filters.Channel != 0 {
		tx = tx.Where("logs.channel_id = ?", filters.Channel)
	}
	if filters.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", filters.Group)
	}
	return tx, nil
}

func logAnalyticsBucketExpression(granularity string, timezoneOffsetMinutes int) string {
	interval := int64(time.Hour / time.Second)
	if granularity == "day" {
		interval = int64(24 * time.Hour / time.Second)
	}
	offset := int64(timezoneOffsetMinutes) * 60
	return fmt.Sprintf("((logs.created_at - (%d)) / %d) * %d + (%d)", offset, interval, interval, offset)
}

func GetLogAnalytics(filters LogAnalyticsFilters) (LogAnalytics, error) {
	analytics := LogAnalytics{
		TokenTrend:        []LogTokenTrendPoint{},
		UserTrend:         []LogUserTrendPoint{},
		GroupDistribution: []LogDistributionItem{},
		ModelDistribution: []LogDistributionItem{},
	}
	if err := validateLogAnalyticsFilters(&filters); err != nil {
		return analytics, err
	}

	base, err := applyLogAnalyticsFilters(LOG_DB.Table("logs"), filters)
	if err != nil {
		return analytics, err
	}
	summarySelect := `
		COUNT(*) FILTER (WHERE logs.type IN (?, ?)) AS total_requests,
		COUNT(*) FILTER (WHERE logs.type = ?) AS success_requests,
		COUNT(*) FILTER (WHERE logs.type = ?) AS failed_requests,
		COALESCE(SUM(logs.prompt_tokens) FILTER (WHERE logs.type = ?), 0) AS input_tokens,
		COALESCE(SUM(logs.completion_tokens) FILTER (WHERE logs.type = ?), 0) AS output_tokens,
		COALESCE(SUM(logs.cache_read_tokens) FILTER (WHERE logs.type = ?), 0) AS cache_read_tokens,
		COALESCE(SUM(logs.cache_write_tokens) FILTER (WHERE logs.type = ?), 0) AS cache_write_tokens,
		COALESCE(SUM(logs.prompt_tokens + logs.completion_tokens) FILTER (WHERE logs.type = ?), 0) AS total_tokens,
		COALESCE(SUM(logs.quota) FILTER (WHERE logs.type = ?), 0) AS total_quota,
		COALESCE(AVG(logs.use_time) FILTER (WHERE logs.type IN (?, ?) AND logs.use_time > 0), 0) AS average_use_time,
		COALESCE(PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY logs.use_time) FILTER (WHERE logs.type IN (?, ?) AND logs.use_time > 0), 0) AS p90_use_time,
		COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY logs.use_time) FILTER (WHERE logs.type IN (?, ?) AND logs.use_time > 0), 0) AS p99_use_time`
	if err := base.Select(
		summarySelect,
		LogTypeConsume, LogTypeError,
		LogTypeConsume, LogTypeError,
		LogTypeConsume, LogTypeConsume, LogTypeConsume, LogTypeConsume,
		LogTypeConsume, LogTypeConsume,
		LogTypeConsume, LogTypeError,
		LogTypeConsume, LogTypeError,
		LogTypeConsume, LogTypeError,
	).Scan(&analytics.Summary).Error; err != nil {
		return analytics, err
	}

	bucketExpression := logAnalyticsBucketExpression(filters.Granularity, filters.TimezoneOffset)
	var tokenRows []struct {
		Timestamp        int64
		InputTokens      int64
		OutputTokens     int64
		CacheReadTokens  int64
		CacheWriteTokens int64
		CacheInputTokens int64
	}
	tokenQuery, err := applyLogAnalyticsFilters(LOG_DB.Table("logs"), filters)
	if err != nil {
		return analytics, err
	}
	if err := tokenQuery.
		Where("logs.type = ?", LogTypeConsume).
		Select(bucketExpression + " AS timestamp, COALESCE(SUM(logs.prompt_tokens), 0) AS input_tokens, COALESCE(SUM(logs.completion_tokens), 0) AS output_tokens, COALESCE(SUM(logs.cache_read_tokens), 0) AS cache_read_tokens, COALESCE(SUM(logs.cache_write_tokens), 0) AS cache_write_tokens, COALESCE(SUM(logs.input_tokens_total), 0) AS cache_input_tokens").
		Group(bucketExpression).
		Order("timestamp ASC").
		Scan(&tokenRows).Error; err != nil {
		return analytics, err
	}
	analytics.TokenTrend = make([]LogTokenTrendPoint, 0, len(tokenRows))
	for _, row := range tokenRows {
		point := LogTokenTrendPoint{
			Timestamp: row.Timestamp, InputTokens: row.InputTokens, OutputTokens: row.OutputTokens,
			CacheReadTokens: row.CacheReadTokens, CacheWriteTokens: row.CacheWriteTokens,
		}
		if row.CacheInputTokens > 0 {
			point.CacheHitRate = float64(row.CacheReadTokens) / float64(row.CacheInputTokens) * 100
		}
		analytics.TokenTrend = append(analytics.TokenTrend, point)
	}

	if err := populateLogUserTrend(&analytics, filters, bucketExpression); err != nil {
		return analytics, err
	}
	if analytics.GroupDistribution, err = getLogDistribution(filters, "logs."+logGroupCol); err != nil {
		return analytics, err
	}
	if analytics.ModelDistribution, err = getLogDistribution(filters, "logs.model_name"); err != nil {
		return analytics, err
	}
	return analytics, nil
}

func populateLogUserTrend(analytics *LogAnalytics, filters LogAnalyticsFilters, bucketExpression string) error {
	query, err := applyLogAnalyticsFilters(LOG_DB.Table("logs"), filters)
	if err != nil {
		return err
	}
	query = query.Where("logs.type = ?", LogTypeConsume)
	if strings.TrimSpace(filters.UserKeyword) != "" {
		userIDs, searchErr := getLogSearchUserIDs(filters.UserKeyword)
		if searchErr != nil {
			return searchErr
		}
		if len(userIDs) == 0 {
			return nil
		}
		query = query.Where("logs.user_id IN ?", userIDs)
	}

	var topUsers []struct {
		UserID int
		Tokens int64
	}
	if err := query.
		Select("logs.user_id, COALESCE(SUM(logs.prompt_tokens + logs.completion_tokens), 0) AS tokens").
		Where("logs.user_id <> 0").
		Group("logs.user_id").
		Order("tokens DESC").
		Limit(filters.UserLimit).
		Scan(&topUsers).Error; err != nil {
		return err
	}
	if len(topUsers) == 0 {
		return nil
	}

	userIDs := make([]int, 0, len(topUsers))
	for _, user := range topUsers {
		userIDs = append(userIDs, user.UserID)
	}
	trendQuery, err := applyLogAnalyticsFilters(LOG_DB.Table("logs"), filters)
	if err != nil {
		return err
	}
	if err := trendQuery.
		Where("logs.type = ? AND logs.user_id IN ?", LogTypeConsume, userIDs).
		Select(bucketExpression + " AS timestamp, logs.user_id, MAX(logs.username) AS username, COALESCE(SUM(logs.prompt_tokens + logs.completion_tokens), 0) AS tokens").
		Group(bucketExpression + ", logs.user_id").
		Order("timestamp ASC").
		Scan(&analytics.UserTrend).Error; err != nil {
		return err
	}

	var users []struct {
		ID     int
		Email  string
		Remark string
	}
	if err := DB.Unscoped().Model(&User{}).Select("id", "email", "remark").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return err
	}
	userDetails := make(map[int]struct{ Email, Remark string }, len(users))
	for _, user := range users {
		userDetails[user.ID] = struct{ Email, Remark string }{Email: user.Email, Remark: user.Remark}
	}
	for index := range analytics.UserTrend {
		details := userDetails[analytics.UserTrend[index].UserID]
		analytics.UserTrend[index].Email = details.Email
		analytics.UserTrend[index].Remark = details.Remark
	}
	return nil
}

func getLogDistribution(filters LogAnalyticsFilters, column string) ([]LogDistributionItem, error) {
	query, err := applyLogAnalyticsFilters(LOG_DB.Table("logs"), filters)
	if err != nil {
		return nil, err
	}
	rows := make([]LogDistributionItem, 0)
	if err := query.
		Where("logs.type = ?", LogTypeConsume).
		Select("COALESCE(" + column + ", '') AS name, COUNT(*) AS requests, COALESCE(SUM(logs.prompt_tokens + logs.completion_tokens), 0) AS tokens, COALESCE(SUM(logs.quota), 0) AS quota").
		Group(column).
		Order("tokens DESC").
		Limit(20).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for index := range rows {
		if strings.TrimSpace(rows[index].Name) == "" {
			rows[index].Name = "Unknown"
		}
	}
	return rows, nil
}

func populateLogUserDetails(logs []*Log) error {
	userIDs := make([]int, 0, len(logs))
	seen := make(map[int]struct{}, len(logs))
	for _, log := range logs {
		if log == nil || log.UserId == 0 {
			continue
		}
		if _, exists := seen[log.UserId]; exists {
			continue
		}
		seen[log.UserId] = struct{}{}
		userIDs = append(userIDs, log.UserId)
	}
	if len(userIDs) == 0 {
		return nil
	}

	var users []struct {
		ID     int
		Email  string
		Remark string
	}
	if err := DB.Unscoped().Model(&User{}).Select("id", "email", "remark").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return err
	}
	userDetails := make(map[int]struct{ Email, Remark string }, len(users))
	for _, user := range users {
		userDetails[user.ID] = struct{ Email, Remark string }{Email: user.Email, Remark: user.Remark}
	}
	for _, log := range logs {
		if log == nil {
			continue
		}
		details := userDetails[log.UserId]
		log.Email = details.Email
		log.Remark = details.Remark
	}
	return nil
}

func GetLogFilterOptions() (LogFilterOptions, error) {
	options := LogFilterOptions{Groups: []string{}, Channels: []LogFilterChannel{}}
	var err error
	options.Groups, err = GetPricingGroupNames()
	if err != nil {
		return options, err
	}

	var channelIDs []int
	if err := LOG_DB.Table("logs").Distinct("channel_id").Where("channel_id <> 0").Pluck("channel_id", &channelIDs).Error; err != nil {
		return options, err
	}
	if len(channelIDs) == 0 {
		return options, nil
	}
	if err := DB.Table("channels").Select("id", "name").Where("id IN ?", channelIDs).Scan(&options.Channels).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return options, nil
		}
		return options, err
	}
	sort.Slice(options.Channels, func(left, right int) bool {
		return options.Channels[left].ID < options.Channels[right].ID
	})
	return options, nil
}
