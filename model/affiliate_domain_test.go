package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAffiliateInviterByHostUsesUserOverrideDomain(t *testing.T) {
	configureAffiliateTest(t, nil)
	truncateTables(t)

	inviter := &User{
		Username: "domain-inviter",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(inviter).Error)
	domain := "A.IKUN.LOVE"
	_, err := SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		AffiliateDomain: &domain,
		ChangeReason:    "enable affiliate domain",
	})
	require.NoError(t, err)

	resolvedID, matched, err := ResolveAffiliateInviterByHost("A.IKUN.LOVE.:443")
	require.NoError(t, err)
	assert.True(t, matched)
	assert.Equal(t, inviter.Id, resolvedID)

	invitee := &User{
		Username: "domain-invitee",
		AffCode:  "DOMAIN-INVITEE",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, invitee.Insert(resolvedID))
	var relation AffiliateReferral
	require.NoError(t, DB.Where("invitee_user_id = ?", invitee.Id).First(&relation).Error)
	assert.Equal(t, inviter.Id, relation.InviterUserID)

	resolvedID, matched, err = ResolveAffiliateInviterByHost("ikun.love")
	require.NoError(t, err)
	assert.False(t, matched)
	assert.Zero(t, resolvedID)
}

func TestSaveAffiliateUserOverrideRejectsDuplicateDomain(t *testing.T) {
	configureAffiliateTest(t, nil)
	truncateTables(t)

	first := &User{Username: "domain-first", AffCode: "DOMAIN-FIRST", Status: common.UserStatusEnabled}
	second := &User{Username: "domain-second", AffCode: "DOMAIN-SECOND", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(first).Error)
	require.NoError(t, DB.Create(second).Error)
	domain := "a.ikun.love"
	_, err := SaveAffiliateUserOverride(first.Id, first.Id, AffiliateUserOverride{
		AffiliateDomain: &domain,
		ChangeReason:    "first domain",
	})
	require.NoError(t, err)

	_, err = SaveAffiliateUserOverride(second.Id, second.Id, AffiliateUserOverride{
		AffiliateDomain: &domain,
		ChangeReason:    "duplicate domain",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAffiliateDomainTaken))
}

func TestSaveAffiliateUserOverrideRejectsReservedDomain(t *testing.T) {
	configureAffiliateTest(t, nil)
	truncateTables(t)

	inviter := &User{Username: "reserved-domain-inviter", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(inviter).Error)
	domain := "api.ikun.love"
	_, err := SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		AffiliateDomain: &domain,
		ChangeReason:    "reserved domain",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAffiliateDomainInvalid)
}

func TestSaveAffiliateUserOverrideRejectsDomainOutsideServerAddress(t *testing.T) {
	configureAffiliateTest(t, nil)
	truncateTables(t)

	originalServerAddress := system_setting.ServerAddress
	t.Cleanup(func() { system_setting.ServerAddress = originalServerAddress })
	system_setting.ServerAddress = "https://ikun.love"

	inviter := &User{Username: "outside-domain-inviter", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(inviter).Error)
	domain := "a.example.com"
	_, err := SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		AffiliateDomain: &domain,
		ChangeReason:    "outside domain",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAffiliateDomainInvalid)
}

func TestSaveAffiliateUserOverridePartialUpdatePreservesAgentDomain(t *testing.T) {
	configureAffiliateTest(t, nil)
	truncateTables(t)

	inviter := &User{Username: "partial-agent", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(inviter).Error)
	domain := "partial.ikun.love"
	rate := int64(1250)
	view, err := SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		AffiliateDomain: &domain,
		RewardRateBps:   &rate,
	})
	require.NoError(t, err)
	require.NotNil(t, view.Override)
	assert.Equal(t, domain, *view.Override.AffiliateDomain)
	assert.Equal(t, rate, *view.Override.RewardRateBps)

	updatedRate := int64(2500)
	view, err = SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		RewardRateBps: &updatedRate,
	})
	require.NoError(t, err)
	require.NotNil(t, view.Override)
	assert.Equal(t, domain, *view.Override.AffiliateDomain)
	assert.Equal(t, updatedRate, *view.Override.RewardRateBps)
}

func TestSaveAffiliateUserOverrideAcceptsSubdomainPrefixAndRemark(t *testing.T) {
	configureAffiliateTest(t, nil)
	truncateTables(t)

	originalServerAddress := system_setting.ServerAddress
	t.Cleanup(func() { system_setting.ServerAddress = originalServerAddress })
	system_setting.ServerAddress = "https://ikun.love"

	inviter := &User{Username: "prefix-agent", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(inviter).Error)
	prefix := "a"
	remark := "华东代理"
	view, err := SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		AffiliateDomain: &prefix,
		Remark:          &remark,
	})
	require.NoError(t, err)
	require.NotNil(t, view.Override)
	require.NotNil(t, view.Override.AffiliateDomain)
	assert.Equal(t, "a.ikun.love", *view.Override.AffiliateDomain)
	assert.Equal(t, remark, view.Remark)

	resolvedID, matched, err := ResolveAffiliateInviterByHost("a.ikun.love:443")
	require.NoError(t, err)
	assert.True(t, matched)
	assert.Equal(t, inviter.Id, resolvedID)

	updatedRemark := "华东代理商"
	view, err = SaveAffiliateUserOverride(inviter.Id, inviter.Id, AffiliateUserOverride{
		Remark: &updatedRemark,
	})
	require.NoError(t, err)
	assert.Equal(t, updatedRemark, view.Remark)
	require.NotNil(t, view.Override)
	require.NotNil(t, view.Override.AffiliateDomain)
	assert.Equal(t, "a.ikun.love", *view.Override.AffiliateDomain)
}
