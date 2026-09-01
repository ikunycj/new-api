package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagedUserGroupsExposeOnlyAccountGroups(t *testing.T) {
	groups := managedUserGroups()

	assert.Equal(t, []string{"default", "vip"}, groups)
	assert.NotContains(t, groups, "toB")
	assert.NotContains(t, groups, "成本套餐")
}

func TestNormalizeManagedUserGroup(t *testing.T) {
	for _, legacy := range []string{"toB", " enterprise "} {
		group, ok := normalizeManagedUserGroup(legacy)
		assert.True(t, ok)
		assert.Equal(t, "vip", group)
	}
	group, ok := normalizeManagedUserGroup("成本套餐")
	assert.False(t, ok)
	assert.Empty(t, group)
}
