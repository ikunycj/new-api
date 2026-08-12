package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPMiddlewareUsesRouteTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(HTTPMiddleware())
	engine.GET("/items/:id", middleware.RouteTag("api"), func(c *gin.Context) {
		c.String(http.StatusCreated, "created")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/items/user-controlled-value", nil)
	engine.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusCreated, recorder.Code)

	registry, err := NewRegistry()
	require.NoError(t, err)
	metrics, err := registry.Gather()
	require.NoError(t, err)

	found := false
	for _, family := range metrics {
		if family.GetName() != "new_api_http_requests_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["route"] == "/items/:id" && labels["status"] == "201" && labels["route_tag"] == "api" {
				found = true
			}
			assert.NotEqual(t, "/items/user-controlled-value", labels["route"])
		}
	}
	assert.True(t, found)
}
