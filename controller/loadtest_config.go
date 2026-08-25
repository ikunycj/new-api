package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

func GetLoadTestConfig(c *gin.Context) {
	settings := operation_setting.GetLoadTestSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"min_duration_seconds": operation_setting.LoadTestMinDurationSeconds,
			"max_duration_seconds": settings.MaxDurationSeconds,
			"min_rps":              1,
			"max_rps":              settings.MaxRPS,
			"min_concurrency":      1,
			"max_concurrency":      settings.MaxConcurrency,
			"max_requests":         operation_setting.LoadTestMaxRequests,
		},
	})
}
