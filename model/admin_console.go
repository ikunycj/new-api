package model

import (
	"errors"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const adminConsoleStatsCacheTTL = time.Minute
const adminConsoleMaxRangeSeconds = int64(90 * 24 * time.Hour / time.Second)

type adminConsoleStatsCacheKey struct {
	startTimestamp int64
	endTimestamp   int64
	customRange    bool
}

type adminConsoleStatsCacheEntry struct {
	expiresAt time.Time
	stats     AdminConsoleStats
}

var adminConsoleStatsCache = struct {
	sync.Mutex
	entries map[adminConsoleStatsCacheKey]adminConsoleStatsCacheEntry
}{entries: make(map[adminConsoleStatsCacheKey]adminConsoleStatsCacheEntry)}

type AdminConsoleKeyStats struct {
	Total   int64 `json:"total"`
	Active  int64 `json:"active"`
	Enabled int64 `json:"enabled"`
}

type AdminConsoleChannelStats struct {
	Total        int64 `json:"total"`
	Enabled      int64 `json:"enabled"`
	AutoDisabled int64 `json:"auto_disabled"`
}

type AdminConsolePeriodStats struct {
	Today int64 `json:"today"`
	Month int64 `json:"month"`
	Total int64 `json:"total"`
}

type AdminConsoleUserStats struct {
	Today       int64 `json:"today"`
	Total       int64 `json:"total"`
	ActiveToday int64 `json:"active_today"`
	ActiveWeek  int64 `json:"active_week"`
	ActiveMonth int64 `json:"active_month"`
}

type AdminConsoleRevenueStats struct {
	Today float64 `json:"today"`
	Month float64 `json:"month"`
	Total float64 `json:"total"`
}

type AdminConsolePerformanceStats struct {
	AverageResponseSeconds  float64 `json:"average_response_seconds"`
	TodayResponseP50Seconds float64 `json:"today_response_p50_seconds"`
	TodayResponseP90Seconds float64 `json:"today_response_p90_seconds"`
	TodayResponseP99Seconds float64 `json:"today_response_p99_seconds"`
	TodayRPMP50             float64 `json:"today_rpm_p50"`
	TodayRPMP90             float64 `json:"today_rpm_p90"`
	TodayRPMP99             float64 `json:"today_rpm_p99"`
	TodayTPMP50             float64 `json:"today_tpm_p50"`
	TodayTPMP90             float64 `json:"today_tpm_p90"`
	TodayTPMP99             float64 `json:"today_tpm_p99"`
	ConcurrencyP50          int64   `json:"concurrency_p50"`
	ConcurrencyP90          int64   `json:"concurrency_p90"`
	ConcurrencyP99          int64   `json:"concurrency_p99"`
	MonthConcurrencyP50     int64   `json:"month_concurrency_p50"`
	MonthConcurrencyP90     int64   `json:"month_concurrency_p90"`
	MonthConcurrencyP95     int64   `json:"month_concurrency_p95"`
}

type AdminConsoleRealtimeStats struct {
	CurrentConcurrency int64   `json:"current_concurrency"`
	ResponseSeconds    float64 `json:"response_seconds"`
	RPM                int64   `json:"rpm"`
	TPM                int64   `json:"tpm"`
}

type AdminConsoleSystemLoad struct {
	CPUUsagePercent     float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent  float64 `json:"memory_usage_percent"`
	StorageUsagePercent float64 `json:"storage_usage_percent"`
}

type AdminConsoleStats struct {
	APIKeys     AdminConsoleKeyStats         `json:"api_keys"`
	Channels    AdminConsoleChannelStats     `json:"channels"`
	Requests    AdminConsolePeriodStats      `json:"requests"`
	Users       AdminConsoleUserStats        `json:"users"`
	Tokens      AdminConsolePeriodStats      `json:"tokens"`
	Quota       AdminConsolePeriodStats      `json:"quota"`
	Revenue     AdminConsoleRevenueStats     `json:"revenue"`
	Performance AdminConsolePerformanceStats `json:"performance"`
	SystemLoad  AdminConsoleSystemLoad       `json:"system_load"`
}

type adminConsoleMainRow struct {
	APIKeysTotal         int64   `gorm:"column:api_keys_total"`
	APIKeysEnabled       int64   `gorm:"column:api_keys_enabled"`
	ChannelsTotal        int64   `gorm:"column:channels_total"`
	ChannelsEnabled      int64   `gorm:"column:channels_enabled"`
	ChannelsAutoDisabled int64   `gorm:"column:channels_auto_disabled"`
	UsersToday           int64   `gorm:"column:users_today"`
	UsersTotal           int64   `gorm:"column:users_total"`
	RevenueToday         float64 `gorm:"column:revenue_today"`
	RevenueMonth         float64 `gorm:"column:revenue_month"`
	RevenueTotal         float64 `gorm:"column:revenue_total"`
}

type adminConsoleLogRow struct {
	RequestsToday           int64   `gorm:"column:requests_today"`
	RequestsMonth           int64   `gorm:"column:requests_month"`
	RequestsTotal           int64   `gorm:"column:requests_total"`
	TokensToday             int64   `gorm:"column:tokens_today"`
	TokensMonth             int64   `gorm:"column:tokens_month"`
	TokensTotal             int64   `gorm:"column:tokens_total"`
	QuotaToday              int64   `gorm:"column:quota_today"`
	QuotaMonth              int64   `gorm:"column:quota_month"`
	QuotaTotal              int64   `gorm:"column:quota_total"`
	ActiveKeysToday         int64   `gorm:"column:active_keys_today"`
	ActiveUsersToday        int64   `gorm:"column:active_users_today"`
	ActiveUsersWeek         int64   `gorm:"column:active_users_week"`
	ActiveUsersMonth        int64   `gorm:"column:active_users_month"`
	ResponseAverageSeconds  float64 `gorm:"column:response_average_seconds"`
	ResponseP50SecondsToday float64 `gorm:"column:response_p50_seconds_today"`
	ResponseP90SecondsToday float64 `gorm:"column:response_p90_seconds_today"`
	ResponseP99SecondsToday float64 `gorm:"column:response_p99_seconds_today"`
}

type adminConsoleThroughputRow struct {
	RPMP50 float64 `gorm:"column:rpm_p50"`
	RPMP90 float64 `gorm:"column:rpm_p90"`
	RPMP99 float64 `gorm:"column:rpm_p99"`
	TPMP50 float64 `gorm:"column:tpm_p50"`
	TPMP90 float64 `gorm:"column:tpm_p90"`
	TPMP99 float64 `gorm:"column:tpm_p99"`
}

type adminConsoleConcurrencyRow struct {
	P50 int64 `gorm:"column:p50"`
	P90 int64 `gorm:"column:p90"`
	P95 int64 `gorm:"column:p95"`
	P99 int64 `gorm:"column:p99"`
}

type adminConsoleRealtimeRow struct {
	ResponseSeconds float64 `gorm:"column:response_seconds"`
	RPM             int64   `gorm:"column:rpm"`
	TPM             int64   `gorm:"column:tpm"`
}

func adminConsoleTimeBounds(now time.Time) (todayStart, tomorrowStart, weekStart, rollingMonthStart, monthStart int64) {
	localNow := now.In(time.Local)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.Local)
	return today.Unix(),
		today.AddDate(0, 0, 1).Unix(),
		today.AddDate(0, 0, -6).Unix(),
		today.AddDate(0, 0, -29).Unix(),
		time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, time.Local).Unix()
}

