package common

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// DiskSpaceInfo 磁盘空间信息
type DiskSpaceInfo struct {
	// 总空间（字节）
	Total uint64 `json:"total"`
	// 可用空间（字节）
	Free uint64 `json:"free"`
	// 已用空间（字节）
	Used uint64 `json:"used"`
	// 使用百分比
	UsedPercent float64 `json:"used_percent"`
}

// SystemStatus 系统状态信息
type SystemStatus struct {
	CPUUsage       float64
	CPUWaitPercent float64
	MemoryUsage    float64
	DiskUsage      float64
}

var latestSystemStatus atomic.Value

var cpuSampleState struct {
	sync.Mutex
	initialized bool
	previous    cpu.TimesStat
}

func init() {
	latestSystemStatus.Store(SystemStatus{})
}

// StartSystemMonitor 启动系统监控
func StartSystemMonitor() {
	go func() {
		for {
			config := GetPerformanceMonitorConfig()
			if !config.Enabled {
				time.Sleep(30 * time.Second)
				continue
			}

			updateSystemStatus()
			time.Sleep(5 * time.Second)
		}
	}()
}

func updateSystemStatus() {
	var status SystemStatus

	// CPU usage excludes iowait so disk stalls do not look like CPU saturation.
	if usage, wait, err := sampleCPU(); err == nil {
		status.CPUUsage = usage
		status.CPUWaitPercent = wait
	}

	// Memory
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		status.MemoryUsage = memInfo.UsedPercent
	}

	// Disk
	diskInfo := GetDiskSpaceInfo()
	if diskInfo.Total > 0 {
		status.DiskUsage = diskInfo.UsedPercent
	}

	latestSystemStatus.Store(status)
}

func sampleCPU() (busyPercent, waitPercent float64, err error) {
	times, err := cpu.Times(false)
	if err != nil || len(times) == 0 {
		return 0, 0, err
	}

	cpuSampleState.Lock()
	defer cpuSampleState.Unlock()
	if !cpuSampleState.initialized {
		cpuSampleState.previous = times[0]
		cpuSampleState.initialized = true
		return 0, 0, nil
	}

	previous := cpuSampleState.previous
	cpuSampleState.previous = times[0]
	return cpuUsageFromDelta(times[0], previous)
}

func cpuUsageFromDelta(current, previous cpu.TimesStat) (busyPercent, waitPercent float64, err error) {
	delta := func(current, old float64) float64 {
		if current <= old {
			return 0
		}
		return current - old
	}
	user := delta(current.User, previous.User)
	system := delta(current.System, previous.System)
	nice := delta(current.Nice, previous.Nice)
	irq := delta(current.Irq, previous.Irq)
	softIRQ := delta(current.Softirq, previous.Softirq)
	steal := delta(current.Steal, previous.Steal)
	iowait := delta(current.Iowait, previous.Iowait)
	idle := delta(current.Idle, previous.Idle)
	total := user + system + nice + irq + softIRQ + steal + iowait + idle
	if total <= 0 {
		return 0, 0, nil
	}
	return (user + system + nice + irq + softIRQ + steal) / total * 100,
		iowait / total * 100,
		nil
}

// GetSystemStatus 获取当前系统状态
func GetSystemStatus() SystemStatus {
	return latestSystemStatus.Load().(SystemStatus)
}

// RefreshSystemStatus forces one synchronous sample for admin dashboards.
// The background monitor continues to own periodic refreshes.
func RefreshSystemStatus() SystemStatus {
	updateSystemStatus()
	return GetSystemStatus()
}
