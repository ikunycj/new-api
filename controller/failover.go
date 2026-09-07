package controller

import (
	"errors"
	"io"
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

// UpdateFailoverRoute saves one billing-group route and its channel bindings
// without replacing unrelated routes in the configuration.
func UpdateFailoverRoute(c *gin.Context) {
	config := &model.BillingGroupRouteConfig{}
	if err := c.ShouldBindJSON(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := model.SaveBillingGroupRouteConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.route.save", map[string]interface{}{
		"route_id":      config.Route.Id,
		"billing_group": config.Route.BillingGroup,
		"channels":      len(config.RouteChannels),
	})
	common.ApiSuccess(c, config)
}

// DeleteFailoverRoute removes a route and all of its channel bindings.
func DeleteFailoverRoute(c *gin.Context) {
	routeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := model.DeleteBillingGroupRoute(routeID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.route.delete", map[string]interface{}{"route_id": routeID})
	common.ApiSuccess(c, nil)
}

type cleanupStaleBillingGroupRoutesRequest struct {
	RouteID int `json:"route_id"`
}

// CleanupStaleFailoverRoutes removes historical route-channel bindings that
// reference deleted channels or channels no longer in the route's group.
// A route_id can be supplied to limit cleanup to one route; zero cleans all.
func CleanupStaleFailoverRoutes(c *gin.Context) {
	request := cleanupStaleBillingGroupRoutesRequest{}
	if err := c.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if routeID := c.Query("route_id"); routeID != "" {
		parsed, err := strconv.Atoi(routeID)
		if err != nil || parsed < 0 {
			if err == nil {
				err = errors.New("route_id must be non-negative")
			}
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
			return
		}
		request.RouteID = parsed
	}
	if request.RouteID < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "route_id must be non-negative"})
		return
	}
	result, err := model.CleanupStaleBillingGroupRoutes(request.RouteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel_routing.route.cleanup_stale", map[string]interface{}{
		"route_id":               request.RouteID,
		"removed_route_channels": result.RemovedRouteChannels,
		"disabled_routes":        result.DisabledRoutes,
	})
	common.ApiSuccess(c, result)
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
