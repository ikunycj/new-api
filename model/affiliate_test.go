package model

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func configureAffiliateTest(t *testing.T, mutate func(*operation_setting.AffiliateSetting)) int64 {
	t.Helper()
	affiliateSetting := operation_setting.GetAffiliateSetting()
	originalAffiliate := *affiliateSetting
	originalExchangeRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		*affiliateSetting = originalAffiliate
		operation_setting.USDExchangeRate = originalExchangeRate
	})
	operation_setting.USDExchangeRate = 1
	*affiliateSetting = operation_setting.AffiliateSetting{
		Enabled:                   true,
		RegistrationRewardTrigger: operation_setting.AffiliateRegistrationTriggerFirstQualifiedTopUp,
		RewardMode:                operation_setting.AffiliateRewardModePercentage,
		CashbackFrequency:         operation_setting.AffiliateCashbackFrequencyEveryTopUp,
		RewardRateBps:             2000,
		UnlimitedReward:           true,
		MinimumTopUpCents:         2000,
		MinimumTransferQuota:      int64(common.QuotaPerUnit),
		ShowInviteeTopUps:         true,
	}
	if mutate != nil {
		mutate(affiliateSetting)
	}
	return int64(common.QuotaPerUnit)
}

func createAffiliateUsers(t *testing.T, inviteeEmail string) (*User, *User) {
	t.Helper()
	inviter := &User{Username: "affiliate-inviter", AffCode: "INVITER1", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(inviter).Error)
	invitee := &User{
		Username: "affiliate-invitee", Email: inviteeEmail, AffCode: "INVITEE1",
		InviterId: inviter.Id, Status: common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(invitee).Error)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if err := initializeAffiliateUserWithTx(tx, inviter, 0, false); err != nil {
			return err
		}
		return initializeAffiliateUserWithTx(tx, invitee, inviter.Id, true)
	}))
	return inviter, invitee
}

func createAffiliateTopUp(t *testing.T, inviteeID int, tradeNo string, money float64, source string) *TopUp {
	t.Helper()
	now := time.Now().Unix()
	topUp := &TopUp{
		UserId: inviteeID, Amount: int64(money), Money: money, TradeNo: tradeNo,
		PaymentMethod: PaymentMethodStripe, PaymentProvider: PaymentProviderStripe,
		CompletionSource: source, Status: common.TopUpStatusSuccess,
		CreateTime: now, CompleteTime: now,
	}
	require.NoError(t, DB.Create(topUp).Error)
	return topUp
}

func createAffiliateUserOverride(t *testing.T, userID int, updatedBy int, override AffiliateUserOverride) *AffiliateUserOverride {
	t.Helper()
	override.ID = 0
	override.UserID = userID
	override.UpdatedBy = updatedBy
	require.NoError(t, DB.Create(&override).Error)
	return &override
}

func enableAffiliateCampaignForTest(t *testing.T, startsAt int64, endsAt int64, holdSeconds int64) *AffiliateCampaign {
	t.Helper()
	campaign, err := getAffiliateCampaignWithTx(DB)
	require.NoError(t, err)
	campaign.Name = "Test referral campaign"
	campaign.Enabled = true
	campaign.StartsAt = startsAt
	campaign.EndsAt = endsAt
	campaign.InviterCashbackRateBps = 2500
	campaign.InviteeBonusRateBps = 2000
	campaign.HoldSeconds = holdSeconds
	require.NoError(t, DB.Save(campaign).Error)
	return campaign
}

func TestAffiliateCampaignCreditsCashAndInviteeBonusOnce(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.Enabled = true
		setting.RewardRateBps = 500
	})
	now := time.Now().Unix()
	enableAffiliateCampaignForTest(t, now-60, now+3600, 0)
	inviter, invitee := createAffiliateUsers(t, "campaign@example.com")
	baseQuota := int64(100 * common.QuotaPerUnit)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", invitee.Id).Update("quota", baseQuota).Error)
	topUp := createAffiliateTopUp(t, invitee.Id, "campaign-one-hundred", 100, TopUpCompletionSourceOnlineWallet)
	topUp.PaidCents = 10_000
	topUp.CreditedQuota = baseQuota
	require.NoError(t, DB.Save(topUp).Error)

	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))

	var reward AffiliateCashReward
	require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).First(&reward).Error)
	assert.Equal(t, int64(2500), reward.CashbackCents)
	assert.Equal(t, AffiliateRewardStatusTransferred, reward.Status)

	var grant AffiliateQuotaGrant
	require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).First(&grant).Error)
	assert.Equal(t, baseQuota/5, grant.BonusQuota)
	var refreshedInvitee User
	require.NoError(t, DB.First(&refreshedInvitee, invitee.Id).Error)
	assert.Equal(t, int(baseQuota+baseQuota/5), refreshedInvitee.Quota)
	summary, err := GetAffiliateSummary(invitee.Id)
	require.NoError(t, err)
	assert.Equal(t, baseQuota/5, summary.LifetimeCampaignBonusQuota)

	var account AffiliateCashAccount
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
	assert.Zero(t, account.AvailableCents)
	assert.Equal(t, int64(2500), account.TransferredCents)
	assert.Equal(t, int64(2500), account.LifetimeEarnedCents)
	var legacyRewardCount int64
	require.NoError(t, DB.Model(&AffiliateReward{}).Where("top_up_id = ?", topUp.Id).Count(&legacyRewardCount).Error)
	assert.Zero(t, legacyRewardCount)
	items, total, err := GetAffiliateInviteeTopUpsSorted(inviter.Id, "", "", "recharge_time_desc", 0, 0, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Zero(t, items[0].AvailableRewardCents)
	assert.Equal(t, AffiliateRewardStatusTransferred, items[0].Status)
}

