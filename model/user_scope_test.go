package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserGroupHasToBAccess(t *testing.T) {
	assert.False(t, UserGroupHasToBAccess("default"))
	assert.True(t, UserGroupHasToBAccess("vip"))
	assert.True(t, UserGroupHasToBAccess("enterprise"))
	assert.True(t, UserGroupHasToBAccess(" ENTERPRISE "))
	assert.False(t, UserGroupHasToBAccess("custom"))
}
