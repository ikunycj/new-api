package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetPricingGroupNamesIncludesRoutedGroups(t *testing.T) {
	previousDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	t.Cleanup(func() { DB = previousDB })
	require.NoError(t, db.AutoMigrate(&BillingGroupRoute{}))
	require.NoError(t, db.Create(&BillingGroupRoute{
		BillingGroup: "通用套餐",
		Enabled:      true,
	}).Error)

	groups, err := GetPricingGroupNames()
	require.NoError(t, err)

	assert.Contains(t, groups, "通用套餐")
}