func TestAffiliateCampaignUsesStartInclusiveEndExclusiveWindow(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.Enabled = false
	})
	now := time.Now().Unix()
	enableAffiliateCampaignForTest(t, now, now+100, 0)
	_, invitee := createAffiliateUsers(t, "window@example.com")

	before := createAffiliateTopUp(t, invitee.Id, "campaign-before", 20, TopUpCompletionSourceOnlineWallet)
	before.CompleteTime = now - 1
	require.NoError(t, DB.Save(before).Error)
	atStart := createAffiliateTopUp(t, invitee.Id, "campaign-start", 20, TopUpCompletionSourceOnlineWallet)
	atStart.CompleteTime = now
	require.NoError(t, DB.Save(atStart).Error)
	atEnd := createAffiliateTopUp(t, invitee.Id, "campaign-end", 20, TopUpCompletionSourceOnlineWallet)
	atEnd.CompleteTime = now + 100
	require.NoError(t, DB.Save(atEnd).Error)

	require.NoError(t, ProcessAffiliateTopUp(before.Id))
	require.NoError(t, ProcessAffiliateTopUp(atStart.Id))
	require.NoError(t, ProcessAffiliateTopUp(atEnd.Id))
	var rewardCount int64
	require.NoError(t, DB.Model(&AffiliateCashReward{}).Count(&rewardCount).Error)
	assert.Equal(t, int64(1), rewardCount)
}

func TestAffiliateCampaignCashReleaseAndTransfer(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.Enabled = false
	})
	now := time.Now().Unix()
	enableAffiliateCampaignForTest(t, now-60, now+3600, 10)
	inviter, invitee := createAffiliateUsers(t, "transfer@example.com")
	topUp := createAffiliateTopUp(t, invitee.Id, "campaign-transfer", 100, TopUpCompletionSourceOnlineWallet)
	topUp.PaidCents = 10_000
	topUp.CreditedQuota = int64(100 * common.QuotaPerUnit)
	require.NoError(t, DB.Save(topUp).Error)
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))

	released, _, err := ReleaseDueAffiliateCashRewards(topUp.CompleteTime+10, 0, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, released)
	var transfer AffiliateCashTransfer
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&transfer).Error)
	assert.Equal(t, int64(2500), transfer.AmountCents)
	assert.Equal(t, int64(25*common.QuotaPerUnit), transfer.CreditedQuota)

	var account AffiliateCashAccount
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
	assert.Zero(t, account.AvailableCents)
	assert.Equal(t, int64(2500), account.TransferredCents)
}

func TestAffiliateEveryTopUpUsesCurrentPayment(t *testing.T) {
	truncateTables(t)
	unit := configureAffiliateTest(t, nil)
	inviter, invitee := createAffiliateUsers(t, "alice@example.com")

	first := createAffiliateTopUp(t, invitee.Id, "affiliate-first-ten", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(first.Id))
	second := createAffiliateTopUp(t, invitee.Id, "affiliate-second-ten", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(second.Id))
	require.NoError(t, ProcessAffiliateTopUp(second.Id))
	third := createAffiliateTopUp(t, invitee.Id, "affiliate-one-yuan", 1, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(third.Id))

	var rewards []AffiliateReward
	require.NoError(t, DB.Where("reward_type = ?", AffiliateRewardTypeCashback).Order("id asc").Find(&rewards).Error)
	require.Len(t, rewards, 3)
	assert.Equal(t, 2*unit, rewards[0].ActualQuota)
	assert.Equal(t, 2*unit, rewards[1].ActualQuota)
	assert.Equal(t, unit/5, rewards[2].ActualQuota)
	assert.Equal(t, int64(1000), rewards[0].CumulativePaidCents)
	assert.Equal(t, int64(1000), rewards[1].CumulativePaidCents)
	assert.Equal(t, int64(100), rewards[2].PaidCents)

	var relation AffiliateReferral
	require.NoError(t, DB.Where("invitee_user_id = ?", invitee.Id).First(&relation).Error)
	assert.Equal(t, AffiliateReferralStatusQualified, relation.Status)
	assert.Equal(t, first.Id, *relation.QualifyingTopUpID)
	assert.Equal(t, 4*unit+unit/5, relation.CashbackRewardedQuota)

	var account AffiliateQuotaAccount
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
	assert.Zero(t, account.AvailableQuota)
	assert.Equal(t, 4*unit+unit/5, account.TransferredQuota)

	var eventCount int64
	require.NoError(t, DB.Model(&AffiliateTopUpEvent{}).Count(&eventCount).Error)
	assert.Equal(t, int64(3), eventCount)
}

