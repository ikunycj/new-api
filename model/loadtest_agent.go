package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

type LoadTestRunStatus string

const (
	LoadTestRunQueued          LoadTestRunStatus = "queued"
	LoadTestRunDispatched      LoadTestRunStatus = "dispatched"
	LoadTestRunRunning         LoadTestRunStatus = "running"
	LoadTestRunCancelRequested LoadTestRunStatus = "cancel_requested"
	LoadTestRunCompleted       LoadTestRunStatus = "completed"
	LoadTestRunFailed          LoadTestRunStatus = "failed"
	LoadTestRunCancelled       LoadTestRunStatus = "cancelled"
)

type LoadTestExecutionMode string

const (
	LoadTestExecutionSingle LoadTestExecutionMode = "single"
	LoadTestExecutionShared LoadTestExecutionMode = "shared"
)

const (
	loadTestSharedJoinWindowSeconds = 10
	loadTestWorkerStaleSeconds      = 30
)

type LoadTestMockChannel struct {
	Slot          int     `json:"slot"`
	FailureRate   float64 `json:"failure_rate"`
	FailureStatus int     `json:"failure_status"`
	LatencyMS     int     `json:"latency_ms"`
}

type LoadTestAgent struct {
	ID              string  `json:"id" gorm:"type:varchar(64);primaryKey"`
	UserID          int     `json:"user_id" gorm:"index;not null"`
	Managed         bool    `json:"managed" gorm:"index"`
	Name            string  `json:"name" gorm:"type:varchar(128);not null"`
	Platform        string  `json:"platform" gorm:"type:varchar(64);not null"`
	Version         string  `json:"version" gorm:"type:varchar(32);not null"`
	CPUCores        int     `json:"cpu_cores"`
	MemoryBytes     int64   `json:"memory_bytes" gorm:"bigint"`
	MaxRPS          int     `json:"max_rps"`
	MaxConcurrency  int     `json:"max_concurrency"`
	SecretHash      *string `json:"-" gorm:"type:char(64);uniqueIndex"`
	PairingCodeHash string  `json:"-" gorm:"type:char(64);index"`
	PairingExpires  int64   `json:"-" gorm:"bigint;index"`
	LastSeenAt      int64   `json:"last_seen_at" gorm:"bigint;index"`
	CreatedAt       int64   `json:"created_at" gorm:"bigint;index"`
	RevokedAt       int64   `json:"revoked_at" gorm:"bigint;index"`
}

type LoadTestRun struct {
	ID                string                `json:"id" gorm:"type:varchar(64);primaryKey"`
	UserID            int                   `json:"user_id" gorm:"index;not null"`
	AgentID           string                `json:"agent_id" gorm:"type:varchar(64);index;not null"`
	TokenID           int                   `json:"token_id" gorm:"index;not null"`
	KeyName           string                `json:"key_name" gorm:"type:varchar(128);not null"`
	PackageName       string                `json:"package_name" gorm:"type:varchar(64);not null"`
	Model             string                `json:"model" gorm:"type:varchar(128);not null"`
	Endpoint          string                `json:"endpoint" gorm:"type:varchar(32);not null"`
	Prompt            string                `json:"prompt" gorm:"type:text;not null"`
	PromptCache       bool                  `json:"prompt_cache"`
	StreamMode        bool                  `json:"stream_mode"`
	MockEnabled       bool                  `json:"mock_enabled"`
	MockFailureRate   float64               `json:"mock_failure_rate"`
	MockFailureStatus int                   `json:"mock_failure_status"`
	MockLatencyMS     int                   `json:"mock_latency_ms"`
	MockChannelsJSON  string                `json:"-" gorm:"type:text"`
	MockChannels      []LoadTestMockChannel `json:"mock_channels" gorm:"-"`
	Workers           []LoadTestRunWorker   `json:"workers" gorm:"-"`
	// Nullable on purpose: existing installations may already have rows when
	// this column is added. Migration backfills empty values to "single".
	ExecutionMode     LoadTestExecutionMode `json:"execution_mode" gorm:"type:varchar(16);index"`
	ExpectedWorkers   int                   `json:"expected_workers"`
	JoinDeadlineAt    int64                 `json:"join_deadline_at" gorm:"bigint"`
	StartAt           int64                 `json:"start_at" gorm:"bigint"`
	TargetURL         string                `json:"-" gorm:"type:varchar(512);not null"`
	DurationSeconds   int                   `json:"duration_seconds" gorm:"not null"`
	RequestsPerSecond int                   `json:"requests_per_second" gorm:"not null"`
	Concurrency       int                   `json:"concurrency" gorm:"not null"`
	MaxOutputTokens   int                   `json:"max_output_tokens" gorm:"not null;default:256"`
	AgentManaged      bool                  `json:"agent_managed" gorm:"index"`
	Status            LoadTestRunStatus     `json:"status" gorm:"type:varchar(32);index;not null"`
	Sent              int64                 `json:"sent" gorm:"bigint"`
	Completed         int64                 `json:"completed" gorm:"bigint"`
	Successes         int64                 `json:"successes" gorm:"bigint"`
	Failures          int64                 `json:"failures" gorm:"bigint"`
	Dropped           int64                 `json:"dropped" gorm:"bigint"`
	InputTokens       int64                 `json:"input_tokens" gorm:"bigint"`
	OutputTokens      int64                 `json:"output_tokens" gorm:"bigint"`
	CacheReadTokens   int64                 `json:"cache_read_tokens" gorm:"bigint"`
	CacheWriteTokens  int64                 `json:"cache_write_tokens" gorm:"bigint"`
	UsageMissing      int64                 `json:"usage_missing" gorm:"bigint"`
	TokenStatsSource  string                `json:"token_stats_source" gorm:"type:varchar(32);default:''"`
	CurrentRPS        float64               `json:"current_rps"`
	P50MS             float64               `json:"p50_ms"`
	P95MS             float64               `json:"p95_ms"`
	P99MS             float64               `json:"p99_ms"`
	ErrorCountsJSON   string                `json:"-" gorm:"type:text"`
	ErrorCounts       map[string]int64      `json:"error_counts" gorm:"-"`
	ErrorMessage      string                `json:"error_message" gorm:"type:text"`
	CreatedAt         int64                 `json:"created_at" gorm:"bigint;index"`
	StartedAt         int64                 `json:"started_at" gorm:"bigint"`
	FinishedAt        int64                 `json:"finished_at" gorm:"bigint"`
	UpdatedAt         int64                 `json:"updated_at" gorm:"bigint;index"`
}

