package middleware

import (
	"context"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

var safeClientTraceID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

func clientTraceID(c *gin.Context) string {
	for _, header := range []string{"X-Request-ID", "X-Load-Test-ID"} {
		value := strings.TrimSpace(c.GetHeader(header))
		if safeClientTraceID.MatchString(value) {
			return value
		}
	}
	return ""
}

func RequestId() func(c *gin.Context) {
	return func(c *gin.Context) {
		id := common.NewRequestId()
		c.Set(common.RequestIdKey, id)
		if traceID := clientTraceID(c); traceID != "" {
			c.Set(common.ClientTraceIdKey, traceID)
		}
		ctx := context.WithValue(c.Request.Context(), common.RequestIdKey, id)
		c.Request = c.Request.WithContext(ctx)
		c.Header(common.RequestIdKey, id)
		c.Next()
	}
}
