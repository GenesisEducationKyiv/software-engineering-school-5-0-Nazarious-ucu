package metrics

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type ResponseWriter struct {
	http.ResponseWriter
	Status int
}

type Metrics struct {
	// SLI Metrics - Service Level Indicators
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Business Metrics - Domain specific
	SubscriptionsCreated  *prometheus.CounterVec
	SubscriptionsActive   prometheus.Gauge
	WeatherProcessingTime *prometheus.HistogramVec
	WeatherFailures       *prometheus.CounterVec

	// System Metrics - Resource utilization
	ServiceUptime prometheus.Gauge
	// Other system metrics - using collectors

	// Error Metrics - Detailed error classification
	BusinessErrors  *prometheus.CounterVec
	TechnicalErrors *prometheus.CounterVec
}

func NewMetrics(serviceName string, db *sql.DB, dbName string) *Metrics {
	m := &Metrics{}

	// Consistent labeling scheme across all metrics
	httpLabels := []string{"method", "endpoint", "status_class"}
	businessLabels := []string{"subscription_type", "payment_method"}
	errorLabels := []string{"error_type", "error_code", "severity"}

	// SLI Metrics - Perfect for SLO definition
	m.HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_http_requests_total",
			Help: "Total number of HTTP requests (SLI: Request Rate)",
		},
		httpLabels,
	)

	m.HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    serviceName + "_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds (SLI: Request Duration)",
			Buckets: prometheus.DefBuckets,
		},
		httpLabels,
	)

	// Business Metrics - Domain specific
	m.SubscriptionsCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_subscriptions_created_total",
			Help: "Total number of subscriptions created",
		},
		businessLabels,
	)

	m.SubscriptionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: serviceName + "_active_subscriptions",
			Help: "Current number of active subscriptions",
		},
	)

	m.WeatherProcessingTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    serviceName + "_weather_processing_time_seconds",
			Help:    "Time taken to process weather requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		httpLabels,
	)

	m.WeatherFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_weather_failures_total",
			Help: "Total number of weather request failures",
		},
		httpLabels,
	)

	// System Metrics - Resource utilization
	m.ServiceUptime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: serviceName + "_service_uptime_seconds",
			Help: "Service uptime in seconds",
		},
	)

	m.BusinessErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_business_errors_total",
			Help: "Total number of business errors",
		},
		errorLabels,
	)
	m.TechnicalErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_technical_errors_total",
			Help: "Total number of technical errors",
		},
		errorLabels,
	)
	// Register all metrics
	prometheus.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.SubscriptionsCreated,
		m.SubscriptionsActive,
		m.WeatherProcessingTime,
		m.WeatherFailures,
		m.ServiceUptime,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewDBStatsCollector(db, dbName),
		m.BusinessErrors,
		m.TechnicalErrors,
	)

	return m
}

func (m *Metrics) InstrumentHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &ResponseWriter{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()

		statusClass := getStatusClass(wrapped.Status)

		labels := prometheus.Labels{"method": r.Method, "endpoint": r.URL.Path, "status_class": statusClass}

		m.HTTPRequestsTotal.With(labels).Inc()
		m.HTTPRequestDuration.With(labels).Observe(duration)
		m.ServiceUptime.SetToCurrentTime()

		if statusClass == "4xx" {
			m.BusinessErrors.With(prometheus.Labels{
				"error_type": "client_error",
				"error_code": http.StatusText(wrapped.Status),
				"severity":   "warning",
			}).Inc()
		}

		if statusClass == "5xx" {
			m.TechnicalErrors.With(prometheus.Labels{
				"error_type": "server_error",
				"error_code": http.StatusText(wrapped.Status),
				"severity":   "critical",
			}).Inc()
		}
	})
}

func getStatusClass(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= http.StatusInternalServerError:
		return "5xx"
	default:
		return "unknown"
	}
}
