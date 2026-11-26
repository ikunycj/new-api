package model

import (
	"math"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAffiliateSubAgentSettlementPostgres(t *testing.T) {
	for _, test := range []struct {
		name     string
		campaign bool
	}{
		{name: "regular cashback"},
		{name: "campaign cashback", campaign: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupPostgresAffiliateTestDB(t)
			const holdSeconds = 60
			const rateBps int64 = 500
			unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
				setting.HoldSeconds = holdSeconds
				setting.RewardRateBps = rateBps
			})
			now := common.GetTimestamp()
			if test.campaign {
				campaign := enableAffiliateCampaignForTest(t, now-60, now+3600, holdSeconds)
				require.NoError(t, DB.Model(campaign).Update("inviter_cashback_rate_bps", rateBps).Error)
				// Existing installations can have either a standalone index or a unique constraint.
				require.NoError(t, DB.Exec("CREATE UNIQUE INDEX idx_affiliate_cash_rewards_top_up_id ON affiliate_cash_rewards (top_up_id)").Error)
				require.NoError(t, migrateAffiliateCashRewardTopUpIndex())
				require.NoError(t, DB.Exec("ALTER TABLE affiliate_cash_rewards ADD CONSTRAINT idx_affiliate_cash_rewards_top_up_id UNIQUE (top_up_id)").Error)
				require.NoError(t, migrateAffiliateCashRewardTopUpIndex())
			}
			parent := &User{Username: "upper-affiliate-agent", AffCode: "UPPERAGENT", Status: common.UserStatusEnabled}
			require.NoError(t, DB.Create(parent).Error)
			child, buyer := createAffiliateUsers(t, "sub-agent-buyer@example.com")
			child.Email = "sub-agent@example.com"
			require.NoError(t, DB.Model(child).Update("email", child.Email).Error)
			association := &AffiliateSubAgent{ChildUserID: child.Id, ParentUserID: parent.Id, RateBps: rateBps, EffectiveAt: now - 60}
			require.NoError(t, DB.Create(association).Error)
			topUp := &TopUp{
				UserId: buyer.Id, TradeNo: "sub-agent-cashback-order", Amount: 100, Money: 100,
				PaymentProvider: PaymentProviderEpay, Status: common.TopUpStatusPending, CreateTime: now,
			}
			require.NoError(t, DB.Create(topUp).Error)
			_, _, completed, err := CompleteEpayTopUp(topUp.TradeNo, "alipay")
			require.NoError(t, err)
			assert.True(t, completed)
			_, _, completed, err = CompleteEpayTopUp(topUp.TradeNo, "alipay")
			require.NoError(t, err)
			assert.False(t, completed)

			var event AffiliateTopUpEvent
			require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).Take(&event).Error)
			var childQuotaRewardID, childCashRewardID, parentQuotaRewardID, parentCashRewardID int
			var dueAt, wantParentQuota, wantParentCents int64
			if test.campaign {
				require.NotNil(t, event.CashRewardID)
				childCashRewardID = *event.CashRewardID
				var rewards []AffiliateCashReward
				require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).Order("id").Find(&rewards).Error)
				require.Len(t, rewards, 2)
				assert.Equal(t, childCashRewardID, rewards[0].ID)
				assert.Equal(t, child.Id, rewards[0].InviterUserID)
				assert.EqualValues(t, 500, rewards[0].CashbackCents)
				assert.Nil(t, rewards[0].SourceRewardID)
				require.NotNil(t, rewards[1].SourceRewardID)
				assert.Equal(t, childCashRewardID, *rewards[1].SourceRewardID)
				assert.Equal(t, parent.Id, rewards[1].InviterUserID)
				assert.EqualValues(t, 500, rewards[1].CashbackCents)
				assert.Equal(t, rateBps, rewards[1].SourceRewardRateBps)
				assert.Equal(t, rewards[0].AvailableAt, rewards[1].AvailableAt)
				parentCashRewardID, dueAt = rewards[1].ID, rewards[1].AvailableAt
				wantParentCents = 500
				duplicate := rewards[0]
				duplicate.ID = 0
				duplicate.IdempotencyKey = "duplicate-direct-campaign-reward"
				err := DB.Transaction(func(tx *gorm.DB) error { return tx.Create(&duplicate).Error })
				var dbError *pgconn.PgError
				require.ErrorAs(t, err, &dbError)
				assert.Equal(t, "23505", dbError.Code)
			} else {
				require.NotNil(t, event.RewardID)
				childQuotaRewardID = *event.RewardID
				var rewards []AffiliateReward
				require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).Order("id").Find(&rewards).Error)
				require.Len(t, rewards, 2)
				assert.Equal(t, childQuotaRewardID, rewards[0].ID)
				assert.Equal(t, child.Id, rewards[0].RecipientUserID)
				assert.Equal(t, 5*unit, rewards[0].ActualQuota)
				assert.Nil(t, rewards[0].SourceRewardID)
				require.NotNil(t, rewards[1].SourceRewardID)
				assert.Equal(t, childQuotaRewardID, *rewards[1].SourceRewardID)
				assert.Equal(t, parent.Id, rewards[1].RecipientUserID)
				assert.Equal(t, child.Id, rewards[1].InviterUserID)
				assert.Equal(t, 5*unit, rewards[1].ActualQuota)
				assert.Equal(t, rateBps, rewards[1].SourceRewardRateBps)
				assert.Equal(t, rewards[0].AvailableAt, rewards[1].AvailableAt)
				parentQuotaRewardID, dueAt = rewards[1].ID, rewards[1].AvailableAt
				wantParentQuota = 5 * unit

				adminRewards, total, err := GetAffiliateAdminRewards(strconv.Itoa(parent.Id), "", 0, 0, 0, 10)
				require.NoError(t, err)
				assert.EqualValues(t, 1, total)
				require.Len(t, adminRewards, 1)
				assert.Equal(t, parent.Id, adminRewards[0].InviterUserID)
				assert.Equal(t, child.Id, adminRewards[0].SourceAgentUserID)
				assert.Equal(t, child.Username, adminRewards[0].SourceAgentUsername)
				_, err = CreateAffiliateRewardBalanceTransfer(child.Id, parentQuotaRewardID, "forbidden-child-transfer")
				assert.ErrorIs(t, err, ErrAffiliateRewardNotFound)
			}

			parentTopUps, total, err := GetAffiliateInviteeTopUps(parent.Id, AffiliateRewardStatusPending, child.Email, 0, 0, 0, 10)
			require.NoError(t, err)
			assert.EqualValues(t, 1, total)
			require.Len(t, parentTopUps, 1)
			parentTopUp := parentTopUps[0]
			assert.Equal(t, parentQuotaRewardID, parentTopUp.RewardID)
			assert.Equal(t, parentCashRewardID, parentTopUp.CashRewardID)
			assert.Equal(t, child.Id, parentTopUp.SourceAgentUserID)
			assert.Equal(t, child.Username, parentTopUp.SourceAgentUsername)
			assert.Equal(t, child.Email, parentTopUp.SourceAgentEmail)
			assert.Equal(t, rateBps, parentTopUp.RewardRateBps)
			assert.False(t, parentTopUp.LegacySubAgentRate)
			assert.Equal(t, wantParentQuota, parentTopUp.RewardQuota)
			assert.Equal(t, wantParentCents, parentTopUp.RewardCents)
			assert.Equal(t, AffiliateRewardStatusPending, parentTopUp.Status)
			assert.Equal(t, dueAt, parentTopUp.AvailableAt)
			assert.Zero(t, parentTopUp.TopUpID)
			assert.Equal(t, event.CompletedAt, parentTopUp.TopUpAt)
			assert.EqualValues(t, 10_000, parentTopUp.PaidCents)
			assert.Empty(t, parentTopUp.InvitationCode)

			childTopUps, total, err := GetAffiliateInviteeTopUps(child.Id, "", buyer.Email, 0, 0, 0, 10)
			require.NoError(t, err)
			assert.EqualValues(t, 1, total)
			require.Len(t, childTopUps, 1)
			childTopUp := childTopUps[0]
			assert.Equal(t, childQuotaRewardID, childTopUp.RewardID)
			assert.Equal(t, childCashRewardID, childTopUp.CashRewardID)
			assert.Zero(t, childTopUp.SourceAgentUserID)
			assert.Equal(t, child.AffCode, childTopUp.InvitationCode)
			assert.Equal(t, "s***r@example.com", parentTopUp.MaskedEmail)
			assert.Equal(t, childTopUp.MaskedEmail, parentTopUp.MaskedEmail)
			assert.Positive(t, parentTopUp.InvitedAt)
			assert.Equal(t, childTopUp.InvitedAt, parentTopUp.InvitedAt)

			parentTopUps, total, err = GetAffiliateInviteeTopUps(parent.Id, "", parentTopUp.MaskedEmail, 0, 0, 0, 10)
			require.NoError(t, err)
			assert.EqualValues(t, 1, total)
			assert.Equal(t, []AffiliateInviteeTopUp{parentTopUp}, parentTopUps)
			for _, keyword := range []string{buyer.Email, child.AffCode} {
				parentTopUps, total, err = GetAffiliateInviteeTopUps(parent.Id, "", keyword, 0, 0, 0, 10)
				require.NoError(t, err)
				assert.Zero(t, total)
				assert.Empty(t, parentTopUps)
			}

			parentSummary, err := GetAffiliateSummary(parent.Id)
			require.NoError(t, err)
			assert.Equal(t, wantParentQuota, parentSummary.Account.PendingQuota)
			assert.Equal(t, wantParentCents, parentSummary.CashAccount.PendingCents)
			assert.Equal(t, dueAt, parentSummary.NextAvailableAt)
			require.NoError(t, DB.Model(association).Update("rate_bps", 250).Error)

			release := ReleaseDueAffiliateRewards
			ledgerModel := any(&AffiliateQuotaLedger{})
			wantWalletQuota := wantParentQuota
			if test.campaign {
				release = ReleaseDueAffiliateCashRewards
				ledgerModel = &AffiliateCashLedger{}
				wantWalletQuota = wantParentCents * unit / 100
			}
			released, _, err := release(dueAt, 0, 20)
			require.NoError(t, err)
			assert.Equal(t, 2, released)
			released, _, err = release(dueAt, 0, 20)
			require.NoError(t, err)
			assert.Zero(t, released)
			require.NoError(t, DB.First(parent, parent.Id).Error)
			require.NoError(t, DB.First(child, child.Id).Error)
			assert.EqualValues(t, wantWalletQuota, parent.Quota)
			assert.EqualValues(t, 5*unit, child.Quota)

			var entryTypes []string
			require.NoError(t, DB.Model(ledgerModel).Where("user_id = ?", parent.Id).Order("id").Pluck("entry_type", &entryTypes).Error)
			assert.Equal(t, []string{AffiliateLedgerTypeRewardPending, AffiliateLedgerTypeRewardAvailable, AffiliateLedgerTypeBalanceTransfer}, entryTypes)
			parentTopUps, _, err = GetAffiliateInviteeTopUps(parent.Id, AffiliateRewardStatusTransferred, child.Username, 0, 0, 0, 10)
			require.NoError(t, err)
			require.Len(t, parentTopUps, 1)
			assert.Equal(t, rateBps, parentTopUps[0].RewardRateBps)
			assert.Equal(t, AffiliateRewardStatusTransferred, parentTopUps[0].Status)
		})
	}
}

