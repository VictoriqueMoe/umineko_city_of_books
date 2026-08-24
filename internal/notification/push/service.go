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

const (
	PlatformWeb = "web"
)

type (
	Notification struct {
		Title string
		Body  string
		Data  map[string]string
	}

	Service interface {
		RegisterToken(ctx context.Context, userID uuid.UUID, token, platform string) error
		UnregisterToken(ctx context.Context, userID uuid.UUID, token string) error
		SendToUser(ctx context.Context, userID uuid.UUID, n Notification)
		Enabled() bool
		OnSettingChanged(key config.SiteSettingKey, value string)
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
		logger.Ctx(ctx).Error().Err(err).Msg("failed to load FCM credentials")
		return
	}

	app, err := firebase.NewApp(ctx, nil, option.WithAuthCredentials(creds))
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to initialise firebase app")
		return
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to initialise firebase messaging client")
		return
	}

	s.client = client
	logger.Ctx(ctx).Info().Msg("FCM push client configured")
}

func (s *service) Enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client != nil
}

func (s *service) RegisterToken(ctx context.Context, userID uuid.UUID, token, platform string) error {
	return s.repo.Upsert(ctx, userID, token, platform)
}

func (s *service) UnregisterToken(ctx context.Context, userID uuid.UUID, token string) error {
	return s.repo.Delete(ctx, userID, token)
}

func buildMessage(reg repository.DeviceRegistration, n Notification) *messaging.Message {
	notification := &messaging.Notification{Title: n.Title, Body: n.Body}

	if reg.Platform == PlatformWeb {
		return &messaging.Message{
			Notification: notification,
			Data:         n.Data,
			Fid:          reg.Token,
		}
	}

	return &messaging.Message{
		Notification: notification,
		Data:         n.Data,
		Token:        reg.Token,
	}
}

func (s *service) SendToUser(ctx context.Context, userID uuid.UUID, n Notification) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return
	}

	registrations, err := s.repo.RegistrationsForUser(ctx, userID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("failed to load device tokens for push")
		return
	}

	if len(registrations) == 0 {
		return
	}

	messages := make([]*messaging.Message, 0, len(registrations))
	for _, reg := range registrations {
		messages = append(messages, buildMessage(reg, n))
	}

	resp, err := client.SendEach(ctx, messages)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("failed to send push notification")
		return
	}

	if resp.FailureCount == 0 {
		return
	}

	var stale []string
	for i, r := range resp.Responses {
		if r.Error != nil && messaging.IsUnregistered(r.Error) {
			stale = append(stale, registrations[i].Token)
		}
	}

	if len(stale) > 0 {
		if err := s.repo.DeleteMany(ctx, userID, stale); err != nil {
			logger.Ctx(ctx).Warn().Err(err).Msg("failed to prune stale device tokens")
		}
	}
}

func (s *service) OnSettingChanged(key config.SiteSettingKey, _ string) {
	if key == config.SettingPushEnabled.Key {
		s.buildClient()
	}
}
