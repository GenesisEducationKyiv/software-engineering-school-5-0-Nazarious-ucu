package cache

import (
	"context"
	"time"
)

type cache[T any] interface {
	Set(ctx context.Context, key string, value T, expiration time.Duration) error
	Get(ctx context.Context, key string, returnValue *T) error
}

type metricsCollector interface {
	ObserveLatency(operation string, duration time.Duration)
	IncrementCounter(metric string, labels ...string)
}

type MetricsDecorator[T any] struct {
	next      cache[T]
	collector metricsCollector
}

func NewMetricsDecorator[T any](next cache[T], collector metricsCollector) *MetricsDecorator[T] {
	return &MetricsDecorator[T]{next: next, collector: collector}
}

func (m *MetricsDecorator[T]) Set(
	ctx context.Context,
	key string,
	value T,
	expiration time.Duration,
) error {
	start := time.Now()
	err := m.next.Set(ctx, key, value, expiration)
	dur := time.Since(start)
	m.collector.ObserveLatency("cache_set", dur)
	if err != nil {
		m.collector.IncrementCounter("cache_set_errors", key)
	} else {
		m.collector.IncrementCounter("cache_set_success", key)
	}
	return err
}

func (m *MetricsDecorator[T]) Get(
	ctx context.Context,
	key string,
	returnValue *T,
) error {
	start := time.Now()
	err := m.next.Get(ctx, key, returnValue)
	dur := time.Since(start)
	m.collector.ObserveLatency("cache_get", dur)
	if err != nil {
		m.collector.IncrementCounter("cache_get_misses", key)
	} else {
		m.collector.IncrementCounter("cache_get_hits", key)
	}
	return err
}
