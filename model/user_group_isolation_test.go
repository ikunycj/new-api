package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLegacyUserGroupsRepairsAccountGroupsOnly(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       9801,
		Username: "legacy-account-group",
		Group:    "premium",
		AffCode:  "legacy-account-group",
	}).Error)
	require.NoError(t, DB.Create(&User{
		Id:       9802,
		Username: "default-account-group",
		Group:    DefaultUserGroup,
		AffCode:  "default-account-group",
	}).Error)
	require.NoError(t, DB.Create(&Channel{
		Id:    9803,
		Name:  "pricing-channel",
		Group: "premium",
	}).Error)

	require.NoError(t, normalizeLegacyUserGroups())

	var legacy User
	require.NoError(t, DB.First(&legacy, 9801).Error)
	assert.Equal(t, DefaultUserGroup, legacy.Group)
	var unchanged User
	require.NoError(t, DB.First(&unchanged, 9802).Error)
	assert.Equal(t, DefaultUserGroup, unchanged.Group)
	var channel Channel
	require.NoError(t, DB.First(&channel, 9803).Error)
	assert.Equal(t, "premium", channel.Group)
}

func TestSubscriptionTransitionCannotWritePricingGroupToUser(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       9811,
		Username: "subscription-group-user",
		Group:    DefaultUserGroup,
		AffCode:  "subscription-group-user",
	}).Error)
	plan := &SubscriptionPlan{
		Id:             9812,
		Title:          "Legacy premium plan",
		DurationUnit:   SubscriptionDurationMonth,
		DurationValue:  1,
		UpgradeGroup:   "premium",
		DowngradeGroup: "standard",
	}
	require.NoError(t, DB.Create(plan).Error)

	// The helper accepts a DB handle directly; using it this way also keeps the
	// single-connection in-memory SQLite test database from self-deadlocking
	// while the helper reads the database clock.
	subscription, err := CreateUserSubscriptionFromPlanTx(DB, 9811, plan, "test")
	require.NoError(t, err)
	require.NotNil(t, subscription)
	assert.Equal(t, DefaultUserGroup, subscription.UpgradeGroup)
	assert.Equal(t, DefaultUserGroup, subscription.DowngradeGroup)

	var user User
	require.NoError(t, DB.First(&user, 9811).Error)
	assert.Equal(t, DefaultUserGroup, user.Group)
}
