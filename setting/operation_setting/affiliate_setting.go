package operation_setting

import (
	"github.com/QuantumNous/new-api/setting/config"
)

const (
	// AffiliateRegistrationTriggerDisabled is retained in rule snapshots so the
	// current configuration explicitly records that registration rewards are off.
	AffiliateRegistrationTriggerDisabled = "disabled"

	// Legacy values remain available for historical rule snapshots and old data.
	AffiliateRegistrationTriggerRegistrationSuccess = "registration_success"
	AffiliateRegistrationTriggerFirstQualifiedTopUp = "first_qualified_topup"

	AffiliateRewardModePercentage = "percentage"
	AffiliateRewardModeFixed      = "fixed"

	AffiliateCashbackFrequencyFirstQualified = "first_qualified"
	AffiliateCashbackFrequencyEveryTopUp     = "every_topup"
)

// AffiliateSetting keeps legacy fields so existing option rows and historical
// reward snapshots remain readable. New settings writes normalize the legacy
// rule fields to the current product behavior in model.UpdateAffiliateSetting.
type AffiliateSetting struct {
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
}

var affiliateSetting = AffiliateSetting{
	Enabled:                   true,
	InviterRewardQuota:        0,
	InviteeRewardQuota:        0,
	RegistrationRewardTrigger: AffiliateRegistrationTriggerDisabled,
	RewardMode:                AffiliateRewardModePercentage,
	CashbackFrequency:         AffiliateCashbackFrequencyEveryTopUp,
	RewardRateBps:             500,
	FixedRewardQuota:          0,
	UnlimitedReward:           true,
	MaximumRewardQuota:        0,
	MinimumTopUpCents:         0,
	HoldSeconds:               3 * 24 * 60 * 60,
	MinimumTransferQuota:      0,
	ShowInviteeTopUps:         true,
}

func init() {
	config.GlobalConfig.Register("affiliate_setting", &affiliateSetting)
}

func GetAffiliateSetting() *AffiliateSetting {
	return &affiliateSetting
}
