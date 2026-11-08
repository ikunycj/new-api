package traj_record

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"math/bits"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
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

	dayLayout = "20060102" // UTC
)

var (
	errDiskLow = errors.New("traj_record: paused, free disk below threshold")

	// dayFileRe matches the files of a 录制日: traj-<YYYYMMDD>-<NNN>.jsonl,
	// compressed or not. Files rotated by v3 (traj-<local time>.jsonl) do
	// not match and are left alone.
	dayFileRe = regexp.MustCompile(`^traj-(\d{8})-(\d{3})\.jsonl(\.zst)?$`)
)

func utcDay(t time.Time) string {
	return t.UTC().Format(dayLayout)
}

func manifestName(day string) string {
	return rotatePrefix + day + ".manifest.json"
}

// fileWriter owns the active JSONL file. Driven only by the writer goroutine.
//
// Files are sealed per UTC day (a 录制日): the recorder rotates at UTC
// midnight and whenever the active file is full. All files of one day form
// one stream that is decoded in order (the prefix index and blobs carry over
// between them); a new day starts self-contained. When a day is over, a
// manifest lists its files, and an uploader only ever looks at manifests.
type fileWriter struct {
	dir        string
	instance   string
	maxBytes   int64
	maxBackups int
	zstdWindow int
	minFree    uint64
	now        func() time.Time

	lastCheck time.Time
	paused    bool

	file *os.File
	buf  *bufio.Writer
	size int64
	day  string // UTC day of the active file
	// legacy is set when the active file was written by format v3, which
	// has no days; it is rotated away at startup under the v3 name.
	legacy bool
	// recovered is the length of a torn last line cut off at startup.
	recovered int64

	// tasks runs compression and manifests in order, off the write path.
	tasks  chan func()
	bgDone chan struct{}
}

func newFileWriter(cfg Config) (*fileWriter, error) {
	return newFileWriterAt(cfg, time.Now)
}

func newFileWriterAt(cfg Config, now func() time.Time) (*fileWriter, error) {
	if err := os.MkdirAll(cfg.Dir, dirPerm); err != nil {
		return nil, fmt.Errorf("create traj dir %q: %w", cfg.Dir, err)
	}
	w := &fileWriter{
		dir:        cfg.Dir,
		instance:   cfg.Instance,
		maxBytes:   int64(cfg.MaxFileMB) << 20,
		maxBackups: cfg.MaxBackups,
		zstdWindow: cfg.ZstdWindowMB << 20,
		minFree:    uint64(cfg.MinFreeDiskMB) << 20,
		now:        now,
		tasks:      make(chan func(), 64),
		bgDone:     make(chan struct{}),
	}
	go func() {
		defer close(w.bgDone)
		for task := range w.tasks {
			task()
		}
	}()
	if err := w.recover(); err != nil {
		w.stopTasks()
		return nil, err
	}
	return w, nil
}

// recover cleans up after a crash and opens the active file:
//   - removes half-written compressed files (*.tmp) and recompresses the
//     rotated files they belonged to;
//   - cuts a torn last line off the active file;
//   - seals the active file if it belongs to an earlier day;
//   - writes the missing manifests of earlier days.
func (w *fileWriter) recover() error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), rotatePrefix) && strings.HasSuffix(e.Name(), ".tmp") {
			_ = os.Remove(filepath.Join(w.dir, e.Name()))
		}
	}
	today := utcDay(w.now())
	for _, e := range entries {
		// Past days are handled by sealPendingDays, which only touches the
		// days it is about to seal; recompressing a day that already has a
		// manifest would make the manifest point at a file that is gone.
		if m := dayFileRe.FindStringSubmatch(e.Name()); m != nil && m[3] == "" && m[1] == today {
			w.compressLater(filepath.Join(w.dir, e.Name()))
		}
	}
	if err := w.cutTornLine(); err != nil {
		return err
	}
	if err := w.open(); err != nil {
		return err
	}
	if w.size > 0 && w.legacy {
		if err := w.rotateLegacy(); err != nil {
			return err
		}
	} else if w.size > 0 && w.day != utcDay(w.now()) {
		if err := w.rotate(); err != nil {
			return err
		}
	}
	w.sealPendingDays()
	return nil
}

// sealPendingDays compresses and seals every 录制日 before today that has no
// manifest yet. Startup and every rotation call it, so a day whose compression
// failed is retried on the next rotation instead of waiting for a restart (a
// day with no manifest is never uploaded).
// sealRetryInterval is how often the writer retries sealing between rotations.
const sealRetryInterval = 10 * time.Minute

