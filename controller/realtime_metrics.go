package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// realtimeDefaultWindowSeconds is the window used by the admin user list when
// the caller does not ask for a specific one.
const realtimeDefaultWindowSeconds = 60

// realtimeAllowedWindows bounds the admin window parameter. The model only
// keeps six hours, so anything beyond that would silently report a shorter span
// under a longer label.
var realtimeAllowedWindows = map[int64]bool{60: true, 300: true, 3600: true}

// GetSelfRealtimeMetrics returns the calling user's own realtime throughput.
//
// The user id comes from the authenticated session, never from a parameter, so
// a user cannot read anyone else's numbers by editing the request. Admin access
// to other users goes through the AdminAuth-gated route below.
func GetSelfRealtimeMetrics(c *gin.Context) {
	userId := c.GetInt("id")
	snapshot := model.GetRealtimeSnapshot(userId)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    snapshot,
	})
}

// GetRealtimeMetricsUsers returns per-user realtime throughput for every user
// this node has served recently. Admin only.
func GetRealtimeMetricsUsers(c *gin.Context) {
	window := int64(realtimeDefaultWindowSeconds)
	if raw := c.Query("window_seconds"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || !realtimeAllowedWindows[parsed] {
			common.ApiErrorMsg(c, "invalid window_seconds")
			return
		}
		window = parsed
	}

	summaries := model.GetRealtimeUserSummaries(window)
	model.FillRealtimeUsernames(summaries)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"window_seconds": window,
			"node_name":      common.NodeName,
			"now":            common.GetTimestamp(),
			"users":          summaries,
		},
	})
}

// GetRealtimeMetricsByUser returns one specific user's snapshot for an admin
// inspecting a support case.
func GetRealtimeMetricsByUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		common.ApiErrorMsg(c, "invalid user id")
		return
	}
	snapshot := model.GetRealtimeSnapshot(userId)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    snapshot,
	})
}
