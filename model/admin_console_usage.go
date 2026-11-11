package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// The admin console used to aggregate request, token and quota totals with one
// scan over the whole logs table. With months of history that table reaches
// tens of GB and the scan ran for half an hour, saturating disk IO for every
// other query. Usage before today now comes from quota_data, the hourly rollup
// written for every consume log, and only today's rows are read from logs
// through the created_at index.
//
// quota_data buckets are hour aligned and flushed every DataExportInterval
// minutes, so for a few minutes after midnight the previous day's last hour
// may still be missing from the totals.

const (
	// Applies to each PostgreSQL analytics query; a slow statement fails
	// instead of holding disk IO for minutes.
	adminConsoleQueryTimeoutMillis = 20000
	adminConsoleErrorBackoff       = 30 * time.Second
	// Month concurrency reads every request interval of the month, so it is
	// refreshed far less often than the rest of the console.
	adminConsoleConcurrencyCacheTTL = 15 * time.Minute
)

type adminConsoleUsageSums struct {
	Requests int64 `gorm:"column:requests"`
	Tokens   int64 `gorm:"column:tokens"`
	Quota    int64 `gorm:"column:quota"`
}

// adminConsoleQuery runs fn with a statement timeout on PostgreSQL. Other
// dialects run fn unchanged.
func adminConsoleQuery(db *gorm.DB, postgres bool, fn func(tx *gorm.DB) error) error {
	if !postgres {
		return fn(db)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(fmt.Sprintf("SET LOCAL statement_timeout = %d", adminConsoleQueryTimeoutMillis)).Error; err != nil {
			return err
		}
		return fn(tx)
	})
}

func adminConsoleLogQuery(fn func(tx *gorm.DB) error) error {
	return adminConsoleQuery(LOG_DB, common.UsingLogDatabase(common.DatabaseTypePostgreSQL), fn)
}

func adminConsoleMainQuery(fn func(tx *gorm.DB) error) error {
	return adminConsoleQuery(DB, common.UsingMainDatabase(common.DatabaseTypePostgreSQL), fn)
}

func adminConsoleLogSums(start, end int64) (adminConsoleUsageSums, error) {
	var sums adminConsoleUsageSums
	err := adminConsoleLogQuery(func(tx *gorm.DB) error {
		return tx.Model(&Log{}).
			Select("COUNT(*) AS requests, COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS tokens, COALESCE(SUM(quota), 0) AS quota").
			Where("type = ? AND created_at >= ? AND created_at < ?", LogTypeConsume, start, end).
			Scan(&sums).Error
	})
	return sums, err
}

// adminConsoleHistorySums totals consume usage in [start, end), which must end
// at or before today. When the data dashboard is disabled quota_data is not
// maintained, so it falls back to scanning logs.
func adminConsoleHistorySums(start, end int64) (adminConsoleUsageSums, error) {
	if !common.DataExportEnabled {
		return adminConsoleLogSums(start, end)
	}
	var sums adminConsoleUsageSums
	err := adminConsoleMainQuery(func(tx *gorm.DB) error {
		return tx.Model(&QuotaData{}).
			Select("COALESCE(SUM(count), 0) AS requests, COALESCE(SUM(token_used), 0) AS tokens, COALESCE(SUM(quota), 0) AS quota").
			Where("created_at >= ? AND created_at < ?", start, end).
			Scan(&sums).Error
	})
	return sums, err
}

func adminConsoleHistoryUserIDs(start, end int64) ([]int, error) {
	var ids []int
	if !common.DataExportEnabled {
		err := adminConsoleLogQuery(func(tx *gorm.DB) error {
			return tx.Model(&Log{}).Distinct().
				Where("type = ? AND user_id <> 0 AND created_at >= ? AND created_at < ?", LogTypeConsume, start, end).
				Pluck("user_id", &ids).Error
		})
		return ids, err
	}
	err := adminConsoleMainQuery(func(tx *gorm.DB) error {
		return tx.Model(&QuotaData{}).Distinct().
			Where("user_id <> 0 AND created_at >= ? AND created_at < ?", start, end).
			Pluck("user_id", &ids).Error
	})
	return ids, err
}

