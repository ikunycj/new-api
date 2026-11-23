package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	projecti18n "github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaygroundRoutesShareGroupOverrideHandling(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "chat completions", path: "/pg/chat/completions", want: true},
		{name: "image generations", path: "/pg/images/generations", want: true},
		{name: "public image generations", path: "/v1/images/generations", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", test.path, nil)
			assert.Equal(t, test.want, isPlaygroundRequest(request.URL.Path))
		})
	}
}

func TestPlaygroundGroupOverrideUsesAccountGroupPermissions(t *testing.T) {
	require.NoError(t, projecti18n.Init())
	previousPricingGroups := setting.UserGroupPricingGroups2JSONString()
	require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(`{"restricted-account":["default"]}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(previousPricingGroups))
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		"POST",
		"/pg/chat/completions",
		strings.NewReader(`{"model":"test-model","group":"vip"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "restricted-account")
	common.SetContextKey(ctx, constant.ContextKeyUsingGroup, "routing-context-with-unrestricted-default")

	Distribute()(ctx)

	assert.True(t, ctx.IsAborted())
	assert.Equal(t, 403, recorder.Code)
	assert.Equal(t, "routing-context-with-unrestricted-default", common.GetContextKeyString(ctx, constant.ContextKeyUsingGroup))
}

func TestDistributeRejectsModelForbiddenByEveryPricingGroup(t *testing.T) {
	require.NoError(t, projecti18n.Init())
	previousAllowlist := ratio_setting.PricingGroupModelAllowlist2JSONString()
	previousRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupModelAllowlistByJSONString(previousAllowlist))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"group-a":1,"group-b":1}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupModelAllowlistByJSONString(`{"group-a":["allowed-a"],"group-b":["allowed-b"]}`))

	for _, group := range []string{"group-a", "auto"} {
		t.Run(group, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"denied-model"}`))
			ctx.Request.Header.Set("Content-Type", "application/json")
			common.SetContextKey(ctx, constant.ContextKeyUsingGroup, group)
			common.SetContextKey(ctx, constant.ContextKeyTokenGroupCandidates, []string{"group-a", "group-b"})
			common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, true)

			Distribute()(ctx)

			var response struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.True(t, ctx.IsAborted())
			assert.Equal(t, http.StatusForbidden, recorder.Code)
			assert.Equal(t, "access_denied", response.Error.Code)
		})
	}
}
