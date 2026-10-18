package traj_record

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/klauspost/compress/zstd"
)

const (
	writeBufferSize   = 4 << 20
	flushInterval     = time.Second
	diskCheckInterval = 10 * time.Second

	activeFileName = "traj.jsonl"
	rotatePrefix   = "traj-"
	filePerm       = 0o640 // owner read/write, group read: records contain user prompts verbatim
	dirPerm        = 0o750
)

var errDiskLow = errors.New("traj_record: paused, free disk below threshold")

// fileWriter owns the active JSONL file. Driven only by the writer goroutine.
//
// Unlike a log writer it does not rotate on its own: the recorder decides when
// to rotate, because rotation must also reset the prefix index so every file
// can be reconstructed without its predecessors.
type fileWriter struct {
	dir        string
	maxBytes   int64
	maxBackups int
	zstdWindow int
	minFree    uint64

	lastCheck time.Time
	paused    bool

	file *os.File
	buf  *bufio.Writer
	size int64

	// bg tracks background compression so close() can wait for it.
	bg sync.WaitGroup
}

func newFileWriter(cfg Config) (*fileWriter, error) {
	if err := os.MkdirAll(cfg.Dir, dirPerm); err != nil {
		return nil, fmt.Errorf("create traj dir %q: %w", cfg.Dir, err)
	}
	w := &fileWriter{
		dir:        cfg.Dir,
		maxBytes:   int64(cfg.MaxFileMB) << 20,
		maxBackups: cfg.MaxBackups,
		zstdWindow: cfg.ZstdWindowMB << 20,
		minFree:    uint64(cfg.MinFreeDiskMB) << 20,
	}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *fileWriter) activePath() string {
	return filepath.Join(w.dir, activeFileName)
}

func (w *fileWriter) open() error {
	f, err := os.OpenFile(w.activePath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, filePerm)
	if err != nil {
		return fmt.Errorf("open traj file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("stat traj file: %w", err)
	}
	w.file = f
	w.size = info.Size()
	w.buf = bufio.NewWriterSize(f, writeBufferSize)
	return nil
}

// diskLow reports whether writing should pause, sampling the filesystem at most
// once per diskCheckInterval. A statfs failure counts as "not low" so an
// unreadable mount cannot silently disable capture.
func (w *fileWriter) diskLow(now time.Time) bool {
	if w.minFree == 0 {
		return false
	}
	if !w.lastCheck.IsZero() && now.Sub(w.lastCheck) < diskCheckInterval {
		return w.paused
	}
	w.lastCheck = now
	free, err := freeDiskBytes(w.dir)
	low := err == nil && free < w.minFree
	if low != w.paused {
		if low {
			common.SysError(fmt.Sprintf("traj record paused: free disk %dMB below %dMB at %s", free>>20, w.minFree>>20, w.dir))
		} else {
			common.SysLog("traj record resumed: free disk recovered")
		}
	}
	w.paused = low
	return low
}

func (w *fileWriter) full() bool {
	return w.size >= w.maxBytes
}

// writeLines appends lines that belong together (a call and the blobs it
// references) and adds the newlines itself so callers cannot corrupt the
// line-oriented format.
func (w *fileWriter) writeLines(lines ...[]byte) error {
	if w.diskLow(time.Now()) {
		return errDiskLow
	}
	for _, line := range lines {
		n, err := w.buf.Write(line)
		w.size += int64(n)
		if err != nil {
			return err
		}
		if err := w.buf.WriteByte('\n'); err != nil {
			return err
		}
		w.size++
	}
	return nil
}

func (w *fileWriter) flush() error {
	if w.buf == nil {
		return nil
	}
	return w.buf.Flush()
}

// rotate renames the active file with a timestamp, opens a fresh one, then
// compresses and prunes in the background.
func (w *fileWriter) rotate() error {
	if err := w.flush(); err != nil {
		return err
	}
	if err := w.file.Close(); err != nil {
		return err
	}
	rotated := filepath.Join(w.dir, rotatePrefix+time.Now().Format("20060102-150405.000")+".jsonl")
	if err := os.Rename(w.activePath(), rotated); err != nil {
		_ = w.open()
		return err
	}
	if err := w.open(); err != nil {
		return err
	}
	dir, maxBackups, window := w.dir, w.maxBackups, w.zstdWindow
	w.bg.Go(func() {
		if window > 0 {
			if err := zstdFile(rotated, window); err != nil {
				common.SysError("traj record compress failed: " + err.Error())
				return
			}
		}
		prune(dir, maxBackups)
	})
	return nil
}

func (w *fileWriter) close() error {
	if w.file == nil {
		return nil
	}
	err := w.flush()
	if cerr := w.file.Close(); err == nil {
		err = cerr
	}
	w.file = nil
	w.bg.Wait()
	return err
}

// zstdFile compresses path to path+".zst" and removes the original on success.
// The window is rounded up to a power of two as zstd requires.
func zstdFile(path string, window int) error {
	window = min(max(1<<bits.Len(uint(window-1)), zstd.MinWindowSize), zstd.MaxWindowSize)
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(path+".zst", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		return err
	}
	zw, err := zstd.NewWriter(out,
		zstd.WithWindowSize(window),
		zstd.WithEncoderLevel(zstd.SpeedBetterCompression),
		zstd.WithEncoderConcurrency(1))
	if err == nil {
		_, err = io.Copy(zw, in)
		if cerr := zw.Close(); err == nil {
			err = cerr
		}
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(path + ".zst")
		return err
	}
	return os.Remove(path)
}

// prune deletes the oldest rotated files beyond maxBackups; 0 keeps all.
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
		if !e.IsDir() && strings.HasPrefix(e.Name(), rotatePrefix) {
			rotated = append(rotated, e.Name())
		}
	}
	if len(rotated) <= maxBackups {
		return
	}
	// Names embed a sortable timestamp, so lexical order is chronological.
	slices.Sort(rotated)
	for _, name := range rotated[:len(rotated)-maxBackups] {
		_ = os.Remove(filepath.Join(dir, name))
	}
}