func TestAffiliateMaximumRewardCapAppliesToPercentageCashback(t *testing.T) {
	truncateTables(t)
	unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.RewardRateBps = 5000
		setting.MaximumRewardQuota = 8 * int64(common.QuotaPerUnit)
	})
	inviter, invitee := createAffiliateUsers(t, "")

	for index, money := range []float64{20, 1, 1} {
		topUp := createAffiliateTopUp(t, invitee.Id, "affiliate-fixed-"+string(rune('a'+index)), money, TopUpCompletionSourceOnlineWallet)
		require.NoError(t, ProcessAffiliateTopUp(topUp.Id))
	}

	var rewards []AffiliateReward
	require.NoError(t, DB.Where("reward_type = ?", AffiliateRewardTypeCashback).Order("id asc").Find(&rewards).Error)
	require.Len(t, rewards, 1)
	assert.Equal(t, 8*unit, rewards[0].ActualQuota)
	var eventCount int64
	require.NoError(t, DB.Model(&AffiliateTopUpEvent{}).Count(&eventCount).Error)
	assert.Equal(t, int64(3), eventCount)

	var account AffiliateQuotaAccount
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
	assert.Zero(t, account.AvailableQuota)
	assert.Equal(t, 8*unit, account.TransferredQuota)
}

func TestAffiliateRegistrationRewardsAreDisabledAndRewardsAutoTransfer(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.RewardRateBps = 1000
		setting.HoldSeconds = 0
	})
	inviter, invitee := createAffiliateUsers(t, "")

	var rewards []AffiliateReward
	require.NoError(t, DB.Find(&rewards).Error)
	assert.Empty(t, rewards)
	var refreshedInvitee User
	require.NoError(t, DB.Select("quota").Where("id = ?", invitee.Id).First(&refreshedInvitee).Error)
	assert.Zero(t, refreshedInvitee.Quota)

	topUp := createAffiliateTopUp(t, invitee.Id, "affiliate-registration-disabled", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))
	var reward AffiliateReward
	require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).First(&reward).Error)
	assert.Equal(t, AffiliateRewardStatusTransferred, reward.Status)
	assert.Equal(t, reward.ActualQuota, reward.TransferredQuota)
	var transfer AffiliateBalanceTransfer
	require.NoError(t, DB.Where("user_id = ? AND amount_quota = ?", inviter.Id, reward.ActualQuota).First(&transfer).Error)
}

func TestAffiliateRewardsAutoTransferAfterHold(t *testing.T) {
	truncateTables(t)
	unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.HoldSeconds = 60
		setting.RewardRateBps = 1000
	})
	inviter, invitee := createAffiliateUsers(t, "auto-transfer@example.com")
	topUp := createAffiliateTopUp(t, invitee.Id, "affiliate-auto-transfer", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))

	var reward AffiliateReward
	require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).First(&reward).Error)
	assert.Equal(t, AffiliateRewardStatusPending, reward.Status)

	released, _, err := ReleaseDueAffiliateRewards(reward.AvailableAt, 0, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, released)
	require.NoError(t, DB.First(&reward, reward.ID).Error)
	assert.Equal(t, AffiliateRewardStatusTransferred, reward.Status)
	assert.Equal(t, reward.ActualQuota, reward.TransferredQuota)

	var account AffiliateQuotaAccount
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
	assert.Zero(t, account.AvailableQuota)
	assert.Equal(t, reward.ActualQuota, account.TransferredQuota)
	var transfer AffiliateBalanceTransfer
	require.NoError(t, DB.Where("user_id = ? AND amount_quota = ?", inviter.Id, reward.ActualQuota).First(&transfer).Error)

	var refreshedInviter User
	require.NoError(t, DB.Select("quota").Where("id = ?", inviter.Id).First(&refreshedInviter).Error)
	assert.Equal(t, int(reward.ActualQuota), refreshedInviter.Quota)
	_ = unit
}

func TestAffiliateQuotaAccountMigratesLegacyBalancesOnce(t *testing.T) {
	truncateTables(t)
	unit := configureAffiliateTest(t, nil)
	user := &User{
		Username: "affiliate-legacy", AffCode: "LEGACY01", Status: common.UserStatusEnabled,
		AffQuota: int(2 * unit), AffHistoryQuota: int(5 * unit),
	}
	require.NoError(t, DB.Create(user).Error)

	account, err := GetAffiliateQuotaAccount(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 2*unit, account.AvailableQuota)
	assert.Equal(t, 3*unit, account.TransferredQuota)
	assert.Equal(t, 5*unit, account.LifetimeEarnedQuota)
	var ledger AffiliateQuotaLedger
	require.NoError(t, DB.Where("entry_type = ?", AffiliateLedgerTypeLegacyMigration).First(&ledger).Error)
	assert.Equal(t, 5*unit, ledger.AmountQuota)
	assert.Equal(t, 2*unit, ledger.AvailableDeltaQuota)
	assert.Equal(t, 3*unit, ledger.TransferredDeltaQuota)

	account, err = GetAffiliateQuotaAccount(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 2*unit, account.AvailableQuota)
	assert.Equal(t, 3*unit, account.TransferredQuota)
	assert.Equal(t, 5*unit, account.LifetimeEarnedQuota)

	var refreshed User
	require.NoError(t, DB.Select("aff_quota", "aff_history").Where("id = ?", user.Id).First(&refreshed).Error)
	assert.Zero(t, refreshed.AffQuota)
	assert.Equal(t, int(5*unit), refreshed.AffHistoryQuota)
}

