package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func IsPublicRelayPath(path string) bool {
	for _, prefix := range []string{"/v1", "/v1beta", "/pg/", "/mj/", "/suno/", "/kling/", "/jimeng/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func RelayPanicRecover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				common.SysLog(fmt.Sprintf("panic detected: %v", err))
				common.SysLog(fmt.Sprintf("stacktrace from panic: %s", string(debug.Stack())))
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"message": fmt.Sprintf("Panic detected, error: %v", err),
						"type":    "new_api_panic",
					},
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
