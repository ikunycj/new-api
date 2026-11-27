package controller

import (
	"context"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetPublicModelLeaderboards(c *gin.Context) {
	modelLeaderboardResponse(c, false)
}

func GetAdminModelLeaderboards(c *gin.Context) {
	modelLeaderboardResponse(c, true)
}

func modelLeaderboardResponse(c *gin.Context, admin bool) {
	settings, snapshot, err := model.GetModelLeaderboards(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data := gin.H{
		"boards":             snapshot.Boards,
		"recommended_models": settings.RecommendedModels,
	}
	if snapshot.UpdatedAt != nil {
		data["updated_at"] = snapshot.UpdatedAt
	}
	if admin {
		data["enabled"] = settings.Enabled
	} else if snapshot.UpdatedAt != nil || len(settings.RecommendedModels) > 0 {
		pricing := model.GetPricing()
		data["recommended_models"] = service.MatchRecommendedModels(settings.RecommendedModels, pricing)
		if snapshot.UpdatedAt != nil {
			for name, entries := range snapshot.Boards {
				snapshot.Boards[name] = service.MatchModelLeaderboardEntries(entries, pricing)
			}
		}
	}
	common.ApiSuccess(c, data)
}

func UpdateModelLeaderboardSettings(c *gin.Context) {
	var settings *model.ModelLeaderboardSettings
	if err := common.DecodeJson(c.Request.Body, &settings); err != nil || settings == nil {
		common.ApiErrorMsg(c, "榜单设置格式不正确")
		return
	}
	if err := model.SaveModelLeaderboardSettings(c.Request.Context(), *settings); err != nil {
		common.ApiError(c, err)
		return
	}
	GetAdminModelLeaderboards(c)
}

func SyncModelLeaderboards(c *gin.Context) {
	task, _, err := service.EnqueueSystemTask(model.SystemTaskTypeModelLeaderboardSync, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 42*time.Second)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			common.ApiErrorMsg(c, "同步仍在进行，请稍后刷新榜单")
			return
		case <-ticker.C:
		}
		current, err := model.GetSystemTaskByTaskID(task.TaskID)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		switch current.Status {
		case model.SystemTaskStatusSucceeded:
			GetAdminModelLeaderboards(c)
			return
		case model.SystemTaskStatusFailed:
			common.ApiErrorMsg(c, "同步榜单失败："+current.Error)
			return
		}
	}
}

type modelLeaderboardSyncHandler struct{}

func (modelLeaderboardSyncHandler) Type() string { return model.SystemTaskTypeModelLeaderboardSync }

func (modelLeaderboardSyncHandler) Enabled() bool {
	settings, err := model.GetModelLeaderboardSettings(context.Background())
	if err != nil {
		common.SysError("read model leaderboard schedule: " + err.Error())
		return false
	}
	return settings.Enabled
}

func (modelLeaderboardSyncHandler) Interval() time.Duration { return 24 * time.Hour }
func (modelLeaderboardSyncHandler) NewPayload() any         { return false }

var modelLeaderboardTimeZone = time.FixedZone("Asia/Shanghai", 8*60*60)

func (modelLeaderboardSyncHandler) Due(now time.Time, latest *model.SystemTask) bool {
	local := now.In(modelLeaderboardTimeZone)
	scheduled := time.Date(local.Year(), local.Month(), local.Day(), 4, 0, 0, 0, local.Location())
	return !now.Before(scheduled) && (latest == nil || latest.CreatedAt < scheduled.Unix())
}

func (handler modelLeaderboardSyncHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	var manual bool
	err := task.DecodePayload(&manual)
	if err == nil && !manual && !handler.Enabled() {
		err = errors.New("榜单自动同步已关闭")
	}
	if err == nil {
		err = service.SyncModelLeaderboards(ctx)
	}
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, nil, nil)
}