func TestAffiliateProfilesGenerateDistinctInvitationCodes(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, nil)
	first := &User{Username: "affiliate-code-one", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(first).Error)
	require.NoError(t, EnsureAffiliateProfile(first.Id))

	second := &User{Username: "affiliate-code-two", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(second).Error)
	require.NoError(t, EnsureAffiliateProfile(second.Id))
	require.NoError(t, DB.Select("aff_code").Where("id = ?", first.Id).First(first).Error)
	require.NoError(t, DB.Select("aff_code").Where("id = ?", second.Id).First(second).Error)
	assert.NotEmpty(t, first.AffCode)
	assert.NotEmpty(t, second.AffCode)
	assert.NotEqual(t, first.AffCode, second.AffCode)
}

func TestAffiliateOverrideAffectsExistingReferralFutureTopUps(t *testing.T) {
	truncateTables(t)
	unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.RewardRateBps = 1000
		setting.MinimumTopUpCents = 0
	})
	inviter, invitee := createAffiliateUsers(t, "")
	first := createAffiliateTopUp(t, invitee.Id, "affiliate-global-rate", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(first.Id))

	rate := int64(5000)
	createAffiliateUserOverride(t, inviter.Id, inviter.Id, AffiliateUserOverride{
		RewardRateBps: &rate,
		ChangeReason:  "partner agreement",
	})
	second := createAffiliateTopUp(t, invitee.Id, "affiliate-override-rate", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(second.Id))

	var rewards []AffiliateReward
	require.NoError(t, DB.Where("reward_type = ?", AffiliateRewardTypeCashback).Order("id asc").Find(&rewards).Error)
	require.Len(t, rewards, 2)
	assert.Equal(t, unit, rewards[0].ActualQuota)
	assert.Equal(t, 5*unit, rewards[1].ActualQuota)
	assert.NotEqual(t, rewards[0].RuleVersionID, rewards[1].RuleVersionID)
}

func TestAffiliateUserOverrideSearchSupportsCustomizedAndUncustomizedUsers(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, nil)
	inviter, invitee := createAffiliateUsers(t, "")
	rate := int64(5000)
	createAffiliateUserOverride(t, inviter.Id, inviter.Id, AffiliateUserOverride{
		RewardRateBps: &rate,
		ChangeReason:  "partner agreement",
	})

	items, total, err := SearchAffiliateUserOverrides("", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, inviter.Id, items[0].UserID)
	require.NotNil(t, items[0].Override)
	require.NotNil(t, items[0].Override.RewardRateBps)
	assert.Equal(t, rate, *items[0].Override.RewardRateBps)
	assert.Equal(t, inviter.Username, items[0].UpdatedByUsername)

	items, total, err = SearchAffiliateUserOverrides(invitee.Username, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, invitee.Id, items[0].UserID)
	assert.Nil(t, items[0].Override)
	assert.Equal(t, "global", items[0].EffectiveRule.Source)

	items, total, err = SearchAffiliateUserOverrides("affiliate-", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, items, 2)
	assert.Equal(t, inviter.Id, items[0].UserID)
	assert.NotNil(t, items[0].Override)
	assert.Equal(t, invitee.Id, items[1].UserID)
	assert.Nil(t, items[1].Override)
}

func TestAffiliateUserOverrideSaveAndDelete(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, nil)
	inviter, _ := createAffiliateUsers(t, "")
	defaultView, err := SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{})
	require.NoError(t, err)
	require.NotNil(t, defaultView.Override)
	assert.Equal(t, "global", defaultView.EffectiveRule.Source)
	require.ErrorIs(t, DeleteAffiliateUserOverride(0), ErrAffiliateRuleInvalid)

	rate := int64(5000)
	view, err := SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		RewardRateBps: &rate,
	})
	require.NoError(t, err)
	require.NotNil(t, view.Override)
	require.NotNil(t, view.Override.RewardRateBps)
	assert.Equal(t, rate, *view.Override.RewardRateBps)
	assert.Empty(t, view.Override.ChangeReason)

	require.NoError(t, DeleteAffiliateUserOverride(inviter.Id))
	view, err = GetAffiliateUserOverrideView(inviter.Id)
	require.NoError(t, err)
	assert.Nil(t, view.Override)
	assert.Equal(t, "global", view.EffectiveRule.Source)
}

