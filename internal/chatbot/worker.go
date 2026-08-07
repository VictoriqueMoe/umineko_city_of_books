package chatbot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	tune, _ := s.snapshot()

	invocationID := uuid.New()
	model := firstNonBlank(j.bot.Model, tune.model)

	status, err := s.reply(ctx, j, tune, invocationID, model)
	if err != nil {
		logger.Log.Error().Err(err).Int("worker", id).Str("bot", j.bot.Username).Str("surface", string(j.ev.Surface)).Msg("chatbot reply failed")
	}

	invocationsTotal.WithLabelValues(string(status)).Inc()
}

func (s *service) reply(ctx context.Context, j job, tune tuning, invocationID uuid.UUID, model string) (repository.InvocationStatus, error) {
	if over, err := s.overQuota(ctx, j.ev.SenderID, tune); err != nil || over {
		if err != nil {
			return repository.InvocationFailed, err
		}

		droppedTotal.WithLabelValues("quota").Inc()

		return repository.InvocationQuota, nil
	}

	var roomID *uuid.UUID
	if j.ev.Surface == SurfaceChat {
		roomID = new(j.ev.ScopeID)
	}

	if err := s.botRepo.CreateInvocation(ctx, invocationID, j.bot.UserID, j.ev.SenderID, roomID, j.ev.ItemID, string(j.ev.Surface), model); err != nil {
		return repository.InvocationFailed, err
	}

	stopTyping := s.startTyping(j.ev, j.bot.UserID)
	defer stopTyping()

	req := openai.CompletionRequest{
		Model:           model,
		SystemPrompt:    j.bot.SystemPrompt,
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

		return repository.InvocationFailed, err
	}

	recordTokens(result)

	body := strings.TrimSpace(result.Text)
	if result.Incomplete || body == "" {
		_ = s.botRepo.CompleteInvocation(ctx, invocationID, usageOf(result), repository.InvocationRefused)

		return repository.InvocationRefused, nil
	}

	stopTyping()

	if sendErr := s.deliver(ctx, j, body); sendErr != nil {
		_ = s.botRepo.CompleteInvocation(ctx, invocationID, usageOf(result), repository.InvocationRefused)

		return repository.InvocationRefused, sendErr
	}

	if err := s.botRepo.CompleteInvocation(ctx, invocationID, usageOf(result), repository.InvocationReplied); err != nil {
		return repository.InvocationReplied, err
	}

	return repository.InvocationReplied, nil
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

func (s *service) overQuota(ctx context.Context, userID uuid.UUID, tune tuning) (bool, error) {
	if tune.perUserPerDay > 0 {
		used, err := s.botRepo.CountUserInvocationsToday(ctx, userID)
		if err != nil {
			return false, err
		}
		if used >= tune.perUserPerDay {
			return true, nil
		}
	}

	if tune.perDay > 0 {
		used, err := s.botRepo.CountInvocationsToday(ctx)
		if err != nil {
			return false, err
		}
		if used >= tune.perDay {
			return true, nil
		}
	}

	return false, nil
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
