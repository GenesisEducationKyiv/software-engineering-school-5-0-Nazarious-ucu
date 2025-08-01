package metrics

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	grpcProm "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"google.golang.org/grpc"
)

const divisor = 100

// Metrics defines all Prometheus metrics for the subscription service.
type Metrics struct {
	// RED (Rate, Errors, Duration) for HTTP
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestsInFlight prometheus.Gauge
	HTTPRequestDuration  *prometheus.HistogramVec

	// Business metrics
	SubscriptionsCreated   *prometheus.CounterVec // by frequency
	SubscriptionsConfirmed prometheus.Counter     // total confirms
	SubscriptionsCanceled  prometheus.Counter     // total unsubscribes

	// Cron job metrics (USE: Utilization, Saturation, Errors)
	CronRuns        *prometheus.CounterVec // by frequency
	CronRunDuration *prometheus.HistogramVec

	// RabbitMQ publish metrics
	RabbitPublishTotal *prometheus.CounterVec // by routing_key, result

	// gRPC server metrics
	// courtesy grpc_prometheus

	// System metrics
	ServiceUptime prometheus.Gauge

	// Errors metrics
	BusinessErrors  *prometheus.CounterVec
	TechnicalErrors *prometheus.CounterVec
}

// NewMetrics creates and registers all metrics under the given namespace.
func NewMetrics(namespace string, db *sql.DB, dbName string) *Metrics {
	registry := prometheus.NewRegistry()
	errorLabels := []string{"error_type", "severity"}
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "HTTP requests total",
			},
			[]string{"method", "endpoint", "status_class"},
		),
		HTTPRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "In-flight HTTP requests",
			},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "Duration of HTTP requests",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		SubscriptionsCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "subscriptions_created_total",
				Help:      "Total subscriptions created",
			},
			[]string{"frequency"},
		),
		SubscriptionsConfirmed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "subscriptions_confirmed_total",
				Help:      "Total subscriptions confirmed",
			},
		),
		SubscriptionsCanceled: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "subscriptions_canceled_total",
				Help:      "Total subscriptions canceled",
			},
		),

		CronRuns: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cron_runs_total",
				Help:      "Cron job executions",
			},
			[]string{"frequency"},
		),
		CronRunDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "cron_run_duration_seconds",
				Help:      "Duration of cron jobs",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"frequency"},
		),

		RabbitPublishTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rabbitmq_publish_total",
				Help:      "RabbitMQ messages published",
			},
			[]string{"routing_key", "result"},
		),

		ServiceUptime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "service_uptime_seconds",
				Help:      "Service uptime in seconds",
			},
		),
		// Goroutines: prometheus.NewGauge(
		//	prometheus.GaugeOpts{
		//		Namespace: namespace,
		//		Name:      "goroutines_current",
		//		Help:      "Current goroutine count",
		//	},
		// ),

		BusinessErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "business_errors_total",
				Help:      "Total business errors",
			},
			errorLabels,
		),

		TechnicalErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "technical_errors_total",
				Help:      "Total technical errors",
			},
			errorLabels,
		),
	}

	// register to default registry
	registry.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestsInFlight,
		m.HTTPRequestDuration,
		m.SubscriptionsCreated,
		m.SubscriptionsConfirmed,
		m.SubscriptionsCanceled,
		m.CronRuns,
		m.CronRunDuration,
		m.RabbitPublishTotal,
		m.ServiceUptime,
		// m.Goroutines,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewDBStatsCollector(db, dbName),
	)

	// gRPC server metrics
	grpcProm.EnableHandlingTimeHistogram()
	// grpcProm.Register(srv)

	// initialize uptime and goroutines
	m.ServiceUptime.SetToCurrentTime()
	// import runtime to get goroutine count
	// runtimeNumGoroutine := func() int {
	//	// avoid cyclical import of "runtime"
	//	return 0
	// }
	// m.Goroutines.Set(float64(runtimeNumGoroutine()))

	return m
}

// HTTPMiddleware instruments Gin HTTP handlers for RED metrics.
func (m *Metrics) HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		m.HTTPRequestsInFlight.Inc()
		c.Next()
		m.HTTPRequestsInFlight.Dec()

		dur := time.Since(start).Seconds()
		status := c.Writer.Status()
		statusClass := fmt.Sprintf("%dxx", status/divisor)

		m.HTTPRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), statusClass).Inc()
		m.HTTPRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(dur)
	}
}

// UnaryServerInterceptor returns a gRPC interceptor for server-side metrics.
func (m *Metrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return grpcProm.UnaryServerInterceptor
}

// StreamServerInterceptor returns a gRPC interceptor for server-side streaming metrics.
func (m *Metrics) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return grpcProm.StreamServerInterceptor
}

// CronJob wraps a function with cron metrics (runs + duration).
func (m *Metrics) CronJob(frequency string, job func()) {
	start := time.Now()
	m.CronRuns.WithLabelValues(frequency).Inc()
	job()
	m.CronRunDuration.WithLabelValues(frequency).Observe(time.Since(start).Seconds())
}

// RecordRabbitPublish logs a publish attempt (routing key) result ("ok" or "error").
func (m *Metrics) RecordRabbitPublish(routingKey string, err error) {
	result := "ok"
	if err != nil {
		result = "error"
	}
	m.RabbitPublishTotal.WithLabelValues(routingKey, result).Inc()
}
