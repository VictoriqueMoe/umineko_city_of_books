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

type (
	claimResult int
)

const (
	claimGranted claimResult = iota
	claimRefused
	claimUnchecked
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

	lease := s.claim(ctx, purposeScopeLease, ev.scopeKey(), ev.ItemID.String(), scopeLeaseTTL)
	if lease == claimRefused {
		droppedTotal.WithLabelValues(string(reasonRoomInflight), string(stagePreTrigger), string(ev.channel())).Inc()

		return
	}

	j := job{ev: ev, bot: bot, useChain: useChain(ev, threaded), leased: lease == claimGranted}

	select {
	case s.jobs <- j:
		queueDepth.Set(float64(len(s.jobs)))
	default:
		s.releaseScope(ctx, j)
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

	return s.claim(ctx, purposeSummonCooldown, key, time.Now().UTC().Format(time.RFC3339Nano), window) != claimRefused
}

func (s *service) takeNoticeSlot(ctx context.Context, userID uuid.UUID, reason Reason) bool {
	key := cache.ChatbotRefusalNotice.Key(userID.String(), string(reason))

	return s.claim(ctx, purposeRefusalNotice, key, time.Now().UTC().Format(time.RFC3339Nano), cache.ChatbotRefusalNotice.TTL) != claimRefused
}

func (s *service) claim(ctx context.Context, purpose, key, value string, ttl time.Duration) claimResult {
	claimed, err := s.cache.SetIfAbsent(ctx, key, value, ttl)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("purpose", purpose).Msg("chatbot could not reach the coordination store, letting this one through")

		return claimUnchecked
	}

	if !claimed {
		return claimRefused
	}

	return claimGranted
}

func (s *service) releaseScope(ctx context.Context, j job) {
	if !j.leased {
		return
	}

	if err := s.cache.Del(ctx, j.ev.scopeKey()); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("purpose", purposeScopeLease).Msg("chatbot could not release a scope lease, the scope stays busy until the lease expires")
	}
}

func (s *service) notify(ev botEvent, bot model.Chatbot, out outcome) {
	ctx, cancel := context.WithTimeout(context.Background(), noticeTimeout)
	defer cancel()

	s.settle(ctx, job{ev: ev, bot: bot}, out)
}
