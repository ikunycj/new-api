package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayRouterRegistersAlphaSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	SetRelayRouter(engine)

	for _, route := range engine.Routes() {
		if route.Method == http.MethodPost && route.Path == "/v1/alpha/search" {
			return
		}
	}

	require.Fail(t, "POST /v1/alpha/search is not registered")
}
