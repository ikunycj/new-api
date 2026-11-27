package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	modelLeaderboardSettingsKey = "ModelLeaderboardSettings"
	modelLeaderboardSnapshotKey = "ModelLeaderboardSnapshot"
	ModelLeaderboardAgent       = "agent"
	ModelLeaderboardCoding      = "coding"
	ModelLeaderboardChat        = "chat"
)

type ModelLeaderboardSettings struct {
	Enabled           bool     `json:"enabled"`
	RecommendedModels []string `json:"recommended_models"`
}

type ModelLeaderboardEntry struct {
	ModelName string   `json:"model_name"`
	Rank      int      `json:"rank"`
	Score     float64  `json:"score"`
	ModelIDs  []string `json:"model_ids,omitempty"`
}

type ModelLeaderboardSnapshot struct {
	UpdatedAt *time.Time                         `json:"updated_at,omitempty"`
	Boards    map[string][]ModelLeaderboardEntry `json:"boards"`
}

func GetModelLeaderboardSettings(ctx context.Context) (ModelLeaderboardSettings, error) {
	settings := ModelLeaderboardSettings{RecommendedModels: []string{}}
	err := readModelLeaderboardOption(ctx, modelLeaderboardSettingsKey, &settings)
	return settings, err
}

func GetModelLeaderboards(ctx context.Context) (ModelLeaderboardSettings, ModelLeaderboardSnapshot, error) {
	settings := ModelLeaderboardSettings{RecommendedModels: []string{}}
	snapshot := ModelLeaderboardSnapshot{Boards: map[string][]ModelLeaderboardEntry{
		ModelLeaderboardAgent: {}, ModelLeaderboardCoding: {}, ModelLeaderboardChat: {},
	}}
	var options []Option
	if err := DB.WithContext(ctx).Where(map[string]any{
		"key": []string{modelLeaderboardSettingsKey, modelLeaderboardSnapshotKey},
	}).Find(&options).Error; err != nil {
		return settings, snapshot, err
	}
	for _, option := range options {
		var target any = &snapshot
		if option.Key == modelLeaderboardSettingsKey {
			target = &settings
		}
		if err := common.UnmarshalJsonStr(option.Value, target); err != nil {
			return settings, snapshot, fmt.Errorf("decode %s: %w", option.Key, err)
		}
	}
	return settings, snapshot, nil
}

func SaveModelLeaderboardSettings(ctx context.Context, settings ModelLeaderboardSettings) error {
	settings.RecommendedModels = normalizeLookupValues(settings.RecommendedModels)
	return saveModelLeaderboardOption(ctx, modelLeaderboardSettingsKey, settings)
}

func SaveModelLeaderboardSnapshot(ctx context.Context, snapshot ModelLeaderboardSnapshot) error {
	return saveModelLeaderboardOption(ctx, modelLeaderboardSnapshotKey, snapshot)
}

func IsModelLeaderboardOption(key string) bool {
	return key == modelLeaderboardSettingsKey || key == modelLeaderboardSnapshotKey
}

func CachedRecommendedModels() ([]string, error) {
	common.OptionMapRWMutex.RLock()
	settingsJSON := common.OptionMap[modelLeaderboardSettingsKey]
	common.OptionMapRWMutex.RUnlock()
	if settingsJSON == "" {
		return nil, nil
	}
	var settings ModelLeaderboardSettings
	if err := common.UnmarshalJsonStr(settingsJSON, &settings); err != nil {
		return nil, err
	}
	return settings.RecommendedModels, nil
}

func readModelLeaderboardOption(ctx context.Context, key string, target any) error {
	var option Option
	err := DB.WithContext(ctx).Take(&option, &Option{Key: key}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := common.UnmarshalJsonStr(option.Value, target); err != nil {
		return fmt.Errorf("decode %s: %w", key, err)
	}
	return nil
}

func saveModelLeaderboardOption(ctx context.Context, key string, value any) error {
	data, err := common.Marshal(value)
	if err != nil {
		return err
	}
	if err := DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&Option{Key: key, Value: string(data)}).Error; err != nil {
		return err
	}
	return applyOptionMap(key, string(data), preparedOptionUpdate{})
}
