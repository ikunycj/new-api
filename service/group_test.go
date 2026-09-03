package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTokenGroupCandidates(t *testing.T) {
	tests := []struct {
		name        string
		userGroup   string
		groups      []string
		wantErr     bool
		messagePart string
	}{
		{name: "ordered concrete groups", userGroup: "default", groups: []string{"default", "vip"}},
		{name: "empty list clears custom selection", userGroup: "default", groups: []string{}},
		{name: "empty candidate", userGroup: "default", groups: []string{"default", " "}, wantErr: true, messagePart: "不能为空"},
		{name: "duplicate candidate", userGroup: "default", groups: []string{"default", "default"}, wantErr: true, messagePart: "不能重复"},
		{name: "virtual auto candidate", userGroup: "default", groups: []string{"auto"}, wantErr: true, messagePart: "不能包含 auto"},
		{name: "too many candidates", userGroup: "default", groups: make([]string, MaxTokenGroupCandidates+1), wantErr: true, messagePart: "不能超过"},
		{name: "candidate name too long", userGroup: "default", groups: []string{strings.Repeat("a", MaxTokenGroupNameLength+1)}, wantErr: true, messagePart: "长度"},
		{name: "unusable candidate", userGroup: "default", groups: []string{"hidden"}, wantErr: true, messagePart: "无权访问"},
		{name: "candidate without ratio", userGroup: "orphan", groups: []string{"orphan"}, wantErr: true, messagePart: "无权访问"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTokenGroupCandidates(tt.userGroup, tt.groups)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.messagePart)
		})
	}
}

func TestValidateTokenGroupCandidatesAllowsMaximum(t *testing.T) {
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousEnabled := ratio_setting.PricingGroupEnabled2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(previousEnabled))
	})

	ratios := make(map[string]float64, MaxTokenGroupCandidates)
	groups := make([]string, 0, MaxTokenGroupCandidates)
	for index := 0; index < MaxTokenGroupCandidates; index++ {
		group := fmt.Sprintf("boundary-group-%d", index)
		groups = append(groups, group)
		ratios[group] = 1
	}
	ratioJSON, err := common.Marshal(ratios)
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(string(ratioJSON)))

	require.NoError(t, ValidateTokenGroupCandidates("default", groups))
	tooMany := append(append([]string(nil), groups...), "boundary-group-overflow")
	err = ValidateTokenGroupCandidates("default", tooMany)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不能超过 16 个")
}

func TestValidateTokenGroupAutoRequiresVirtualPermission(t *testing.T) {
	err := ValidateTokenGroup("default", "auto")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无权访问")

	require.NoError(t, ValidateTokenGroupCandidates("default", []string{"default", "vip"}))
}

func TestAccountGroupDoesNotBecomePricingGroup(t *testing.T) {
	const accountGroup = "legacy-account-group-only"
	_, ok := GetUserGroupPricingGroups(accountGroup)[accountGroup]
	assert.False(t, ok)

	err := ValidateTokenGroup(accountGroup, accountGroup)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无权访问")
}

func TestDisabledPricingGroupIsHiddenAndRejectedForTokens(t *testing.T) {
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousEnabled := ratio_setting.PricingGroupEnabled2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(previousEnabled))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(`{"default":true,"vip":false}`))

	groups := GetUserGroupPricingGroups("default")
	assert.Contains(t, groups, "default")
	assert.NotContains(t, groups, "vip")

	err := ValidateTokenGroupCandidates("default", []string{"vip"})
	require.ErrorContains(t, err, "已关闭")
}

func TestNormalizeTokenGroupRetryTimes(t *testing.T) {
	normalized, err := NormalizeTokenGroupRetryTimes(
		[]string{"openai-low", "claude-low"},
		map[string]int{"openai-low": 0, "unused": 7},
	)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{
		"openai-low": 0,
	}, normalized)

	for _, value := range []int{-1, MaxTokenGroupRetryTimes + 1} {
		_, err = NormalizeTokenGroupRetryTimes(
			[]string{"openai-low"},
			map[string]int{"openai-low": value},
		)
		require.Error(t, err)
	}
}
