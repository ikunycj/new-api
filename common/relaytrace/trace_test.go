package relaytrace

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// testConfig returns an enabled config writing into a temp dir.
func testConfig(t *testing.T) Config {
	t.Helper()
	cfg := Config{
		Enabled:    true,
		Dir:        t.TempDir(),
		SampleRate: 100,
		MaxBodyKB:  defaultMaxBodyKB,
		QueueSize:  1024,
		MaxFileMB:  defaultMaxFileMB,
		MaxBackups: 5,
	}
	cfg.normalize()
	return cfg
}

// readLines returns the non-empty lines of the active trace file.
func readLines(t *testing.T, dir string) []string {
	t.Helper()
	f, err := os.Open(filepath.Join(dir, activeFileName))
	if err != nil {
		t.Fatalf("open trace file: %v", err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan trace file: %v", err)
	}
	return lines
}

func TestSubmitAndWrite(t *testing.T) {
	cfg := testConfig(t)
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}

	const n = 100
	for i := 0; i < n; i++ {
		Submit(&Record{Rid: "rid-" + string(rune('a'+i%26)), Ts: int64(i), Status: 200})
	}
	Close()

	lines := readLines(t, cfg.Dir)
	if len(lines) != n {
		t.Fatalf("expected %d lines, got %d", n, len(lines))
	}
	// Every line must independently unmarshal: that is the core JSONL promise.
	for i, line := range lines {
		var r Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestQueueFullDropsNotBlocks(t *testing.T) {
	cfg := testConfig(t)
	cfg.QueueSize = 1
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer Close()

	// Submitting far more than the queue can hold must still return promptly:
	// the relay path is never allowed to block on trace persistence.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 20000; i++ {
			Submit(&Record{Rid: "flood", Status: 200})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Submit blocked when queue was full")
	}

	_, _, dropped, _ := Stats()
	if dropped == 0 {
		t.Fatal("expected some records to be dropped under flood")
	}
}

func TestJSONLIntegrityOnConcurrency(t *testing.T) {
	cfg := testConfig(t)
	cfg.QueueSize = 1 << 16
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}

	const goroutines, each = 50, 100
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < each; i++ {
				Submit(&Record{
					Rid:  "concurrent",
					Ts:   int64(g*each + i),
					Resp: strings.Repeat("x", 512),
				})
			}
		}(g)
	}
	wg.Wait()
	Close()

	lines := readLines(t, cfg.Dir)
	if len(lines) != goroutines*each {
		t.Fatalf("expected %d lines, got %d (interleaved or lost writes)", goroutines*each, len(lines))
	}
	for i, line := range lines {
		var r Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("line %d corrupted under concurrency: %v", i, err)
		}
	}
}

func TestRawMessageZeroEscape(t *testing.T) {
	cfg := testConfig(t)
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}

	body := `{"model":"claude-opus-4-8","messages":[{"role":"user","content":"hi"}]}`
	Submit(&Record{Rid: "raw", Req: json.RawMessage(body)})
	Close()

	lines := readLines(t, cfg.Dir)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	// RawMessage must inline verbatim; escaped quotes would mean the body was
	// marshalled as a string, doubling size and hurting readability.
	if strings.Contains(lines[0], `\"`) {
		t.Fatalf("request body was escaped instead of inlined: %s", lines[0])
	}
	if !strings.Contains(lines[0], `"model":"claude-opus-4-8"`) {
		t.Fatalf("request body not inlined verbatim: %s", lines[0])
	}
}

func TestNonJSONFallback(t *testing.T) {
	cfg := testConfig(t)
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Multipart audio uploads are not JSON and must not break the line format.
	Submit(&Record{Rid: "raw-fallback", ReqRaw: "--boundary\r\nContent-Disposition: form-data\r\n\r\n\x00\x01binary"})
	Close()

	lines := readLines(t, cfg.Dir)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var r Record
	if err := json.Unmarshal([]byte(lines[0]), &r); err != nil {
		t.Fatalf("non-JSON body broke the line: %v", err)
	}
	if !strings.Contains(r.ReqRaw, "boundary") {
		t.Fatalf("req_raw not preserved: %q", r.ReqRaw)
	}
}

