package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetPackageComparison(c *gin.Context) {
	parts := strings.Split(c.Query("plan_ids"), ",")
	planIDs := make([]int, 0, len(parts))
	seen := make(map[int]struct{}, len(parts))
	for _, part := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			common.ApiErrorMsg(c, "plan_ids must contain positive integers")
			return
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		planIDs = append(planIDs, id)
	}
	start, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if start <= 0 {
		start = time.Now().Add(-24 * time.Hour).Unix()
	}
	if end <= 0 {
		end = time.Now().Unix()
	}
	stats, err := model.GetPackageComparisonStats(planIDs, start, end, c.Query("model_name"), c.Query("group"))
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{
		"start_timestamp": start,
		"end_timestamp":   end,
		"plans":           stats,
	})
}
