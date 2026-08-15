package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/observability"
)

const clusterCircuitAllowScript = `
local open_until = tonumber(redis.call('HGET', KEYS[1], 'open_until_ms') or '0')
if open_until > tonumber(ARGV[1]) then
  return 0
end
if open_until > 0 then
  local probes = tonumber(redis.call('HGET', KEYS[1], 'half_open_probes') or '0')
  if probes >= tonumber(ARGV[2]) then
    return 2
  end
  redis.call('HINCRBY', KEYS[1], 'half_open_probes', 1)
  redis.call('PEXPIRE', KEYS[1], ARGV[3])
  return 3
end
return 1
`

const clusterCircuitFailureScript = `
local now = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local threshold = tonumber(ARGV[3])
local cooldown_ms = tonumber(ARGV[4])
local window_start = tonumber(redis.call('HGET', KEYS[1], 'window_started_ms') or '0')
local failures = tonumber(redis.call('HGET', KEYS[1], 'failures') or '0')
if window_start == 0 or now - window_start > window_ms then
  window_start = now
  failures = 0
end
failures = failures + 1
local open_until = tonumber(redis.call('HGET', KEYS[1], 'open_until_ms') or '0')
if failures >= threshold then
  open_until = now + cooldown_ms
  redis.call('HSET', KEYS[1], 'half_open_probes', 0)
end
redis.call('HSET', KEYS[1], 'window_started_ms', window_start, 'failures', failures, 'open_until_ms', open_until)
redis.call('PEXPIRE', KEYS[1], ARGV[5])
if open_until > now then return 1 end
return 0
`

type clusterCircuit struct {
	failures        int
	windowStartedAt time.Time
	openUntil       time.Time
	halfOpenProbes  int
}

var clusterCircuits = struct {
	sync.Mutex
	values map[string]*clusterCircuit
}{values: make(map[string]*clusterCircuit)}

func ClusterCircuitAllows(clusterID int, route string, policy model.RuntimeFailoverPolicy) bool {
	if clusterID <= 0 {
		return true
	}
	if common.RedisEnabled && common.RDB != nil {
		state, err := common.RDB.Eval(
			context.Background(),
			clusterCircuitAllowScript,
			[]string{clusterCircuitRedisKey(clusterID, route)},
			time.Now().UnixMilli(),
			policy.CircuitHalfOpenRequests,
			circuitRedisTTL(policy).Milliseconds(),
		).Int64()
		if err == nil {
			switch state {
			case 0:
				observability.SetClusterCircuitState(clusterID, route, "open")
				return false
			case 2:
				observability.SetClusterCircuitState(clusterID, route, "half_open")
				return false
			case 3:
				observability.SetClusterCircuitState(clusterID, route, "half_open")
				return true
			default:
				observability.SetClusterCircuitState(clusterID, route, "closed")
				return true
			}
		}
		common.SysError(fmt.Sprintf("failover circuit Redis allow failed: %v", err))
	}
	return localClusterCircuitAllows(clusterID, route, policy)
}

func localClusterCircuitAllows(clusterID int, route string, policy model.RuntimeFailoverPolicy) bool {
	key := clusterCircuitKey(clusterID, route)
	now := time.Now()
	clusterCircuits.Lock()
	defer clusterCircuits.Unlock()
	state := clusterCircuits.values[key]
	if state == nil {
		observability.SetClusterCircuitState(clusterID, route, "closed")
		return true
	}
	if state.openUntil.After(now) {
		observability.SetClusterCircuitState(clusterID, route, "open")
		return false
	}
	if !state.openUntil.IsZero() {
		if state.halfOpenProbes >= policy.CircuitHalfOpenRequests {
			observability.SetClusterCircuitState(clusterID, route, "half_open")
			return false
		}
		state.halfOpenProbes++
		observability.SetClusterCircuitState(clusterID, route, "half_open")
	}
	return true
}

func RecordClusterCircuitSuccess(clusterID int, route string) {
	if clusterID <= 0 {
		return
	}
	if common.RedisEnabled && common.RDB != nil {
		if err := common.RDB.Del(context.Background(), clusterCircuitRedisKey(clusterID, route)).Err(); err != nil {
			common.SysError(fmt.Sprintf("failover circuit Redis reset failed: %v", err))
		}
	}
	key := clusterCircuitKey(clusterID, route)
	clusterCircuits.Lock()
	delete(clusterCircuits.values, key)
	clusterCircuits.Unlock()
	observability.SetClusterCircuitState(clusterID, route, "closed")
}

func RecordClusterCircuitFailure(clusterID int, route string, policy model.RuntimeFailoverPolicy) {
	if clusterID <= 0 {
		return
	}
	if common.RedisEnabled && common.RDB != nil {
		isOpen, err := common.RDB.Eval(
			context.Background(),
			clusterCircuitFailureScript,
			[]string{clusterCircuitRedisKey(clusterID, route)},
			time.Now().UnixMilli(),
			int64(time.Duration(policy.CircuitWindowSeconds)*time.Second/time.Millisecond),
			policy.CircuitFailureThreshold,
			int64(time.Duration(policy.CircuitCooldownSeconds)*time.Second/time.Millisecond),
			circuitRedisTTL(policy).Milliseconds(),
		).Int64()
		if err == nil {
			if isOpen == 1 {
				observability.SetClusterCircuitState(clusterID, route, "open")
			}
			return
		}
		common.SysError(fmt.Sprintf("failover circuit Redis failure update failed: %v", err))
	}
	localRecordClusterCircuitFailure(clusterID, route, policy)
}

func localRecordClusterCircuitFailure(clusterID int, route string, policy model.RuntimeFailoverPolicy) {
	key := clusterCircuitKey(clusterID, route)
	now := time.Now()
	clusterCircuits.Lock()
	state := clusterCircuits.values[key]
	if state == nil {
		state = &clusterCircuit{windowStartedAt: now}
		clusterCircuits.values[key] = state
	}
	window := time.Duration(policy.CircuitWindowSeconds) * time.Second
	if now.Sub(state.windowStartedAt) > window {
		state.failures = 0
		state.windowStartedAt = now
	}
	state.failures++
	if state.failures >= policy.CircuitFailureThreshold {
		state.openUntil = now.Add(time.Duration(policy.CircuitCooldownSeconds) * time.Second)
		state.halfOpenProbes = 0
	}
	isOpen := state.openUntil.After(now)
	clusterCircuits.Unlock()
	if isOpen {
		observability.SetClusterCircuitState(clusterID, route, "open")
	}
}

func clusterCircuitKey(clusterID int, route string) string {
	return fmt.Sprintf("%d:%s", clusterID, route)
}

func clusterCircuitRedisKey(clusterID int, route string) string {
	routeHash := sha256.Sum256([]byte(route))
	return fmt.Sprintf("alltoken:failover:circuit:%d:%x", clusterID, routeHash[:8])
}

func circuitRedisTTL(policy model.RuntimeFailoverPolicy) time.Duration {
	ttl := time.Duration(policy.CircuitWindowSeconds+policy.CircuitCooldownSeconds) * time.Second
	if ttl < time.Minute {
		return time.Minute
	}
	return ttl * 2
}
