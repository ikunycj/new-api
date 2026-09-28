package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/groupbench"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

const (
	groupBenchDefaultIntervalMinutes = 60
	groupBenchMinIntervalMinutes     = 15
	groupBenchMaxIntervalMinutes     = 24 * 60
	// groupBenchWorkers bounds in-flight bench requests per round; upstreams
	// allow far more, this only keeps one round from flooding the gateway.
	groupBenchWorkers        = 16
	groupBenchRequestTimeout = 10 * time.Minute
	groupBenchMaxBodyBytes   = 4 << 20
	groupBenchLogLabel       = "分组巡检"
	groupBenchMaxErrorLength = 1000
)

var groupBenchDefaultModels = []model.GroupBenchModel{
	{Model: "gpt-6-sol", EndpointType: string(constant.EndpointTypeOpenAI)},
	{Model: "claude-opus-5", EndpointType: string(constant.EndpointTypeAnthropic)},
}

// groupBenchHandler checks every 5 minutes for groups whose next round is due.
// Rounds align to the interval boundary (on the hour for 60 minutes), so a run
// starts at most one check period after the boundary.
type groupBenchHandler struct{}

func (groupBenchHandler) Type() string            { return model.SystemTaskTypeGroupBench }
func (groupBenchHandler) Enabled() bool           { return model.HasActiveGroupBench() }
func (groupBenchHandler) Interval() time.Duration { return 5 * time.Minute }
func (groupBenchHandler) NewPayload() any         { return nil }

func (groupBenchHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	testUserID, err := resolveChannelTestUserID(nil)
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	summary := map[string]int{}
	// Loop so a manual request that arrives mid-round is served by this task
	// instead of waiting for the next scheduled one.
	for ctx.Err() == nil {
		now := common.GetTimestamp()
		configs, err := model.ListDueGroupBenchConfigs(now)
		if err != nil {
			finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, summary, err)
			return
		}
		if len(configs) == 0 {
			break
		}
		for _, config := range configs {
			if ctx.Err() != nil {
				break
			}
			runs, err := runGroupBenchRound(ctx, config, testUserID, now)
			if err != nil {
				common.SysError(fmt.Sprintf("group bench round failed: group=%s err=%v", config.GroupName, err))
				summary["failed_rounds"]++
				continue
			}
			summary["rounds"]++
			summary["runs"] += runs
		}
	}
	if ctx.Err() != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, summary, ctx.Err())
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, summary, nil)
}

// groupBenchNextRunAt returns the first interval boundary after now.
func groupBenchNextRunAt(now int64, intervalMinutes int) int64 {
	if intervalMinutes <= 0 {
		intervalMinutes = groupBenchDefaultIntervalMinutes
	}
	step := int64(intervalMinutes) * 60
	return (now/step + 1) * step
}

// runGroupBenchRound tests every (channel, model) target of the group once and
// always reschedules it, so a broken config does not retry every pass.
func runGroupBenchRound(ctx context.Context, config *model.GroupBenchConfig, testUserID int, roundAt int64) (int, error) {
	nextRunAt := groupBenchNextRunAt(roundAt, config.IntervalMinutes)
	runs, roundErr := benchGroupTargets(ctx, config, testUserID, roundAt)
	if err := model.FinishGroupBenchRound(config, roundAt, nextRunAt); err != nil {
		return runs, errors.Join(roundErr, err)
	}
	return runs, roundErr
}

func benchGroupTargets(ctx context.Context, config *model.GroupBenchConfig, testUserID int, roundAt int64) (int, error) {
	preset, ok := groupbench.GetPreset(config.Preset)
	if !ok {
		return 0, fmt.Errorf("unknown preset %q", config.Preset)
	}
	benchModels, err := config.GetModels()
	if err != nil {
		return 0, err
	}
	endpointByModel := make(map[string]string, len(benchModels))
	for _, benchModel := range benchModels {
		endpointByModel[benchModel.Model] = benchModel.EndpointType
	}
	targets, err := model.ListGroupBenchTargets(config.GroupName, lo.Keys(endpointByModel))
	if err != nil {
		return 0, err
	}

	channels := map[int]*model.Channel{}
	for _, target := range targets {
		if _, seen := channels[target.ChannelId]; seen {
			continue
		}
		channel, err := model.GetChannelById(target.ChannelId, true)
		if err != nil {
			common.SysError(fmt.Sprintf("group bench load channel #%d failed: %v", target.ChannelId, err))
			channels[target.ChannelId] = nil
			continue
		}
		channels[target.ChannelId] = channel
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, groupBenchWorkers)
	var mu sync.Mutex
	saved := 0
	for _, target := range targets {
		channel := channels[target.ChannelId]
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		endpointType := endpointByModel[target.Model]
		benchModel := target.Model
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			run, artifact := benchChannelModel(ctx, config.GroupName, preset, channel, benchModel, endpointType, testUserID, roundAt)
			if err := model.SaveGroupBenchRun(run, artifact); err != nil {
				common.SysError(fmt.Sprintf("group bench save run failed: group=%s channel=#%d model=%s err=%v", config.GroupName, channel.Id, benchModel, err))
				return
			}
			mu.Lock()
			saved++
			mu.Unlock()
		}()
	}
	wg.Wait()
	return saved, nil
}

