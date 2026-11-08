package traj_record

import (
	"github.com/QuantumNous/new-api/common"

	"sync/atomic"
	"time"
)

// statsInterval is how often a stats line is written; one is also written
// before every rotation and on close.
const statsInterval = time.Minute

// familyStats counts HTTP calls of one API family at each point where one can
// leave the pipeline. Over a sealed day:
//
//	seen = filtered + body_error + sampled_out + dropped_queue + enqueued
//	enqueued = written + disk_paused + write_failed
//
// enqueued counts the calls the writer took off the queue. Calls still in
// flight or queued at the end of a day show up as enqueued/written on the
// next day.
type familyStats struct {
	seen, filtered, bodyError, sampledOut, enqueued, droppedQueue atomic.Int64
	written, diskPaused, writeFailed                              atomic.Int64
}

type statsCounts struct {
	Seen         int64 `json:"seen"`
	Filtered     int64 `json:"filtered,omitempty"`
	BodyError    int64 `json:"body_error,omitempty"`
	SampledOut   int64 `json:"sampled_out,omitempty"`
	Enqueued     int64 `json:"enqueued"`
	DroppedQueue int64 `json:"dropped_queue,omitempty"`
	Written      int64 `json:"written"`
	DiskPaused   int64 `json:"disk_paused,omitempty"`
	WriteFailed  int64 `json:"write_failed,omitempty"`
}

func (s *familyStats) load() statsCounts {
	return statsCounts{
		Seen: s.seen.Load(), Filtered: s.filtered.Load(), BodyError: s.bodyError.Load(),
		SampledOut: s.sampledOut.Load(), Enqueued: s.enqueued.Load(), DroppedQueue: s.droppedQueue.Load(),
		Written: s.written.Load(), DiskPaused: s.diskPaused.Load(), WriteFailed: s.writeFailed.Load(),
	}
}

func (a statsCounts) sub(b statsCounts) statsCounts {
	return statsCounts{
		Seen: a.Seen - b.Seen, Filtered: a.Filtered - b.Filtered, BodyError: a.BodyError - b.BodyError,
		SampledOut: a.SampledOut - b.SampledOut, Enqueued: a.Enqueued - b.Enqueued,
		DroppedQueue: a.DroppedQueue - b.DroppedQueue, Written: a.Written - b.Written,
		DiskPaused: a.DiskPaused - b.DiskPaused, WriteFailed: a.WriteFailed - b.WriteFailed,
	}
}

// statsRecord is the "stats" line: the counts since the previous stats line
// (Since) of this process, per API family with any activity.
type statsRecord struct {
	V        int                    `json:"v"`
	Type     string                 `json:"type"`
	Instance string                 `json:"instance"`
	Since    int64                  `json:"since"`
	Ts       int64                  `json:"ts"`
	Families map[string]statsCounts `json:"families"`
}

func newFamilyStats() map[string]*familyStats {
	m := map[string]*familyStats{}
	for _, cl := range clients {
		m[cl.apiFamily] = &familyStats{}
	}
	return m
}

// writeStats writes the counts since the last stats line. Writer goroutine only.
func (r *recorder) writeStats(now time.Time) {
	rec := statsRecord{V: recordVersion, Type: "stats", Instance: r.cfg.Instance,
		Since: r.statsSince.UnixMilli(), Ts: now.UnixMilli(), Families: map[string]statsCounts{}}
	cur := map[string]statsCounts{}
	for family, s := range r.stats {
		cur[family] = s.load()
		if d := cur[family].sub(r.statsLast[family]); d != (statsCounts{}) {
			rec.Families[family] = d
		}
	}
	line, err := common.Marshal(rec)
	if err == nil {
		err = r.fw.writeLines(line)
	}
	if err != nil {
		// Keep the baseline so the next stats line still covers this period.
		r.warnf("traj record write stats failed: %v", err)
		return
	}
	r.statsLast, r.statsSince = cur, now
}
