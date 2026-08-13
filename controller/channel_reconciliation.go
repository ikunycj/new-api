package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type createChannelCostEntryRequest struct {
	StartAt   int64   `json:"start_at"`
	EndAt     int64   `json:"end_at"`
	AmountUSD float64 `json:"amount_usd"`
	Currency  string  `json:"currency"`
	Source    string  `json:"source"`
	Note      string  `json:"note"`
}

func GetChannelReconciliation(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelId <= 0 {
		common.ApiErrorMsg(c, "invalid channel id")
		return
	}
	startAt, startErr := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endAt, endErr := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if startErr != nil || endErr != nil {
		common.ApiErrorMsg(c, "invalid reconciliation time range")
		return
	}
	result, err := model.GetChannelReconciliation(channelId, startAt, endAt)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func CreateChannelCostEntry(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelId <= 0 {
		common.ApiErrorMsg(c, "invalid channel id")
		return
	}
	var request createChannelCostEntryRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	entry := &model.ChannelCostEntry{
		ChannelId: channelId,
		StartAt:   request.StartAt,
		EndAt:     request.EndAt,
		AmountUSD: request.AmountUSD,
		Currency:  request.Currency,
		Source:    request.Source,
		Note:      request.Note,
		CreatedBy: c.GetInt("id"),
	}
	if err := model.CreateChannelCostEntry(entry); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, entry)
}

func DeleteChannelCostEntry(c *gin.Context) {
	channelId, channelErr := strconv.Atoi(c.Param("id"))
	entryId, entryErr := strconv.Atoi(c.Param("cost_id"))
	if channelErr != nil || entryErr != nil || channelId <= 0 || entryId <= 0 {
		common.ApiErrorMsg(c, "invalid channel cost entry id")
		return
	}
	if err := model.DeleteChannelCostEntry(channelId, entryId); err != nil {
		if errors.Is(err, model.ErrChannelCostEntryNotFound) {
			common.ApiErrorMsg(c, "channel cost entry not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, true)
}
