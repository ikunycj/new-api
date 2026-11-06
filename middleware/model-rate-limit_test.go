package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMemoryModelRateLimitRejectsAfterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := int(time.Now().UnixNano() & 0x3fffffff)

	handler := memoryRateLimitHandler(60, 1, 0)

	first, _ := gin.CreateTestContext(httptest.NewRecorder())
	first.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	first.Set("id", userID)
	handler(first)

	recorder := httptest.NewRecorder()
	second, _ := gin.CreateTestContext(recorder)
	second.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	second.Set("id", userID)
	handler(second)

	assert.True(t, second.IsAborted())
	assert.Equal(t, http.StatusTooManyRequests, second.Writer.Status())
}
