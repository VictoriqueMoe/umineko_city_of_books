package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"umineko_city_of_books/internal/repository"
)

const (
	CookieName = "ut_session"
	Duration   = 30 * 24 * time.Hour
)

type (
	Manager struct {
		repo repository.SessionRepository
	}
)

func NewManager(repo repository.SessionRepository) *Manager {
	return &Manager{repo: repo}
}

func (m *Manager) Create(ctx context.Context, userID int) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	expiresAt := time.Now().Add(Duration)
	if err := m.repo.Create(ctx, token, userID, expiresAt); err != nil {
		return "", err
	}

	return token, nil
}

func (m *Manager) Validate(ctx context.Context, token string) (int, error) {
	userID, expiresAt, err := m.repo.GetUserID(ctx, token)
	if err != nil {
		return 0, err
	}

	if time.Now().After(expiresAt) {
		m.repo.Delete(ctx, token)
		return 0, fmt.Errorf("session expired")
	}

	return userID, nil
}

func (m *Manager) Delete(ctx context.Context, token string) error {
	return m.repo.Delete(ctx, token)
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
