package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const GroupBenchRunRetention = 30 * 24 * time.Hour

// GroupBenchModel is one model a group bench round tests on every eligible
// channel. EndpointType follows constant.EndpointType ("openai", "anthropic",
// "openai-response", ...).
type GroupBenchModel struct {
	Model        string `json:"model"`
	EndpointType string `json:"endpoint_type"`
}

// GroupBenchConfig holds the periodic bench settings of one pricing group.
// Pricing groups have no ID table, so the group name is the key.
type GroupBenchConfig struct {
	Id                int    `json:"id"`
	GroupName         string `json:"group" gorm:"type:varchar(64);uniqueIndex;not null"`
	Enabled           bool   `json:"enabled" gorm:"index;not null"`
	Preset            string `json:"preset" gorm:"type:varchar(64);not null"`
	Models            string `json:"-" gorm:"type:text"`
	IntervalMinutes   int    `json:"interval_minutes" gorm:"not null"`
	LastRunAt         *int64 `json:"last_run_at" gorm:"bigint"`
	NextRunAt         int64  `json:"next_run_at" gorm:"bigint;index"`
	ManualRequestedAt *int64 `json:"manual_requested_at" gorm:"bigint;index"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt         int64  `json:"updated_at" gorm:"bigint"`
}

// GroupBenchRun is the outcome of one (channel, model) request in a round. The
// produced artifact lives in GroupBenchArtifact so list queries stay narrow.
type GroupBenchRun struct {
	Id            int64  `json:"id"`
	GroupName     string `json:"group" gorm:"type:varchar(64);not null;index:idx_group_bench_run_round,priority:1;index:idx_group_bench_run_series,priority:1"`
	RoundAt       int64  `json:"round_at" gorm:"bigint;not null;index:idx_group_bench_run_round,priority:2;index:idx_group_bench_run_series,priority:4"`
	ChannelId     int    `json:"channel_id" gorm:"not null;index:idx_group_bench_run_series,priority:2"`
	ChannelName   string `json:"channel_name" gorm:"type:varchar(128)"`
	Model         string `json:"model" gorm:"type:varchar(128);not null;index:idx_group_bench_run_series,priority:3"`
	EndpointType  string `json:"endpoint_type" gorm:"type:varchar(32)"`
	Preset        string `json:"preset" gorm:"type:varchar(64)"`
	Success       bool   `json:"success" gorm:"not null"`
	ErrorMessage  string `json:"error_message,omitempty" gorm:"type:text"`
	LatencyMs     int64  `json:"latency_ms" gorm:"bigint"`
	InputTokens   int    `json:"input_tokens"`
	OutputTokens  int    `json:"output_tokens"`
	StopReason    string `json:"stop_reason" gorm:"type:varchar(64)"`
	ResponseModel string `json:"response_model" gorm:"type:varchar(128)"`
	Truncated     bool   `json:"truncated" gorm:"not null"`
	HasArtifact   bool   `json:"has_artifact" gorm:"not null"`
	Metrics       string `json:"metrics" gorm:"type:text"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint"`
}

// GroupBenchArtifact stores the extracted output (e.g. SVG source) of a run.
// Content is TEXT everywhere, so producers must keep it under MySQL's 64KB
// limit and flag Clipped when they cut it.
type GroupBenchArtifact struct {
	RunId       int64  `json:"run_id" gorm:"primaryKey;autoIncrement:false"`
	ContentType string `json:"content_type" gorm:"type:varchar(64);not null"`
	Content     string `json:"content" gorm:"type:text"`
	Bytes       int    `json:"bytes"`
	Clipped     bool   `json:"clipped" gorm:"not null"`
}

func (config *GroupBenchConfig) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if config.CreatedAt == 0 {
		config.CreatedAt = now
	}
	config.UpdatedAt = now
	return nil
}

func (config *GroupBenchConfig) BeforeUpdate(_ *gorm.DB) error {
	config.UpdatedAt = common.GetTimestamp()
	return nil
}

func (run *GroupBenchRun) BeforeCreate(_ *gorm.DB) error {
	if run.CreatedAt == 0 {
		run.CreatedAt = common.GetTimestamp()
	}
	return nil
}

func (config *GroupBenchConfig) GetModels() ([]GroupBenchModel, error) {
	models := []GroupBenchModel{}
	if strings.TrimSpace(config.Models) == "" {
		return models, nil
	}
	if err := common.UnmarshalJsonStr(config.Models, &models); err != nil {
		return nil, err
	}
	return models, nil
}

func (config *GroupBenchConfig) SetModels(models []GroupBenchModel) error {
	data, err := common.Marshal(models)
	if err != nil {
		return err
	}
	config.Models = string(data)
	return nil
}