const adminConsoleMaxInt64 = int64(^uint64(0) >> 1)

type adminConsoleTimeRange struct {
	startTimestamp       int64
	endTimestamp         int64
	analysisEndTimestamp int64
	custom               bool
}

func normalizeAdminConsoleTimeRange(now time.Time, startTimestamp, endTimestamp int64) (adminConsoleTimeRange, error) {
	if startTimestamp <= 0 || endTimestamp <= 0 || endTimestamp < startTimestamp || endTimestamp == adminConsoleMaxInt64 {
		return adminConsoleTimeRange{}, errors.New("invalid admin console time range")
	}
	if endTimestamp-startTimestamp > adminConsoleMaxRangeSeconds {
		return adminConsoleTimeRange{}, errors.New("admin console time range cannot exceed 90 days")
	}

	endExclusive := endTimestamp + 1
	analysisEnd := endExclusive
	nowExclusive := now.Unix() + 1
	if startTimestamp < nowExclusive && analysisEnd > nowExclusive {
		analysisEnd = nowExclusive
	}
	return adminConsoleTimeRange{
		startTimestamp:       startTimestamp,
		endTimestamp:         endExclusive,
		analysisEndTimestamp: analysisEnd,
		custom:               true,
	}, nil
}

func getAdminConsoleStatsAt(now time.Time) (AdminConsoleStats, error) {
	return getAdminConsoleStatsAtRange(now, adminConsoleTimeRange{})
}

