package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type AdminConsoleKeyStats struct {
	Total   int64 `json:"total"`
	Active  int64 `json:"active"`
	Enabled int64 `json:"enabled"`
}

type AdminConsoleAccountStats struct {
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
}

type AdminConsoleStats struct {
	APIKeys     AdminConsoleKeyStats         `json:"api_keys"`
	Accounts    AdminConsoleAccountStats     `json:"accounts"`
	Requests    AdminConsolePeriodStats      `json:"requests"`
	Users       AdminConsoleUserStats        `json:"users"`
	Tokens      AdminConsolePeriodStats      `json:"tokens"`
	Quota       AdminConsolePeriodStats      `json:"quota"`
	Revenue     AdminConsoleRevenueStats     `json:"revenue"`
	Performance AdminConsolePerformanceStats `json:"performance"`
}

type adminConsoleLogTotals struct {
	Tokens int64 `gorm:"column:tokens"`
	Quota  int64 `gorm:"column:quota"`
}

type adminConsolePerformanceRow struct {
	RPM                    int64   `gorm:"column:rpm"`
	TPM                    int64   `gorm:"column:tpm"`
	AverageResponseSeconds float64 `gorm:"column:average_response_seconds"`
}

type adminConsoleActiveUsersRow struct {
	Active int64 `gorm:"column:active"`
}

type adminConsoleActiveKeysRow struct {
	Active int64 `gorm:"column:active"`
}

type adminConsoleRevenueRow struct {
	Amount float64 `gorm:"column:amount"`
}

func adminConsoleConsumeQuery(startTimestamp, endTimestamp int64) *gorm.DB {
	query := LOG_DB.Table("logs").Where("type = ?", LogTypeConsume)
	if startTimestamp != 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		query = query.Where("created_at < ?", endTimestamp)
	}
	return query
}

func getAdminConsoleLogTotals(startTimestamp, endTimestamp int64) (adminConsoleLogTotals, error) {
	var totals adminConsoleLogTotals
	err := adminConsoleConsumeQuery(startTimestamp, endTimestamp).
		Select("COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0) AS tokens, COALESCE(SUM(quota), 0) AS quota").
		Scan(&totals).Error
	return totals, err
}

func countAdminConsoleLogs(startTimestamp, endTimestamp int64) (int64, error) {
	var count int64
	err := adminConsoleConsumeQuery(startTimestamp, endTimestamp).Count(&count).Error
	return count, err
}

func countAdminConsoleActiveUsers(startTimestamp, endTimestamp int64) (int64, error) {
	var activeUsers adminConsoleActiveUsersRow
	err := adminConsoleConsumeQuery(startTimestamp, endTimestamp).
		Where("user_id <> 0").
		Select("COUNT(DISTINCT user_id) AS active").
		Scan(&activeUsers).Error
	return activeUsers.Active, err
}

func countAdminConsoleActiveKeys(startTimestamp, endTimestamp int64) (int64, error) {
	var activeKeys adminConsoleActiveKeysRow
	err := adminConsoleConsumeQuery(startTimestamp, endTimestamp).
		Where("token_id <> 0").
		Select("COUNT(DISTINCT token_id) AS active").
		Scan(&activeKeys).Error
	return activeKeys.Active, err
}

