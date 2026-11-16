package model

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AffiliateReferralStatusBound     = "bound"
	AffiliateReferralStatusQualified = "qualified"

	AffiliateRewardTypeInviter  = "inviter_reward"
	AffiliateRewardTypeInvitee  = "invitee_reward"
	AffiliateRewardTypeCashback = "cashback"

	AffiliateRewardStatusPending     = "pending"
	AffiliateRewardStatusAvailable   = "available"
	AffiliateRewardStatusTransferred = "transferred"
	AffiliateRewardStatusCredited    = "credited"
	AffiliateRewardStatusAdjusted    = "adjusted"

	AffiliateLedgerTypeRewardPending   = "reward_pending"
	AffiliateLedgerTypeRewardAvailable = "reward_available"
	AffiliateLedgerTypeBalanceTransfer = "balance_transfer"
	AffiliateLedgerTypeAdjustment      = "adjustment"
	AffiliateLedgerTypeLegacyMigration = "legacy_migration"

	AffiliateReferenceReward     = "reward"
	AffiliateReferenceTransfer   = "balance_transfer"
	AffiliateReferenceAdjustment = "adjustment"
	AffiliateReferenceLegacy     = "legacy_aff_quota"

	AffiliateAdjustmentStatusApplied        = "applied"
	AffiliateAdjustmentStatusManualRequired = "manual_required"

	TopUpCompletionSourceOnlineWallet = "online_wallet"
	TopUpCompletionSourceAdmin        = "admin"
)

var (
	ErrAffiliateDisabled         = errors.New("affiliate cashback is disabled")
	ErrAffiliateAmountInvalid    = errors.New("invalid affiliate amount")
	ErrAffiliateBalance          = errors.New("insufficient available cashback balance")
	ErrAffiliateTopUpsHidden     = errors.New("invitee top-up records are not available")
	ErrAffiliateRewardNotFound   = errors.New("affiliate reward not found")
	ErrAffiliateRuleInvalid      = errors.New("invalid affiliate rule")
	ErrAffiliateTransferTooSmall = errors.New("cashback transfer is below the minimum")
	ErrAffiliateDomainInvalid    = errors.New("invalid affiliate domain")
	ErrAffiliateDomainTaken      = errors.New("affiliate domain is already assigned")
)

type AffiliateRuleVersion struct {
	ID                        int    `json:"id"`
	InviterUserID             int    `json:"inviter_user_id" gorm:"uniqueIndex:idx_affiliate_rule_version;not null"`
	ConfigHash                string `json:"-" gorm:"type:varchar(64);uniqueIndex:idx_affiliate_rule_version;not null"`
	Source                    string `json:"source" gorm:"type:varchar(32);not null"`
	Enabled                   bool   `json:"enabled" gorm:"not null"`
	InviterRewardQuota        int64  `json:"inviter_reward_quota" gorm:"type:bigint;not null"`
	InviteeRewardQuota        int64  `json:"invitee_reward_quota" gorm:"type:bigint;not null"`
	RegistrationRewardTrigger string `json:"registration_reward_trigger" gorm:"type:varchar(32);not null"`
	RewardMode                string `json:"reward_mode" gorm:"type:varchar(20);not null"`
	CashbackFrequency         string `json:"cashback_frequency" gorm:"type:varchar(24);not null"`
	RewardRateBps             int64  `json:"reward_rate_bps" gorm:"type:bigint;not null"`
	FixedRewardQuota          int64  `json:"fixed_reward_quota" gorm:"type:bigint;not null"`
	UnlimitedReward           bool   `json:"unlimited_reward" gorm:"not null"`
	MaximumRewardQuota        int64  `json:"maximum_reward_quota" gorm:"type:bigint;not null"`
	MinimumTopUpCents         int64  `json:"minimum_topup_cents" gorm:"type:bigint;not null"`
	HoldSeconds               int64  `json:"hold_seconds" gorm:"type:bigint;not null"`
	MinimumTransferQuota      int64  `json:"minimum_transfer_quota" gorm:"type:bigint;not null"`
	CreatedAt                 int64  `json:"created_at" gorm:"autoCreateTime"`
}

type AffiliateUserOverride struct {
	ID                        int     `json:"id"`
	UserID                    int     `json:"user_id" gorm:"uniqueIndex;not null"`
	Enabled                   *bool   `json:"enabled"`
	InviterRewardQuota        *int64  `json:"inviter_reward_quota" gorm:"type:bigint"`
	InviteeRewardQuota        *int64  `json:"invitee_reward_quota" gorm:"type:bigint"`
	RegistrationRewardTrigger *string `json:"registration_reward_trigger" gorm:"type:varchar(32)"`
	RewardMode                *string `json:"reward_mode" gorm:"type:varchar(20)"`
	CashbackFrequency         *string `json:"cashback_frequency" gorm:"type:varchar(24)"`
	RewardRateBps             *int64  `json:"reward_rate_bps" gorm:"type:bigint"`
	FixedRewardQuota          *int64  `json:"fixed_reward_quota" gorm:"type:bigint"`
	UnlimitedReward           *bool   `json:"unlimited_reward"`
	MaximumRewardQuota        *int64  `json:"maximum_reward_quota" gorm:"type:bigint"`
	MinimumTopUpCents         *int64  `json:"minimum_topup_cents" gorm:"type:bigint"`
	HoldSeconds               *int64  `json:"hold_seconds" gorm:"type:bigint"`
	MinimumTransferQuota      *int64  `json:"minimum_transfer_quota" gorm:"type:bigint"`
	ShowInviteeTopUps         *bool   `json:"show_invitee_topups"`
	AffiliateDomain           *string `json:"affiliate_domain" gorm:"type:varchar(255);uniqueIndex"`
	Remark                    *string `json:"remark" gorm:"-"`
	UpdatedBy                 int     `json:"updated_by" gorm:"index;not null"`
	UpdatedAt                 int64   `json:"updated_at" gorm:"autoUpdateTime"`
	ChangeReason              string  `json:"change_reason" gorm:"type:varchar(500);not null"`
}

type AffiliateReferral struct {
	ID                    int    `json:"id"`
	InviteeUserID         int    `json:"invitee_user_id" gorm:"uniqueIndex;not null"`
	InviterUserID         int    `json:"inviter_user_id" gorm:"index;not null"`
	CodeSnapshot          string `json:"code_snapshot" gorm:"type:varchar(32);index;not null"`
	Status                string `json:"status" gorm:"type:varchar(20);index;not null"`
	BoundAt               int64  `json:"bound_at" gorm:"index;not null"`
	QualifiedAt           int64  `json:"qualified_at" gorm:"not null"`
	QualifyingTopUpID     *int   `json:"qualifying_topup_id,omitempty" gorm:"uniqueIndex"`
	CumulativePaidCents   int64  `json:"cumulative_paid_cents" gorm:"type:bigint;not null"`
	CashbackRewardedQuota int64  `json:"cashback_rewarded_quota" gorm:"type:bigint;not null"`
}

type AffiliateReward struct {
	ID                      int     `json:"id"`
	ReferralID              int     `json:"referral_id" gorm:"index;not null"`
	InviterUserID           int     `json:"inviter_user_id" gorm:"index;not null"`
	InviteeUserID           int     `json:"invitee_user_id" gorm:"index;not null"`
	RecipientUserID         int     `json:"recipient_user_id" gorm:"index;not null"`
	RewardType              string  `json:"reward_type" gorm:"type:varchar(24);index;not null"`
	TopUpID                 *int    `json:"topup_id,omitempty" gorm:"index"`
	RuleVersionID           int     `json:"rule_version_id" gorm:"index;not null"`
	PaidCents               int64   `json:"paid_cents" gorm:"type:bigint;not null"`
	CumulativePaidCents     int64   `json:"cumulative_paid_cents" gorm:"type:bigint;not null"`
	RewardMode              string  `json:"reward_mode" gorm:"type:varchar(20);not null"`
	CashbackFrequency       string  `json:"cashback_frequency" gorm:"type:varchar(24);not null"`
	RewardRateBps           int64   `json:"reward_rate_bps" gorm:"type:bigint;not null"`
	FixedRewardQuota        int64   `json:"fixed_reward_quota" gorm:"type:bigint;not null"`
	UnlimitedReward         bool    `json:"unlimited_reward" gorm:"not null"`
	MaximumRewardQuota      int64   `json:"maximum_reward_quota" gorm:"type:bigint;not null"`
	MinimumTopUpCents       int64   `json:"minimum_topup_cents" gorm:"type:bigint;not null"`
	HoldSeconds             int64   `json:"hold_seconds" gorm:"type:bigint;not null"`
	QuotaPerUnitSnapshot    float64 `json:"quota_per_unit_snapshot" gorm:"not null"`
	CNYExchangeRateSnapshot float64 `json:"cny_exchange_rate_snapshot" gorm:"not null"`
	ActualQuota             int64   `json:"actual_quota" gorm:"type:bigint;not null"`
	AdjustedQuota           int64   `json:"adjusted_quota" gorm:"type:bigint;not null"`
	TransferredQuota        int64   `json:"transferred_quota" gorm:"type:bigint;not null"`
	Status                  string  `json:"status" gorm:"type:varchar(20);index;not null"`
	AvailableAt             int64   `json:"available_at" gorm:"index;not null"`
	CreatedAt               int64   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt               int64   `json:"updated_at" gorm:"autoUpdateTime"`
	IdempotencyKey          string  `json:"-" gorm:"type:varchar(128);uniqueIndex;not null"`
}

