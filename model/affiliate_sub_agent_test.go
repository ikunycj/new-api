package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAffiliateSubAgentAdminBindingsPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	directInviter, child := createAffiliateUsers(t, "sub-agent@example.com")
	parent := User{Username: "parent-agent", AffCode: "PARENT01", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&parent).Error)
	anotherChild := User{Username: "another-agent", AffCode: "PARENT02", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&anotherChild).Error)
	directRate, upperRate, reducedRate, remove := int64(2000), int64(15_001), int64(3500), int64(-1)
	createAffiliateUserOverride(t, child.Id, parent.Id, AffiliateUserOverride{RewardRateBps: &directRate})

	_, err := SaveAffiliateAgents(parent.Id, []AffiliateAgentUpdate{{
		UserID: parent.Id,
		SubAgents: []AffiliateSubAgentUpdate{
			{UserID: child.Id, RateBps: &upperRate},
			{UserID: anotherChild.Id, RateBps: &reducedRate},
		},
	}})
	require.NoError(t, err)
	items, err := GetAffiliateAdminSubAgents(parent.Id)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, child.Id, items[0].UserID)
	assert.Equal(t, upperRate, items[0].RateBps)
	assert.Equal(t, anotherChild.Id, items[1].UserID)
	assert.Equal(t, reducedRate, items[1].RateBps)

	var childOverride AffiliateUserOverride
	require.NoError(t, DB.Where("user_id = ?", child.Id).First(&childOverride).Error)
	require.NotNil(t, childOverride.RewardRateBps)
	assert.Equal(t, directRate, *childOverride.RewardRateBps)
	var newAgent AffiliateUserOverride
	require.NoError(t, DB.Where("user_id = ?", anotherChild.Id).First(&newAgent).Error)
	var referral AffiliateReferral
	require.NoError(t, DB.Where("invitee_user_id = ?", child.Id).First(&referral).Error)
	assert.Equal(t, directInviter.Id, referral.InviterUserID)
	invalidRate := int64(-2)
	_, err = SaveAffiliateAgents(parent.Id, []AffiliateAgentUpdate{{
		UserID: parent.Id, SubAgents: []AffiliateSubAgentUpdate{{UserID: child.Id, RateBps: &invalidRate}},
	}})
	require.ErrorContains(t, err, "大于或等于 0%")

	_, err = SaveAffiliateAgents(parent.Id, []AffiliateAgentUpdate{{
		UserID: directInviter.Id, SubAgents: []AffiliateSubAgentUpdate{{UserID: child.Id, RateBps: &upperRate}},
	}})
	require.ErrorContains(t, err, "已有上级")
	_, err = SaveAffiliateAgents(parent.Id, []AffiliateAgentUpdate{{
		UserID: child.Id, SubAgents: []AffiliateSubAgentUpdate{{UserID: directInviter.Id, RateBps: &upperRate}},
	}})
	require.ErrorContains(t, err, "仅支持一层")

	_, err = SaveAffiliateAgents(parent.Id, []AffiliateAgentUpdate{{
		UserID: parent.Id, SubAgents: []AffiliateSubAgentUpdate{
			{UserID: child.Id, RateBps: &reducedRate},
			{UserID: anotherChild.Id, RateBps: &remove},
		},
	}})
	require.NoError(t, err)
	items, err = GetAffiliateAdminSubAgents(parent.Id)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, reducedRate, items[0].RateBps)

	require.NoError(t, DeleteAffiliateUserOverride(parent.Id))
	items, err = GetAffiliateAdminSubAgents(parent.Id)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestAffiliateAdminAgentListIncludesSubAgentsPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	parents := []User{
		{Username: "parent-a", AffCode: "PARENT01", Status: common.UserStatusEnabled},
		{Username: "parent-b", AffCode: "PARENT02", Status: common.UserStatusEnabled},
		{Username: "parent-c", AffCode: "PARENT03", Status: common.UserStatusEnabled},
	}
	children := []User{
		{Username: "child-a", Email: "a@example.com", AffCode: "CHILD01", Status: common.UserStatusEnabled},
		{Username: "child-b", Email: "b@example.com", AffCode: "CHILD02", Status: common.UserStatusEnabled},
		{Username: "child-c", Email: "c@example.com", AffCode: "CHILD03", Status: common.UserStatusEnabled},
	}
	require.NoError(t, DB.Create(&parents).Error)
	require.NoError(t, DB.Create(&children).Error)

	firstRate, secondRate, pausedRate := int64(500), int64(15_001), int64(0)
	_, err := SaveAffiliateAgents(parents[0].Id, []AffiliateAgentUpdate{
		{UserID: parents[0].Id, SubAgents: []AffiliateSubAgentUpdate{
			{UserID: children[1].Id, RateBps: &secondRate},
			{UserID: children[0].Id, RateBps: &firstRate},
		}},
		{UserID: parents[1].Id, SubAgents: []AffiliateSubAgentUpdate{
			{UserID: children[2].Id, RateBps: &pausedRate},
		}},
		{UserID: parents[2].Id},
	})
	require.NoError(t, err)

	items, total, err := SearchAffiliateUserOverrides("parent-", 0, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, items, 2)
	assert.Equal(t, parents[0].Id, items[0].UserID)
	assert.Equal(t, []AffiliateSubAgentView{
		{UserID: children[0].Id, Username: children[0].Username, Email: children[0].Email, RateBps: firstRate},
		{UserID: children[1].Id, Username: children[1].Username, Email: children[1].Email, RateBps: secondRate},
	}, items[0].SubAgents)
	assert.Equal(t, parents[1].Id, items[1].UserID)
	assert.Equal(t, []AffiliateSubAgentView{
		{UserID: children[2].Id, Username: children[2].Username, Email: children[2].Email, RateBps: pausedRate},
	}, items[1].SubAgents)

	items, total, err = SearchAffiliateUserOverrides("parent-", 2, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, items, 1)
	assert.Equal(t, parents[2].Id, items[0].UserID)
	data, err := common.Marshal(items[0])
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"sub_agents"`)

	detail, err := GetAffiliateUserOverrideView(parents[0].Id)
	require.NoError(t, err)
	assert.Empty(t, detail.SubAgents)
}