type LoadTestRunWorkerStatus string

const (
	LoadTestRunWorkerJoined    LoadTestRunWorkerStatus = "joined"
	LoadTestRunWorkerRunning   LoadTestRunWorkerStatus = "running"
	LoadTestRunWorkerCompleted LoadTestRunWorkerStatus = "completed"
	LoadTestRunWorkerFailed    LoadTestRunWorkerStatus = "failed"
	LoadTestRunWorkerCancelled LoadTestRunWorkerStatus = "cancelled"
	LoadTestRunWorkerLost      LoadTestRunWorkerStatus = "lost"
)

// LoadTestRunWorker is an ephemeral execution record for one process joined
// to a shared logical Agent. Its counters are snapshots, so retries of the
// same progress payload replace the row instead of double-counting a run.
type LoadTestRunWorker struct {
	ID                  string                  `json:"id" gorm:"type:varchar(96);primaryKey"`
	RunID               string                  `json:"run_id" gorm:"type:varchar(64);uniqueIndex:idx_loadtest_worker_run_worker,priority:1;index;not null"`
	AgentID             string                  `json:"agent_id" gorm:"type:varchar(64);index;not null"`
	WorkerID            string                  `json:"worker_id" gorm:"type:varchar(96);uniqueIndex:idx_loadtest_worker_run_worker,priority:2;not null"`
	Slot                int                     `json:"slot"`
	Name                string                  `json:"name" gorm:"type:varchar(128)"`
	Platform            string                  `json:"platform" gorm:"type:varchar(64)"`
	CPUCores            int                     `json:"cpu_cores"`
	MemoryBytes         int64                   `json:"memory_bytes" gorm:"bigint"`
	MaxRPS              int                     `json:"max_rps"`
	MaxConcurrency      int                     `json:"max_concurrency"`
	AssignedRPS         int                     `json:"assigned_rps"`
	AssignedConcurrency int                     `json:"assigned_concurrency"`
	Status              LoadTestRunWorkerStatus `json:"status" gorm:"type:varchar(16);index;not null"`
	Sent                int64                   `json:"sent" gorm:"bigint"`
	Completed           int64                   `json:"completed" gorm:"bigint"`
	Successes           int64                   `json:"successes" gorm:"bigint"`
	Failures            int64                   `json:"failures" gorm:"bigint"`
	Dropped             int64                   `json:"dropped" gorm:"bigint"`
	InputTokens         int64                   `json:"input_tokens" gorm:"bigint"`
	OutputTokens        int64                   `json:"output_tokens" gorm:"bigint"`
	CacheReadTokens     int64                   `json:"cache_read_tokens" gorm:"bigint"`
	CacheWriteTokens    int64                   `json:"cache_write_tokens" gorm:"bigint"`
	UsageMissing        int64                   `json:"usage_missing" gorm:"bigint"`
	TokenStatsSource    string                  `json:"token_stats_source" gorm:"type:varchar(32);default:''"`
	CurrentRPS          float64                 `json:"current_rps"`
	P50MS               float64                 `json:"p50_ms"`
	P95MS               float64                 `json:"p95_ms"`
	P99MS               float64                 `json:"p99_ms"`
	ErrorCountsJSON     string                  `json:"-" gorm:"type:text"`
	ErrorCounts         map[string]int64        `json:"error_counts" gorm:"-"`
	ErrorMessage        string                  `json:"error_message" gorm:"type:text"`
	LastSeenAt          int64                   `json:"last_seen_at" gorm:"bigint;index"`
	StartedAt           int64                   `json:"started_at" gorm:"bigint"`
	FinishedAt          int64                   `json:"finished_at" gorm:"bigint"`
}

