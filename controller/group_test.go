package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedUserGroupsOnlyContainsDefault(t *testing.T) {
	previous := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previous))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"enterprise-pricing":0.5}`))

	assert.Equal(t, []string{"default"}, managedUserGroups())
	assert.False(t, isManagedUserGroup("enterprise-pricing"))
}
