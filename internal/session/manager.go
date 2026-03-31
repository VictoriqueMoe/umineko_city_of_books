package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

const (
	CookieName      = "ut_session"
	defaultDuration = 30 * 24 * time.Hour
)

type (
	Manager struct {
		repo        repository.SessionRepository
		settingsSvc settings.Service
	}
)

func NewManager(repo repository.SessionRepository, settingsSvc settings.Service) *Manager {
	return &Manager{repo: repo, settingsSvc: settingsSvc}
}

func (m *Manager) Create(ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	days := m.settingsSvc.GetInt(ctx, config.SettingSessionDurationDays)
	duration := defaultDuration
	if days > 0 {
		duration = time.Duration(days) * 24 * time.Hour
	}

	expiresAt := time.Now().Add(duration)
	if err := m.repo.Create(ctx, token, userID, expiresAt); err != nil {
		return "", err
	}

	return token, nil
}

func (m *Manager) Validate(ctx context.Context, token string) (uuid.UUID, error) {
	userID, expiresAt, err := m.repo.GetUserID(ctx, token)
	if err != nil {
		return uuid.Nil, err
	}

	if time.Now().After(expiresAt) {
		m.repo.Delete(ctx, token)
		return uuid.Nil, fmt.Errorf("session expired")
	}

	return userID, nil
}

func (m *Manager) Delete(ctx context.Context, token string) error {
	return m.repo.Delete(ctx, token)
}

func (m *Manager) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	return m.repo.DeleteAllForUser(ctx, userID)
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