type LoadTestRunAggregate struct {
	Sent             int64
	Completed        int64
	Successes        int64
	Failures         int64
	Dropped          int64
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	UsageMissing     int64
	TokenStatsSource string
	CurrentRPS       float64
	P50MS            float64
	P95MS            float64
	P99MS            float64
	ErrorCounts      map[string]int64
}

type LoadTestAgentRuntime struct {
	Name           string
	Platform       string
	Version        string
	CPUCores       int
	MemoryBytes    int64
	MaxRPS         int
	MaxConcurrency int
}

func hashLoadTestCredential(value string) string {
	return fmt.Sprintf("%x", common.Sha256Raw([]byte(value)))
}

func NewLoadTestAgentPairing(userID int, pairingCode string, expiresAt int64) (*LoadTestAgent, error) {
	return newLoadTestAgentPairing(userID, pairingCode, expiresAt, false)
}

func NewManagedLoadTestAgentPairing(userID int, pairingCode string, expiresAt int64) (*LoadTestAgent, error) {
	return newLoadTestAgentPairing(userID, pairingCode, expiresAt, true)
}

func newLoadTestAgentPairing(userID int, pairingCode string, expiresAt int64, managed bool) (*LoadTestAgent, error) {
	agentID, err := common.GenerateRandomCharsKey(24)
	if err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	agent := &LoadTestAgent{
		ID:              "agent_" + agentID,
		UserID:          userID,
		Managed:         managed,
		PairingCodeHash: hashLoadTestCredential(strings.ToUpper(strings.TrimSpace(pairingCode))),
		PairingExpires:  expiresAt,
		CreatedAt:       now,
	}
	if err := DB.Create(agent).Error; err != nil {
		return nil, err
	}
	return agent, nil
}

func PairLoadTestAgent(pairingCode, secret string, runtime LoadTestAgentRuntime) (*LoadTestAgent, error) {
	now := common.GetTimestamp()
	codeHash := hashLoadTestCredential(strings.ToUpper(strings.TrimSpace(pairingCode)))
	secretHash := hashLoadTestCredential(secret)
	result := DB.Model(&LoadTestAgent{}).
		Where("pairing_code_hash = ? AND pairing_expires >= ? AND secret_hash IS NULL AND revoked_at = 0", codeHash, now).
		Updates(map[string]any{
			"name":              runtime.Name,
			"platform":          runtime.Platform,
			"version":           runtime.Version,
			"cpu_cores":         runtime.CPUCores,
			"memory_bytes":      runtime.MemoryBytes,
			"max_rps":           runtime.MaxRPS,
			"max_concurrency":   runtime.MaxConcurrency,
			"secret_hash":       &secretHash,
			"pairing_code_hash": "",
			"pairing_expires":   0,
			"last_seen_at":      now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("pairing code is invalid or expired")
	}
	var agent LoadTestAgent
	if err := DB.Where("secret_hash = ?", secretHash).First(&agent).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

func AuthenticateLoadTestAgent(secret string) (*LoadTestAgent, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("agent credential is required")
	}
	var agent LoadTestAgent
	err := DB.Where("secret_hash = ? AND revoked_at = 0", hashLoadTestCredential(secret)).First(&agent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("agent credential is invalid")
		}
		return nil, err
	}
	return &agent, nil
}

func ListLoadTestAgents(userID int) ([]LoadTestAgent, error) {
	var agents []LoadTestAgent
	err := DB.Where("user_id = ? AND managed = ? AND secret_hash IS NOT NULL AND revoked_at = 0", userID, false).Order("id desc").Find(&agents).Error
	return agents, err
}

func ListManagedLoadTestAgents() ([]LoadTestAgent, error) {
	var agents []LoadTestAgent
	err := DB.Where("managed = ? AND secret_hash IS NOT NULL AND revoked_at = 0", true).Order("id desc").Find(&agents).Error
	return agents, err
}

