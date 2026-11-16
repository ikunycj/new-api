package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAffiliateSettlementCursorPostgres(t *testing.T) {
	for _, test := range []struct {
		name     string
		campaign bool
		settle   func(int64, int, int) (int, int, error)
	}{
		{name: "quota", settle: ReleaseDueAffiliateRewards},
		{name: "cash", campaign: true, settle: ReleaseDueAffiliateCashRewards},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupPostgresAffiliateTestDB(t)
			unit := configureAffiliateTest(t, nil)
			now := common.GetTimestamp()
			var campaign *AffiliateCampaign
			if test.campaign {
				campaign = enableAffiliateCampaignForTest(t, now-60, now+3600, 0)
			}
			inviter, invitee := createAffiliateUsers(t, "blocked@example.com")
			require.NoError(t, DB.Model(inviter).Update("quota", common.MaxQuota).Error)
			first := createAffiliateTopUp(t, invitee.Id, "blocked-reward", 5, TopUpCompletionSourceOnlineWallet)
			require.NoError(t, ProcessAffiliateTopUp(first.Id))

			other := &User{Username: "other-agent", AffCode: "OTHERAGENT", Status: common.UserStatusEnabled}
			otherInvitee := &User{Username: "other-invitee", AffCode: "OTHERINVITEE", Status: common.UserStatusEnabled}
			require.NoError(t, DB.Create(other).Error)
			otherInvitee.InviterId = other.Id
			require.NoError(t, DB.Create(otherInvitee).Error)
			require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
				return initializeAffiliateUserWithTx(tx, otherInvitee, other.Id, false)
			}))
			if campaign != nil {
				require.NoError(t, DB.Model(campaign).Update("hold_seconds", 60).Error)
			} else {
				operation_setting.GetAffiliateSetting().HoldSeconds = 60
			}
			second := createAffiliateTopUp(t, otherInvitee.Id, "payable-reward", 5, TopUpCompletionSourceOnlineWallet)
			require.NoError(t, ProcessAffiliateTopUp(second.Id))

			rewardModel := interface{}(&AffiliateReward{})
			credit := unit
			if test.campaign {
				rewardModel = &AffiliateCashReward{}
				credit = 5 * unit / 4
			}
			var rewards []struct {
				ID          int
				Status      string
				AvailableAt int64
			}
			require.NoError(t, DB.Model(rewardModel).Select("id", "status", "available_at").Order("id asc").Find(&rewards).Error)
			require.Len(t, rewards, 2)
			assert.Equal(t, AffiliateRewardStatusAvailable, rewards[0].Status)
			assert.Equal(t, AffiliateRewardStatusPending, rewards[1].Status)
			dueAt := rewards[1].AvailableAt

			// A page with no released reward still advances past a full wallet.
			released, cursor, err := test.settle(dueAt, 0, 1)
			require.NoError(t, err)
			assert.Zero(t, released)
			require.Equal(t, rewards[0].ID, cursor)
			released, cursor, err = test.settle(dueAt, cursor, 1)
			require.NoError(t, err)
			assert.Equal(t, 1, released)
			require.Equal(t, rewards[1].ID, cursor)
			released, nextCursor, err := test.settle(dueAt, cursor, 1)
			require.NoError(t, err)
			assert.Zero(t, released)
			assert.Equal(t, cursor, nextCursor)
			require.NoError(t, DB.First(other, other.Id).Error)
			assert.Equal(t, int(credit), other.Quota)
			require.NoError(t, DB.Model(rewardModel).Select("id", "status", "available_at").Order("id asc").Find(&rewards).Error)
			assert.Equal(t, AffiliateRewardStatusAvailable, rewards[0].Status)
			assert.Equal(t, AffiliateRewardStatusTransferred, rewards[1].Status)

			// A new run retries deferred rewards and leaves completed ones alone.
			require.NoError(t, DB.Model(inviter).Update("quota", common.MaxQuota-int(credit)).Error)
			released, cursor, err = test.settle(dueAt, 0, 1)
			require.NoError(t, err)
			assert.Zero(t, released)
			assert.Equal(t, rewards[0].ID, cursor)
			released, nextCursor, err = test.settle(dueAt, cursor, 1)
			require.NoError(t, err)
			assert.Zero(t, released)
			assert.Equal(t, cursor, nextCursor)
			require.NoError(t, DB.First(inviter, inviter.Id).Error)
			assert.Equal(t, common.MaxQuota, inviter.Quota)
			require.NoError(t, DB.First(other, other.Id).Error)
			assert.Equal(t, int(credit), other.Quota)
		})
	}
}