func TestAffiliateSubAgentPayoutCapacityPostgres(t *testing.T) {
	for _, test := range []struct {
		name             string
		campaign         bool
		rateBps          int64
		parentAtCapacity bool
		wantParentReward bool
	}{
		{name: "quota payout exceeds ledger range", rateBps: math.MaxInt64},
		{name: "cash payout exceeds ledger range", campaign: true, rateBps: math.MaxInt64},
		{name: "quota account cannot fit another reward", rateBps: 15_000, parentAtCapacity: true},
		{name: "cash account cannot fit another reward", campaign: true, rateBps: 15_000, parentAtCapacity: true},
		{name: "quota payout stays available above wallet limit", rateBps: 10_000_000, wantParentReward: true},
		{name: "cash payout stays available above wallet limit", campaign: true, rateBps: 10_000_000, wantParentReward: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupPostgresAffiliateTestDB(t)
			unit := configureAffiliateTest(t, nil)
			now := common.GetTimestamp()
			if test.campaign {
				enableAffiliateCampaignForTest(t, now-60, now+3600, 0)
			}
			parent := &User{Username: "upper-agent", AffCode: "UPPERAGENT", Status: common.UserStatusEnabled, Quota: 1}
			require.NoError(t, DB.Create(parent).Error)
			child, buyer := createAffiliateUsers(t, "sub-agent-buyer@example.com")
			require.NoError(t, DB.Create(&AffiliateSubAgent{ChildUserID: child.Id, ParentUserID: parent.Id, RateBps: test.rateBps, EffectiveAt: now - 60}).Error)
			if test.parentAtCapacity {
				if test.campaign {
					require.NoError(t, DB.Create(&AffiliateCashAccount{UserID: parent.Id, LifetimeEarnedCents: math.MaxInt64, AvailableCents: math.MaxInt64}).Error)
				} else {
					require.NoError(t, DB.Create(&AffiliateQuotaAccount{UserID: parent.Id, LifetimeEarnedQuota: math.MaxInt64, AvailableQuota: math.MaxInt64, LegacyAffQuotaMigrated: true}).Error)
				}
			}
			topUp := &TopUp{UserId: buyer.Id, TradeNo: "sub-agent-big-reward", Amount: 500, Money: 500, PaymentProvider: PaymentProviderEpay, Status: common.TopUpStatusPending, CreateTime: now}
			require.NoError(t, DB.Create(topUp).Error)
			_, _, completed, err := CompleteEpayTopUp(topUp.TradeNo, "alipay")
			require.NoError(t, err)
			assert.True(t, completed)
			require.NoError(t, DB.First(topUp, topUp.Id).Error)
			assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
			require.NoError(t, DB.First(child, child.Id).Error)
			wantChildQuota := 100 * unit
			if test.campaign {
				wantChildQuota = 125 * unit
			}
			assert.EqualValues(t, wantChildQuota, child.Quota)
			require.NoError(t, DB.First(parent, parent.Id).Error)
			assert.Equal(t, 1, parent.Quota)

			rewards, total, err := GetAffiliateInviteeTopUps(parent.Id, "", "", 0, 0, 0, 10)
			require.NoError(t, err)
			if test.wantParentReward {
				assert.EqualValues(t, 1, total)
				require.Len(t, rewards, 1)
				assert.Equal(t, AffiliateRewardStatusAvailable, rewards[0].Status)
				if test.campaign {
					assert.EqualValues(t, 50_000_000, rewards[0].RewardCents)
				} else {
					assert.Equal(t, 500_000*unit, rewards[0].RewardQuota)
				}
			} else {
				assert.Zero(t, total)
				assert.Empty(t, rewards)
			}
		})
	}
}

