package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	projecti18n "github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/setting"
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
