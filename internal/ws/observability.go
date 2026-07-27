package ws

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	wsInboundMessages = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_inbound_messages_total",
			Help: "Number of WebSocket frames received from clients by message type.",
		},
		[]string{"type"},
	)
	wsInboundDropped = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_inbound_dropped_total",
			Help: "Number of inbound WebSocket frames dropped by the per-connection rate limiter.",
		},
		[]string{"authed"},
	)
	wsConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ws_connections",
			Help: "Number of WebSocket connections currently open by hub.",
		},
		[]string{"hub", "authed"},
	)
	wsConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_connections_total",
			Help: "Number of WebSocket connections opened by hub.",
		},
		[]string{"hub", "authed"},
	)
	wsInboundTokens = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ws_inbound_limiter_tokens",
			Help:    "Rate limiter tokens remaining when an inbound frame was received.",
			Buckets: []float64{0, 1, 5, 10, 25, 50, 100, 150, 200},
		},
	)
)

func init() {
	prometheus.MustRegister(wsInboundMessages, wsInboundDropped, wsConnections, wsConnectionsTotal, wsInboundTokens)
}

func recordInbound(msgType string, tokens float64) {
	wsInboundMessages.WithLabelValues(msgType).Inc()
	wsInboundTokens.Observe(tokens)
}

func recordDropped(authed bool) {
	wsInboundDropped.WithLabelValues(authedLabel(authed)).Inc()
}

func recordConnectionOpened(hub string, authed bool) {
	label := authedLabel(authed)
	wsConnections.WithLabelValues(hub, label).Inc()
	wsConnectionsTotal.WithLabelValues(hub, label).Inc()
}

func recordConnectionClosed(hub string, authed bool) {
	wsConnections.WithLabelValues(hub, authedLabel(authed)).Dec()
}

func authedLabel(authed bool) string {
	if authed {
		return "true"
	}

	return "false"
}
