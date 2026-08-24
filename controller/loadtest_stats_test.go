package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLoadTestChannelStatsRejectsInvalidBody(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	user := &model.User{Username: "loadtest-invalid-body", Group: "default", AffCode: "loadtest-invalid-body"}
	require.NoError(t, db.Create(user).Error)
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Set("role", common.RoleAdminUser)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/self/loadtest-stats", strings.NewReader("{"))

	GetLoadTestChannelStats(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, "invalid request_ids", response.Message)
}

func TestGetLoadTestChannelStatsRejectsTooManyRequestIDs(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	user := &model.User{Username: "loadtest-too-many", Group: "default", AffCode: "loadtest-too-many"}
	require.NoError(t, db.Create(user).Error)
	gin.SetMode(gin.TestMode)
	requestIDs := strings.Repeat(`"request",`, maxLoadTestStatsRequestIDs) + `"request"`
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Set("role", common.RoleAdminUser)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/self/loadtest-stats", strings.NewReader(`{"request_ids":[`+requestIDs+`]}`))

	GetLoadTestChannelStats(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, "too many request_ids", response.Message)
}

func TestGetLoadTestChannelStatsAllowsCommonUser(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	model.InitChannelRoutingCache()
	user := &model.User{Username: "loadtest-common", Group: "default", UserType: model.UserTypeToB, AffCode: "loadtest-common"}
	require.NoError(t, db.Create(user).Error)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Set("role", common.RoleCommonUser)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/self/loadtest-stats", strings.NewReader(`{"request_ids":[]}`))

	GetLoadTestChannelStats(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestGetLoadTestChannelStatsRejectsDefaultUser(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	model.InitChannelRoutingCache()
	user := &model.User{Username: "loadtest-default", Group: "default", AffCode: "loadtest-default"}
	require.NoError(t, db.Create(user).Error)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Set("role", common.RoleCommonUser)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/self/loadtest-stats", strings.NewReader(`{"request_ids":[]}`))

	GetLoadTestChannelStats(ctx)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestGetLoadTestChannelStatsUsesUserTypeInsteadOfBillingGroup(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	model.InitChannelRoutingCache()
	user := &model.User{Username: "loadtest-toc", Group: "toB", UserType: model.UserTypeToC, AffCode: "loadtest-toc"}
	require.NoError(t, db.Create(user).Error)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Set("role", common.RoleCommonUser)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/self/loadtest-stats", strings.NewReader(`{"request_ids":[]}`))

	GetLoadTestChannelStats(ctx)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestGetLoadTestChannelStatsAllowsAdmin(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	user := &model.User{Username: "loadtest-admin", Group: "default", AffCode: "loadtest-admin"}
	require.NoError(t, db.Create(user).Error)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Set("role", common.RoleAdminUser)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/self/loadtest-stats", strings.NewReader(`{"request_ids":[]}`))

	GetLoadTestChannelStats(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
}