func getAdminConsoleStatsAtRange(now time.Time, selectedRange adminConsoleTimeRange) (AdminConsoleStats, error) {
	var stats AdminConsoleStats
	todayStart, tomorrowStart, weekStart, rollingMonthStart, monthStart := adminConsoleTimeBounds(now)
	metricStart, metricEnd := todayStart, tomorrowStart
	monthMetricStart, monthMetricEnd := monthStart, tomorrowStart
	weekMetricStart, weekMetricEnd := weekStart, tomorrowStart
	rollingMonthMetricStart, rollingMonthMetricEnd := rollingMonthStart, tomorrowStart
	analysisStart, analysisEnd := todayStart, now.Unix()
	concurrencyStart, concurrencyEnd := monthStart, now.Unix()
	if selectedRange.custom {
		metricStart = selectedRange.startTimestamp
		metricEnd = selectedRange.endTimestamp
		monthMetricStart, monthMetricEnd = metricStart, metricEnd
		weekMetricStart, weekMetricEnd = metricStart, metricEnd
		rollingMonthMetricStart, rollingMonthMetricEnd = metricStart, metricEnd
		analysisStart = selectedRange.startTimestamp
		analysisEnd = selectedRange.analysisEndTimestamp
		concurrencyStart = selectedRange.startTimestamp
		concurrencyEnd = selectedRange.analysisEndTimestamp
	}

	var mainRow adminConsoleMainRow
	mainQuery := `
		SELECT
			(SELECT COUNT(*) FROM tokens WHERE deleted_at IS NULL) AS api_keys_total,
			(SELECT COUNT(*) FROM tokens WHERE deleted_at IS NULL AND status = ?) AS api_keys_enabled,
			(SELECT COUNT(*) FROM channels) AS channels_total,
			(SELECT COUNT(*) FROM channels WHERE status = ?) AS channels_enabled,
			(SELECT COUNT(*) FROM channels WHERE status = ?) AS channels_auto_disabled,
			(SELECT COUNT(*) FROM users WHERE created_at >= ? AND created_at < ?) AS users_today,
			(SELECT COUNT(*) FROM users) AS users_total,
			COALESCE(SUM(CASE WHEN completed_at >= ? AND completed_at < ? THEN paid_amount ELSE 0 END), 0) AS revenue_today,
			COALESCE(SUM(CASE WHEN completed_at >= ? AND completed_at < ? THEN paid_amount ELSE 0 END), 0) AS revenue_month,
			COALESCE(SUM(paid_amount), 0) AS revenue_total
		FROM (
			SELECT
				CASE WHEN complete_time > 0 THEN complete_time ELSE create_time END AS completed_at,
				CASE WHEN paid_cents > 0 THEN paid_cents / 100.0 ELSE money END AS paid_amount
			FROM top_ups
			WHERE status = ?
				AND (paid_cents > 0 OR money > 0)
				AND COALESCE(payment_method, '') <> ?
		) successful_topups`
	if err := DB.Raw(
		mainQuery,
		common.TokenStatusEnabled,
		common.ChannelStatusEnabled,
		common.ChannelStatusAutoDisabled,
		metricStart,
		metricEnd,
		metricStart,
		metricEnd,
		monthMetricStart,
		monthMetricEnd,
		common.TopUpStatusSuccess,
		PaymentMethodBalance,
	).Scan(&mainRow).Error; err != nil {
		return stats, err
	}

	var logRow adminConsoleLogRow
	logQuery := `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= ? AND created_at < ?) AS requests_today,
			COUNT(*) FILTER (WHERE created_at >= ? AND created_at < ?) AS requests_month,
			COUNT(*) AS requests_total,
			COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS tokens_today,
			COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS tokens_month,
			COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint), 0) AS tokens_total,
			COALESCE(SUM(quota) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS quota_today,
			COALESCE(SUM(quota) FILTER (WHERE created_at >= ? AND created_at < ?), 0) AS quota_month,
			COALESCE(SUM(quota), 0) AS quota_total,
			COUNT(DISTINCT token_id) FILTER (WHERE token_id <> 0 AND created_at >= ? AND created_at < ?) AS active_keys_today,
			COUNT(DISTINCT user_id) FILTER (WHERE user_id <> 0 AND created_at >= ? AND created_at < ?) AS active_users_today,
			COUNT(DISTINCT user_id) FILTER (WHERE user_id <> 0 AND created_at >= ? AND created_at < ?) AS active_users_week,
			COUNT(DISTINCT user_id) FILTER (WHERE user_id <> 0 AND created_at >= ? AND created_at < ?) AS active_users_month,
			COALESCE(AVG(use_time) FILTER (WHERE use_time > 0 AND created_at >= ? AND created_at < ?), 0) AS response_average_seconds,
			COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY use_time) FILTER (WHERE use_time > 0 AND created_at >= ? AND created_at < ?), 0) AS response_p50_seconds_today,
			COALESCE(PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY use_time) FILTER (WHERE use_time > 0 AND created_at >= ? AND created_at < ?), 0) AS response_p90_seconds_today,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY use_time) FILTER (WHERE use_time > 0 AND created_at >= ? AND created_at < ?), 0) AS response_p99_seconds_today
		FROM logs
		WHERE type = ?`
	if err := LOG_DB.Raw(
		logQuery,
		metricStart, metricEnd,
		monthMetricStart, monthMetricEnd,
		metricStart, metricEnd,
		monthMetricStart, monthMetricEnd,
		metricStart, metricEnd,
		monthMetricStart, monthMetricEnd,
		metricStart, metricEnd,
		metricStart, metricEnd,
		weekMetricStart, weekMetricEnd,
		rollingMonthMetricStart, rollingMonthMetricEnd,
		metricStart, metricEnd,
		metricStart, metricEnd,
		metricStart, metricEnd,
		metricStart, metricEnd,
		LogTypeConsume,
	).Scan(&logRow).Error; err != nil {
		return stats, err
	}

	var throughputRow adminConsoleThroughputRow
	// Include every elapsed minute in the selected analysis range, including
	// periods with no traffic.
	throughputQuery := `
		WITH minutes AS (
			SELECT generate_series(?::bigint / 60, (?::bigint - 1) / 60) AS minute
		),
		minute_usage AS (
			SELECT
				created_at / 60 AS minute,
				COUNT(*) AS rpm,
				SUM(prompt_tokens::bigint + completion_tokens::bigint) AS tpm
			FROM logs
			WHERE type = ? AND created_at >= ? AND created_at < ?
			GROUP BY created_at / 60
		),
		minute_buckets AS (
			SELECT
				minutes.minute,
				COALESCE(minute_usage.rpm, 0) AS rpm,
				COALESCE(minute_usage.tpm, 0) AS tpm
			FROM minutes
			LEFT JOIN minute_usage USING (minute)
		)
		SELECT
			COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY rpm), 0) AS rpm_p50,
			COALESCE(PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY rpm), 0) AS rpm_p90,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY rpm), 0) AS rpm_p99,
			COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY tpm), 0) AS tpm_p50,
			COALESCE(PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY tpm), 0) AS tpm_p90,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY tpm), 0) AS tpm_p99
		FROM minute_buckets`
	if err := LOG_DB.Raw(
		throughputQuery,
		analysisStart,
		analysisEnd,
		LogTypeConsume,
		analysisStart,
		analysisEnd,
	).Scan(&throughputRow).Error; err != nil {
		return stats, err
	}

	var concurrencyRow adminConsoleConcurrencyRow
	// Consume logs are written at completion, so rebuild time-weighted request
	// intervals and include the zero-concurrency segments across the selected range.
	concurrencyQuery := `
		WITH request_intervals AS (
			SELECT
				GREATEST(created_at - use_time, ?) AS start_at,
				LEAST(created_at, ?) AS end_at
			FROM logs
			WHERE type = ?
				AND use_time > 0
				AND created_at > ?
				AND created_at - use_time < ?
		),
		events AS (
			SELECT ?::bigint AS event_at, 0::bigint AS delta
			UNION ALL
			SELECT ?::bigint AS event_at, 0::bigint AS delta
			UNION ALL
			SELECT start_at AS event_at, 1::bigint AS delta
			FROM request_intervals
			WHERE start_at < end_at
			UNION ALL
			SELECT end_at AS event_at, -1::bigint AS delta
			FROM request_intervals
			WHERE start_at < end_at
		),
		event_totals AS (
			SELECT event_at, SUM(delta) AS delta
			FROM events
			GROUP BY event_at
		),
		segments AS (
			SELECT
				event_at,
				LEAD(event_at) OVER (ORDER BY event_at) AS next_event_at,
				SUM(delta) OVER (
					ORDER BY event_at
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) AS concurrency
			FROM event_totals
		),
		weighted AS (
			SELECT concurrency, SUM(next_event_at - event_at) AS duration
			FROM segments
			WHERE concurrency >= 0 AND next_event_at > event_at
			GROUP BY concurrency
		),
		ranked AS (
			SELECT
				concurrency,
				SUM(duration) OVER (ORDER BY concurrency) AS cumulative_duration,
				SUM(duration) OVER () AS total_duration
			FROM weighted
		)
		SELECT
			COALESCE(MIN(concurrency) FILTER (WHERE cumulative_duration >= total_duration * 0.50), 0) AS p50,
			COALESCE(MIN(concurrency) FILTER (WHERE cumulative_duration >= total_duration * 0.90), 0) AS p90,
			COALESCE(MIN(concurrency) FILTER (WHERE cumulative_duration >= total_duration * 0.95), 0) AS p95,
			COALESCE(MIN(concurrency) FILTER (WHERE cumulative_duration >= total_duration * 0.99), 0) AS p99
		FROM ranked`
	if err := LOG_DB.Raw(
		concurrencyQuery,
		concurrencyStart,
		concurrencyEnd,
		LogTypeConsume,
		concurrencyStart,
		concurrencyEnd,
		concurrencyStart,
		concurrencyEnd,
	).Scan(&concurrencyRow).Error; err != nil {
		return stats, err
	}

	stats.APIKeys = AdminConsoleKeyStats{Total: mainRow.APIKeysTotal, Active: logRow.ActiveKeysToday, Enabled: mainRow.APIKeysEnabled}
	stats.Channels = AdminConsoleChannelStats{Total: mainRow.ChannelsTotal, Enabled: mainRow.ChannelsEnabled, AutoDisabled: mainRow.ChannelsAutoDisabled}
	stats.Requests = AdminConsolePeriodStats{Today: logRow.RequestsToday, Month: logRow.RequestsMonth, Total: logRow.RequestsTotal}
	stats.Users = AdminConsoleUserStats{
		Today: mainRow.UsersToday, Total: mainRow.UsersTotal,
		ActiveToday: logRow.ActiveUsersToday, ActiveWeek: logRow.ActiveUsersWeek, ActiveMonth: logRow.ActiveUsersMonth,
	}
	stats.Tokens = AdminConsolePeriodStats{Today: logRow.TokensToday, Month: logRow.TokensMonth, Total: logRow.TokensTotal}
	stats.Quota = AdminConsolePeriodStats{Today: logRow.QuotaToday, Month: logRow.QuotaMonth, Total: logRow.QuotaTotal}
	stats.Revenue = AdminConsoleRevenueStats{Today: mainRow.RevenueToday, Month: mainRow.RevenueMonth, Total: mainRow.RevenueTotal}
	stats.Performance = AdminConsolePerformanceStats{
		AverageResponseSeconds:  logRow.ResponseAverageSeconds,
		TodayResponseP50Seconds: logRow.ResponseP50SecondsToday,
		TodayResponseP90Seconds: logRow.ResponseP90SecondsToday,
		TodayResponseP99Seconds: logRow.ResponseP99SecondsToday,
		TodayRPMP50:             throughputRow.RPMP50,
		TodayRPMP90:             throughputRow.RPMP90,
		TodayRPMP99:             throughputRow.RPMP99,
		TodayTPMP50:             throughputRow.TPMP50,
		TodayTPMP90:             throughputRow.TPMP90,
		TodayTPMP99:             throughputRow.TPMP99,
		ConcurrencyP50:          concurrencyRow.P50,
		ConcurrencyP90:          concurrencyRow.P90,
		ConcurrencyP99:          concurrencyRow.P99,
		MonthConcurrencyP50:     concurrencyRow.P50,
		MonthConcurrencyP90:     concurrencyRow.P90,
		MonthConcurrencyP95:     concurrencyRow.P95,
	}
	stats.SystemLoad = getAdminConsoleSystemLoad(false)
	return stats, nil
}

