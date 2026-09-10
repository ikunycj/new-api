package controller

import (
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// RegisterChannelHealthScoreInjection wires the live health score lookup into
// the routing layer. The routing logic lives in the model package, which cannot
// import service (service imports model); so the getter is injected here in
// the same way the probe executor is. Without this registration, routing falls
// back to the legacy PreviousDayProbeSuccessRate even when the subsystem runs
// in active mode.
func RegisterChannelHealthScoreInjection() {
	model.SetChannelHealthScoreGetter(func(channelID int, route string, modelName string) (float64, bool) {
		snapshot, ok := service.GetChannelHealthSnapshot(channelID, route, modelName)
		if !ok {
			return 0, false
		}
		return snapshot.Score, true
	})
}

// GetChannelHealth exposes the read-only channel health scores collected by the
// EWMA scorer. It is the primary observability surface for phase 1: operators
// run the subsystem in observe mode and inspect real scores here before ever
// letting them influence routing.
func GetChannelHealth(c *gin.Context) {
	snapshots := service.ListChannelHealthSnapshots()
	if snapshots == nil {
		snapshots = []service.ChannelHealthSnapshot{}
	}
	common.ApiSuccess(c, gin.H{
		"enabled": common.IsChannelHealthEnabled(),
		"active":  common.IsChannelHealthActive(),
		"mode":    common.ChannelHealthMode(),
		"config": gin.H{
			"half_life_seconds":         common.ChannelHealthHalfLifeSeconds(),
			"min_samples":               common.ChannelHealthMinSamples(),
			"latency_half_life_seconds": common.ChannelHealthLatencyHalfLifeSeconds(),
			"state_ttl_seconds":         common.ChannelHealthStateTTLSeconds(),
			"probe_enabled":             common.IsChannelHealthProbeEnabled(),
			"probe_interval_seconds":    common.ChannelHealthProbeIntervalSeconds(),
			"probe_idle_grace_seconds":  common.ChannelHealthProbeIdleGraceSeconds(),
			"history_enabled":           common.IsChannelHealthHistoryEnabled(),
			"history_bucket_seconds":    common.ChannelHealthHistoryBucketSeconds(),
			"history_retention_days":    common.ChannelHealthHistoryRetentionDays(),
		},
		"probe_route": service.ChannelHealthProbeRoute,
		"probe_stats": service.GetChannelProbeRunStats(),
		"channels":    snapshots,
	})
}

// GetChannelHealthHistory serves the persisted score time series. It is a
// separate endpoint from GetChannelHealth because the two have different cost
// profiles and refresh needs: the snapshot is cheap and polled often, the
// history is a range scan the client only needs when it draws the chart.
func GetChannelHealthHistory(c *gin.Context) {
	hours, err := strconv.Atoi(c.DefaultQuery("hours", "6"))
	if err != nil || hours <= 0 {
		hours = 6
	}
	// Cap the window so a hand-crafted query cannot ask for a full-table scan.
	if hours > 24*30 {
		hours = 24 * 30
	}

	startTs := model.ChannelHealthHistoryStartTime(hours)
	rows, err := model.GetChannelHealthHistory(startTs, time.Now().Unix())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if rows == nil {
		rows = []model.ChannelHealthHistory{}
	}
	common.ApiSuccess(c, gin.H{
		"enabled":        common.IsChannelHealthHistoryEnabled(),
		"bucket_seconds": common.ChannelHealthHistoryBucketSeconds(),
		"retention_days": common.ChannelHealthHistoryRetentionDays(),
		"probe_route":    service.ChannelHealthProbeRoute,
		"hours":          hours,
		"points":         rows,
	})
}
