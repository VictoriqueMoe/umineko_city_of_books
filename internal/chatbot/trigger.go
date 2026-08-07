package chatbot

import (
	"context"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/repository"

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
		droppedTotal.WithLabelValues("permission").Inc()

		return
	}

	if !ev.IsDM && !s.takeCooldown(ev.SenderID, tune.cooldown) {
		droppedTotal.WithLabelValues("cooldown").Inc()

		return
	}

	key := ev.scopeKey()
	if _, busy := s.inScope.LoadOrStore(key, struct{}{}); busy {
		droppedTotal.WithLabelValues("room_inflight").Inc()

		return
	}

	select {
	case s.jobs <- job{ev: ev, bot: bot, useChain: useChain(ev, threaded)}:
	default:
		s.inScope.Delete(key)
		droppedTotal.WithLabelValues("queue_full").Inc()
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

func selectBot(ev botEvent, bots map[uuid.UUID]repository.Chatbot) (repository.Chatbot, bool, bool) {
	if ev.IsDM {
		for _, memberID := range ev.Audience {
			if bot, ok := bots[memberID]; ok && memberID != ev.SenderID {
				return bot, false, true
			}
		}

		return repository.Chatbot{}, false, false
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

	return repository.Chatbot{}, false, false
}

func (s *service) allowedToSummon(userID uuid.UUID, tune tuning) bool {
	if !tune.requirePermission {
		return true
	}

	return s.authzSvc.Can(context.Background(), userID, authz.PermUseChatbot)
}

func (s *service) takeCooldown(userID uuid.UUID, window time.Duration) bool {
	if window <= 0 {
		return true
	}

	now := time.Now()

	prev, ok := s.lastUse.Load(userID)
	if ok {
		last, valid := prev.(time.Time)
		if valid && now.Sub(last) < window {
			return false
		}
	}

	s.lastUse.Store(userID, now)

	return true
}