// GetGroupBenchConfig returns (nil, nil) when the group has no config yet.
func GetGroupBenchConfig(group string) (*GroupBenchConfig, error) {
	var config GroupBenchConfig
	err := DB.Where("group_name = ?", group).First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func SaveGroupBenchConfig(config *GroupBenchConfig) error {
	if config.Id == 0 {
		return DB.Create(config).Error
	}
	return DB.Save(config).Error
}

func ListGroupBenchConfigs() ([]*GroupBenchConfig, error) {
	var configs []*GroupBenchConfig
	err := DB.Order("group_name asc").Find(&configs).Error
	return configs, err
}

// ListDueGroupBenchConfigs returns enabled configs whose next run has come, plus
// any config with a pending manual request regardless of enablement.
func ListDueGroupBenchConfigs(now int64) ([]*GroupBenchConfig, error) {
	var configs []*GroupBenchConfig
	err := DB.Where("(enabled = ? AND next_run_at <= ?) OR manual_requested_at IS NOT NULL", true, now).
		Order("group_name asc").
		Find(&configs).Error
	return configs, err
}

// HasActiveGroupBench reports whether any group is enabled or has a pending
// manual request, so the scheduler creates no task rows while the feature is idle.
func HasActiveGroupBench() bool {
	var count int64
	err := DB.Model(&GroupBenchConfig{}).
		Where("enabled = ? OR manual_requested_at IS NOT NULL", true).
		Limit(1).
		Count(&count).Error
	return err == nil && count > 0
}

// RequestGroupBenchRun marks the group for a run on the next task pass.
func RequestGroupBenchRun(configId int, now int64) error {
	return DB.Model(&GroupBenchConfig{}).
		Where("id = ?", configId).
		Updates(map[string]any{"manual_requested_at": now, "updated_at": now}).Error
}

// GroupBenchTarget is one (channel, model) pair a round tests.
type GroupBenchTarget struct {
	ChannelId int    `json:"channel_id"`
	Model     string `json:"model"`
}

// ListGroupBenchTargets returns the enabled abilities of the group for the
// given models, ordered for stable round output.
func ListGroupBenchTargets(group string, models []string) ([]GroupBenchTarget, error) {
	var targets []GroupBenchTarget
	if len(models) == 0 {
		return targets, nil
	}
	err := DB.Model(&Ability{}).
		Select("channel_id, model").
		Where(commonGroupCol+" = ? AND enabled = ? AND model IN ?", group, true, models).
		Order("channel_id asc, model asc").
		Scan(&targets).Error
	return targets, err
}

// FinishGroupBenchRound reschedules the group and prunes runs past retention.
// A manual request made after the round started is kept for the next pass.
func FinishGroupBenchRound(config *GroupBenchConfig, roundAt int64, nextRunAt int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		if err := tx.Model(&GroupBenchConfig{}).
			Where("id = ?", config.Id).
			Updates(map[string]any{
				"last_run_at": roundAt,
				"next_run_at": nextRunAt,
				"updated_at":  now,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&GroupBenchConfig{}).
			Where("id = ? AND manual_requested_at <= ?", config.Id, roundAt).
			Update("manual_requested_at", nil).Error; err != nil {
			return err
		}
		cutoff := roundAt - int64(GroupBenchRunRetention/time.Second)
		expired := tx.Model(&GroupBenchRun{}).Select("id").Where("group_name = ? AND round_at < ?", config.GroupName, cutoff)
		if err := tx.Where("run_id IN (?)", expired).Delete(&GroupBenchArtifact{}).Error; err != nil {
			return err
		}
		return tx.Where("group_name = ? AND round_at < ?", config.GroupName, cutoff).Delete(&GroupBenchRun{}).Error
	})
}

func SaveGroupBenchRun(run *GroupBenchRun, artifact *GroupBenchArtifact) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		run.HasArtifact = artifact != nil
		if err := tx.Create(run).Error; err != nil {
			return err
		}
		if artifact == nil {
			return nil
		}
		artifact.RunId = run.Id
		return tx.Create(artifact).Error
	})
}

type GroupBenchRunFilter struct {
	Group     string
	ChannelId int
	Model     string
	Since     int64
	Limit     int
}

func ListGroupBenchRuns(filter GroupBenchRunFilter) ([]*GroupBenchRun, error) {
	query := DB.Where("group_name = ?", filter.Group)
	if filter.ChannelId > 0 {
		query = query.Where("channel_id = ?", filter.ChannelId)
	}
	if filter.Model != "" {
		query = query.Where("model = ?", filter.Model)
	}
	if filter.Since > 0 {
		query = query.Where("round_at >= ?", filter.Since)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	var runs []*GroupBenchRun
	err := query.Order("round_at desc, channel_id asc, model asc").Limit(limit).Find(&runs).Error
	return runs, err
}

func GetGroupBenchArtifact(runId int64) (*GroupBenchArtifact, error) {
	var artifact GroupBenchArtifact
	if err := DB.Where("run_id = ?", runId).First(&artifact).Error; err != nil {
		return nil, err
	}
	return &artifact, nil
}
