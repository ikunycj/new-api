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
		rateBps  int64
	}{
		{name: "regular cashback at 150 percent", rateBps: 15_000},
		{name: "campaign cash at 150 percent truncates to whole cents", campaign: true, rateBps: 15_000},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupPostgresAffiliateTestDB(t)
			const holdSeconds = 60
			unit := configureAffiliateTest(t, func(setting *operation_setting.AffiliateSetting) {
				setting.HoldSeconds = holdSeconds
			})
			now := common.GetTimestamp()
			if test.campaign {
				enableAffiliateCampaignForTest(t, now-60, now+3600, holdSeconds)
				// Existing installations can have either a standalone index or a unique constraint.
				require.NoError(t, DB.Exec("CREATE UNIQUE INDEX idx_affiliate_cash_rewards_top_up_id ON affiliate_cash_rewards (top_up_id)").Error)
				require.NoError(t, migrateAffiliateCashRewardTopUpIndex())
				require.NoError(t, DB.Exec("ALTER TABLE affiliate_cash_rewards ADD CONSTRAINT idx_affiliate_cash_rewards_top_up_id UNIQUE (top_up_id)").Error)
				require.NoError(t, migrateAffiliateCashRewardTopUpIndex())
			}
			parent := &User{Username: "upper-affiliate-agent", AffCode: "UPPERAGENT", Status: common.UserStatusEnabled}
			require.NoError(t, DB.Create(parent).Error)
			child, buyer := createAffiliateUsers(t, "sub-agent-buyer@example.com")
			association := &AffiliateSubAgent{ChildUserID: child.Id, ParentUserID: parent.Id, RateBps: test.rateBps, EffectiveAt: now - 60}
			require.NoError(t, DB.Create(association).Error)
			topUp := &TopUp{
				UserId: buyer.Id, TradeNo: "sub-agent-cashback-order", Amount: 5, Money: 5,
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
			var directID, parentRewardID int
			var dueAt, wantParentQuota, wantParentCents, wantChildQuota int64
			if test.campaign {
				require.NotNil(t, event.CashRewardID)
				directID = *event.CashRewardID
				var rewards []AffiliateCashReward
				require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).Order("id").Find(&rewards).Error)
				require.Len(t, rewards, 2)
				assert.Equal(t, directID, rewards[0].ID)
				assert.Equal(t, child.Id, rewards[0].InviterUserID)
				assert.Equal(t, int64(125), rewards[0].CashbackCents)
				assert.Nil(t, rewards[0].SourceRewardID)
				require.NotNil(t, rewards[1].SourceRewardID)
				assert.Equal(t, directID, *rewards[1].SourceRewardID)
				assert.Equal(t, parent.Id, rewards[1].InviterUserID)
				assert.Equal(t, int64(187), rewards[1].CashbackCents)
				assert.Equal(t, test.rateBps, rewards[1].SourceRewardRateBps)
				assert.Equal(t, rewards[0].AvailableAt, rewards[1].AvailableAt)
				parentRewardID, dueAt = rewards[1].ID, rewards[1].AvailableAt
				wantParentCents = 187
				wantChildQuota = 125 * unit / 100
				duplicate := rewards[0]
				duplicate.ID = 0
				duplicate.IdempotencyKey = "duplicate-direct-campaign-reward"
				err := DB.Transaction(func(tx *gorm.DB) error { return tx.Create(&duplicate).Error })
				var dbError *pgconn.PgError
				require.ErrorAs(t, err, &dbError)
				assert.Equal(t, "23505", dbError.Code)
			} else {
				require.NotNil(t, event.RewardID)
				directID = *event.RewardID
				var rewards []AffiliateReward
				require.NoError(t, DB.Where("top_up_id = ?", topUp.Id).Order("id").Find(&rewards).Error)
				require.Len(t, rewards, 2)
				assert.Equal(t, directID, rewards[0].ID)
				assert.Equal(t, child.Id, rewards[0].RecipientUserID)
				assert.Equal(t, unit, rewards[0].ActualQuota)
				assert.Nil(t, rewards[0].SourceRewardID)
				require.NotNil(t, rewards[1].SourceRewardID)
				assert.Equal(t, directID, *rewards[1].SourceRewardID)
				assert.Equal(t, parent.Id, rewards[1].RecipientUserID)
				assert.Equal(t, child.Id, rewards[1].InviterUserID)
				assert.Equal(t, 3*unit/2, rewards[1].ActualQuota)
				assert.Equal(t, test.rateBps, rewards[1].SourceRewardRateBps)
				assert.Equal(t, rewards[0].AvailableAt, rewards[1].AvailableAt)
				parentRewardID, dueAt = rewards[1].ID, rewards[1].AvailableAt
				wantParentQuota, wantChildQuota = 3*unit/2, unit

				adminRewards, total, err := GetAffiliateAdminRewards(strconv.Itoa(parent.Id), "", 0, 0, 0, 10)
				require.NoError(t, err)
				assert.EqualValues(t, 1, total)
				require.Len(t, adminRewards, 1)
				assert.Equal(t, parent.Id, adminRewards[0].InviterUserID)
				assert.Equal(t, child.Id, adminRewards[0].SourceAgentUserID)
				assert.Equal(t, child.Username, adminRewards[0].SourceAgentUsername)
				_, err = CreateAffiliateRewardBalanceTransfer(child.Id, parentRewardID, "forbidden-child-transfer")
				assert.ErrorIs(t, err, ErrAffiliateRewardNotFound)
			}

			parentRewards, total, err := GetAffiliateSubAgentRewards(parent.Id, 0, 10)
			require.NoError(t, err)
			assert.EqualValues(t, 1, total)
			require.Len(t, parentRewards, 1)
			assert.Equal(t, parentRewardID, parentRewards[0].ID)
			assert.Equal(t, child.Id, parentRewards[0].SourceAgentUserID)
			assert.Equal(t, child.Username, parentRewards[0].SourceAgentUsername)
			assert.Equal(t, test.rateBps, parentRewards[0].RateBps)
			assert.Equal(t, wantParentQuota, parentRewards[0].RewardQuota)
			assert.Equal(t, wantParentCents, parentRewards[0].RewardCents)
			assert.Equal(t, AffiliateRewardStatusPending, parentRewards[0].Status)
			assert.Equal(t, dueAt, parentRewards[0].AvailableAt)
			childRewards, total, err := GetAffiliateSubAgentRewards(child.Id, 0, 10)
			require.NoError(t, err)
			assert.Zero(t, total)
			assert.Empty(t, childRewards)
			parentTopUps, total, err := GetAffiliateInviteeTopUps(parent.Id, "", "", 0, 0, 0, 10)
			require.NoError(t, err)
			assert.Zero(t, total)
			assert.Empty(t, parentTopUps)

			parentSummary, err := GetAffiliateSummary(parent.Id)
			require.NoError(t, err)
			assert.True(t, parentSummary.HasSubAgentRewards)
			assert.Equal(t, wantParentQuota, parentSummary.Account.PendingQuota)
			assert.Equal(t, wantParentCents, parentSummary.CashAccount.PendingCents)
			assert.Equal(t, dueAt, parentSummary.NextAvailableAt)
			require.NoError(t, DB.Model(association).Update("rate_bps", 500).Error)

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
			assert.EqualValues(t, wantChildQuota, child.Quota)

			var entryTypes []string
			require.NoError(t, DB.Model(ledgerModel).Where("user_id = ?", parent.Id).Order("id").Pluck("entry_type", &entryTypes).Error)
			assert.Equal(t, []string{AffiliateLedgerTypeRewardPending, AffiliateLedgerTypeRewardAvailable, AffiliateLedgerTypeBalanceTransfer}, entryTypes)
			parentRewards, _, err = GetAffiliateSubAgentRewards(parent.Id, 0, 10)
			require.NoError(t, err)
			require.Len(t, parentRewards, 1)
			assert.Equal(t, test.rateBps, parentRewards[0].RateBps)
			assert.Equal(t, AffiliateRewardStatusTransferred, parentRewards[0].Status)
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

			rewards, total, err := GetAffiliateSubAgentRewards(parent.Id, 0, 10)
			require.NoError(t, err)
			if test.wantParentReward {
				assert.EqualValues(t, 1, total)
				require.Len(t, rewards, 1)
				assert.Equal(t, AffiliateRewardStatusAvailable, rewards[0].Status)
				if test.campaign {
					assert.Equal(t, int64(12_500_000), rewards[0].RewardCents)
				} else {
					assert.Equal(t, 100_000*unit, rewards[0].RewardQuota)
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
