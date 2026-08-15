package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetFailoverConfig(c *gin.Context) {
	config, err := model.GetFailoverConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

func GetFailoverMonitoring(c *gin.Context) {
	common.ApiSuccess(c, service.GetFailoverMonitoringSnapshot(c.Request.Context()))
}

func GetChannelFailoverBindings(c *gin.Context) {
	bindings, err := model.GetChannelFailoverBindings()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, bindings)
}

func GetClusterConfiguration(c *gin.Context) {
	snapshot, err := model.GetClusterConfigurationSnapshot()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, snapshot)
}

func GetFailoverGrafanaAuth(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func UpdateFailoverConfig(c *gin.Context) {
	config := &model.FailoverConfig{}
	if err := c.ShouldBindJSON(config); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SaveFailoverConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitFailoverCache()
	model.InitChannelCache()
	recordManageAudit(c, "failover.config.update", map[string]interface{}{
		"clusters": len(config.Clusters),
		"pools":    len(config.Pools),
		"policies": len(config.Policies),
	})
	common.ApiSuccess(c, nil)
}

func UpdateChannelFailoverBindings(c *gin.Context) {
	request := &model.ChannelFailoverBindingsUpdate{}
	if err := c.ShouldBindJSON(request); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SaveChannelFailoverBindings(request.Bindings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitChannelCache()
	bound := 0
	for _, binding := range request.Bindings {
		if binding.ClusterId > 0 {
			bound++
		}
	}
	recordManageAudit(c, "failover.bindings.update", map[string]interface{}{
		"updated": len(request.Bindings),
		"bound":   bound,
		"unbound": len(request.Bindings) - bound,
	})
	common.ApiSuccess(c, nil)
}

func UpdateClusterConfiguration(c *gin.Context) {
	request := &model.ClusterConfiguration{}
	if err := c.ShouldBindJSON(request); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SaveClusterConfiguration(request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	model.InitFailoverCache()
	model.InitChannelCache()
	recordManageAudit(c, "failover.cluster.update", map[string]interface{}{
		"cluster_id":    request.Id,
		"billing_group": request.BillingGroup,
		"routes":        len(request.Routes),
	})
	common.ApiSuccess(c, gin.H{"id": request.Id})
}
