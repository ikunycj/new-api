package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
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

type LogCacheTrendDimension string

const (
	LogCacheTrendDimensionGroup   LogCacheTrendDimension = "group"
	LogCacheTrendDimensionChannel LogCacheTrendDimension = "channel"
)

type LogCacheTrendFilters struct {
	Dimension      LogCacheTrendDimension
	StartTimestamp int64
	EndTimestamp   int64
	Granularity    string
	TimezoneOffset int
}

type LogCacheTrendPoint struct {
	Timestamp             int64   `json:"timestamp"`
	Name                  string  `json:"name"`
	ChannelID             int     `json:"channel_id,omitempty"`
	CacheInputTokens      int64   `json:"cache_input_tokens"`
	CacheReadTokens       int64   `json:"cache_read_tokens"`
	CacheWriteTokens      int64   `json:"cache_write_tokens"`
	CacheHitRequests      int64   `json:"cache_hit_requests"`
	CacheEligibleRequests int64   `json:"cache_eligible_requests"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
}

type logCacheTrendMetricRow struct {
	Timestamp             int64 `gorm:"column:timestamp"`
	CacheInputTokens      int64 `gorm:"column:cache_input_tokens"`
	CacheReadTokens       int64 `gorm:"column:cache_read_tokens"`
	CacheWriteTokens      int64 `gorm:"column:cache_write_tokens"`
	CacheHitRequests      int64 `gorm:"column:cache_hit_requests"`
	CacheEligibleRequests int64 `gorm:"column:cache_eligible_requests"`
}

type logCacheTrendGroupRow struct {
	logCacheTrendMetricRow
	Name string `gorm:"column:name"`
}

type logCacheTrendChannelRow struct {
	logCacheTrendMetricRow
	ChannelID int `gorm:"column:channel_id"`
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
	switch granularity {
	case "day":
		interval = int64(24 * time.Hour / time.Second)
	case "week":
		interval = int64(7 * 24 * time.Hour / time.Second)
	}
	offset := int64(timezoneOffsetMinutes) * 60
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return fmt.Sprintf("intDiv(logs.created_at - (%d), %d) * %d + (%d)", offset, interval, interval, offset)
	}
	if common.UsingLogDatabase(common.DatabaseTypeMySQL) {
		return fmt.Sprintf("FLOOR((logs.created_at - (%d)) / %d) * %d + (%d)", offset, interval, interval, offset)
	}
	return fmt.Sprintf("((logs.created_at - (%d)) / %d) * %d + (%d)", offset, interval, interval, offset)
}

func validateLogCacheTrendFilters(filters *LogCacheTrendFilters) error {
	if filters.Dimension != LogCacheTrendDimensionGroup && filters.Dimension != LogCacheTrendDimensionChannel {
		return errors.New("dimension must be group or channel")
	}
	analyticsFilters := LogAnalyticsFilters{
		StartTimestamp: filters.StartTimestamp,
		EndTimestamp:   filters.EndTimestamp,
		Granularity:    filters.Granularity,
		TimezoneOffset: filters.TimezoneOffset,
	}
	if filters.Granularity == "week" {
		// The existing analytics endpoint only supports hour/day; validate the
		// shared range and timezone fields without normalizing cache week data.
		analyticsFilters.Granularity = "hour"
	}
	if err := validateLogAnalyticsFilters(&analyticsFilters); err != nil {
		return err
	}
	if filters.Granularity != "day" && filters.Granularity != "week" {
		filters.Granularity = "hour"
	}
	return nil
}

func cacheTrendMetricSelect(bucketExpression string) string {
	return bucketExpression + ` AS timestamp,
		COALESCE(SUM(logs.input_tokens_total), 0) AS cache_input_tokens,
		COALESCE(SUM(logs.cache_read_tokens), 0) AS cache_read_tokens,
		COALESCE(SUM(logs.cache_write_tokens), 0) AS cache_write_tokens,
		COALESCE(SUM(CASE WHEN logs.cache_read_tokens > 0 THEN 1 ELSE 0 END), 0) AS cache_hit_requests,
		COUNT(*) AS cache_eligible_requests`
}

func cacheTrendPointFromMetrics(metrics logCacheTrendMetricRow) LogCacheTrendPoint {
	point := LogCacheTrendPoint{
		Timestamp:             metrics.Timestamp,
		CacheInputTokens:      metrics.CacheInputTokens,
		CacheReadTokens:       metrics.CacheReadTokens,
		CacheWriteTokens:      metrics.CacheWriteTokens,
		CacheHitRequests:      metrics.CacheHitRequests,
		CacheEligibleRequests: metrics.CacheEligibleRequests,
	}
	if point.CacheInputTokens > 0 {
		point.CacheHitRate = float64(point.CacheReadTokens) / float64(point.CacheInputTokens) * 100
	}
	return point
}

func getLogCacheTrendByGroup(filters LogCacheTrendFilters, bucketExpression string) ([]LogCacheTrendPoint, error) {
	groupColumn := "logs." + logGroupCol
	var rows []logCacheTrendGroupRow
	query := LOG_DB.Table("logs").
		Where("logs.type = ? AND logs.cache_stats_available = ?", LogTypeConsume, true).
		Where("logs.created_at >= ? AND logs.created_at <= ?", filters.StartTimestamp, filters.EndTimestamp).
		Select(groupColumn + " AS name, " + cacheTrendMetricSelect(bucketExpression)).
		Group(groupColumn + ", " + bucketExpression).
		Order("timestamp ASC, name ASC")
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	points := make([]LogCacheTrendPoint, 0, len(rows))
	for _, row := range rows {
		point := cacheTrendPointFromMetrics(row.logCacheTrendMetricRow)
		point.Name = row.Name
		if strings.TrimSpace(point.Name) == "" {
			point.Name = "未记录分组"
		}
		points = append(points, point)
	}
	return points, nil
}

func getLogCacheTrendChannelNames(channelIDs []int) map[int]string {
	names := make(map[int]string, len(channelIDs))
	if len(channelIDs) == 0 || DB == nil {
		return names
	}
	var channels []struct {
		ID   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := DB.Table("channels").Select("id", "name").Where("id IN ?", channelIDs).Scan(&channels).Error; err != nil {
		return names
	}
	for _, channel := range channels {
		names[channel.ID] = strings.TrimSpace(channel.Name)
	}
	return names
}

func getLogCacheTrendByChannel(filters LogCacheTrendFilters, bucketExpression string) ([]LogCacheTrendPoint, error) {
	var rows []logCacheTrendChannelRow
	channelColumn := "logs.channel_id"
	query := LOG_DB.Table("logs").
		Where("logs.type = ? AND logs.cache_stats_available = ?", LogTypeConsume, true).
		Where("logs.created_at >= ? AND logs.created_at <= ?", filters.StartTimestamp, filters.EndTimestamp).
		Select(channelColumn + " AS channel_id, " + cacheTrendMetricSelect(bucketExpression)).
		Group(channelColumn + ", " + bucketExpression).
		Order("timestamp ASC, channel_id ASC")
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	channelIDs := make([]int, 0, len(rows))
	seenChannelIDs := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seenChannelIDs[row.ChannelID]; ok {
			continue
		}
		seenChannelIDs[row.ChannelID] = struct{}{}
		channelIDs = append(channelIDs, row.ChannelID)
	}
	channelNames := getLogCacheTrendChannelNames(channelIDs)

	points := make([]LogCacheTrendPoint, 0, len(rows))
	for _, row := range rows {
		point := cacheTrendPointFromMetrics(row.logCacheTrendMetricRow)
		point.ChannelID = row.ChannelID
		point.Name = channelNames[row.ChannelID]
		if point.Name == "" {
			if row.ChannelID == 0 {
				point.Name = "未记录渠道"
			} else {
				point.Name = fmt.Sprintf("渠道 #%d", row.ChannelID)
			}
		}
		points = append(points, point)
	}
	return points, nil
}

func GetLogCacheTrend(filters LogCacheTrendFilters) ([]LogCacheTrendPoint, error) {
	points := []LogCacheTrendPoint{}
	if err := validateLogCacheTrendFilters(&filters); err != nil {
		return points, err
	}
	bucketExpression := logAnalyticsBucketExpression(filters.Granularity, filters.TimezoneOffset)
	if filters.Dimension == LogCacheTrendDimensionGroup {
		return getLogCacheTrendByGroup(filters, bucketExpression)
	}
	return getLogCacheTrendByChannel(filters, bucketExpression)
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