func TestAffiliateSubAgentRewardEligibilityPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	unit := configureAffiliateTest(t, nil)
	child, buyer := createAffiliateUsers(t, "sub-agent-eligibility@example.com")
	parent := &User{Username: "upper-affiliate-agent", AffCode: "UPPERAGENT", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(parent).Error)
	var referral AffiliateReferral
	require.NoError(t, DB.Where("invitee_user_id = ?", buyer.Id).Take(&referral).Error)
	association := &AffiliateSubAgent{ChildUserID: child.Id, ParentUserID: parent.Id, RateBps: 10_000, EffectiveAt: referral.BoundAt + 1}
	require.NoError(t, DB.Create(association).Error)

	past := createAffiliateTopUp(t, buyer.Id, "before-sub-agent-binding", 5, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, DB.Model(past).Update("complete_time", referral.BoundAt).Error)
	require.NoError(t, ProcessAffiliateTopUp(past.Id))
	require.NoError(t, DB.Model(association).Update("effective_at", referral.BoundAt).Error)

	// An upper agent may be an existing customer of the child agent. Only the
	// child earns direct cashback when the upper agent makes its own payment.
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return bindAffiliateInviteeWithTx(tx, child.Id, parent.Id)
	}))
	self := createAffiliateTopUp(t, parent.Id, "upper-agent-own-payment", 5, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(self.Id))

	operation_setting.GetAffiliateSetting().RewardRateBps = 0
	withoutReward := createAffiliateTopUp(t, buyer.Id, "child-rate-zero", 5, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(withoutReward.Id))

	var rewards []AffiliateReward
	require.NoError(t, DB.Order("id").Find(&rewards).Error)
	require.Len(t, rewards, 2)
	for _, reward := range rewards {
		assert.Equal(t, child.Id, reward.RecipientUserID)
		assert.Equal(t, unit, reward.ActualQuota)
		assert.Nil(t, reward.SourceRewardID)
	}
}

