package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ChannelHealthHistory persists the channel health score as a time series so the
// dashboard can show a trend that survives a page reload, a different browser or
// a server restart. The live score itself stays in memory and off the hot path;
// this table is written by a periodic flush and is never read by routing.
//
// Unlike perf_metrics, which accumulates counters and averages them at read
// time, a health score is a gauge: adding two scores together is meaningless.
// Each bucket therefore stores a running mean, which requires carrying the
// sample count so the upsert can compute an incremental weighted average rather
// than just adding. The alternative -- last-write-wins -- would make the series
// depend on which node happened to flush last.
type ChannelHealthHistory struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	ChannelId int    `json:"channel_id" gorm:"uniqueIndex:idx_chh_channel_route_family_bucket,priority:1;index:idx_chh_channel"`
	Route     string `json:"route" gorm:"size:128;uniqueIndex:idx_chh_channel_route_family_bucket,priority:2"`
	Family    string `json:"family" gorm:"size:64;uniqueIndex:idx_chh_channel_route_family_bucket,priority:3"`
	BucketTs  int64  `json:"bucket_ts" gorm:"uniqueIndex:idx_chh_channel_route_family_bucket,priority:4;index:idx_chh_bucket_ts"`

	// Gauges, stored as the mean over the bucket.
	Score        float64 `json:"score" gorm:"default:0"`
	Availability float64 `json:"availability" gorm:"default:0"`
	LatencyScore float64 `json:"latency_score" gorm:"default:0"`
	LatencyMs    float64 `json:"latency_ms" gorm:"default:0"`
	Samples      float64 `json:"samples" gorm:"default:0"`

	// Confident counts how many observations in this bucket were confident,
	// so the UI can distinguish "trusted low score" from "not enough data".
	ConfidentCount int64 `json:"confident_count" gorm:"default:0"`
	// ObservationCount is the divisor behind every mean above.
	ObservationCount int64 `json:"observation_count" gorm:"default:0"`
}

func (ChannelHealthHistory) TableName() string {
	return "channel_health_histories"
}

// UpsertChannelHealthHistory folds one observation into its bucket, maintaining
// a running mean per gauge.
//
// The incremental mean is computed inside the SQL statement so concurrent nodes
// cannot lose an update to a read-modify-write race:
//
//	new_mean = (old_mean * old_n + value * n) / (old_n + n)
//
// Note the deliberate ordering: observation_count is updated last, because the
// gauge expressions reference its pre-update value. Postgres and MySQL both
// evaluate all right-hand sides against the existing row, so this is safe, but
// the ordering is kept explicit to document the dependency.
func UpsertChannelHealthHistory(entry *ChannelHealthHistory) error {
	if entry == nil {
		return nil
	}
	if entry.ObservationCount <= 0 {
		return nil
	}
	if entry.ChannelId <= 0 {
		return errors.New("channel health history requires a channel id")
	}

	table := entry.TableName()
	weightedMean := func(column string, incoming float64) clause.Expr {
		return gorm.Expr(
			"("+table+"."+column+" * "+table+".observation_count + ?) / ("+table+".observation_count + ?)",
			incoming*float64(entry.ObservationCount),
			entry.ObservationCount,
		)
	}

	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "channel_id"},
			{Name: "route"},
			{Name: "family"},
			{Name: "bucket_ts"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"score":         weightedMean("score", entry.Score),
			"availability":  weightedMean("availability", entry.Availability),
			"latency_score": weightedMean("latency_score", entry.LatencyScore),
			"latency_ms":    weightedMean("latency_ms", entry.LatencyMs),
			"samples":       weightedMean("samples", entry.Samples),
			"confident_count": gorm.Expr(
				table+".confident_count + ?", entry.ConfidentCount),
			"observation_count": gorm.Expr(
				table+".observation_count + ?", entry.ObservationCount),
		}),
	}).Create(entry).Error
}

// GetChannelHealthHistory returns every bucket in the window, oldest first, so
// the frontend can plot it without re-sorting.
func GetChannelHealthHistory(startTs int64, endTs int64) ([]ChannelHealthHistory, error) {
	var rows []ChannelHealthHistory
	if DB == nil {
		return rows, nil
	}
	err := DB.Model(&ChannelHealthHistory{}).
		Where("bucket_ts >= ? AND bucket_ts <= ?", startTs, endTs).
		Order("bucket_ts ASC").
		Find(&rows).Error
	return rows, err
}

// DeleteChannelHealthHistoryBefore enforces retention. Without it this table
// grows without bound, which matters more here than for perf_metrics because
// every tracked channel/route/family triple writes a row every bucket whether
// or not it saw traffic.
func DeleteChannelHealthHistoryBefore(cutoffTs int64) error {
	if cutoffTs <= 0 || DB == nil {
		return nil
	}
	return DB.Where("bucket_ts < ?", cutoffTs).Delete(&ChannelHealthHistory{}).Error
}

// ChannelHealthHistoryBucket truncates a unix timestamp to the start of its
// bucket. Exported so the flush loop and any test agree on bucket boundaries.
func ChannelHealthHistoryBucket(ts int64, bucketSeconds int) int64 {
	if bucketSeconds <= 0 {
		bucketSeconds = 60
	}
	return ts - ts%int64(bucketSeconds)
}

// ChannelHealthHistoryStartTime converts a lookback window in hours into a unix
// timestamp, defaulting to 24h.
func ChannelHealthHistoryStartTime(hours int) int64 {
	if hours <= 0 {
		hours = 24
	}
	return time.Now().Add(-time.Duration(hours) * time.Hour).Unix()
}
