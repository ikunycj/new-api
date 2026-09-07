package controller

import (
	"net/http"
	"strconv"

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
	if !common.MonitoringEnabled {
		c.Status(http.StatusNotFound)
		return
	}
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

func CreateBillingGroupRoute(c *gin.Context) {
	route := &model.BillingGroupRoute{}
	if err := c.ShouldBindJSON(route); err != nil {
		common.ApiError(c, err)
		return
	}
	route.Id = 0
	if err := model.SaveBillingGroupRoute(route); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.config.update", map[string]interface{}{"operation": "route_create", "route_id": route.Id, "billing_group": route.BillingGroup})
	common.ApiSuccess(c, route)
}

func UpdateBillingGroupRoute(c *gin.Context) {
	routeID, err := strconv.Atoi(c.Param("route_id"))
	if err != nil || routeID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid route id"})
		return
	}
	route := &model.BillingGroupRoute{}
	if err := c.ShouldBindJSON(route); err != nil {
		common.ApiError(c, err)
		return
	}
	route.Id = routeID
	if err := model.SaveBillingGroupRoute(route); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.config.update", map[string]interface{}{"operation": "route_update", "route_id": route.Id, "billing_group": route.BillingGroup})
	common.ApiSuccess(c, route)
}

func DeleteBillingGroupRoute(c *gin.Context) {
	routeID, err := strconv.Atoi(c.Param("route_id"))
	if err != nil || routeID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid route id"})
		return
	}
	if err := model.DeleteBillingGroupRoute(routeID); err != nil {
		common.ApiError(c, err)
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.config.update", map[string]interface{}{"operation": "route_delete", "route_id": routeID})
	common.ApiSuccess(c, nil)
}

func SaveBillingGroupRouteChannel(c *gin.Context) {
	routeID, err := strconv.Atoi(c.Param("route_id"))
	if err != nil || routeID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid route id"})
		return
	}
	entry := &model.BillingGroupChannel{}
	if err := c.ShouldBindJSON(entry); err != nil {
		common.ApiError(c, err)
		return
	}
	if pathChannelID := c.Param("channel_id"); pathChannelID != "" {
		channelID, parseErr := strconv.Atoi(pathChannelID)
		if parseErr != nil || channelID <= 0 || channelID != entry.ChannelId {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "route channel path does not match request body"})
			return
		}
	}
	if err := model.SaveBillingGroupRouteChannel(routeID, entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.config.update", map[string]interface{}{"operation": "route_channel_save", "route_id": routeID, "channel_id": entry.ChannelId})
	common.ApiSuccess(c, entry)
}

func DeleteBillingGroupRouteChannel(c *gin.Context) {
	routeID, routeErr := strconv.Atoi(c.Param("route_id"))
	channelID, channelErr := strconv.Atoi(c.Param("channel_id"))
	if routeErr != nil || channelErr != nil || routeID <= 0 || channelID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid route channel id"})
		return
	}
	if err := model.DeleteBillingGroupRouteChannel(routeID, channelID); err != nil {
		common.ApiError(c, err)
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.config.update", map[string]interface{}{"operation": "route_channel_delete", "route_id": routeID, "channel_id": channelID})
	common.ApiSuccess(c, nil)
}

type billingGroupTypeUpdateRequest struct {
	BillingGroup string `json:"billing_group"`
	GroupType    string `json:"group_type"`
}

func UpdateBillingGroupType(c *gin.Context) {
	var request billingGroupTypeUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := model.UpdateBillingGroupType(request.BillingGroup, request.GroupType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.group_type.update", map[string]interface{}{
		"billing_group": request.BillingGroup,
		"group_type":    request.GroupType,
	})
	common.ApiSuccess(c, nil)
}
