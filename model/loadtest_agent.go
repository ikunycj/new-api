package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
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
	ID                string            `json:"id" gorm:"type:varchar(64);primaryKey"`
	UserID            int               `json:"user_id" gorm:"index;not null"`
	AgentID           string            `json:"agent_id" gorm:"type:varchar(64);index;not null"`
	TokenID           int               `json:"token_id" gorm:"index;not null"`
	KeyName           string            `json:"key_name" gorm:"type:varchar(128);not null"`
	PackageName       string            `json:"package_name" gorm:"type:varchar(64);not null"`
	Model             string            `json:"model" gorm:"type:varchar(128);not null"`
	Endpoint          string            `json:"endpoint" gorm:"type:varchar(32);not null"`
	Prompt            string            `json:"prompt" gorm:"type:text;not null"`
	PromptCache       bool              `json:"prompt_cache"`
	MockEnabled       bool              `json:"mock_enabled"`
	MockFailureRate   float64           `json:"mock_failure_rate"`
	MockFailureStatus int               `json:"mock_failure_status"`
	MockLatencyMS     int               `json:"mock_latency_ms"`
	TargetURL         string            `json:"-" gorm:"type:varchar(512);not null"`
	DurationSeconds   int               `json:"duration_seconds" gorm:"not null"`
	RequestsPerSecond int               `json:"requests_per_second" gorm:"not null"`
	Concurrency       int               `json:"concurrency" gorm:"not null"`
	AgentManaged      bool              `json:"agent_managed" gorm:"index"`
	Status            LoadTestRunStatus `json:"status" gorm:"type:varchar(32);index;not null"`
	Sent              int64             `json:"sent" gorm:"bigint"`
	Completed         int64             `json:"completed" gorm:"bigint"`
	Successes         int64             `json:"successes" gorm:"bigint"`
	Failures          int64             `json:"failures" gorm:"bigint"`
	Dropped           int64             `json:"dropped" gorm:"bigint"`
	InputTokens       int64             `json:"input_tokens" gorm:"bigint"`
	OutputTokens      int64             `json:"output_tokens" gorm:"bigint"`
	CacheReadTokens   int64             `json:"cache_read_tokens" gorm:"bigint"`
	CacheWriteTokens  int64             `json:"cache_write_tokens" gorm:"bigint"`
	CurrentRPS        float64           `json:"current_rps"`
	P50MS             float64           `json:"p50_ms"`
	P95MS             float64           `json:"p95_ms"`
	P99MS             float64           `json:"p99_ms"`
	ErrorCountsJSON   string            `json:"-" gorm:"type:text"`
	ErrorCounts       map[string]int64  `json:"error_counts" gorm:"-"`
	ErrorMessage      string            `json:"error_message" gorm:"type:text"`
	CreatedAt         int64             `json:"created_at" gorm:"bigint;index"`
	StartedAt         int64             `json:"started_at" gorm:"bigint"`
	FinishedAt        int64             `json:"finished_at" gorm:"bigint"`
	UpdatedAt         int64             `json:"updated_at" gorm:"bigint;index"`
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
	run.CreatedAt = now
	run.UpdatedAt = now
	return DB.Create(run).Error
}

func ListLoadTestRuns(userID, limit int) ([]LoadTestRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var runs []LoadTestRun
	err := DB.Where("user_id = ?", userID).Order("created_at desc").Limit(limit).Find(&runs).Error
	for index := range runs {
		decodeLoadTestRunErrors(&runs[index])
	}
	return runs, err
}

func GetLoadTestRun(userID int, runID string) (*LoadTestRun, error) {
	var run LoadTestRun
	if err := DB.Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
		return nil, err
	}
	decodeLoadTestRunErrors(&run)
	return &run, nil
}

func ClaimLoadTestRun(agentID string) (*LoadTestRun, error) {
	var run LoadTestRun
	err := DB.Where("agent_id = ? AND status = ?", agentID, LoadTestRunQueued).Order("created_at asc").First(&run).Error
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

func decodeLoadTestRunErrors(run *LoadTestRun) {
	run.ErrorCounts = map[string]int64{}
	if run.ErrorCountsJSON != "" {
		_ = common.UnmarshalJsonStr(run.ErrorCountsJSON, &run.ErrorCounts)
	}
}