func GetAdminConsoleStats() (AdminConsoleStats, error) {
	now := time.Now()
	todayStart, tomorrowStart, _, _, _ := adminConsoleTimeBounds(now)
	return getCachedAdminConsoleStats(now, adminConsoleTimeRange{
		startTimestamp: todayStart,
		endTimestamp:   tomorrowStart,
	}, false)
}

// GetAdminConsoleStatsForRange returns console metrics for the inclusive Unix
// timestamp range. The range is normalized to an exclusive end internally.
func GetAdminConsoleStatsForRange(startTimestamp, endTimestamp int64) (AdminConsoleStats, error) {
	now := time.Now()
	selectedRange, err := normalizeAdminConsoleTimeRange(now, startTimestamp, endTimestamp)
	if err != nil {
		return AdminConsoleStats{}, err
	}
	return getCachedAdminConsoleStats(now, selectedRange, true)
}

func getCachedAdminConsoleStats(now time.Time, selectedRange adminConsoleTimeRange, customRange bool) (AdminConsoleStats, error) {
	key := adminConsoleStatsCacheKey{
		startTimestamp: selectedRange.startTimestamp,
		endTimestamp:   selectedRange.endTimestamp,
		customRange:    customRange,
	}
	adminConsoleStatsCache.Lock()
	defer adminConsoleStatsCache.Unlock()
	for key, entry := range adminConsoleStatsCache.entries {
		if !now.Before(entry.expiresAt) {
			delete(adminConsoleStatsCache.entries, key)
		}
	}
	if entry, ok := adminConsoleStatsCache.entries[key]; ok && now.Before(entry.expiresAt) {
		return entry.stats, nil
	}
	stats, err := getAdminConsoleStatsAtRange(now, selectedRange)
	if err != nil {
		return AdminConsoleStats{}, err
	}
	adminConsoleStatsCache.entries[key] = adminConsoleStatsCacheEntry{
		expiresAt: now.Add(adminConsoleStatsCacheTTL),
		stats:     stats,
	}
	return stats, nil
}

