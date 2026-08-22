package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

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

func populateLogUserDetails(logs []*Log) error {
	if len(logs) == 0 {
		return nil
	}

	ids := make([]int, 0, len(logs))
	seen := make(map[int]struct{}, len(logs))
	for _, log := range logs {
		if log.UserId == 0 {
			continue
		}
		if _, ok := seen[log.UserId]; ok {
			continue
		}
		seen[log.UserId] = struct{}{}
		ids = append(ids, log.UserId)
	}
	if len(ids) == 0 {
		return nil
	}

	var users []struct {
		ID     int
		Email  string
		Remark string
	}
	if err := DB.Unscoped().Model(&User{}).Select("id", "email", "remark").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return err
	}
	userMap := make(map[int]struct {
		Email  string
		Remark string
	}, len(users))
	for _, user := range users {
		userMap[user.ID] = struct {
			Email  string
			Remark string
		}{Email: user.Email, Remark: user.Remark}
	}
	for _, log := range logs {
		user, ok := userMap[log.UserId]
		if !ok {
			continue
		}
		log.Email = user.Email
		log.Remark = user.Remark
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
	if filters.StartTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", filters.StartTimestamp)
	}
	if filters.EndTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", filters.EndTimestamp)
	}
	if filters.Channel != 0 {
		tx = tx.Where("logs.channel_id = ?", filters.Channel)
	}
	if filters.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", filters.Group)
	}
	return tx, nil
}

func logAnalyticsBucketExpression(granularity string) string {
	interval := int64(3600)
	if granularity == "day" {
		interval = 86400
	}
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return fmt.Sprintf("intDiv(logs.created_at, %d) * %d", interval, interval)
	}
	if common.UsingLogDatabase(common.DatabaseTypeMySQL) {
		return fmt.Sprintf("FLOOR(logs.created_at / %d) * %d", interval, interval)
	}
	return fmt.Sprintf("(logs.created_at / %d) * %d", interval, interval)
}

func getLogUseTimePercentile(base *gorm.DB, count int64, percentile float64) (float64, error) {
	if count == 0 {
		return 0, nil
	}
	offset := int(math.Ceil(float64(count)*percentile)) - 1
	if offset < 0 {
		offset = 0
	}
	var value int
	if err := base.Select("logs.use_time").Order("logs.use_time ASC").Offset(offset).Limit(1).Scan(&value).Error; err != nil {
		return 0, err
	}
	return float64(value), nil
}

