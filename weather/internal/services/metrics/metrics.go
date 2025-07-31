package metrics

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	grpc_prom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"google.golang.org/grpc"
)

const divisor = 100

// Metrics holds Prometheus metric vectors for the weather service.
type Metrics struct {
	// HTTP server metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// gRPC server metrics (using grpc_prometheus)

	// Domain metrics
	WeatherRequestsTotal *prometheus.CounterVec
	WeatherErrorsTotal   *prometheus.CounterVec
}

// NewMetrics constructs and registers all weather-service metrics.
func NewMetrics(serviceName string) *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "http_requests_total",
				Help:      "Total HTTP requests received",
			},
			[]string{"method", "endpoint", "status_class"},
		),

		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: serviceName,
				Name:      "http_request_duration_seconds",
				Help:      "Histogram of HTTP request latencies",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		WeatherRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "weather_requests_total",
				Help:      "Total number of weather data requests",
			},
			[]string{"city"},
		),

		WeatherErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "weather_errors_total",
				Help:      "Total number of weather data errors",
			},
			[]string{"city", "error_type"},
		),

		// ServiceUptime: prometheus.NewGauge(
		//	prometheus.GaugeOpts{
		//		Namespace: serviceName,
		//		Name:      "service_uptime_seconds",
		//		Help:      "Service uptime in seconds",
		//	},
		// ),
	}

	// register
	reg.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.WeatherRequestsTotal,
		m.WeatherErrorsTotal,
		// m.ServiceUptime,
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(
				collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile("/sched/latencies:seconds")},
			),
		),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// enable grpc handling time histograms
	grpc_prom.EnableHandlingTimeHistogram()

	return m
}

// HTTPMiddleware returns a Gin middleware to instrument HTTP endpoints.
func (m *Metrics) HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		d := time.Since(start)
		// m.ServiceUptime.SetToCurrentTime()

		status := c.Writer.Status()
		statusClass := getStatusClass(status)

		labels := prometheus.Labels{
			"method":       c.Request.Method,
			"endpoint":     c.FullPath(),
			"status_class": statusClass,
		}
		m.HTTPRequestsTotal.With(labels).Inc()
		m.HTTPRequestDuration.With(prometheus.Labels{
			"method":   c.Request.Method,
			"endpoint": c.FullPath(),
		}).Observe(d.Seconds())

		// domain metrics
		city := c.Query("city")
		m.WeatherRequestsTotal.WithLabelValues(city).Inc()
		if statusClass == "5xx" {
			m.WeatherErrorsTotal.WithLabelValues(city, "server_error").Inc()
		}
		if statusClass == "4xx" {
			m.WeatherErrorsTotal.WithLabelValues(city, "client_error").Inc()
		}
	}
}

// UnaryInterceptor returns a gRPC UnaryServerInterceptor for metrics.
func (m *Metrics) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return grpc_prom.UnaryServerInterceptor
}

// StreamInterceptor returns a gRPC StreamServerInterceptor for metrics.
func (m *Metrics) StreamInterceptor() grpc.StreamServerInterceptor {
	return grpc_prom.StreamServerInterceptor
}

func getStatusClass(code int) string {
	return fmt.Sprintf("%dxx", code/divisor)
}
