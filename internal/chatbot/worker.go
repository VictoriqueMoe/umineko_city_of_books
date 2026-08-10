package chatbot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.quit:
			return
		default:
		}

		select {
		case <-s.quit:
			return
		case j := <-s.jobs:
			s.run(id, j)
		}
	}
}

func (s *service) run(id int, j job) {
	defer s.inScope.Delete(j.ev.scopeKey())

	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	if j.notice.reason != reasonNone {
		s.settle(ctx, j, j.notice)

		return
	}

	tune, _ := s.snapshot()

	invocationID := uuid.New()
	model := firstNonBlank(j.bot.Model, tune.model)

	out := s.reply(ctx, j, tune, invocationID, model)

	invocationsTotal.WithLabelValues(string(out.status)).Inc()

	if out.reason == reasonNone {
		return
	}

	logOutcome(id, j, out)
	s.settle(ctx, j, out)
}

func logOutcome(id int, j job, out outcome) {
	entry := logger.Log.Error().
		Int("worker", id).
		Str("bot", j.bot.Username).
		Str("surface", string(j.ev.Surface)).
		Str("reason", string(out.reason)).
		Str("stage", string(out.stage))

	if out.detail != "" {
		entry = entry.Str("detail", out.detail)
	}

	if out.err != nil {
		entry = entry.Err(out.err)
	}

	entry.Msg("chatbot could not answer")
}

func (s *service) settle(ctx context.Context, j job, out outcome) {
	droppedTotal.WithLabelValues(string(out.reason), string(out.stage), string(j.ev.Surface)).Inc()

	if out.reason.policy() && !takeSlot(&s.lastNotice, noticeKey{user: j.ev.SenderID, reason: out.reason}, refusalCooldown) {
		noticesTotal.WithLabelValues(string(out.reason), "suppressed").Inc()

		return
	}

	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), noticeTimeout)
	defer cancel()

	if err := s.deliver(sendCtx, j, s.noticeText(sendCtx, out)); err != nil {
		noticesTotal.WithLabelValues(string(out.reason), "failed").Inc()
		silentTotal.WithLabelValues(string(out.reason), string(out.stage)).Inc()

		logger.Log.Error().Err(err).
			Str("bot", j.bot.Username).
			Str("reason", string(out.reason)).
			Msg("chatbot could not deliver its explanation, the member was left with nothing")

		return
	}

	noticesTotal.WithLabelValues(string(out.reason), "delivered").Inc()
}

func (s *service) reply(ctx context.Context, j job, tune tuning, invocationID uuid.UUID, model string) outcome {
	quota, err := s.overQuota(ctx, j.ev.SenderID, tune)
	if err != nil {
		return outcome{reason: reasonInternal, stage: stagePreModel, status: repository.InvocationFailed, err: err}
	}
	if quota.over {
		reason := reasonQuotaUser
		if quota.global {
			reason = reasonQuotaSite
		}

		return outcome{reason: reason, stage: stagePreModel, status: repository.InvocationQuota, clearsAt: quota.clearsAt}
	}

	var roomID *uuid.UUID
	if j.ev.Surface == SurfaceChat {
		roomID = new(j.ev.ScopeID)
	}

	if err := s.botRepo.CreateInvocation(ctx, invocationID, j.bot.UserID, j.ev.SenderID, roomID, j.ev.ItemID, string(j.ev.Surface), model); err != nil {
		return outcome{reason: reasonInternal, stage: stagePreModel, status: repository.InvocationFailed, err: err}
	}

	stopTyping := s.startTyping(j.ev, j.bot.UserID)
	defer stopTyping()

	req := openai.CompletionRequest{
		Model:           model,
		SystemPrompt:    systemPrompt(j.bot.SystemPrompt),
		Messages:        s.buildMessages(ctx, j, tune),
		ReasoningEffort: firstNonBlank(j.bot.ReasoningEffort, tune.reasoningEffort),
		Verbosity:       firstNonBlank(j.bot.Verbosity, tune.verbosity),
		MaxOutputTokens: firstPositive(j.bot.MaxOutputTokens, tune.maxOutputTokens),
		CacheKey:        j.bot.UserID.String(),
		SafetyID:        safetyIdentifier(j.ev.SenderID),
	}

	result, err := s.openaiSvc.Complete(ctx, req)
	if err != nil {
		_ = s.botRepo.CompleteInvocation(ctx, invocationID, repository.InvocationUsage{}, repository.InvocationFailed)
		stopTyping()

		return classifyProvider(ctx, err)
	}

	recordTokens(result)

	body := strings.TrimSpace(result.Text)
	if body == "" {
		_ = s.botRepo.CompleteInvocation(ctx, invocationID, usageOf(result), repository.InvocationRefused)
		stopTyping()

		return outcome{
			reason: reasonEmptyReply,
			stage:  stagePostModel,
			status: repository.InvocationRefused,
			detail: result.IncompleteReason,
		}
	}

	if result.Incomplete {
		logger.Log.Warn().
			Str("bot", j.bot.Username).
			Str("reason", firstNonBlank(result.IncompleteReason, "unknown")).
			Int("completion_tokens", result.CompletionTokens).
			Int("reasoning_tokens", result.ReasoningTokens).
			Msg("chatbot reply was cut short, delivering what it produced")
	}

	stopTyping()

	if sendErr := s.deliver(ctx, j, body); sendErr != nil {
		_ = s.botRepo.CompleteInvocation(ctx, invocationID, usageOf(result), repository.InvocationRefused)

		return classifyDelivery(sendErr)
	}

	if err := s.botRepo.CompleteInvocation(ctx, invocationID, usageOf(result), repository.InvocationReplied); err != nil {
		logger.Log.Error().Err(err).Str("bot", j.bot.Username).Msg("chatbot answered but the invocation could not be closed")
	}

	return outcome{status: repository.InvocationReplied}
}

