package telemetry

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	mu               sync.Mutex
	tp               *sdktrace.TracerProvider
	currentProcessor sdktrace.SpanProcessor
	currentDSN       string
)

func Init(ctx context.Context, serviceName, version, endpoint string) error {
	mu.Lock()
	defer mu.Unlock()

	if tp != nil {
		return nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		),
		resource.WithHost(),
		resource.WithProcessPID(),
	)
	if err != nil {
		return fmt.Errorf("otel resource: %w", err)
	}

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return applyLocked(ctx, endpoint)
}

func Apply(endpoint string) error {
	mu.Lock()
	defer mu.Unlock()
	if tp == nil {
		return nil
	}
	return applyLocked(context.Background(), endpoint)
}

func applyLocked(ctx context.Context, endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == currentDSN {
		return nil
	}

	if currentProcessor != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = currentProcessor.Shutdown(shutdownCtx)
		cancel()
		tp.UnregisterSpanProcessor(currentProcessor)
		currentProcessor = nil
	}

	currentDSN = endpoint

	if endpoint == "" {
		logger.Log.Info().Msg("otel tracing disabled")
		return nil
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(endpoint + "/v1/traces")}
	if strings.HasPrefix(endpoint, "http://") {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return fmt.Errorf("otlp exporter: %w", err)
	}

	processor := sdktrace.NewBatchSpanProcessor(exporter, sdktrace.WithBatchTimeout(5*time.Second))
	tp.RegisterSpanProcessor(processor)
	currentProcessor = processor

	logger.Log.Info().Str("endpoint", endpoint).Msg("otel tracing enabled")
	return nil
}

func Shutdown() {
	mu.Lock()
	defer mu.Unlock()
	if tp == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = tp.Shutdown(ctx)
	tp = nil
	currentProcessor = nil
}

type SettingsListener struct{}

func NewSettingsListener() *SettingsListener {
	return &SettingsListener{}
}

func (l *SettingsListener) OnSettingChanged(key config.SiteSettingKey, value string) {
	if key != config.SettingOTLPEndpoint.Key {
		return
	}
	if err := Apply(value); err != nil {
		logger.Log.Warn().Err(err).Msg("failed to apply otel endpoint change")
	}
}
