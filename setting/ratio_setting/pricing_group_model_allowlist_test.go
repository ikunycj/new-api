package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePricingGroupModelAllowlist(t *testing.T) {
	ratios := map[string]float64{"default": 1}
	allowlist, err := ParsePricingGroupModelAllowlist(`{"default":["model-b","model-a"]}`, ratios)
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"default": {"model-a", "model-b"}}, allowlist)

	for _, tc := range []struct {
		json string
		want string
	}{
		{`null`, "必须是 JSON 对象"},
		{`{"other":["model-a"]}`, "包含不存在的分组"},
		{`{"auto":["model-a"]}`, "无效的分组名"},
		{`{"default":[]}`, "必须至少选择一个模型"},
		{`{"default":["model-a","model-a"]}`, "包含重复的模型"},
		{`{"default":[" model-a"]}`, "无效的模型名"},
	} {
		t.Run(tc.json, func(t *testing.T) {
			_, err := ParsePricingGroupModelAllowlist(tc.json, ratios)
			require.ErrorContains(t, err, tc.want)
		})
	}
}