func GetLoadTestAgent(userID int, agentID string) (*LoadTestAgent, error) {
	var agent LoadTestAgent
	if err := DB.Where("id = ? AND user_id = ? AND managed = ? AND secret_hash IS NOT NULL AND revoked_at = 0", agentID, userID, false).First(&agent).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

func GetUsableLoadTestAgent(userID int, agentID string) (*LoadTestAgent, error) {
	var agent LoadTestAgent
	err := DB.Where("id = ? AND secret_hash IS NOT NULL AND revoked_at = 0 AND (managed = ? OR (managed = ? AND user_id = ?))", agentID, true, false, userID).First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

func TouchLoadTestAgent(agentID string, runtime LoadTestAgentRuntime) error {
	var agent LoadTestAgent
	if err := DB.Select("managed").Where("id = ? AND revoked_at = 0", agentID).First(&agent).Error; err != nil {
		return err
	}
	updates := map[string]any{"last_seen_at": common.GetTimestamp()}
	if strings.TrimSpace(runtime.Name) != "" {
		updates["name"] = runtime.Name
	}
	if strings.TrimSpace(runtime.Platform) != "" {
		updates["platform"] = runtime.Platform
	}
	if strings.TrimSpace(runtime.Version) != "" {
		updates["version"] = runtime.Version
	}
	updates["cpu_cores"] = runtime.CPUCores
	updates["memory_bytes"] = runtime.MemoryBytes
	if !agent.Managed {
		updates["max_rps"] = runtime.MaxRPS
		updates["max_concurrency"] = runtime.MaxConcurrency
	}
	return DB.Model(&LoadTestAgent{}).Where("id = ? AND revoked_at = 0", agentID).Updates(updates).Error
}

func UpdateManagedLoadTestAgentCapacity(agentID string, maxRPS, maxConcurrency int) (*LoadTestAgent, error) {
	result := DB.Model(&LoadTestAgent{}).
		Where("id = ? AND managed = ? AND secret_hash IS NOT NULL AND revoked_at = 0", agentID, true).
		Updates(map[string]any{"max_rps": maxRPS, "max_concurrency": maxConcurrency})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var agent LoadTestAgent
	if err := DB.Where("id = ?", agentID).First(&agent).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

func RevokeLoadTestAgent(userID int, agentID string) error {
	now := common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&LoadTestAgent{}).
			Where("id = ? AND user_id = ? AND revoked_at = 0", agentID, userID).
			Updates(map[string]any{"revoked_at": now, "secret_hash": nil, "pairing_code_hash": "", "pairing_expires": 0})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&LoadTestRun{}).
			Where("agent_id = ? AND user_id = ? AND status IN ?", agentID, userID, []LoadTestRunStatus{LoadTestRunQueued, LoadTestRunDispatched, LoadTestRunRunning, LoadTestRunCancelRequested}).
			Updates(map[string]any{"status": LoadTestRunCancelled, "finished_at": now, "updated_at": now, "error_message": "agent revoked"}).Error
	})
}

func RevokeManagedLoadTestAgent(agentID string) error {
	now := common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&LoadTestAgent{}).
			Where("id = ? AND managed = ? AND revoked_at = 0", agentID, true).
			Updates(map[string]any{"revoked_at": now, "secret_hash": nil, "pairing_code_hash": "", "pairing_expires": 0})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&LoadTestRun{}).
			Where("agent_id = ? AND status IN ?", agentID, []LoadTestRunStatus{LoadTestRunQueued, LoadTestRunDispatched, LoadTestRunRunning, LoadTestRunCancelRequested}).
			Updates(map[string]any{"status": LoadTestRunCancelled, "finished_at": now, "updated_at": now, "error_message": "managed agent revoked"}).Error
	})
}

func CreateLoadTestRun(run *LoadTestRun) error {
	now := common.GetTimestamp()
	if run.ID == "" {
		randomID, err := common.GenerateRandomCharsKey(24)
		if err != nil {
			return err
		}
		run.ID = "loadtest_" + randomID
	}
	run.Status = LoadTestRunQueued
	if run.ExecutionMode == "" {
		run.ExecutionMode = LoadTestExecutionSingle
	}
	if run.MaxOutputTokens < 1 {
		run.MaxOutputTokens = operation_setting.LoadTestDefaultMaxOutputTokens
	}
	run.CreatedAt = now
	run.UpdatedAt = now
	return DB.Create(run).Error
}

func splitLoadTestCapacity(total, workers, slot int) int {
	if total <= 0 || workers <= 0 || slot < 1 || slot > workers {
		return 0
	}
	base, remainder := total/workers, total%workers
	if slot <= remainder {
		return base + 1
	}
	return base
}

func newLoadTestRunWorker(run *LoadTestRun, workerID string, slot int, now int64) (*LoadTestRunWorker, error) {
	randomID, err := common.GenerateRandomCharsKey(16)
	if err != nil {
		return nil, err
	}
	return &LoadTestRunWorker{
		ID:                  "loadtest_worker_" + randomID,
		RunID:               run.ID,
		AgentID:             run.AgentID,
		WorkerID:            workerID,
		Slot:                slot,
		AssignedRPS:         splitLoadTestCapacity(run.RequestsPerSecond, run.ExpectedWorkers, slot),
		AssignedConcurrency: splitLoadTestCapacity(run.Concurrency, run.ExpectedWorkers, slot),
		Status:              LoadTestRunWorkerJoined,
		LastSeenAt:          now,
	}, nil
}

