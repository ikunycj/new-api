package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func (result testResult) iqTestRun(channelID int, testModel, engine string, elapsed time.Duration) (*model.ChannelIQTestRun, error) {
	run := &model.ChannelIQTestRun{
		ChannelID: channelID, TestModel: testModel, Engine: engine,
		CompletedAt: common.GetTimestamp(), DurationMs: common.GetPointer(elapsed.Milliseconds()),
		Status: model.IQTestSucceeded,
	}
	analysis := result.fingerprintAnalysis
	if analysis != nil {
		data, err := common.Marshal(analysis)
		if err != nil {
			return nil, err
		}
		run.Analysis = data
	}
	switch {
	case result.localErr != nil:
		run.Status, run.ErrorMessage = model.IQTestFailed, result.localErr.Error()
	case result.newAPIError != nil:
		run.Status, run.ErrorMessage = model.IQTestFailed, result.newAPIError.Error()
	case engine == channelIQTestEnginePelican:
		run.HTML = result.response
	case analysis.Decision != "scored" || analysis.Probability == nil:
		run.Status, run.ErrorMessage = model.IQTestInconclusive, analysis.Evidence.Reason
	}
	return run, nil
}

func GetLatestChannelIQTestRuns(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorMsg(c, "无效的渠道 ID")
		return
	}
	var request struct {
		Models []string `json:"models"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || len(request.Models) == 0 || len(request.Models) > 100 {
		common.ApiErrorMsg(c, "请选择 1 至 100 个模型")
		return
	}
	runs, err := model.LatestChannelIQTestRuns(c.Request.Context(), channelID, request.Models)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, runs)
}

func GetChannelIQTestHistory(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	testModel := strings.TrimSpace(c.Query("model"))
	page := common.GetPageQuery(c)
	if err != nil || channelID <= 0 || testModel == "" || page.Page < 1 || page.PageSize < 1 {
		common.ApiErrorMsg(c, "无效的渠道、模型或分页参数")
		return
	}
	runs, total, err := model.ListChannelIQTestRuns(c.Request.Context(), channelID, testModel, page.GetStartIdx(), page.PageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	page.SetItems(runs)
	page.SetTotal(int(total))
	common.ApiSuccess(c, page)
}