func TestAffiliatePaidTopUpAtInviterWalletLimitPostgres(t *testing.T) {
	for _, campaign := range []bool{false, true} {
		name := "percentage"
		if campaign {
			name = "campaign"
		}
		t.Run(name, func(t *testing.T) {
			setupPostgresAffiliateTestDB(t)
			unit := configureAffiliateTest(t, nil)
			if campaign {
				now := common.GetTimestamp()
				enableAffiliateCampaignForTest(t, now-60, now+3600, 0)
			}
			inviter, invitee := createAffiliateUsers(t, "wallet-limit@example.com")
			require.NoError(t, DB.Model(inviter).Update("quota", common.MaxQuota).Error)
			topUp := &TopUp{
				UserId: invitee.Id, TradeNo: "paid-at-limit", Amount: 5, Money: 5,
				PaymentProvider: PaymentProviderEpay, Status: common.TopUpStatusPending,
				CreateTime: common.GetTimestamp(),
			}
			require.NoError(t, DB.Create(topUp).Error)
			completed, quota, changed, err := CompleteEpayTopUp(topUp.TradeNo, "alipay")
			require.NoError(t, err)
			assert.True(t, changed)
			assert.Equal(t, common.TopUpStatusSuccess, completed.Status)
			assert.Equal(t, int(5*unit), quota)
			_, _, repeated, err := CompleteEpayTopUp(topUp.TradeNo, "alipay")
			require.NoError(t, err)
			assert.False(t, repeated)
			require.NoError(t, DB.First(invitee, invitee.Id).Error)
			wantInvitee := 5 * unit
			if campaign {
				wantInvitee += unit
			}
			assert.Equal(t, int(wantInvitee), invitee.Quota)

			if campaign {
				var reward AffiliateCashReward
				require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).First(&reward).Error)
				assert.Equal(t, AffiliateRewardStatusAvailable, reward.Status)
				assert.Zero(t, reward.TransferredCents)
				credit := int(reward.CashbackCents * unit / 100)
				require.NoError(t, DB.Model(inviter).Update("quota", common.MaxQuota-credit).Error)
				_, _, err = ReleaseDueAffiliateCashRewards(common.GetTimestamp(), 0, 20)
				require.NoError(t, err)
				require.NoError(t, DB.First(&reward, reward.ID).Error)
				assert.Equal(t, AffiliateRewardStatusTransferred, reward.Status)
				assert.Equal(t, reward.CashbackCents, reward.TransferredCents)
			} else {
				var reward AffiliateReward
				require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).First(&reward).Error)
				assert.Equal(t, AffiliateRewardStatusAvailable, reward.Status)
				assert.Zero(t, reward.TransferredQuota)
				assert.Equal(t, unit, reward.ActualQuota)
				require.NoError(t, DB.Model(inviter).Update("quota", common.MaxQuota-int(unit)).Error)
				_, _, err = ReleaseDueAffiliateRewards(common.GetTimestamp(), 0, 20)
				require.NoError(t, err)
				require.NoError(t, DB.First(&reward, reward.ID).Error)
				assert.Equal(t, AffiliateRewardStatusTransferred, reward.Status)
				assert.Equal(t, unit, reward.TransferredQuota)
			}
			require.NoError(t, DB.First(inviter, inviter.Id).Error)
			assert.Equal(t, common.MaxQuota, inviter.Quota)
		})
	}
}