func getAdminConsoleRealtimeStatsAt(now time.Time) (AdminConsoleRealtimeStats, error) {
	var row adminConsoleRealtimeRow
	nowTimestamp := now.Unix()
	query := `
		SELECT
			COUNT(*) AS rpm,
			COALESCE(SUM(prompt_tokens::bigint + completion_tokens::bigint), 0) AS tpm,
			COALESCE(AVG(use_time) FILTER (WHERE use_time > 0), 0) AS response_seconds
		FROM logs
		WHERE type = ? AND created_at >= ? AND created_at < ?`
	if err := LOG_DB.Raw(
		query,
		LogTypeConsume,
		nowTimestamp-59,
		nowTimestamp+1,
	).Scan(&row).Error; err != nil {
		return AdminConsoleRealtimeStats{}, err
	}
	return AdminConsoleRealtimeStats{
		ResponseSeconds: row.ResponseSeconds,
		RPM:             row.RPM,
		TPM:             row.TPM,
	}, nil
}

func GetAdminConsoleRealtimeStats() (AdminConsoleRealtimeStats, error) {
	return getAdminConsoleRealtimeStatsAt(time.Now())
}

// GetAdminConsoleRealtimeStatsForRange validates the selected range while
// retaining the realtime endpoint's rolling 60-second snapshot semantics.
func GetAdminConsoleRealtimeStatsForRange(startTimestamp, endTimestamp int64) (AdminConsoleRealtimeStats, error) {
	if _, err := normalizeAdminConsoleTimeRange(time.Now(), startTimestamp, endTimestamp); err != nil {
		return AdminConsoleRealtimeStats{}, err
	}
	return GetAdminConsoleRealtimeStats()
}

func GetAdminConsoleSystemLoad() AdminConsoleSystemLoad {
	return getAdminConsoleSystemLoad(false)
}

func getAdminConsoleSystemLoad(refresh bool) AdminConsoleSystemLoad {
	systemStatus := common.GetSystemStatus()
	if refresh {
		systemStatus = common.RefreshSystemStatus()
	}
	return AdminConsoleSystemLoad{
		CPUUsagePercent:     systemStatus.CPUUsage,
		MemoryUsagePercent:  systemStatus.MemoryUsage,
		StorageUsagePercent: systemStatus.DiskUsage,
	}
}
