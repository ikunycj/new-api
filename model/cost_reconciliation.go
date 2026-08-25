package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// CostReconciliationRollup stores the low-write, query-oriented view of the
// final consume logs. Monetary values are integer micros of USD.
type CostReconciliationRollup struct {
	ID                         int64  `json:"id" gorm:"primaryKey"`
	BucketStart                int64  `json:"bucket_start" gorm:"bigint;uniqueIndex:idx_cost_rollup_bucket_dimension,priority:1;index"`
	UserID                     int    `json:"user_id" gorm:"uniqueIndex:idx_cost_rollup_bucket_dimension,priority:2;index"`
	TokenID                    int    `json:"token_id" gorm:"uniqueIndex:idx_cost_rollup_bucket_dimension,priority:3;index"`
	ChannelID                  int    `json:"channel_id" gorm:"uniqueIndex:idx_cost_rollup_bucket_dimension,priority:4;index"`
	Group                      string `json:"group" gorm:"type:varchar(64);uniqueIndex:idx_cost_rollup_bucket_dimension,priority:5;index"`
	RequestCount               int64  `json:"request_count"`
	UserChargeUSDMicros        int64  `json:"user_charge_usd_micros"`
	EstimatedCostUSDMicros     int64  `json:"estimated_cost_usd_micros"`
	SuccessfulCostUSDMicros    int64  `json:"successful_cost_usd_micros"`
	RetryCostUSDMicros         int64  `json:"retry_cost_usd_micros"`
	FailedPartialCostUSDMicros int64  `json:"failed_partial_cost_usd_micros"`
	DiffUSDMicros              int64  `json:"diff_usd_micros"`
	EstimatedCount             int64  `json:"estimated_count"`
	UnavailableCount           int64  `json:"unavailable_count"`
	UpdatedAt                  int64  `json:"updated_at" gorm:"bigint;index"`
}

type CostReconciliationQuery struct {
	StartTimestamp int64
	EndTimestamp   int64
	UserID         int
	TokenID        int
	ChannelID      int
	Group          string
	Keyword        string
	TokenName      string
	Limit          int
	Offset         int
}

type CostReconciliationTotals struct {
	RequestCount               int64 `json:"request_count"`
	UserChargeUSDMicros        int64 `json:"user_charge_usd_micros"`
	EstimatedCostUSDMicros     int64 `json:"estimated_cost_usd_micros"`
	SuccessfulCostUSDMicros    int64 `json:"successful_cost_usd_micros"`
	RetryCostUSDMicros         int64 `json:"retry_cost_usd_micros"`
	FailedPartialCostUSDMicros int64 `json:"failed_partial_cost_usd_micros"`
	DiffUSDMicros              int64 `json:"diff_usd_micros"`
	EstimatedCount             int64 `json:"estimated_count"`
	UnavailableCount           int64 `json:"unavailable_count"`
}

type costSnapshot struct {
	UserChargeUSDMicros        int64
	EstimatedCostUSDMicros     int64
	SuccessfulCostUSDMicros    int64
	RetryCostUSDMicros         int64
	FailedPartialCostUSDMicros int64
	Status                     string
}

func (s costSnapshot) available() bool { return s.Status == "estimated" }

func bucketStart(timestamp int64) int64 {
	return timestamp - timestamp%int64((time.Hour).Seconds())
}

func parseCostSnapshot(other string) costSnapshot {
	data, err := common.StrToMap(other)
	if err != nil {
		return costSnapshot{Status: "unavailable"}
	}
	raw, ok := data["cost_reconciliation"].(map[string]interface{})
	if !ok {
		return costSnapshot{Status: "unavailable"}
	}
	read := func(key string) int64 {
		value, ok := raw[key].(float64)
		if !ok {
			return 0
		}
		return int64(value)
	}
	return costSnapshot{
		UserChargeUSDMicros:        read("user_charge_usd_micros"),
		EstimatedCostUSDMicros:     read("estimated_cost_usd_micros"),
		SuccessfulCostUSDMicros:    read("successful_cost_usd_micros"),
		RetryCostUSDMicros:         read("retry_cost_usd_micros"),
		FailedPartialCostUSDMicros: read("failed_partial_cost_usd_micros"),
		Status:                     fmt.Sprint(raw["status"]),
	}
}