func TestAffiliateAgentDefaultAndDomainClearingPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	configureAffiliateTest(t, nil)
	previousAddress := system_setting.ServerAddress
	t.Cleanup(func() { system_setting.ServerAddress = previousAddress })
	system_setting.ServerAddress = "https://example.com"
	agent := &User{Username: "default-agent", AffCode: "DEFAULTAGENT", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(agent).Error)
	views, err := SaveAffiliateAgents(agent.Id, []AffiliateAgentUpdate{{UserID: agent.Id}})
	require.NoError(t, err)
	require.Len(t, views, 1)
	require.NotNil(t, views[0].Override)
	assert.Nil(t, views[0].Override.RewardRateBps)
	assert.Equal(t, "global", views[0].EffectiveRule.Source)
	operation_setting.GetAffiliateSetting().RewardRateBps = 3500
	view, err := GetAffiliateUserOverrideView(agent.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(3500), view.EffectiveRule.RewardRateBps)
	domain := "sales"
	_, err = SaveAffiliateUserOverride(agent.Id, agent.Id, AffiliateUserOverride{AffiliateDomain: &domain})
	require.NoError(t, err)
	_, matched, err := ResolveAffiliateInviterByHost("sales.example.com")
	require.NoError(t, err)
	assert.True(t, matched)
	domain = ""
	view, err = SaveAffiliateUserOverride(agent.Id, agent.Id, AffiliateUserOverride{AffiliateDomain: &domain})
	require.NoError(t, err)
	require.NotNil(t, view.Override)
	assert.Nil(t, view.Override.AffiliateDomain)
	_, matched, err = ResolveAffiliateInviterByHost("sales.example.com")
	require.NoError(t, err)
	assert.False(t, matched)
	items, total, err := SearchAffiliateUserOverrides("", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, agent.Id, items[0].UserID)
}

func TestAffiliateAgentBatchBindingAndRollbackPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	configureAffiliateTest(t, nil)
	inviter, existingInvitee := createAffiliateUsers(t, "existing@example.com")
	other := &User{Username: "other-agent", AffCode: "OTHERAGENT", Status: common.UserStatusEnabled}
	free := &User{Username: "free-invitee", AffCode: "FREEINVITEE", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(other).Error)
	require.NoError(t, DB.Create(free).Error)
	remark := "must roll back"
	_, err := SaveAffiliateAgents(inviter.Id, []AffiliateAgentUpdate{
		{UserID: inviter.Id, Override: AffiliateUserOverride{Remark: &remark}, InviteeUserIDs: []int{free.Id}},
		{UserID: other.Id, InviteeUserIDs: []int{existingInvitee.Id}},
	})
	require.ErrorContains(t, err, "用户已绑定其他邀请者")
	require.NoError(t, DB.First(inviter, inviter.Id).Error)
	assert.Empty(t, inviter.Remark)
	require.NoError(t, DB.First(free, free.Id).Error)
	assert.Zero(t, free.InviterId)
	var count int64
	require.NoError(t, DB.Model(&AffiliateUserOverride{}).Count(&count).Error)
	assert.Zero(t, count)
	require.NoError(t, DB.Model(&AffiliateReferral{}).Where("invitee_user_id = ?", free.Id).Count(&count).Error)
	assert.Zero(t, count)

	updates := []AffiliateAgentUpdate{{UserID: other.Id, InviteeUserIDs: []int{free.Id}}}
	views, err := SaveAffiliateAgents(inviter.Id, updates)
	require.NoError(t, err)
	require.Len(t, views, 1)
	assert.Equal(t, int64(1), views[0].InviteInfo.InvitedCount)
	_, err = SaveAffiliateAgents(inviter.Id, updates)
	require.NoError(t, err)
	require.NoError(t, DB.First(other, other.Id).Error)
	assert.Equal(t, 1, other.AffCount)
	require.NoError(t, DB.First(free, free.Id).Error)
	assert.Equal(t, other.Id, free.InviterId)
	var referral AffiliateReferral
	require.NoError(t, DB.Where("invitee_user_id = ?", free.Id).First(&referral).Error)
	assert.Equal(t, other.Id, referral.InviterUserID)
	past := createAffiliateTopUp(t, free.Id, "before-binding", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, DB.Model(past).Update("complete_time", referral.BoundAt-1).Error)
	require.NoError(t, ProcessAffiliateTopUp(past.Id))
	require.NoError(t, DB.Model(&AffiliateReward{}).Count(&count).Error)
	assert.Zero(t, count)
	future := createAffiliateTopUp(t, free.Id, "after-binding", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, DB.Model(future).Update("complete_time", referral.BoundAt+1).Error)
	require.NoError(t, ProcessAffiliateTopUp(future.Id))
	var reward AffiliateReward
	require.NoError(t, DB.Where("top_up_id = ?", future.Id).First(&reward).Error)
	assert.Equal(t, other.Id, reward.RecipientUserID)
}
