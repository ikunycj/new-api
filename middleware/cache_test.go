package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheHeadersByResourceType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousVersion := common.Version
	common.Version = "test-build"
	t.Cleanup(func() {
		common.Version = previousVersion
	})

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "hashed javascript",
			path:     "/static/js/index.0123abcdef.js",
			expected: "public, max-age=31536000, immutable",
		},
		{
			name:     "hashed docs payload",
			path:     "/static/docs/zh/codex.90ef12ab34cd.json",
			expected: "public, max-age=31536000, immutable",
		},
		{
			name:     "docs manifest",
			path:     "/static/docs/manifest.json",
			expected: "no-cache, must-revalidate",
		},
		{
			name:     "unhashed logo",
			path:     "/logo.png",
			expected: "no-cache, must-revalidate",
		},
		{
			name:     "application route",
			path:     "/dashboard",
			expected: "no-cache, must-revalidate",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(Cache())
			router.GET("/*path", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusNoContent, recorder.Code)
			assert.Equal(t, test.expected, recorder.Header().Get("Cache-Control"))
			assert.Equal(t, "test-build", recorder.Header().Get("Cache-Version"))
		})
	}
}
