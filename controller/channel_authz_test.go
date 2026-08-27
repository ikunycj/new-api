package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelHasSensitiveChanges(t *testing.T) {
	baseURL := "https://api.example.com"
	headerOverride := `{"Authorization":"Bearer {api_key}"}`
	origin := &model.Channel{
		Type:           1,
		Key:            "old-key",
		BaseURL:        &baseURL,
		HeaderOverride: &headerOverride,
		Models:         "gpt-4o",
		Group:          "default",
	}

	t.Run("non-sensitive routing fields", func(t *testing.T) {
		updated := PatchChannel{Channel: *origin}
		updated.Models = "gpt-4o,gpt-4o-mini"
		updated.Group = "vip"

		assert.False(t, channelHasSensitiveChanges(&updated, origin, map[string]any{
			"models": updated.Models,
			"group":  updated.Group,
		}))
	})

	t.Run("key change", func(t *testing.T) {
		updated := PatchChannel{Channel: *origin}
		updated.Key = "new-key"

		assert.True(t, channelHasSensitiveChanges(&updated, origin, map[string]any{"key": updated.Key}))
	})

	t.Run("blank key is unchanged", func(t *testing.T) {
		updated := PatchChannel{Channel: *origin}
		updated.Key = "  \n\t "

		assert.False(t, channelHasSensitiveChanges(&updated, origin, map[string]any{"key": updated.Key}))
	})

	t.Run("base url change", func(t *testing.T) {
		updated := PatchChannel{Channel: *origin}
		newBaseURL := "https://leak.example.com"
		updated.BaseURL = &newBaseURL

		assert.True(t, channelHasSensitiveChanges(&updated, origin, map[string]any{"base_url": newBaseURL}))
	})

	t.Run("header override change", func(t *testing.T) {
		updated := PatchChannel{Channel: *origin}
		newHeaderOverride := `{"X-Key":"{api_key}"}`
		updated.HeaderOverride = &newHeaderOverride

		assert.True(t, channelHasSensitiveChanges(&updated, origin, map[string]any{"header_override": newHeaderOverride}))
	})

	t.Run("unknown routing field is rejected", func(t *testing.T) {
		updated := PatchChannel{}
		updated.Id = origin.Id

		assert.True(t, channelHasSensitiveChanges(&updated, origin, map[string]any{"retired_route_field": 10}))
	})

	t.Run("unknown field fails closed", func(t *testing.T) {
		updated := PatchChannel{Channel: *origin}

		assert.True(t, channelHasSensitiveChanges(&updated, origin, map[string]any{"future_secret_field": "x"}))
	})

	t.Run("status is operational", func(t *testing.T) {
		updated := PatchChannel{Channel: *origin}
		updated.Status = common.ChannelStatusManuallyDisabled

		assert.False(t, channelHasSensitiveChanges(&updated, origin, map[string]any{"status": updated.Status}))
	})

	t.Run("read-only fields are ignored by sensitivity check", func(t *testing.T) {
		updated := PatchChannel{Channel: *origin}
		updated.Balance = 99
		updated.UsedQuota = 100
		updated.ResponseTime = 200

		assert.False(t, channelHasSensitiveChanges(&updated, origin, map[string]any{
			"balance":       updated.Balance,
			"used_quota":    updated.UsedQuota,
			"response_time": updated.ResponseTime,
		}))
	})
}

func TestClearChannelReadOnlyFields(t *testing.T) {
	channel := PatchChannel{Channel: model.Channel{
		CreatedTime:        11,
		TestTime:           22,
		ResponseTime:       33,
		Balance:            44.5,
		BalanceUpdatedTime: 55,
		UsedQuota:          66,
		Models:             "gpt-4o",
		Group:              "default",
	}}

	clearChannelReadOnlyFields(&channel, map[string]any{
		"created_time":         channel.CreatedTime,
		"test_time":            channel.TestTime,
		"response_time":        channel.ResponseTime,
		"balance":              channel.Balance,
		"balance_updated_time": channel.BalanceUpdatedTime,
		"used_quota":           channel.UsedQuota,
		"models":               channel.Models,
		"group":                channel.Group,
	})

	assert.Zero(t, channel.CreatedTime)
	assert.Zero(t, channel.TestTime)
	assert.Zero(t, channel.ResponseTime)
	assert.Zero(t, channel.Balance)
	assert.Zero(t, channel.BalanceUpdatedTime)
	assert.Zero(t, channel.UsedQuota)
	assert.Equal(t, "gpt-4o", channel.Models)
	assert.Equal(t, "default", channel.Group)
}

