package controller

import (
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
		},
		"probe_route": service.ChannelHealthProbeRoute,
		"probe_stats": service.GetChannelProbeRunStats(),
		"channels":    snapshots,
	})
}