// ClaimLoadTestRunForWorker joins a worker to a shared run. The run row is
// locked while selecting a slot so two application instances cannot assign
// the same slot. A repeated poll from the same worker returns its existing
// snapshot and never creates another execution record.
func ClaimLoadTestRunForWorker(agentID, workerID string) (*LoadTestRunWorker, error) {
	agentID = strings.TrimSpace(agentID)
	workerID = strings.TrimSpace(workerID)
	if agentID == "" || workerID == "" {
		return nil, errors.New("agent and worker identifiers are required")
	}
	var claimed *LoadTestRunWorker
	err := DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		var run LoadTestRun
		err := lockForUpdate(tx).
			Where("agent_id = ? AND execution_mode = ? AND status IN ?", agentID, LoadTestExecutionShared, []LoadTestRunStatus{LoadTestRunDispatched, LoadTestRunRunning}).
			Order("created_at asc").First(&run).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = lockForUpdate(tx).
				Where("agent_id = ? AND status = ?", agentID, LoadTestRunQueued).
				Order("created_at asc").First(&run).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			if run.ExecutionMode != LoadTestExecutionShared || run.ExpectedWorkers < 2 {
				return nil
			}
			run.JoinDeadlineAt = now + loadTestSharedJoinWindowSeconds
			run.StartAt = run.JoinDeadlineAt
			if err := tx.Model(&LoadTestRun{}).Where("id = ? AND status = ?", run.ID, LoadTestRunQueued).
				Updates(map[string]any{"status": LoadTestRunDispatched, "started_at": now, "join_deadline_at": run.JoinDeadlineAt, "start_at": run.StartAt, "updated_at": now}).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if run.ExecutionMode != LoadTestExecutionShared || run.ExpectedWorkers < 2 || (run.JoinDeadlineAt > 0 && now > run.JoinDeadlineAt) {
			return nil
		}

		var existing LoadTestRunWorker
		existingErr := tx.Where("run_id = ? AND worker_id = ?", run.ID, workerID).First(&existing).Error
		if existingErr == nil {
			switch existing.Status {
			case LoadTestRunWorkerCompleted, LoadTestRunWorkerFailed, LoadTestRunWorkerCancelled, LoadTestRunWorkerLost:
				// A process keeps its WorkerID for its lifetime. Once that worker
				// has reported a terminal result, a later poll must not replay the
				// same logical run while other workers are still finishing.
				return nil
			}
			existing.LastSeenAt = now
			if err := tx.Model(&LoadTestRunWorker{}).Where("id = ?", existing.ID).Update("last_seen_at", now).Error; err != nil {
				return err
			}
			claimed = &existing
			return nil
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return existingErr
		}

		var workers []LoadTestRunWorker
		if err := tx.Where("run_id = ?", run.ID).Order("slot asc").Find(&workers).Error; err != nil {
			return err
		}
		if len(workers) >= run.ExpectedWorkers {
			return nil
		}
		usedSlots := make(map[int]struct{}, len(workers))
		for _, worker := range workers {
			usedSlots[worker.Slot] = struct{}{}
		}
		slot := 1
		for ; slot <= run.ExpectedWorkers; slot++ {
			if _, exists := usedSlots[slot]; !exists {
				break
			}
		}
		worker, err := newLoadTestRunWorker(&run, workerID, slot, now)
		if err != nil {
			return err
		}
		if err := tx.Create(worker).Error; err != nil {
			return err
		}
		claimed = worker
		return nil
	})
	return claimed, err
}

func UpdateLoadTestRunWorkerProgress(runID, workerID string, updates map[string]any) error {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(workerID) == "" {
		return errors.New("run and worker identifiers are required")
	}
	updates["status"] = LoadTestRunWorkerRunning
	updates["last_seen_at"] = common.GetTimestamp()
	result := DB.Model(&LoadTestRunWorker{}).
		Where("run_id = ? AND worker_id = ? AND status IN ?", runID, workerID, []LoadTestRunWorkerStatus{LoadTestRunWorkerJoined, LoadTestRunWorkerRunning}).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("load-test worker is not active")
	}
	return DB.Model(&LoadTestRun{}).Where("id = ? AND status = ?", runID, LoadTestRunStatus(LoadTestRunDispatched)).Updates(map[string]any{"status": LoadTestRunRunning, "updated_at": common.GetTimestamp()}).Error
}

func TouchLoadTestRunWorker(runID, workerID string, runtime LoadTestAgentRuntime) error {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(workerID) == "" {
		return errors.New("run and worker identifiers are required")
	}
	updates := map[string]any{
		"last_seen_at":    common.GetTimestamp(),
		"name":            runtime.Name,
		"platform":        runtime.Platform,
		"cpu_cores":       runtime.CPUCores,
		"memory_bytes":    runtime.MemoryBytes,
		"max_rps":         runtime.MaxRPS,
		"max_concurrency": runtime.MaxConcurrency,
	}
	result := DB.Model(&LoadTestRunWorker{}).
		Where("run_id = ? AND worker_id = ? AND status IN ?", runID, workerID, []LoadTestRunWorkerStatus{LoadTestRunWorkerJoined, LoadTestRunWorkerRunning}).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("load-test worker is not active")
	}
	return nil
}

