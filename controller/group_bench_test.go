package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service/groupbench"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupBenchNextRunAt(t *testing.T) {
	const hour = int64(3600)
	base := int64(1790000000) / hour * hour
	tests := []struct {
		name     string
		now      int64
		interval int
		want     int64
	}{
		{name: "mid hour", now: base + 1234, interval: 60, want: base + hour},
		{name: "exactly on the hour moves to next", now: base, interval: 60, want: base + hour},
		{name: "quarter hour", now: base + 16*60, interval: 15, want: base + 30*60},
		{name: "zero interval falls back to hourly", now: base + 1, interval: 0, want: base + hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, groupBenchNextRunAt(tt.now, tt.interval))
		})
	}
}

func TestBuildGroupBenchRequest(t *testing.T) {
	preset, ok := groupbench.GetPreset(groupbench.DefaultPresetKey)
	require.True(t, ok)

	chat, ok := buildGroupBenchRequest(preset, "claude-opus-5", "anthropic").(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	assert.True(t, *chat.Stream)
	assert.Equal(t, preset.MaxTokens, *chat.MaxTokens)
	assert.Nil(t, chat.Temperature)

	responses, ok := buildGroupBenchRequest(preset, "gpt-6-sol", "openai-response").(*dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.True(t, *responses.Stream)
	assert.Equal(t, preset.MaxTokens, *responses.MaxOutputTokens)
	assert.Contains(t, string(responses.Input), "pelican")
}
