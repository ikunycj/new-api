package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/pkg/observability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryExposesRelayObservabilityMetrics(t *testing.T) {
	observability.RecordRequest(observability.ProviderIkun, observability.ErrorSuccess, http.StatusOK, time.Millisecond)
	observability.RecordAttempt(observability.ProviderIkun, observability.ErrorUpstreamRateLimit, http.StatusTooManyRequests, time.Millisecond)
	observability.RecordRetry(observability.ProviderIkun, observability.ErrorUpstreamRateLimit)
	observability.RecordClientCancellation(observability.ProviderIkun, "upstream")
	finishInFlight := observability.IncInFlight(observability.ProviderIkun)
	finishInFlight()

	registry, err := NewRegistry()
	require.NoError(t, err)
	families, err := registry.Gather()
	require.NoError(t, err)

	found := make(map[string]bool)
	for _, family := range families {
		found[family.GetName()] = true
	}
	for _, name := range []string{
		"new_api_relay_requests_total",
		"new_api_relay_request_duration_seconds",
		"new_api_relay_attempts_total",
		"new_api_relay_attempt_duration_seconds",
		"new_api_relay_retries_total",
		"new_api_relay_in_flight",
		"new_api_relay_client_cancellations_total",
	} {
		assert.True(t, found[name], name)
	}
}

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
