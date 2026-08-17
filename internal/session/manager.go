package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

const (
	CookieName      = "ut_session"
	bearerPrefix    = "Bearer "
	defaultDuration = 30 * 24 * time.Hour
)

type (
	Disconnector interface {
		DisconnectUser(userID uuid.UUID) int
	}

	Manager struct {
		repo         repository.SessionRepository
		settingsSvc  settings.Service
		disconnector Disconnector
	}
)

func NewManager(repo repository.SessionRepository, settingsSvc settings.Service) *Manager {
	return &Manager{repo: repo, settingsSvc: settingsSvc}
}

func (m *Manager) SetDisconnector(d Disconnector) {
	m.disconnector = d
}

func (m *Manager) Disconnect(userID uuid.UUID) {
	if m.disconnector == nil {
		return
	}

	m.disconnector.DisconnectUser(userID)
}

func (m *Manager) Issue(ctx context.Context) (string, time.Time, error) {
	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate token: %w", err)
	}

	days := m.settingsSvc.GetInt(ctx, config.SettingSessionDurationDays)
	duration := defaultDuration
	if days > 0 {
		duration = time.Duration(days) * 24 * time.Hour
	}

	return token, time.Now().Add(duration), nil
}

func (m *Manager) Create(ctx context.Context, userID uuid.UUID) (string, error) {
	token, expiresAt, err := m.Issue(ctx)
	if err != nil {
		return "", err
	}

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
	if err := m.repo.DeleteAllForUser(ctx, userID); err != nil {
		return err
	}

	m.Disconnect(userID)

	return nil
}

func (m *Manager) CleanExpired(ctx context.Context) (int, error) {
	return m.repo.CleanExpired(ctx)
}

func (m *Manager) DeleteAllForUserExcept(ctx context.Context, userID uuid.UUID, keepToken string) error {
	if keepToken == "" {
		return m.DeleteAllForUser(ctx, userID)
	}

	if err := m.repo.DeleteAllForUserExcept(ctx, userID, keepToken); err != nil {
		return err
	}

	m.Disconnect(userID)

	return nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func BearerToken(authorization string) string {
	if len(authorization) < len(bearerPrefix) {
		return ""
	}

	if !strings.EqualFold(authorization[:len(bearerPrefix)], bearerPrefix) {
		return ""
	}

	return strings.TrimSpace(authorization[len(bearerPrefix):])
}
