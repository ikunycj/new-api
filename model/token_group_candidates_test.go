package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenGroupCandidatesRoundTrip(t *testing.T) {
	token := Token{}
	want := []string{"openai-low", "claude-low"}

	require.NoError(t, token.SetGroupCandidates(want))
	assert.Equal(t, `["openai-low","claude-low"]`, token.GroupCandidates)

	got, err := token.GetGroupCandidates()
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestTokenGroupCandidatesEmptyAndMalformed(t *testing.T) {
	token := Token{}
	groups, err := token.GetGroupCandidates()
	require.NoError(t, err)
	assert.Empty(t, groups)
	assert.NotNil(t, groups)

	token.GroupCandidates = "not-json"
	_, err = token.GetGroupCandidates()
	require.Error(t, err)

	token.GroupCandidates = "[]"
	_, err = token.GetGroupCandidates()
	require.Error(t, err)
}

func TestTokenGroupCandidatesStorageIsNotExposedByModelJSON(t *testing.T) {
	token := Token{
		Group:           "auto",
		GroupCandidates: `["openai-low","claude-low"]`,
		GroupRetryTimes: `{"openai-low":0,"claude-low":3}`,
	}
	data, err := common.Marshal(token)
	require.NoError(t, err)

	assert.False(t, strings.Contains(string(data), "group_candidates"))
	assert.False(t, strings.Contains(string(data), "group_retry_times"))
	assert.False(t, strings.Contains(string(data), "openai-low"))
}

func TestTokenGroupRetryTimesRoundTrip(t *testing.T) {
	token := Token{}
	want := map[string]int{"openai-low": 0, "claude-low": 3}

	require.NoError(t, token.SetGroupRetryTimes(want))
	got, err := token.GetGroupRetryTimes()
	require.NoError(t, err)
	assert.Equal(t, want, got)

	require.NoError(t, token.SetGroupRetryTimes(nil))
	assert.Empty(t, token.GroupRetryTimes)
	got, err = token.GetGroupRetryTimes()
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.NotNil(t, got)
}

func TestTokenGroupRetryTimesRejectsMalformedStorage(t *testing.T) {
	token := Token{GroupRetryTimes: "not-json"}
	_, err := token.GetGroupRetryTimes()
	require.Error(t, err)
}
