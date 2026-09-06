package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckRouteRateLimitOption(t *testing.T) {
	testCases := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{name: "enabled accepts boolean", key: "GlobalApiRateLimitEnabled", value: "true"},
		{name: "enabled rejects invalid boolean", key: "GlobalWebRateLimitEnabled", value: "enabled", wantErr: true},
		{name: "request limit accepts positive value", key: "CriticalRateLimitNum", value: "20"},
		{name: "request limit rejects zero", key: "GlobalApiRateLimitNum", value: "0", wantErr: true},
		{name: "window accepts positive value", key: "GlobalWebRateLimitDuration", value: "180"},
		{name: "window rejects overflow", key: "CriticalRateLimitDuration", value: "100000001", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := CheckRouteRateLimitOption(testCase.key, testCase.value)
			if testCase.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
