package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// usageStatsMaxModelQueryLen bounds the model filter, matching the realtime
// endpoint so the two behave the same on the same input.
const usageStatsMaxModelQueryLen = 64

// GetSelfUsageStats returns the calling user's historical usage — requests,
// tokens, cache hit rate and spend — over an arbitrary time range, optionally
// narrowed to one key and/or model.
//
// This is the long-range counterpart to /realtime/self. The realtime endpoint
// reads in-process rings capped at six hours; this one reads the durable hourly
// rollup, so it can answer for a month but cannot resolve finer than an hour.
//
// As with the realtime endpoint, the user id is taken from the session only. A
// token_id naming another account's key narrows within this user's rows and so
// matches nothing, rather than reaching that account's data.
func GetSelfUsageStats(c *gin.Context) {
	query := model.UsageStatsQuery{UserID: c.GetInt("id")}

	if raw := c.Query("start_timestamp"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			common.ApiErrorMsg(c, "invalid start_timestamp")
			return
		}
		query.StartTime = parsed
	}
	if raw := c.Query("end_timestamp"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			common.ApiErrorMsg(c, "invalid end_timestamp")
			return
		}
		query.EndTime = parsed
	}
	if raw := c.Query("bucket_seconds"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			common.ApiErrorMsg(c, "invalid bucket_seconds")
			return
		}
		query.BucketSeconds = parsed
	}
	if raw := c.Query("token_id"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			common.ApiErrorMsg(c, "invalid token_id")
			return
		}
		query.TokenID = parsed
	}
	if raw := c.Query("model"); raw != "" {
		if len(raw) > usageStatsMaxModelQueryLen {
			common.ApiErrorMsg(c, "invalid model")
			return
		}
		query.Model = raw
	}

	result, err := model.GetUsageStats(query)
	if err != nil {
		// The model validates the range and granularity, so its error is the
		// caller's to fix and is surfaced rather than replaced.
		common.ApiErrorMsg(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}
