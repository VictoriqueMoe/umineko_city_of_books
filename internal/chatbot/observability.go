package chatbot

import (
	"github.com/prometheus/client_golang/prometheus"

	"umineko_city_of_books/internal/model"
)

type (
	tokenKind string

	dropLabel struct {
		reason Reason
		stage  stage
	}
)

const (
	tokenPrompt       tokenKind = "prompt"
	tokenCachedPrompt tokenKind = "cached_prompt"
	tokenCacheWrite   tokenKind = "cache_write"
	tokenCompletion   tokenKind = "completion"
	tokenReasoning    tokenKind = "reasoning"

	noticeDelivered  = "delivered"
	noticeSuppressed = "suppressed"
	noticeFailed     = "failed"
)

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

var (
	seedChannels = []Channel{ChannelDM, ChannelGroup, ChannelPost, ChannelPostComment}

	seedStatuses = []model.InvocationStatus{
		model.InvocationReplied,
		model.InvocationRefused,
		model.InvocationFailed,
		model.InvocationQuota,
	}

	seedTokenKinds = []tokenKind{tokenPrompt, tokenCachedPrompt, tokenCacheWrite, tokenCompletion, tokenReasoning}

	seedNoticeResults = []string{noticeDelivered, noticeSuppressed, noticeFailed}

	seedDrops = []dropLabel{
		{reasonNotPermitted, stagePreTrigger},
		{reasonCooldown, stagePreTrigger},
		{reasonRoomInflight, stagePreTrigger},
		{reasonQueueFull, stagePreTrigger},
		{reasonInternal, stagePreModel},
		{reasonQuotaUser, stagePreModel},
		{reasonQuotaSite, stagePreModel},
		{reasonProviderLimited, stagePostModel},
		{reasonNotConfigured, stagePostModel},
		{reasonTimeout, stagePostModel},
		{reasonProviderDown, stagePostModel},
		{reasonEmptyReply, stagePostModel},
		{reasonFiltered, stagePostModel},
		{reasonUndeliverable, stagePostModel},
	}
)

func init() {
	prometheus.MustRegister(invocationsTotal, droppedTotal, tokensTotal, noticesTotal, silentTotal, queueDepth)

	seedMetrics()
}

func seedMetrics() {
	for _, channel := range seedChannels {
		for _, status := range seedStatuses {
			invocationsTotal.WithLabelValues(string(status), string(channel)).Add(0)
		}

		for _, kind := range seedTokenKinds {
			tokensTotal.WithLabelValues(string(kind), string(channel)).Add(0)
		}

		for _, drop := range seedDrops {
			droppedTotal.WithLabelValues(string(drop.reason), string(drop.stage), string(channel)).Add(0)
		}
	}

	for _, drop := range seedDrops {
		silentTotal.WithLabelValues(string(drop.reason), string(drop.stage)).Add(0)

		for _, result := range seedNoticeResults {
			noticesTotal.WithLabelValues(string(drop.reason), result).Add(0)
		}
	}
}
