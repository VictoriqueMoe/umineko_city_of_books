package middleware

import (
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "umineko_city_of_books"

func Tracing() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		tracer := otel.Tracer(tracerName)
		propagator := otel.GetTextMapPropagator()
		carrier := fiberHeaderCarrier{ctx: ctx}
		parentCtx := propagator.Extract(ctx.Context(), carrier)

		path := ctx.Path()
		method := ctx.Method()

		spanCtx, span := tracer.Start(parentCtx, method+" "+path,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(method),
				semconv.URLPath(path),
				semconv.URLScheme(ctx.Protocol()),
				semconv.UserAgentOriginal(string(ctx.Request().Header.UserAgent())),
			),
		)
		defer span.End()

		ctx.SetContext(spanCtx)

		sc := span.SpanContext()
		if sc.IsValid() {
			ctx.Locals("trace_id", sc.TraceID().String())
			ctx.Locals("span_id", sc.SpanID().String())
			ctx.Set("X-Trace-ID", sc.TraceID().String())
		}

		err := ctx.Next()

		status := ctx.Response().StatusCode()
		span.SetAttributes(semconv.HTTPResponseStatusCode(status))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if status >= 500 {
			span.SetStatus(codes.Error, "server error")
		}
		return err
	}
}

type fiberHeaderCarrier struct {
	ctx fiber.Ctx
}

func (c fiberHeaderCarrier) Get(key string) string {
	return string(c.ctx.Request().Header.Peek(key))
}

func (c fiberHeaderCarrier) Set(key, value string) {
	c.ctx.Request().Header.Set(key, value)
}

func (c fiberHeaderCarrier) Keys() []string {
	keys := make([]string, 0)
	c.ctx.Request().Header.VisitAll(func(k, _ []byte) {
		keys = append(keys, string(k))
	})
	return keys
}

var _ propagation.TextMapCarrier = fiberHeaderCarrier{}
