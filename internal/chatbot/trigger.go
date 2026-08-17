package chatbot

import (
	"context"
	"sync"
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
		s.notify(ev, bot, outcome{reason: reasonNotPermitted, stage: stagePreTrigger})

		return
	}

	if !ev.IsDM && !s.takeCooldown(ev.SenderID, tune.cooldown) {
		droppedTotal.WithLabelValues("cooldown", string(stagePreTrigger), string(ev.channel())).Inc()

		return
	}

	key := ev.scopeKey()
	if _, busy := s.inScope.LoadOrStore(key, struct{}{}); busy {
		droppedTotal.WithLabelValues("room_inflight", string(stagePreTrigger), string(ev.channel())).Inc()

		return
	}

	select {
	case s.jobs <- job{ev: ev, bot: bot, useChain: useChain(ev, threaded)}:
		queueDepth.Set(float64(len(s.jobs)))
	default:
		s.inScope.Delete(key)
		droppedTotal.WithLabelValues("queue_full", string(stagePreTrigger), string(ev.channel())).Inc()
		silentTotal.WithLabelValues("queue_full", string(stagePreTrigger)).Inc()
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
	return takeSlot(&s.lastUse, userID, window)
}

func takeSlot(seen *sync.Map, key any, window time.Duration) bool {
	if window <= 0 {
		return true
	}

	now := time.Now()

	prev, ok := seen.Load(key)
	if ok {
		last, valid := prev.(time.Time)
		if valid && now.Sub(last) < window {
			return false
		}
	}

	seen.Store(key, now)

	return true
}

func (s *service) notify(ev botEvent, bot repository.Chatbot, out outcome) {
	ctx, cancel := context.WithTimeout(context.Background(), noticeTimeout)
	defer cancel()

	s.settle(ctx, job{ev: ev, bot: bot}, out)
}
