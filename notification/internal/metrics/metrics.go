package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Metrics holds Prometheus collectors for the notification service.
type Metrics struct {
	ConsumerMessagesTotal      *prometheus.CounterVec
	ConsumerProcessingDuration *prometheus.HistogramVec
	ConsumerErrorsTotal        *prometheus.CounterVec

	EmailSentTotal *prometheus.CounterVec

	EmailErrorsTotal *prometheus.CounterVec

	ServiceUptime prometheus.Gauge
}

// NewMetrics constructs and registers all metrics under the given service namespace.
func NewMetrics(serviceName string) *Metrics {
	prog := prometheus.NewRegistry()
	m := &Metrics{
		ConsumerMessagesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "consumer_messages_total",
				Help:      "Total number of RabbitMQ messages consumed",
			},
			[]string{"event_type"},
		),
		ConsumerProcessingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: serviceName,
				Name:      "consumer_processing_duration_seconds",
				Help:      "Histogram of message processing durations",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"event_type"},
		),
		ConsumerErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "consumer_errors_total",
				Help:      "Total number of errors while processing messages",
			},
			[]string{"event_type", "error_type"},
		),
		EmailSentTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "email_sent_total",
				Help:      "Total number of emails sent",
			},
			[]string{"event_type"},
		),
		EmailErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "email_errors_total",
				Help:      "Total number of email send failures",
			},
			[]string{"event_type", "error_type"},
		),
		ServiceUptime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: serviceName,
				Name:      "service_uptime_timestamp",
				Help:      "Current UNIX timestamp in seconds (service uptime)",
			},
		),
	}

	// Register everything, plus Go & process collectors
	prog.MustRegister(
		m.ConsumerMessagesTotal,
		m.ConsumerProcessingDuration,
		m.ConsumerErrorsTotal,
		m.EmailSentTotal,
		m.EmailErrorsTotal,
		m.ServiceUptime,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// Initialize uptime to now
	m.ServiceUptime.Set(float64(time.Now().Unix()))

	return m
}
