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
	"github.com/go-redis/redis/v8"
)

const channelCircuitAllowScript = `
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

const channelCircuitFailureScript = `
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

type channelCircuit struct {
	failures        int
	windowStartedAt time.Time
	openUntil       time.Time
	halfOpenProbes  int
}

var channelCircuits = struct {
	sync.Mutex
	values map[string]*channelCircuit
}{values: make(map[string]*channelCircuit)}

func ChannelCircuitAllows(channelID int, route string, policy model.RuntimeRoutingPolicy) bool {
	if channelID <= 0 {
		return true
	}
	if common.RedisEnabled && common.RDB != nil {
		state, err := common.RDB.Eval(
			context.Background(),
			channelCircuitAllowScript,
			[]string{channelCircuitRedisKey(channelID, route)},
			time.Now().UnixMilli(),
			policy.CircuitHalfOpenRequests,
			circuitRedisTTL(policy).Milliseconds(),
		).Int64()
		if err == nil {
			switch state {
			case 0:
				observability.SetChannelCircuitState(channelID, route, "open")
				return false
			case 2:
				observability.SetChannelCircuitState(channelID, route, "half_open")
				return false
			case 3:
				observability.SetChannelCircuitState(channelID, route, "half_open")
				return true
			default:
				observability.SetChannelCircuitState(channelID, route, "closed")
				return true
			}
		}
		common.SysError(fmt.Sprintf("failover circuit Redis allow failed: %v", err))
	}
	return localChannelCircuitAllows(channelID, route, policy)
}

// ChannelCircuitIsOpen is a read-only circuit check for candidate building.
// Unlike ChannelCircuitAllows it does not consume a half-open probe slot; the
// selected request reserves that slot immediately before it is sent.
func ChannelCircuitIsOpen(channelID int, route string) bool {
	if channelID <= 0 || route == "" {
		return false
	}
	now := time.Now()
	if common.RedisEnabled && common.RDB != nil {
		openUntil, err := common.RDB.HGet(context.Background(), channelCircuitRedisKey(channelID, route), "open_until_ms").Int64()
		if err == nil {
			return openUntil > now.UnixMilli()
		}
		// A missing key is the normal closed-circuit state. Other Redis errors
		// fall through to the local mirror rather than making routing fail open
		// with a stale local value.
		if err != redis.Nil {
			common.SysError(fmt.Sprintf("failover circuit Redis read failed: %v", err))
		}
	}
	key := channelCircuitKey(channelID, route)
	channelCircuits.Lock()
	defer channelCircuits.Unlock()
	state := channelCircuits.values[key]
	return state != nil && state.openUntil.After(now)
}

func localChannelCircuitAllows(channelID int, route string, policy model.RuntimeRoutingPolicy) bool {
	key := channelCircuitKey(channelID, route)
	now := time.Now()
	channelCircuits.Lock()
	defer channelCircuits.Unlock()
	state := channelCircuits.values[key]
	if state == nil {
		observability.SetChannelCircuitState(channelID, route, "closed")
		return true
	}
	if state.openUntil.After(now) {
		observability.SetChannelCircuitState(channelID, route, "open")
		return false
	}
	if !state.openUntil.IsZero() {
		if state.halfOpenProbes >= policy.CircuitHalfOpenRequests {
			observability.SetChannelCircuitState(channelID, route, "half_open")
			return false
		}
		state.halfOpenProbes++
		observability.SetChannelCircuitState(channelID, route, "half_open")
	}
	return true
}

func RecordChannelCircuitSuccess(channelID int, route string) {
	if channelID <= 0 {
		return
	}
	if common.RedisEnabled && common.RDB != nil {
		if err := common.RDB.Del(context.Background(), channelCircuitRedisKey(channelID, route)).Err(); err != nil {
			common.SysError(fmt.Sprintf("failover circuit Redis reset failed: %v", err))
		}
	}
	key := channelCircuitKey(channelID, route)
	channelCircuits.Lock()
	delete(channelCircuits.values, key)
	channelCircuits.Unlock()
	observability.SetChannelCircuitState(channelID, route, "closed")
}

func RecordChannelCircuitFailure(channelID int, route string, policy model.RuntimeRoutingPolicy) {
	if channelID <= 0 {
		return
	}
	if common.RedisEnabled && common.RDB != nil {
		isOpen, err := common.RDB.Eval(
			context.Background(),
			channelCircuitFailureScript,
			[]string{channelCircuitRedisKey(channelID, route)},
			time.Now().UnixMilli(),
			int64(time.Duration(policy.CircuitWindowSeconds)*time.Second/time.Millisecond),
			policy.CircuitFailureThreshold,
			int64(time.Duration(policy.CircuitCooldownSeconds)*time.Second/time.Millisecond),
			circuitRedisTTL(policy).Milliseconds(),
		).Int64()
		if err == nil {
			if isOpen == 1 {
				observability.SetChannelCircuitState(channelID, route, "open")
			}
			return
		}
		common.SysError(fmt.Sprintf("failover circuit Redis failure update failed: %v", err))
	}
	localRecordChannelCircuitFailure(channelID, route, policy)
}

func localRecordChannelCircuitFailure(channelID int, route string, policy model.RuntimeRoutingPolicy) {
	key := channelCircuitKey(channelID, route)
	now := time.Now()
	channelCircuits.Lock()
	state := channelCircuits.values[key]
	if state == nil {
		state = &channelCircuit{windowStartedAt: now}
		channelCircuits.values[key] = state
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
	channelCircuits.Unlock()
	if isOpen {
		observability.SetChannelCircuitState(channelID, route, "open")
	}
}

func channelCircuitKey(channelID int, route string) string {
	return fmt.Sprintf("%d:%s", channelID, route)
}

func channelCircuitRedisKey(channelID int, route string) string {
	routeHash := sha256.Sum256([]byte(route))
	return fmt.Sprintf("new_api:routing:channel:circuit:%d:%x", channelID, routeHash[:8])
}

func circuitRedisTTL(policy model.RuntimeRoutingPolicy) time.Duration {
	ttl := time.Duration(policy.CircuitWindowSeconds+policy.CircuitCooldownSeconds) * time.Second
	if ttl < time.Minute {
		return time.Minute
	}
	return ttl * 2
}
