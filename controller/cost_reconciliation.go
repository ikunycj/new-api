package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetCostReconciliation(c *gin.Context) {
	page := common.GetPageQuery(c)
	start, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	now := common.GetTimestamp()
	if start <= 0 {
		start = now - 24*60*60
	}
	if end <= 0 {
		end = now
	}
	if end <= start || end-start > 31*24*60*60 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "cost reconciliation range must be between 1 minute and 31 days"})
		return
	}
	userID, _ := strconv.Atoi(c.Query("user_id"))
	tokenID, _ := strconv.Atoi(c.Query("token_id"))
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	if channelID == 0 {
		channelID, _ = strconv.Atoi(c.Query("channel"))
	}
	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("username")
	}
	rows, total, totals, err := model.ListCostReconciliationRollups(model.CostReconciliationQuery{
		StartTimestamp: start,
		EndTimestamp:   end,
		UserID:         userID,
		TokenID:        tokenID,
		ChannelID:      channelID,
		Group:          c.Query("group"),
		Keyword:        keyword,
		TokenName:      c.Query("token_name"),
		Limit:          page.GetPageSize(),
		Offset:         page.GetStartIdx(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	page.SetTotal(int(total))
	page.SetItems(rows)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"items": rows, "total": total, "totals": totals, "page": page.GetPage(), "page_size": page.GetPageSize()},
	})
}