func TestAffiliateSubAgentRewardUsesOnlyCurrentPaymentPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.RewardRateBps = 500
	})
	child, buyer := createAffiliateUsers(t, "sub-agent-cumulative@example.com")
	parent := &User{Username: "upper-affiliate-agent", AffCode: "UPPERAGENT", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(parent).Error)

	// Existing referrals can carry payments recorded before they qualified and
	// before the child was assigned to a parent.
	var referral AffiliateReferral
	require.NoError(t, DB.Where("invitee_user_id = ?", buyer.Id).Take(&referral).Error)
	require.NoError(t, DB.Model(&referral).Update("cumulative_paid_cents", 7_500).Error)
	earlier := createAffiliateTopUp(t, buyer.Id, "before-sub-agent-assignment", 75, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, DB.Model(earlier).Update("complete_time", referral.BoundAt).Error)
	require.NoError(t, DB.Create(&AffiliateTopUpEvent{
		ReferralID: referral.ID, TopUpID: earlier.Id, InviterUserID: child.Id, InviteeUserID: buyer.Id,
		MaskedEmail: maskAffiliateEmail(buyer.Email, buyer.Id), PaidCents: 7_500, CumulativePaidCents: 7_500,
		CompletedAt: referral.BoundAt,
	}).Error)
	boundAt := referral.BoundAt + 1
	require.NoError(t, DB.Create(&AffiliateSubAgent{
		ChildUserID: child.Id, ParentUserID: parent.Id, RateBps: 500, EffectiveAt: boundAt,
	}).Error)

	current := createAffiliateTopUp(t, buyer.Id, "qualifying-sub-agent-payment", 100, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, DB.Model(current).Update("complete_time", boundAt).Error)
	require.NoError(t, ProcessAffiliateTopUp(current.Id))

	var direct AffiliateReward
	require.NoError(t, DB.Where("top_up_id = ? AND recipient_user_id = ?", current.Id, child.Id).Take(&direct).Error)
	assert.EqualValues(t, 17_500, direct.CumulativePaidCents)
	assert.Equal(t, 35*unit/4, direct.ActualQuota)
	var shared AffiliateReward
	require.NoError(t, DB.Where("source_reward_id = ? AND recipient_user_id = ?", direct.ID, parent.Id).Take(&shared).Error)
	assert.EqualValues(t, 10_000, shared.PaidCents)
	assert.Equal(t, 5*unit, shared.ActualQuota)
}