func TestAffiliateUserOverrideSaveRejectsPreferencesAndRules(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, nil)
	inviter, _ := createAffiliateUsers(t, "")
	rate := int64(5000)

	enabled := true
	registrationTrigger := operation_setting.AffiliateRegistrationTriggerRegistrationSuccess
	cashbackFrequency := operation_setting.AffiliateCashbackFrequencyEveryTopUp
	rewardMode := operation_setting.AffiliateRewardModeFixed
	unlimited := true
	minimumTransfer := int64(common.QuotaPerUnit)
	showInviteeTopUps := true
	tests := []struct {
		name   string
		mutate func(*AffiliateUserOverride)
	}{
		{name: "enabled", mutate: func(override *AffiliateUserOverride) { override.Enabled = &enabled }},
		{name: "registration reward trigger", mutate: func(override *AffiliateUserOverride) {
			override.RegistrationRewardTrigger = &registrationTrigger
		}},
		{name: "cashback frequency", mutate: func(override *AffiliateUserOverride) {
			override.CashbackFrequency = &cashbackFrequency
		}},
		{name: "reward mode", mutate: func(override *AffiliateUserOverride) {
			override.RewardMode = &rewardMode
		}},
		{name: "unlimited reward", mutate: func(override *AffiliateUserOverride) {
			override.UnlimitedReward = &unlimited
		}},
		{name: "minimum transfer", mutate: func(override *AffiliateUserOverride) {
			override.MinimumTransferQuota = &minimumTransfer
		}},
		{name: "show invitee top ups", mutate: func(override *AffiliateUserOverride) {
			override.ShowInviteeTopUps = &showInviteeTopUps
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			override := AffiliateUserOverride{RewardRateBps: &rate}
			test.mutate(&override)
			_, err := SaveAffiliateUserOverride(inviter.Id, inviter.Id, override)
			require.ErrorIs(t, err, ErrAffiliateRuleInvalid)
		})
	}

	fixedReward := int64(common.QuotaPerUnit)
	_, err := SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		FixedRewardQuota: &fixedReward,
	})
	require.ErrorIs(t, err, ErrAffiliateRuleInvalid)

	setting := operation_setting.GetAffiliateSetting()
	setting.RewardMode = operation_setting.AffiliateRewardModeFixed
	_, err = SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		RewardRateBps: &rate,
	})
	require.NoError(t, err)
}

func TestAffiliateAdminCompletedTopUpIsExcluded(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, nil)
	now := time.Now().Unix()
	enableAffiliateCampaignForTest(t, now-60, now+3600, 0)
	_, invitee := createAffiliateUsers(t, "")
	topUp := createAffiliateTopUp(t, invitee.Id, "affiliate-admin-topup", 100, TopUpCompletionSourceAdmin)
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))

	var count int64
	require.NoError(t, DB.Model(&AffiliateTopUpEvent{}).Count(&count).Error)
	assert.Zero(t, count)
	require.NoError(t, DB.Model(&AffiliateReward{}).Where("reward_type = ?", AffiliateRewardTypeCashback).Count(&count).Error)
	assert.Zero(t, count)
	require.NoError(t, DB.Model(&AffiliateCashReward{}).Count(&count).Error)
	assert.Zero(t, count)
	require.NoError(t, DB.Model(&AffiliateQuotaGrant{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestAffiliateInviteeTopUpsMasksEmail(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.MinimumTopUpCents = 0
	})
	inviter, invitee := createAffiliateUsers(t, "alice@example.com")
	topUp := createAffiliateTopUp(t, invitee.Id, "affiliate-visible-topup", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))

	items, total, err := GetAffiliateInviteeTopUps(inviter.Id, "", "", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "a***e@example.com", items[0].MaskedEmail)
	assert.Equal(t, int64(1000), items[0].PaidCents)

	items, total, err = GetAffiliateInviteeTopUps(inviter.Id, "", "a***e@example.com", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	items, total, err = GetAffiliateInviteeTopUps(inviter.Id, "", "alice@example.com", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "a***e@example.com", items[0].MaskedEmail)

	items, total, err = GetAffiliateInviteeTopUps(inviter.Id, "", "ALICE", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "a***e@example.com", items[0].MaskedEmail)

}

func TestAffiliateInviteeTopUpsSearchesRechargeTimeAndSorts(t *testing.T) {
	truncateTables(t)
	configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.MinimumTopUpCents = 0
	})
	inviter, invitee := createAffiliateUsers(t, "timeline@example.com")
	require.NoError(t, DB.Model(&AffiliateReferral{}).Where("invitee_user_id = ?", invitee.Id).Update("bound_at", time.Date(2026, time.July, 31, 0, 0, 0, 0, time.Local).Unix()).Error)
	first := createAffiliateTopUp(t, invitee.Id, "affiliate-timeline-first", 10, TopUpCompletionSourceOnlineWallet)
	first.CompleteTime = time.Date(2026, time.August, 1, 9, 30, 0, 0, time.Local).Unix()
	require.NoError(t, DB.Model(first).Update("complete_time", first.CompleteTime).Error)
	require.NoError(t, ProcessAffiliateTopUp(first.Id))
	second := createAffiliateTopUp(t, invitee.Id, "affiliate-timeline-second", 20, TopUpCompletionSourceOnlineWallet)
	second.CompleteTime = time.Date(2026, time.August, 2, 14, 20, 0, 0, time.Local).Unix()
	require.NoError(t, DB.Model(second).Update("complete_time", second.CompleteTime).Error)
	require.NoError(t, ProcessAffiliateTopUp(second.Id))

	items, total, err := GetAffiliateInviteeTopUpsSorted(inviter.Id, "", "2026-08-02", "recharge_time_desc", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, second.Id, items[0].TopUpID)

	items, total, err = GetAffiliateInviteeTopUpsSorted(inviter.Id, "", "2026-08-02 14:20", "recharge_time_desc", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, second.Id, items[0].TopUpID)

	items, total, err = GetAffiliateInviteeTopUpsSorted(inviter.Id, "", "", "recharge_time_asc", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, items, 2)
	assert.Equal(t, first.Id, items[0].TopUpID)
	assert.Equal(t, second.Id, items[1].TopUpID)
}

