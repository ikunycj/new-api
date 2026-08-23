package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetFailoverConfig(c *gin.Context) {
	config, err := model.GetChannelRoutingConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

func GetFailoverMonitoring(c *gin.Context) {
	common.ApiSuccess(c, service.GetFailoverMonitoringSnapshot(c.Request.Context()))
}

func GetFailoverGrafanaAuth(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func UpdateFailoverConfig(c *gin.Context) {
	config := &model.ChannelRoutingConfig{}
	if err := c.ShouldBindJSON(config); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SaveChannelRoutingConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.config.update", map[string]interface{}{
		"routes":         len(config.Routes),
		"route_channels": len(config.RouteChannels),
	})
	common.ApiSuccess(c, nil)
}
