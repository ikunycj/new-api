package operation_setting

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

const (
	LoadTestMinDurationSeconds        = 5
	LoadTestDefaultMaxDurationSeconds = 600
	LoadTestHardMaxDurationSeconds    = 3600
	LoadTestDefaultMaxRPS             = 20
	LoadTestHardMaxRPS                = 10_000
	LoadTestDefaultMaxConcurrency     = 10
	LoadTestHardMaxConcurrency        = 10_000
)

type LoadTestSetting struct {
	MaxDurationSeconds int `json:"max_duration_seconds"`
	MaxRPS             int `json:"max_rps"`
	MaxConcurrency     int `json:"max_concurrency"`
}

var loadTestSetting = LoadTestSetting{
	MaxDurationSeconds: LoadTestDefaultMaxDurationSeconds,
	MaxRPS:             LoadTestDefaultMaxRPS,
	MaxConcurrency:     LoadTestDefaultMaxConcurrency,
}

func init() {
	config.GlobalConfig.Register("loadtest_setting", &loadTestSetting)
}

func GetLoadTestSetting() LoadTestSetting {
	setting := loadTestSetting
	if setting.MaxDurationSeconds < LoadTestMinDurationSeconds {
		setting.MaxDurationSeconds = LoadTestMinDurationSeconds
	}
	if setting.MaxDurationSeconds > LoadTestHardMaxDurationSeconds {
		setting.MaxDurationSeconds = LoadTestHardMaxDurationSeconds
	}
	if setting.MaxRPS < 1 {
		setting.MaxRPS = 1
	}
	if setting.MaxRPS > LoadTestHardMaxRPS {
		setting.MaxRPS = LoadTestHardMaxRPS
	}
	if setting.MaxConcurrency < 1 {
		setting.MaxConcurrency = 1
	}
	if setting.MaxConcurrency > LoadTestHardMaxConcurrency {
		setting.MaxConcurrency = LoadTestHardMaxConcurrency
	}
	return setting
}

func ValidateLoadTestOption(key, value string) error {
	var min, max int
	switch key {
	case "loadtest_setting.max_duration_seconds":
		min, max = LoadTestMinDurationSeconds, LoadTestHardMaxDurationSeconds
	case "loadtest_setting.max_rps":
		min, max = 1, LoadTestHardMaxRPS
	case "loadtest_setting.max_concurrency":
		min, max = 1, LoadTestHardMaxConcurrency
	default:
		return nil
	}

	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < min || number > max {
		return fmt.Errorf("%s must be an integer between %d and %d", key, min, max)
	}
	return nil
}
