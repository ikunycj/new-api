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
