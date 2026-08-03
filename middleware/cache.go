package middleware

import (
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

var hashedAssetPattern = regexp.MustCompile(`(?:^|[.-])[0-9a-f]{8,64}(?:[.-]|$)`)

func Cache() func(c *gin.Context) {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/static/") && hashedAssetPattern.MatchString(path) {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-cache, must-revalidate")
		}
		c.Header("Cache-Version", common.Version)
		c.Next()
	}
}
