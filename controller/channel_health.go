package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

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