func TestCloseFlushesBuffer(t *testing.T) {
	cfg := testConfig(t)
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// A single small record stays inside the 4MB buffer; only Close guarantees
	// it reaches disk.
	Submit(&Record{Rid: "flush-me", Status: 200})
	Close()

	lines := readLines(t, cfg.Dir)
	if len(lines) != 1 {
		t.Fatalf("expected buffered record to be flushed on Close, got %d lines", len(lines))
	}
	if !strings.Contains(lines[0], "flush-me") {
		t.Fatalf("unexpected content: %s", lines[0])
	}
}

func TestRotation(t *testing.T) {
	cfg := testConfig(t)
	cfg.MaxFileMB = 1
	cfg.MaxBackups = 5
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// ~4KB per record, 512 records => ~2MB, forcing at least one rotation.
	payload := strings.Repeat("y", 4<<10)
	for i := 0; i < 512; i++ {
		Submit(&Record{Rid: "rotate", Resp: payload})
	}
	Close()

	entries, err := os.ReadDir(cfg.Dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var rotated int
	var haveActive bool
	for _, e := range entries {
		switch {
		case e.Name() == activeFileName:
			haveActive = true
		case strings.HasPrefix(e.Name(), "trace-"):
			rotated++
		}
	}
	if !haveActive {
		t.Fatal("active trace file missing after rotation")
	}
	if rotated == 0 {
		t.Fatal("expected at least one rotated file")
	}
}

func TestDisabledZeroCost(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Enabled: false, Dir: dir}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer Close()

	if Enabled() {
		t.Fatal("Enabled() must be false when disabled")
	}
	// Must be a safe no-op, not a panic, and must not create files.
	Submit(&Record{Rid: "ignored"})
	if ShouldRecord(500, 1, "m") {
		t.Fatal("ShouldRecord must be false when disabled")
	}

	if _, err := os.Stat(filepath.Join(dir, activeFileName)); !os.IsNotExist(err) {
		t.Fatal("no trace file should be created when disabled")
	}
}

func TestShouldRecordErrorAlways(t *testing.T) {
	cfg := testConfig(t)
	cfg.SampleRate = 0
	cfg.ErrorAlways = true
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer Close()

	if ShouldRecord(200, 1, "gpt-4o") {
		t.Fatal("success must not be recorded at SampleRate=0")
	}
	for _, status := range []int{400, 429, 500, 502} {
		if !ShouldRecord(status, 1, "gpt-4o") {
			t.Fatalf("status %d must be recorded when ErrorAlways is on", status)
		}
	}
}

func TestShouldRecordAllowLists(t *testing.T) {
	cfg := testConfig(t)
	cfg.SampleRate = 0
	cfg.ErrorAlways = false
	cfg.Users = map[int]bool{38: true}
	cfg.Models = map[string]bool{"claude-opus-4-8": true}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer Close()

	if !ShouldRecord(200, 38, "gpt-4o") {
		t.Fatal("allow-listed user must be recorded regardless of sample rate")
	}
	if !ShouldRecord(200, 1, "claude-opus-4-8") {
		t.Fatal("allow-listed model must be recorded regardless of sample rate")
	}
	if ShouldRecord(200, 1, "gpt-4o") {
		t.Fatal("non-listed request must not be recorded at SampleRate=0")
	}
}

func TestDetectSignature(t *testing.T) {
	cases := []struct {
		name         string
		body         string
		wantNonEmpty bool
		wantPresent  bool
	}{
		{
			name:         "streaming signature delta",
			body:         `data: {"type":"content_block_delta","delta":{"type":"signature_delta","signature":"EqQBCkYIARgC"}}`,
			wantNonEmpty: true,
			wantPresent:  true,
		},
		{
			name:         "unary thinking block",
			body:         `{"content":[{"type":"thinking","thinking":"...","signature":"abc123"}]}`,
			wantNonEmpty: true,
			wantPresent:  true,
		},
		{
			// The case that matters: a proxy stripped the value but kept the key.
			name:         "empty signature means stripped",
			body:         `{"delta":{"type":"signature_delta","signature":""}}`,
			wantNonEmpty: false,
			wantPresent:  true,
		},
		{
			name:         "spaced empty signature",
			body:         `{"signature" : ""}`,
			wantNonEmpty: false,
			wantPresent:  true,
		},
		{
			// Normal for models without thinking; must be distinguishable from stripped.
			name:         "no signature key at all",
			body:         `{"choices":[{"message":{"content":"hello"}}]}`,
			wantNonEmpty: false,
			wantPresent:  false,
		},
		{
			name:         "empty then non-empty still detected",
			body:         `{"a":{"signature":""},"b":{"signature":"real"}}`,
			wantNonEmpty: true,
			wantPresent:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotNonEmpty, gotPresent := DetectSignature([]byte(tc.body))
			if gotNonEmpty != tc.wantNonEmpty {
				t.Errorf("hasNonEmpty = %v, want %v", gotNonEmpty, tc.wantNonEmpty)
			}
			if gotPresent != tc.wantPresent {
				t.Errorf("keyPresent = %v, want %v", gotPresent, tc.wantPresent)
			}
		})
	}
}

