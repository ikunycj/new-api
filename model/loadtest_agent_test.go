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

	paired, err := PairLoadTestAgent("abcd1234", "agent-secret", LoadTestAgentRuntime{Name: "MacBook", Platform: "darwin/arm64", Version: "0.1.0"})
	require.NoError(t, err)
	assert.Equal(t, agent.ID, paired.ID)
	assert.Equal(t, 7, paired.UserID)
	assert.Empty(t, paired.PairingCodeHash)
	assert.NotNil(t, paired.SecretHash)

	_, err = PairLoadTestAgent("ABCD1234", "another-secret", LoadTestAgentRuntime{Name: "Other", Platform: "linux/amd64", Version: "0.1.0"})
	assert.ErrorContains(t, err, "invalid or expired")

	_, err = GetLoadTestAgent(8, agent.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	authenticated, err := AuthenticateLoadTestAgent("agent-secret")
	require.NoError(t, err)
	assert.Equal(t, agent.ID, authenticated.ID)
}

func TestLoadTestRunClaimsOnceAndPreservesTerminalResult(t *testing.T) {
	setupLoadTestAgentDB(t)
	mockChannels := []LoadTestMockChannel{
		{Slot: 1, MaxRPS: 10, FailureRate: 0.1, FailureStatus: 503, LatencyMS: 50},
		{Slot: 2, MaxRPS: 20, FailureRate: 0.2, FailureStatus: 0, LatencyMS: 100},
		{Slot: 3, MaxRPS: 30, FailureRate: 0, FailureStatus: 429, LatencyMS: 0},
	}
	mockChannelsJSON, err := EncodeLoadTestMockChannels(mockChannels)
	require.NoError(t, err)
	run := &LoadTestRun{
		UserID: 7, AgentID: "agent-a", TokenID: 11, KeyName: "stability",
		PackageName: "stable", Model: "claude-opus-4-8", Endpoint: "anthropic",
		Prompt: "Reply OK", TargetURL: "https://example.com", DurationSeconds: 30,
		RequestsPerSecond: 20, Concurrency: 100,
		MockEnabled: true, MockChannelsJSON: mockChannelsJSON,
	}
	require.NoError(t, CreateLoadTestRun(run))

	claimed, err := ClaimLoadTestRun("agent-a")
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, LoadTestRunDispatched, claimed.Status)
	assert.Equal(t, mockChannels, claimed.MockChannels)
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
	assert.Equal(t, mockChannels, stored.MockChannels)
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
	_, err = PairLoadTestAgent("ABCD1234", "agent-secret", LoadTestAgentRuntime{Name: "MacBook", Platform: "darwin/arm64", Version: "0.1.0"})
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

func TestManagedLoadTestAgentsAreSharedWithoutExposingOtherLocalAgents(t *testing.T) {
	setupLoadTestAgentDB(t)
	localAgent, err := NewLoadTestAgentPairing(7, "LOCAL123", common.GetTimestamp()+300)
	require.NoError(t, err)
	_, err = PairLoadTestAgent("LOCAL123", "local-secret", LoadTestAgentRuntime{Name: "Local", Platform: "darwin/arm64", Version: "0.2.0"})
	require.NoError(t, err)
	managedAgent, err := NewManagedLoadTestAgentPairing(1, "SHARED12", common.GetTimestamp()+300)
	require.NoError(t, err)
	_, err = PairLoadTestAgent("SHARED12", "managed-secret", LoadTestAgentRuntime{
		Name: "Shared", Platform: "linux/amd64", Version: "0.2.0", MaxRPS: 200, MaxConcurrency: 500,
	})
	require.NoError(t, err)

	_, err = GetUsableLoadTestAgent(8, localAgent.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	shared, err := GetUsableLoadTestAgent(8, managedAgent.ID)
	require.NoError(t, err)
	assert.True(t, shared.Managed)
	assert.Equal(t, 200, shared.MaxRPS)

	localAgents, err := ListLoadTestAgents(8)
	require.NoError(t, err)
	assert.Empty(t, localAgents)
	managedAgents, err := ListManagedLoadTestAgents()
	require.NoError(t, err)
	require.Len(t, managedAgents, 1)
	assert.Equal(t, managedAgent.ID, managedAgents[0].ID)
}

func TestManagedAgentCapacitySurvivesHeartbeatAndCanBeUpdated(t *testing.T) {
	setupLoadTestAgentDB(t)
	agent, err := NewManagedLoadTestAgentPairing(1, "SHARED12", common.GetTimestamp()+300)
	require.NoError(t, err)
	_, err = PairLoadTestAgent("SHARED12", "managed-secret", LoadTestAgentRuntime{
		Name: "Shared", Platform: "linux/amd64", Version: "0.2.0", MaxRPS: 50, MaxConcurrency: 100,
	})
	require.NoError(t, err)

	updated, err := UpdateManagedLoadTestAgentCapacity(agent.ID, 200, 500)
	require.NoError(t, err)
	assert.Equal(t, 200, updated.MaxRPS)
	assert.Equal(t, 500, updated.MaxConcurrency)

	require.NoError(t, TouchLoadTestAgent(agent.ID, LoadTestAgentRuntime{
		Name: "Shared", Platform: "linux/amd64", Version: "0.2.1", CPUCores: 2,
		MemoryBytes: 1 << 30, MaxRPS: 50, MaxConcurrency: 100,
	}))
	refreshed, err := GetUsableLoadTestAgent(8, agent.ID)
	require.NoError(t, err)
	assert.Equal(t, 200, refreshed.MaxRPS)
	assert.Equal(t, 500, refreshed.MaxConcurrency)
}

func TestRevokingManagedAgentCancelsRunsFromAllUsers(t *testing.T) {
	setupLoadTestAgentDB(t)
	agent, err := NewManagedLoadTestAgentPairing(1, "SHARED12", common.GetTimestamp()+300)
	require.NoError(t, err)
	_, err = PairLoadTestAgent("SHARED12", "managed-secret", LoadTestAgentRuntime{Name: "Shared", Platform: "linux/amd64", Version: "0.2.0"})
	require.NoError(t, err)
	run := &LoadTestRun{
		UserID: 8, AgentID: agent.ID, AgentManaged: true, TokenID: 11, KeyName: "stable",
		PackageName: "stable", Model: "gpt-test", Endpoint: "openai", Prompt: "OK",
		TargetURL: "https://example.com", DurationSeconds: 30, RequestsPerSecond: 20, Concurrency: 100,
	}
	require.NoError(t, CreateLoadTestRun(run))
	require.NoError(t, RevokeManagedLoadTestAgent(agent.ID))

	stored, err := GetLoadTestRun(8, run.ID)
	require.NoError(t, err)
	assert.Equal(t, LoadTestRunCancelled, stored.Status)
	assert.Equal(t, "managed agent revoked", stored.ErrorMessage)
}
