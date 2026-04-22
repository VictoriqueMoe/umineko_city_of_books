package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var (
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds by method and route.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route", "status"},
	)
	httpRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being served.",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestDuration, httpRequestsInFlight)
}

func Metrics() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		path := ctx.Path()
		if isScrapeExempt(path) {
			return ctx.Next()
		}
		start := time.Now()
		httpRequestsInFlight.Inc()
		err := ctx.Next()
		httpRequestsInFlight.Dec()

		route := ctx.Route().Path
		if route == "" || route == "/*" {
			route = "other"
		}
		status := strconv.Itoa(ctx.Response().StatusCode())
		httpRequestDuration.
			WithLabelValues(ctx.Method(), route, status).
			Observe(time.Since(start).Seconds())
		return err
	}
}

func MetricsHandler() fiber.Handler {
	h := promhttp.Handler()
	return func(ctx fiber.Ctx) error {
		fasthttpadaptor.NewFastHTTPHandler(h)(ctx.RequestCtx())
		return nil
	}
}

func isScrapeExempt(path string) bool {
	return path == "/metrics" ||
		strings.HasPrefix(path, "/debug/pprof") ||
		strings.HasPrefix(path, "/static/assets/") ||
		strings.HasPrefix(path, "/assets/") ||
		strings.HasPrefix(path, "/uploads/")
}
