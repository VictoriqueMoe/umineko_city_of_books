package cache

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "umineko_city_of_books/internal/cache"

	statsTimeout = 2 * time.Second
)

type (
	observabilityHook struct {
		tracer trace.Tracer
	}

	statsCollector struct {
		manager *Manager

		up           *prometheus.Desc
		totalConns   *prometheus.Desc
		idleConns    *prometheus.Desc
		staleConns   *prometheus.Desc
		poolHits     *prometheus.Desc
		poolMisses   *prometheus.Desc
		poolTimeouts *prometheus.Desc
		serverKeys   *prometheus.Desc
		memoryUsed   *prometheus.Desc
		memoryMax    *prometheus.Desc
		evictedKeys  *prometheus.Desc
	}
)

var (
	cacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Number of Valkey cache lookups that returned a value.",
	})
	cacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Number of Valkey cache lookups that found no value.",
	})
	commandDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_command_duration_seconds",
			Help:    "Valkey command duration in seconds by command.",
			Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"command"},
	)
	commandErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_command_errors_total",
			Help: "Number of Valkey commands that failed by command.",
		},
		[]string{"command"},
	)
)

func init() {
	prometheus.MustRegister(cacheHits, cacheMisses, commandDuration, commandErrors)
}

func newObservabilityHook() *observabilityHook {
	return &observabilityHook{tracer: otel.Tracer(tracerName)}
}

func (h *observabilityHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *observabilityHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		name := cmd.Name()

		ctx, span := h.tracer.Start(ctx, "valkey "+name, trace.WithSpanKind(trace.SpanKindClient))
		start := time.Now()

		err := next(ctx, cmd)

		observe(span, name, start, err)

		return err
	}
}

func (h *observabilityHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		ctx, span := h.tracer.Start(ctx, "valkey pipeline", trace.WithSpanKind(trace.SpanKindClient))
		start := time.Now()

		err := next(ctx, cmds)

		observe(span, "pipeline", start, err)

		return err
	}
}

func observe(span trace.Span, command string, start time.Time, err error) {
	commandDuration.WithLabelValues(command).Observe(time.Since(start).Seconds())

	if err != nil && !errors.Is(err, redis.Nil) {
		commandErrors.WithLabelValues(command).Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}

func registerStatsCollector(m *Manager) {
	_ = prometheus.Register(newStatsCollector(m))
}

func newStatsCollector(m *Manager) *statsCollector {
	return &statsCollector{
		manager:      m,
		up:           prometheus.NewDesc("cache_up", "Whether the Valkey cache is configured and reachable (1) or not (0).", nil, nil),
		totalConns:   prometheus.NewDesc("cache_pool_total_connections", "Total number of connections in the Valkey pool.", nil, nil),
		idleConns:    prometheus.NewDesc("cache_pool_idle_connections", "Number of idle connections in the Valkey pool.", nil, nil),
		staleConns:   prometheus.NewDesc("cache_pool_stale_connections_total", "Number of stale connections removed from the Valkey pool.", nil, nil),
		poolHits:     prometheus.NewDesc("cache_pool_hits_total", "Number of times a free connection was found in the Valkey pool.", nil, nil),
		poolMisses:   prometheus.NewDesc("cache_pool_misses_total", "Number of times a free connection was not found in the Valkey pool.", nil, nil),
		poolTimeouts: prometheus.NewDesc("cache_pool_timeouts_total", "Number of times a wait for a Valkey connection timed out.", nil, nil),
		serverKeys:   prometheus.NewDesc("cache_server_keys", "Number of keys stored in the Valkey cache database.", nil, nil),
		memoryUsed:   prometheus.NewDesc("cache_server_memory_used_bytes", "Bytes of memory used by the Valkey server.", nil, nil),
		memoryMax:    prometheus.NewDesc("cache_server_memory_max_bytes", "Configured Valkey maxmemory limit in bytes (0 means unlimited).", nil, nil),
		evictedKeys:  prometheus.NewDesc("cache_server_evicted_keys_total", "Number of keys evicted by the Valkey server due to maxmemory.", nil, nil),
	}
}

func (c *statsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.totalConns
	ch <- c.idleConns
	ch <- c.staleConns
	ch <- c.poolHits
	ch <- c.poolMisses
	ch <- c.poolTimeouts
	ch <- c.serverKeys
	ch <- c.memoryUsed
	ch <- c.memoryMax
	ch <- c.evictedKeys
}

func (c *statsCollector) Collect(ch chan<- prometheus.Metric) {
	client := c.manager.current()
	if client == nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}

	stats := client.PoolStats()
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(stats.TotalConns))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(stats.IdleConns))
	ch <- prometheus.MustNewConstMetric(c.staleConns, prometheus.CounterValue, float64(stats.StaleConns))
	ch <- prometheus.MustNewConstMetric(c.poolHits, prometheus.CounterValue, float64(stats.Hits))
	ch <- prometheus.MustNewConstMetric(c.poolMisses, prometheus.CounterValue, float64(stats.Misses))
	ch <- prometheus.MustNewConstMetric(c.poolTimeouts, prometheus.CounterValue, float64(stats.Timeouts))

	ctx, cancel := context.WithTimeout(context.Background(), statsTimeout)
	defer cancel()

	size, err := client.DBSize(ctx).Result()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)
	ch <- prometheus.MustNewConstMetric(c.serverKeys, prometheus.GaugeValue, float64(size))

	info, err := client.Info(ctx, "memory", "stats").Result()
	if err != nil {
		return
	}

	if value, ok := infoInt(info, "used_memory"); ok {
		ch <- prometheus.MustNewConstMetric(c.memoryUsed, prometheus.GaugeValue, value)
	}
	if value, ok := infoInt(info, "maxmemory"); ok {
		ch <- prometheus.MustNewConstMetric(c.memoryMax, prometheus.GaugeValue, value)
	}
	if value, ok := infoInt(info, "evicted_keys"); ok {
		ch <- prometheus.MustNewConstMetric(c.evictedKeys, prometheus.CounterValue, value)
	}
}

func infoInt(info, field string) (float64, bool) {
	prefix := field + ":"

	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		value, err := strconv.ParseFloat(strings.TrimPrefix(line, prefix), 64)
		if err != nil {
			return 0, false
		}

		return value, true
	}

	return 0, false
}
