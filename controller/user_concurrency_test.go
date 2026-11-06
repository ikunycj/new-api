package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserListResponseIncludesConcurrencyAcrossKeysAndGroups(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = previousRedisEnabled })

	users := []*model.User{
		{Id: 81001, Username: "active", Quota: 500},
		{Id: 81002, Username: "idle", Quota: 200},
	}
	firstContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(firstContext, constant.ContextKeyTokenId, 81011)
	finishFirst := service.BeginPricingGroupActivity(firstContext, "paid", users[0].Id, "user-list-first")
	t.Cleanup(finishFirst)
	secondContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(secondContext, constant.ContextKeyTokenId, 81012)
	finishSecond := service.BeginPricingGroupActivity(secondContext, "backup", users[0].Id, "user-list-second")
	t.Cleanup(finishSecond)

	encoded, err := common.Marshal(buildUserListResponses(users))
	require.NoError(t, err)
	var response []struct {
		ID                  int    `json:"id"`
		Username            string `json:"username"`
		Quota               int    `json:"quota"`
		CurrentConcurrency  *int   `json:"current_concurrency"`
		ConcurrencyDegraded *bool  `json:"concurrency_degraded"`
	}
	require.NoError(t, common.Unmarshal(encoded, &response))
	require.Len(t, response, 2)
	for i, item := range response {
		assert.Equal(t, users[i].Id, item.ID)
		assert.Equal(t, users[i].Username, item.Username)
		assert.Equal(t, users[i].Quota, item.Quota)
		require.NotNil(t, item.CurrentConcurrency)
		require.NotNil(t, item.ConcurrencyDegraded)
		assert.False(t, *item.ConcurrencyDegraded)
	}
	assert.Equal(t, 2, *response[0].CurrentConcurrency)
	assert.Zero(t, *response[1].CurrentConcurrency)

	service.UpdatePricingGroupActivity(firstContext, "backup")
	assert.Equal(t, 2, buildUserListResponses(users)[0].CurrentConcurrency)
	finishFirst()
	assert.Equal(t, 1, buildUserListResponses(users)[0].CurrentConcurrency)
	finishSecond()
	assert.Zero(t, buildUserListResponses(users)[0].CurrentConcurrency)
}

func TestUserListResponseMarksUnavailableCrossNodeConcurrency(t *testing.T) {
	previousRedisEnabled, previousRDB := common.RedisEnabled, common.RDB
	common.RedisEnabled, common.RDB = true, nil
	t.Cleanup(func() { common.RedisEnabled, common.RDB = previousRedisEnabled, previousRDB })

	response := buildUserListResponses([]*model.User{{Id: 82001}, {Id: 82002}})
	require.Len(t, response, 2)
	for _, item := range response {
		assert.True(t, item.ConcurrencyDegraded)
	}

	encoded, err := common.Marshal(buildUserListResponses(nil))
	require.NoError(t, err)
	assert.JSONEq(t, "[]", string(encoded))
}
