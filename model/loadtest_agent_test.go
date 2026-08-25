package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupLoadTestAgentDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&LoadTestAgent{}, &LoadTestRun{}))
	previousDB := DB
	DB = db
	t.Cleanup(func() { DB = previousDB })
	return db
}

func TestLoadTestAgentPairingIsOneTimeAndScopedToOwner(t *testing.T) {
	setupLoadTestAgentDB(t)
	agent, err := NewLoadTestAgentPairing(7, "ABCD1234", common.GetTimestamp()+300)
	require.NoError(t, err)

	paired, err := PairLoadTestAgent("abcd1234", "agent-secret", "MacBook", "darwin/arm64", "0.1.0")
	require.NoError(t, err)
	assert.Equal(t, agent.ID, paired.ID)
	assert.Equal(t, 7, paired.UserID)
	assert.Empty(t, paired.PairingCodeHash)
	assert.NotNil(t, paired.SecretHash)

	_, err = PairLoadTestAgent("ABCD1234", "another-secret", "Other", "linux/amd64", "0.1.0")
	assert.ErrorContains(t, err, "invalid or expired")

	_, err = GetLoadTestAgent(8, agent.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	authenticated, err := AuthenticateLoadTestAgent("agent-secret")
	require.NoError(t, err)
	assert.Equal(t, agent.ID, authenticated.ID)
}

func TestLoadTestRunClaimsOnceAndPreservesTerminalResult(t *testing.T) {
	setupLoadTestAgentDB(t)
	run := &LoadTestRun{
		UserID: 7, AgentID: "agent-a", TokenID: 11, KeyName: "stability",
		PackageName: "stable", Model: "claude-opus-4-8", Endpoint: "anthropic",
		Prompt: "Reply OK", TargetURL: "https://example.com", DurationSeconds: 30,
		RequestsPerSecond: 20, Concurrency: 100,
	}
	require.NoError(t, CreateLoadTestRun(run))

	claimed, err := ClaimLoadTestRun("agent-a")
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, LoadTestRunDispatched, claimed.Status)
	secondClaim, err := ClaimLoadTestRun("agent-a")
	require.NoError(t, err)
	assert.Nil(t, secondClaim)

	require.NoError(t, UpdateLoadTestRunProgress("agent-a", run.ID, map[string]any{
		"sent": int64(200), "completed": int64(190), "successes": int64(188), "failures": int64(2),
	}))
	require.NoError(t, FinishLoadTestRun("agent-a", run.ID, LoadTestRunCompleted, map[string]any{
		"sent": int64(200), "completed": int64(200), "successes": int64(198), "failures": int64(2),
	}))

	stored, err := GetLoadTestRun(7, run.ID)
	require.NoError(t, err)
	assert.Equal(t, LoadTestRunCompleted, stored.Status)
	assert.Equal(t, int64(198), stored.Successes)
	assert.NotZero(t, stored.FinishedAt)
	assert.Error(t, UpdateLoadTestRunProgress("agent-a", run.ID, map[string]any{"completed": int64(201)}))
}

func TestQueuedLoadTestRunCancelsWithoutAgent(t *testing.T) {
	setupLoadTestAgentDB(t)
	run := &LoadTestRun{
		UserID: 7, AgentID: "agent-a", TokenID: 11, KeyName: "stable",
		PackageName: "stable", Model: "gpt-test", Endpoint: "openai", Prompt: "OK",
		TargetURL: "https://example.com", DurationSeconds: 30, RequestsPerSecond: 20, Concurrency: 100,
	}
	require.NoError(t, CreateLoadTestRun(run))
	require.NoError(t, RequestLoadTestRunCancellation(7, run.ID))

	stored, err := GetLoadTestRun(7, run.ID)
	require.NoError(t, err)
	assert.Equal(t, LoadTestRunCancelled, stored.Status)
	assert.NotZero(t, stored.FinishedAt)
}

func TestLoadTestRunCancellationIsIdempotentWhileAgentStops(t *testing.T) {
	setupLoadTestAgentDB(t)
	run := &LoadTestRun{
		UserID: 7, AgentID: "agent-a", TokenID: 11, KeyName: "stable",
		PackageName: "stable", Model: "gpt-test", Endpoint: "openai", Prompt: "OK",
		TargetURL: "https://example.com", DurationSeconds: 30, RequestsPerSecond: 20, Concurrency: 100,
	}
	require.NoError(t, CreateLoadTestRun(run))
	claimed, err := ClaimLoadTestRun("agent-a")
	require.NoError(t, err)
	require.NotNil(t, claimed)

	require.NoError(t, RequestLoadTestRunCancellation(7, run.ID))
	require.NoError(t, RequestLoadTestRunCancellation(7, run.ID))

	stored, err := GetLoadTestRun(7, run.ID)
	require.NoError(t, err)
	assert.Equal(t, LoadTestRunCancelRequested, stored.Status)
}

func TestRevokingAgentTerminatesCancelRequestedRun(t *testing.T) {
	setupLoadTestAgentDB(t)
	agent, err := NewLoadTestAgentPairing(7, "ABCD1234", common.GetTimestamp()+300)
	require.NoError(t, err)
	_, err = PairLoadTestAgent("ABCD1234", "agent-secret", "MacBook", "darwin/arm64", "0.1.0")
	require.NoError(t, err)
	run := &LoadTestRun{
		UserID: 7, AgentID: agent.ID, TokenID: 11, KeyName: "stable",
		PackageName: "stable", Model: "gpt-test", Endpoint: "openai", Prompt: "OK",
		TargetURL: "https://example.com", DurationSeconds: 30, RequestsPerSecond: 20, Concurrency: 100,
	}
	require.NoError(t, CreateLoadTestRun(run))
	claimed, err := ClaimLoadTestRun(agent.ID)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.NoError(t, RequestLoadTestRunCancellation(7, run.ID))

	require.NoError(t, RevokeLoadTestAgent(7, agent.ID))
	stored, err := GetLoadTestRun(7, run.ID)
	require.NoError(t, err)
	assert.Equal(t, LoadTestRunCancelled, stored.Status)
	assert.Equal(t, "agent revoked", stored.ErrorMessage)
}