func emptyReplyMessage(reason string) string {
	switch reason {
	case openai.IncompleteContentFilter:
		return "I began an answer to that and thought better of it. Try asking me another way."
	case openai.IncompleteMaxOutputTokens:
		return "I ran out of room before I got a word out. Ask me something narrower and I will manage it."
	default:
		return "I have nothing to say to that, which is unlike me. Try me again."
	}
}

func trimBase(raw string) string {
	return strings.TrimSuffix(strings.TrimSpace(raw), "/")
}

func (s *service) deliver(ctx context.Context, j job, body string) error {
	if j.ev.Surface.gameBoard() {
		var parentID *uuid.UUID
		if j.ev.Surface == SurfacePostComment {
			parentID = new(j.ev.ItemID)
		}

		_, err := s.postSvc.CreateComment(ctx, j.ev.ScopeID, j.bot.UserID, dto.CreateCommentRequest{Body: body, ParentID: parentID})

		return err
	}

	replyTo := new(j.ev.ItemID)
	_, err := s.chatSvc.SendMessage(ctx, j.bot.UserID, j.ev.ScopeID, dto.SendMessageRequest{Body: body, ReplyToID: replyTo}, nil)

	return err
}

func (s *service) overQuota(ctx context.Context, userID uuid.UUID, tune tuning) (quotaState, error) {
	if tune.perUserPerDay > 0 {
		used, err := s.botRepo.CountUserInvocationsToday(ctx, userID)
		if err != nil {
			return quotaState{}, err
		}
		if used >= tune.perUserPerDay {
			oldest, oldErr := s.botRepo.OldestUserInvocationToday(ctx, userID)
			if oldErr != nil {
				logger.Log.Warn().Err(oldErr).Msg("chatbot: could not work out when the member quota frees up")
			}

			return quotaState{over: true, clearsAt: quotaClearsAt(oldest)}, nil
		}
	}

	if tune.perDay > 0 {
		used, err := s.botRepo.CountInvocationsToday(ctx)
		if err != nil {
			return quotaState{}, err
		}
		if used >= tune.perDay {
			oldest, oldErr := s.botRepo.OldestInvocationToday(ctx)
			if oldErr != nil {
				logger.Log.Warn().Err(oldErr).Msg("chatbot: could not work out when the site quota frees up")
			}

			return quotaState{over: true, global: true, clearsAt: quotaClearsAt(oldest)}, nil
		}
	}

	return quotaState{}, nil
}

func quotaClearsAt(oldest time.Time) time.Time {
	if oldest.IsZero() {
		return time.Time{}
	}

	return oldest.Add(quotaWindow)
}

func (s *service) quotaMessage(q quotaState) string {
	wait := humaniseWait(time.Until(q.clearsAt))

	if q.global {
		return fmt.Sprintf("Everyone has been keeping me busy and the whole site has reached its message limit. Try me again %s.", wait)
	}

	return fmt.Sprintf("You have reached your message limit for the moment. Try me again %s.", wait)
}

func humaniseWait(d time.Duration) string {
	switch {
	case d <= 0:
		return "shortly"
	case d < time.Minute:
		return "in less than a minute"
	case d < time.Hour:
		minutes := int(d.Round(time.Minute).Minutes())
		if minutes == 1 {
			return "in about a minute"
		}

		return fmt.Sprintf("in about %d minutes", minutes)
	default:
		hours := int(d.Round(time.Hour).Hours())
		if hours <= 1 {
			return "in about an hour"
		}

		return fmt.Sprintf("in about %d hours", hours)
	}
}

func (s *service) startTyping(ev botEvent, botUserID uuid.UUID) func() {
	if ev.Surface != SurfaceChat || len(ev.Audience) == 0 {
		return func() {}
	}

	stop := make(chan struct{})
	var once sync.Once

	msg := ws.Message{
		Type: "typing",
		Data: map[string]any{
			"room_id": ev.ScopeID.String(),
			"user_id": botUserID.String(),
		},
	}

	send := func() {
		for _, memberID := range ev.Audience {
			s.hub.SendToUser(memberID, msg)
		}
	}

	send()

	go func() {
		ticker := time.NewTicker(typingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-s.quit:
				return
			case <-ticker.C:
				send()
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(stop)
		})
	}
}

func safetyIdentifier(userID uuid.UUID) string {
	sum := sha256.Sum256([]byte(safetyIDSalt + userID.String()))

	return hex.EncodeToString(sum[:])
}

func firstNonBlank(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

func firstPositive(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}

	return 0
}

func usageOf(result *openai.CompletionResult) repository.InvocationUsage {
	return repository.InvocationUsage{
		PromptTokens:       result.PromptTokens,
		CachedPromptTokens: result.CachedPromptTokens,
		CacheWriteTokens:   result.CacheWriteTokens,
		CompletionTokens:   result.CompletionTokens,
		ReasoningTokens:    result.ReasoningTokens,
	}
}

func recordTokens(result *openai.CompletionResult) {
	tokensTotal.WithLabelValues("prompt").Add(float64(result.PromptTokens))
	tokensTotal.WithLabelValues("cached_prompt").Add(float64(result.CachedPromptTokens))
	tokensTotal.WithLabelValues("cache_write").Add(float64(result.CacheWriteTokens))
	tokensTotal.WithLabelValues("completion").Add(float64(result.CompletionTokens))
	tokensTotal.WithLabelValues("reasoning").Add(float64(result.ReasoningTokens))
}
