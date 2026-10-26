package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/relaytrace"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/traj_record"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Protect independent selection, response streaming, and capture limits in the
// same middleware order used by the production HTTP relay router.
func TestTrajectoryRelayTraceCoexistence(t *testing.T) {
	for _, tc := range []struct {
		name              string
		trace, trajectory bool
		rate              float64
		ua, path          string
		traceRate         int
		wantTrajectory    bool
	}{
		{"both disabled", false, false, 1, "claude-cli/test", "/v1/messages", 100, false},
		{"trace only", true, false, 1, "claude-cli/test", "/v1/messages", 100, false},
		{"trajectory only", false, true, 1, "claude-cli/test", "/v1/messages", 100, true},
		{"both enabled", true, true, 1, "claude-cli/test", "/v1/messages", 100, true},
		{"trace sampled out", true, true, 1, "claude-cli/test", "/v1/messages", 0, true},
		{"trajectory sampled out", true, true, 0, "claude-cli/test", "/v1/messages", 100, false},
		{"unrecognized client", true, true, 1, "curl/test", "/v1/messages", 100, true},
		{"chat completions", true, true, 1, "opencode/test", "/v1/chat/completions", 100, true},
		{"unrecorded route", true, true, 1, "curl/test", "/v1/embeddings", 100, false},
		{"codex responses", true, true, 1, "codex_exec/test", "/v1/responses", 100, true},
		{"codex compaction", true, true, 1, "codex_exec/test", "/v1/responses/compact", 100, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			traceDir, trajDir := t.TempDir(), t.TempDir()
			require.NoError(t, relaytrace.Init(relaytrace.Config{Enabled: tc.trace, Dir: traceDir, SampleRate: tc.traceRate, MaxBodyKB: 1}))
			t.Cleanup(relaytrace.Close)
			require.NoError(t, traj_record.Init(traj_record.Config{Enabled: tc.trajectory, Dir: trajDir, SampleRate: &tc.rate}))
			t.Cleanup(traj_record.Close)
			body := `{"model":"test","messages":[{"role":"user","content":"` + strings.Repeat("x", 1500) + `"}],"metadata":{"user_id":"user_dev_account_acc_session_session-1"}}`
			if strings.HasPrefix(tc.path, "/v1/responses") {
				body = `{"model":"test","input":[{"role":"user","content":"` + strings.Repeat("x", 1500) + `"}],"client_metadata":{"session_id":"session-1"}}`
			}
			chunk, end := "data: "+strings.Repeat("y", 1500)+"\n\n", "data: [DONE]\n\n"
			writer := newDeadlineRecorder()
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(common.RequestIdKey, "coexist-rid")
				common.SetContextKey(c, constant.ContextKeyUserId, 7)
				c.Next()
			})
			router.Use(RelayTrace(), traj_record.Middleware())
			router.POST(tc.path, func(c *gin.Context) {
				raw, err := io.ReadAll(c.Request.Body)
				require.NoError(t, err)
				assert.Equal(t, body, string(raw))
				c.Request.Body = io.NopCloser(bytes.NewReader(raw))
				var parsed map[string]json.RawMessage
				require.NoError(t, common.UnmarshalBodyReusable(c, &parsed))
				assert.Equal(t, `"test"`, string(parsed["model"]))
				require.NoError(t, http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(time.Minute)))
				c.Header("Content-Type", "text/event-stream")
				c.Header("X-Test-Header", "preserved")
				c.Status(http.StatusAccepted)
				_, err = c.Writer.WriteString(chunk)
				require.NoError(t, err)
				c.Writer.Flush()
				assert.True(t, writer.Flushed)
				assert.Equal(t, chunk, writer.Body.String(), "first chunk reaches client before completion")
				_, err = c.Writer.Write([]byte(end))
				require.NoError(t, err)
			})
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(body))
			req.Header.Set("User-Agent", tc.ua)
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(writer, req)
			traj_record.Close()
			traces := readRecords(t, traceDir)
			assert.Equal(t, http.StatusAccepted, writer.Code)
			assert.Equal(t, "preserved", writer.Header().Get("X-Test-Header"))
			assert.Equal(t, chunk+end, writer.Body.String())
			if tc.trace && tc.traceRate > 0 {
				require.Len(t, traces, 1)
				assert.Equal(t, "coexist-rid", traces[0].Rid)
				assert.Equal(t, http.StatusAccepted, traces[0].Status)
				assert.True(t, traces[0].ReqTruncated)
				assert.True(t, traces[0].RespTruncated)
				assert.Equal(t, (chunk + end)[:1024], traces[0].Resp)
			} else {
				assert.Empty(t, traces)
			}
			data, err := os.ReadFile(filepath.Join(trajDir, "traj.jsonl"))
			if !tc.trajectory {
				require.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			require.NoError(t, err)
			calls := 0
			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				var rec struct {
					Type, Rid, Resp string
					ReqSize         int  `json:"req_size"`
					RespTruncated   bool `json:"resp_truncated"`
				}
				require.NoError(t, common.Unmarshal(scanner.Bytes(), &rec))
				if rec.Type != "call" {
					continue
				}
				calls++
				assert.Equal(t, "coexist-rid", rec.Rid)
				assert.Equal(t, len(body), rec.ReqSize)
				assert.Equal(t, chunk+end, rec.Resp)
				assert.False(t, rec.RespTruncated)
			}
			require.NoError(t, scanner.Err())
			want := 0
			if tc.wantTrajectory {
				want = 1
			}
			assert.Equal(t, want, calls)
		})
	}
}
