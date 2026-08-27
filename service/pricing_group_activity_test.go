package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingGroupActivityTracksConnectionsAndUniqueUsers(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	localPricingGroupActivity.Lock()
	previousGroups := localPricingGroupActivity.groups
	localPricingGroupActivity.groups = make(map[string]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		localPricingGroupActivity.Lock()
		localPricingGroupActivity.groups = previousGroups
		localPricingGroupActivity.Unlock()
	})
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	finishFirst := BeginPricingGroupActivity(ctx, "paid", 7, "request-1")
	finishSecond := BeginPricingGroupActivity(ctx, "paid", 7, "request-2")
	finishThird := BeginPricingGroupActivity(ctx, "paid", 8, "request-3")
	t.Cleanup(finishFirst)
	t.Cleanup(finishSecond)
	t.Cleanup(finishThird)

	activity := GetPricingGroupActivity([]string{"paid", "idle"})
	require.Contains(t, activity, "paid")
	assert.Equal(t, PricingGroupActivity{Users: 2, Connections: 3}, activity["paid"])
	assert.Equal(t, PricingGroupActivity{}, activity["idle"])
	assert.Equal(t, 3, GetTotalPricingGroupConnections([]string{"paid", "idle"}))

	finishFirst()
	activity = GetPricingGroupActivity([]string{"paid"})
	assert.Equal(t, PricingGroupActivity{Users: 2, Connections: 2}, activity["paid"])

	finishSecond()
	activity = GetPricingGroupActivity([]string{"paid"})
	assert.Equal(t, PricingGroupActivity{Users: 1, Connections: 1}, activity["paid"])

	finishThird()
	activity = GetPricingGroupActivity([]string{"paid"})
	assert.Equal(t, PricingGroupActivity{}, activity["paid"])
}

func TestPricingGroupActivityMovesWithCrossGroupRetry(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	localPricingGroupActivity.Lock()
	previousGroups := localPricingGroupActivity.groups
	localPricingGroupActivity.groups = make(map[string]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		localPricingGroupActivity.Lock()
		localPricingGroupActivity.groups = previousGroups
		localPricingGroupActivity.Unlock()
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	finish := BeginPricingGroupActivity(ctx, "paid", 7, "request-1")
	t.Cleanup(finish)
	UpdatePricingGroupActivity(ctx, "backup")

	activity := GetPricingGroupActivity([]string{"paid", "backup"})
	assert.Equal(t, PricingGroupActivity{}, activity["paid"])
	assert.Equal(t, PricingGroupActivity{Users: 1, Connections: 1}, activity["backup"])
}