func FinishLoadTestRunWorker(runID, workerID string, status LoadTestRunWorkerStatus, updates map[string]any) error {
	switch status {
	case LoadTestRunWorkerCompleted, LoadTestRunWorkerFailed, LoadTestRunWorkerCancelled:
	default:
		return errors.New("invalid terminal worker status")
	}
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(workerID) == "" {
		return errors.New("run and worker identifiers are required")
	}
	now := common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		var worker LoadTestRunWorker
		if err := lockForUpdate(tx).Where("run_id = ? AND worker_id = ?", runID, workerID).First(&worker).Error; err != nil {
			return err
		}
		if worker.Status == LoadTestRunWorkerCompleted || worker.Status == LoadTestRunWorkerFailed || worker.Status == LoadTestRunWorkerCancelled {
			return nil
		}
		updates["status"] = status
		updates["finished_at"] = now
		updates["last_seen_at"] = now
		result := tx.Model(&LoadTestRunWorker{}).
			Where("id = ? AND status IN ?", worker.ID, []LoadTestRunWorkerStatus{LoadTestRunWorkerJoined, LoadTestRunWorkerRunning}).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("load-test worker is not active")
		}

		var run LoadTestRun
		if err := lockForUpdate(tx).Where("id = ?", runID).First(&run).Error; err != nil {
			return err
		}
		if run.ExecutionMode != LoadTestExecutionShared {
			return nil
		}
		var workers []LoadTestRunWorker
		if err := tx.Where("run_id = ?", runID).Find(&workers).Error; err != nil {
			return err
		}
		staleBefore := now - loadTestWorkerStaleSeconds
		for index := range workers {
			worker := &workers[index]
			if (worker.Status == LoadTestRunWorkerJoined || worker.Status == LoadTestRunWorkerRunning) && worker.LastSeenAt > 0 && worker.LastSeenAt < staleBefore {
				if err := tx.Model(&LoadTestRunWorker{}).Where("id = ? AND status IN ?", worker.ID, []LoadTestRunWorkerStatus{LoadTestRunWorkerJoined, LoadTestRunWorkerRunning}).Updates(map[string]any{
					"status": LoadTestRunWorkerLost, "finished_at": now, "error_message": "worker heartbeat timed out", "last_seen_at": now,
				}).Error; err != nil {
					return err
				}
				worker.Status = LoadTestRunWorkerLost
				worker.FinishedAt = now
				worker.ErrorMessage = "worker heartbeat timed out"
			}
		}
		terminal := 0
		anyFailed := false
		anyCancelled := false
		for _, item := range workers {
			switch item.Status {
			case LoadTestRunWorkerCompleted:
				terminal++
			case LoadTestRunWorkerFailed:
				terminal++
				anyFailed = true
			case LoadTestRunWorkerLost:
				terminal++
				anyFailed = true
			case LoadTestRunWorkerCancelled:
				terminal++
				anyCancelled = true
			}
		}
		// If a worker never joined, close its slot after the scheduled run
		// window. This prevents a partially provisioned test from remaining
		// active forever while making the missing capacity visible as failed.
		missingWorkersExpired := run.StartAt > 0 && common.GetTimestamp() >= run.StartAt+int64(run.DurationSeconds)+5 && len(workers) < run.ExpectedWorkers
		if terminal < len(workers) && !missingWorkersExpired {
			return nil
		}
		if missingWorkersExpired {
			anyFailed = true
		}
		if terminal < run.ExpectedWorkers && !missingWorkersExpired {
			return nil
		}
		aggregate := aggregateLoadTestRunWorkers(workers)
		statusValue := LoadTestRunCompleted
		if anyFailed {
			statusValue = LoadTestRunFailed
		} else if anyCancelled {
			statusValue = LoadTestRunCancelled
		}
		errorJSON, err := EncodeLoadTestErrorCounts(aggregate.ErrorCounts)
		if err != nil {
			return err
		}
		finalResult := tx.Model(&LoadTestRun{}).Where("id = ? AND status IN ?", runID, []LoadTestRunStatus{LoadTestRunDispatched, LoadTestRunRunning, LoadTestRunCancelRequested}).Updates(map[string]any{
			"status": statusValue, "finished_at": common.GetTimestamp(), "updated_at": common.GetTimestamp(),
			"sent": aggregate.Sent, "completed": aggregate.Completed, "successes": aggregate.Successes,
			"failures": aggregate.Failures, "dropped": aggregate.Dropped, "input_tokens": aggregate.InputTokens,
			"output_tokens": aggregate.OutputTokens, "cache_read_tokens": aggregate.CacheReadTokens,
			"cache_write_tokens": aggregate.CacheWriteTokens, "usage_missing": aggregate.UsageMissing, "token_stats_source": aggregate.TokenStatsSource, "current_rps": aggregate.CurrentRPS,
			"p50_ms": aggregate.P50MS, "p95_ms": aggregate.P95MS, "p99_ms": aggregate.P99MS,
			"error_counts_json": errorJSON,
		})
		if finalResult.Error != nil {
			return finalResult.Error
		}
		if finalResult.RowsAffected == 0 || LOG_DB == nil || aggregate.Successes <= 0 {
			return nil
		}
		// The consume log is authoritative for billing. Only replace worker
		// counters when every successful request has produced a correlated log;
		// otherwise retain the Agent response counters and expose that source.
		stats, statsErr := GetLoadTestTokenStatsByRunID(run.UserID, run.ID)
		if statsErr != nil || stats.Requests < aggregate.Successes {
			return nil
		}
		return tx.Model(&LoadTestRun{}).Where("id = ?", runID).Updates(map[string]any{
			"input_tokens": stats.InputTokens, "output_tokens": stats.OutputTokens,
			"cache_read_tokens": stats.CacheReadTokens, "cache_write_tokens": stats.CacheWriteTokens,
			"token_stats_source": "server_logs", "updated_at": common.GetTimestamp(),
		}).Error
	})
}

func GetLoadTestRunAggregate(runID string) (LoadTestRunAggregate, error) {
	var workers []LoadTestRunWorker
	if err := DB.Where("run_id = ?", runID).Find(&workers).Error; err != nil {
		return LoadTestRunAggregate{}, err
	}
	return aggregateLoadTestRunWorkers(workers), nil
}