func TestAffiliateSubAgentLegacyRewardInInviteeTopUpsPostgres(t *testing.T) {
	setupPostgresAffiliateTestDB(t)
	unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
		setting.HoldSeconds = 3600
	})
	child, downstream := createAffiliateUsers(t, "legacy-downstream@example.com")
	parent := &User{Username: "upper-affiliate-agent", AffCode: "UPPERAGENT", Status: common.UserStatusEnabled}
	buyer := &User{Username: "parent-direct-invitee", Email: "parent-buyer@example.com", AffCode: "PARENTBUYER", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(parent).Error)
	require.NoError(t, DB.Create(buyer).Error)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return bindAffiliateInviteeWithTx(tx, parent.Id, buyer.Id)
	}))
	direct := createAffiliateTopUp(t, buyer.Id, "parent-direct-payment", 50, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, ProcessAffiliateTopUp(direct.Id))
	downstreamTopUp := createAffiliateTopUp(t, downstream.Id, "legacy-downstream-payment", 50, TopUpCompletionSourceOnlineWallet)
	require.NoError(t, DB.Model(downstreamTopUp).Update("complete_time", direct.CompleteTime+1).Error)
	require.NoError(t, ProcessAffiliateTopUp(downstreamTopUp.Id))

	var source AffiliateReward
	require.NoError(t, DB.Where("top_up_id = ?", downstreamTopUp.Id).Take(&source).Error)
	legacy := source
	legacy.ID = 0
	legacy.RecipientUserID = parent.Id
	legacy.SourceRewardID = &source.ID
	legacy.SourceRewardRateBps = 15_000
	legacy.ActualQuota = 15 * unit
	legacy.IdempotencyKey = "legacy-sub-agent-cashback"
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		saved, err := createAffiliateRewardWithTx(tx, legacy)
		if err == nil {
			legacy.ID = saved.ID
		}
		return err
	}))

	rows, total, err := GetAffiliateInviteeTopUpsSorted(parent.Id, "", "", "recharge_time_asc", 0, 0, 0, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, rows, 1)
	assert.Equal(t, direct.Id, rows[0].TopUpID)
	assert.Zero(t, rows[0].SourceAgentUserID)
	assert.False(t, rows[0].LegacySubAgentRate)

	rows, total, err = GetAffiliateInviteeTopUpsSorted(parent.Id, "", "", "recharge_time_asc", 0, 0, 1, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, rows, 1)
	assert.Equal(t, direct.CompleteTime+1, rows[0].TopUpAt)
	assert.Zero(t, rows[0].TopUpID)
	assert.Equal(t, legacy.ID, rows[0].RewardID)
	assert.Equal(t, child.Id, rows[0].SourceAgentUserID)
	assert.EqualValues(t, 15_000, rows[0].RewardRateBps)
	assert.Equal(t, 15*unit, rows[0].RewardQuota)
	assert.True(t, rows[0].LegacySubAgentRate)
}