func TestFilterHeadersRedactsCredentials(t *testing.T) {
	headers := map[string]string{
		"Authorization":       "Bearer sk-secret",
		"X-Api-Key":           "sk-ant-secret",
		"X-Goog-Api-Key":      "goog-secret",
		"Cookie":              "session=secret",
		"Proxy-Authorization": "Basic secret",
		"Content-Type":        "application/json",
		"Anthropic-Version":   "2023-06-01",
		"User-Agent":          "claude-cli/1.0",
	}
	out := FilterHeaders(func(k string) string { return headers[k] })

	for _, banned := range []string{"Authorization", "X-Api-Key", "X-Goog-Api-Key", "Cookie", "Proxy-Authorization"} {
		if _, ok := out[banned]; ok {
			t.Fatalf("credential header %q must never be recorded", banned)
		}
	}
	if out["Content-Type"] != "application/json" {
		t.Fatalf("Content-Type should be kept, got %q", out["Content-Type"])
	}
	if out["Anthropic-Version"] != "2023-06-01" {
		t.Fatalf("Anthropic-Version should be kept, got %q", out["Anthropic-Version"])
	}
}

func TestConfigFromEnvNormalizes(t *testing.T) {
	t.Setenv(EnvEnabled, "true")
	t.Setenv(EnvSampleRate, "500") // out of range, must clamp to 100
	t.Setenv(EnvMaxBodyKB, "-1")   // invalid, must fall back to default
	t.Setenv(EnvQueueSize, "0")    // invalid, must fall back to default
	t.Setenv(EnvUsers, "38, 42 ,,bad")
	t.Setenv(EnvModels, "claude-opus-4-8, gpt-4o")

	cfg := ConfigFromEnv()
	if !cfg.Enabled {
		t.Fatal("Enabled should be true")
	}
	if cfg.SampleRate != 100 {
		t.Fatalf("SampleRate should clamp to 100, got %d", cfg.SampleRate)
	}
	if cfg.MaxBodyKB != defaultMaxBodyKB {
		t.Fatalf("MaxBodyKB should fall back to %d, got %d", defaultMaxBodyKB, cfg.MaxBodyKB)
	}
	if cfg.QueueSize != defaultQueueSize {
		t.Fatalf("QueueSize should fall back to %d, got %d", defaultQueueSize, cfg.QueueSize)
	}
	if !cfg.Users[38] || !cfg.Users[42] || len(cfg.Users) != 2 {
		t.Fatalf("Users parsed incorrectly: %v", cfg.Users)
	}
	if !cfg.Models["claude-opus-4-8"] || !cfg.Models["gpt-4o"] {
		t.Fatalf("Models parsed incorrectly: %v", cfg.Models)
	}
}

func TestReInitReplacesTracer(t *testing.T) {
	first := testConfig(t)
	if err := Init(first); err != nil {
		t.Fatalf("Init first: %v", err)
	}
	Submit(&Record{Rid: "first"})

	// Re-Init must stop the previous tracer and flush its data, not leak it.
	second := testConfig(t)
	if err := Init(second); err != nil {
		t.Fatalf("Init second: %v", err)
	}
	Submit(&Record{Rid: "second"})
	Close()

	firstLines := readLines(t, first.Dir)
	if len(firstLines) != 1 || !strings.Contains(firstLines[0], "first") {
		t.Fatalf("first tracer data not flushed on re-Init: %v", firstLines)
	}
	secondLines := readLines(t, second.Dir)
	if len(secondLines) != 1 || !strings.Contains(secondLines[0], "second") {
		t.Fatalf("second tracer did not receive data: %v", secondLines)
	}
}

// --- free disk guard ---