func TestChannelUpdateColumnsIncludeExplicitZeroFields(t *testing.T) {
	columns := channelUpdateColumns(map[string]any{
		"auto_probe_enabled":     false,
		"probe_interval_seconds": 0,
		"price_multiplier":       0,
		"price_multiplier_mode":  "",
		"multi_key_mode":         "single",
		"name":                   "updated",
		"balance":                0,
		"key_mode":               "replace",
	})

	assert.ElementsMatch(t, []string{
		"auto_probe_enabled",
		"probe_interval_seconds",
		"price_multiplier",
		"price_multiplier_mode",
		"channel_info",
		"name",
	}, columns)
}

func TestChannelUpdateColumnsPersistMultiKeyMetadataForKeyOperations(t *testing.T) {
	assert.ElementsMatch(t, []string{"key", "channel_info"}, channelUpdateColumns(map[string]any{
		"key": "new-key",
	}))
	assert.ElementsMatch(t, []string{"channel_info"}, channelUpdateColumns(map[string]any{
		"key_mode": "append",
	}))
	for name, value := range map[string]any{
		"empty":      "",
		"whitespace": "  \n\t ",
		"null":       nil,
	} {
		t.Run(name+" key is unchanged", func(t *testing.T) {
			assert.Empty(t, channelUpdateColumns(map[string]any{"key": value}))
		})
	}
}

func TestValidateChannelRequiresPricingGroup(t *testing.T) {
	tests := []struct {
		name  string
		group string
		valid bool
	}{
		{name: "empty", group: "", valid: false},
		{name: "whitespace", group: "  ", valid: false},
		{name: "empty list", group: ",,", valid: false},
		{name: "single group", group: "standard", valid: true},
		{name: "multiple groups", group: "standard, premium", valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &model.Channel{Key: "test-key", Models: "gpt-4o", TestModel: common.GetPointer("gpt-4o"), Group: tt.group}
			err := validateChannel(channel, true)
			if tt.valid {
				require.NoError(t, err)
				assert.Equal(t, strings.ReplaceAll(tt.group, " ", ""), channel.Group)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "pricing group")
			}
		})
	}
}

func TestValidateChannelRequiresAtLeastOneModel(t *testing.T) {
	for name, models := range map[string]string{
		"empty":      "",
		"whitespace": "  ",
		"empty list": " , , ",
	} {
		t.Run(name, func(t *testing.T) {
			err := validateChannel(&model.Channel{Key: "test-key", Models: models, TestModel: common.GetPointer("gpt-4o"), Group: "default"}, true)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "at least one model")
		})
	}
}

func TestValidateChannelRequiresTestModel(t *testing.T) {
	err := validateChannel(&model.Channel{Key: "test-key", Models: "gpt-4o", Group: "default"}, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test model")
}

func TestUpdateChannelRejectsStatusField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/channel/",
		bytes.NewBufferString(`{"id":1,"status":2}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateChannel(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
}

func TestChannelStatusValidation(t *testing.T) {
	assert.True(t, isManageableChannelStatus(common.ChannelStatusEnabled))
	assert.True(t, isManageableChannelStatus(common.ChannelStatusManuallyDisabled))
	assert.False(t, isManageableChannelStatus(common.ChannelStatusAutoDisabled))
	assert.False(t, isManageableChannelStatus(0))
}

// TestChannelFieldsAreClassified guards the fail-closed sensitivity check: every
// JSON field of PatchChannel (including the embedded model.Channel) must be listed
// in channelSensitiveFields, channelNonSensitiveFields, or
// channelOperationalFields. A newly added field that is left unclassified will
// fail this test, forcing a conscious permission decision instead of silently
// defaulting either way.
func TestChannelFieldsAreClassified(t *testing.T) {
	classified := func(name string) bool {
		if _, ok := channelSensitiveFields[name]; ok {
			return true
		}
		if _, ok := channelNonSensitiveFields[name]; ok {
			return true
		}
		if _, ok := channelOperationalFields[name]; ok {
			return true
		}
		_, ok := channelReadOnlyFields[name]
		return ok
	}

	var collect func(rt reflect.Type) []string
	collect = func(rt reflect.Type) []string {
		var names []string
		for i := 0; i < rt.NumField(); i++ {
			field := rt.Field(i)
			if field.Anonymous && field.Type.Kind() == reflect.Struct {
				names = append(names, collect(field.Type)...)
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			names = append(names, name)
		}
		return names
	}

	for _, name := range collect(reflect.TypeOf(PatchChannel{})) {
		assert.Truef(t, classified(name),
			"channel field %q is not classified; add it to channelSensitiveFields, channelNonSensitiveFields, channelOperationalFields, or channelReadOnlyFields in channel_authz.go", name)
	}
}
