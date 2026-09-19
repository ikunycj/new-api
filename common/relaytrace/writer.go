package relaytrace

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	// writeBufferSize batches roughly 200 records into a single write syscall.
	// Without buffering each record would trap into the kernel individually,
	// which dominates cost at high request rates.
	writeBufferSize = 4 << 20 // 4MB

	// flushInterval bounds how long a record can sit in the buffer unseen.
	// Needed because a low-traffic gateway may take minutes to fill 4MB.
	flushInterval = time.Second

	activeFileName = "trace.jsonl"
	rotateSuffix   = ".jsonl"
	filePerm       = 0o600 // owner-only: traces contain user prompts verbatim
	dirPerm        = 0o700
)

// fileWriter owns the active trace file. It is driven exclusively by the single
// writer goroutine in tracer, so it needs no internal locking.
type fileWriter struct {
	dir        string
	maxBytes   int64
	maxBackups int

	// minFree is the free-space floor; 0 disables the guard. checkEvery bounds
	// how often the filesystem is sampled so statfs stays off the hot path.
	minFree    uint64
	checkEvery time.Duration

	// lastCheck/paused cache the most recent disk verdict between samples.
	lastCheck time.Time
	paused    bool

	file *os.File
	buf  *bufio.Writer
	size int64

	// bg tracks background compression/pruning so close() can wait for it.
	// Without this the goroutine outlives shutdown and writes files after the
	// process believes it is done, which also trips the race detector.
	bg sync.WaitGroup
}

// errDiskLow signals that writing is paused to protect the filesystem. Returned
// instead of writing so the caller counts it as a drop rather than a failure.
var errDiskLow = errors.New("relaytrace: paused, free disk below threshold")

func newFileWriter(cfg Config) (*fileWriter, error) {
	if err := os.MkdirAll(cfg.Dir, dirPerm); err != nil {
		return nil, fmt.Errorf("create trace dir %q: %w", cfg.Dir, err)
	}
	w := &fileWriter{
		dir:        cfg.Dir,
		maxBytes:   cfg.MaxFileBytes(),
		maxBackups: cfg.MaxBackups,
		minFree:    cfg.MinFreeDiskBytes(),
		checkEvery: time.Duration(cfg.DiskCheckInterval) * time.Second,
	}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

// diskLow reports whether writing should pause, sampling the filesystem at most
// once per checkEvery.
//
// A statfs failure is treated as "not low" on purpose: an unreadable mount must
// not silently disable auditing. The disk-full case surfaces as a write error
// instead, which is already handled.
func (w *fileWriter) diskLow(now time.Time) bool {
	if w.minFree == 0 {
		return false
	}
	if !w.lastCheck.IsZero() && now.Sub(w.lastCheck) < w.checkEvery {
		return w.paused
	}
	w.lastCheck = now

	free, err := freeDiskBytes(w.dir)
	if err != nil {
		w.paused = false
		return false
	}
	low := free < w.minFree
	// Log only on transition so a sustained low-disk condition cannot itself
	// flood the very filesystem the guard is protecting.
	if low != w.paused {
		if low {
			common.SysError(fmt.Sprintf(
				"relay trace paused: free disk %dMB below threshold %dMB at %s",
				free>>20, w.minFree>>20, w.dir))
		} else {
			common.SysLog(fmt.Sprintf(
				"relay trace resumed: free disk %dMB recovered above %dMB",
				free>>20, w.minFree>>20))
		}
	}
	w.paused = low
	return low
}

func (w *fileWriter) activePath() string {
	return filepath.Join(w.dir, activeFileName)
}

func (w *fileWriter) open() error {
	path := w.activePath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, filePerm)
	if err != nil {
		return fmt.Errorf("open trace file %q: %w", path, err)
	}
	// Resume size accounting across restarts so rotation is not deferred
	// indefinitely by frequent process restarts.
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("stat trace file %q: %w", path, err)
	}
	w.file = f
	w.size = info.Size()
	w.buf = bufio.NewWriterSize(f, writeBufferSize)
	return nil
}

