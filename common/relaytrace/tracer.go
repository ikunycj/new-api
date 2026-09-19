package relaytrace

import (
	"encoding/json"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// tracer owns the submit queue and the single writer goroutine.
//
// Exactly one goroutine writes, so the file needs no lock: the channel itself is
// the queue and serialization point. Adding more writers would require locking
// and could interleave partial lines.
type tracer struct {
	cfg  Config
	ch   chan *Record
	done chan struct{}
	wg   sync.WaitGroup

	submitted atomic.Int64
	written   atomic.Int64
	dropped   atomic.Int64
	failed    atomic.Int64

	// diskPaused counts records discarded by the free-space guard, kept apart
	// from failed so "the disk is full" is distinguishable from "writing broke".
	diskPaused atomic.Int64
}

var (
	mu     sync.RWMutex
	active *tracer
)

// Init starts trace collection. It is safe to call when disabled: the tracer
// stays nil and Submit becomes a no-op.
//
// An error here must not prevent server startup; the caller logs and continues.
func Init(cfg Config) error {
	cfg.normalize()

	mu.Lock()
	defer mu.Unlock()

	if active != nil {
		active.stop()
		active = nil
	}
	if !cfg.Enabled {
		return nil
	}

	fw, err := newFileWriter(cfg)
	if err != nil {
		return err
	}

	t := &tracer{
		cfg:  cfg,
		ch:   make(chan *Record, cfg.QueueSize),
		done: make(chan struct{}),
	}
	t.wg.Add(1)
	go t.run(fw)
	active = t

	common.SysLog("relay trace enabled, dir: " + cfg.Dir)
	return nil
}

// Close drains the queue and flushes buffered data. Called during graceful
// shutdown so in-flight records are not lost.
func Close() {
	mu.Lock()
	t := active
	active = nil
	mu.Unlock()

	if t != nil {
		t.stop()
	}
}

// Enabled reports whether collection is on. The middleware checks this first to
// keep the disabled path free of any allocation.
func Enabled() bool {
	mu.RLock()
	t := active
	mu.RUnlock()
	return t != nil
}

// Stats returns cumulative counters for observability.
func Stats() (submitted, written, dropped, failed int64) {
	mu.RLock()
	t := active
	mu.RUnlock()
	if t == nil {
		return 0, 0, 0, 0
	}
	return t.submitted.Load(), t.written.Load(), t.dropped.Load(), t.failed.Load()
}

// DiskPaused returns how many records the free-space guard discarded. A nonzero
// and growing value means the trace filesystem needs attention.
func DiskPaused() int64 {
	mu.RLock()
	t := active
	mu.RUnlock()
	if t == nil {
		return 0
	}
	return t.diskPaused.Load()
}

// Config returns the active configuration, or a disabled zero value.
func ActiveConfig() Config {
	mu.RLock()
	t := active
	mu.RUnlock()
	if t == nil {
		return Config{}
	}
	return t.cfg
}

// Submit hands a record to the writer without blocking.
//
// The default branch is the safety valve for the whole feature: if the disk is
// full or the writer stalls, records are discarded and the relay request
// proceeds untouched. An audit log must never take down the request path.
func Submit(r *Record) {
	if r == nil {
		return
	}
	mu.RLock()
	t := active
	mu.RUnlock()
	if t == nil {
		return
	}
	select {
	case t.ch <- r:
		t.submitted.Add(1)
	default:
		t.dropped.Add(1)
	}
}

// ShouldRecord decides whether a finished request is worth persisting.
//
// Evaluated after the handler runs, because the status code is needed to apply
// ErrorAlways. Allow-lists win over sampling so a targeted investigation does
// not require raising the global rate.
func ShouldRecord(status, userId int, model string) bool {
	mu.RLock()
	t := active
	mu.RUnlock()
	if t == nil {
		return false
	}
	cfg := t.cfg

	if cfg.ErrorAlways && (status < 200 || status >= 300) {
		return true
	}
	if len(cfg.Users) > 0 && cfg.Users[userId] {
		return true
	}
	if len(cfg.Models) > 0 && cfg.Models[model] {
		return true
	}
	if cfg.SampleRate <= 0 {
		return false
	}
	if cfg.SampleRate >= 100 {
		return true
	}
	return rand.Intn(100) < cfg.SampleRate
}

// run is the single writer loop.
func (t *tracer) run(fw *fileWriter) {
	defer t.wg.Done()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	defer func() {
		// Drain whatever is still queued before closing, so a shutdown does not
		// discard records that were already accepted.
		for {
			select {
			case r := <-t.ch:
				t.persist(fw, r)
			default:
				_ = fw.close()
				return
			}
		}
	}()

	for {
		select {
		case r := <-t.ch:
			t.persist(fw, r)
		case <-ticker.C:
			if err := fw.flush(); err != nil {
				t.failed.Add(1)
			}
		case <-t.done:
			return
		}
	}
}

func (t *tracer) persist(fw *fileWriter, r *Record) {
	line, err := json.Marshal(r)
	if err != nil {
		t.failed.Add(1)
		return
	}
	if err := fw.writeLine(line); err != nil {
		// A disk-space pause is a deliberate drop, not a malfunction. Counting
		// it separately keeps `failed` meaningful as "something is broken".
		if errors.Is(err, errDiskLow) {
			t.diskPaused.Add(1)
			return
		}
		t.failed.Add(1)
		return
	}
	t.written.Add(1)
}

func (t *tracer) stop() {
	close(t.done)
	t.wg.Wait()
}