func TestAffiliateRewardBalanceTransferMovesOnlySelectedReward(t *testing.T) {
	truncateTables(t)
	unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.MinimumTopUpCents = 0
		setting.RewardMode = operation_setting.AffiliateRewardModeFixed
		setting.FixedRewardQuota = 2 * int64(common.QuotaPerUnit)
	})
	inviter, invitee := createAffiliateUsers(t, "transfer-row@example.com")
	first := createAffiliateTopUp(t, invitee.Id, "affiliate-row-first", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(first.Id))
	second := createAffiliateTopUp(t, invitee.Id, "affiliate-row-second", 10, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(second.Id))

	var rewards []AffiliateReward
	require.NoError(t, DB.Where("reward_type = ?", AffiliateRewardTypeCashback).Order("id asc").Find(&rewards).Error)
	require.Len(t, rewards, 2)
	var firstReward, secondReward AffiliateReward
	require.NoError(t, DB.First(&firstReward, rewards[0].ID).Error)
	require.NoError(t, DB.First(&secondReward, rewards[1].ID).Error)
	assert.Equal(t, AffiliateRewardStatusTransferred, firstReward.Status)
	assert.Equal(t, AffiliateRewardStatusTransferred, secondReward.Status)
	assert.Equal(t, 2*unit, firstReward.TransferredQuota)
	assert.Equal(t, 2*unit, secondReward.TransferredQuota)

	_, err := CreateAffiliateRewardBalanceTransfer(inviter.Id, rewards[0].ID, "row-transfer-again")
	assert.ErrorIs(t, err, ErrAffiliateBalance)
}

func TestAffiliateAdjustmentLeavesTransferredAmountForManualHandling(t *testing.T) {
	truncateTables(t)
	unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.RewardRateBps = 10000
		setting.MinimumTopUpCents = 0
	})
	inviter, invitee := createAffiliateUsers(t, "adjusted@example.com")
	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).Update("remark", "East VIP agent").Error)
	topUp := createAffiliateTopUp(t, invitee.Id, "affiliate-adjustment", 5, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))
	var reward AffiliateReward
	require.NoError(t, DB.Where("reward_type = ?", AffiliateRewardTypeCashback).First(&reward).Error)
	adjustment, err := AdjustAffiliateReward(reward.ID, inviter.Id, 5*unit, "refunded payment", "refund-adjustment-1")
	require.NoError(t, err)
	assert.Zero(t, adjustment.AppliedQuota)
	assert.Equal(t, 5*unit, adjustment.PendingManualQuota)
	assert.Equal(t, AffiliateAdjustmentStatusManualRequired, adjustment.Status)

	repeated, err := AdjustAffiliateReward(reward.ID, inviter.Id, 5*unit, "refunded payment", "refund-adjustment-1")
	require.NoError(t, err)
	assert.Equal(t, adjustment.ID, repeated.ID)

	items, total, err := GetAffiliateAdminRewards("affiliate-invitee", "", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, reward.ID, items[0].ID)
	assert.Zero(t, items[0].AdjustedQuota)
	assert.Equal(t, "adjusted@example.com", items[0].InviteeEmail)

	items, total, err = GetAffiliateAdminRewards("east vip", "", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, items, 1)

	items, total, err = GetAffiliateAdminRewards("affiliate-adjustment", "", 0, 0, 0, 10)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, items)

	var account AffiliateQuotaAccount
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
	assert.Zero(t, account.AvailableQuota)
	assert.Equal(t, 5*unit, account.TransferredQuota)
}

func setupPostgresAffiliateTestDB(t *testing.T) {
	t.Helper()
	setupPostgresAnalyticsTestDB(t,
		&User{}, &TopUp{}, &AffiliateRuleVersion{}, &AffiliateUserOverride{},
		&AffiliateReferral{}, &AffiliateReward{}, &AffiliateQuotaAccount{},
		&AffiliateQuotaLedger{}, &AffiliateBalanceTransfer{}, &AffiliateAdjustment{},
		&AffiliateTopUpEvent{}, &AffiliateCampaign{}, &AffiliateCashReward{},
		&AffiliateCashAccount{}, &AffiliateCashLedger{}, &AffiliateCashTransfer{},
		&AffiliateQuotaGrant{},
	)
}

