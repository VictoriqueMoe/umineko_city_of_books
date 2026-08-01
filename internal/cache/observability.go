package cache

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/valkey-io/valkey-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "umineko_city_of_books/internal/cache"

	statsTimeout = 2 * time.Second

	pipelineCommand = "pipeline"
)

type (
	observabilityHook struct {
		tracer trace.Tracer
	}

	statsCollector struct {
		manager *Manager

		up          *prometheus.Desc
		serverKeys  *prometheus.Desc
		memoryUsed  *prometheus.Desc
		memoryMax   *prometheus.Desc
		evictedKeys *prometheus.Desc

		connectedClients  *prometheus.Desc
		blockedClients    *prometheus.Desc
		connectionsTotal  *prometheus.Desc
		rejectedConns     *prometheus.Desc
		commandsProcessed *prometheus.Desc
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

func (h *observabilityHook) Do(client valkey.Client, ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
	name := commandName(cmd.Commands())

	ctx, span := h.tracer.Start(ctx, "valkey "+name, trace.WithSpanKind(trace.SpanKindClient))
	start := time.Now()

	resp := client.Do(ctx, cmd)

	observe(span, name, start, resp.Error())

	return resp
}

func (h *observabilityHook) DoMulti(client valkey.Client, ctx context.Context, multi ...valkey.Completed) []valkey.ValkeyResult {
	ctx, span := h.tracer.Start(ctx, "valkey "+pipelineCommand, trace.WithSpanKind(trace.SpanKindClient))
	start := time.Now()

	resps := client.DoMulti(ctx, multi...)

	observe(span, pipelineCommand, start, firstError(resps))

	return resps
}

func (h *observabilityHook) DoCache(client valkey.Client, ctx context.Context, cmd valkey.Cacheable, ttl time.Duration) valkey.ValkeyResult {
	name := commandName(cmd.Commands())

	ctx, span := h.tracer.Start(ctx, "valkey "+name, trace.WithSpanKind(trace.SpanKindClient))
	start := time.Now()

	resp := client.DoCache(ctx, cmd, ttl)

	observe(span, name, start, resp.Error())

	return resp
}

func (h *observabilityHook) DoMultiCache(client valkey.Client, ctx context.Context, multi ...valkey.CacheableTTL) []valkey.ValkeyResult {
	ctx, span := h.tracer.Start(ctx, "valkey "+pipelineCommand, trace.WithSpanKind(trace.SpanKindClient))
	start := time.Now()

	resps := client.DoMultiCache(ctx, multi...)

	observe(span, pipelineCommand, start, firstError(resps))

	return resps
}

func (h *observabilityHook) Receive(client valkey.Client, ctx context.Context, subscribe valkey.Completed, fn func(msg valkey.PubSubMessage)) error {
	return client.Receive(ctx, subscribe, fn)
}

func (h *observabilityHook) DoStream(client valkey.Client, ctx context.Context, cmd valkey.Completed) valkey.ValkeyResultStream {
	return client.DoStream(ctx, cmd)
}

func (h *observabilityHook) DoMultiStream(client valkey.Client, ctx context.Context, multi ...valkey.Completed) valkey.MultiValkeyResultStream {
	return client.DoMultiStream(ctx, multi...)
}

func commandName(parts []string) string {
	if len(parts) == 0 {
		return "unknown"
	}

	return parts[0]
}

func firstError(resps []valkey.ValkeyResult) error {
	for i := range resps {
		err := resps[i].Error()
		if err != nil && !valkey.IsValkeyNil(err) {
			return err
		}
	}

	return nil
}

func observe(span trace.Span, command string, start time.Time, err error) {
	commandDuration.WithLabelValues(command).Observe(time.Since(start).Seconds())

	if err != nil && !valkey.IsValkeyNil(err) {
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
		manager:     m,
		up:          prometheus.NewDesc("cache_up", "Whether the Valkey cache is configured and reachable (1) or not (0).", nil, nil),
		serverKeys:  prometheus.NewDesc("cache_server_keys", "Number of keys stored in the Valkey cache database.", nil, nil),
		memoryUsed:  prometheus.NewDesc("cache_server_memory_used_bytes", "Bytes of memory used by the Valkey server.", nil, nil),
		memoryMax:   prometheus.NewDesc("cache_server_memory_max_bytes", "Configured Valkey maxmemory limit in bytes (0 means unlimited).", nil, nil),
		evictedKeys: prometheus.NewDesc("cache_server_evicted_keys_total", "Number of keys evicted by the Valkey server due to maxmemory.", nil, nil),

		connectedClients:  prometheus.NewDesc("cache_server_connected_clients", "Number of client connections currently open on the Valkey server.", nil, nil),
		blockedClients:    prometheus.NewDesc("cache_server_blocked_clients", "Number of clients currently blocked on a blocking command.", nil, nil),
		connectionsTotal:  prometheus.NewDesc("cache_server_connections_received_total", "Number of connections the Valkey server has accepted since start.", nil, nil),
		rejectedConns:     prometheus.NewDesc("cache_server_rejected_connections_total", "Number of connections the Valkey server rejected because maxclients was reached.", nil, nil),
		commandsProcessed: prometheus.NewDesc("cache_server_commands_processed_total", "Number of commands the Valkey server has processed since start.", nil, nil),
	}
}

func (c *statsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.serverKeys
	ch <- c.memoryUsed
	ch <- c.memoryMax
	ch <- c.evictedKeys
	ch <- c.connectedClients
	ch <- c.blockedClients
	ch <- c.connectionsTotal
	ch <- c.rejectedConns
	ch <- c.commandsProcessed
}

func (c *statsCollector) Collect(ch chan<- prometheus.Metric) {
	client := c.manager.current()
	if client == nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), statsTimeout)
	defer cancel()

	size, err := client.Do(ctx, client.B().Dbsize().Build()).AsInt64()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)
	ch <- prometheus.MustNewConstMetric(c.serverKeys, prometheus.GaugeValue, float64(size))

	info, err := client.Do(ctx, client.B().Info().Section("memory", "stats", "clients").Build()).ToString()
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

	if value, ok := infoInt(info, "connected_clients"); ok {
		ch <- prometheus.MustNewConstMetric(c.connectedClients, prometheus.GaugeValue, value)
	}
	if value, ok := infoInt(info, "blocked_clients"); ok {
		ch <- prometheus.MustNewConstMetric(c.blockedClients, prometheus.GaugeValue, value)
	}
	if value, ok := infoInt(info, "total_connections_received"); ok {
		ch <- prometheus.MustNewConstMetric(c.connectionsTotal, prometheus.CounterValue, value)
	}
	if value, ok := infoInt(info, "rejected_connections"); ok {
		ch <- prometheus.MustNewConstMetric(c.rejectedConns, prometheus.CounterValue, value)
	}
	if value, ok := infoInt(info, "total_commands_processed"); ok {
		ch <- prometheus.MustNewConstMetric(c.commandsProcessed, prometheus.CounterValue, value)
	}
}

func infoInt(info, field string) (float64, bool) {
	prefix := field + ":"

	for line := range strings.SplitSeq(info, "\n") {
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
