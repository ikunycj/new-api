package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFingerprintAliases = `[[" deepseek-v4.1-flash-1017 ","deepseek-flash"],["gpt-6.1-sol","gpt-6.1-astra","gpt-6.2-astra"]]`

func TestFingerprintModelAliasMatching(t *testing.T) {
	aliases, err := parseFingerprintModelAliases(testFingerprintAliases)
	require.NoError(t, err)
	for _, tt := range []struct {
		name       string
		testModel  string
		prediction string
		match      bool
	}{
		{"configured provider alias", "deepseek-v4.1-flash-1017", "DEEPSEEK-FLASH", true},
		{"three names are equivalent", "gpt-6.1-sol", "gpt-6.2-astra", true},
		{"symmetric match", "gpt-6.1-astra", "gpt-6.1-sol", true},
		{"separate groups", "deepseek-flash", "gpt-6.1-sol", false},
		{"prefix is not an alias", "gpt-6.1-sol", "gpt-6.2-astra-preview", false},
		{"unconfigured exact match", "custom-model", "CUSTOM-MODEL", true},
		{"unconfigured mismatch", "custom-model", "other-model", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.match, aliases.sameModel(tt.testModel, tt.prediction))
		})
	}
}

func TestFingerprintModelAliasesRejectInvalidGroups(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value string
		want  string
	}{
		{"not a nested array", `["a","b"]`, "模型名数组的数组"},
		{"null is not an empty array", `null`, "模型名数组的数组"},
		{"single name", `[["a"]]`, "至少需要两个"},
		{"blank name", `[["a"," "]]`, "包含空模型名"},
		{"separator inside a name", `[["a、b","c"]]`, "包含分隔符"},
		{"duplicate ignoring case and spaces", `[["a"," A "]]`, "重复"},
		{"overlapping groups", `[["a","b"],["c"," B "]]`, "同时出现在第 1 组和第 2 组"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFingerprintModelAliases(tt.value)
			require.ErrorContains(t, err, tt.want)
		})
	}
}
