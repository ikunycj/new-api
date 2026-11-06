package model

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const logRollupStateID = 1

// ChannelUsageRollup is one complete local calendar day for one channel.
// Costs are reduced to additive values, so the table has no pricing dimensions.
type ChannelUsageRollup struct {
	BucketStart      int64   `gorm:"primaryKey;autoIncrement:false;index:idx_channel_usage_day;index:idx_channel_usage_channel_day,priority:1"`
	ChannelID        int     `gorm:"primaryKey;autoIncrement:false;index:idx_channel_usage_channel_day,priority:2"`
	RequestCount     int64   `gorm:"not null;default:0"`
	PromptTokens     int64   `gorm:"not null;default:0"`
	CompletionTokens int64   `gorm:"not null;default:0"`
	Quota            int64   `gorm:"not null;default:0"`
	BaseCostUSD      float64 `gorm:"not null;default:0"`
	BaseCostCNY      float64 `gorm:"not null;default:0"`
	CostIncomplete   bool    `gorm:"not null;default:false"`
	UpdatedAt        int64   `gorm:"not null;index"`
}

// LogRollupState is one progress row, not a statistics table.
// ReadyBefore is the first local day that is not complete. ExcludedBefore is
// an administrator-owned deletion boundary that the worker must never rebuild.
type LogRollupState struct {
	ID             int   `gorm:"primaryKey;autoIncrement:false"`
	ReadyBefore    int64 `gorm:"not null;default:0"`
	ExcludedBefore int64 `gorm:"not null;default:0"`
	UpdatedAt      int64 `gorm:"not null"`
}

func (LogRollupState) TableName() string { return "log_rollup_state" }

type LogRollupResult struct {
	DaysProcessed int   `json:"days_processed"`
	RowsProcessed int   `json:"rows_processed"`
	ReadyBefore   int64 `json:"ready_before"`
}

func logRollupEnabled() bool {
	return common.UsingLogDatabase(common.DatabaseTypePostgreSQL) &&
		common.GetEnvOrDefaultBool("LOG_ROLLUP_ENABLED", true)
}

func localDayStart(t time.Time) int64 {
	local := t.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local).Unix()
}

func nextLocalDay(start int64) int64 {
	return time.Unix(start, 0).In(time.Local).AddDate(0, 0, 1).Unix()
}

func ensureLogRollupState(tx *gorm.DB, cutoff int64) (*LogRollupState, error) {
	initialReady := cutoff
	var firstLog struct{ CreatedAt int64 }
	if err := tx.Model(&Log{}).Where("type = ?", LogTypeConsume).Order("created_at ASC").Select("created_at").Limit(1).Scan(&firstLog).Error; err != nil {
		return nil, err
	}
	if firstLog.CreatedAt > 0 {
		initialReady = localDayStart(time.Unix(firstLog.CreatedAt, 0))
	}
	if err := tx.Exec(`INSERT INTO log_rollup_state (id, ready_before, excluded_before, updated_at)
        VALUES (?, ?, 0, ?) ON CONFLICT (id) DO NOTHING`, logRollupStateID, initialReady, common.GetTimestamp()).Error; err != nil {
		return nil, err
	}
	var state LogRollupState
	if err := lockForUpdate(tx).First(&state, logRollupStateID).Error; err != nil {
		return nil, err
	}
	return &state, nil
}

func getLogRollupState(ctx context.Context) (LogRollupState, error) {
	var state LogRollupState
	if !logRollupEnabled() {
		return state, errors.New("log rollup is disabled")
	}
	err := LOG_DB.WithContext(ctx).Where("id = ?", logRollupStateID).First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cutoff := localDayStart(time.Now().AddDate(0, 0, -1))
		err = LOG_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			_, createErr := ensureLogRollupState(tx, cutoff)
			return createErr
		})
		if err == nil {
			err = LOG_DB.WithContext(ctx).Where("id = ?", logRollupStateID).First(&state).Error
		}
	}
	return state, err
}

func parseFrozenPricing(raw string) (float64, float64, bool) {
	groupRatio := 1.0
	billingRate := 1.0
	var values map[string]json.RawMessage
	if strings.TrimSpace(raw) == "" || common.Unmarshal([]byte(raw), &values) != nil {
		return 0, 0, true
	}
	groupText := rawJSONText(values["group_ratio"])
	if groupText == "" {
		return 0, 0, true
	}
	if value := groupText; value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || parsed <= 0 || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, 0, true
		}
		groupRatio = parsed
	}
	if value := rawJSONText(values["billing_usd_to_cny_rate"]); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || parsed <= 0 || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, 0, true
		}
		billingRate = parsed
	}
	return groupRatio, billingRate, common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0)
}

