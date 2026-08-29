package model

import (
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// portableAdminConsoleLog contains only the columns needed by the fallback
// analytics path.  It deliberately avoids dialect-specific percentile and
// time-series SQL; SQLite and MySQL can therefore serve the same console API
// as PostgreSQL.
type portableAdminConsoleLog struct {
	CreatedAt        int64 `gorm:"column:created_at"`
	PromptTokens     int64 `gorm:"column:prompt_tokens"`
	CompletionTokens int64 `gorm:"column:completion_tokens"`
	Quota            int64 `gorm:"column:quota"`
	TokenID          int   `gorm:"column:token_id"`
	UserID           int   `gorm:"column:user_id"`
	UseTime          int   `gorm:"column:use_time"`
}

func getAdminConsoleStatsPortable(now time.Time) (AdminConsoleStats, error) {
	var stats AdminConsoleStats
	todayStart, tomorrowStart, weekStart, rollingMonthStart, monthStart := adminConsoleTimeBounds(now)
	metricEnd := now.Unix()

	var count int64
	if err := DB.Model(&Token{}).Where("deleted_at IS NULL").Count(&count).Error; err != nil {
		return stats, err
	}
	stats.APIKeys.Total = count
	if err := DB.Model(&Token{}).Where("deleted_at IS NULL AND status = ?", common.TokenStatusEnabled).Count(&count).Error; err != nil {
		return stats, err
	}
	stats.APIKeys.Enabled = count
	if err := DB.Model(&Channel{}).Count(&stats.Channels.Total).Error; err != nil {
		return stats, err
	}
	if err := DB.Model(&Channel{}).Where("status = ?", common.ChannelStatusEnabled).Count(&stats.Channels.Enabled).Error; err != nil {
		return stats, err
	}
	if err := DB.Model(&Channel{}).Where("status = ?", common.ChannelStatusAutoDisabled).Count(&stats.Channels.AutoDisabled).Error; err != nil {
		return stats, err
	}
	if err := DB.Model(&User{}).Where("created_at >= ? AND created_at < ?", todayStart, tomorrowStart).Count(&stats.Users.Today).Error; err != nil {
		return stats, err
	}
	if err := DB.Model(&User{}).Count(&stats.Users.Total).Error; err != nil {
		return stats, err
	}

	var topups []TopUp
	if err := DB.Where("status = ? AND (paid_cents > 0 OR money > 0) AND COALESCE(payment_method, '') <> ?", common.TopUpStatusSuccess, PaymentMethodBalance).Find(&topups).Error; err != nil {
		return stats, err
	}
	for _, topUp := range topups {
		completedAt := topUp.CreateTime
		if topUp.CompleteTime > 0 {
			completedAt = topUp.CompleteTime
		}
		amount := topUp.Money
		if topUp.PaidCents > 0 {
			amount = float64(topUp.PaidCents) / 100
		}
		stats.Revenue.Total += amount
		if completedAt >= todayStart && completedAt < tomorrowStart {
			stats.Revenue.Today += amount
		}
		if completedAt >= monthStart && completedAt < tomorrowStart {
			stats.Revenue.Month += amount
		}
	}

	var logRow struct {
		RequestsToday, RequestsMonth, RequestsTotal int64
		TokensToday, TokensMonth, TokensTotal       int64
		QuotaToday, QuotaMonth, QuotaTotal          int64
		ActiveKeysToday, ActiveUsersToday           int64
		ActiveUsersWeek, ActiveUsersMonth           int64
	}
	logQuery := `SELECT
		SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END) AS requests_today,
		SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END) AS requests_month,
		COUNT(*) AS requests_total,
		COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN prompt_tokens + completion_tokens ELSE 0 END), 0) AS tokens_today,
		COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN prompt_tokens + completion_tokens ELSE 0 END), 0) AS tokens_month,
		COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS tokens_total,
		COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN quota ELSE 0 END), 0) AS quota_today,
		COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN quota ELSE 0 END), 0) AS quota_month,
		COALESCE(SUM(quota), 0) AS quota_total,
		COUNT(DISTINCT CASE WHEN token_id <> 0 AND created_at >= ? AND created_at < ? THEN token_id END) AS active_keys_today,
		COUNT(DISTINCT CASE WHEN user_id <> 0 AND created_at >= ? AND created_at < ? THEN user_id END) AS active_users_today,
		COUNT(DISTINCT CASE WHEN user_id <> 0 AND created_at >= ? AND created_at < ? THEN user_id END) AS active_users_week,
		COUNT(DISTINCT CASE WHEN user_id <> 0 AND created_at >= ? AND created_at < ? THEN user_id END) AS active_users_month
		FROM logs WHERE type = ?`
	if err := LOG_DB.Raw(logQuery,
		todayStart, tomorrowStart, monthStart, tomorrowStart,
		todayStart, tomorrowStart, monthStart, tomorrowStart,
		todayStart, tomorrowStart, monthStart, tomorrowStart,
		todayStart, tomorrowStart, todayStart, tomorrowStart,
		weekStart, tomorrowStart, rollingMonthStart, tomorrowStart,
		LogTypeConsume).Scan(&logRow).Error; err != nil {
		return stats, err
	}
	stats.Requests = AdminConsolePeriodStats{Today: logRow.RequestsToday, Month: logRow.RequestsMonth, Total: logRow.RequestsTotal}
	stats.Tokens = AdminConsolePeriodStats{Today: logRow.TokensToday, Month: logRow.TokensMonth, Total: logRow.TokensTotal}
	stats.Quota = AdminConsolePeriodStats{Today: logRow.QuotaToday, Month: logRow.QuotaMonth, Total: logRow.QuotaTotal}
	stats.APIKeys.Active = logRow.ActiveKeysToday
	stats.Users.ActiveToday, stats.Users.ActiveWeek, stats.Users.ActiveMonth = logRow.ActiveUsersToday, logRow.ActiveUsersWeek, logRow.ActiveUsersMonth

	var responseTimes []int
	if err := LOG_DB.Model(&Log{}).Where("type = ? AND use_time > 0 AND created_at >= ? AND created_at < ?", LogTypeConsume, todayStart, tomorrowStart).Pluck("use_time", &responseTimes).Error; err != nil {
		return stats, err
	}
	stats.Performance.TodayResponseP50Seconds = portablePercentile(responseTimes, 0.50)
	stats.Performance.TodayResponseP90Seconds = portablePercentile(responseTimes, 0.90)
	stats.Performance.TodayResponseP99Seconds = portablePercentile(responseTimes, 0.99)

	var throughputLogs []portableAdminConsoleLog
	if err := LOG_DB.Model(&Log{}).Select("created_at, prompt_tokens, completion_tokens").Where("type = ? AND created_at >= ? AND created_at < ?", LogTypeConsume, todayStart, metricEnd).Find(&throughputLogs).Error; err != nil {
		return stats, err
	}
	rpm, tpm := portableThroughputPercentiles(throughputLogs, todayStart, metricEnd)
	stats.Performance.TodayRPMP50, stats.Performance.TodayRPMP90, stats.Performance.TodayRPMP99 = rpm[0], rpm[1], rpm[2]
	stats.Performance.TodayTPMP50, stats.Performance.TodayTPMP90, stats.Performance.TodayTPMP99 = tpm[0], tpm[1], tpm[2]

	var concurrencyLogs []portableAdminConsoleLog
	if err := LOG_DB.Model(&Log{}).Select("created_at, use_time").Where("type = ? AND use_time > 0 AND created_at >= ? AND created_at < ?", LogTypeConsume, monthStart, metricEnd).Find(&concurrencyLogs).Error; err != nil {
		return stats, err
	}
	concurrency := portableConcurrencyPercentiles(concurrencyLogs, monthStart, metricEnd)
	stats.Performance.MonthConcurrencyP50, stats.Performance.MonthConcurrencyP90, stats.Performance.MonthConcurrencyP95 = concurrency[0], concurrency[1], concurrency[2]
	stats.SystemLoad = getAdminConsoleSystemLoad(false)
	return stats, nil
}

func portablePercentile(values []int, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]int(nil), values...)
	sort.Ints(ordered)
	position := percentile * float64(len(ordered)-1)
	lower := int(position)
	upper := lower
	if upper+1 < len(ordered) {
		upper++
	}
	return float64(ordered[lower]) + (float64(ordered[upper])-float64(ordered[lower]))*(position-float64(lower))
}

