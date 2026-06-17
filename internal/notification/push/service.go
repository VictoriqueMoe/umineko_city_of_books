package push

import (
	"context"
	"sync"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"cloud.google.com/go/auth/credentials"
	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

type (
	Notification struct {
		Title string
		Body  string
		Data  map[string]string
	}

	Service interface {
		RegisterToken(ctx context.Context, userID uuid.UUID, token, platform string) error
		UnregisterToken(ctx context.Context, token string) error
		SendToUser(ctx context.Context, userID uuid.UUID, n Notification)
		Enabled() bool
	}

	service struct {
		settingsSvc settings.Service
		repo        repository.DeviceTokenRepository
		credsFile   string
		mu          sync.RWMutex
		client      *messaging.Client
	}
)

func NewService(settingsSvc settings.Service, repo repository.DeviceTokenRepository, credsFile string) Service {
	svc := &service{
		settingsSvc: settingsSvc,
		repo:        repo,
		credsFile:   credsFile,
	}
	svc.buildClient()
	return svc
}

func (s *service) buildClient() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.client = nil

	if !s.settingsSvc.GetBool(context.Background(), config.SettingPushEnabled) {
		return
	}

	if s.credsFile == "" {
		logger.Log.Warn().Msg("push_enabled is on but FCM_CREDENTIALS_FILE is not set, native push disabled")
		return
	}

	ctx := context.Background()
	creds, err := credentials.NewCredentialsFromFile(credentials.ServiceAccount, s.credsFile, &credentials.DetectOptions{
		Scopes: []string{"https://www.googleapis.com/auth/firebase.messaging"},
	})
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to load FCM credentials")
		return
	}

	app, err := firebase.NewApp(ctx, nil, option.WithAuthCredentials(creds))
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to initialise firebase app")
		return
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to initialise firebase messaging client")
		return
	}

	s.client = client
	logger.Log.Info().Msg("FCM push client configured")
}

func (s *service) Enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client != nil
}

func (s *service) RegisterToken(ctx context.Context, userID uuid.UUID, token, platform string) error {
	return s.repo.Upsert(ctx, userID, token, platform)
}

func (s *service) UnregisterToken(ctx context.Context, token string) error {
	return s.repo.Delete(ctx, token)
}

func (s *service) SendToUser(ctx context.Context, userID uuid.UUID, n Notification) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return
	}

	tokens, err := s.repo.TokensForUser(ctx, userID)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("failed to load device tokens for push")
		return
	}

	if len(tokens) == 0 {
		return
	}

	msg := &messaging.MulticastMessage{
		Tokens:       tokens,
		Notification: &messaging.Notification{Title: n.Title, Body: n.Body},
		Data:         n.Data,
	}

	resp, err := client.SendEachForMulticast(ctx, msg)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("failed to send push notification")
		return
	}

	if resp.FailureCount == 0 {
		return
	}

	var stale []string
	for i, r := range resp.Responses {
		if r.Error != nil && messaging.IsUnregistered(r.Error) {
			stale = append(stale, tokens[i])
		}
	}

	if len(stale) > 0 {
		if err := s.repo.DeleteMany(ctx, stale); err != nil {
			logger.Log.Warn().Err(err).Msg("failed to prune stale device tokens")
		}
	}
}

type SettingListener struct {
	svc *service
}

func NewSettingListener(svc Service) *SettingListener {
	return &SettingListener{svc: svc.(*service)}
}

func (l *SettingListener) OnSettingChanged(key config.SiteSettingKey, _ string) {
	if key == config.SettingPushEnabled.Key {
		l.svc.buildClient()
	}
}
