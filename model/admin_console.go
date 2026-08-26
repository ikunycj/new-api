package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
)

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
	RPM                    int64   `json:"rpm"`
	TPM                    int64   `json:"tpm"`
	AverageResponseSeconds float64 `json:"average_response_seconds"`
	P90ResponseSeconds     float64 `json:"p90_response_seconds"`
	P99ResponseSeconds     float64 `json:"p99_response_seconds"`
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
	RequestsToday          int64   `gorm:"column:requests_today"`
	RequestsTotal          int64   `gorm:"column:requests_total"`
	TokensToday            int64   `gorm:"column:tokens_today"`
	TokensTotal            int64   `gorm:"column:tokens_total"`
	QuotaToday             int64   `gorm:"column:quota_today"`
	QuotaTotal             int64   `gorm:"column:quota_total"`
	ActiveKeysToday        int64   `gorm:"column:active_keys_today"`
	ActiveUsersToday       int64   `gorm:"column:active_users_today"`
	ActiveUsersWeek        int64   `gorm:"column:active_users_week"`
	ActiveUsersMonth       int64   `gorm:"column:active_users_month"`
	RPM                    int64   `gorm:"column:rpm"`
	TPM                    int64   `gorm:"column:tpm"`
	AverageResponseSeconds float64 `gorm:"column:average_response_seconds"`
	P90ResponseSeconds     float64 `gorm:"column:p90_response_seconds"`
	P99ResponseSeconds     float64 `gorm:"column:p99_response_seconds"`
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

func getAdminConsoleStatsAt(now time.Time) (AdminConsoleStats, error) {
	var stats AdminConsoleStats
	todayStart, tomorrowStart, weekStart, rollingMonthStart, monthStart := adminConsoleTimeBounds(now)

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
		todayStart,
		tomorrowStart,
		todayStart,
		tomorrowStart,
		monthStart,
		tomorrowStart,
		common.TopUpStatusSuccess,
		PaymentMethodBalance,
	).Scan(&mainRow).Error; err != nil {
		return stats, err
	}

	var logRow adminConsoleLogRow
	logQuery := `
		SELECT
			COUNT(*) FILTER (WHERE type = ? AND created_at >= ? AND created_at < ?) AS requests_today,
			COUNT(*) FILTER (WHERE type = ?) AS requests_total,
			COALESCE(SUM(prompt_tokens + completion_tokens) FILTER (WHERE type = ? AND created_at >= ? AND created_at < ?), 0) AS tokens_today,
			COALESCE(SUM(prompt_tokens + completion_tokens) FILTER (WHERE type = ?), 0) AS tokens_total,
			COALESCE(SUM(quota) FILTER (WHERE type = ? AND created_at >= ? AND created_at < ?), 0) AS quota_today,
			COALESCE(SUM(quota) FILTER (WHERE type = ?), 0) AS quota_total,
			COUNT(DISTINCT token_id) FILTER (WHERE type = ? AND token_id <> 0 AND created_at >= ? AND created_at < ?) AS active_keys_today,
			COUNT(DISTINCT user_id) FILTER (WHERE type = ? AND user_id <> 0 AND created_at >= ? AND created_at < ?) AS active_users_today,
			COUNT(DISTINCT user_id) FILTER (WHERE type = ? AND user_id <> 0 AND created_at >= ? AND created_at < ?) AS active_users_week,
			COUNT(DISTINCT user_id) FILTER (WHERE type = ? AND user_id <> 0 AND created_at >= ? AND created_at < ?) AS active_users_month,
			COUNT(*) FILTER (WHERE type = ? AND created_at >= ? AND created_at <= ?) AS rpm,
			COALESCE(SUM(prompt_tokens + completion_tokens) FILTER (WHERE type = ? AND created_at >= ? AND created_at <= ?), 0) AS tpm,
			COALESCE(AVG(use_time) FILTER (WHERE type = ? AND use_time > 0 AND created_at >= ? AND created_at < ?), 0) AS average_response_seconds,
			COALESCE(PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY use_time) FILTER (WHERE type = ? AND use_time > 0 AND created_at >= ? AND created_at < ?), 0) AS p90_response_seconds,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY use_time) FILTER (WHERE type = ? AND use_time > 0 AND created_at >= ? AND created_at < ?), 0) AS p99_response_seconds
		FROM logs`
	nowTimestamp := now.Unix()
	if err := LOG_DB.Raw(
		logQuery,
		LogTypeConsume, todayStart, tomorrowStart,
		LogTypeConsume,
		LogTypeConsume, todayStart, tomorrowStart,
		LogTypeConsume,
		LogTypeConsume, todayStart, tomorrowStart,
		LogTypeConsume,
		LogTypeConsume, todayStart, tomorrowStart,
		LogTypeConsume, todayStart, tomorrowStart,
		LogTypeConsume, weekStart, tomorrowStart,
		LogTypeConsume, rollingMonthStart, tomorrowStart,
		LogTypeConsume, nowTimestamp-60, nowTimestamp,
		LogTypeConsume, nowTimestamp-60, nowTimestamp,
		LogTypeConsume, todayStart, tomorrowStart,
		LogTypeConsume, todayStart, tomorrowStart,
		LogTypeConsume, todayStart, tomorrowStart,
	).Scan(&logRow).Error; err != nil {
		return stats, err
	}

	stats.APIKeys = AdminConsoleKeyStats{Total: mainRow.APIKeysTotal, Active: logRow.ActiveKeysToday, Enabled: mainRow.APIKeysEnabled}
	stats.Channels = AdminConsoleChannelStats{Total: mainRow.ChannelsTotal, Enabled: mainRow.ChannelsEnabled, AutoDisabled: mainRow.ChannelsAutoDisabled}
	stats.Requests = AdminConsolePeriodStats{Today: logRow.RequestsToday, Total: logRow.RequestsTotal}
	stats.Users = AdminConsoleUserStats{
		Today: mainRow.UsersToday, Total: mainRow.UsersTotal,
		ActiveToday: logRow.ActiveUsersToday, ActiveWeek: logRow.ActiveUsersWeek, ActiveMonth: logRow.ActiveUsersMonth,
	}
	stats.Tokens = AdminConsolePeriodStats{Today: logRow.TokensToday, Total: logRow.TokensTotal}
	stats.Quota = AdminConsolePeriodStats{Today: logRow.QuotaToday, Total: logRow.QuotaTotal}
	stats.Revenue = AdminConsoleRevenueStats{Today: mainRow.RevenueToday, Month: mainRow.RevenueMonth, Total: mainRow.RevenueTotal}
	stats.Performance = AdminConsolePerformanceStats{
		RPM:                    logRow.RPM,
		TPM:                    logRow.TPM,
		AverageResponseSeconds: logRow.AverageResponseSeconds,
		P90ResponseSeconds:     logRow.P90ResponseSeconds,
		P99ResponseSeconds:     logRow.P99ResponseSeconds,
	}
	stats.SystemLoad = getAdminConsoleSystemLoad(false)
	return stats, nil
}

func GetAdminConsoleStats() (AdminConsoleStats, error) {
	return getAdminConsoleStatsAt(time.Now())
}

func GetAdminConsoleSystemLoad() AdminConsoleSystemLoad {
	return getAdminConsoleSystemLoad(true)
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