// drainQueue waits for the writer goroutine to consume everything submitted so
// far. Counters must be read before Close(), which clears the active tracer.
func drainQueue(t *testing.T, want int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		submitted, written, _, failed := Stats()
		if submitted == want && written+failed+DiskPaused() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	submitted, written, dropped, failed := Stats()
	t.Fatalf("queue did not drain: submitted=%d written=%d dropped=%d failed=%d paused=%d (want %d)",
		submitted, written, dropped, failed, DiskPaused(), want)
}

func TestDiskGuardPausesWritesWhenSpaceLow(t *testing.T) {
	cfg := testConfig(t)
	// Threshold far above any real free space, so the guard must trip.
	cfg.MinFreeDiskMB = 1 << 30 // 1PB
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(Close)

	const n = 20
	for i := 0; i < n; i++ {
		Submit(&Record{Rid: "paused", Resp: "x"})
	}
	drainQueue(t, n)

	if got := DiskPaused(); got != n {
		t.Fatalf("expected %d records discarded by the disk guard, got %d", n, got)
	}
	_, written, _, failed := Stats()
	if written != 0 {
		t.Errorf("guard let %d records through", written)
	}
	// A deliberate pause must not be reported as a malfunction.
	if failed != 0 {
		t.Errorf("disk pause counted as failure (failed=%d)", failed)
	}
}

func TestDiskGuardDisabledByZero(t *testing.T) {
	cfg := testConfig(t)
	cfg.MinFreeDiskMB = 0
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(Close)

	Submit(&Record{Rid: "allowed", Resp: "x"})
	drainQueue(t, 1)

	if got := DiskPaused(); got != 0 {
		t.Errorf("guard ran while disabled (diskPaused=%d)", got)
	}
	if _, written, _, _ := Stats(); written != 1 {
		t.Errorf("want 1 record written, got %d", written)
	}
}

// TestDiskGuardDefaultDoesNotBlockNormalRuns guards against a threshold so high
// that ordinary machines silently stop recording. The default must be satisfied
// by any host with room to run the test suite at all.
func TestDiskGuardDefaultDoesNotBlockNormalRuns(t *testing.T) {
	cfg := testConfig(t) // normalize() applies defaultMinFreeDiskMB
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(Close)

	Submit(&Record{Rid: "normal", Resp: "x"})
	drainQueue(t, 1)

	if got := DiskPaused(); got != 0 {
		free, err := freeDiskBytes(cfg.Dir)
		t.Fatalf("default threshold paused writing on this host "+
			"(diskPaused=%d, free=%dMB, threshold=%dMB, statfs err=%v)",
			got, free>>20, cfg.MinFreeDiskMB, err)
	}
}

// TestDiskCheckIsCachedBetweenSamples proves the guard does not put a statfs
// syscall on every record: within one interval the verdict is reused.
func TestDiskCheckIsCachedBetweenSamples(t *testing.T) {
	w := &fileWriter{
		dir:        t.TempDir(),
		minFree:    1, // any real disk clears this, so paused stays false
		checkEvery: time.Hour,
	}
	start := time.Now()
	if w.diskLow(start) {
		t.Fatal("unexpected pause with a 1-byte threshold")
	}
	first := w.lastCheck

	// Inside the interval the cached verdict is returned and no resample occurs.
	if w.diskLow(start.Add(time.Minute)) {
		t.Fatal("cached verdict flipped without a resample")
	}
	if !w.lastCheck.Equal(first) {
		t.Error("guard resampled before the interval elapsed")
	}

	// Past the interval it samples again.
	if w.diskLow(start.Add(2 * time.Hour)) {
		t.Fatal("unexpected pause after resample")
	}
	if w.lastCheck.Equal(first) {
		t.Error("guard failed to resample after the interval")
	}
}

func TestFreeDiskBytesReportsPlausibleValue(t *testing.T) {
	free, err := freeDiskBytes(t.TempDir())
	if err != nil {
		t.Fatalf("freeDiskBytes: %v", err)
	}
	if free == 0 {
		t.Fatal("reported zero free bytes on a writable temp dir")
	}
}

// TestDiskGuardUnreadableDirDoesNotPause encodes the fail-open choice: an
// unreadable mount must not silently switch auditing off.
func TestDiskGuardUnreadableDirDoesNotPause(t *testing.T) {
	w := &fileWriter{
		dir:        filepath.Join(t.TempDir(), "does-not-exist"),
		minFree:    1 << 40,
		checkEvery: time.Millisecond,
	}
	if w.diskLow(time.Now()) {
		t.Error("statfs failure should fail open, not pause writing")
	}
}
