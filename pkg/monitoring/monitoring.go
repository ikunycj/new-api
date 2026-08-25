package monitoring

import (
	"database/sql"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/observability"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests handled by the application.",
	}, []string{"method", "route", "status", "route_tag"})
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "new_api",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
	}, []string{"method", "route", "status", "route_tag"})
	httpResponseSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "new_api",
		Subsystem: "http",
		Name:      "response_size_bytes",
		Help:      "HTTP response size in bytes.",
		Buckets:   prometheus.ExponentialBuckets(128, 4, 9),
	}, []string{"method", "route", "status", "route_tag"})
	httpInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "new_api",
		Subsystem: "http",
		Name:      "requests_in_flight",
		Help:      "Current number of HTTP requests being handled.",
	})
	channelInfoDesc = prometheus.NewDesc(
		"alltoken_channel_info",
		"Configured channels and their current routing metadata.",
		[]string{"channel_id", "channel_name", "channel_label", "status", "group"},
		nil,
	)
)

type channelInfoCollector struct{}

type alertmanagerWebhook struct {
	Status string `json:"status"`
	Alerts []struct {
		Status      string            `json:"status"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
		StartsAt    string            `json:"startsAt"`
	} `json:"alerts"`
}

func (channelInfoCollector) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- channelInfoDesc
}

func (channelInfoCollector) Collect(metrics chan<- prometheus.Metric) {
	if model.DB == nil {
		return
	}
	var channels []model.Channel
	if err := model.DB.Select("id", "name", "status", "group").Order("id ASC").Find(&channels).Error; err != nil {
		common.SysError("collect channel info metrics: " + err.Error())
		return
	}
	for _, channel := range channels {
		status := "unknown"
		switch channel.Status {
		case common.ChannelStatusEnabled:
			status = "enabled"
		case common.ChannelStatusManuallyDisabled:
			status = "manually_disabled"
		case common.ChannelStatusAutoDisabled:
			status = "auto_disabled"
		}
		metrics <- prometheus.MustNewConstMetric(
			channelInfoDesc,
			prometheus.GaugeValue,
			1,
			strconv.Itoa(channel.Id),
			channel.Name,
			fmt.Sprintf("#%d %s", channel.Id, channel.Name),
			status,
			channel.Group,
		)
	}
}

// HTTPMiddleware records bounded-cardinality HTTP metrics. Gin's route template
// is used instead of the raw URL, so user-controlled path values never become labels.
func HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		httpInFlight.Inc()
		defer httpInFlight.Dec()

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		routeTag := c.GetString(middleware.RouteTagKey)
		if routeTag == "" {
			routeTag = "unknown"
		}
		status := strconv.Itoa(c.Writer.Status())
		labels := []string{c.Request.Method, route, status, routeTag}
		httpRequests.WithLabelValues(labels...).Inc()
		httpDuration.WithLabelValues(labels...).Observe(time.Since(startedAt).Seconds())
		responseSize := c.Writer.Size()
		if responseSize < 0 {
			responseSize = 0
		}
		httpResponseSize.WithLabelValues(labels...).Observe(float64(responseSize))

		if errorClass := c.GetString(observability.ContextErrorClassKey); errorClass != "" {
			duration := time.Since(startedAt)
			observability.RecordRequest(observability.ProviderOther, 0, errorClass, c.Writer.Status(), duration)
			observability.LogEvent(c, observability.Event{
				Event:         "request_finished",
				RequestID:     c.GetString(common.RequestIdKey),
				ClientTraceID: c.GetString(common.ClientTraceIdKey),
				Provider:      observability.ProviderOther,
				Route:         route,
				Status:        c.Writer.Status(),
				ErrorClass:    errorClass,
				DurationMS:    duration.Milliseconds(),
			})
		}
	}
}

// NewRegistry returns the complete application registry. It is exported to keep
// the collector set directly testable without starting a TCP listener.
func NewRegistry() (*prometheus.Registry, error) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		httpRequests,
		httpDuration,
		httpResponseSize,
		httpInFlight,
		channelInfoCollector{},
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: "new_api",
			Name:      "build_info",
			Help:      "Application build information.",
			ConstLabels: prometheus.Labels{
				"version": common.Version,
			},
		}, func() float64 { return 1 }),
	)
	for _, collector := range observability.Collectors() {
		registry.MustRegister(collector)
	}

	if model.DB != nil {
		mainDB, err := model.DB.DB()
		if err != nil {
			return nil, fmt.Errorf("access main database pool: %w", err)
		}
		registerDatabaseCollectors(registry, "main", mainDB)
	}
	if model.LOG_DB != nil && model.LOG_DB != model.DB {
		logDB, err := model.LOG_DB.DB()
		if err != nil {
			return nil, fmt.Errorf("access log database pool: %w", err)
		}
		registerDatabaseCollectors(registry, "log", logDB)
	}
	if common.RedisEnabled && common.RDB != nil {
		registerRedisCollectors(registry)
	}

	return registry, nil
}

func Start() (*http.Server, error) {
	if os.Getenv("ENABLE_METRICS") != "true" {
		return nil, nil
	}
	registry, err := NewRegistry()
	if err != nil {
		return nil, err
	}
	port := common.GetEnvOrDefault("METRICS_PORT", 8006)
	address := net.JoinHostPort(common.GetEnvOrDefaultString("METRICS_BIND_ADDRESS", "0.0.0.0"), strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen for metrics on %s: %w", address, err)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("/internal/alerts/email", newAlertEmailHandler(common.SendEmail))
	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			common.SysError("metrics server stopped: " + err.Error())
		}
	}()
	return server, nil
}

func newAlertEmailHandler(sendEmail func(subject, receiver, content string) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		recipient, err := mail.ParseAddress(strings.TrimSpace(request.URL.Query().Get("to")))
		if err != nil || recipient.Address == "" {
			http.Error(w, "invalid recipient", http.StatusBadRequest)
			return
		}

		request.Body = http.MaxBytesReader(w, request.Body, 64<<10)
		var payload alertmanagerWebhook
		if err := common.DecodeJson(request.Body, &payload); err != nil {
			http.Error(w, "invalid alert payload", http.StatusBadRequest)
			return
		}
		if len(payload.Alerts) == 0 {
			http.Error(w, "alert payload is empty", http.StatusBadRequest)
			return
		}

		status := strings.ToLower(strings.TrimSpace(payload.Status))
		if status != "resolved" {
			status = "firing"
		}
		subject := fmt.Sprintf("[AllToken] Profit margin alert %s", status)
		var content strings.Builder
		content.WriteString("<h2>Profit margin protection alert</h2>")
		content.WriteString("<p>Status: <strong>" + html.EscapeString(status) + "</strong></p>")
		matchedAlerts := 0
		for _, alert := range payload.Alerts {
			if alert.Labels["alertname"] != "AllTokenProfitMarginRisk" {
				continue
			}
			matchedAlerts++
			content.WriteString("<hr><p><strong>" + html.EscapeString(alert.Annotations["summary"]) + "</strong></p>")
			content.WriteString("<p>" + html.EscapeString(alert.Annotations["description"]) + "</p>")
			if alert.StartsAt != "" {
				content.WriteString("<p>Started at: " + html.EscapeString(alert.StartsAt) + "</p>")
			}
		}
		if matchedAlerts == 0 {
			http.Error(w, "profit margin alert is missing", http.StatusBadRequest)
			return
		}
		if err := sendEmail(subject, recipient.Address, content.String()); err != nil {
			common.SysError("send profit guard alert email: " + err.Error())
			http.Error(w, "email delivery failed", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func registerDatabaseCollectors(registry *prometheus.Registry, name string, database *sql.DB) {
	states := map[string]func(sql.DBStats) float64{
		"max_open": func(stats sql.DBStats) float64 { return float64(stats.MaxOpenConnections) },
		"open":     func(stats sql.DBStats) float64 { return float64(stats.OpenConnections) },
		"in_use":   func(stats sql.DBStats) float64 { return float64(stats.InUse) },
		"idle":     func(stats sql.DBStats) float64 { return float64(stats.Idle) },
	}
	for state, value := range states {
		state := state
		value := value
		registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace:   "new_api",
			Subsystem:   "database",
			Name:        "connections",
			Help:        "Database connection pool size by state.",
			ConstLabels: prometheus.Labels{"database": name, "state": state},
		}, func() float64 { return value(database.Stats()) }))
	}
	counters := map[string]func(sql.DBStats) float64{
		"wait_count":            func(stats sql.DBStats) float64 { return float64(stats.WaitCount) },
		"wait_duration_seconds": func(stats sql.DBStats) float64 { return stats.WaitDuration.Seconds() },
		"max_idle_closed":       func(stats sql.DBStats) float64 { return float64(stats.MaxIdleClosed) },
		"max_lifetime_closed":   func(stats sql.DBStats) float64 { return float64(stats.MaxLifetimeClosed) },
	}
	for metric, value := range counters {
		metric := metric
		value := value
		registry.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace:   "new_api",
			Subsystem:   "database",
			Name:        metric + "_total",
			Help:        "Cumulative database pool statistic " + metric + ".",
			ConstLabels: prometheus.Labels{"database": name},
		}, func() float64 { return value(database.Stats()) }))
	}
}

func registerRedisCollectors(registry *prometheus.Registry) {
	poolStates := map[string]func() float64{
		"total": func() float64 { return float64(common.RDB.PoolStats().TotalConns) },
		"idle":  func() float64 { return float64(common.RDB.PoolStats().IdleConns) },
		"stale": func() float64 { return float64(common.RDB.PoolStats().StaleConns) },
	}
	for state, value := range poolStates {
		state := state
		value := value
		registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace:   "new_api",
			Subsystem:   "redis",
			Name:        "pool_connections",
			Help:        "Redis client connection pool size by state.",
			ConstLabels: prometheus.Labels{"state": state},
		}, value))
	}
	poolCounters := map[string]func() float64{
		"hits":     func() float64 { return float64(common.RDB.PoolStats().Hits) },
		"misses":   func() float64 { return float64(common.RDB.PoolStats().Misses) },
		"timeouts": func() float64 { return float64(common.RDB.PoolStats().Timeouts) },
	}
	for metric, value := range poolCounters {
		metric := metric
		value := value
		registry.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "new_api",
			Subsystem: "redis",
			Name:      "pool_" + metric + "_total",
			Help:      "Cumulative Redis client pool " + metric + ".",
		}, value))
	}
}