// benchChannelModel sends the preset to one channel. Failures are only
// recorded: bench results never disable channels or feed channel health.
func benchChannelModel(ctx context.Context, group string, preset *groupbench.Preset, channel *model.Channel, benchModel string, endpointType string, testUserID int, roundAt int64) (*model.GroupBenchRun, *model.GroupBenchArtifact) {
	// Codex OAuth channels only speak the Responses protocol.
	switch {
	case channel.Type == constant.ChannelTypeCodex:
		endpointType = string(constant.EndpointTypeOpenAIResponse)
	case endpointType == "" && strings.Contains(strings.ToLower(benchModel), "claude"):
		endpointType = string(constant.EndpointTypeAnthropic)
	case endpointType == "":
		endpointType = string(constant.EndpointTypeOpenAI)
	}

	run := &model.GroupBenchRun{
		GroupName:    group,
		RoundAt:      roundAt,
		ChannelId:    channel.Id,
		ChannelName:  channel.Name,
		Model:        benchModel,
		EndpointType: endpointType,
		Preset:       preset.Key,
	}

	requestCtx, cancel := context.WithTimeout(ctx, groupBenchRequestTimeout)
	defer cancel()
	started := time.Now()
	result := testChannelWithOverride(requestCtx, channel, testUserID, benchModel, endpointType, true, channelTestOverride{
		Request:      buildGroupBenchRequest(preset, benchModel, endpointType),
		Group:        group,
		LogLabel:     groupBenchLogLabel,
		QuietBody:    true,
		MaxBodyBytes: groupBenchMaxBodyBytes,
	})
	run.LatencyMs = time.Since(started).Milliseconds()
	if result.localErr != nil {
		run.ErrorMessage, _ = groupbench.ClipUTF8(result.localErr.Error(), groupBenchMaxErrorLength)
		return run, nil
	}
	if result.usage != nil {
		run.InputTokens = result.usage.PromptTokens
		run.OutputTokens = result.usage.CompletionTokens
	}

	answer := groupbench.ExtractResponseText(result.respBody)
	run.StopReason = answer.StopReason
	run.ResponseModel = answer.Model
	run.Truncated = lo.Contains([]string{"length", "max_tokens", "max_output_tokens"}, answer.StopReason)

	extracted, ok := preset.Extract(answer.Text)
	if !ok {
		run.ErrorMessage = "no artifact found in answer"
		fallback := groupbench.FallbackArtifact(answer.Text)
		return run, &model.GroupBenchArtifact{
			ContentType: fallback.ContentType,
			Content:     fallback.Content,
			Bytes:       len(fallback.Content),
			Clipped:     fallback.Clipped,
		}
	}
	run.Success = true
	run.Truncated = run.Truncated || extracted.Clipped
	if metrics, err := common.Marshal(preset.Analyze(extracted.Content)); err == nil {
		run.Metrics = string(metrics)
	}
	return run, &model.GroupBenchArtifact{
		ContentType: extracted.ContentType,
		Content:     extracted.Content,
		Bytes:       len(extracted.Content),
		Clipped:     extracted.Clipped,
	}
}

// buildGroupBenchRequest streams every request: long generations otherwise
// sit silent for minutes and trip idle timeouts along the path. Temperature is
// left unset because reasoning models reject it.
func buildGroupBenchRequest(preset *groupbench.Preset, benchModel string, endpointType string) dto.Request {
	if constant.EndpointType(endpointType) == constant.EndpointTypeOpenAIResponse {
		input, _ := common.Marshal([]map[string]string{{"role": "user", "content": preset.Prompt}})
		return &dto.OpenAIResponsesRequest{
			Model:           benchModel,
			Input:           input,
			MaxOutputTokens: lo.ToPtr(preset.MaxTokens),
			Stream:          lo.ToPtr(true),
		}
	}
	return &dto.GeneralOpenAIRequest{
		Model:         benchModel,
		Stream:        lo.ToPtr(true),
		StreamOptions: &dto.StreamOptions{IncludeUsage: true},
		Messages:      []dto.Message{{Role: "user", Content: preset.Prompt}},
		MaxTokens:     lo.ToPtr(preset.MaxTokens),
	}
}

type groupBenchConfigResponse struct {
	Group           string                  `json:"group"`
	Enabled         bool                    `json:"enabled"`
	Preset          string                  `json:"preset"`
	Models          []model.GroupBenchModel `json:"models"`
	IntervalMinutes int                     `json:"interval_minutes"`
	LastRunAt       *int64                  `json:"last_run_at"`
	NextRunAt       int64                   `json:"next_run_at"`
	ManualPending   bool                    `json:"manual_pending"`
	Presets         []*groupbench.Preset    `json:"presets"`
	GroupModels     []string                `json:"group_models"`
}