type AffiliateQuotaAccount struct {
	ID                     int   `json:"id"`
	UserID                 int   `json:"user_id" gorm:"uniqueIndex;not null"`
	PendingQuota           int64 `json:"pending_quota" gorm:"type:bigint;not null"`
	AvailableQuota         int64 `json:"available_quota" gorm:"type:bigint;not null"`
	TransferredQuota       int64 `json:"transferred_quota" gorm:"type:bigint;not null"`
	LifetimeEarnedQuota    int64 `json:"lifetime_earned_quota" gorm:"type:bigint;not null"`
	LegacyAffQuotaMigrated bool  `json:"-" gorm:"not null"`
	CreatedAt              int64 `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt              int64 `json:"updated_at" gorm:"autoUpdateTime"`
}

type AffiliateQuotaLedger struct {
	ID                      int    `json:"id"`
	UserID                  int    `json:"user_id" gorm:"index;not null"`
	AccountID               int    `json:"account_id" gorm:"index;not null"`
	EntryType               string `json:"entry_type" gorm:"type:varchar(40);index;not null"`
	ReferenceType           string `json:"reference_type" gorm:"type:varchar(32);index;not null"`
	ReferenceID             int    `json:"reference_id" gorm:"index;not null"`
	AmountQuota             int64  `json:"amount_quota" gorm:"type:bigint;not null"`
	PendingDeltaQuota       int64  `json:"pending_delta_quota" gorm:"type:bigint;not null"`
	AvailableDeltaQuota     int64  `json:"available_delta_quota" gorm:"type:bigint;not null"`
	TransferredDeltaQuota   int64  `json:"transferred_delta_quota" gorm:"type:bigint;not null"`
	PendingBalanceQuota     int64  `json:"pending_balance_quota" gorm:"type:bigint;not null"`
	AvailableBalanceQuota   int64  `json:"available_balance_quota" gorm:"type:bigint;not null"`
	TransferredBalanceQuota int64  `json:"transferred_balance_quota" gorm:"type:bigint;not null"`
	IdempotencyKey          string `json:"-" gorm:"type:varchar(128);uniqueIndex;not null"`
	CreatedAt               int64  `json:"created_at" gorm:"autoCreateTime;index"`
}

type AffiliateBalanceTransfer struct {
	ID                     int    `json:"id"`
	UserID                 int    `json:"user_id" gorm:"uniqueIndex:idx_affiliate_transfer_request;index;not null"`
	RequestKey             string `json:"-" gorm:"type:varchar(64);uniqueIndex:idx_affiliate_transfer_request;not null"`
	AmountQuota            int64  `json:"amount_quota" gorm:"type:bigint;not null"`
	AffiliateBalanceBefore int64  `json:"affiliate_balance_before" gorm:"type:bigint;not null"`
	AffiliateBalanceAfter  int64  `json:"affiliate_balance_after" gorm:"type:bigint;not null"`
	UserQuotaBefore        int    `json:"user_quota_before" gorm:"type:int;not null"`
	UserQuotaAfter         int    `json:"user_quota_after" gorm:"type:int;not null"`
	CreatedAt              int64  `json:"created_at" gorm:"autoCreateTime;index"`
}

type AffiliateAdjustment struct {
	ID                 int    `json:"id"`
	RewardID           int    `json:"reward_id" gorm:"uniqueIndex:idx_affiliate_adjustment_request;index;not null"`
	RequestKey         string `json:"-" gorm:"type:varchar(64);uniqueIndex:idx_affiliate_adjustment_request;not null"`
	AdminUserID        int    `json:"admin_user_id" gorm:"index;not null"`
	RequestedQuota     int64  `json:"requested_quota" gorm:"type:bigint;not null"`
	AppliedQuota       int64  `json:"applied_quota" gorm:"type:bigint;not null"`
	PendingManualQuota int64  `json:"pending_manual_quota" gorm:"type:bigint;not null"`
	Status             string `json:"status" gorm:"type:varchar(24);index;not null"`
	Reason             string `json:"reason" gorm:"type:varchar(500);not null"`
	CreatedAt          int64  `json:"created_at" gorm:"autoCreateTime;index"`
}

type AffiliateTopUpEvent struct {
	ID                  int    `json:"id"`
	ReferralID          int    `json:"referral_id" gorm:"index;not null"`
	TopUpID             int    `json:"topup_id" gorm:"uniqueIndex;not null"`
	InviterUserID       int    `json:"inviter_user_id" gorm:"index;not null"`
	InviteeUserID       int    `json:"invitee_user_id" gorm:"index;not null"`
	MaskedEmail         string `json:"masked_email" gorm:"type:varchar(255);index;not null"`
	PaidCents           int64  `json:"paid_cents" gorm:"type:bigint;not null"`
	CumulativePaidCents int64  `json:"cumulative_paid_cents" gorm:"type:bigint;not null"`
	Qualified           bool   `json:"qualified" gorm:"index;not null"`
	RewardID            *int   `json:"reward_id,omitempty" gorm:"index"`
	CashRewardID        *int   `json:"cash_reward_id,omitempty" gorm:"index"`
	QuotaGrantID        *int   `json:"quota_grant_id,omitempty" gorm:"index"`
	CompletedAt         int64  `json:"completed_at" gorm:"index;not null"`
	CreatedAt           int64  `json:"created_at" gorm:"autoCreateTime"`
}

type AffiliateEffectiveRule struct {
	Enabled                   bool   `json:"enabled"`
	InviterRewardQuota        int64  `json:"inviter_reward_quota"`
	InviteeRewardQuota        int64  `json:"invitee_reward_quota"`
	RegistrationRewardTrigger string `json:"registration_reward_trigger"`
	RewardMode                string `json:"reward_mode"`
	CashbackFrequency         string `json:"cashback_frequency"`
	RewardRateBps             int64  `json:"reward_rate_bps"`
	FixedRewardQuota          int64  `json:"fixed_reward_quota"`
	UnlimitedReward           bool   `json:"unlimited_reward"`
	MaximumRewardQuota        int64  `json:"maximum_reward_quota"`
	MinimumTopUpCents         int64  `json:"minimum_topup_cents"`
	HoldSeconds               int64  `json:"hold_seconds"`
	MinimumTransferQuota      int64  `json:"minimum_transfer_quota"`
	ShowInviteeTopUps         bool   `json:"show_invitee_topups"`
	Source                    string `json:"source"`
}

type AffiliateSummary struct {
	Enabled                    bool                   `json:"enabled"`
	ReferralCode               string                 `json:"referral_code"`
	Currency                   string                 `json:"currency"`
	ReferralCount              int64                  `json:"referral_count"`
	QualifiedCount             int64                  `json:"qualified_count"`
	NextAvailableAt            int64                  `json:"next_available_at"`
	LifetimeCampaignBonusQuota int64                  `json:"lifetime_campaign_bonus_quota"`
	Rule                       AffiliateEffectiveRule `json:"rule"`
	Account                    AffiliateQuotaAccount  `json:"account"`
	CashAccount                AffiliateCashAccount   `json:"cash_account"`
	Campaign                   AffiliateCampaign      `json:"campaign"`
}

type AffiliateInviteeTopUp struct {
	ID                     int    `json:"id"`
	RewardID               int    `json:"reward_id"`
	CashRewardID           int    `json:"cash_reward_id"`
	MaskedEmail            string `json:"masked_email"`
	InvitedAt              int64  `json:"invited_at"`
	InvitationCode         string `json:"invitation_code"`
	TopUpID                int    `json:"topup_id"`
	TopUpAt                int64  `json:"topup_at"`
	PaidCents              int64  `json:"paid_cents"`
	RewardMode             string `json:"reward_mode"`
	RewardRateBps          int64  `json:"reward_rate_bps"`
	FixedRewardQuota       int64  `json:"fixed_reward_quota"`
	RewardQuota            int64  `json:"reward_quota"`
	RewardCents            int64  `json:"reward_cents"`
	AvailableRewardQuota   int64  `json:"available_reward_quota"`
	AvailableRewardCents   int64  `json:"available_reward_cents"`
	TransferredRewardQuota int64  `json:"transferred_reward_quota"`
	TransferredRewardCents int64  `json:"transferred_reward_cents"`
	AvailableAt            int64  `json:"available_at"`
	Status                 string `json:"status"`
}

type AffiliateUserOverrideView struct {
	UserID            int                     `json:"user_id"`
	Username          string                  `json:"username"`
	Email             string                  `json:"email"`
	Remark            string                  `json:"remark"`
	InviteInfo        AffiliateUserInviteInfo `json:"invite_info"`
	UpdatedByUsername string                  `json:"updated_by_username"`
	Override          *AffiliateUserOverride  `json:"override"`
	GlobalRule        AffiliateEffectiveRule  `json:"global_rule"`
	EffectiveRule     AffiliateEffectiveRule  `json:"effective_rule"`
}

type AffiliateUserInviteInfo struct {
	UserID                  int    `json:"user_id"`
	InvitedCount            int64  `json:"invited_count"`
	IncomeCents             int64  `json:"income_cents"`
	InviterUserID           int    `json:"inviter_user_id"`
	InviterUsername         string `json:"inviter_username"`
	InviterEmail            string `json:"inviter_email"`
	LifetimeEarnedQuota     int64  `json:"-"`
	LifetimeEarnedCashCents int64  `json:"-"`
}

type AffiliateAdminReferralActivity struct {
	ID                      int    `json:"id"`
	InviteeUserID           int    `json:"invitee_user_id"`
	Username                string `json:"username"`
	Email                   string `json:"email"`
	OperationType           string `json:"operation_type"`
	TopUpAmountCents        int64  `json:"topup_amount_cents" gorm:"column:topup_amount_cents"`
	CashbackAmountCents     int64  `json:"cashback_amount_cents"`
	CumulativeCashbackCents int64  `json:"cumulative_cashback_cents"`
	OccurredAt              int64  `json:"occurred_at"`
}

const (
	AffiliateActivityInvitation = "invitation"
	AffiliateActivityTopUp      = "top_up"
)

type AffiliateAdminReward struct {
	ID               int    `json:"id"`
	InviterUserID    int    `json:"inviter_user_id"`
	InviterUsername  string `json:"inviter_username"`
	InviterEmail     string `json:"inviter_email"`
	InviteeUserID    int    `json:"invitee_user_id"`
	InviteeUsername  string `json:"invitee_username"`
	InviteeEmail     string `json:"invitee_email"`
	InvitationCode   string `json:"invitation_code"`
	TopUpID          int    `json:"topup_id"`
	TradeNo          string `json:"trade_no"`
	PaidCents        int64  `json:"paid_cents"`
	ActualQuota      int64  `json:"actual_quota"`
	AdjustedQuota    int64  `json:"adjusted_quota"`
	TransferredQuota int64  `json:"transferred_quota"`
	Status           string `json:"status"`
	AvailableAt      int64  `json:"available_at"`
	CreatedAt        int64  `json:"created_at"`
}

func normalizedAffiliateSetting() operation_setting.AffiliateSetting {
	return normalizeAffiliateSettingValues(*operation_setting.GetAffiliateSetting())
}

func globalAffiliateRule() AffiliateEffectiveRule {
	setting := normalizedAffiliateSetting()
	return AffiliateEffectiveRule{
		Enabled:                   setting.Enabled,
		InviterRewardQuota:        setting.InviterRewardQuota,
		InviteeRewardQuota:        setting.InviteeRewardQuota,
		RegistrationRewardTrigger: setting.RegistrationRewardTrigger,
		RewardMode:                setting.RewardMode,
		CashbackFrequency:         setting.CashbackFrequency,
		RewardRateBps:             setting.RewardRateBps,
		FixedRewardQuota:          setting.FixedRewardQuota,
		UnlimitedReward:           setting.UnlimitedReward,
		MaximumRewardQuota:        setting.MaximumRewardQuota,
		MinimumTopUpCents:         setting.MinimumTopUpCents,
		HoldSeconds:               setting.HoldSeconds,
		MinimumTransferQuota:      setting.MinimumTransferQuota,
		ShowInviteeTopUps:         setting.ShowInviteeTopUps,
		Source:                    "global",
	}
}

func validateAffiliateRule(rule AffiliateEffectiveRule) error {
	maxQuota := int64(common.MaxQuota)
	if rule.InviterRewardQuota < 0 || rule.InviterRewardQuota > maxQuota ||
		rule.InviteeRewardQuota < 0 || rule.InviteeRewardQuota > maxQuota ||
		rule.FixedRewardQuota < 0 || rule.FixedRewardQuota > maxQuota ||
		rule.MaximumRewardQuota < 0 || rule.MaximumRewardQuota > maxQuota ||
		rule.MinimumTransferQuota < 0 || rule.MinimumTransferQuota > maxQuota ||
		rule.RewardRateBps < 0 || rule.RewardRateBps > 10_000 ||
		rule.MinimumTopUpCents < 0 || rule.HoldSeconds < 0 || rule.HoldSeconds > 365*24*60*60 {
		return ErrAffiliateRuleInvalid
	}
	if !rule.UnlimitedReward && rule.MaximumRewardQuota <= 0 {
		return ErrAffiliateRuleInvalid
	}
	if rule.RegistrationRewardTrigger != operation_setting.AffiliateRegistrationTriggerDisabled ||
		rule.RewardMode != operation_setting.AffiliateRewardModePercentage ||
		rule.CashbackFrequency != operation_setting.AffiliateCashbackFrequencyEveryTopUp ||
		rule.InviterRewardQuota != 0 || rule.InviteeRewardQuota != 0 ||
		rule.FixedRewardQuota != 0 || rule.MinimumTopUpCents != 0 ||
		rule.MinimumTransferQuota != 0 {
		return ErrAffiliateRuleInvalid
	}
	return nil
}

func applyAffiliateOverride(rule AffiliateEffectiveRule, override *AffiliateUserOverride) AffiliateEffectiveRule {
	if override == nil {
		return rule
	}
	applied := false
	if override.RewardRateBps != nil {
		rule.RewardRateBps = *override.RewardRateBps
		applied = true
	}
	if override.HoldSeconds != nil {
		rule.HoldSeconds = *override.HoldSeconds
		applied = true
	}
	if applied {
		rule.Source = "user_override"
	}
	return rule
}

func effectiveAffiliateRuleWithTx(tx *gorm.DB, inviterUserID int) (AffiliateEffectiveRule, *AffiliateUserOverride, error) {
	rule := globalAffiliateRule()
	var override AffiliateUserOverride
	if err := tx.Where("user_id = ?", inviterUserID).First(&override).Error; err == nil {
		rule = applyAffiliateOverride(rule, &override)
		if err := validateAffiliateRule(rule); err != nil {
			return AffiliateEffectiveRule{}, nil, err
		}
		return rule, &override, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return AffiliateEffectiveRule{}, nil, err
	}
	if err := validateAffiliateRule(rule); err != nil {
		return AffiliateEffectiveRule{}, nil, err
	}
	return rule, nil, nil
}

func GetAffiliateEffectiveRule(inviterUserID int) (AffiliateEffectiveRule, error) {
	rule, _, err := effectiveAffiliateRuleWithTx(DB, inviterUserID)
	return rule, err
}

func affiliateRuleHash(rule AffiliateEffectiveRule) string {
	value := fmt.Sprintf("%t|%d|%d|%s|%s|%s|%d|%d|%t|%d|%d|%d|%d|%s",
		rule.Enabled, rule.InviterRewardQuota, rule.InviteeRewardQuota,
		rule.RegistrationRewardTrigger, rule.RewardMode, rule.CashbackFrequency,
		rule.RewardRateBps, rule.FixedRewardQuota, rule.UnlimitedReward,
		rule.MaximumRewardQuota, rule.MinimumTopUpCents, rule.HoldSeconds,
		rule.MinimumTransferQuota, rule.Source)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

func ensureAffiliateRuleVersionWithTx(tx *gorm.DB, inviterUserID int, rule AffiliateEffectiveRule) (*AffiliateRuleVersion, error) {
	hash := affiliateRuleHash(rule)
	var version AffiliateRuleVersion
	if err := tx.Where("inviter_user_id = ? AND config_hash = ?", inviterUserID, hash).First(&version).Error; err == nil {
		return &version, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	version = AffiliateRuleVersion{
		InviterUserID: inviterUserID, ConfigHash: hash, Source: rule.Source,
		Enabled: rule.Enabled, InviterRewardQuota: rule.InviterRewardQuota,
		InviteeRewardQuota:        rule.InviteeRewardQuota,
		RegistrationRewardTrigger: rule.RegistrationRewardTrigger,
		RewardMode:                rule.RewardMode, CashbackFrequency: rule.CashbackFrequency,
		RewardRateBps: rule.RewardRateBps, FixedRewardQuota: rule.FixedRewardQuota,
		UnlimitedReward: rule.UnlimitedReward, MaximumRewardQuota: rule.MaximumRewardQuota,
		MinimumTopUpCents: rule.MinimumTopUpCents, HoldSeconds: rule.HoldSeconds,
		MinimumTransferQuota: rule.MinimumTransferQuota,
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "inviter_user_id"}, {Name: "config_hash"}},
		DoNothing: true,
	}).Create(&version)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		if err := tx.Where("inviter_user_id = ? AND config_hash = ?", inviterUserID, hash).First(&version).Error; err != nil {
			return nil, err
		}
	}
	return &version, nil
}

func appendAffiliateQuotaLedgerWithTx(tx *gorm.DB, account *AffiliateQuotaAccount, entry AffiliateQuotaLedger) error {
	entry.UserID = account.UserID
	entry.AccountID = account.ID
	entry.PendingBalanceQuota = account.PendingQuota
	entry.AvailableBalanceQuota = account.AvailableQuota
	entry.TransferredBalanceQuota = account.TransferredQuota
	return tx.Create(&entry).Error
}

func ensureAffiliateQuotaAccountWithTx(tx *gorm.DB, userID int) (*AffiliateQuotaAccount, error) {
	var account AffiliateQuotaAccount
	if err := tx.Where("user_id = ?", userID).First(&account).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		account = AffiliateQuotaAccount{UserID: userID}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).Create(&account)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			if err := tx.Where("user_id = ?", userID).First(&account).Error; err != nil {
				return nil, err
			}
		}
	}
	if account.LegacyAffQuotaMigrated {
		if err := lockForUpdate(tx).Where("id = ?", account.ID).First(&account).Error; err != nil {
			return nil, err
		}
		return &account, nil
	}
	var user User
	if err := lockForUpdate(tx).Select("id", "aff_quota", "aff_history").Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	if err := lockForUpdate(tx).Where("id = ?", account.ID).First(&account).Error; err != nil {
		return nil, err
	}
	if account.LegacyAffQuotaMigrated {
		return &account, nil
	}
	legacyQuota := int64(user.AffQuota)
	legacyLifetimeQuota := int64(user.AffHistoryQuota)
	if legacyLifetimeQuota < legacyQuota {
		legacyLifetimeQuota = legacyQuota
	}
	legacyTransferredQuota := legacyLifetimeQuota - legacyQuota
	if legacyQuota > 0 {
		account.AvailableQuota += legacyQuota
		if err := tx.Model(&User{}).Where("id = ?", userID).Update("aff_quota", 0).Error; err != nil {
			return nil, err
		}
	}
	account.TransferredQuota += legacyTransferredQuota
	account.LifetimeEarnedQuota += legacyLifetimeQuota
	account.LegacyAffQuotaMigrated = true
	if err := tx.Save(&account).Error; err != nil {
		return nil, err
	}
	if legacyLifetimeQuota > 0 {
		if err := appendAffiliateQuotaLedgerWithTx(tx, &account, AffiliateQuotaLedger{
			EntryType: AffiliateLedgerTypeLegacyMigration, ReferenceType: AffiliateReferenceLegacy,
			ReferenceID: userID, AmountQuota: legacyLifetimeQuota,
			AvailableDeltaQuota: legacyQuota, TransferredDeltaQuota: legacyTransferredQuota,
			IdempotencyKey: "legacy-aff-quota:" + strconv.Itoa(userID),
		}); err != nil {
			return nil, err
		}
	}
	return &account, nil
}

func ensureAffiliateCodeWithTx(tx *gorm.DB, user *User) error {
	if strings.TrimSpace(user.AffCode) != "" {
		return nil
	}
	code := strings.ToUpper(strconv.FormatInt(int64(user.Id), 36)) + "-" + common.GetRandomString(8)
	result := tx.Model(&User{}).Where("id = ? AND (aff_code = '' OR aff_code IS NULL)", user.Id).Update("aff_code", code)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		user.AffCode = code
		return nil
	}
	if err := tx.Select("aff_code").Where("id = ?", user.Id).First(user).Error; err == nil && user.AffCode != "" {
		return nil
	}
	return errors.New("failed to create affiliate code")
}

func createAffiliateRewardWithTx(tx *gorm.DB, relation *AffiliateReferral, rule AffiliateEffectiveRule, versionID int, rewardType string, recipientUserID int, amountQuota int64, topUpID *int, paidCents int64, cumulativePaidCents int64, idempotencyKey string) (*AffiliateReward, bool, error) {
	var existing AffiliateReward
	if err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
		return &existing, false, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	if rewardType != AffiliateRewardTypeInvitee {
		var recipient User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", recipientUserID).First(&recipient).Error; err != nil {
			return nil, false, err
		}
	}
	now := common.GetTimestamp()
	status := AffiliateRewardStatusCredited
	availableAt := now
	if rewardType != AffiliateRewardTypeInvitee {
		availableAt = now + rule.HoldSeconds
		status = AffiliateRewardStatusAvailable
		if rule.HoldSeconds > 0 {
			status = AffiliateRewardStatusPending
		}
	}
	exchangeRate := operation_setting.USDExchangeRate
	if exchangeRate <= 0 {
		exchangeRate = 1
	}
	reward := AffiliateReward{
		ReferralID: relation.ID, InviterUserID: relation.InviterUserID,
		InviteeUserID: relation.InviteeUserID, RecipientUserID: recipientUserID,
		RewardType: rewardType, TopUpID: topUpID, RuleVersionID: versionID,
		PaidCents: paidCents, CumulativePaidCents: cumulativePaidCents,
		RewardMode: rule.RewardMode, CashbackFrequency: rule.CashbackFrequency,
		RewardRateBps: rule.RewardRateBps, FixedRewardQuota: rule.FixedRewardQuota,
		UnlimitedReward: rule.UnlimitedReward, MaximumRewardQuota: rule.MaximumRewardQuota,
		MinimumTopUpCents: rule.MinimumTopUpCents, HoldSeconds: rule.HoldSeconds,
		QuotaPerUnitSnapshot: common.QuotaPerUnit, CNYExchangeRateSnapshot: exchangeRate,
		ActualQuota: amountQuota, Status: status, AvailableAt: availableAt,
		IdempotencyKey: idempotencyKey,
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(&reward)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		if err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err != nil {
			return nil, false, err
		}
		return &existing, false, nil
	}
	if amountQuota <= 0 {
		return &reward, true, nil
	}
	if rewardType == AffiliateRewardTypeInvitee {
		var invitee User
		if err := lockForUpdate(tx).Select("id", "quota").Where("id = ?", recipientUserID).First(&invitee).Error; err != nil {
			return nil, false, err
		}
		quotaAfter := int64(invitee.Quota) + amountQuota
		if quotaAfter > int64(common.MaxQuota) {
			return nil, false, ErrAffiliateAmountInvalid
		}
		if err := tx.Model(&User{}).Where("id = ?", recipientUserID).Update("quota", int(quotaAfter)).Error; err != nil {
			return nil, false, err
		}
		return &reward, true, nil
	}
	account, err := ensureAffiliateQuotaAccountWithTx(tx, recipientUserID)
	if err != nil {
		return nil, false, err
	}
	entry := AffiliateQuotaLedger{
		ReferenceType: AffiliateReferenceReward, ReferenceID: reward.ID,
		AmountQuota: amountQuota, IdempotencyKey: "reward:create:" + strconv.Itoa(reward.ID),
	}
	if status == AffiliateRewardStatusPending {
		account.PendingQuota += amountQuota
		entry.EntryType = AffiliateLedgerTypeRewardPending
		entry.PendingDeltaQuota = amountQuota
	} else {
		account.AvailableQuota += amountQuota
		entry.EntryType = AffiliateLedgerTypeRewardAvailable
		entry.AvailableDeltaQuota = amountQuota
	}
	account.LifetimeEarnedQuota += amountQuota
	if err := tx.Save(account).Error; err != nil {
		return nil, false, err
	}
	if err := appendAffiliateQuotaLedgerWithTx(tx, account, entry); err != nil {
		return nil, false, err
	}
	if status == AffiliateRewardStatusAvailable {
		if err := transferAvailableAffiliateRewardWithTx(tx, reward.ID); err != nil {
			return nil, false, err
		}
	}
	return &reward, true, nil
}

func bindAffiliateReferralWithTx(tx *gorm.DB, invitee *User, inviterID int, issueRegistrationRewards bool) error {
	if inviterID <= 0 || inviterID == invitee.Id {
		return nil
	}
	var existing AffiliateReferral
	if err := tx.Where("invitee_user_id = ?", invitee.Id).First(&existing).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var inviter User
	if err := tx.Where("id = ?", inviterID).First(&inviter).Error; err != nil {
		return err
	}
	if err := ensureAffiliateCodeWithTx(tx, &inviter); err != nil {
		return err
	}
	relation := AffiliateReferral{
		InviteeUserID: invitee.Id, InviterUserID: inviterID, CodeSnapshot: inviter.AffCode,
		Status: AffiliateReferralStatusBound, BoundAt: common.GetTimestamp(),
	}
	if err := tx.Create(&relation).Error; err != nil {
		return err
	}
	if err := tx.Model(&User{}).Where("id = ?", inviterID).Update("aff_count", gorm.Expr("aff_count + 1")).Error; err != nil {
		return err
	}
	// Registration rewards were removed from the current product behavior. Keep
	// the parameter for call-site compatibility with the existing registration
	// transaction, but do not create inviter or invitee rewards here.
	_ = issueRegistrationRewards
	return nil
}

func initializeAffiliateUserWithTx(tx *gorm.DB, user *User, inviterID int, issueRegistrationRewards bool) error {
	if err := ensureAffiliateCodeWithTx(tx, user); err != nil {
		return err
	}
	if _, err := ensureAffiliateQuotaAccountWithTx(tx, user.Id); err != nil {
		return err
	}
	return bindAffiliateReferralWithTx(tx, user, inviterID, issueRegistrationRewards)
}

func EnsureAffiliateProfile(userID int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		return initializeAffiliateUserWithTx(tx, &user, user.InviterId, false)
	})
}

func topUpPaidCents(topUp *TopUp) (int64, error) {
	if topUp == nil {
		return 0, ErrAffiliateAmountInvalid
	}
	if topUp.PaidCents > 0 {
		return topUp.PaidCents, nil
	}
	if topUp.Money <= 0 {
		return 0, ErrAffiliateAmountInvalid
	}
	amount := decimal.NewFromFloat(topUp.Money).Mul(decimal.NewFromInt(100)).Round(0)
	max := decimal.NewFromInt(int64(^uint64(0) >> 1))
	if amount.LessThanOrEqual(decimal.Zero) || amount.GreaterThan(max) {
		return 0, ErrAffiliateAmountInvalid
	}
	return amount.IntPart(), nil
}

func calculateAffiliateCashbackQuota(baseCents int64, rule AffiliateEffectiveRule) (int64, error) {
	if rule.RewardMode == operation_setting.AffiliateRewardModeFixed {
		return rule.FixedRewardQuota, nil
	}
	if baseCents <= 0 || rule.RewardRateBps <= 0 {
		return 0, nil
	}
	exchangeRate := operation_setting.USDExchangeRate
	if exchangeRate <= 0 {
		exchangeRate = 1
	}
	quotaValue := decimal.NewFromInt(baseCents).
		Div(decimal.NewFromInt(100)).
		Div(decimal.NewFromFloat(exchangeRate)).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Mul(decimal.NewFromInt(rule.RewardRateBps)).
		Div(decimal.NewFromInt(10_000))
	quota, clamp := common.QuotaFromDecimalChecked(quotaValue)
	if clamp != nil {
		return 0, clamp
	}
	return int64(quota), nil
}

func qualifyAffiliateTopUpWithTx(tx *gorm.DB, topUp *TopUp) error {
	if topUp == nil || topUp.Status != common.TopUpStatusSuccess || topUp.CompletionSource != TopUpCompletionSourceOnlineWallet {
		return nil
	}
	var relation AffiliateReferral
	if err := lockForUpdate(tx).Where("invitee_user_id = ?", topUp.UserId).First(&relation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var existing AffiliateTopUpEvent
	if err := tx.Where("top_up_id = ?", topUp.Id).First(&existing).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	paidCents, err := topUpPaidCents(topUp)
	if err != nil {
		return err
	}
	completedAt := topUp.CompleteTime
	if completedAt <= 0 {
		completedAt = common.GetTimestamp()
	}
	if completedAt < relation.BoundAt {
		return nil
	}
	var invitee User
	if err := tx.Select("email").Where("id = ?", relation.InviteeUserID).First(&invitee).Error; err != nil {
		return err
	}
	event := AffiliateTopUpEvent{
		ReferralID: relation.ID, TopUpID: topUp.Id, InviterUserID: relation.InviterUserID,
		InviteeUserID: relation.InviteeUserID,
		MaskedEmail:   maskAffiliateEmail(invitee.Email, relation.InviteeUserID), PaidCents: paidCents,
		CumulativePaidCents: relation.CumulativePaidCents, Qualified: relation.Status == AffiliateReferralStatusQualified,
		CompletedAt: completedAt,
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "top_up_id"}},
		DoNothing: true,
	}).Create(&event)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	var inviter User
	if err := tx.Select("id", "status").Where("id = ?", relation.InviterUserID).First(&inviter).Error; err != nil {
		return err
	}
	if inviter.Status != common.UserStatusEnabled {
		return nil
	}
	campaign, err := activeAffiliateCampaignWithTx(tx, completedAt)
	if err != nil {
		return err
	}
	if campaign != nil {
		return settleAffiliateCampaignTopUpWithTx(tx, campaign, topUp, &relation, &event, paidCents)
	}
	rule, _, err := effectiveAffiliateRuleWithTx(tx, relation.InviterUserID)
	if err != nil {
		return err
	}
	if !rule.Enabled {
		return nil
	}
	baseCents := paidCents
	firstQualified := relation.Status != AffiliateReferralStatusQualified
	if firstQualified {
		relation.CumulativePaidCents += paidCents
		event.CumulativePaidCents = relation.CumulativePaidCents
		if relation.CumulativePaidCents < rule.MinimumTopUpCents {
			if err := tx.Save(&relation).Error; err != nil {
				return err
			}
			return tx.Save(&event).Error
		}
		now := common.GetTimestamp()
		relation.Status = AffiliateReferralStatusQualified
		relation.QualifiedAt = now
		relation.QualifyingTopUpID = &topUp.Id
		event.Qualified = true
		baseCents = relation.CumulativePaidCents
	} else if rule.CashbackFrequency != operation_setting.AffiliateCashbackFrequencyEveryTopUp {
		return tx.Save(&event).Error
	}
	rewardQuota, err := calculateAffiliateCashbackQuota(baseCents, rule)
	if err != nil {
		return err
	}
	if rewardQuota <= 0 {
		if err := tx.Save(&relation).Error; err != nil {
			return err
		}
		return tx.Save(&event).Error
	}
	version, err := ensureAffiliateRuleVersionWithTx(tx, relation.InviterUserID, rule)
	if err != nil {
		return err
	}
	reward, _, err := createAffiliateRewardWithTx(tx, &relation, rule, version.ID,
		AffiliateRewardTypeCashback, relation.InviterUserID, rewardQuota, &topUp.Id,
		paidCents, event.CumulativePaidCents, fmt.Sprintf("cashback:%d:%d", relation.ID, topUp.Id))
	if err != nil {
		return err
	}
	event.RewardID = &reward.ID
	relation.CashbackRewardedQuota += rewardQuota
	if err := tx.Save(&relation).Error; err != nil {
		return err
	}
	return tx.Save(&event).Error
}

// Invalidate only after committing wallet changes, so a cache miss cannot
// repopulate the previous balance from an uncommitted transaction.
func invalidateAffiliateUserCache(userID int) {
	if err := InvalidateUserCache(userID); err != nil {
		common.SysError(fmt.Sprintf("invalidate affiliate wallet cache for user %d: %v", userID, err))
	}
}

func invalidateAffiliateTopUpUserCaches(inviteeID int) {
	invalidateAffiliateUserCache(inviteeID)
	var relation AffiliateReferral
	err := DB.Select("inviter_user_id").Where("invitee_user_id = ?", inviteeID).First(&relation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	if err != nil {
		common.SysError(fmt.Sprintf("find affiliate wallet recipient for user %d: %v", inviteeID, err))
		return
	}
	invalidateAffiliateUserCache(relation.InviterUserID)
}

func ProcessAffiliateTopUp(topUpID int) error {
	var topUp TopUp
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ?", topUpID).First(&topUp).Error; err != nil {
			return err
		}
		if topUp.Status != common.TopUpStatusSuccess {
			return ErrTopUpStatusInvalid
		}
		return qualifyAffiliateTopUpWithTx(tx, &topUp)
	})
	if err == nil {
		invalidateAffiliateTopUpUserCaches(topUp.UserId)
	}
	return err
}

func lockAffiliateRewardWithRecipient(tx *gorm.DB, rewardID int) (*AffiliateReward, error) {
	var identity AffiliateReward
	if err := tx.Select("recipient_user_id").Where("id = ?", rewardID).First(&identity).Error; err != nil {
		return nil, err
	}
	var recipient User
	if err := lockForUpdate(tx).Select("id").Where("id = ?", identity.RecipientUserID).First(&recipient).Error; err != nil {
		return nil, err
	}
	var reward AffiliateReward
	if err := lockForUpdate(tx).Where("id = ?", rewardID).First(&reward).Error; err != nil {
		return nil, err
	}
	return &reward, nil
}

func transferAvailableAffiliateRewardWithTx(tx *gorm.DB, rewardID int) error {
	reward, err := lockAffiliateRewardWithRecipient(tx, rewardID)
	if err != nil {
		return err
	}
	if reward.Status != AffiliateRewardStatusAvailable {
		return nil
	}
	amountQuota := reward.ActualQuota - reward.AdjustedQuota - reward.TransferredQuota
	if amountQuota <= 0 {
		return nil
	}
	// The recipient and reward stay locked through the wallet and ledger writes.
	// A later increase advances from the previous transferred amount, so retries
	// cannot collide with the reward's original automatic transfer.
	requestKey := fmt.Sprintf("auto-reward:%d:%d", reward.ID, reward.TransferredQuota)
	var user User
	if err := lockForUpdate(tx).Select("id", "quota").Where("id = ?", reward.RecipientUserID).First(&user).Error; err != nil {
		return err
	}
	account, err := ensureAffiliateQuotaAccountWithTx(tx, reward.RecipientUserID)
	if err != nil {
		return err
	}
	if account.AvailableQuota < amountQuota {
		return ErrAffiliateBalance
	}
	quotaAfter := int64(user.Quota) + amountQuota
	if quotaAfter > int64(common.MaxQuota) {
		// Keep the reward available until the wallet has room. A cashback
		// transfer must not reject an otherwise successful customer payment.
		return nil
	}
	transfer := AffiliateBalanceTransfer{
		UserID: user.Id, RequestKey: requestKey, AmountQuota: amountQuota,
		AffiliateBalanceBefore: account.AvailableQuota,
		AffiliateBalanceAfter:  account.AvailableQuota - amountQuota,
		UserQuotaBefore:        user.Quota, UserQuotaAfter: int(quotaAfter),
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "request_key"}},
		DoNothing: true,
	}).Create(&transfer)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	reward.TransferredQuota += amountQuota
	if reward.TransferredQuota+reward.AdjustedQuota >= reward.ActualQuota {
		reward.Status = AffiliateRewardStatusTransferred
	}
	account.AvailableQuota -= amountQuota
	account.TransferredQuota += amountQuota
	if err := tx.Save(reward).Error; err != nil {
		return err
	}
	if err := tx.Save(account).Error; err != nil {
		return err
	}
	if err := tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", int(quotaAfter)).Error; err != nil {
		return err
	}
	return appendAffiliateQuotaLedgerWithTx(tx, account, AffiliateQuotaLedger{
		EntryType: AffiliateLedgerTypeBalanceTransfer, ReferenceType: AffiliateReferenceTransfer,
		ReferenceID: transfer.ID, AmountQuota: amountQuota,
		AvailableDeltaQuota: -amountQuota, TransferredDeltaQuota: amountQuota,
		IdempotencyKey: "balance-transfer:" + strconv.Itoa(transfer.ID),
	})
}

func releaseAffiliateRewardWithTx(tx *gorm.DB, rewardID int, now int64) (bool, error) {
	reward, err := lockAffiliateRewardWithRecipient(tx, rewardID)
	if err != nil {
		return false, err
	}
	if reward.Status != AffiliateRewardStatusPending || reward.AvailableAt > now {
		return false, nil
	}
	account, err := ensureAffiliateQuotaAccountWithTx(tx, reward.RecipientUserID)
	if err != nil {
		return false, err
	}
	remaining := reward.ActualQuota - reward.AdjustedQuota - reward.TransferredQuota
	if remaining < 0 || account.PendingQuota < remaining {
		return false, ErrAffiliateBalance
	}
	account.PendingQuota -= remaining
	account.AvailableQuota += remaining
	reward.Status = AffiliateRewardStatusAvailable
	if err := tx.Save(account).Error; err != nil {
		return false, err
	}
	if err := tx.Save(reward).Error; err != nil {
		return false, err
	}
	if err := appendAffiliateQuotaLedgerWithTx(tx, account, AffiliateQuotaLedger{
		EntryType: AffiliateLedgerTypeRewardAvailable, ReferenceType: AffiliateReferenceReward,
		ReferenceID: reward.ID, AmountQuota: remaining, PendingDeltaQuota: -remaining,
		AvailableDeltaQuota: remaining, IdempotencyKey: "reward:release:" + strconv.Itoa(reward.ID),
	}); err != nil {
		return false, err
	}
	return true, nil
}

// ReleaseDueAffiliateRewards processes one page after afterID. The returned
// cursor advances for every inspected reward, including wallet-full rewards,
// and is unchanged when no rows remain.
func ReleaseDueAffiliateRewards(now int64, afterID int, limit int) (int, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	var rewards []AffiliateReward
	if err := DB.Select("id", "recipient_user_id").Model(&AffiliateReward{}).
		Where("id > ?", afterID).
		Where("(status = ? AND available_at <= ?) OR (status = ? AND actual_quota - adjusted_quota - transferred_quota > 0)", AffiliateRewardStatusPending, now, AffiliateRewardStatusAvailable).
		Order("id asc").Limit(limit).Find(&rewards).Error; err != nil {
		return 0, afterID, err
	}
	released := 0
	for _, reward := range rewards {
		id := reward.ID
		changed := false
		err := DB.Transaction(func(tx *gorm.DB) error {
			var err error
			changed, err = releaseAffiliateRewardWithTx(tx, id, now)
			if err != nil {
				return err
			}
			return transferAvailableAffiliateRewardWithTx(tx, id)
		})
		if err != nil {
			return released, afterID, err
		}
		afterID = id
		invalidateAffiliateUserCache(reward.RecipientUserID)
		if changed {
			released++
		}
	}
	return released, afterID, nil
}

func releaseDueAffiliateRewardsForUser(userID int, now int64) error {
	for afterID := 0; ; {
		var ids []int
		if err := DB.Model(&AffiliateReward{}).
			Where("id > ?", afterID).
			Where("recipient_user_id = ? AND ((status = ? AND available_at <= ?) OR (status = ? AND actual_quota - adjusted_quota - transferred_quota > 0))", userID, AffiliateRewardStatusPending, now, AffiliateRewardStatusAvailable).
			Order("id asc").Limit(500).Pluck("id", &ids).Error; err != nil {
			return err
		}
		for _, id := range ids {
			if err := DB.Transaction(func(tx *gorm.DB) error {
				if _, err := releaseAffiliateRewardWithTx(tx, id, now); err != nil {
					return err
				}
				return transferAvailableAffiliateRewardWithTx(tx, id)
			}); err != nil {
				return err
			}
			afterID = id
			invalidateAffiliateUserCache(userID)
		}
		if len(ids) < 500 {
			return nil
		}
	}
}

func GetAffiliateQuotaAccount(userID int) (*AffiliateQuotaAccount, error) {
	if err := EnsureAffiliateProfile(userID); err != nil {
		return nil, err
	}
	if err := releaseDueAffiliateRewardsForUser(userID, common.GetTimestamp()); err != nil {
		return nil, err
	}
	var account AffiliateQuotaAccount
	if err := DB.Where("user_id = ?", userID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func GetAffiliateSummary(userID int) (*AffiliateSummary, error) {
	account, err := GetAffiliateQuotaAccount(userID)
	if err != nil {
		return nil, err
	}
	cashAccount, err := GetAffiliateCashAccount(userID)
	if err != nil {
		return nil, err
	}
	campaign, err := GetAffiliateCampaign()
	if err != nil {
		return nil, err
	}
	var user User
	if err := DB.Select("aff_code").Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	rule, err := GetAffiliateEffectiveRule(userID)
	if err != nil {
		return nil, err
	}
	var referralCount, qualifiedCount int64
	if err := DB.Model(&AffiliateReferral{}).Where("inviter_user_id = ?", userID).Count(&referralCount).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&AffiliateReferral{}).Where("inviter_user_id = ? AND status = ?", userID, AffiliateReferralStatusQualified).Count(&qualifiedCount).Error; err != nil {
		return nil, err
	}
	var nextAvailableAt int64
	if err := DB.Model(&AffiliateReward{}).
		Where("recipient_user_id = ? AND status = ?", userID, AffiliateRewardStatusPending).
		Select("COALESCE(MIN(available_at), 0)").Scan(&nextAvailableAt).Error; err != nil {
		return nil, err
	}
	var cashNextAvailableAt int64
	if err := DB.Model(&AffiliateCashReward{}).
		Where("inviter_user_id = ? AND status = ?", userID, AffiliateRewardStatusPending).
		Select("COALESCE(MIN(available_at), 0)").Scan(&cashNextAvailableAt).Error; err != nil {
		return nil, err
	}
	if nextAvailableAt == 0 || (cashNextAvailableAt > 0 && cashNextAvailableAt < nextAvailableAt) {
		nextAvailableAt = cashNextAvailableAt
	}
	var lifetimeCampaignBonusQuota int64
	if err := DB.Model(&AffiliateQuotaGrant{}).
		Where("user_id = ? AND status = ?", userID, AffiliateRewardStatusCredited).
		Select("COALESCE(SUM(bonus_quota), 0)").Scan(&lifetimeCampaignBonusQuota).Error; err != nil {
		return nil, err
	}
	if campaign.Enabled {
		rule.Enabled = true
		rule.RewardMode = operation_setting.AffiliateRewardModePercentage
		rule.CashbackFrequency = operation_setting.AffiliateCashbackFrequencyEveryTopUp
		rule.RewardRateBps = campaign.InviterCashbackRateBps
	}
	return &AffiliateSummary{
		Enabled: rule.Enabled, ReferralCode: user.AffCode, Currency: "CNY",
		ReferralCount: referralCount, QualifiedCount: qualifiedCount,
		NextAvailableAt: nextAvailableAt, LifetimeCampaignBonusQuota: lifetimeCampaignBonusQuota,
		Rule: rule, Account: *account,
		CashAccount: *cashAccount, Campaign: *campaign,
	}, nil
}

func CreateAffiliateBalanceTransfer(userID int, amountQuota int64, requestKey string) (*AffiliateBalanceTransfer, error) {
	requestKey = strings.TrimSpace(requestKey)
	if amountQuota <= 0 || amountQuota > int64(common.MaxQuota) || requestKey == "" || len(requestKey) > 64 {
		return nil, ErrAffiliateAmountInvalid
	}
	if err := releaseDueAffiliateRewardsForUser(userID, common.GetTimestamp()); err != nil {
		return nil, err
	}
	var transfer AffiliateBalanceTransfer
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND request_key = ?", userID, requestKey).First(&transfer).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var user User
		if err := lockForUpdate(tx).Select("id", "quota").Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		var rewards []AffiliateReward
		if err := lockForUpdate(tx).
			Where("recipient_user_id = ? AND status = ? AND actual_quota - adjusted_quota - transferred_quota > 0", userID, AffiliateRewardStatusAvailable).
			Order("available_at asc, id asc").Find(&rewards).Error; err != nil {
			return err
		}
		account, err := ensureAffiliateQuotaAccountWithTx(tx, userID)
		if err != nil {
			return err
		}
		if account.AvailableQuota < amountQuota {
			return ErrAffiliateBalance
		}
		quotaAfter := int64(user.Quota) + amountQuota
		if quotaAfter > int64(common.MaxQuota) {
			return ErrAffiliateAmountInvalid
		}
		transfer = AffiliateBalanceTransfer{
			UserID: userID, RequestKey: requestKey, AmountQuota: amountQuota,
			AffiliateBalanceBefore: account.AvailableQuota,
			AffiliateBalanceAfter:  account.AvailableQuota - amountQuota,
			UserQuotaBefore:        user.Quota, UserQuotaAfter: int(quotaAfter),
		}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "request_key"}},
			DoNothing: true,
		}).Create(&transfer)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return tx.Where("user_id = ? AND request_key = ?", userID, requestKey).First(&transfer).Error
		}
		remaining := amountQuota
		for index := range rewards {
			available := rewards[index].ActualQuota - rewards[index].AdjustedQuota - rewards[index].TransferredQuota
			allocated := available
			if allocated > remaining {
				allocated = remaining
			}
			rewards[index].TransferredQuota += allocated
			if rewards[index].TransferredQuota+rewards[index].AdjustedQuota >= rewards[index].ActualQuota {
				rewards[index].Status = AffiliateRewardStatusTransferred
			}
			if err := tx.Save(&rewards[index]).Error; err != nil {
				return err
			}
			remaining -= allocated
			if remaining == 0 {
				break
			}
		}
		if remaining != 0 {
			return ErrAffiliateBalance
		}
		account.AvailableQuota -= amountQuota
		account.TransferredQuota += amountQuota
		if err := tx.Save(account).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", userID).Update("quota", int(quotaAfter)).Error; err != nil {
			return err
		}
		return appendAffiliateQuotaLedgerWithTx(tx, account, AffiliateQuotaLedger{
			EntryType: AffiliateLedgerTypeBalanceTransfer, ReferenceType: AffiliateReferenceTransfer,
			ReferenceID: transfer.ID, AmountQuota: amountQuota, AvailableDeltaQuota: -amountQuota,
			TransferredDeltaQuota: amountQuota, IdempotencyKey: "balance-transfer:" + strconv.Itoa(transfer.ID),
		})
	})
	if err != nil {
		return nil, err
	}
	_ = InvalidateUserCache(userID)
	return &transfer, nil
}

// CreateAffiliateRewardBalanceTransfer transfers one cashback reward in full.
// The legacy amount-based transfer remains available for older clients.
func CreateAffiliateRewardBalanceTransfer(userID int, rewardID int, requestKey string) (*AffiliateBalanceTransfer, error) {
	requestKey = strings.TrimSpace(requestKey)
	if userID <= 0 || rewardID <= 0 || requestKey == "" || len(requestKey) > 64 {
		return nil, ErrAffiliateAmountInvalid
	}
	if err := releaseDueAffiliateRewardsForUser(userID, common.GetTimestamp()); err != nil {
		return nil, err
	}
	var transfer AffiliateBalanceTransfer
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND request_key = ?", userID, requestKey).First(&transfer).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var user User
		if err := lockForUpdate(tx).Select("id", "quota").Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		reward, err := lockAffiliateRewardWithRecipient(tx, rewardID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAffiliateRewardNotFound
			}
			return err
		}
		if reward.InviterUserID != userID || reward.RewardType != AffiliateRewardTypeCashback {
			return ErrAffiliateRewardNotFound
		}
		if reward.Status != AffiliateRewardStatusAvailable || reward.AvailableAt > common.GetTimestamp() {
			return ErrAffiliateBalance
		}
		amountQuota := reward.ActualQuota - reward.AdjustedQuota - reward.TransferredQuota
		if amountQuota <= 0 {
			return ErrAffiliateBalance
		}
		account, err := ensureAffiliateQuotaAccountWithTx(tx, userID)
		if err != nil {
			return err
		}
		if account.AvailableQuota < amountQuota {
			return ErrAffiliateBalance
		}
		quotaAfter := int64(user.Quota) + amountQuota
		if quotaAfter > int64(common.MaxQuota) {
			return ErrAffiliateAmountInvalid
		}
		transfer = AffiliateBalanceTransfer{
			UserID: userID, RequestKey: requestKey, AmountQuota: amountQuota,
			AffiliateBalanceBefore: account.AvailableQuota,
			AffiliateBalanceAfter:  account.AvailableQuota - amountQuota,
			UserQuotaBefore:        user.Quota, UserQuotaAfter: int(quotaAfter),
		}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "request_key"}},
			DoNothing: true,
		}).Create(&transfer)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return tx.Where("user_id = ? AND request_key = ?", userID, requestKey).First(&transfer).Error
		}
		reward.TransferredQuota += amountQuota
		reward.Status = AffiliateRewardStatusTransferred
		if err := tx.Save(reward).Error; err != nil {
			return err
		}
		account.AvailableQuota -= amountQuota
		account.TransferredQuota += amountQuota
		if err := tx.Save(account).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", userID).Update("quota", int(quotaAfter)).Error; err != nil {
			return err
		}
		return appendAffiliateQuotaLedgerWithTx(tx, account, AffiliateQuotaLedger{
			EntryType: AffiliateLedgerTypeBalanceTransfer, ReferenceType: AffiliateReferenceTransfer,
			ReferenceID: transfer.ID, AmountQuota: amountQuota, AvailableDeltaQuota: -amountQuota,
			TransferredDeltaQuota: amountQuota, IdempotencyKey: "balance-transfer:" + strconv.Itoa(transfer.ID),
		})
	})
	if err != nil {
		return nil, err
	}
	_ = InvalidateUserCache(userID)
	return &transfer, nil
}

func GetAffiliateBalanceTransfers(userID int, startIdx int, limit int) ([]AffiliateBalanceTransfer, int64, error) {
	var items []AffiliateBalanceTransfer
	var total int64
	query := DB.Model(&AffiliateBalanceTransfer{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id desc").Offset(startIdx).Limit(limit).Find(&items).Error
	return items, total, err
}

func maskAffiliateEmail(email string, userID int) string {
	email = strings.TrimSpace(email)
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Sprintf("user ****%04d", userID%10_000)
	}
	local := []rune(parts[0])
	switch len(local) {
	case 1:
		local = []rune("*")
	case 2:
		local = append(local[:1], '*')
	default:
		local = append(append(append([]rune{}, local[:1]...), []rune("***")...), local[len(local)-1])
	}
	return string(local) + "@" + parts[1]
}

func affiliateTopUpTimeRange(keyword string) (int64, int64, bool) {
	formats := []struct {
		layout string
		step   func(time.Time) time.Time
	}{
		{layout: "2006-01-02 15:04:05", step: func(value time.Time) time.Time { return value.Add(time.Second) }},
		{layout: "2006-01-02 15:04", step: func(value time.Time) time.Time { return value.Add(time.Minute) }},
		{layout: "2006-01-02", step: func(value time.Time) time.Time { return value.AddDate(0, 0, 1) }},
		{layout: "2006-01", step: func(value time.Time) time.Time { return value.AddDate(0, 1, 0) }},
		{layout: "2006", step: func(value time.Time) time.Time { return value.AddDate(1, 0, 0) }},
	}
	for _, format := range formats {
		value, err := time.ParseInLocation(format.layout, keyword, time.Local)
		if err != nil {
			continue
		}
		return value.Unix(), format.step(value).Unix(), true
	}
	return 0, 0, false
}

func GetAffiliateInviteeTopUps(userID int, status string, keyword string, startAt int64, endAt int64, startIdx int, limit int) ([]AffiliateInviteeTopUp, int64, error) {
	return GetAffiliateInviteeTopUpsSorted(userID, status, keyword, "recharge_time_desc", startAt, endAt, startIdx, limit)
}

func GetAffiliateInviteeTopUpsSorted(userID int, status string, keyword string, sort string, startAt int64, endAt int64, startIdx int, limit int) ([]AffiliateInviteeTopUp, int64, error) {
	rule, err := GetAffiliateEffectiveRule(userID)
	if err != nil {
		return nil, 0, err
	}
	if !rule.ShowInviteeTopUps {
		return nil, 0, ErrAffiliateTopUpsHidden
	}
	if err := releaseDueAffiliateRewardsForUser(userID, common.GetTimestamp()); err != nil {
		return nil, 0, err
	}
	if err := releaseDueAffiliateCashRewardsForUser(userID, common.GetTimestamp()); err != nil {
		return nil, 0, err
	}
	type row struct {
		EventID              int
		TopUpID              int
		InviteeUserID        int
		MaskedEmail          string
		BoundAt              int64
		CodeSnapshot         string
		CompletedAt          int64
		PaidCents            int64
		Qualified            bool
		RewardID             *int
		RewardMode           *string
		RewardRateBps        *int64
		FixedRewardQuota     *int64
		ActualQuota          *int64
		AdjustedQuota        *int64
		TransferredQuota     *int64
		AvailableAt          *int64
		RewardStatus         *string
		CashRewardID         *int
		CashRewardRateBps    *int64
		CashbackCents        *int64
		CashTransferredCents *int64
		CashAvailableAt      *int64
		CashRewardStatus     *string
	}
	query := DB.Table("affiliate_top_up_events AS e").
		Select("e.id AS event_id, e.top_up_id, e.invitee_user_id, e.masked_email, r.bound_at, r.code_snapshot, e.completed_at, e.paid_cents, e.qualified, e.reward_id, ar.reward_mode, ar.reward_rate_bps, ar.fixed_reward_quota, ar.actual_quota, ar.adjusted_quota, ar.transferred_quota, ar.available_at, ar.status AS reward_status, e.cash_reward_id, acr.reward_rate_bps AS cash_reward_rate_bps, acr.cashback_cents, acr.transferred_cents AS cash_transferred_cents, acr.available_at AS cash_available_at, acr.status AS cash_reward_status").
		Joins("JOIN affiliate_referrals AS r ON r.id = e.referral_id").
		Joins("JOIN users AS invitee ON invitee.id = e.invitee_user_id").
		Joins("LEFT JOIN affiliate_rewards AS ar ON ar.id = e.reward_id").
		Joins("LEFT JOIN affiliate_cash_rewards AS acr ON acr.id = e.cash_reward_id").
		Where("e.inviter_user_id = ?", userID)
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		normalizedKeyword := strings.ToLower(keyword)
		pattern := "%" + strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(normalizedKeyword, "!", "!!"), "%", "!%"), "_", "!_") + "%"
		if rangeStart, rangeEnd, ok := affiliateTopUpTimeRange(keyword); ok {
			query = query.Where("(LOWER(e.masked_email) LIKE ? ESCAPE '!' OR LOWER(invitee.email) LIKE ? ESCAPE '!' OR LOWER(r.code_snapshot) LIKE ? ESCAPE '!' OR (e.completed_at >= ? AND e.completed_at < ?))", pattern, pattern, pattern, rangeStart, rangeEnd)
		} else {
			query = query.Where("(LOWER(e.masked_email) LIKE ? ESCAPE '!' OR LOWER(invitee.email) LIKE ? ESCAPE '!' OR LOWER(r.code_snapshot) LIKE ? ESCAPE '!')", pattern, pattern, pattern)
		}
	}
	if startAt > 0 {
		query = query.Where("e.completed_at >= ?", startAt)
	}
	if endAt > 0 {
		query = query.Where("e.completed_at <= ?", endAt)
	}
	switch status {
	case "unqualified":
		query = query.Where("e.qualified = ?", false)
	case AffiliateRewardStatusPending, AffiliateRewardStatusAvailable, AffiliateRewardStatusTransferred, AffiliateRewardStatusAdjusted:
		query = query.Where("(ar.status = ? OR acr.status = ?)", status, status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []row
	order := "e.completed_at desc, e.id desc"
	if sort == "recharge_time_asc" {
		order = "e.completed_at asc, e.id asc"
	}
	if err := query.Order(order).Offset(startIdx).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]AffiliateInviteeTopUp, 0, len(rows))
	for _, current := range rows {
		item := AffiliateInviteeTopUp{
			ID: current.EventID, MaskedEmail: current.MaskedEmail,
			InvitedAt: current.BoundAt, InvitationCode: current.CodeSnapshot,
			TopUpID: current.TopUpID, TopUpAt: current.CompletedAt, PaidCents: current.PaidCents,
			Status: "unqualified",
		}
		if current.Qualified {
			item.Status = AffiliateRewardStatusAvailable
		}
		if current.RewardID != nil {
			item.RewardID = *current.RewardID
			if current.RewardMode != nil {
				item.RewardMode = *current.RewardMode
			}
			if current.RewardRateBps != nil {
				item.RewardRateBps = *current.RewardRateBps
			}
			if current.FixedRewardQuota != nil {
				item.FixedRewardQuota = *current.FixedRewardQuota
			}
			if current.ActualQuota != nil {
				item.RewardQuota = *current.ActualQuota
			}
			if current.AdjustedQuota != nil {
				item.RewardQuota -= *current.AdjustedQuota
			}
			if current.TransferredQuota != nil {
				item.TransferredRewardQuota = *current.TransferredQuota
				item.AvailableRewardQuota = item.RewardQuota - *current.TransferredQuota
			}
			if item.AvailableRewardQuota < 0 {
				item.AvailableRewardQuota = 0
			}
			if current.AvailableAt != nil {
				item.AvailableAt = *current.AvailableAt
			}
			if current.RewardStatus != nil {
				item.Status = *current.RewardStatus
			}
		}
		if current.CashRewardID != nil {
			item.CashRewardID = *current.CashRewardID
			item.RewardMode = operation_setting.AffiliateRewardModePercentage
			if current.CashRewardRateBps != nil {
				item.RewardRateBps = *current.CashRewardRateBps
			}
			if current.CashbackCents != nil {
				item.RewardCents = *current.CashbackCents
				item.AvailableRewardCents = *current.CashbackCents
			}
			if current.CashTransferredCents != nil {
				item.TransferredRewardCents = *current.CashTransferredCents
				item.AvailableRewardCents -= *current.CashTransferredCents
			}
			if item.AvailableRewardCents < 0 {
				item.AvailableRewardCents = 0
			}
			if current.CashAvailableAt != nil {
				item.AvailableAt = *current.CashAvailableAt
			}
			if current.CashRewardStatus != nil {
				item.Status = *current.CashRewardStatus
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}

func GetAffiliateAdminRewards(keyword string, status string, startAt int64, endAt int64, startIdx int, limit int) ([]AffiliateAdminReward, int64, error) {
	query := DB.Table("affiliate_rewards AS ar").
		Select("ar.id, ar.inviter_user_id, inviter.username AS inviter_username, inviter.email AS inviter_email, ar.invitee_user_id, invitee.username AS invitee_username, invitee.email AS invitee_email, r.code_snapshot AS invitation_code, ar.top_up_id, t.trade_no, ar.paid_cents, ar.actual_quota, ar.adjusted_quota, ar.transferred_quota, ar.status, ar.available_at, ar.created_at").
		Joins("JOIN affiliate_referrals AS r ON r.id = ar.referral_id").
		Joins("JOIN users AS inviter ON inviter.id = ar.inviter_user_id").
		Joins("JOIN users AS invitee ON invitee.id = ar.invitee_user_id").
		Joins("LEFT JOIN top_ups AS t ON t.id = ar.top_up_id").
		Where("ar.reward_type = ?", AffiliateRewardTypeCashback)

	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		pattern, err := sanitizeLikePattern(strings.ToLower(keyword))
		if err != nil {
			return nil, 0, err
		}
		pattern = "%" + pattern + "%"
		condition := "LOWER(inviter.username) LIKE ? ESCAPE '!' OR LOWER(inviter.email) LIKE ? ESCAPE '!' OR LOWER(invitee.username) LIKE ? ESCAPE '!' OR LOWER(invitee.email) LIKE ? ESCAPE '!' OR LOWER(inviter.remark) LIKE ? ESCAPE '!'"
		args := []interface{}{pattern, pattern, pattern, pattern, pattern}
		if userID, err := strconv.Atoi(keyword); err == nil && userID > 0 {
			condition += " OR ar.inviter_user_id = ? OR ar.invitee_user_id = ?"
			args = append(args, userID, userID)
		}
		query = query.Where("("+condition+")", args...)
	}

	switch status {
	case AffiliateRewardStatusPending, AffiliateRewardStatusAvailable, AffiliateRewardStatusTransferred, AffiliateRewardStatusAdjusted:
		query = query.Where("ar.status = ?", status)
	}
	if startAt > 0 {
		query = query.Where("ar.created_at >= ?", startAt)
	}
	if endAt > 0 {
		query = query.Where("ar.created_at <= ?", endAt)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []AffiliateAdminReward
	if err := query.Order("ar.created_at desc, ar.id desc").Offset(startIdx).Limit(limit).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func affiliateServerBaseDomain() string {
	serverAddress := strings.TrimSpace(system_setting.ServerAddress)
	if serverAddress == "" {
		return ""
	}
	parsed, err := url.Parse(serverAddress)
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	baseDomain := strings.ToLower(parsed.Hostname())
	if baseDomain == "localhost" || net.ParseIP(baseDomain) != nil {
		return ""
	}
	return baseDomain
}

func normalizeAffiliateDomain(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
	if strings.ContainsAny(value, "/\\?:@") || net.ParseIP(value) != nil {
		return "", ErrAffiliateDomainInvalid
	}
	if !strings.Contains(value, ".") {
		if len(value) > 63 || strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
			return "", ErrAffiliateDomainInvalid
		}
		for _, char := range value {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return "", ErrAffiliateDomainInvalid
			}
		}
		if _, reserved := reservedAffiliateSubdomainLabels[value]; reserved {
			return "", ErrAffiliateDomainInvalid
		}
		if baseDomain := affiliateServerBaseDomain(); baseDomain != "" {
			value += "." + baseDomain
		} else {
			return value, nil
		}
	}
	if len(value) > 253 {
		return "", ErrAffiliateDomainInvalid
	}
	labels := strings.Split(value, ".")
	if len(labels) < 3 {
		return "", ErrAffiliateDomainInvalid
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", ErrAffiliateDomainInvalid
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return "", ErrAffiliateDomainInvalid
			}
		}
	}
	return value, nil
}

var reservedAffiliateSubdomainLabels = map[string]struct{}{
	"admin":  {},
	"api":    {},
	"cdn":    {},
	"static": {},
	"www":    {},
}

func affiliateDomainMatchesServer(value string) bool {
	labels := strings.Split(value, ".")
	if len(labels) == 0 {
		return false
	}
	if _, reserved := reservedAffiliateSubdomainLabels[labels[0]]; reserved {
		return false
	}

	baseDomain := affiliateServerBaseDomain()
	if baseDomain == "" {
		return true
	}
	suffix := "." + baseDomain
	if !strings.HasSuffix(value, suffix) {
		return false
	}
	prefix := strings.TrimSuffix(value, suffix)
	return prefix != "" && !strings.Contains(prefix, ".")
}

// ResolveAffiliateInviterByHost resolves a configured affiliate domain to the
// inviter user ID. Unknown, malformed, disabled, and root hosts are treated as
// an unmatched host so callers can preserve the ordinary registration path.
func ResolveAffiliateInviterByHost(host string) (int, bool, error) {
	normalized, err := normalizeAffiliateDomain(host)
	if err != nil || normalized == "" {
		return 0, false, nil
	}
	if !affiliateDomainMatchesServer(normalized) {
		return 0, false, nil
	}

	var override struct {
		UserID int `gorm:"column:user_id"`
	}
	err = DB.Table("affiliate_user_overrides").
		Select("user_id").
		Where("affiliate_domain = ?", normalized).
		First(&override).Error
	if errors.Is(err, gorm.ErrRecordNotFound) && strings.Contains(normalized, ".") {
		prefix := strings.SplitN(normalized, ".", 2)[0]
		err = DB.Table("affiliate_user_overrides").
			Select("user_id").
			Where("affiliate_domain = ?", prefix).
			First(&override).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}

	var user User
	if err := DB.Select("id", "status").
		Where("id = ? AND deleted_at IS NULL", override.UserID).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if user.Status != common.UserStatusEnabled {
		return 0, false, nil
	}
	return user.Id, true, nil
}

// An override row also records agent membership when all reward fields inherit
// the global settings and no domain has been assigned.
const affiliateUserOverrideConfiguredCondition = "affiliate_user_overrides.id IS NOT NULL"

func hasAffiliateRewardOverrideValues(override *AffiliateUserOverride) bool {
	if override == nil {
		return false
	}
	return override.RewardRateBps != nil ||
		override.MaximumRewardQuota != nil ||
		override.HoldSeconds != nil
}

// Agent-specific settings may override only the active global configuration
// fields. Registration rewards, thresholds, visibility, and transfer limits are
// legacy behavior and cannot be written through the agent API.
func hasAffiliateUserOverrideDisallowedValues(override *AffiliateUserOverride) bool {
	if override == nil {
		return false
	}
	return override.Enabled != nil ||
		override.InviterRewardQuota != nil ||
		override.InviteeRewardQuota != nil ||
		override.RegistrationRewardTrigger != nil ||
		override.RewardMode != nil ||
		override.CashbackFrequency != nil ||
		override.FixedRewardQuota != nil ||
		override.UnlimitedReward != nil ||
		override.MinimumTopUpCents != nil ||
		override.MinimumTransferQuota != nil ||
		override.ShowInviteeTopUps != nil
}

func SaveAffiliateUserOverride(userID int, adminUserID int, override AffiliateUserOverride) (*AffiliateUserOverrideView, error) {
	if err := DB.Transaction(func(tx *gorm.DB) error {
		return saveAffiliateUserOverrideWithTx(tx, userID, adminUserID, override)
	}); err != nil {
		return nil, err
	}
	return GetAffiliateUserOverrideView(userID)
}

type AffiliateAgentUpdate struct {
	UserID         int                   `json:"user_id"`
	Override       AffiliateUserOverride `json:"override"`
	InviteeUserIDs []int                 `json:"invitee_user_ids"`
}

// SaveAffiliateAgents commits the complete form together. A failed row leaves
// every row unchanged, and the whole save consumes one critical-rate-limit slot.
func SaveAffiliateAgents(adminUserID int, updates []AffiliateAgentUpdate) ([]*AffiliateUserOverrideView, error) {
	if adminUserID <= 0 || len(updates) == 0 || len(updates) > 1000 {
		return nil, ErrAffiliateRuleInvalid
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			if err := saveAffiliateUserOverrideWithTx(tx, update.UserID, adminUserID, update.Override); err != nil {
				return fmt.Errorf("代理人 %d：%w", update.UserID, err)
			}
			if len(update.InviteeUserIDs) > 100 {
				return ErrAffiliateRuleInvalid
			}
			for _, inviteeID := range update.InviteeUserIDs {
				if err := bindAffiliateInviteeWithTx(tx, update.UserID, inviteeID); err != nil {
					return fmt.Errorf("被邀请者 %d：%w", inviteeID, err)
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	views := make([]*AffiliateUserOverrideView, 0, len(updates))
	for _, update := range updates {
		view, err := GetAffiliateUserOverrideView(update.UserID)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func bindAffiliateInviteeWithTx(tx *gorm.DB, inviterID int, inviteeID int) error {
	if inviterID <= 0 || inviteeID <= 0 || inviterID == inviteeID {
		return errors.New("邀请者与被邀请者必须是不同的有效用户")
	}
	var invitee User
	if err := lockForUpdate(tx).Where("id = ?", inviteeID).First(&invitee).Error; err != nil {
		return err
	}
	if invitee.InviterId != 0 && invitee.InviterId != inviterID {
		return errors.New("用户已绑定其他邀请者")
	}
	var referral AffiliateReferral
	err := tx.Where("invitee_user_id = ?", inviteeID).First(&referral).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && referral.InviterUserID != inviterID {
		return errors.New("用户已绑定其他邀请者")
	}
	if invitee.InviterId == inviterID {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := bindAffiliateReferralWithTx(tx, &invitee, inviterID, false); err != nil {
			return err
		}
	}
	return tx.Model(&invitee).Update("inviter_id", inviterID).Error
}

func saveAffiliateUserOverrideWithTx(tx *gorm.DB, userID int, adminUserID int, override AffiliateUserOverride) error {
	changeReason := strings.TrimSpace(override.ChangeReason)
	hasDomainUpdate := override.AffiliateDomain != nil
	if override.Remark != nil {
		remark := strings.TrimSpace(*override.Remark)
		if len(remark) > 255 {
			return ErrAffiliateRuleInvalid
		}
		override.Remark = &remark
	}
	if userID <= 0 || adminUserID <= 0 || len(changeReason) > 500 ||
		hasAffiliateUserOverrideDisallowedValues(&override) {
		return ErrAffiliateRuleInvalid
	}
	if override.AffiliateDomain != nil {
		normalizedDomain, err := normalizeAffiliateDomain(*override.AffiliateDomain)
		if err != nil {
			return err
		}
		if normalizedDomain != "" && !affiliateDomainMatchesServer(normalizedDomain) {
			return ErrAffiliateDomainInvalid
		}
		if normalizedDomain == "" {
			override.AffiliateDomain = nil
		} else {
			override.AffiliateDomain = &normalizedDomain
		}
	}
	if hasAffiliateRewardOverrideValues(&override) {
		rule := applyAffiliateOverride(globalAffiliateRule(), &override)
		if err := validateAffiliateRule(rule); err != nil {
			return err
		}
	}
	override.UserID = userID
	override.ID = 0
	override.UpdatedBy = adminUserID
	override.ChangeReason = changeReason
	var user User
	if err := tx.Select("id", "remark").Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	var existing AffiliateUserOverride
	if err := tx.Where("user_id = ?", userID).First(&existing).Error; err == nil {
		if override.RewardRateBps == nil {
			override.RewardRateBps = existing.RewardRateBps
		}
		if override.MaximumRewardQuota == nil {
			override.MaximumRewardQuota = existing.MaximumRewardQuota
		}
		if override.HoldSeconds == nil {
			override.HoldSeconds = existing.HoldSeconds
		}
		if !hasDomainUpdate {
			override.AffiliateDomain = existing.AffiliateDomain
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if override.Remark != nil {
		if err := tx.Model(&User{}).Where("id = ?", userID).Update("remark", *override.Remark).Error; err != nil {
			return err
		}
		override.Remark = nil
	}
	if override.AffiliateDomain != nil {
		var count int64
		if err := tx.Model(&AffiliateUserOverride{}).
			Where("affiliate_domain = ? AND user_id <> ?", *override.AffiliateDomain, userID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrAffiliateDomainTaken
		}
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		UpdateAll: true,
	}).Create(&override).Error
}

func DeleteAffiliateUserOverride(userID int) error {
	if userID <= 0 {
		return ErrAffiliateRuleInvalid
	}
	return DB.Where("user_id = ?", userID).Delete(&AffiliateUserOverride{}).Error
}

func SearchAffiliateUserOverrides(keyword string, startIdx int, limit int) ([]*AffiliateUserOverrideView, int64, error) {
	if startIdx < 0 {
		startIdx = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	query := DB.Table("users").
		Joins("LEFT JOIN affiliate_user_overrides ON affiliate_user_overrides.user_id = users.id").
		Where("users.deleted_at IS NULL")
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		query = query.Where(affiliateUserOverrideConfiguredCondition)
	} else {
		pattern, err := sanitizeLikePattern(strings.ToLower(keyword))
		if err != nil {
			return nil, 0, err
		}
		pattern = "%" + pattern + "%"
		condition := "LOWER(users.username) LIKE ? ESCAPE '!' OR LOWER(users.email) LIKE ? ESCAPE '!' OR LOWER(users.display_name) LIKE ? ESCAPE '!' OR LOWER(users.remark) LIKE ? ESCAPE '!' OR LOWER(affiliate_user_overrides.affiliate_domain) LIKE ? ESCAPE '!'"
		args := []interface{}{pattern, pattern, pattern, pattern, pattern}
		if userID, err := strconv.Atoi(keyword); err == nil && userID > 0 {
			condition = "users.id = ? OR " + condition
			args = append([]interface{}{userID}, args...)
		}
		query = query.Where("("+condition+")", args...)
	}
	if keyword == "" {
		query = query.Order("affiliate_user_overrides.updated_at DESC, users.id ASC")
	} else {
		query = query.Order("CASE WHEN " + affiliateUserOverrideConfiguredCondition + " THEN 0 ELSE 1 END, users.username ASC, users.id ASC")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var userIDs []int
	if err := query.Select("users.id").Offset(startIdx).Limit(limit).Pluck("users.id", &userIDs).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*AffiliateUserOverrideView, 0, len(userIDs))
	for _, userID := range userIDs {
		item, err := getAffiliateUserOverrideView(userID)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	inviteInfos, err := GetAffiliateUserInviteInfos(userIDs)
	if err != nil {
		return nil, 0, err
	}
	inviteInfoByUserID := make(map[int]AffiliateUserInviteInfo, len(inviteInfos))
	for _, inviteInfo := range inviteInfos {
		inviteInfoByUserID[inviteInfo.UserID] = inviteInfo
	}
	for _, item := range items {
		item.InviteInfo = inviteInfoByUserID[item.UserID]
	}
	return items, total, nil
}

func GetAffiliateUserInviteInfos(userIDs []int) ([]AffiliateUserInviteInfo, error) {
	if len(userIDs) == 0 {
		return []AffiliateUserInviteInfo{}, nil
	}
	var items []AffiliateUserInviteInfo
	if err := DB.Table("users AS u").
		Select(`u.id AS user_id,
COALESCE(quota_account.lifetime_earned_quota, u.aff_history, 0) AS lifetime_earned_quota,
COALESCE(cash_account.lifetime_earned_cents, 0) AS lifetime_earned_cash_cents,
u.inviter_id AS inviter_user_id,
COALESCE(inviter.username, '') AS inviter_username,
COALESCE(inviter.email, '') AS inviter_email`).
		Joins("LEFT JOIN affiliate_quota_accounts AS quota_account ON quota_account.user_id = u.id").
		Joins("LEFT JOIN affiliate_cash_accounts AS cash_account ON cash_account.user_id = u.id").
		Joins("LEFT JOIN users AS inviter ON inviter.id = u.inviter_id").
		Where("u.id IN ?", userIDs).
		Scan(&items).Error; err != nil {
		return nil, err
	}
	var invitedCounts []struct {
		UserID       int
		InvitedCount int64
	}
	if err := DB.Table("users AS invitee").
		Select("COALESCE(NULLIF(referral.inviter_user_id, 0), invitee.inviter_id) AS user_id, COUNT(DISTINCT invitee.id) AS invited_count").
		Joins("LEFT JOIN affiliate_referrals AS referral ON referral.invitee_user_id = invitee.id").
		Where("invitee.inviter_id IN ? OR referral.inviter_user_id IN ?", userIDs, userIDs).
		Group("COALESCE(NULLIF(referral.inviter_user_id, 0), invitee.inviter_id)").
		Scan(&invitedCounts).Error; err != nil {
		return nil, err
	}
	byUserID := make(map[int]*AffiliateUserInviteInfo, len(items))
	for i := range items {
		byUserID[items[i].UserID] = &items[i]
	}
	for _, count := range invitedCounts {
		if info := byUserID[count.UserID]; info != nil {
			info.InvitedCount = count.InvitedCount
		}
	}
	exchangeRate := operation_setting.USDExchangeRate
	if exchangeRate <= 0 {
		exchangeRate = 1
	}
	for i := range items {
		quotaIncome := decimal.NewFromInt(items[i].LifetimeEarnedQuota).
			Div(decimal.NewFromFloat(common.QuotaPerUnit)).
			Mul(decimal.NewFromFloat(exchangeRate)).
			Mul(decimal.NewFromInt(100)).Round(0).IntPart()
		items[i].IncomeCents = quotaIncome + items[i].LifetimeEarnedCashCents
	}
	return items, nil
}

func GetAffiliateUserOverrideView(userID int) (*AffiliateUserOverrideView, error) {
	view, err := getAffiliateUserOverrideView(userID)
	if err != nil {
		return nil, err
	}
	inviteInfos, err := GetAffiliateUserInviteInfos([]int{userID})
	if err != nil {
		return nil, err
	}
	if len(inviteInfos) > 0 {
		view.InviteInfo = inviteInfos[0]
	}
	return view, nil
}

func getAffiliateUserOverrideView(userID int) (*AffiliateUserOverrideView, error) {
	var user User
	if err := DB.Select("id", "username", "email", "remark").Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	global := globalAffiliateRule()
	var override AffiliateUserOverride
	var overridePtr *AffiliateUserOverride
	updatedByUsername := ""
	if err := DB.Where("user_id = ?", userID).First(&override).Error; err == nil {
		overridePtr = &override
		if override.UpdatedBy > 0 {
			var updatedBy User
			if err := DB.Unscoped().Select("username").Where("id = ?", override.UpdatedBy).First(&updatedBy).Error; err == nil {
				updatedByUsername = updatedBy.Username
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	effective := applyAffiliateOverride(global, overridePtr)
	return &AffiliateUserOverrideView{
		UserID: user.Id, Username: user.Username, Email: user.Email, Remark: user.Remark,
		UpdatedByUsername: updatedByUsername,
		Override:          overridePtr, GlobalRule: global, EffectiveRule: effective,
	}, nil
}

func GetAffiliateAdminReferralActivities(inviterUserID int, keyword string, operationType string, startIdx int, limit int) ([]AffiliateAdminReferralActivity, int64, error) {
	if inviterUserID <= 0 {
		return nil, 0, ErrAffiliateRuleInvalid
	}
	if startIdx < 0 {
		startIdx = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if operationType != "" && operationType != AffiliateActivityInvitation && operationType != AffiliateActivityTopUp {
		return nil, 0, ErrAffiliateRuleInvalid
	}

	activityCTE := `WITH activity AS (
  SELECT invitee.id AS id, invitee.id AS invitee_user_id,
         COALESCE(invitee.username, '') AS username, COALESCE(invitee.email, '') AS email,
         '` + AffiliateActivityInvitation + `' AS operation_type,
         COALESCE(NULLIF(referral.bound_at, 0), invitee.created_at) AS occurred_at,
         CAST(0 AS BIGINT) AS topup_amount_cents,
         CAST(0 AS BIGINT) AS cashback_amount_cents
  FROM users AS invitee
  LEFT JOIN affiliate_referrals AS referral ON referral.invitee_user_id = invitee.id
  WHERE invitee.inviter_id = ? OR referral.inviter_user_id = ?

  UNION ALL

  SELECT topup.id AS id, invitee.id AS invitee_user_id,
         COALESCE(invitee.username, '') AS username, COALESCE(invitee.email, '') AS email,
         '` + AffiliateActivityTopUp + `' AS operation_type,
         COALESCE(NULLIF(topup_event.completed_at, 0), NULLIF(topup.complete_time, 0), topup.create_time) AS occurred_at,
         CAST(COALESCE(NULLIF(topup.paid_cents, 0), ROUND(topup.money * 100), 0) AS BIGINT) AS topup_amount_cents,
         CAST(CASE WHEN cash_reward.id IS NOT NULL THEN cash_reward.cashback_cents
                   ELSE COALESCE(ROUND((quota_reward.actual_quota - quota_reward.adjusted_quota)
                     * quota_reward.cny_exchange_rate_snapshot * 100
                     / NULLIF(quota_reward.quota_per_unit_snapshot, 0)), 0)
              END AS BIGINT) AS cashback_amount_cents
  FROM top_ups AS topup
  JOIN users AS invitee ON invitee.id = topup.user_id
  LEFT JOIN affiliate_referrals AS referral ON referral.invitee_user_id = invitee.id
  LEFT JOIN affiliate_top_up_events AS topup_event ON topup_event.top_up_id = topup.id
  LEFT JOIN affiliate_rewards AS quota_reward ON quota_reward.id = topup_event.reward_id
  LEFT JOIN affiliate_cash_rewards AS cash_reward ON cash_reward.id = topup_event.cash_reward_id
  WHERE topup.status = ?
    AND topup.completion_source = ?
    AND COALESCE(NULLIF(topup_event.completed_at, 0), NULLIF(topup.complete_time, 0), topup.create_time) >= CASE
      WHEN referral.inviter_user_id = ? THEN COALESCE(NULLIF(referral.bound_at, 0), invitee.created_at)
      ELSE invitee.created_at
    END
    AND (invitee.inviter_id = ? OR EXISTS (
      SELECT 1 FROM affiliate_referrals AS referral
      WHERE referral.invitee_user_id = invitee.id AND referral.inviter_user_id = ?
    ))
), activity_with_cumulative AS (
  SELECT activity.*,
         SUM(cashback_amount_cents) OVER (
           PARTITION BY invitee_user_id
           ORDER BY occurred_at, CASE WHEN operation_type = '` + AffiliateActivityInvitation + `' THEN 0 ELSE 1 END, id
           ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
         ) AS cumulative_cashback_cents,
         SUM(cashback_amount_cents) OVER (
           PARTITION BY invitee_user_id
         ) AS lifetime_cashback_cents
  FROM activity
)`
	args := []interface{}{
		inviterUserID,
		inviterUserID,
		common.TopUpStatusSuccess,
		TopUpCompletionSourceOnlineWallet,
		inviterUserID,
		inviterUserID,
		inviterUserID,
	}
	filters := make([]string, 0, 2)
	filterArgs := make([]interface{}, 0, 5)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		pattern := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.ToLower(keyword))
		pattern = "%" + pattern + "%"
		filters = append(filters, "(CAST(invitee_user_id AS TEXT) = ? OR LOWER(username) LIKE ? ESCAPE '!' OR LOWER(email) LIKE ? ESCAPE '!')")
		filterArgs = append(filterArgs, keyword, pattern, pattern)
	}
	if operationType != "" {
		filters = append(filters, "operation_type = ?")
		filterArgs = append(filterArgs, operationType)
	}
	whereClause := ""
	if len(filters) > 0 {
		whereClause = " WHERE " + strings.Join(filters, " AND ")
	}

	var total int64
	if err := DB.Raw(activityCTE+" SELECT COUNT(*) FROM activity_with_cumulative"+whereClause, append(args, filterArgs...)...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []AffiliateAdminReferralActivity
	query := activityCTE + " SELECT id, invitee_user_id, username, email, operation_type, topup_amount_cents, cashback_amount_cents, CASE WHEN operation_type = '" + AffiliateActivityInvitation + "' THEN lifetime_cashback_cents ELSE cumulative_cashback_cents END AS cumulative_cashback_cents, occurred_at FROM activity_with_cumulative" + whereClause + " ORDER BY occurred_at DESC, id DESC LIMIT ? OFFSET ?"
	if err := DB.Raw(query, append(append(args, filterArgs...), limit, startIdx)...).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// AdjustAffiliateReward records a deduction for a positive amount and an increase
// for a negative amount. AdjustedQuota keeps the net deduction from ActualQuota.
func AdjustAffiliateReward(rewardID int, adminUserID int, amountQuota int64, reason string, requestKey string) (*AffiliateAdjustment, error) {
	reason = strings.TrimSpace(reason)
	requestKey = strings.TrimSpace(requestKey)
	if rewardID <= 0 || adminUserID <= 0 || amountQuota == 0 || amountQuota < -int64(common.MaxQuota) || amountQuota > int64(common.MaxQuota) || reason == "" || len(reason) > 500 || requestKey == "" || len(requestKey) > 64 {
		return nil, ErrAffiliateAmountInvalid
	}
	var adjustment AffiliateAdjustment
	var recipientUserID int
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("reward_id = ? AND request_key = ?", rewardID, requestKey).First(&adjustment).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		reward, err := lockAffiliateRewardWithRecipient(tx, rewardID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAffiliateRewardNotFound
			}
			return err
		}
		recipientUserID = reward.RecipientUserID
		if reward.RewardType != AffiliateRewardTypeCashback {
			return ErrAffiliateAmountInvalid
		}
		isIncrease := amountQuota < 0
		if isIncrease {
			if reward.ActualQuota-reward.AdjustedQuota-amountQuota > int64(common.MaxQuota) {
				return ErrAffiliateAmountInvalid
			}
			// A fully deducted frozen reward can become pending again. An unlocked
			// reward, including one already transferred, receives the increase now.
			if reward.Status == AffiliateRewardStatusPending || reward.AvailableAt > common.GetTimestamp() {
				reward.Status = AffiliateRewardStatusPending
			} else {
				reward.Status = AffiliateRewardStatusAvailable
			}
		}
		applied := amountQuota
		manual := int64(0)
		if !isIncrease {
			remaining := reward.ActualQuota - reward.AdjustedQuota - reward.TransferredQuota
			if remaining < 0 {
				remaining = 0
			}
			if applied > remaining {
				applied = remaining
			}
			manual = amountQuota - applied
		}
		status := AffiliateAdjustmentStatusApplied
		if manual > 0 {
			status = AffiliateAdjustmentStatusManualRequired
		}
		adjustment = AffiliateAdjustment{
			RewardID: rewardID, RequestKey: requestKey, AdminUserID: adminUserID, RequestedQuota: amountQuota,
			AppliedQuota: applied, PendingManualQuota: manual, Status: status, Reason: reason,
		}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "reward_id"}, {Name: "request_key"}},
			DoNothing: true,
		}).Create(&adjustment)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return tx.Where("reward_id = ? AND request_key = ?", rewardID, requestKey).First(&adjustment).Error
		}
		if applied == 0 {
			return nil
		}
		account, err := ensureAffiliateQuotaAccountWithTx(tx, reward.RecipientUserID)
		if err != nil {
			return err
		}
		entry := AffiliateQuotaLedger{
			EntryType: AffiliateLedgerTypeAdjustment, ReferenceType: AffiliateReferenceAdjustment,
			ReferenceID: adjustment.ID, AmountQuota: -applied,
			IdempotencyKey: "adjustment:" + strconv.Itoa(adjustment.ID),
		}
		if reward.Status == AffiliateRewardStatusPending {
			if account.PendingQuota < applied {
				return ErrAffiliateBalance
			}
			account.PendingQuota -= applied
			entry.PendingDeltaQuota = -applied
		} else {
			if account.AvailableQuota < applied {
				return ErrAffiliateBalance
			}
			account.AvailableQuota -= applied
			entry.AvailableDeltaQuota = -applied
		}
		if isIncrease {
			account.LifetimeEarnedQuota += -applied
		}
		reward.AdjustedQuota += applied
		if reward.AdjustedQuota+reward.TransferredQuota >= reward.ActualQuota {
			reward.Status = AffiliateRewardStatusAdjusted
		}
		if err := tx.Save(account).Error; err != nil {
			return err
		}
		if err := tx.Save(reward).Error; err != nil {
			return err
		}
		if err := appendAffiliateQuotaLedgerWithTx(tx, account, entry); err != nil {
			return err
		}
		if isIncrease && reward.Status == AffiliateRewardStatusAvailable {
			return transferAvailableAffiliateRewardWithTx(tx, reward.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if recipientUserID > 0 {
		invalidateAffiliateUserCache(recipientUserID)
	}
	return &adjustment, nil
}

func UpdateAffiliateSetting(next operation_setting.AffiliateSetting) error {
	// Only the active global fields are writable. Normalize all legacy fields so
	// old option rows cannot re-enable removed registration or transfer rules.
	setting := normalizedAffiliateSetting()
	setting.Enabled = next.Enabled
	setting.RewardRateBps = next.RewardRateBps
	setting.MaximumRewardQuota = next.MaximumRewardQuota
	setting.HoldSeconds = next.HoldSeconds
	setting = normalizeAffiliateSettingValues(setting)

	rule := AffiliateEffectiveRule{
		Enabled:                   setting.Enabled,
		InviterRewardQuota:        setting.InviterRewardQuota,
		InviteeRewardQuota:        setting.InviteeRewardQuota,
		RegistrationRewardTrigger: setting.RegistrationRewardTrigger,
		RewardMode:                setting.RewardMode,
		CashbackFrequency:         setting.CashbackFrequency,
		RewardRateBps:             setting.RewardRateBps,
		FixedRewardQuota:          setting.FixedRewardQuota,
		UnlimitedReward:           setting.UnlimitedReward,
		MaximumRewardQuota:        setting.MaximumRewardQuota,
		MinimumTopUpCents:         setting.MinimumTopUpCents,
		HoldSeconds:               setting.HoldSeconds,
		MinimumTransferQuota:      setting.MinimumTransferQuota,
		ShowInviteeTopUps:         setting.ShowInviteeTopUps,
		Source:                    "global",
	}
	if err := validateAffiliateRule(rule); err != nil {
		return err
	}
	values := map[string]string{
		"affiliate_setting.enabled":                     strconv.FormatBool(setting.Enabled),
		"affiliate_setting.inviter_reward_quota":        strconv.FormatInt(setting.InviterRewardQuota, 10),
		"affiliate_setting.invitee_reward_quota":        strconv.FormatInt(setting.InviteeRewardQuota, 10),
		"affiliate_setting.registration_reward_trigger": setting.RegistrationRewardTrigger,
		"affiliate_setting.reward_mode":                 setting.RewardMode,
		"affiliate_setting.cashback_frequency":          setting.CashbackFrequency,
		"affiliate_setting.reward_rate_bps":             strconv.FormatInt(setting.RewardRateBps, 10),
		"affiliate_setting.fixed_reward_quota":          strconv.FormatInt(setting.FixedRewardQuota, 10),
		"affiliate_setting.unlimited_reward":            strconv.FormatBool(setting.UnlimitedReward),
		"affiliate_setting.maximum_reward_quota":        strconv.FormatInt(setting.MaximumRewardQuota, 10),
		"affiliate_setting.minimum_topup_cents":         strconv.FormatInt(setting.MinimumTopUpCents, 10),
		"affiliate_setting.hold_seconds":                strconv.FormatInt(setting.HoldSeconds, 10),
		"affiliate_setting.minimum_transfer_quota":      strconv.FormatInt(setting.MinimumTransferQuota, 10),
		"affiliate_setting.show_invitee_topups":         strconv.FormatBool(setting.ShowInviteeTopUps),
	}
	return UpdateOptionsBulk(values)
}

func normalizeAffiliateSettingValues(setting operation_setting.AffiliateSetting) operation_setting.AffiliateSetting {
	if setting.RewardRateBps < 0 {
		setting.RewardRateBps = 0
	} else if setting.RewardRateBps > 10_000 {
		setting.RewardRateBps = 10_000
	}
	// The per-invitee cashback cap is retired. Keep the persisted field for
	// compatibility with existing option rows, but always expose the unlimited
	// behavior so legacy values cannot re-enable the removed restriction.
	setting.MaximumRewardQuota = 0
	if setting.HoldSeconds < 0 {
		setting.HoldSeconds = 0
	} else if setting.HoldSeconds > 365*24*60*60 {
		setting.HoldSeconds = 365 * 24 * 60 * 60
	}
	setting.InviterRewardQuota = 0
	setting.InviteeRewardQuota = 0
	setting.RegistrationRewardTrigger = operation_setting.AffiliateRegistrationTriggerDisabled
	setting.RewardMode = operation_setting.AffiliateRewardModePercentage
	setting.CashbackFrequency = operation_setting.AffiliateCashbackFrequencyEveryTopUp
	setting.FixedRewardQuota = 0
	setting.UnlimitedReward = true
	setting.MinimumTopUpCents = 0
	setting.MinimumTransferQuota = 0
	setting.ShowInviteeTopUps = true
	return setting
}

func GetGlobalAffiliateSetting() operation_setting.AffiliateSetting {
	return normalizedAffiliateSetting()
}