func TestAffiliateAdjustmentIncreasePostgres(t *testing.T) {
	for _, test := range []struct {
		name         string
		holdSeconds  int64
		fullyDeduct  bool
		wantNetUnits int64
	}{
		{name: "frozen", holdSeconds: 3600, wantNetUnits: 3},
		{name: "already transferred", wantNetUnits: 3},
		{name: "fully deducted frozen reward", holdSeconds: 3600, fullyDeduct: true, wantNetUnits: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupPostgresAffiliateTestDB(t)
			unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
				setting.HoldSeconds = test.holdSeconds
			})
			inviter, invitee := createAffiliateUsers(t, "cashback@example.com")
			topUp := createAffiliateTopUp(t, invitee.Id, "increase-order", 5, TopUpCompletionSourceOnlineWallet)
			require.NoError(t, ProcessAffiliateTopUp(topUp.Id))
			var reward AffiliateReward
			require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).First(&reward).Error)
			require.Equal(t, unit, reward.ActualQuota)
			if test.fullyDeduct {
				_, err := AdjustAffiliateReward(reward.ID, inviter.Id, unit, "refund", "deduct-all")
				require.NoError(t, err)
			}

			adjustment, err := AdjustAffiliateReward(reward.ID, inviter.Id, -2*unit, "additional cashback", "increase-1")
			require.NoError(t, err)
			assert.Equal(t, -2*unit, adjustment.RequestedQuota)
			assert.Equal(t, -2*unit, adjustment.AppliedQuota)
			assert.Zero(t, adjustment.PendingManualQuota)
			assert.Equal(t, AffiliateAdjustmentStatusApplied, adjustment.Status)
			repeated, err := AdjustAffiliateReward(reward.ID, inviter.Id, -2*unit, "additional cashback", "increase-1")
			require.NoError(t, err)
			assert.Equal(t, adjustment.ID, repeated.ID)

			var account AffiliateQuotaAccount
			require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
			require.NoError(t, DB.First(&reward, reward.ID).Error)
			assert.Equal(t, unit, reward.ActualQuota)
			assert.Equal(t, test.wantNetUnits*unit, reward.ActualQuota-reward.AdjustedQuota)
			assert.Equal(t, 3*unit, account.LifetimeEarnedQuota)
			var ledger AffiliateQuotaLedger
			require.NoError(t, DB.Where("reference_type = ? AND reference_id = ?", AffiliateReferenceAdjustment, adjustment.ID).First(&ledger).Error)
			assert.Equal(t, 2*unit, ledger.AmountQuota)
			if test.holdSeconds > 0 {
				assert.Equal(t, AffiliateRewardStatusPending, reward.Status)
				assert.Equal(t, test.wantNetUnits*unit, account.PendingQuota)
				assert.Equal(t, 2*unit, ledger.PendingDeltaQuota)
				released, _, err := ReleaseDueAffiliateRewards(reward.AvailableAt, 0, 20)
				require.NoError(t, err)
				assert.Equal(t, 1, released)
			} else {
				assert.Equal(t, AffiliateRewardStatusTransferred, reward.Status)
				assert.Equal(t, 2*unit, ledger.AvailableDeltaQuota)
				_, err := AdjustAffiliateReward(reward.ID, inviter.Id, -unit, "second increase", "increase-2")
				require.NoError(t, err)
			}

			require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
			require.NoError(t, DB.First(&reward, reward.ID).Error)
			var user User
			require.NoError(t, DB.First(&user, inviter.Id).Error)
			assert.Zero(t, account.PendingQuota)
			assert.Zero(t, account.AvailableQuota)
			assert.Equal(t, reward.ActualQuota-reward.AdjustedQuota, reward.TransferredQuota)
			assert.Equal(t, reward.TransferredQuota, account.TransferredQuota)
			assert.Equal(t, reward.TransferredQuota, int64(user.Quota))
		})
	}
}

func TestAffiliateAdjustmentBoundsAndRollbackPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	unit := configureAffiliateTest(t, nil)
	inviter, invitee := createAffiliateUsers(t, "bounds@example.com")
	topUp := createAffiliateTopUp(t, invitee.Id, "bounds-order", 5, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))
	var reward AffiliateReward
	require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).First(&reward).Error)
	// Preserve the transfer format written by the previous implementation.
	require.NoError(t, DB.Model(&AffiliateBalanceTransfer{}).Where("user_id = ?", inviter.Id).
		Update("request_key", "auto-reward:"+strconv.Itoa(reward.ID)).Error)
	for _, amount := range []int64{0, math.MinInt64, int64(common.MaxQuota) + 1, -int64(common.MaxQuota) - 1, -int64(common.MaxQuota)} {
		_, err := AdjustAffiliateReward(reward.ID, inviter.Id, amount, "invalid amount", "invalid")
		assert.ErrorIs(t, err, ErrAffiliateAmountInvalid)
	}
	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).Update("quota", common.MaxQuota).Error)
	adjustment, err := AdjustAffiliateReward(reward.ID, inviter.Id, -unit, "wallet full; keep cashback available", "overflow")
	require.NoError(t, err)
	assert.Equal(t, -unit, adjustment.AppliedQuota)
	require.NoError(t, DB.First(&reward, reward.ID).Error)
	assert.Equal(t, -unit, reward.AdjustedQuota)
	assert.Equal(t, unit, reward.TransferredQuota)
	assert.Equal(t, AffiliateRewardStatusAvailable, reward.Status)
	var account AffiliateQuotaAccount
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
	assert.Equal(t, 2*unit, account.LifetimeEarnedQuota)
	assert.Equal(t, unit, account.AvailableQuota)
	var user User
	require.NoError(t, DB.First(&user, inviter.Id).Error)
	assert.Equal(t, common.MaxQuota, user.Quota)

	// Existing deployments already have an automatic transfer for this reward.
	// Once wallet space is restored, that record must not block the increase.
	require.NoError(t, DB.Model(inviter).Update("quota", common.MaxQuota-int(unit)).Error)
	_, _, err = ReleaseDueAffiliateRewards(common.GetTimestamp(), 0, 20)
	require.NoError(t, err)
	require.NoError(t, DB.First(&user, inviter.Id).Error)
	assert.Equal(t, common.MaxQuota, user.Quota)
	require.NoError(t, DB.First(&reward, reward.ID).Error)
	assert.Equal(t, AffiliateRewardStatusTransferred, reward.Status)
	assert.Equal(t, 2*unit, reward.TransferredQuota)
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
	assert.Zero(t, account.AvailableQuota)
	assert.Equal(t, 2*unit, account.TransferredQuota)

	// Repeating both the adjustment request and settlement cannot credit twice.
	repeated, err := AdjustAffiliateReward(reward.ID, inviter.Id, -unit, "wallet full; keep cashback available", "overflow")
	require.NoError(t, err)
	assert.Equal(t, adjustment.ID, repeated.ID)
	_, _, err = ReleaseDueAffiliateRewards(common.GetTimestamp(), 0, 20)
	require.NoError(t, err)
	require.NoError(t, DB.First(&user, inviter.Id).Error)
	assert.Equal(t, common.MaxQuota, user.Quota)
	var transfers int64
	require.NoError(t, DB.Model(&AffiliateBalanceTransfer{}).Where("user_id = ?", inviter.Id).Count(&transfers).Error)
	assert.Equal(t, int64(2), transfers)
}