type groupBenchConfigRequest struct {
	Group           string                  `json:"group"`
	Enabled         bool                    `json:"enabled"`
	Preset          string                  `json:"preset"`
	Models          []model.GroupBenchModel `json:"models"`
	IntervalMinutes int                     `json:"interval_minutes"`
}

// GetGroupBenchConfig returns the group's bench config, or the defaults when
// the group has never been configured.
func GetGroupBenchConfig(c *gin.Context) {
	group := strings.TrimSpace(c.Query("group"))
	if group == "" {
		common.ApiErrorMsg(c, "group is required")
		return
	}
	config, err := model.GetGroupBenchConfig(group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	response := groupBenchConfigResponse{
		Group:           group,
		Preset:          groupbench.DefaultPresetKey,
		Models:          groupBenchDefaultModels,
		IntervalMinutes: groupBenchDefaultIntervalMinutes,
		Presets:         groupbench.ListPresets(),
		GroupModels:     model.GetGroupEnabledModels(group),
	}
	if config != nil {
		models, err := config.GetModels()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		response.Enabled = config.Enabled
		response.Preset = config.Preset
		response.Models = models
		response.IntervalMinutes = config.IntervalMinutes
		response.LastRunAt = config.LastRunAt
		response.NextRunAt = config.NextRunAt
		response.ManualPending = config.ManualRequestedAt != nil
	}
	common.ApiSuccess(c, response)
}

func UpdateGroupBenchConfig(c *gin.Context) {
	var req groupBenchConfigRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	req.Group = strings.TrimSpace(req.Group)
	if req.Group == "" {
		common.ApiErrorMsg(c, "group is required")
		return
	}
	if _, ok := groupbench.GetPreset(req.Preset); !ok {
		common.ApiErrorMsg(c, fmt.Sprintf("unknown preset %q", req.Preset))
		return
	}
	if req.IntervalMinutes < groupBenchMinIntervalMinutes || req.IntervalMinutes > groupBenchMaxIntervalMinutes {
		common.ApiErrorMsg(c, fmt.Sprintf("interval_minutes must be between %d and %d", groupBenchMinIntervalMinutes, groupBenchMaxIntervalMinutes))
		return
	}
	models := make([]model.GroupBenchModel, 0, len(req.Models))
	seen := map[string]bool{}
	for _, benchModel := range req.Models {
		benchModel.Model = strings.TrimSpace(benchModel.Model)
		benchModel.EndpointType = strings.TrimSpace(benchModel.EndpointType)
		if benchModel.Model == "" || seen[benchModel.Model] {
			continue
		}
		seen[benchModel.Model] = true
		models = append(models, benchModel)
	}
	if len(models) == 0 {
		common.ApiErrorMsg(c, "at least one model is required")
		return
	}

	config, err := model.GetGroupBenchConfig(req.Group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if config == nil {
		config = &model.GroupBenchConfig{GroupName: req.Group}
	}
	rescheduled := !config.Enabled || config.IntervalMinutes != req.IntervalMinutes
	config.Enabled = req.Enabled
	config.Preset = req.Preset
	config.IntervalMinutes = req.IntervalMinutes
	if rescheduled {
		config.NextRunAt = groupBenchNextRunAt(common.GetTimestamp(), req.IntervalMinutes)
	}
	if err := config.SetModels(models); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SaveGroupBenchConfig(config); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"next_run_at": config.NextRunAt})
}

// RunGroupBenchNow queues a round for the group even when the schedule is off.
func RunGroupBenchNow(c *gin.Context) {
	group := strings.TrimSpace(c.Query("group"))
	config, err := model.GetGroupBenchConfig(group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if config == nil {
		common.ApiErrorMsg(c, "save the group bench config before running it")
		return
	}
	if err := model.RequestGroupBenchRun(config.Id, common.GetTimestamp()); err != nil {
		common.ApiError(c, err)
		return
	}
	task, created, err := service.EnqueueSystemTask(model.SystemTaskTypeGroupBench, nil)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"task_id": task.TaskID, "created": created})
}

func ListGroupBenchRuns(c *gin.Context) {
	group := strings.TrimSpace(c.Query("group"))
	if group == "" {
		common.ApiErrorMsg(c, "group is required")
		return
	}
	channelId, _ := strconv.Atoi(c.Query("channel_id"))
	since, _ := strconv.ParseInt(c.Query("since"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	runs, err := model.ListGroupBenchRuns(model.GroupBenchRunFilter{
		Group:     group,
		ChannelId: channelId,
		Model:     strings.TrimSpace(c.Query("model")),
		Since:     since,
		Limit:     limit,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": runs})
}

// GetGroupBenchArtifact returns the artifact as JSON rather than serving the
// SVG directly, so model-written markup is never rendered on the site origin.
func GetGroupBenchArtifact(c *gin.Context) {
	runId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || runId <= 0 {
		common.ApiErrorMsg(c, "invalid run id")
		return
	}
	artifact, err := model.GetGroupBenchArtifact(runId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		common.ApiErrorMsg(c, "artifact not found")
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, artifact)
}
