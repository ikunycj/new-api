package traj_record

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSessionSampling(t *testing.T) {
	rate := 0.5
	r := &recorder{cfg: Config{SampleRate: &rate}}
	// Fixed vectors preserve decisions across deployments and refactors.
	for _, tc := range []struct {
		session string
		want    bool
	}{
		{"session-0", false}, {"session-1", true},
	} {
		session, want := tc.session, tc.want
		require.Equal(t, want, r.sampleRequest(codexClient, 7, nil, codexBody(session, "root", "")))
		for _, body := range [][]byte{
			codexBody(session, "child", "root", `{"type":"message","role":"user","content":"child"}`),
			codexBody(session, "grandchild", "child"),
			codexBody(session, "root", "", `{"type":"compaction","encrypted_content":"summary"}`),
		} {
			restarted := &recorder{cfg: Config{SampleRate: &rate, Instance: "other"}}
			require.Equal(t, want, restarted.sampleRequest(codexClient, 7, nil, body))
		}
		for _, name := range []string{"Session-Id", "Session_id"} {
			h := http.Header{}
			h.Set(name, session)
			require.Equal(t, want, r.sampleRequest(codexClient, 7, h, []byte(`{"input":[]}`)))
		}
		wantClaude := r.sampleRequest(claudeClient, 7, nil, claudeBody(session))
		require.Equal(t, wantClaude, r.sampleRequest(claudeClient, 7, nil, claudeBody(session, `{"role":"user","content":"summary or child"}`)))
	}
	require.True(t, r.sampleRequest(codexClient, 7, nil, codexBody("", "child", "root")))
	require.True(t, r.sampleRequest(claudeClient, 7, nil, []byte(`invalid`)))
}

func TestSamplingConfig(t *testing.T) {
	for _, raw := range []string{"0", "0.1", "1", "-1", "2", "NaN", "+Inf", "oops", ""} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv(EnvSampleRate, raw)
			cfg := ConfigFromEnv()
			want := 0.0
			if raw == "0.1" {
				want = 0.1
			}
			if raw == "1" {
				want = 1
			}
			require.Equal(t, want, *cfg.SampleRate)
		})
	}
	r := &recorder{}
	require.True(t, r.sampleRequest(claudeClient, 7, nil, nil))
}

func TestSampledOutMiddlewarePassThrough(t *testing.T) {
	rate := 0.0
	require.NoError(t, Init(Config{Enabled: true, Dir: t.TempDir(), SampleRate: &rate}))
	t.Cleanup(Close)
	r := current()
	engine := gin.New()
	engine.Use(Middleware())
	engine.POST("/v1/messages", func(c *gin.Context) {
		_, captured := c.Writer.(*captureWriter)
		require.False(t, captured)
		body, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Equal(t, `{"messages":[]}`, string(body))
		c.String(http.StatusOK, "response")
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"messages":[]}`))
	req.Header.Set("User-Agent", "claude-cli/test")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	Close()
	require.Equal(t, "response", w.Body.String())
	require.Zero(t, r.written.Load())
	require.Zero(t, r.dropped.Load())
}
