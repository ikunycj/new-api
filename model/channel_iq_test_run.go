package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	IQTestSucceeded    = "success"
	IQTestFailed       = "error"
	IQTestInconclusive = "inconclusive"
)

// ChannelIQTestRun belongs to the requested model, including models other than the routing probe model.
type ChannelIQTestRun struct {
	ID           int64     `json:"id" gorm:"primaryKey;index:idx_channel_iq_latest,priority:4,sort:desc"`
	ChannelID    int       `json:"channel_id" gorm:"not null;index:idx_channel_iq_latest,priority:1"`
	TestModel    string    `json:"model" gorm:"not null;index:idx_channel_iq_latest,priority:2"`
	Engine       string    `json:"engine" gorm:"not null"`
	Source       string    `json:"source" gorm:"not null"`
	CompletedAt  int64     `json:"completed_at" gorm:"not null;index:idx_channel_iq_latest,priority:3,sort:desc"`
	DurationMs   *int64    `json:"duration_ms,omitempty"`
	Status       string    `json:"status" gorm:"not null"`
	ErrorMessage string    `json:"error_message,omitempty" gorm:"type:text"`
	Analysis     JSONValue `json:"analysis,omitempty" gorm:"type:jsonb"`
	HTML         string    `json:"html,omitempty" gorm:"type:text"`
}

// Older versions kept just the last scored fingerprint for the configured model.
// Preserve only facts actually present there. The source and duration of that test were not recorded.
func backfillChannelIQSnapshots() error {
	var channels []Channel
	return DB.Model(&Channel{}).Select("id, other_info").Where("other_info LIKE ?", `%"iq_probe_test_model"%`).
		Where(`NOT EXISTS (SELECT 1 FROM channel_iq_test_runs r WHERE r.channel_id = channels.id AND r.source = 'snapshot')`).
		FindInBatches(&channels, 100, func(batch *gorm.DB, _ int) error {
			type snapshot struct {
				ChannelID   int      `json:"channel_id"`
				Engine      string   `json:"iq_probe_engine"`
				TestModel   string   `json:"iq_probe_test_model"`
				Prediction  string   `json:"iq_probe_prediction"`
				Confidence  *float64 `json:"iq_probe_confidence"`
				CompletedAt int64    `json:"iq_probe_at"`
			}
			var snapshots []snapshot
			for _, channel := range channels {
				var s snapshot
				if err := common.UnmarshalJsonStr(channel.OtherInfo, &s); err != nil {
					common.SysLog(fmt.Sprintf("skipping malformed IQ snapshot for channel %d: %v", channel.Id, err))
					continue
				}
				s.ChannelID = channel.Id
				if s.Engine == DefaultChannelIQProbeEngine && s.CompletedAt > 0 && strings.TrimSpace(s.TestModel) != "" && strings.TrimSpace(s.Prediction) != "" && s.Confidence != nil && *s.Confidence >= 0 && *s.Confidence <= 1 {
					snapshots = append(snapshots, s)
				}
			}
			if len(snapshots) == 0 {
				return nil
			}
			payload, err := common.Marshal(snapshots)
			if err != nil {
				return err
			}
			// Lock the same channels as live writers, then check for existing runs in a fresh statement.
			return batch.Transaction(func(tx *gorm.DB) error {
				ids := make([]int, 0, len(snapshots))
				for _, s := range snapshots {
					ids = append(ids, s.ChannelID)
				}
				var existing []int
				if err := lockForUpdate(tx).Model(&Channel{}).Where("id IN ?", ids).Order("id").Pluck("id", &existing).Error; err != nil {
					return err
				}
				return tx.Exec(`INSERT INTO channel_iq_test_runs (channel_id, test_model, engine, source, completed_at, status, analysis)
					SELECT s.channel_id, s.iq_probe_test_model, s.iq_probe_engine, 'snapshot', s.iq_probe_at, 'success',
					       jsonb_build_object('decision', 'scored', 'prediction', s.iq_probe_prediction, 'probability', s.iq_probe_confidence)
					FROM jsonb_to_recordset(?::jsonb) AS s(channel_id bigint, iq_probe_test_model text, iq_probe_engine text, iq_probe_at bigint, iq_probe_prediction text, iq_probe_confidence double precision)
					JOIN channels c ON c.id = s.channel_id
					WHERE NOT EXISTS (SELECT 1 FROM channel_iq_test_runs r WHERE r.channel_id = s.channel_id AND (r.source = 'snapshot' OR
					    (r.engine = s.iq_probe_engine AND r.test_model = s.iq_probe_test_model AND r.completed_at = s.iq_probe_at AND r.status = 'success')))`, string(payload)).Error
			})
		}).Error
}

func LatestChannelIQTestRuns(ctx context.Context, channelID int, models []string) ([]ChannelIQTestRun, error) {
	var runs []ChannelIQTestRun
	err := DB.WithContext(ctx).Select("DISTINCT ON (test_model) *").
		Where("channel_id = ? AND test_model IN ?", channelID, models).
		Order("test_model, completed_at DESC, id DESC").Find(&runs).Error
	return runs, err
}

func ListChannelIQTestRuns(ctx context.Context, channelID int, testModel string, offset, limit int) ([]ChannelIQTestRun, int64, error) {
	query := DB.WithContext(ctx).Model(&ChannelIQTestRun{}).Where("channel_id = ? AND test_model = ?", channelID, testModel)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var runs []ChannelIQTestRun
	err := query.Order("completed_at DESC, id DESC").Offset(offset).Limit(limit).Find(&runs).Error
	return runs, total, err
}
