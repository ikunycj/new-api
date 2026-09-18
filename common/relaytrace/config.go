// Package relaytrace records user-facing relay request/response bodies into
// local JSONL files for audit and post-hoc debugging.
//
// Design decisions (see docs/relay-trace.md):
//   - JSONL (one JSON object per line), not a single big JSON array: append-only
//     writes, a corrupted line never invalidates the file, and grep/jq work directly.
//   - Local file, not DB / MQ: the existing `logs` table is the hottest write path
//     in the system and is queried with SELECT *; adding multi-KB body columns there
//     would slow every log page and add TOAST write amplification. Local sequential
//     append costs microseconds instead of milliseconds.
//   - Metadata is NOT duplicated here. Only `rid` (request id) is stored, which joins
//     back to the `logs` table for user/token/channel/model/quota. This keeps traces
//     small and keeps `logs` authoritative.
package relaytrace

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// Config controls trace collection and persistence. All values come from
// environment variables so the feature can be toggled without a code change.
type Config struct {
	// Enabled is the master switch. When false, Submit is a no-op and the
	// middleware returns immediately without touching the request.
	Enabled bool

	// Dir is the directory holding trace.jsonl and its rotated backups.
	Dir string

	// SampleRate is an integer percentage in [0,100]. 0 disables sampling of
	// successful requests (errors may still be recorded, see ErrorAlways).
	SampleRate int

	// ErrorAlways records every non-2xx response regardless of SampleRate.
	// Errors are the most valuable samples and are rare, so this is on by default.
	ErrorAlways bool

	// MaxBodyKB caps each of request/response body separately. Bodies beyond
	// the cap are truncated and flagged, bounding memory per in-flight request.
	MaxBodyKB int

	// QueueSize is the capacity of the submit channel. When full, records are
	// dropped rather than blocking the relay path.
	QueueSize int

	// MaxFileMB triggers rotation once the active file exceeds this size.
	MaxFileMB int

	// MaxBackups is how many rotated files to keep; older ones are deleted.
	MaxBackups int

	// Users and Models are allow-lists that force 100% recording when matched,
	// for targeted debugging without raising the global sample rate.
	Users  map[int]bool
	Models map[string]bool

	// DetectSignature scans Claude responses for a non-empty `signature` and
	// records the result as `has_signature`, so channels returning empty
	// thinking signatures can be found with a single jq filter instead of
	// re-parsing SSE after the fact.
	DetectSignature bool
}

// Environment variable names.
const (
	EnvEnabled         = "RELAY_TRACE_ENABLED"
	EnvDir             = "RELAY_TRACE_DIR"
	EnvSampleRate      = "RELAY_TRACE_SAMPLE_RATE"
	EnvErrorAlways     = "RELAY_TRACE_ERROR_ALWAYS"
	EnvMaxBodyKB       = "RELAY_TRACE_MAX_BODY_KB"
	EnvQueueSize       = "RELAY_TRACE_QUEUE_SIZE"
	EnvMaxFileMB       = "RELAY_TRACE_MAX_FILE_MB"
	EnvMaxBackups      = "RELAY_TRACE_MAX_BACKUPS"
	EnvUsers           = "RELAY_TRACE_USERS"
	EnvModels          = "RELAY_TRACE_MODELS"
	EnvDetectSignature = "RELAY_TRACE_DETECT_SIGNATURE"
)

// Default values. The feature is off by default; when switched on, the
// recommended production setting is SampleRate=0 + ErrorAlways=true, which
// records only failures and keeps storage negligible.
const (
	defaultDir        = "./logs/relay-trace"
	defaultSampleRate = 0
	defaultMaxBodyKB  = 256
	defaultQueueSize  = 8192
	defaultMaxFileMB  = 512
	defaultMaxBackups = 30
)

// ConfigFromEnv builds a Config from environment variables, clamping each value
// into a sane range so a typo can never produce an unusable configuration.
func ConfigFromEnv() Config {
	cfg := Config{
		Enabled:         common.GetEnvOrDefaultBool(EnvEnabled, false),
		Dir:             common.GetEnvOrDefaultString(EnvDir, defaultDir),
		SampleRate:      common.GetEnvOrDefault(EnvSampleRate, defaultSampleRate),
		ErrorAlways:     common.GetEnvOrDefaultBool(EnvErrorAlways, true),
		MaxBodyKB:       common.GetEnvOrDefault(EnvMaxBodyKB, defaultMaxBodyKB),
		QueueSize:       common.GetEnvOrDefault(EnvQueueSize, defaultQueueSize),
		MaxFileMB:       common.GetEnvOrDefault(EnvMaxFileMB, defaultMaxFileMB),
		MaxBackups:      common.GetEnvOrDefault(EnvMaxBackups, defaultMaxBackups),
		Users:           parseIntSet(common.GetEnvOrDefaultString(EnvUsers, "")),
		Models:          parseStringSet(common.GetEnvOrDefaultString(EnvModels, "")),
		DetectSignature: common.GetEnvOrDefaultBool(EnvDetectSignature, true),
	}
	cfg.normalize()
	return cfg
}

// normalize clamps out-of-range values to defaults or bounds.
func (c *Config) normalize() {
	if c.Dir == "" {
		c.Dir = defaultDir
	}
	if c.SampleRate < 0 {
		c.SampleRate = 0
	}
	if c.SampleRate > 100 {
		c.SampleRate = 100
	}
	if c.MaxBodyKB <= 0 {
		c.MaxBodyKB = defaultMaxBodyKB
	}
	if c.QueueSize <= 0 {
		c.QueueSize = defaultQueueSize
	}
	if c.MaxFileMB <= 0 {
		c.MaxFileMB = defaultMaxFileMB
	}
	if c.MaxBackups < 0 {
		c.MaxBackups = 0
	}
}

// MaxBodyBytes is MaxBodyKB expressed in bytes.
func (c *Config) MaxBodyBytes() int {
	return c.MaxBodyKB << 10
}

// MaxFileBytes is MaxFileMB expressed in bytes.
func (c *Config) MaxFileBytes() int64 {
	return int64(c.MaxFileMB) << 20
}

// parseIntSet parses "1,2,3" into a set, ignoring blanks and non-numeric items.
func parseIntSet(raw string) map[int]bool {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	out := make(map[int]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if n, err := strconv.Atoi(part); err == nil {
			out[n] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseStringSet parses "a,b,c" into a set, ignoring blanks.
func parseStringSet(raw string) map[string]bool {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	out := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out[part] = true
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