func rawJSONText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if raw[0] == '"' && common.Unmarshal(raw, &text) == nil {
		return text
	}
	return string(raw)
}

type rollupAccumulator struct {
	ChannelUsageRollup
}

func addRollupLog(rows map[int]*rollupAccumulator, log *Log, now int64) {
	if log == nil || log.ChannelId <= 0 {
		return
	}
	row := rows[log.ChannelId]
	if row == nil {
		row = &rollupAccumulator{ChannelUsageRollup: ChannelUsageRollup{BucketStart: localDayStart(time.Unix(log.CreatedAt, 0)), ChannelID: log.ChannelId}}
		rows[log.ChannelId] = row
	}
	row.RequestCount++
	row.PromptTokens += int64(log.PromptTokens)
	row.CompletionTokens += int64(log.CompletionTokens)
	row.Quota += int64(log.Quota)
	groupRatio, billingRate, incomplete := parseFrozenPricing(log.Other)
	if incomplete || log.Quota < 0 {
		row.CostIncomplete = true
	} else {
		cny := float64(log.Quota) / common.QuotaPerUnit / groupRatio
		row.BaseCostCNY += cny
		row.BaseCostUSD += cny / billingRate
	}
	row.UpdatedAt = now
}

func rebuildLogRollupDay(ctx context.Context, dayStart int64) (int, error) {
	var processed int
	err := LOG_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cutoff := localDayStart(time.Now().AddDate(0, 0, -1))
		state, err := ensureLogRollupState(tx, cutoff)
		if err != nil {
			return err
		}
		if dayStart < state.ExcludedBefore || dayStart != state.ReadyBefore {
			return nil
		}
		// Aggregate additive fields in PostgreSQL first. Grouping by the raw
		// pricing JSON keeps malformed historical values tolerant: the small
		// result set is parsed by the existing Go helper below.
		type aggregateRow struct {
			ChannelID        int
			Other            string
			RequestCount     int64
			PromptTokens     int64
			CompletionTokens int64
			Quota            int64
		}
		var aggregateRows []aggregateRow
		if err := tx.Model(&Log{}).
			Select(`channel_id, other,
				COUNT(*) AS request_count,
				COALESCE(SUM(prompt_tokens::bigint), 0) AS prompt_tokens,
				COALESCE(SUM(completion_tokens::bigint), 0) AS completion_tokens,
				COALESCE(SUM(quota::bigint), 0) AS quota`).
			Where("type = ? AND created_at >= ? AND created_at < ? AND channel_id > 0", LogTypeConsume, dayStart, nextLocalDay(dayStart)).
			Group("channel_id, other").
			Scan(&aggregateRows).Error; err != nil {
			return err
		}
		rows := make(map[int]*rollupAccumulator)
		now := common.GetTimestamp()
		for _, aggregate := range aggregateRows {
			row := rows[aggregate.ChannelID]
			if row == nil {
				row = &rollupAccumulator{ChannelUsageRollup: ChannelUsageRollup{
					BucketStart: dayStart,
					ChannelID:   aggregate.ChannelID,
				}}
				rows[aggregate.ChannelID] = row
			}
			row.RequestCount += aggregate.RequestCount
			row.PromptTokens += aggregate.PromptTokens
			row.CompletionTokens += aggregate.CompletionTokens
			row.Quota += aggregate.Quota
			groupRatio, billingRate, incomplete := parseFrozenPricing(aggregate.Other)
			if incomplete || aggregate.Quota < 0 {
				row.CostIncomplete = true
			} else {
				cny := float64(aggregate.Quota) / common.QuotaPerUnit / groupRatio
				row.BaseCostCNY += cny
				row.BaseCostUSD += cny / billingRate
			}
			row.UpdatedAt = now
		}
		if err := tx.Where("bucket_start = ?", dayStart).Delete(&ChannelUsageRollup{}).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if err := tx.Create(&row.ChannelUsageRollup).Error; err != nil {
				return err
			}
			processed += int(row.RequestCount)
		}
		return tx.Model(&LogRollupState{}).Where("id = ?", logRollupStateID).Updates(map[string]any{
			"ready_before": nextLocalDay(dayStart),
			"updated_at":   common.GetTimestamp(),
		}).Error
	})
	return processed, err
}

