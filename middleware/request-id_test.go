package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{name: "request id", headers: map[string]string{"X-Request-ID": "client-123"}, want: "client-123"},
		{name: "load test fallback", headers: map[string]string{"X-Load-Test-ID": "load:test_123"}, want: "load:test_123"},
		{name: "request id wins", headers: map[string]string{"X-Request-ID": "first", "X-Load-Test-ID": "second"}, want: "first"},
		{name: "invalid whitespace", headers: map[string]string{"X-Request-ID": "bad id"}, want: ""},
		{name: "invalid control", headers: map[string]string{"X-Request-ID": "bad\nvalue"}, want: ""},
		{name: "too long", headers: map[string]string{"X-Request-ID": strings.Repeat("a", 129)}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/", nil)
			for key, value := range tt.headers {
				c.Request.Header.Set(key, value)
			}
			assert.Equal(t, tt.want, clientTraceID(c))
		})
	}
}

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
	assert.Equal(t, "client-trace", c.GetString(common.ClientTraceIdKey))
}