// writeLine appends one JSONL line, rotating first if the file would exceed the
// size limit. The newline is added here so callers cannot forget it and corrupt
// the line-oriented format.
func (w *fileWriter) writeLine(line []byte) error {
	// Checked before rotation: rotation itself writes a gzip file, so proceeding
	// while the disk is nearly full would consume the very headroom being
	// defended.
	if w.diskLow(time.Now()) {
		return errDiskLow
	}
	if w.maxBytes > 0 && w.size+int64(len(line))+1 > w.maxBytes {
		if err := w.rotate(); err != nil {
			// Rotation failure must not stop trace collection; keep appending to
			// the current file and let the next attempt retry.
			return err
		}
	}
	n, err := w.buf.Write(line)
	w.size += int64(n)
	if err != nil {
		return err
	}
	if err := w.buf.WriteByte('\n'); err != nil {
		return err
	}
	w.size++
	return nil
}

// flush pushes buffered bytes into the OS page cache.
//
// Deliberately does NOT fsync: a single fsync costs 1-10ms and would erase the
// microsecond-scale advantage of local append. The trade-off is losing at most
// the last flushInterval of records on power loss, which is acceptable for audit
// traces.
func (w *fileWriter) flush() error {
	if w.buf == nil {
		return nil
	}
	return w.buf.Flush()
}

// rotate closes the active file, renames it with a timestamp, restarts a fresh
// active file, then compresses and prunes in the background.
func (w *fileWriter) rotate() error {
	if err := w.flush(); err != nil {
		return err
	}
	if err := w.file.Close(); err != nil {
		return err
	}
	rotated := filepath.Join(w.dir, fmt.Sprintf("trace-%s%s", time.Now().Format("20060102-150405.000"), rotateSuffix))
	if err := os.Rename(w.activePath(), rotated); err != nil {
		// Reopen so collection continues even if the rename failed.
		_ = w.open()
		return err
	}
	if err := w.open(); err != nil {
		return err
	}
	// Compression and pruning are best-effort and must never block writes.
	// Captured by value so the goroutine never reads mutable fileWriter state.
	dir, maxBackups := w.dir, w.maxBackups
	w.bg.Add(1)
	go func(path string) {
		defer w.bg.Done()
		if err := gzipFile(path); err != nil {
			return
		}
		prune(dir, maxBackups)
	}(rotated)
	return nil
}

func (w *fileWriter) close() error {
	if w.file == nil {
		return nil
	}
	if err := w.flush(); err != nil {
		_ = w.file.Close()
		w.file = nil
		w.bg.Wait()
		return err
	}
	err := w.file.Close()
	w.file = nil
	// Wait for in-flight gzip/prune so shutdown leaves no files half-written.
	w.bg.Wait()
	return err
}

// gzipFile compresses path to path+".gz" and removes the original on success.
func gzipFile(path string) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(path+".gz", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		return err
	}
	zw := gzip.NewWriter(out)
	if _, err := io.Copy(zw, in); err != nil {
		_ = zw.Close()
		_ = out.Close()
		_ = os.Remove(path + ".gz")
		return err
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}

// prune deletes the oldest rotated files beyond maxBackups. Takes its inputs as
// arguments so it can run off the writer goroutine without sharing state.
func prune(dir string, maxBackups int) {
	if maxBackups <= 0 {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var rotated []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == activeFileName || !strings.HasPrefix(name, "trace-") {
			continue
		}
		rotated = append(rotated, name)
	}
	if len(rotated) <= maxBackups {
		return
	}
	// Names embed a sortable timestamp, so lexical order is chronological.
	sort.Strings(rotated)
	for _, name := range rotated[:len(rotated)-maxBackups] {
		_ = os.Remove(filepath.Join(dir, name))
	}
}
