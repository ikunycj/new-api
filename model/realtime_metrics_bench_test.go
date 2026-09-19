package model

import (
	"testing"
	"time"
)

// BenchmarkRecordRealtimeUsage measures the cost the relay path pays per
// billable request. This runs on every consume, so it has to stay in the
// tens-of-nanoseconds range; a regression here would be felt under load even
// though nothing is persisted.
func BenchmarkRecordRealtimeUsage(b *testing.B) {
	original := realtimeNow
	realtimeNow = func() int64 { return time.Now().Unix() }
	b.Cleanup(func() { realtimeNow = original })

	// Without this the recording path returns at its first branch and the
	// benchmark times an empty function call instead of the ring.
	originalEnabled := realtimeEnabled
	realtimeEnabled = true
	b.Cleanup(func() { realtimeEnabled = originalEnabled })

	realtimeRegistry.mu.Lock()
	originalUsers := realtimeRegistry.users
	realtimeRegistry.users = make(map[int]*realtimeRing)
	realtimeRegistry.mu.Unlock()
	b.Cleanup(func() {
		realtimeRegistry.mu.Lock()
		realtimeRegistry.users = originalUsers
		realtimeRegistry.mu.Unlock()
	})

	// Warm the ring so the benchmark measures the steady-state path (lookup +
	// slot add), not the one-off allocation of the first request.
	RecordRealtimeUsage(1, 1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			RecordRealtimeUsage(1, 1000)
		}
	})
}

// BenchmarkRecordRealtimeUsageManyUsers measures the same path with a distinct
// user per iteration, which forces a registry write lock and a ring allocation
// on every call — the worst case rather than the common one.
func BenchmarkRecordRealtimeUsageManyUsers(b *testing.B) {
	original := realtimeNow
	realtimeNow = func() int64 { return time.Now().Unix() }
	b.Cleanup(func() { realtimeNow = original })

	// Without this the recording path returns at its first branch and the
	// benchmark times an empty function call instead of the ring.
	originalEnabled := realtimeEnabled
	realtimeEnabled = true
	b.Cleanup(func() { realtimeEnabled = originalEnabled })

	realtimeRegistry.mu.Lock()
	originalUsers := realtimeRegistry.users
	realtimeRegistry.users = make(map[int]*realtimeRing)
	realtimeRegistry.mu.Unlock()
	b.Cleanup(func() {
		realtimeRegistry.mu.Lock()
		realtimeRegistry.users = originalUsers
		realtimeRegistry.mu.Unlock()
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RecordRealtimeUsage(i+1, 1000)
	}
}

// BenchmarkGetRealtimeSnapshot measures the read path a dashboard hits on every
// poll, which walks the whole ring to build the series.
func BenchmarkGetRealtimeSnapshot(b *testing.B) {
	original := realtimeNow
	realtimeNow = func() int64 { return time.Now().Unix() }
	b.Cleanup(func() { realtimeNow = original })

	// Without this the recording path returns at its first branch and the
	// benchmark times an empty function call instead of the ring.
	originalEnabled := realtimeEnabled
	realtimeEnabled = true
	b.Cleanup(func() { realtimeEnabled = originalEnabled })

	realtimeRegistry.mu.Lock()
	originalUsers := realtimeRegistry.users
	realtimeRegistry.users = make(map[int]*realtimeRing)
	realtimeRegistry.mu.Unlock()
	b.Cleanup(func() {
		realtimeRegistry.mu.Lock()
		realtimeRegistry.users = originalUsers
		realtimeRegistry.mu.Unlock()
	})

	for i := 0; i < 100; i++ {
		RecordRealtimeUsage(1, 1000)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetRealtimeSnapshot(1)
	}
}

// BenchmarkRecordRealtimeCacheUsage is the entry point the relay path actually
// calls. It exists next to BenchmarkRecordRealtimeUsage so the cost of carrying
// the two cache counters can be read off directly as the difference between the
// two, rather than being asserted.
func BenchmarkRecordRealtimeCacheUsage(b *testing.B) {
	original := realtimeNow
	realtimeNow = func() int64 { return time.Now().Unix() }
	b.Cleanup(func() { realtimeNow = original })

	originalEnabled := realtimeEnabled
	realtimeEnabled = true
	b.Cleanup(func() { realtimeEnabled = originalEnabled })

	realtimeRegistry.mu.Lock()
	originalUsers := realtimeRegistry.users
	realtimeRegistry.users = make(map[int]*realtimeRing)
	realtimeRegistry.mu.Unlock()
	b.Cleanup(func() {
		realtimeRegistry.mu.Lock()
		realtimeRegistry.users = originalUsers
		realtimeRegistry.mu.Unlock()
	})

	RecordRealtimeCacheUsage(1, 1000, 800, 900, true)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			RecordRealtimeCacheUsage(1, 1000, 800, 900, true)
		}
	})
}