func TestAffiliateAdjustmentDecreaseFrozenPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.HoldSeconds = 3600
	})
	inviter, invitee := createAffiliateUsers(t, "deduct@example.com")
	topUp := createAffiliateTopUp(t, invitee.Id, "deduct-order", 5, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))
	var reward AffiliateReward
	require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).First(&reward).Error)
	adjustment, err := AdjustAffiliateReward(reward.ID, inviter.Id, unit/2, "partial refund", "deduct-half")
	require.NoError(t, err)
	assert.Equal(t, unit/2, adjustment.AppliedQuota)
	assert.Zero(t, adjustment.PendingManualQuota)
	var account AffiliateQuotaAccount
	require.NoError(t, DB.Where("user_id = ?", inviter.Id).First(&account).Error)
	assert.Equal(t, unit/2, account.PendingQuota)
	released, _, err := ReleaseDueAffiliateRewards(reward.AvailableAt, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, released)
	var user User
	require.NoError(t, DB.First(&user, inviter.Id).Error)
	assert.Equal(t, unit/2, int64(user.Quota))
}

func TestAffiliateAdminRewardsSearchAndEmailPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	configureAffiliateTest(t, nil)
	inviter, invitee := createAffiliateUsers(t, "bob@customer.example")
	require.NoError(t, DB.Model(inviter).Updates(map[string]interface{}{
		"username": "AliceAgent", "email": "alice@sales.example", "remark": "华东 VIP_!客户",
	}).Error)
	require.NoError(t, DB.Model(invitee).Update("username", "BobCustomer").Error)
	topUp := createAffiliateTopUp(t, invitee.Id, "order-only-742", 5, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(topUp.Id))
	for _, keyword := range []string{
		"lice", "BOBC", "SALES.EXAMPLE", "customer.ex", "华东", "vip_!", strconv.Itoa(inviter.Id), strconv.Itoa(invitee.Id),
	} {
		t.Run(keyword, func(t *testing.T) {
			items, total, err := GetAffiliateAdminRewards(keyword, "", 0, 0, 0, 10)
			require.NoError(t, err)
			assert.Equal(t, int64(1), total)
			require.Len(t, items, 1)
			assert.Equal(t, "bob@customer.example", items[0].InviteeEmail)
		})
	}
	for _, keyword := range []string{"order-only", inviter.AffCode, "vipX!"} {
		items, total, err := GetAffiliateAdminRewards(keyword, "", 0, 0, 0, 10)
		require.NoError(t, err)
		assert.Zero(t, total)
		assert.Empty(t, items)
	}
}

func TestAffiliateGlobalRuleUsesCurrentPercentageCashbackContract(t *testing.T) {
	configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.InviterRewardQuota = 3 * int64(common.QuotaPerUnit)
		setting.InviteeRewardQuota = 2 * int64(common.QuotaPerUnit)
		setting.RegistrationRewardTrigger = operation_setting.AffiliateRegistrationTriggerRegistrationSuccess
		setting.RewardMode = operation_setting.AffiliateRewardModeFixed
		setting.FixedRewardQuota = 4 * int64(common.QuotaPerUnit)
		setting.CashbackFrequency = operation_setting.AffiliateCashbackFrequencyFirstQualified
		setting.MinimumTopUpCents = 1000
		setting.MinimumTransferQuota = 500 * int64(common.QuotaPerUnit)
		setting.MaximumRewardQuota = 0
	})

	rule := globalAffiliateRule()
	assert.Zero(t, rule.InviterRewardQuota)
	assert.Zero(t, rule.InviteeRewardQuota)
	assert.Equal(t, operation_setting.AffiliateRegistrationTriggerDisabled, rule.RegistrationRewardTrigger)
	assert.Equal(t, operation_setting.AffiliateRewardModePercentage, rule.RewardMode)
	assert.Equal(t, operation_setting.AffiliateCashbackFrequencyEveryTopUp, rule.CashbackFrequency)
	assert.Zero(t, rule.FixedRewardQuota)
	assert.Zero(t, rule.MinimumTopUpCents)
	assert.Zero(t, rule.MinimumTransferQuota)
	assert.True(t, rule.UnlimitedReward)
}
