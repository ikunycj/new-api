package common

import (
	"testing"

	"github.com/shirou/gopsutil/cpu"
	"github.com/stretchr/testify/require"
)

func TestCPUUsageSeparatesIOWait(t *testing.T) {
	busy, wait, err := cpuUsageFromDelta(
		cpu.TimesStat{User: 1, System: 1, Iowait: 98, Idle: 0},
		cpu.TimesStat{},
	)
	require.NoError(t, err)
	require.InDelta(t, 2, busy, 0.001)
	require.InDelta(t, 98, wait, 0.001)
}

func TestCPUUsageCountsSystemWorkButNotIOWait(t *testing.T) {
	busy, wait, err := cpuUsageFromDelta(
		cpu.TimesStat{User: 50, System: 20, Iowait: 10, Idle: 20},
		cpu.TimesStat{},
	)
	require.NoError(t, err)
	require.InDelta(t, 70, busy, 0.001)
	require.InDelta(t, 10, wait, 0.001)
}
