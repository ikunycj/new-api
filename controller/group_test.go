package controller

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagedUserGroupsAlwaysExposeToCAndToB(t *testing.T) {
	groups := managedUserGroups()

	assert.Contains(t, groups, "default")
	assert.Contains(t, groups, "toB")
	sorted := slices.Clone(groups)
	slices.Sort(sorted)
	assert.Equal(t, sorted, groups)
}
