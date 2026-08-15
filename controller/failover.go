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
	model.InitChannelCache()
	recordManageAudit(c, "failover.config.update", map[string]interface{}{
		"clusters": len(config.Clusters),
		"pools":    len(config.Pools),
		"policies": len(config.Policies),
	})
	common.ApiSuccess(c, nil)
}