func GetLogAnalytics(filters LogAnalyticsFilters) (analytics LogAnalytics, err error) {
	if filters.Granularity != "day" {
		filters.Granularity = "hour"
	}
	if filters.UserLimit != 20 {
		filters.UserLimit = 10
	}

	base, err := applyLogAnalyticsFilters(LOG_DB.Table("logs"), filters)
	if err != nil {
		return analytics, err
	}

	summarySelect := `
		COALESCE(SUM(CASE WHEN logs.type IN (?, ?) THEN 1 ELSE 0 END), 0) AS total_requests,
		COALESCE(SUM(CASE WHEN logs.type = ? THEN 1 ELSE 0 END), 0) AS success_requests,
		COALESCE(SUM(CASE WHEN logs.type = ? THEN 1 ELSE 0 END), 0) AS failed_requests,
		COALESCE(SUM(CASE WHEN logs.type = ? THEN logs.prompt_tokens ELSE 0 END), 0) AS input_tokens,
		COALESCE(SUM(CASE WHEN logs.type = ? THEN logs.completion_tokens ELSE 0 END), 0) AS output_tokens,
		COALESCE(SUM(CASE WHEN logs.type = ? THEN logs.cache_read_tokens ELSE 0 END), 0) AS cache_read_tokens,
		COALESCE(SUM(CASE WHEN logs.type = ? THEN logs.cache_write_tokens ELSE 0 END), 0) AS cache_write_tokens,
		COALESCE(SUM(CASE WHEN logs.type = ? THEN logs.prompt_tokens + logs.completion_tokens ELSE 0 END), 0) AS total_tokens,
		COALESCE(SUM(CASE WHEN logs.type = ? THEN logs.quota ELSE 0 END), 0) AS total_quota,
		COALESCE(AVG(CASE WHEN logs.type IN (?, ?) AND logs.use_time > 0 THEN logs.use_time ELSE NULL END), 0) AS average_use_time`
	if err = base.Select(summarySelect,
		LogTypeConsume, LogTypeError,
		LogTypeConsume, LogTypeError,
		LogTypeConsume, LogTypeConsume, LogTypeConsume, LogTypeConsume,
		LogTypeConsume, LogTypeConsume,
		LogTypeConsume, LogTypeError,
	).Scan(&analytics.Summary).Error; err != nil {
		return analytics, err
	}

	durationQuery := base.Where("logs.type IN ? AND logs.use_time > 0", []int{LogTypeConsume, LogTypeError})
	var durationCount int64
	if err = durationQuery.Count(&durationCount).Error; err != nil {
		return analytics, err
	}
	if analytics.Summary.P90UseTime, err = getLogUseTimePercentile(durationQuery, durationCount, 0.90); err != nil {
		return analytics, err
	}
	if analytics.Summary.P99UseTime, err = getLogUseTimePercentile(durationQuery, durationCount, 0.99); err != nil {
		return analytics, err
	}

	bucketExpr := logAnalyticsBucketExpression(filters.Granularity)
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
	if err = tokenQuery.
		Where("logs.type = ?", LogTypeConsume).
		Select(bucketExpr + " AS timestamp, COALESCE(SUM(logs.prompt_tokens), 0) AS input_tokens, COALESCE(SUM(logs.completion_tokens), 0) AS output_tokens, COALESCE(SUM(logs.cache_read_tokens), 0) AS cache_read_tokens, COALESCE(SUM(logs.cache_write_tokens), 0) AS cache_write_tokens, COALESCE(SUM(logs.input_tokens_total), 0) AS cache_input_tokens").
		Group(bucketExpr).Order("timestamp ASC").Scan(&tokenRows).Error; err != nil {
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

	if err = populateLogUserTrend(&analytics, filters, bucketExpr); err != nil {
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

func populateLogUserTrend(analytics *LogAnalytics, filters LogAnalyticsFilters, bucketExpr string) error {
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
			analytics.UserTrend = []LogUserTrendPoint{}
			return nil
		}
		query = query.Where("logs.user_id IN ?", userIDs)
	}

	var topUsers []struct {
		UserID   int
		Username string
		Tokens   int64
	}
	if err = query.Select("logs.user_id, MAX(logs.username) AS username, COALESCE(SUM(logs.prompt_tokens + logs.completion_tokens), 0) AS tokens").
		Group("logs.user_id").Order("tokens DESC").Limit(filters.UserLimit).Scan(&topUsers).Error; err != nil {
		return err
	}
	if len(topUsers) == 0 {
		analytics.UserTrend = []LogUserTrendPoint{}
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
	if err = trendQuery.Where("logs.type = ? AND logs.user_id IN ?", LogTypeConsume, userIDs).
		Select(bucketExpr + " AS timestamp, logs.user_id, MAX(logs.username) AS username, COALESCE(SUM(logs.prompt_tokens + logs.completion_tokens), 0) AS tokens").
		Group(bucketExpr + ", logs.user_id").Order("timestamp ASC").Scan(&analytics.UserTrend).Error; err != nil {
		return err
	}

	var users []struct {
		ID     int
		Email  string
		Remark string
	}
	if err = DB.Unscoped().Model(&User{}).Select("id", "email", "remark").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return err
	}
	userMap := make(map[int]struct{ Email, Remark string }, len(users))
	for _, user := range users {
		userMap[user.ID] = struct{ Email, Remark string }{Email: user.Email, Remark: user.Remark}
	}
	for i := range analytics.UserTrend {
		user := userMap[analytics.UserTrend[i].UserID]
		analytics.UserTrend[i].Email = user.Email
		analytics.UserTrend[i].Remark = user.Remark
	}
	return nil
}

func getLogDistribution(filters LogAnalyticsFilters, column string) ([]LogDistributionItem, error) {
	query, err := applyLogAnalyticsFilters(LOG_DB.Table("logs"), filters)
	if err != nil {
		return nil, err
	}
	var rows []LogDistributionItem
	if err = query.Where("logs.type = ?", LogTypeConsume).
		Select("COALESCE(" + column + ", '') AS name, COUNT(*) AS requests, COALESCE(SUM(logs.prompt_tokens + logs.completion_tokens), 0) AS tokens, COALESCE(SUM(logs.quota), 0) AS quota").
		Group(column).Order("tokens DESC").Limit(20).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		if strings.TrimSpace(rows[i].Name) == "" {
			rows[i].Name = "Unknown"
		}
	}
	return rows, nil
}

func GetLogFilterOptions() (options LogFilterOptions, err error) {
	if err = LOG_DB.Table("logs").Distinct(logGroupCol).Where(logGroupCol+" <> ''").Pluck(logGroupCol, &options.Groups).Error; err != nil {
		return options, err
	}
	sort.Strings(options.Groups)

	var channelIDs []int
	if err = LOG_DB.Table("logs").Distinct("channel_id").Where("channel_id <> 0").Pluck("channel_id", &channelIDs).Error; err != nil {
		return options, err
	}
	if len(channelIDs) == 0 {
		options.Channels = []LogFilterChannel{}
		return options, nil
	}
	if err = DB.Table("channels").Select("id", "name").Where("id IN ?", channelIDs).Scan(&options.Channels).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return options, nil
		}
		return options, err
	}
	sort.Slice(options.Channels, func(i, j int) bool {
		return options.Channels[i].ID < options.Channels[j].ID
	})
	return options, nil
}