func (w *fileWriter) sealPendingDays() {
	today := utcDay(w.now())
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		common.SysError("traj record read dir failed: " + err.Error())
		return
	}
	pending := map[string][]string{}
	for _, e := range entries {
		m := dayFileRe.FindStringSubmatch(e.Name())
		if m == nil || m[1] >= today {
			continue
		}
		if _, err := os.Stat(filepath.Join(w.dir, manifestName(m[1]))); err == nil {
			continue
		}
		if _, ok := pending[m[1]]; !ok {
			pending[m[1]] = nil
		}
		if m[3] == "" {
			pending[m[1]] = append(pending[m[1]], e.Name())
		}
	}
	for _, day := range slices.Sorted(maps.Keys(pending)) {
		for _, name := range pending[day] {
			w.compressLater(filepath.Join(w.dir, name))
		}
		w.seal(day)
	}
}

// cutTornLine truncates the active file after its last newline: a crash can
// leave half a record, which no reader could parse.
func (w *fileWriter) cutTornLine() error {
	f, err := os.OpenFile(w.activePath(), os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		return err
	}
	const chunk = 1 << 20
	end := info.Size()
	for pos := end; pos > 0; {
		n := min(int64(chunk), pos)
		pos -= n
		buf := make([]byte, n)
		if _, err := f.ReadAt(buf, pos); err != nil {
			return err
		}
		if i := bytes.LastIndexByte(buf, '\n'); i >= 0 {
			keep := pos + int64(i) + 1
			if keep == end {
				return nil
			}
			w.recovered = end - keep
			return f.Truncate(keep)
		}
	}
	w.recovered = end
	return f.Truncate(0)
}

func (w *fileWriter) activePath() string {
	return filepath.Join(w.dir, activeFileName)
}

// open opens the active file for appending. Its day is the day of its first
// line's "ts" (every file starts with segment_start or file_start); an empty
// or unreadable file belongs to today.
func (w *fileWriter) open() error {
	f, err := os.OpenFile(w.activePath(), os.O_CREATE|os.O_RDWR|os.O_APPEND, filePerm)
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
	w.day = utcDay(w.now())
	if w.size > 0 {
		var first struct {
			V  int   `json:"v"`
			Ts int64 `json:"ts"`
		}
		line, _ := bufio.NewReader(io.NewSectionReader(f, 0, w.size)).ReadBytes('\n')
		if common.Unmarshal(line, &first) == nil && first.Ts > 0 {
			w.day = utcDay(time.UnixMilli(first.Ts))
		} else {
			w.day = utcDay(info.ModTime())
		}
		w.legacy = first.V > 0 && first.V < 4
	}
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

// rotate closes the active file as the next file of its day, opens a fresh
// one for the current day, and compresses and prunes in the background.
func (w *fileWriter) rotate() error {
	if err := w.flush(); err != nil {
		return err
	}
	if err := w.file.Sync(); err != nil {
		return err
	}
	if err := w.file.Close(); err != nil {
		return err
	}
	rotated := filepath.Join(w.dir, fmt.Sprintf("%s%s-%03d.jsonl", rotatePrefix, w.day, w.nextNumber(w.day)))
	if err := os.Rename(w.activePath(), rotated); err != nil {
		_ = w.open()
		return err
	}
	syncDir(w.dir)
	if err := w.open(); err != nil {
		return err
	}
	w.compressLater(rotated)
	return nil
}

// rotateLegacy moves a v3 active file aside the way v3 rotated it
// (traj-<local time>.jsonl, compressed), outside of any 录制日.
func (w *fileWriter) rotateLegacy() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	rotated := filepath.Join(w.dir, rotatePrefix+w.now().Format("20060102-150405.000")+".jsonl")
	if err := os.Rename(w.activePath(), rotated); err != nil {
		return err
	}
	syncDir(w.dir)
	if err := w.open(); err != nil {
		return err
	}
	w.compressLater(rotated)
	return nil
}

// nextNumber is one past the highest file number of day.
func (w *fileWriter) nextNumber(day string) int {
	entries, _ := os.ReadDir(w.dir)
	next := 0
	for _, e := range entries {
		if m := dayFileRe.FindStringSubmatch(e.Name()); m != nil && m[1] == day {
			n, _ := strconv.Atoi(m[2])
			next = max(next, n+1)
		}
	}
	return next
}

