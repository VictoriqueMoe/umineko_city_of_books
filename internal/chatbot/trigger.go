package chatbot

import (
	"context"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"

	"github.com/google/uuid"
)

func (s *service) observe(ev botEvent) {
	tune, bots := s.snapshot()
	if !tune.enabled || len(bots) == 0 || s.closing.Load() {
		return
	}

	bot, threaded, ok := selectBot(ev, bots)
	if !ok {
		return
	}

	if !s.allowedToSummon(ev.SenderID, tune) {
		s.notify(ev, bot, outcome{reason: reasonNotPermitted, stage: stagePreTrigger})

		return
	}

	ctx := context.Background()

	if !s.takeCooldown(ctx, ev.SenderID, tune.cooldown) {
		droppedTotal.WithLabelValues(string(reasonCooldown), string(stagePreTrigger), string(ev.channel())).Inc()

		return
	}

	select {
	case s.jobs <- job{ev: ev, bot: bot, useChain: useChain(ev, threaded)}:
		queueDepth.Set(float64(len(s.jobs)))
	default:
		droppedTotal.WithLabelValues(string(reasonQueueFull), string(stagePreTrigger), string(ev.channel())).Inc()
		silentTotal.WithLabelValues(string(reasonQueueFull), string(stagePreTrigger)).Inc()
	}
}

func useChain(ev botEvent, threaded bool) bool {
	if ev.ParentID == nil {
		return false
	}

	if ev.Surface == SurfaceChat {
		return true
	}

	return threaded
}

func selectBot(ev botEvent, bots map[uuid.UUID]model.Chatbot) (model.Chatbot, bool, bool) {
	if ev.IsDM {
		for _, memberID := range ev.Audience {
			if bot, ok := bots[memberID]; ok && memberID != ev.SenderID {
				return bot, false, true
			}
		}

		return model.Chatbot{}, false, false
	}

	if ev.ParentAuthor != uuid.Nil {
		if bot, ok := bots[ev.ParentAuthor]; ok {
			return bot, true, true
		}
	}

	for mentionedID := range ev.MentionedIDs {
		if bot, ok := bots[mentionedID]; ok {
			return bot, false, true
		}
	}

	return model.Chatbot{}, false, false
}

func (s *service) allowedToSummon(userID uuid.UUID, tune tuning) bool {
	if !tune.requirePermission {
		return true
	}

	return s.authzSvc.Can(context.Background(), userID, authz.PermUseChatbot)
}

func (s *service) takeCooldown(ctx context.Context, userID uuid.UUID, window time.Duration) bool {
	if window <= 0 {
		return true
	}

	key := cache.ChatbotSummonCooldown.Key(userID.String())

	return s.claim(ctx, purposeSummonCooldown, key, window)
}

func (s *service) takeNoticeSlot(ctx context.Context, userID uuid.UUID, reason Reason) bool {
	key := cache.ChatbotRefusalNotice.Key(userID.String(), string(reason))

	return s.claim(ctx, purposeRefusalNotice, key, cache.ChatbotRefusalNotice.TTL)
}

func (s *service) claim(ctx context.Context, purpose string, key cache.Key, window time.Duration) bool {
	allowance, err := s.cache.Allow(ctx, key, 1, window)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("purpose", purpose).Msg("chatbot could not reach the coordination store, letting this one through")

		return true
	}

	return allowance.Allowed
}

func (s *service) notify(ev botEvent, bot model.Chatbot, out outcome) {
	ctx, cancel := context.WithTimeout(context.Background(), noticeTimeout)
	defer cancel()

	s.settle(ctx, job{ev: ev, bot: bot}, out)
}