// RunLogRollup processes complete local days until the time budget expires.
func RunLogRollup(ctx context.Context, budget time.Duration) (LogRollupResult, error) {
	result := LogRollupResult{}
	if !logRollupEnabled() {
		return result, nil
	}
	if budget <= 0 {
		budget = 30 * time.Second
	}
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		state, err := getLogRollupState(ctx)
		if err != nil {
			return result, err
		}
		cutoff := localDayStart(time.Now().AddDate(0, 0, -1))
		if state.ReadyBefore >= cutoff {
			result.ReadyBefore = state.ReadyBefore
			return result, nil
		}
		processed, err := rebuildLogRollupDay(ctx, state.ReadyBefore)
		if err != nil {
			return result, err
		}
		result.DaysProcessed++
		result.RowsProcessed += processed
	}
	state, err := getLogRollupState(ctx)
	if err == nil {
		result.ReadyBefore = state.ReadyBefore
	}
	return result, err
}

func queryChannelUsageFromRollup(channelIDs []int, state LogRollupState, now time.Time) (map[int]ChannelUsage, error) {
	usage := make(map[int]ChannelUsage, len(channelIDs))
	if len(channelIDs) == 0 {
		return usage, nil
	}
	readyBefore := state.ReadyBefore
	todayStart := localDayStart(now)
	if readyBefore <= 0 || readyBefore > todayStart {
		readyBefore = todayStart
	}
	query := LOG_DB.Table("channel_usage_rollups").Where("channel_id IN ? AND bucket_start < ?", channelIDs, readyBefore)
	if state.ExcludedBefore > 0 {
		query = query.Where("bucket_start >= ?", state.ExcludedBefore)
	}
	var rows []ChannelUsageRollup
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		value := usage[row.ChannelID]
		value.TotalTokens += row.PromptTokens + row.CompletionTokens
		value.TotalBaseCostUSD += row.BaseCostUSD
		value.TotalBaseCostCNY += row.BaseCostCNY
		value.TotalCostIncomplete = value.TotalCostIncomplete || row.CostIncomplete
		usage[row.ChannelID] = value
	}
	for _, id := range channelIDs {
		if _, ok := usage[id]; !ok {
			usage[id] = ChannelUsage{}
		}
	}
	return usage, nil
}

// GetLogRollupState is used by statistics and maintenance boundaries.
func GetLogRollupState(ctx context.Context) (LogRollupState, error) {
	return getLogRollupState(ctx)
}

// DeleteChannelUsageRollupsBefore permanently removes aggregate buckets and
// advances both state boundaries so the worker cannot recreate them.
func DeleteChannelUsageRollupsBefore(ctx context.Context, cutoff int64) (int64, error) {
	if !logRollupEnabled() || cutoff <= 0 {
		return 0, nil
	}
	todayStart := localDayStart(time.Now())
	if cutoff > todayStart || cutoff != localDayStart(time.Unix(cutoff, 0)) {
		return 0, errors.New("rollup cleanup cutoff must be a local calendar day no later than today")
	}
	var deleted int64
	err := LOG_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		state, err := ensureLogRollupState(tx, localDayStart(time.Now().AddDate(0, 0, -1)))
		if err != nil {
			return err
		}
		result := tx.Where("bucket_start < ?", cutoff).Delete(&ChannelUsageRollup{})
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected
		updates := map[string]any{"updated_at": common.GetTimestamp()}
		if cutoff > state.ExcludedBefore {
			updates["excluded_before"] = cutoff
		}
		if cutoff > state.ReadyBefore {
			updates["ready_before"] = cutoff
		}
		return tx.Model(&LogRollupState{}).Where("id = ?", logRollupStateID).Updates(updates).Error
	})
	return deleted, err
}

// CanDeleteLogsBefore prevents source logs from being removed before their
// corresponding complete days have either been aggregated or excluded.
func CanDeleteLogsBefore(ctx context.Context, targetTimestamp int64) (bool, error) {
	if !logRollupEnabled() {
		return true, nil
	}
	state, err := getLogRollupState(ctx)
	if err != nil {
		return false, err
	}
	return state.ReadyBefore >= requiredRollupBoundaryForLogCleanup(targetTimestamp), nil
}

// requiredRollupBoundaryForLogCleanup returns the first day that must remain
// outside the raw-log cleanup range. A cutoff at midnight excludes the whole
// day and needs no aggregation for it; a cutoff inside a day includes part of
// that day and therefore requires the complete day rollup first.
func requiredRollupBoundaryForLogCleanup(targetTimestamp int64) int64 {
	dayStart := localDayStart(time.Unix(targetTimestamp, 0))
	if targetTimestamp == dayStart {
		return dayStart
	}
	return nextLocalDay(dayStart)
}
