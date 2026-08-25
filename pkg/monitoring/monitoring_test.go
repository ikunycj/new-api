package monitoring

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/observability"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRegistryExposesRelayObservabilityMetrics(t *testing.T) {
	const channelID = 38
	observability.RecordRequest(observability.ProviderIkun, channelID, observability.ErrorSuccess, http.StatusOK, time.Millisecond)
	observability.RecordAttempt(observability.ProviderIkun, channelID, observability.ErrorUpstreamRateLimit, http.StatusTooManyRequests, time.Millisecond)
	observability.RecordRetry(observability.ProviderIkun, channelID, observability.ErrorUpstreamRateLimit)
	observability.RecordClientCancellation(observability.ProviderIkun, channelID, "upstream")
	finishInFlight := observability.IncInFlight(observability.ProviderIkun, channelID)
	finishInFlight()

	registry, err := NewRegistry()
	require.NoError(t, err)
	families, err := registry.Gather()
	require.NoError(t, err)

	found := make(map[string]bool)
	channelLabels := make(map[string]bool)
	for _, family := range families {
		found[family.GetName()] = true
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "channel_id" && label.GetValue() == "38" {
					channelLabels[family.GetName()] = true
				}
			}
		}
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
		assert.True(t, channelLabels[name], name+" channel_id")
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

func TestRegistryExposesEveryConfiguredChannel(t *testing.T) {
	previousDB := model.DB
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.Channel{}))
	model.DB = testDB
	t.Cleanup(func() { model.DB = previousDB })

	require.NoError(t, testDB.Create(&[]model.Channel{
		{Id: 38, Name: "Primary", Status: common.ChannelStatusEnabled, Group: "common"},
		{Id: 39, Name: "Backup", Status: common.ChannelStatusManuallyDisabled, Group: "common"},
	}).Error)

	registry, err := NewRegistry()
	require.NoError(t, err)
	families, err := registry.Gather()
	require.NoError(t, err)

	channels := map[string]map[string]string{}
	for _, family := range families {
		if family.GetName() != "alltoken_channel_info" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			channels[labels["channel_id"]] = labels
		}
	}
	require.Len(t, channels, 2)
	assert.Equal(t, "#38 Primary", channels["38"]["channel_label"])
	assert.Equal(t, "enabled", channels["38"]["status"])
	assert.Equal(t, "manually_disabled", channels["39"]["status"])
}

func TestAlertEmailHandlerSendsProfitGuardNotification(t *testing.T) {
	var subject, receiver, content string
	handler := newAlertEmailHandler(func(gotSubject, gotReceiver, gotContent string) error {
		subject, receiver, content = gotSubject, gotReceiver, gotContent
		return nil
	})
	payload := []byte(`{"status":"firing","alerts":[{"status":"firing","labels":{"alertname":"AllTokenProfitMarginRisk"},"annotations":{"summary":"Profit margin risk detected","description":"Three warnings were recorded in two hours."},"startsAt":"2026-08-25T12:00:00Z"}]}`)
	request := httptest.NewRequest(http.MethodPost, "/internal/alerts/email?to=654125664%40qq.com", bytes.NewReader(payload))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, "[AllToken] Profit margin alert firing", subject)
	assert.Equal(t, "654125664@qq.com", receiver)
	assert.Contains(t, content, "Three warnings were recorded in two hours.")
}

func TestAlertEmailHandlerRejectsInvalidRecipient(t *testing.T) {
	handler := newAlertEmailHandler(func(_, _, _ string) error {
		t.Fatal("email sender must not be called")
		return nil
	})
	request := httptest.NewRequest(http.MethodPost, "/internal/alerts/email?to=invalid", bytes.NewReader([]byte(`{"status":"firing","alerts":[{}]}`)))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
