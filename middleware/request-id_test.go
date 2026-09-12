package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDCannotBeOverriddenByClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set(common.RequestIdKey, "attacker-controlled")
	c.Request.Header.Set("X-Request-ID", "client-trace")

	RequestId()(c)

	serverID := c.GetString(common.RequestIdKey)
	require.NotEmpty(t, serverID)
	assert.NotEqual(t, "attacker-controlled", serverID)
	assert.Equal(t, serverID, recorder.Header().Get(common.RequestIdKey))
}
