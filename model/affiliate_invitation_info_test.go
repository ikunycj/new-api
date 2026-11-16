package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAffiliateAdminReferralActivitiesIncludeInviteAndTopUps(t *testing.T) {
	truncateTables(t)
	previousExchangeRate := operation_setting.USDExchangeRate
	operation_setting.USDExchangeRate = 1
	t.Cleanup(func() { operation_setting.USDExchangeRate = previousExchangeRate })
	unit := int64(common.QuotaPerUnit)
	inviter := &User{Username: "activity-inviter", Email: "inviter@example.com", AffCode: "ACTIVITYINVITER", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(inviter).Error)
	invitee := &User{
		Username: "activity-invitee", Email: "invitee@example.com", AffCode: "ACTIVITYINVITEE", InviterId: inviter.Id,
		Status: common.UserStatusEnabled, CreatedAt: 100,
	}
	require.NoError(t, DB.Create(invitee).Error)
	referral := &AffiliateReferral{
		InviteeUserID: invitee.Id, InviterUserID: inviter.Id, CodeSnapshot: "ACTIVITY",
		Status: AffiliateReferralStatusQualified, BoundAt: 100,
	}
	require.NoError(t, DB.Create(referral).Error)

	firstTopUp := &TopUp{
		UserId: invitee.Id, Money: 10, PaidCents: 1000, TradeNo: "activity-topup-1",
		Status: common.TopUpStatusSuccess, CompletionSource: TopUpCompletionSourceOnlineWallet,
		CreateTime: 190, CompleteTime: 200,
	}
	secondTopUp := &TopUp{
		UserId: invitee.Id, Money: 20, PaidCents: 2000, TradeNo: "activity-topup-2",
		Status: common.TopUpStatusSuccess, CompletionSource: TopUpCompletionSourceOnlineWallet,
		CreateTime: 290, CompleteTime: 300,
	}
	preBindingTopUp := &TopUp{
		UserId: invitee.Id, Money: 5, PaidCents: 500, TradeNo: "activity-pre-binding-topup",
		Status: common.TopUpStatusSuccess, CompletionSource: TopUpCompletionSourceOnlineWallet,
		CreateTime: 90,
	}
	adminTopUp := &TopUp{
		UserId: invitee.Id, Money: 30, PaidCents: 3000, TradeNo: "activity-admin-topup",
		Status: common.TopUpStatusSuccess, CompletionSource: TopUpCompletionSourceAdmin,
		CreateTime: 390, CompleteTime: 400,
	}
	require.NoError(t, DB.Create(firstTopUp).Error)
	require.NoError(t, DB.Create(secondTopUp).Error)
	require.NoError(t, DB.Create(preBindingTopUp).Error)
	require.NoError(t, DB.Create(adminTopUp).Error)
	quotaReward := &AffiliateReward{
		ReferralID: referral.ID, InviterUserID: inviter.Id, InviteeUserID: invitee.Id,
		RecipientUserID: inviter.Id, RewardType: AffiliateRewardTypeCashback,
		TopUpID: &firstTopUp.Id, QuotaPerUnitSnapshot: common.QuotaPerUnit,
		CNYExchangeRateSnapshot: 1, ActualQuota: 2 * unit,
		IdempotencyKey: "activity-quota-reward", Status: AffiliateRewardStatusTransferred,
	}
	require.NoError(t, DB.Create(quotaReward).Error)
	cashReward := &AffiliateCashReward{
		CampaignID: 1, ReferralID: referral.ID, InviterUserID: inviter.Id,
		InviteeUserID: invitee.Id, TopUpID: secondTopUp.Id, CashbackCents: 300,
		Status: AffiliateRewardStatusTransferred, IdempotencyKey: "activity-cash-reward",
	}
	require.NoError(t, DB.Create(cashReward).Error)
	firstRewardID, cashRewardID := quotaReward.ID, cashReward.ID
	require.NoError(t, DB.Create(&AffiliateTopUpEvent{
		ReferralID: referral.ID, TopUpID: firstTopUp.Id, InviterUserID: inviter.Id,
		InviteeUserID: invitee.Id, PaidCents: 1000, CompletedAt: 200,
		RewardID: &firstRewardID,
	}).Error)
	require.NoError(t, DB.Create(&AffiliateTopUpEvent{
		ReferralID: referral.ID, TopUpID: secondTopUp.Id, InviterUserID: inviter.Id,
		InviteeUserID: invitee.Id, PaidCents: 2000, CompletedAt: 300,
		CashRewardID: &cashRewardID,
	}).Error)
	require.NoError(t, DB.Create(&AffiliateQuotaAccount{
		UserID: inviter.Id, LifetimeEarnedQuota: 2 * unit,
	}).Error)
	require.NoError(t, DB.Create(&AffiliateCashAccount{
		UserID: inviter.Id, LifetimeEarnedCents: 300,
	}).Error)

	items, total, err := GetAffiliateAdminReferralActivities(inviter.Id, "activity-invitee", "", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, items, 3)
	assert.Equal(t, AffiliateActivityTopUp, items[0].OperationType)
	assert.Equal(t, int64(2000), items[0].TopUpAmountCents)
	assert.Equal(t, int64(300), items[0].CashbackAmountCents)
	assert.Equal(t, int64(500), items[0].CumulativeCashbackCents)
	assert.Equal(t, int64(1000), items[1].TopUpAmountCents)
	assert.Equal(t, int64(200), items[1].CashbackAmountCents)
	assert.Equal(t, int64(200), items[1].CumulativeCashbackCents)
	assert.Equal(t, AffiliateActivityInvitation, items[2].OperationType)
	assert.Zero(t, items[2].TopUpAmountCents)
	assert.Zero(t, items[2].CashbackAmountCents)
	assert.Equal(t, int64(500), items[2].CumulativeCashbackCents)

	topUps, total, err := GetAffiliateAdminReferralActivities(inviter.Id, "", AffiliateActivityTopUp, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, topUps, 2)

	infos, err := GetAffiliateUserInviteInfos([]int{inviter.Id, invitee.Id})
	require.NoError(t, err)
	require.Len(t, infos, 2)
	byUserID := make(map[int]AffiliateUserInviteInfo, len(infos))
	for _, info := range infos {
		byUserID[info.UserID] = info
	}
	assert.Equal(t, int64(1), byUserID[inviter.Id].InvitedCount)
	assert.Equal(t, int64(500), byUserID[inviter.Id].IncomeCents)
	assert.Equal(t, inviter.Id, byUserID[invitee.Id].InviterUserID)
	assert.Equal(t, inviter.Email, byUserID[invitee.Id].InviterEmail)
}

func TestAffiliateAdminReferralActivitiesPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	inviter := &User{
		Username: "postgres-activity-inviter", Email: "postgres-inviter@example.com",
		AffCode: "POSTGRESACTIVITY", Status: common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(inviter).Error)
	invitee := &User{
		Username: "postgres-activity-invitee", Email: "postgres-invitee@example.com",
		AffCode: "POSTGRESINVITEE", InviterId: inviter.Id, Status: common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(invitee).Error)
	referral := &AffiliateReferral{
		InviteeUserID: invitee.Id, InviterUserID: inviter.Id, CodeSnapshot: "POSTGRES",
		Status: AffiliateReferralStatusBound, BoundAt: 100,
	}
	require.NoError(t, DB.Create(referral).Error)
	topUp := &TopUp{
		UserId: invitee.Id, Money: 12.34, PaidCents: 1234, TradeNo: "postgres-activity-topup",
		Status: common.TopUpStatusSuccess, CompletionSource: TopUpCompletionSourceOnlineWallet,
		CreateTime: 190, CompleteTime: 200,
	}
	require.NoError(t, DB.Create(topUp).Error)
	require.NoError(t, DB.Create(&AffiliateTopUpEvent{
		ReferralID: referral.ID, TopUpID: topUp.Id, InviterUserID: inviter.Id,
		InviteeUserID: invitee.Id, PaidCents: 1234, CompletedAt: 200,
	}).Error)

	items, total, err := GetAffiliateAdminReferralActivities(inviter.Id, "postgres-activity-invitee", "", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, items, 2)
	assert.Equal(t, AffiliateActivityTopUp, items[0].OperationType)
	assert.Equal(t, int64(1234), items[0].TopUpAmountCents)
	assert.Equal(t, AffiliateActivityInvitation, items[1].OperationType)

	infos, err := GetAffiliateUserInviteInfos([]int{inviter.Id, invitee.Id})
	require.NoError(t, err)
	require.Len(t, infos, 2)
}