func (w *fileWriter) compressLater(path string) {
	dir, maxBackups, window := w.dir, w.maxBackups, w.zstdWindow
	w.tasks <- func() {
		if window > 0 {
			if err := zstdFile(path, window); err != nil {
				common.SysError("traj record compress failed: " + err.Error())
				return
			}
		}
		prune(dir, maxBackups)
	}
}

// seal writes the manifest of day once its queued compressions are done.
func (w *fileWriter) seal(day string) {
	compress := w.zstdWindow > 0
	w.tasks <- func() {
		if err := writeManifest(w.dir, w.instance, day, w.now(), compress); err != nil {
			common.SysError("traj record manifest failed: " + err.Error())
		}
	}
}

type manifestFile struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Sha256 string `json:"sha256"`
}

type manifest struct {
	V        int            `json:"v"`
	Type     string         `json:"type"`
	Instance string         `json:"instance"`
	Day      string         `json:"day"`
	Ts       int64          `json:"ts"`
	Files    []manifestFile `json:"files"`
}

// writeManifest lists the files of day in order. With compression on it fails
// unless every file is compressed: a failed compression would otherwise be
// sealed under its uncompressed name, and the next startup would recompress it
// and leave the manifest pointing at a file that no longer exists. No manifest
// means the day is not uploaded yet; startup retries the compression and then
// the manifest.
func writeManifest(dir, instance, day string, now time.Time, compress bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	m := manifest{V: recordVersion, Type: "manifest", Instance: instance, Day: day, Ts: now.UnixMilli(), Files: []manifestFile{}}
	numbers := map[string]bool{}
	for _, e := range entries {
		match := dayFileRe.FindStringSubmatch(e.Name())
		if match == nil || match[1] != day {
			continue
		}
		if numbers[match[2]] {
			return fmt.Errorf("day %s file %s exists both compressed and not", day, match[2])
		}
		if compress && match[3] == "" {
			return fmt.Errorf("day %s file %s is not compressed yet", day, match[2])
		}
		numbers[match[2]] = true
		path := filepath.Join(dir, e.Name())
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		h := sha256.New()
		size, err := io.Copy(h, f)
		_ = f.Close()
		if err != nil {
			return err
		}
		m.Files = append(m.Files, manifestFile{Name: e.Name(), Size: size, Sha256: hex.EncodeToString(h.Sum(nil))})
	}
	// ReadDir sorts by name, and names sort by day and number.
	data, err := common.Marshal(m)
	if err != nil {
		return err
	}
	return writeFileSync(filepath.Join(dir, manifestName(day)), append(data, '\n'))
}

// writeFileSync writes data via a temporary file, so the target either does
// not exist or is complete.
func writeFileSync(path string, data []byte) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	syncDir(filepath.Dir(path))
	return nil
}

func syncDir(dir string) {
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
}

func (w *fileWriter) close() error {
	if w.file == nil {
		return nil
	}
	err := w.flush()
	if serr := w.file.Sync(); err == nil {
		err = serr
	}
	if cerr := w.file.Close(); err == nil {
		err = cerr
	}
	w.file = nil
	w.stopTasks()
	return err
}

func (w *fileWriter) stopTasks() {
	close(w.tasks)
	<-w.bgDone
}

// zstdFile compresses path to path+".zst" and removes the original on success.
// It writes path+".zst.tmp" first, so a crash never leaves a partial .zst.
// The window is rounded up to a power of two as zstd requires.
func zstdFile(path string, window int) error {
	window = min(max(1<<bits.Len(uint(window-1)), zstd.MinWindowSize), zstd.MaxWindowSize)
	in, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// Already compressed, by a task queued for the same file (sealPendingDays
		// may queue one that rotate already did). Not an error.
		if _, serr := os.Stat(path + ".zst"); serr == nil {
			return nil
		}
	}
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := path + ".zst.tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
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
	if err == nil {
		err = out.Sync()
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, path+".zst")
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	syncDir(filepath.Dir(path))
	return os.Remove(path)
}

// prune deletes the oldest rotated files beyond maxBackups; 0 keeps all.
// Manifests are not counted or deleted.
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
		if !e.IsDir() && strings.HasPrefix(name, rotatePrefix) && (strings.HasSuffix(name, ".jsonl") || strings.HasSuffix(name, ".jsonl.zst")) {
			rotated = append(rotated, name)
		}
	}
	if len(rotated) <= maxBackups {
		return
	}
	// Names embed a sortable timestamp or day, so lexical order is chronological.
	slices.Sort(rotated)
	for _, name := range rotated[:len(rotated)-maxBackups] {
		_ = os.Remove(filepath.Join(dir, name))
	}
}
