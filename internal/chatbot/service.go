package chatbot

import (
	"cmp"
	"context"
	"slices"

	"strings"
	"sync"
	"sync/atomic"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	chatbotKeyPrefix = "chatbot_"
	workerCount      = 2
	queueCap         = 32
	jobTimeout       = 90 * time.Second
	typingInterval   = 3 * time.Second
	messageBodyMax   = 20000
	replyQuoteMax    = 160
	promptCharBudget = 200000
	safetyIDSalt     = "umineko-chatbot-safety-id:"
	shutdownMessage  = "chatbot worker pool drained"
	refusalCooldown  = 10 * time.Minute
	quotaWindow      = 24 * time.Hour
	noticeTimeout    = 15 * time.Second
)

type (
	Service interface {
		chat.MessageObserver
		post.CommentObserver
		Enabled() bool
		OnSettingsBatchChanged(keys []config.SiteSettingKey)
		Reload()
		Listing() []dto.ChatbotSummary
		Shutdown(ctx context.Context) error
	}

	tuning struct {
		enabled           bool
		model             string
		reasoningEffort   string
		verbosity         string
		maxOutputTokens   int
		contextMessages   int
		maxReplyChain     int
		requirePermission bool
		cooldown          time.Duration
		perUserPerDay     int
		perDay            int
	}

	job struct {
		ev       botEvent
		bot      repository.Chatbot
		useChain bool
		notice   outcome
	}

	noticeKey struct {
		user   uuid.UUID
		reason Reason
	}

	quotaState struct {
		over     bool
		global   bool
		clearsAt time.Time
	}

	service struct {
		openaiSvc   openai.Service
		chatSvc     chat.Service
		postSvc     post.Service
		chatRepo    repository.ChatRepository
		postRepo    repository.PostRepository
		botRepo     repository.ChatbotRepository
		auditRepo   repository.AuditLogRepository
		authzSvc    authz.Service
		settingsSvc settings.Service
		hub         *ws.Hub

		mu         sync.RWMutex
		tune       tuning
		bots       map[uuid.UUID]repository.Chatbot
		loaded     bool
		lastUse    sync.Map
		lastNotice sync.Map
		inScope    sync.Map

		jobs    chan job
		quit    chan struct{}
		wg      sync.WaitGroup
		closing atomic.Bool
	}
)

func NewService(
	openaiSvc openai.Service,
	chatSvc chat.Service,
	postSvc post.Service,
	chatRepo repository.ChatRepository,
	postRepo repository.PostRepository,
	botRepo repository.ChatbotRepository,
	auditRepo repository.AuditLogRepository,
	authzSvc authz.Service,
	settingsSvc settings.Service,
	hub *ws.Hub,
) Service {
	s := &service{
		openaiSvc:   openaiSvc,
		chatSvc:     chatSvc,
		postSvc:     postSvc,
		chatRepo:    chatRepo,
		postRepo:    postRepo,
		botRepo:     botRepo,
		auditRepo:   auditRepo,
		authzSvc:    authzSvc,
		settingsSvc: settingsSvc,
		hub:         hub,
		bots:        make(map[uuid.UUID]repository.Chatbot),
		jobs:        make(chan job, queueCap),
		quit:        make(chan struct{}),
	}

	s.reload()

	s.wg.Add(workerCount)
	for i := range workerCount {
		go s.worker(i)
	}

	return s
}

func (s *service) Enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tune.enabled
}

func (s *service) OnSettingsBatchChanged(keys []config.SiteSettingKey) {
	if slices.ContainsFunc(keys, isChatbotKey) {
		s.reload()
	}
}

func isChatbotKey(key config.SiteSettingKey) bool {
	return strings.HasPrefix(string(key), chatbotKeyPrefix)
}

func (s *service) reload() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	next := tuning{
		enabled:           s.settingsSvc.GetBool(ctx, config.SettingChatbotEnabled),
		model:             s.settingsSvc.Get(ctx, config.SettingChatbotModel),
		reasoningEffort:   s.settingsSvc.Get(ctx, config.SettingChatbotReasoningEffort),
		verbosity:         s.settingsSvc.Get(ctx, config.SettingChatbotVerbosity),
		maxOutputTokens:   s.settingsSvc.GetInt(ctx, config.SettingChatbotMaxOutputTokens),
		contextMessages:   s.settingsSvc.GetInt(ctx, config.SettingChatbotContextMessages),
		maxReplyChain:     s.settingsSvc.GetInt(ctx, config.SettingChatbotMaxReplyChain),
		requirePermission: s.settingsSvc.GetBool(ctx, config.SettingChatbotRequirePermission),
		cooldown:          time.Duration(s.settingsSvc.GetInt(ctx, config.SettingChatbotReplyCooldownSeconds)) * time.Second,
		perUserPerDay:     s.settingsSvc.GetInt(ctx, config.SettingChatbotMaxRepliesPerUserDay),
		perDay:            s.settingsSvc.GetInt(ctx, config.SettingChatbotMaxRepliesPerDay),
	}

	bots := make(map[uuid.UUID]repository.Chatbot)
	if next.enabled {
		rows, err := s.botRepo.ListBots(ctx)
		if err != nil {
			logger.Log.Error().Err(err).Msg("chatbot: load bots, keeping the previously loaded set")

			s.mu.Lock()
			s.tune = next
			s.mu.Unlock()

			return
		}

		for _, row := range rows {
			if row.Enabled {
				bots[row.UserID] = row
			}
		}
	}

	s.mu.Lock()
	s.tune = next
	s.bots = bots
	s.loaded = true
	s.mu.Unlock()

	online := make([]uuid.UUID, 0, len(bots))
	for id := range bots {
		online = append(online, id)
	}

	s.hub.SetAlwaysOnline(online)

	s.hub.BroadcastPublic(ws.Message{
		Type: "chatbots_changed",
		Data: map[string]any{},
	})
}

func (s *service) Reload() {
	s.reload()
}

func (s *service) Listing() []dto.ChatbotSummary {
	tune, bots := s.snapshot()

	out := make([]dto.ChatbotSummary, 0, len(bots))
	if !tune.enabled || !s.openaiSvc.Enabled() {
		return out
	}

	for _, bot := range bots {
		out = append(out, dto.ChatbotSummary{
			UserID:      bot.UserID,
			Username:    bot.Username,
			DisplayName: bot.DisplayName,
			AvatarURL:   bot.AvatarURL,
		})
	}

	slices.SortFunc(out, func(a, b dto.ChatbotSummary) int {
		return cmp.Compare(strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName))
	})

	return out
}

func (s *service) snapshot() (tuning, map[uuid.UUID]repository.Chatbot) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tune, s.bots
}

func (s *service) Shutdown(ctx context.Context) error {
	if !s.closing.CompareAndSwap(false, true) {
		return nil
	}

	close(s.quit)

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Log.Info().Msg(shutdownMessage)

		return nil
	case <-ctx.Done():
		logger.Log.Warn().Int("abandoned", len(s.jobs)).Msg("chatbot drain timed out")

		return ctx.Err()
	}
}
