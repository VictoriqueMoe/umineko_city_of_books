package chatbot

import "github.com/prometheus/client_golang/prometheus"

var (
	invocationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chatbot_invocations_total",
			Help: "Chatbot invocations by final status.",
		},
		[]string{"status"},
	)

	droppedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chatbot_dropped_total",
			Help: "Chatbot triggers dropped before reaching the provider, by reason.",
		},
		[]string{"reason"},
	)

	tokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chatbot_tokens_total",
			Help: "Chatbot tokens consumed by kind.",
		},
		[]string{"kind"},
	)
)

func init() {
	prometheus.MustRegister(invocationsTotal, droppedTotal, tokensTotal)
}
