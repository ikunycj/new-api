package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const maxLoadTestStatsRequestIDs = 10000
const maxLoadTestStatsRequestBodyBytes = 2 << 20

type loadTestStatsRequest struct {
	RequestIDs []string `json:"request_ids"`
}

func GetLoadTestChannelStats(c *gin.Context) {
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil || model.NormalizeUserType(user.UserType) != model.UserTypeToB {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "load test demo is available to ToB users only"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLoadTestStatsRequestBodyBytes)
	var request loadTestStatsRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request_ids"})
		return
	}
	if len(request.RequestIDs) > maxLoadTestStatsRequestIDs {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "too many request_ids"})
		return
	}

	unique := make([]string, 0, len(request.RequestIDs))
	seen := make(map[string]struct{}, len(request.RequestIDs))
	for _, requestID := range request.RequestIDs {
		requestID = strings.TrimSpace(requestID)
		if requestID == "" || len(requestID) > 128 {
			continue
		}
		if _, exists := seen[requestID]; exists {
			continue
		}
		seen[requestID] = struct{}{}
		unique = append(unique, requestID)
	}

	stats, err := model.GetLoadTestChannelStats(c.GetInt("id"), unique)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}