func aggregateLoadTestRunWorkers(workers []LoadTestRunWorker) LoadTestRunAggregate {
	aggregate := LoadTestRunAggregate{ErrorCounts: map[string]int64{}}
	for index := range workers {
		worker := &workers[index]
		aggregate.Sent += worker.Sent
		aggregate.Completed += worker.Completed
		aggregate.Successes += worker.Successes
		aggregate.Failures += worker.Failures
		aggregate.Dropped += worker.Dropped
		aggregate.InputTokens += worker.InputTokens
		aggregate.OutputTokens += worker.OutputTokens
		aggregate.CacheReadTokens += worker.CacheReadTokens
		aggregate.CacheWriteTokens += worker.CacheWriteTokens
		aggregate.UsageMissing += worker.UsageMissing
		if worker.TokenStatsSource == "server_logs" {
			aggregate.TokenStatsSource = "server_logs"
		} else if aggregate.TokenStatsSource == "" && worker.TokenStatsSource != "" {
			aggregate.TokenStatsSource = worker.TokenStatsSource
		}
		aggregate.CurrentRPS += worker.CurrentRPS
		aggregate.P50MS = max(aggregate.P50MS, worker.P50MS)
		aggregate.P95MS = max(aggregate.P95MS, worker.P95MS)
		aggregate.P99MS = max(aggregate.P99MS, worker.P99MS)
		if worker.ErrorCounts == nil && worker.ErrorCountsJSON != "" {
			_ = common.UnmarshalJsonStr(worker.ErrorCountsJSON, &worker.ErrorCounts)
		}
		for code, count := range worker.ErrorCounts {
			aggregate.ErrorCounts[code] += count
		}
	}
	return aggregate
}

func hydrateLoadTestRunAggregate(run *LoadTestRun) {
	if run == nil {
		return
	}
	if run.ExecutionMode == LoadTestExecutionShared {
		aggregate, err := GetLoadTestRunAggregate(run.ID)
		if err != nil {
			return
		}
		run.Sent = aggregate.Sent
		run.Completed = aggregate.Completed
		run.Successes = aggregate.Successes
		run.Failures = aggregate.Failures
		run.Dropped = aggregate.Dropped
		run.InputTokens = aggregate.InputTokens
		run.OutputTokens = aggregate.OutputTokens
		run.CacheReadTokens = aggregate.CacheReadTokens
		run.CacheWriteTokens = aggregate.CacheWriteTokens
		run.UsageMissing = aggregate.UsageMissing
		run.TokenStatsSource = aggregate.TokenStatsSource
		run.CurrentRPS = aggregate.CurrentRPS
		run.P50MS = aggregate.P50MS
		run.P95MS = aggregate.P95MS
		run.P99MS = aggregate.P99MS
		run.ErrorCounts = aggregate.ErrorCounts
		var workers []LoadTestRunWorker
		if DB.Where("run_id = ?", run.ID).Order("slot asc").Find(&workers).Error == nil {
			for index := range workers {
				decodeLoadTestRunWorkerJSON(&workers[index])
			}
			run.Workers = workers
		}
	}
	if LOG_DB != nil && (run.Status == LoadTestRunCompleted || run.Status == LoadTestRunFailed || run.Status == LoadTestRunCancelled) && run.Successes > 0 {
		if stats, statsErr := GetLoadTestTokenStatsByRunID(run.UserID, run.ID); statsErr == nil && stats.Requests >= run.Successes {
			run.InputTokens = stats.InputTokens
			run.OutputTokens = stats.OutputTokens
			run.CacheReadTokens = stats.CacheReadTokens
			run.CacheWriteTokens = stats.CacheWriteTokens
			run.TokenStatsSource = "server_logs"
		}
	}
}

func decodeLoadTestRunWorkerJSON(worker *LoadTestRunWorker) {
	worker.ErrorCounts = map[string]int64{}
	if worker.ErrorCountsJSON != "" {
		_ = common.UnmarshalJsonStr(worker.ErrorCountsJSON, &worker.ErrorCounts)
	}
}

func ListLoadTestRuns(userID, limit int) ([]LoadTestRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var runs []LoadTestRun
	err := DB.Where("user_id = ?", userID).Order("created_at desc").Limit(limit).Find(&runs).Error
	for index := range runs {
		decodeLoadTestRunJSON(&runs[index])
		hydrateLoadTestRunAggregate(&runs[index])
	}
	return runs, err
}

func GetLoadTestRun(userID int, runID string) (*LoadTestRun, error) {
	var run LoadTestRun
	if err := DB.Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
		return nil, err
	}
	decodeLoadTestRunJSON(&run)
	hydrateLoadTestRunAggregate(&run)
	return &run, nil
}

func GetLoadTestRunByID(runID string) (*LoadTestRun, error) {
	var run LoadTestRun
	if err := DB.Where("id = ?", runID).First(&run).Error; err != nil {
		return nil, err
	}
	decodeLoadTestRunJSON(&run)
	hydrateLoadTestRunAggregate(&run)
	return &run, nil
}

