package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchModelLeaderboardEntries(t *testing.T) {
	catalog := []model.Pricing{
		{ModelName: "claude-fable-5-2"},
		{ModelName: "claude-fable-5-2-max"},
		{ModelName: "claude-fable-5-2-high"},
		{ModelName: "claude-fable-5.2"},
		{ModelName: "gemini-4-carbon"},
		{ModelName: "gemini-4-carbon-high"},
		{ModelName: "gpt-5.3-codex"},
		{ModelName: "claude-fable-5"},
	}
	entries := []model.ModelLeaderboardEntry{
		{ModelName: "Claude Fable 5.2 (Max)", Rank: 1, Score: 1486.52},
		{ModelName: "Gemini 4 Carbon (High)", Rank: 7, Score: 1290.85},
		{ModelName: "Claude Fable 5.2 (High)", Rank: 9, Score: 1281.27},
		{ModelName: "GPT-5.3-Codex", Rank: 10, Score: 1250},
		{ModelName: "Claude Fable 5 (Version Preview)", Rank: 17, Score: 1200},
	}

	matched := MatchModelLeaderboardEntries(entries, catalog)
	require.Len(t, matched, 5)
	assert.Equal(t, model.ModelLeaderboardEntry{
		ModelName: "Claude Fable 5.2 (Max)", Rank: 1, Score: 1486.52,
		ModelIDs: []string{"claude-fable-5-2", "claude-fable-5-2-max", "claude-fable-5.2"},
	}, matched[0])
	assert.Equal(t, []string{"gemini-4-carbon", "gemini-4-carbon-high"}, matched[1].ModelIDs)
	assert.Equal(t, []string{"claude-fable-5-2-high"}, matched[2].ModelIDs)
	assert.Equal(t, []string{"gpt-5.3-codex"}, matched[3].ModelIDs)
	assert.Empty(t, matched[4].ModelIDs)
}
