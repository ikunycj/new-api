package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLoadTestOption(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{name: "duration lower bound", key: "loadtest_setting.max_duration_seconds", value: "5"},
		{name: "rps upper bound", key: "loadtest_setting.max_rps", value: "10000"},
		{name: "concurrency upper bound", key: "loadtest_setting.max_concurrency", value: "10000"},
		{name: "output tokens upper bound", key: "loadtest_setting.max_output_tokens", value: "8192"},
		{name: "rps too large", key: "loadtest_setting.max_rps", value: "10001", wantErr: true},
		{name: "rps not integer", key: "loadtest_setting.max_rps", value: "2.5", wantErr: true},
		{name: "concurrency zero", key: "loadtest_setting.max_concurrency", value: "0", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateLoadTestOption(test.key, test.value)
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetLoadTestSettingClampsInvalidRuntimeValues(t *testing.T) {
	original := loadTestSetting
	t.Cleanup(func() { loadTestSetting = original })
	loadTestSetting = LoadTestSetting{
		MaxDurationSeconds: 9999,
		MaxRPS:             0,
		MaxConcurrency:     -1,
	}

	got := GetLoadTestSetting()
	assert.Equal(t, LoadTestHardMaxDurationSeconds, got.MaxDurationSeconds)
	assert.Equal(t, 1, got.MaxRPS)
	assert.Equal(t, 1, got.MaxConcurrency)
}

func TestGetLoadTestSettingClampsOutputTokens(t *testing.T) {
	original := loadTestSetting
	t.Cleanup(func() { loadTestSetting = original })
	loadTestSetting.MaxOutputTokens = 0
	assert.Equal(t, 1, GetLoadTestSetting().MaxOutputTokens)
	loadTestSetting.MaxOutputTokens = LoadTestHardMaxOutputTokens + 1
	assert.Equal(t, LoadTestHardMaxOutputTokens, GetLoadTestSetting().MaxOutputTokens)
}
