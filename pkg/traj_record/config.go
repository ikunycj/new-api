// Package traj_record stores coding-agent trajectories passing through the
// gateway: every call of a recorded client (Claude Code on /v1/messages, Codex
// on /v1/responses) with its raw response, the conversation history
// deduplicated against earlier requests of the same session.
//
// The design rationale lives in README.md next to this file.
package traj_record

import (
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// Config controls trajectory capture. All values come from environment
// variables so the feature can be toggled without a code change.
type Config struct {
	// Enabled is the master switch. When false the middleware returns
	// immediately and allocates nothing.
	Enabled bool

	// Dir holds traj.jsonl and its rotated, compressed predecessors.
	Dir string

	// Instance identifies this gateway process in every record, so files from
	// several instances can be merged offline. Defaults to the hostname.
	Instance string

	// Clients maps each recorded client (see clients.go) to the User-Agent
	// prefixes that identify it. Defaults to every known client with its
	// default prefixes.
	Clients map[string][]string

	// Users optionally restricts capture to these user ids. Empty records all.
	Users map[int]bool

	// SampleRate is the fraction of sessions captured. Nil means 1 (all).
	// A pointer distinguishes an explicit zero from the default.
	SampleRate *float64

	// QueueSize bounds how many finished requests may wait for the writer.
	// Each entry holds a full request and response, so this also bounds memory.
	QueueSize int

	// EnqueueTimeoutMs is how long a request may wait for queue space before
	// its record is dropped. Dropping is counted and logged, never silent.
	EnqueueTimeoutMs int

	// MaxRespMB caps the captured response. Responses are never clipped below
	// this; the cap only exists so a runaway stream cannot exhaust memory.
	MaxRespMB int

	// MaxFileMB triggers rotation once the active file exceeds this size.
	MaxFileMB int

	// MaxBackups is how many rotated files to keep. 0 keeps everything,
	// which is the default because the files are training data, not logs.
	MaxBackups int

	// ZstdWindowMB is the zstd window used to compress rotated files. A long
	// window lets compression exploit repetition across records (shared system
	// prompts, tool definitions, re-sent messages). 0 disables compression.
	ZstdWindowMB int

	// MinFreeDiskMB pauses writing while the filesystem has less free space
	// than this. 0 disables the guard.
	MinFreeDiskMB int

	// IndexTTLMin forgets sessions idle for longer than this. A request in a
	// forgotten session is written as a full keyframe, so this only trades
	// memory for file size, never correctness.
	IndexTTLMin int

	// IndexMaxEntries bounds the prefix index (one entry per stored message).
	// Least recently used sessions are evicted beyond it.
	IndexMaxEntries int
}

const (
	EnvEnabled         = "TRAJ_RECORD_ENABLED"
	EnvDir             = "TRAJ_RECORD_DIR"
	EnvInstance        = "TRAJ_RECORD_INSTANCE"
	EnvClients         = "TRAJ_RECORD_CLIENTS"
	EnvUsers           = "TRAJ_RECORD_USERS"
	EnvSampleRate      = "TRAJ_RECORD_SAMPLE_RATE"
	EnvQueueSize       = "TRAJ_RECORD_QUEUE_SIZE"
	EnvEnqueueTimeout  = "TRAJ_RECORD_ENQUEUE_TIMEOUT_MS"
	EnvMaxRespMB       = "TRAJ_RECORD_MAX_RESP_MB"
	EnvMaxFileMB       = "TRAJ_RECORD_MAX_FILE_MB"
	EnvMaxBackups      = "TRAJ_RECORD_MAX_BACKUPS"
	EnvZstdWindowMB    = "TRAJ_RECORD_ZSTD_WINDOW_MB"
	EnvMinFreeDiskMB   = "TRAJ_RECORD_MIN_FREE_DISK_MB"
	EnvIndexTTLMin     = "TRAJ_RECORD_INDEX_TTL_MIN"
	EnvIndexMaxEntries = "TRAJ_RECORD_INDEX_MAX_ENTRIES"
)

const (
	defaultDir              = "./logs/traj-record"
	defaultQueueSize        = 256
	defaultEnqueueTimeoutMs = 200
	defaultMaxRespMB        = 64
	defaultMaxFileMB        = 1024
	defaultZstdWindowMB     = 128
	defaultMinFreeDiskMB    = 2048
	defaultIndexTTLMin      = 360
	defaultIndexMaxEntries  = 2_000_000
)

// ConfigFromEnv builds a Config from environment variables, clamping each value
// into a sane range so a typo can never produce an unusable configuration.
func ConfigFromEnv() Config {
	cfg := Config{
		Enabled:          common.GetEnvOrDefaultBool(EnvEnabled, false),
		Dir:              common.GetEnvOrDefaultString(EnvDir, defaultDir),
		Instance:         common.GetEnvOrDefaultString(EnvInstance, ""),
		QueueSize:        common.GetEnvOrDefault(EnvQueueSize, defaultQueueSize),
		EnqueueTimeoutMs: common.GetEnvOrDefault(EnvEnqueueTimeout, defaultEnqueueTimeoutMs),
		MaxRespMB:        common.GetEnvOrDefault(EnvMaxRespMB, defaultMaxRespMB),
		MaxFileMB:        common.GetEnvOrDefault(EnvMaxFileMB, defaultMaxFileMB),
		MaxBackups:       common.GetEnvOrDefault(EnvMaxBackups, 0),
		ZstdWindowMB:     common.GetEnvOrDefault(EnvZstdWindowMB, defaultZstdWindowMB),
		MinFreeDiskMB:    common.GetEnvOrDefault(EnvMinFreeDiskMB, defaultMinFreeDiskMB),
		IndexTTLMin:      common.GetEnvOrDefault(EnvIndexTTLMin, defaultIndexTTLMin),
		IndexMaxEntries:  common.GetEnvOrDefault(EnvIndexMaxEntries, defaultIndexMaxEntries),
	}
	if raw, ok := os.LookupEnv(EnvSampleRate); ok {
		rate, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil || math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 || rate > 1 {
			common.SysError("traj record: invalid sample rate; capture disabled")
			rate = 0
		}
		cfg.SampleRate = &rate
	}
	allNames := make([]string, len(clients))
	for i, cl := range clients {
		allNames[i] = cl.name
	}
	for _, name := range splitList(common.GetEnvOrDefaultString(EnvClients, strings.Join(allNames, ","))) {
		cl := clientByName(name)
		if cl == nil {
			common.SysError("traj record: unknown client " + name)
			continue
		}
		if cfg.Clients == nil {
			cfg.Clients = map[string][]string{}
		}
		cfg.Clients[name] = splitList(common.GetEnvOrDefaultString(uaPrefixesEnv(name), strings.Join(cl.uaPrefixes, ",")))
	}
	for _, s := range splitList(common.GetEnvOrDefaultString(EnvUsers, "")) {
		if id, err := strconv.Atoi(s); err == nil {
			if cfg.Users == nil {
				cfg.Users = map[int]bool{}
			}
			cfg.Users[id] = true
		}
	}
	cfg.normalize()
	return cfg
}

func (c *Config) normalize() {
	if c.SampleRate != nil {
		rate := *c.SampleRate
		if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 || rate > 1 {
			common.SysError("traj record: invalid sample rate; capture disabled")
			rate = 0
		}
		c.SampleRate = &rate
	}
	if c.Dir == "" {
		c.Dir = defaultDir
	}
	if c.Instance == "" {
		c.Instance, _ = os.Hostname()
	}
	if len(c.Clients) == 0 {
		c.Clients = map[string][]string{}
		for _, cl := range clients {
			c.Clients[cl.name] = cl.uaPrefixes
		}
	}
	if c.QueueSize <= 0 {
		c.QueueSize = defaultQueueSize
	}
	if c.EnqueueTimeoutMs < 0 {
		c.EnqueueTimeoutMs = 0
	}
	if c.MaxRespMB <= 0 {
		c.MaxRespMB = defaultMaxRespMB
	}
	if c.MaxFileMB <= 0 {
		c.MaxFileMB = defaultMaxFileMB
	}
	c.MaxBackups = max(c.MaxBackups, 0)
	c.ZstdWindowMB = max(c.ZstdWindowMB, 0)
	c.MinFreeDiskMB = max(c.MinFreeDiskMB, 0)
	if c.IndexTTLMin <= 0 {
		c.IndexTTLMin = defaultIndexTTLMin
	}
	if c.IndexMaxEntries <= 0 {
		c.IndexMaxEntries = defaultIndexMaxEntries
	}
}

// uaPrefixesEnv names the variable overriding a client's User-Agent prefixes,
// e.g. TRAJ_RECORD_CODEX_UA_PREFIXES.
func uaPrefixesEnv(client string) string {
	return "TRAJ_RECORD_" + strings.ToUpper(client) + "_UA_PREFIXES"
}

func splitList(raw string) []string {
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
