package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingGroupActivityTracksConnectionsAndUniqueUsers(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	localPricingGroupActivity.Lock()
	previousGroups := localPricingGroupActivity.groups
	previousUsers := localPricingGroupActivity.users
	localPricingGroupActivity.groups = make(map[string]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.users = make(map[int]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		localPricingGroupActivity.Lock()
		localPricingGroupActivity.groups = previousGroups
		localPricingGroupActivity.users = previousUsers
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
	count, degraded := GetUserInFlightRequests(7)
	assert.Equal(t, 2, count)
	assert.False(t, degraded)

	finishFirst()
	activity = GetPricingGroupActivity([]string{"paid"})
	assert.Equal(t, PricingGroupActivity{Users: 2, Connections: 2}, activity["paid"])
	count, degraded = GetUserInFlightRequests(7)
	assert.Equal(t, 1, count)
	assert.False(t, degraded)

	finishSecond()
	activity = GetPricingGroupActivity([]string{"paid"})
	assert.Equal(t, PricingGroupActivity{Users: 1, Connections: 1}, activity["paid"])

	finishThird()
	activity = GetPricingGroupActivity([]string{"paid"})
	assert.Equal(t, PricingGroupActivity{}, activity["paid"])
	count, degraded = GetUserInFlightRequests(8)
	assert.Zero(t, count)
	assert.False(t, degraded)
}

func TestPricingGroupActivityMovesWithCrossGroupRetry(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	localPricingGroupActivity.Lock()
	previousGroups := localPricingGroupActivity.groups
	previousUsers := localPricingGroupActivity.users
	localPricingGroupActivity.groups = make(map[string]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.users = make(map[int]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		localPricingGroupActivity.Lock()
		localPricingGroupActivity.groups = previousGroups
		localPricingGroupActivity.users = previousUsers
		localPricingGroupActivity.Unlock()
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	finish := BeginPricingGroupActivity(ctx, "paid", 7, "request-1")
	t.Cleanup(finish)
	UpdatePricingGroupActivity(ctx, "backup")

	activity := GetPricingGroupActivity([]string{"paid", "backup"})
	assert.Equal(t, PricingGroupActivity{}, activity["paid"])
	assert.Equal(t, PricingGroupActivity{Users: 1, Connections: 1}, activity["backup"])
	count, degraded := GetUserInFlightRequests(7)
	assert.Equal(t, 1, count)
	assert.False(t, degraded)
	finish()
	count, degraded = GetUserInFlightRequests(7)
	assert.Zero(t, count)
	assert.False(t, degraded)
}

func TestPricingGroupActivityTracksUserBeforeGroupSelection(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	localPricingGroupActivity.Lock()
	previousGroups := localPricingGroupActivity.groups
	previousUsers := localPricingGroupActivity.users
	localPricingGroupActivity.groups = make(map[string]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.users = make(map[int]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		localPricingGroupActivity.Lock()
		localPricingGroupActivity.groups = previousGroups
		localPricingGroupActivity.users = previousUsers
		localPricingGroupActivity.Unlock()
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	finish := BeginPricingGroupActivity(ctx, "", 7, "request-1")
	count, degraded := GetUserInFlightRequests(7)
	assert.Equal(t, 1, count)
	assert.False(t, degraded)

	UpdatePricingGroupActivity(ctx, "paid")
	assert.Equal(t, PricingGroupActivity{Users: 1, Connections: 1}, GetPricingGroupActivity([]string{"paid"})["paid"])
	finish()
	count, degraded = GetUserInFlightRequests(7)
	assert.Zero(t, count)
	assert.False(t, degraded)
}

func TestGetUserInFlightRequestsUsesUserIndexAndIsolatesUsers(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	localPricingGroupActivity.Lock()
	previousGroups := localPricingGroupActivity.groups
	previousUsers := localPricingGroupActivity.users
	localPricingGroupActivity.groups = map[string]map[string]pricingGroupActivityEntry{
		"paid": {
			"7|node-a|request-1": {userID: 7, expiresAt: time.Now().Add(time.Minute).UnixMilli()},
		},
		"backup": {
			"7|node-a|request-1": {userID: 7, expiresAt: time.Now().Add(time.Minute).UnixMilli()},
			"7|node-b|request-2": {userID: 7, expiresAt: time.Now().Add(time.Minute).UnixMilli()},
			"8|node-a|request-3": {userID: 8, expiresAt: time.Now().Add(time.Minute).UnixMilli()},
		},
	}
	localPricingGroupActivity.users = map[int]map[string]pricingGroupActivityEntry{
		7: {
			"7|node-a|request-1": {userID: 7, expiresAt: time.Now().Add(time.Minute).UnixMilli()},
			"7|node-b|request-2": {userID: 7, expiresAt: time.Now().Add(time.Minute).UnixMilli()},
			"7|node-c|expired":   {userID: 7, expiresAt: time.Now().Add(-time.Minute).UnixMilli()},
		},
		8: {
			"8|node-a|request-3": {userID: 8, expiresAt: time.Now().Add(time.Minute).UnixMilli()},
		},
	}
	localPricingGroupActivity.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		localPricingGroupActivity.Lock()
		localPricingGroupActivity.groups = previousGroups
		localPricingGroupActivity.users = previousUsers
		localPricingGroupActivity.Unlock()
	})

	count, degraded := GetUserInFlightRequests(7)
	assert.Equal(t, 2, count)
	assert.False(t, degraded)

	count, degraded = GetUserInFlightRequests(8)
	assert.Equal(t, 1, count)
	assert.False(t, degraded)
}

func TestGetUserInFlightRequestsReportsMissingRedisAsDegraded(t *testing.T) {
	previousRedisEnabled, previousRDB := common.RedisEnabled, common.RDB
	common.RedisEnabled = true
	common.RDB = nil
	localPricingGroupActivity.Lock()
	previousUsers := localPricingGroupActivity.users
	localPricingGroupActivity.users = make(map[int]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled, common.RDB = previousRedisEnabled, previousRDB
		localPricingGroupActivity.Lock()
		localPricingGroupActivity.users = previousUsers
		localPricingGroupActivity.Unlock()
	})

	count, degraded := GetUserInFlightRequests(7)
	assert.Zero(t, count)
	assert.True(t, degraded)
}

func TestGetTokenInFlightRequestsIsolatesApiKeys(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	localPricingGroupActivity.Lock()
	previousGroups := localPricingGroupActivity.groups
	previousUsers := localPricingGroupActivity.users
	previousTokens := localPricingGroupActivity.tokens
	localPricingGroupActivity.groups = make(map[string]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.users = make(map[int]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.tokens = make(map[int]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		localPricingGroupActivity.Lock()
		localPricingGroupActivity.groups = previousGroups
		localPricingGroupActivity.users = previousUsers
		localPricingGroupActivity.tokens = previousTokens
		localPricingGroupActivity.Unlock()
	})

	newContext := func(tokenID int) *gin.Context {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		common.SetContextKey(ctx, constant.ContextKeyTokenId, tokenID)
		return ctx
	}
	first := BeginPricingGroupActivity(newContext(101), "paid", 7, "request-1")
	second := BeginPricingGroupActivity(newContext(101), "paid", 7, "request-2")
	other := BeginPricingGroupActivity(newContext(202), "paid", 7, "request-3")
	t.Cleanup(first)
	t.Cleanup(second)
	t.Cleanup(other)

	counts, degraded := GetTokenInFlightRequests([]int{101, 202, 303, 101})
	assert.Equal(t, map[int]int{101: 2, 202: 1, 303: 0}, counts)
	assert.False(t, degraded)

	first()
	counts, degraded = GetTokenInFlightRequests([]int{101, 202})
	assert.Equal(t, map[int]int{101: 1, 202: 1}, counts)
	assert.False(t, degraded)
}

func TestGetTokenInFlightRequestsReportsMissingRedisAsDegraded(t *testing.T) {
	previousRedisEnabled, previousRDB := common.RedisEnabled, common.RDB
	common.RedisEnabled = true
	common.RDB = nil
	localPricingGroupActivity.Lock()
	previousTokens := localPricingGroupActivity.tokens
	localPricingGroupActivity.tokens = make(map[int]map[string]pricingGroupActivityEntry)
	localPricingGroupActivity.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled, common.RDB = previousRedisEnabled, previousRDB
		localPricingGroupActivity.Lock()
		localPricingGroupActivity.tokens = previousTokens
		localPricingGroupActivity.Unlock()
	})

	counts, degraded := GetTokenInFlightRequests([]int{101})
	assert.Equal(t, map[int]int{101: 0}, counts)
	assert.True(t, degraded)
}