func getAdminConsoleRevenue(startTimestamp, endTimestamp int64) (float64, error) {
	query := DB.Model(&TopUp{}).
		Where("status = ?", common.TopUpStatusSuccess).
		Where("money > 0").
		Where("payment_method IS NULL OR payment_method <> ?", PaymentMethodBalance)
	// A few legacy successful orders predate complete_time. Use their creation
	// time as a fallback so historical income is not silently omitted.
	completedAt := "CASE WHEN complete_time > 0 THEN complete_time ELSE create_time END"
	if startTimestamp != 0 {
		query = query.Where(completedAt+" >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		query = query.Where(completedAt+" < ?", endTimestamp)
	}
	var revenue adminConsoleRevenueRow
	err := query.Select("COALESCE(SUM(money), 0) AS amount").Scan(&revenue).Error
	return revenue.Amount, err
}

func getAdminConsoleStatsAt(now time.Time) (AdminConsoleStats, error) {
	var stats AdminConsoleStats
	localNow := now.In(time.Local)
	startOfDay := time.Date(
		localNow.Year(),
		localNow.Month(),
		localNow.Day(),
		0,
		0,
		0,
		0,
		time.Local,
	)
	startTimestamp := startOfDay.Unix()
	nextDayTimestamp := startOfDay.AddDate(0, 0, 1).Unix()
	weekStartTimestamp := startOfDay.AddDate(0, 0, -6).Unix()
	monthStartTimestamp := startOfDay.AddDate(0, 0, -29).Unix()
	startOfMonth := time.Date(
		localNow.Year(),
		localNow.Month(),
		1,
		0,
		0,
		0,
		0,
		time.Local,
	)
	startOfMonthTimestamp := startOfMonth.Unix()
	var err error

	if err := DB.Model(&Token{}).Count(&stats.APIKeys.Total).Error; err != nil {
		return stats, err
	}
	if stats.APIKeys.Active, err = countAdminConsoleActiveKeys(startTimestamp, nextDayTimestamp); err != nil {
		return stats, err
	}
	if err := DB.Model(&Token{}).Where("status = ?", common.TokenStatusEnabled).Count(&stats.APIKeys.Enabled).Error; err != nil {
		return stats, err
	}

	if err := DB.Model(&Channel{}).Count(&stats.Accounts.Total).Error; err != nil {
		return stats, err
	}
	if err := DB.Model(&Channel{}).Where("status = ?", common.ChannelStatusEnabled).Count(&stats.Accounts.Enabled).Error; err != nil {
		return stats, err
	}
	if err := DB.Model(&Channel{}).Where("status = ?", common.ChannelStatusAutoDisabled).Count(&stats.Accounts.AutoDisabled).Error; err != nil {
		return stats, err
	}

	if err := DB.Unscoped().Model(&User{}).Count(&stats.Users.Total).Error; err != nil {
		return stats, err
	}
	if err := DB.Unscoped().Model(&User{}).
		Where("created_at >= ? AND created_at < ?", startTimestamp, nextDayTimestamp).
		Count(&stats.Users.Today).Error; err != nil {
		return stats, err
	}

	if stats.Requests.Total, err = countAdminConsoleLogs(0, 0); err != nil {
		return stats, err
	}
	if stats.Requests.Today, err = countAdminConsoleLogs(startTimestamp, nextDayTimestamp); err != nil {
		return stats, err
	}

	totalLogTotals, err := getAdminConsoleLogTotals(0, 0)
	if err != nil {
		return stats, err
	}
	todayLogTotals, err := getAdminConsoleLogTotals(startTimestamp, nextDayTimestamp)
	if err != nil {
		return stats, err
	}
	stats.Tokens.Total = totalLogTotals.Tokens
	stats.Tokens.Today = todayLogTotals.Tokens
	stats.Quota.Total = totalLogTotals.Quota
	stats.Quota.Today = todayLogTotals.Quota

	if stats.Users.ActiveToday, err = countAdminConsoleActiveUsers(startTimestamp, nextDayTimestamp); err != nil {
		return stats, err
	}
	if stats.Users.ActiveWeek, err = countAdminConsoleActiveUsers(weekStartTimestamp, nextDayTimestamp); err != nil {
		return stats, err
	}
	if stats.Users.ActiveMonth, err = countAdminConsoleActiveUsers(monthStartTimestamp, nextDayTimestamp); err != nil {
		return stats, err
	}

	if stats.Revenue.Today, err = getAdminConsoleRevenue(startTimestamp, nextDayTimestamp); err != nil {
		return stats, err
	}
	if stats.Revenue.Month, err = getAdminConsoleRevenue(startOfMonthTimestamp, nextDayTimestamp); err != nil {
		return stats, err
	}
	if stats.Revenue.Total, err = getAdminConsoleRevenue(0, 0); err != nil {
		return stats, err
	}

	var performance adminConsolePerformanceRow
	if err := adminConsoleConsumeQuery(localNow.Unix()-60, 0).
		Select("COUNT(*) AS rpm, COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0) AS tpm, 0 AS average_response_seconds").
		Scan(&performance).Error; err != nil {
		return stats, err
	}
	stats.Performance.RPM = performance.RPM
	stats.Performance.TPM = performance.TPM
	if err := adminConsoleConsumeQuery(startTimestamp, nextDayTimestamp).
		Where("use_time > 0").
		Select("COALESCE(AVG(use_time), 0) AS average_response_seconds").
		Scan(&performance).Error; err != nil {
		return stats, err
	}
	stats.Performance.AverageResponseSeconds = performance.AverageResponseSeconds

	return stats, nil
}

func GetAdminConsoleStats() (AdminConsoleStats, error) {
	return getAdminConsoleStatsAt(time.Now())
}