// getAdminConsoleUsage fills request, token and quota periods plus active key
// and user counts, reading logs only for today.
func getAdminConsoleUsage(now time.Time) (adminConsoleLogRow, error) {
	var row adminConsoleLogRow
	todayStart, tomorrowStart, weekStart, rollingMonthStart, monthStart := adminConsoleTimeBounds(now)

	today, err := adminConsoleLogSums(todayStart, tomorrowStart)
	if err != nil {
		return row, err
	}
	monthBeforeToday, err := adminConsoleHistorySums(monthStart, todayStart)
	if err != nil {
		return row, err
	}
	allBeforeToday, err := adminConsoleHistorySums(0, todayStart)
	if err != nil {
		return row, err
	}

	var todayUserIDs []int
	err = adminConsoleLogQuery(func(tx *gorm.DB) error {
		if err := tx.Model(&Log{}).Distinct().
			Where("type = ? AND user_id <> 0 AND created_at >= ? AND created_at < ?", LogTypeConsume, todayStart, tomorrowStart).
			Pluck("user_id", &todayUserIDs).Error; err != nil {
			return err
		}
		return tx.Model(&Log{}).
			Where("type = ? AND token_id <> 0 AND created_at >= ? AND created_at < ?", LogTypeConsume, todayStart, tomorrowStart).
			Distinct("token_id").
			Count(&row.ActiveKeysToday).Error
	})
	if err != nil {
		return row, err
	}
	weekUserIDs, err := adminConsoleHistoryUserIDs(weekStart, todayStart)
	if err != nil {
		return row, err
	}
	monthUserIDs, err := adminConsoleHistoryUserIDs(rollingMonthStart, todayStart)
	if err != nil {
		return row, err
	}

	row.RequestsToday, row.TokensToday, row.QuotaToday = today.Requests, today.Tokens, today.Quota
	row.RequestsMonth = monthBeforeToday.Requests + today.Requests
	row.TokensMonth = monthBeforeToday.Tokens + today.Tokens
	row.QuotaMonth = monthBeforeToday.Quota + today.Quota
	row.RequestsTotal = allBeforeToday.Requests + today.Requests
	row.TokensTotal = allBeforeToday.Tokens + today.Tokens
	row.QuotaTotal = allBeforeToday.Quota + today.Quota
	row.ActiveUsersToday = int64(len(todayUserIDs))
	row.ActiveUsersWeek = countDistinctUserIDs(weekUserIDs, todayUserIDs)
	row.ActiveUsersMonth = countDistinctUserIDs(monthUserIDs, todayUserIDs)
	return row, nil
}

func countDistinctUserIDs(lists ...[]int) int64 {
	seen := make(map[int]struct{})
	for _, ids := range lists {
		for _, id := range ids {
			seen[id] = struct{}{}
		}
	}
	return int64(len(seen))
}

var adminConsoleConcurrencyCache = struct {
	sync.Mutex
	expiresAt time.Time
	value     [3]int64
}{}

// cachedAdminConsoleConcurrency returns month concurrency percentiles, keeping
// the previous value when a refresh fails so one slow query cannot take the
// whole console down.
func cachedAdminConsoleConcurrency(now time.Time, compute func() ([3]int64, error)) [3]int64 {
	adminConsoleConcurrencyCache.Lock()
	defer adminConsoleConcurrencyCache.Unlock()
	if now.Before(adminConsoleConcurrencyCache.expiresAt) {
		return adminConsoleConcurrencyCache.value
	}
	value, err := compute()
	if err != nil {
		common.SysError("admin console month concurrency unavailable: " + err.Error())
		adminConsoleConcurrencyCache.expiresAt = now.Add(adminConsoleErrorBackoff)
		return adminConsoleConcurrencyCache.value
	}
	adminConsoleConcurrencyCache.value = value
	adminConsoleConcurrencyCache.expiresAt = now.Add(adminConsoleConcurrencyCacheTTL)
	return value
}