func ClaimLoadTestRun(agentID string) (*LoadTestRun, error) {
	var run LoadTestRun
	err := DB.Where("agent_id = ? AND status = ? AND (execution_mode <> ? OR execution_mode IS NULL OR execution_mode = ?)", agentID, LoadTestRunQueued, LoadTestExecutionShared, "").Order("created_at asc").First(&run).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	now := common.GetTimestamp()
	result := DB.Model(&LoadTestRun{}).
		Where("id = ? AND agent_id = ? AND status = ?", run.ID, agentID, LoadTestRunQueued).
		Updates(map[string]any{"status": LoadTestRunDispatched, "started_at": now, "updated_at": now})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	if err := DB.Where("id = ?", run.ID).First(&run).Error; err != nil {
		return nil, err
	}
	decodeLoadTestRunJSON(&run)
	return &run, nil
}

func GetAgentCancelRequest(agentID string) (*LoadTestRun, error) {
	var run LoadTestRun
	err := DB.Where("agent_id = ? AND status = ?", agentID, LoadTestRunCancelRequested).Order("updated_at asc").First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &run, err
}

func UpdateLoadTestRunProgress(agentID, runID string, updates map[string]any) error {
	updates["status"] = LoadTestRunRunning
	updates["updated_at"] = common.GetTimestamp()
	result := DB.Model(&LoadTestRun{}).
		Where("id = ? AND agent_id = ? AND status IN ?", runID, agentID, []LoadTestRunStatus{LoadTestRunDispatched, LoadTestRunRunning}).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("load-test run is not active")
	}
	return nil
}

func FinishLoadTestRun(agentID, runID string, status LoadTestRunStatus, updates map[string]any) error {
	if status != LoadTestRunCompleted && status != LoadTestRunFailed && status != LoadTestRunCancelled {
		return errors.New("invalid terminal load-test status")
	}
	now := common.GetTimestamp()
	updates["status"] = status
	updates["finished_at"] = now
	updates["updated_at"] = now
	result := DB.Model(&LoadTestRun{}).
		Where("id = ? AND agent_id = ? AND status IN ?", runID, agentID, []LoadTestRunStatus{LoadTestRunDispatched, LoadTestRunRunning, LoadTestRunCancelRequested}).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("load-test run is not active")
	}
	return nil
}

// ReconcileLoadTestRunTokenStats replaces Agent-reported token counters with
// authoritative consume-log totals after a terminal run, including shared
// runs where workers only report local response counters.
func ReconcileLoadTestRunTokenStats(runID string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return errors.New("run identifier is required")
	}
	var run LoadTestRun
	if err := DB.Where("id = ?", runID).First(&run).Error; err != nil {
		return err
	}
	if run.Status != LoadTestRunCompleted && run.Status != LoadTestRunFailed && run.Status != LoadTestRunCancelled {
		return nil
	}
	stats, err := GetLoadTestTokenStatsByRunID(run.UserID, run.ID)
	if err != nil {
		return err
	}
	if stats.Requests < run.Successes {
		return nil
	}
	return DB.Model(&LoadTestRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"input_tokens": stats.InputTokens, "output_tokens": stats.OutputTokens,
		"cache_read_tokens": stats.CacheReadTokens, "cache_write_tokens": stats.CacheWriteTokens,
		"token_stats_source": "server_logs", "updated_at": common.GetTimestamp(),
	}).Error
}

func RequestLoadTestRunCancellation(userID int, runID string) error {
	now := common.GetTimestamp()
	queued := DB.Model(&LoadTestRun{}).
		Where("id = ? AND user_id = ? AND status = ?", runID, userID, LoadTestRunQueued).
		Updates(map[string]any{"status": LoadTestRunCancelled, "finished_at": now, "updated_at": now})
	if queued.Error != nil {
		return queued.Error
	}
	if queued.RowsAffected > 0 {
		return nil
	}
	result := DB.Model(&LoadTestRun{}).
		Where("id = ? AND user_id = ? AND status IN ?", runID, userID, []LoadTestRunStatus{LoadTestRunDispatched, LoadTestRunRunning, LoadTestRunCancelRequested}).
		Updates(map[string]any{"status": LoadTestRunCancelRequested, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("load-test run cannot be cancelled")
	}
	return nil
}

func EncodeLoadTestErrorCounts(counts map[string]int64) (string, error) {
	if len(counts) == 0 {
		return "", nil
	}
	data, err := common.Marshal(counts)
	return string(data), err
}

func EncodeLoadTestMockChannels(channels []LoadTestMockChannel) (string, error) {
	if len(channels) == 0 {
		return "", nil
	}
	data, err := common.Marshal(channels)
	return string(data), err
}

func decodeLoadTestRunJSON(run *LoadTestRun) {
	run.ErrorCounts = map[string]int64{}
	if run.ErrorCountsJSON != "" {
		_ = common.UnmarshalJsonStr(run.ErrorCountsJSON, &run.ErrorCounts)
	}
	run.MockChannels = []LoadTestMockChannel{}
	if run.MockChannelsJSON != "" {
		_ = common.UnmarshalJsonStr(run.MockChannelsJSON, &run.MockChannels)
	}
}
