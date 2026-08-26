/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package service

import (
	"sync"
	"sync/atomic"
)

type channelConcurrencyState struct {
	current atomic.Int64
}

var channelConcurrencyStates sync.Map // map[int]*channelConcurrencyState

func channelConcurrencyStateFor(channelID int) *channelConcurrencyState {
	if state, ok := channelConcurrencyStates.Load(channelID); ok {
		return state.(*channelConcurrencyState)
	}
	state := &channelConcurrencyState{}
	actual, _ := channelConcurrencyStates.LoadOrStore(channelID, state)
	return actual.(*channelConcurrencyState)
}

// TryAcquireChannelConcurrency reserves one active request for a channel. The
// limit is evaluated atomically with the increment, so concurrent requests
// cannot exceed the configured maximum.
func TryAcquireChannelConcurrency(channelID, limit int) bool {
	if channelID <= 0 || limit <= 0 {
		return false
	}
	state := channelConcurrencyStateFor(channelID)
	for {
		current := state.current.Load()
		if current >= int64(limit) {
			return false
		}
		if state.current.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

// ReleaseChannelConcurrency releases a reservation previously acquired for a
// channel. It is intentionally idempotent when no reservation is present.
func ReleaseChannelConcurrency(channelID int) {
	if channelID <= 0 {
		return
	}
	stateValue, ok := channelConcurrencyStates.Load(channelID)
	if !ok {
		return
	}
	state := stateValue.(*channelConcurrencyState)
	for {
		current := state.current.Load()
		if current <= 0 {
			return
		}
		if state.current.CompareAndSwap(current, current-1) {
			return
		}
	}
}

func CurrentChannelConcurrency(channelID int) int {
	if channelID <= 0 {
		return 0
	}
	stateValue, ok := channelConcurrencyStates.Load(channelID)
	if !ok {
		return 0
	}
	return int(stateValue.(*channelConcurrencyState).current.Load())
}
