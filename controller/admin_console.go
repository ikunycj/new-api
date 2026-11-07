package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

func parseAdminConsoleTimeRange(c *gin.Context) (startTimestamp, endTimestamp int64, provided, ok bool) {
	startValue, endValue := c.Query("start_timestamp"), c.Query("end_timestamp")
	if startValue == "" && endValue == "" {
		return 0, 0, false, true
	}
	if startValue == "" || endValue == "" {
		common.ApiErrorMsg(c, "start_timestamp and end_timestamp are required together")
		return 0, 0, true, false
	}
	startTimestamp, startErr := strconv.ParseInt(startValue, 10, 64)
	endTimestamp, endErr := strconv.ParseInt(endValue, 10, 64)
	if startErr != nil || startTimestamp <= 0 {
		common.ApiErrorMsg(c, "invalid start_timestamp")
		return 0, 0, true, false
	}
	if endErr != nil || endTimestamp <= 0 {
		common.ApiErrorMsg(c, "invalid end_timestamp")
		return 0, 0, true, false
	}
	if endTimestamp < startTimestamp {
		common.ApiErrorMsg(c, "invalid time range")
		return 0, 0, true, false
	}
	return startTimestamp, endTimestamp, true, true
}

func GetAdminConsole(c *gin.Context) {
	startTimestamp, endTimestamp, provided, ok := parseAdminConsoleTimeRange(c)
	if !ok {
		return
	}
	var (
		stats model.AdminConsoleStats
		err   error
	)
	if provided {
		stats, err = model.GetAdminConsoleStatsForRange(startTimestamp, endTimestamp)
	} else {
		stats, err = model.GetAdminConsoleStats()
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}

func GetAdminConsoleSystemLoad(c *gin.Context) {
	common.ApiSuccess(c, model.GetAdminConsoleSystemLoad())
}

func GetAdminConsoleRealtime(c *gin.Context) {
	startTimestamp, endTimestamp, provided, ok := parseAdminConsoleTimeRange(c)
	if !ok {
		return
	}
	var (
		stats model.AdminConsoleRealtimeStats
		err   error
	)
	if provided {
		stats, err = model.GetAdminConsoleRealtimeStatsForRange(startTimestamp, endTimestamp)
	} else {
		stats, err = model.GetAdminConsoleRealtimeStats()
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	stats.CurrentConcurrency = int64(service.GetTotalPricingGroupConnections(ratio_setting.GetPricingGroupOrder()))
	common.ApiSuccess(c, stats)
}
