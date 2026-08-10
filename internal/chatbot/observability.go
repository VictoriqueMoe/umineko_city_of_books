package chatbot

import "github.com/prometheus/client_golang/prometheus"

var (
	invocationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chatbot_invocations_total",
			Help: "Chatbot invocations by final status and the channel the summon came from.",
		},
		[]string{"status", "channel"},
	)

	droppedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chatbot_dropped_total",
			Help: "Chatbot summons that produced no answer, by reason, pipeline stage and channel.",
		},
		[]string{"reason", "stage", "channel"},
	)

	noticesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chatbot_notices_total",
			Help: "Explanations the chatbot tried to deliver in place of an answer, by reason and result.",
		},
		[]string{"reason", "result"},
	)

	silentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chatbot_silent_total",
			Help: "Summons that produced no user-visible output at all, not even an explanation. Should stay at zero.",
		},
		[]string{"reason", "stage"},
	)

	queueDepth = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "chatbot_queue_depth",
			Help: "Chatbot jobs waiting for a worker.",
		},
	)

	tokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chatbot_tokens_total",
			Help: "Chatbot tokens consumed by kind and the channel the summon came from.",
		},
		[]string{"kind", "channel"},
	)
)

func init() {
	prometheus.MustRegister(invocationsTotal, droppedTotal, tokensTotal, noticesTotal, silentTotal, queueDepth)
}
