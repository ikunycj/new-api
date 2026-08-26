package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetAdminConsole(c *gin.Context) {
	stats, err := model.GetAdminConsoleStats()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}

func GetAdminConsoleSystemLoad(c *gin.Context) {
	common.ApiSuccess(c, model.GetAdminConsoleSystemLoad())
}
