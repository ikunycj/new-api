package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type realtimeSnapshotResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    model.RealtimeSnapshot `json:"data"`
}

type realtimeUsersResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		WindowSeconds int                         `json:"window_seconds"`
		NodeName      string                      `json:"node_name"`
		Now           int64                       `json:"now"`
		Users         []model.RealtimeUserSummary `json:"users"`
	} `json:"data"`
}

// enableRealtimeForTest turns the in-process counters on and gives the test a
// clean registry, mirroring what main.go does at startup.
func enableRealtimeForTest(t *testing.T) {
	t.Helper()
	// The admin list resolves usernames through the shared DB, so the harness
	// has to be in place before any handler runs.
	setupModelListControllerTestDB(t)
	model.EnableRealtimeMetricsForTest()
	t.Cleanup(model.ResetRealtimeMetricsForTest)
}

func TestGetSelfRealtimeMetricsIgnoresClientSuppliedUserId(t *testing.T) {
	enableRealtimeForTest(t)

	// The caller is user 1, but the query string and a body both try to steer
	// the handler at user 2. The handler must take the id from the session only,
	// which is what stops a user reading another account's throughput.
	model.RecordRealtimeUsage(1, 100)
	model.RecordRealtimeUsage(2, 999_999)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/data/realtime/self?user_id=2&id=2&username=bob",
		nil,
	)

	GetSelfRealtimeMetrics(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload realtimeSnapshotResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success, payload.Message)

	require.Equal(t, 1, payload.Data.UserID, "the snapshot must describe the session user")
	for _, window := range payload.Data.Windows {
		require.Equal(t, 1, window.Requests,
			"only the session user's traffic may be counted (%ds window)", window.WindowSeconds)
		require.Equal(t, 100, window.Tokens,
			"another user's tokens must never leak into the response (%ds window)", window.WindowSeconds)
	}
}

func TestGetSelfRealtimeMetricsWithNoTrafficReturnsZeroes(t *testing.T) {
	enableRealtimeForTest(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 12345)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/realtime/self", nil)

	GetSelfRealtimeMetrics(ctx)

	var payload realtimeSnapshotResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Equal(t, 12345, payload.Data.UserID)
	require.Len(t, payload.Data.Windows, 3)
	for _, window := range payload.Data.Windows {
		require.Zero(t, window.Requests)
		require.Zero(t, window.RPM)
	}
}

func TestGetRealtimeMetricsUsersRejectsNonAllowedWindow(t *testing.T) {
	enableRealtimeForTest(t)

	// An arbitrary window would be reported under a label the model cannot
	// actually back, because the ring only retains six hours.
	for _, window := range []string{"0", "-60", "86400", "abc"} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(
			http.MethodGet, "/api/data/realtime/users?window_seconds="+window, nil)

		GetRealtimeMetricsUsers(ctx)

		var payload realtimeUsersResponse
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
		require.False(t, payload.Success, "window_seconds=%s must be rejected", window)
		require.Equal(t, "invalid window_seconds", payload.Message)
	}
}

func TestGetRealtimeMetricsUsersDefaultsToSixtySeconds(t *testing.T) {
	enableRealtimeForTest(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/realtime/users", nil)

	GetRealtimeMetricsUsers(ctx)

	var payload realtimeUsersResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Equal(t, 60, payload.Data.WindowSeconds)
}

func TestGetRealtimeMetricsUsersOmitsIdleUsers(t *testing.T) {
	enableRealtimeForTest(t)

	model.RecordRealtimeUsage(1, 100)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/realtime/users", nil)

	GetRealtimeMetricsUsers(ctx)

	var payload realtimeUsersResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Len(t, payload.Data.Users, 1)
	require.Equal(t, 1, payload.Data.Users[0].UserID)
}

func TestGetRealtimeMetricsByUserRejectsInvalidId(t *testing.T) {
	enableRealtimeForTest(t)

	for _, id := range []string{"abc", "0", "-5"} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Params = gin.Params{{Key: "id", Value: id}}
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/realtime/users/"+id, nil)

		GetRealtimeMetricsByUser(ctx)

		var payload realtimeSnapshotResponse
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
		require.False(t, payload.Success, "id=%s must be rejected", id)
		require.Equal(t, "invalid user id", payload.Message)
	}
}

func TestGetRealtimeMetricsByUserReturnsThatUser(t *testing.T) {
	enableRealtimeForTest(t)

	model.RecordRealtimeUsage(77, 250)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "77"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/realtime/users/77", nil)

	GetRealtimeMetricsByUser(ctx)

	var payload realtimeSnapshotResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Equal(t, 77, payload.Data.UserID)
	require.Equal(t, 1, payload.Data.Windows[0].Requests)
	require.Equal(t, 250, payload.Data.Windows[0].Tokens)
}

func TestGetSelfRealtimeMetricsSerializesCacheHitRate(t *testing.T) {
	enableRealtimeForTest(t)

	// 800 of 1000 input tokens served from cache.
	model.RecordRealtimeCacheUsage(1, 500, 800, 1000, true)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/realtime/self", nil)

	GetSelfRealtimeMetrics(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload realtimeSnapshotResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success, payload.Message)

	for _, window := range payload.Data.Windows {
		require.Equal(t, 800, window.CacheReadTokens, "%ds window", window.WindowSeconds)
		require.Equal(t, 1000, window.InputTokensTotal, "%ds window", window.WindowSeconds)
		require.NotNil(t, window.CacheHitRate, "%ds window", window.WindowSeconds)
		require.InDelta(t, 0.8, *window.CacheHitRate, 1e-9, "%ds window", window.WindowSeconds)
	}
}

func TestGetSelfRealtimeMetricsEmitsNullCacheHitRateWithoutSample(t *testing.T) {
	enableRealtimeForTest(t)

	// Traffic with no cache metadata. The wire format must carry an explicit
	// null so the client can tell "unmeasured" from "0% hit rate" — a plain 0
	// here would be rendered as a confident zero on the dashboard.
	model.RecordRealtimeCacheUsage(1, 500, 0, 1000, false)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/realtime/self", nil)

	GetSelfRealtimeMetrics(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"cache_hit_rate":null`)

	var payload realtimeSnapshotResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	for _, window := range payload.Data.Windows {
		require.Nil(t, window.CacheHitRate, "%ds window", window.WindowSeconds)
		require.Zero(t, window.InputTokensTotal, "%ds window", window.WindowSeconds)
	}
}
