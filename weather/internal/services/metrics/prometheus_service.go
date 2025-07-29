package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PromCollector struct {
	hist *prometheus.HistogramVec
	cnt  *prometheus.CounterVec
}

func NewPromCollector() *PromCollector {
	hist := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "weather_app",
			Name:      "cache_operation_duration_seconds",
			Help:      "Cache operation latencies",
		},
		[]string{"operation"},
	)
	cnt := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "weather_app",
			Name:      "cache_operations_total",
			Help:      "Cache operation counts",
		},
		[]string{"operation", "result"},
	)
	prometheus.MustRegister(hist, cnt)
	return &PromCollector{hist: hist, cnt: cnt}
}

func (p *PromCollector) ObserveLatency(op string, d time.Duration) {
	p.hist.WithLabelValues(op).Observe(d.Seconds())
}

func (p *PromCollector) IncrementCounter(metric string, labels ...string) {
	p.cnt.WithLabelValues(append([]string{metric}, labels...)...).Inc()
}
