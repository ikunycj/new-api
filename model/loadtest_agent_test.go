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
	require.NoError(t, db.AutoMigrate(&LoadTestAgent{}, &LoadTestRun{}, &LoadTestRunWorker{}))
	previousDB := DB
	DB = db
	t.Cleanup(func() { DB = previousDB })
	return db
}

func TestSharedLoadTestRunAssignsDistinctWorkerShards(t *testing.T) {
	setupLoadTestAgentDB(t)
	run := &LoadTestRun{
		UserID: 7, AgentID: "agent-shared", TokenID: 11, KeyName: "stable",
		PackageName: "stable", Model: "gpt-test", Endpoint: "openai", Prompt: "OK",
		TargetURL: "https://example.com", DurationSeconds: 30, RequestsPerSecond: 100,
		Concurrency: 40, ExecutionMode: LoadTestExecutionShared, ExpectedWorkers: 2,
	}
	require.NoError(t, CreateLoadTestRun(run))

	first, err := ClaimLoadTestRunForWorker(run.AgentID, "worker-a")
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, 50, first.AssignedRPS)
	assert.Equal(t, 20, first.AssignedConcurrency)

	second, err := ClaimLoadTestRunForWorker(run.AgentID, "worker-b")
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, 50, second.AssignedRPS)
	assert.Equal(t, 20, second.AssignedConcurrency)
	assert.NotEqual(t, first.WorkerID, second.WorkerID)

	retry, err := ClaimLoadTestRunForWorker(run.AgentID, "worker-a")
	require.NoError(t, err)
	require.NotNil(t, retry)
	assert.Equal(t, first.WorkerID, retry.WorkerID)
	assert.Equal(t, first.AssignedRPS, retry.AssignedRPS)
	require.NoError(t, FinishLoadTestRunWorker(run.ID, "worker-a", LoadTestRunWorkerCompleted, map[string]any{}))
	terminalRetry, err := ClaimLoadTestRunForWorker(run.AgentID, "worker-a")
	require.NoError(t, err)
	assert.Nil(t, terminalRetry)
}

func TestNormalizeLoadTestExecutionModesBackfillsLegacyRows(t *testing.T) {
	setupLoadTestAgentDB(t)
	run := &LoadTestRun{
		UserID: 7, AgentID: "agent-legacy", TokenID: 11, KeyName: "stable",
		PackageName: "stable", Model: "gpt-test", Endpoint: "openai", Prompt: "OK",
		TargetURL: "https://example.com", DurationSeconds: 30, RequestsPerSecond: 1,
		Concurrency: 1,
	}
	require.NoError(t, CreateLoadTestRun(run))
	require.NoError(t, DB.Model(&LoadTestRun{}).Where("id = ?", run.ID).Update("execution_mode", "").Error)
	require.NoError(t, normalizeLoadTestExecutionModes())
	var stored LoadTestRun
	require.NoError(t, DB.First(&stored, "id = ?", run.ID).Error)
	assert.Equal(t, LoadTestExecutionSingle, stored.ExecutionMode)
}

func TestSharedLoadTestRunAggregatesWorkerProgressOnlyOnce(t *testing.T) {
	setupLoadTestAgentDB(t)
	run := &LoadTestRun{
		UserID: 7, AgentID: "agent-shared", TokenID: 11, KeyName: "stable",
		PackageName: "stable", Model: "gpt-test", Endpoint: "openai", Prompt: "OK",
		TargetURL: "https://example.com", DurationSeconds: 30, RequestsPerSecond: 100,
		Concurrency: 40, ExecutionMode: LoadTestExecutionShared, ExpectedWorkers: 2,
	}
	require.NoError(t, CreateLoadTestRun(run))
	first, err := ClaimLoadTestRunForWorker(run.AgentID, "worker-a")
	require.NoError(t, err)
	second, err := ClaimLoadTestRunForWorker(run.AgentID, "worker-b")
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotNil(t, second)

	progress := map[string]any{
		"sent": int64(50), "completed": int64(48), "successes": int64(45), "failures": int64(3),
		"dropped": int64(2), "current_rps": 50.0,
	}
	require.NoError(t, UpdateLoadTestRunWorkerProgress(run.ID, "worker-a", progress))
	require.NoError(t, UpdateLoadTestRunWorkerProgress(run.ID, "worker-b", map[string]any{
		"sent": int64(50), "completed": int64(50), "successes": int64(49), "failures": int64(1),
		"dropped": int64(0), "current_rps": 50.0,
	}))
	// A repeated progress payload from the same worker replaces its snapshot,
	// it must not double-count the parent aggregate.
	require.NoError(t, UpdateLoadTestRunWorkerProgress(run.ID, "worker-a", progress))

	aggregate, err := GetLoadTestRunAggregate(run.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), aggregate.Sent)
	assert.Equal(t, int64(98), aggregate.Completed)
	assert.Equal(t, int64(94), aggregate.Successes)
	assert.Equal(t, int64(4), aggregate.Failures)
	assert.Equal(t, int64(2), aggregate.Dropped)
}

func TestSharedLoadTestRunMarksStaleWorkerAsFailed(t *testing.T) {
	setupLoadTestAgentDB(t)
	run := &LoadTestRun{
		UserID: 7, AgentID: "agent-shared", TokenID: 11, KeyName: "stable",
		PackageName: "stable", Model: "gpt-test", Endpoint: "openai", Prompt: "OK",
		TargetURL: "https://example.com", DurationSeconds: 30, RequestsPerSecond: 100,
		Concurrency: 40, ExecutionMode: LoadTestExecutionShared, ExpectedWorkers: 2,
	}
	require.NoError(t, CreateLoadTestRun(run))
	first, err := ClaimLoadTestRunForWorker(run.AgentID, "worker-a")
	require.NoError(t, err)
	_, err = ClaimLoadTestRunForWorker(run.AgentID, "worker-b")
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NoError(t, UpdateLoadTestRunWorkerProgress(run.ID, "worker-a", map[string]any{
		"sent": int64(10), "completed": int64(10), "successes": int64(10),
	}))
	require.NoError(t, DB.Model(&LoadTestRunWorker{}).Where("run_id = ? AND worker_id = ?", run.ID, "worker-a").Update("last_seen_at", common.GetTimestamp()-60).Error)
	require.NoError(t, FinishLoadTestRunWorker(run.ID, "worker-b", LoadTestRunWorkerCompleted, map[string]any{
		"sent": int64(10), "completed": int64(10), "successes": int64(10),
	}))
	stored, err := GetLoadTestRun(7, run.ID)
	require.NoError(t, err)
	assert.Equal(t, LoadTestRunFailed, stored.Status)
	assert.Equal(t, "worker heartbeat timed out", stored.Workers[0].ErrorMessage)
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
		{Slot: 1, FailureRate: 0.1, FailureStatus: 503, LatencyMS: 50},
		{Slot: 2, FailureRate: 0.2, FailureStatus: 0, LatencyMS: 100},
		{Slot: 3, FailureRate: 0, FailureStatus: 429, LatencyMS: 0},
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