func portableThroughputPercentiles(logs []portableAdminConsoleLog, start, end int64) ([3]float64, [3]float64) {
	var rpm, tpm [3]float64
	if end <= start {
		return rpm, tpm
	}
	buckets := make(map[int64][2]int64)
	for _, log := range logs {
		bucket := log.CreatedAt / 60
		values := buckets[bucket]
		values[0]++
		values[1] += log.PromptTokens + log.CompletionTokens
		buckets[bucket] = values
	}
	rpmValues := make([]int, 0)
	tpmValues := make([]int, 0)
	for bucket := start / 60; bucket <= (end-1)/60; bucket++ {
		values := buckets[bucket]
		rpmValues = append(rpmValues, int(values[0]))
		tpmValues = append(tpmValues, int(values[1]))
	}
	rpm[0], rpm[1], rpm[2] = portablePercentile(rpmValues, .50), portablePercentile(rpmValues, .90), portablePercentile(rpmValues, .99)
	tpm[0], tpm[1], tpm[2] = portablePercentile(tpmValues, .50), portablePercentile(tpmValues, .90), portablePercentile(tpmValues, .99)
	return rpm, tpm
}

func portableConcurrencyPercentiles(logs []portableAdminConsoleLog, start, end int64) [3]int64 {
	var result [3]int64
	if end <= start {
		return result
	}
	events := map[int64]int64{start: 0, end: 0}
	for _, log := range logs {
		intervalStart := log.CreatedAt - int64(log.UseTime)
		if intervalStart < start {
			intervalStart = start
		}
		intervalEnd := log.CreatedAt
		if intervalEnd > end {
			intervalEnd = end
		}
		if intervalStart >= intervalEnd {
			continue
		}
		events[intervalStart]++
		events[intervalEnd]--
	}
	times := make([]int64, 0, len(events))
	for timestamp := range events {
		times = append(times, timestamp)
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	durationByConcurrency := make(map[int64]int64)
	current := int64(0)
	for index, timestamp := range times {
		if index > 0 && timestamp > times[index-1] {
			durationByConcurrency[current] += timestamp - times[index-1]
		}
		current += events[timestamp]
	}
	totalDuration := end - start
	for _, percentile := range []float64{.50, .90, .95} {
		target := int64(float64(totalDuration) * percentile)
		if target < 1 {
			target = 1
		}
		var cumulative int64
		for concurrency := int64(0); ; concurrency++ {
			cumulative += durationByConcurrency[concurrency]
			if cumulative >= target {
				if percentile == .50 {
					result[0] = concurrency
				} else if percentile == .90 {
					result[1] = concurrency
				} else {
					result[2] = concurrency
				}
				break
			}
			if concurrency > int64(len(logs))+1 {
				break
			}
		}
	}
	return result
}

func getAdminConsoleRealtimeStatsPortable(now time.Time) (AdminConsoleRealtimeStats, error) {
	var row adminConsoleRealtimeRow
	query := `SELECT COUNT(*) AS rpm,
		COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS tpm,
		COALESCE(AVG(CASE WHEN use_time > 0 THEN use_time END), 0) AS response_seconds
		FROM logs WHERE type = ? AND created_at >= ? AND created_at < ?`
	if err := LOG_DB.Raw(query, LogTypeConsume, now.Unix()-59, now.Unix()+1).Scan(&row).Error; err != nil {
		return AdminConsoleRealtimeStats{}, err
	}
	return AdminConsoleRealtimeStats{ResponseSeconds: row.ResponseSeconds, RPM: row.RPM, TPM: row.TPM}, nil
}