// RebuildCostReconciliationRollup recomputes a bounded UTC time window. It is
// idempotent and safe to retry after a worker crash.
func RebuildCostReconciliationRollup(startTimestamp, endTimestamp int64) (int64, error) {
	if LOG_DB == nil || DB == nil {
		return 0, errors.New("database is not initialized")
	}
	if endTimestamp <= startTimestamp {
		return 0, errors.New("invalid cost reconciliation window")
	}
	startBucket := bucketStart(startTimestamp)
	endBucket := bucketStart(endTimestamp - 1)
	var logs []Log
	// Rebuild complete buckets because old rows are replaced bucket-by-bucket;
	// reading partial edge buckets would otherwise delete data outside the read
	// range when a window starts in the middle of an hour.
	readStart := startBucket
	readEnd := endBucket + int64(time.Hour.Seconds())
	query := LOG_DB.Where("type = ? AND created_at >= ? AND created_at < ?", LogTypeConsume, readStart, readEnd)
	if err := query.Find(&logs).Error; err != nil {
		return 0, err
	}
	type key struct {
		Bucket, UserID, TokenID, ChannelID int64
		Group                              string
	}
	rollups := make(map[key]*CostReconciliationRollup)
	for _, log := range logs {
		snapshot := parseCostSnapshot(log.Other)
		k := key{bucketStart(log.CreatedAt), int64(log.UserId), int64(log.TokenId), int64(log.ChannelId), log.Group}
		row := rollups[k]
		if row == nil {
			row = &CostReconciliationRollup{BucketStart: k.Bucket, UserID: int(k.UserID), TokenID: int(k.TokenID), ChannelID: int(k.ChannelID), Group: k.Group}
			rollups[k] = row
		}
		row.RequestCount++
		row.UserChargeUSDMicros += snapshot.UserChargeUSDMicros
		row.EstimatedCostUSDMicros += snapshot.EstimatedCostUSDMicros
		row.SuccessfulCostUSDMicros += snapshot.SuccessfulCostUSDMicros
		row.RetryCostUSDMicros += snapshot.RetryCostUSDMicros
		row.FailedPartialCostUSDMicros += snapshot.FailedPartialCostUSDMicros
		row.DiffUSDMicros += snapshot.UserChargeUSDMicros - snapshot.EstimatedCostUSDMicros
		if snapshot.available() {
			row.EstimatedCount++
		} else {
			row.UnavailableCount++
		}
	}
	now := common.GetTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("bucket_start >= ? AND bucket_start <= ?", startBucket, endBucket).Delete(&CostReconciliationRollup{}).Error; err != nil {
			return err
		}
		for _, row := range rollups {
			row.UpdatedAt = now
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return int64(len(rollups)), err
}

func applyCostRollupFilters(tx *gorm.DB, query CostReconciliationQuery) *gorm.DB {
	if query.StartTimestamp > 0 {
		tx = tx.Where("bucket_start >= ?", bucketStart(query.StartTimestamp))
	}
	if query.EndTimestamp > 0 {
		tx = tx.Where("bucket_start < ?", bucketStart(query.EndTimestamp)+int64(time.Hour.Seconds()))
	}
	if query.UserID > 0 {
		tx = tx.Where("user_id = ?", query.UserID)
	}
	if query.TokenID > 0 {
		tx = tx.Where("token_id = ?", query.TokenID)
	}
	if query.ChannelID > 0 {
		tx = tx.Where("channel_id = ?", query.ChannelID)
	}
	if strings.TrimSpace(query.Group) != "" {
		tx = tx.Where(commonGroupCol+" = ?", strings.TrimSpace(query.Group))
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		pattern := "%" + escapeLikeLiteral(keyword) + "%"
		users := DB.Model(&User{}).Select("id").Where("(username LIKE ? ESCAPE '!' OR display_name LIKE ? ESCAPE '!' OR email LIKE ? ESCAPE '!' OR remark LIKE ? ESCAPE '!')", pattern, pattern, pattern, pattern)
		tx = tx.Where("user_id IN (?)", users)
	}
	if tokenName := strings.TrimSpace(query.TokenName); tokenName != "" {
		pattern := "%" + escapeLikeLiteral(tokenName) + "%"
		tokens := DB.Model(&Token{}).Select("id").Where("name LIKE ? ESCAPE '!'", pattern)
		tx = tx.Where("token_id IN (?)", tokens)
	}
	return tx
}

func ListCostReconciliationRollups(query CostReconciliationQuery) ([]CostReconciliationRollup, int64, CostReconciliationTotals, error) {
	if DB == nil {
		return nil, 0, CostReconciliationTotals{}, errors.New("database is not initialized")
	}
	base := applyCostRollupFilters(DB.Model(&CostReconciliationRollup{}), query)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, CostReconciliationTotals{}, err
	}
	var totals CostReconciliationTotals
	if err := base.Select("COALESCE(SUM(request_count),0) as request_count, COALESCE(SUM(user_charge_usd_micros),0) as user_charge_usd_micros, COALESCE(SUM(estimated_cost_usd_micros),0) as estimated_cost_usd_micros, COALESCE(SUM(successful_cost_usd_micros),0) as successful_cost_usd_micros, COALESCE(SUM(retry_cost_usd_micros),0) as retry_cost_usd_micros, COALESCE(SUM(failed_partial_cost_usd_micros),0) as failed_partial_cost_usd_micros, COALESCE(SUM(diff_usd_micros),0) as diff_usd_micros, COALESCE(SUM(estimated_count),0) as estimated_count, COALESCE(SUM(unavailable_count),0) as unavailable_count").Scan(&totals).Error; err != nil {
		return nil, 0, CostReconciliationTotals{}, err
	}
	limit := query.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	var rows []CostReconciliationRollup
	if err := base.Order("bucket_start desc, id desc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, CostReconciliationTotals{}, err
	}
	return rows, total, totals, nil
}
