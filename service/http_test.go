package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestShouldCopyUpstreamHeaderHidesProviderIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	assert.True(t, ShouldCopyUpstreamHeader(ctx, "Content-Type", []string{"application/json"}))
	assert.True(t, ShouldCopyUpstreamHeader(ctx, "Retry-After", []string{"1"}))
	assert.False(t, ShouldCopyUpstreamHeader(ctx, "X-Channel-Name", []string{"private-provider"}))
	assert.False(t, ShouldCopyUpstreamHeader(ctx, "X-Request-ID", []string{"provider-request"}))
	assert.False(t, ShouldCopyUpstreamHeader(ctx, common.RequestIdKey, []string{"provider-request"}))
	assert.Equal(t, "provider-request", ctx.GetString(common.UpstreamRequestIdKey))
}
