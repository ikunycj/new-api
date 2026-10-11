package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetTokenRealtimeMetrics is the API-key counterpart of
// GetSelfRealtimeMetrics: the same realtime rings and response shape, but
// authenticated with the sk- token instead of the dashboard session, so
// integrations can poll RPM / TPM / concurrency from their own scripts.
//
// TokenAuthReadOnly puts the owning user's id into the context, so the user id
// is still never caller-supplied and the endpoint stays inside one account's
// data. The response is the account-level snapshot by default, matching the
// session route; an optional token_id query narrows it to one key, and a
// non-owner id would simply match nothing because the filter is applied inside
// this user's own rings. Or one model via the model parameter.
func GetTokenRealtimeMetrics(c *gin.Context) {
	userId := c.GetInt("id")
	filter, ok := parseRealtimeFilter(c)
	if !ok {
		return
	}
	snapshot := model.GetRealtimeSnapshotFiltered(userId, filter)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    snapshot,
	})
}
