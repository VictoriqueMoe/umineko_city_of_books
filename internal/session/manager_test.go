package session

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) (*Manager, *repository.MockSessionRepository, *settings.MockService) {
	repo := repository.NewMockSessionRepository(t)
	settingsSvc := settings.NewMockService(t)
	return NewManager(repo, settingsSvc), repo, settingsSvc
}

func TestGenerateToken_ReturnsHex64(t *testing.T) {
	// given / when
	token, err := generateToken()

	// then
	require.NoError(t, err)
	assert.Len(t, token, 64)
	_, decodeErr := hex.DecodeString(token)
	assert.NoError(t, decodeErr)
}

func TestGenerateToken_UniquePerCall(t *testing.T) {
	// given / when
	a, err1 := generateToken()
	b, err2 := generateToken()

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, a, b)
}

func TestCreate_HappyPath(t *testing.T) {
	// given
	mgr, repo, settingsSvc := newTestManager(t)
	userID := uuid.New()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingSessionDurationDays).Return(30)
	repo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.MatchedBy(func(expiresAt time.Time) bool {
		diff := time.Until(expiresAt)
		return diff > 29*24*time.Hour && diff < 31*24*time.Hour
	})).Return(nil)

	// when
	token, err := mgr.Create(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Len(t, token, 64)
}

func TestCreate_UsesDefaultDurationWhenSettingNonPositive(t *testing.T) {
	// given
	mgr, repo, settingsSvc := newTestManager(t)
	userID := uuid.New()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingSessionDurationDays).Return(0)
	repo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.MatchedBy(func(expiresAt time.Time) bool {
		diff := time.Until(expiresAt)
		return diff > 29*24*time.Hour && diff < 31*24*time.Hour
	})).Return(nil)

	// when
	_, err := mgr.Create(context.Background(), userID)

	// then
	require.NoError(t, err)
}

func TestCreate_RepoErrorBubbles(t *testing.T) {
	// given
	mgr, repo, settingsSvc := newTestManager(t)
	userID := uuid.New()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingSessionDurationDays).Return(30)
	repo.EXPECT().Create(mock.Anything, mock.Anything, userID, mock.Anything).Return(errors.New("db down"))

	// when
	token, err := mgr.Create(context.Background(), userID)

	// then
	require.Error(t, err)
	assert.Empty(t, token)
}

func TestValidate_HappyPath(t *testing.T) {
	// given
	mgr, repo, _ := newTestManager(t)
	userID := uuid.New()
	token := "some-token"
	repo.EXPECT().GetUserID(mock.Anything, token).Return(userID, time.Now().Add(time.Hour), nil)

	// when
	got, err := mgr.Validate(context.Background(), token)

	// then
	require.NoError(t, err)
	assert.Equal(t, userID, got)
}

func TestValidate_RepoErrorBubbles(t *testing.T) {
	// given
	mgr, repo, _ := newTestManager(t)
	token := "bad-token"
	repo.EXPECT().GetUserID(mock.Anything, token).Return(uuid.Nil, time.Time{}, errors.New("not found"))

	// when
	got, err := mgr.Validate(context.Background(), token)

	// then
	require.Error(t, err)
	assert.Equal(t, uuid.Nil, got)
}

func TestValidate_ExpiredDeletesAndReturnsError(t *testing.T) {
	// given
	mgr, repo, _ := newTestManager(t)
	token := "expired-token"
	userID := uuid.New()
	repo.EXPECT().GetUserID(mock.Anything, token).Return(userID, time.Now().Add(-time.Hour), nil)
	repo.EXPECT().Delete(mock.Anything, token).Return(nil)

	// when
	got, err := mgr.Validate(context.Background(), token)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
	assert.Equal(t, uuid.Nil, got)
}

func TestDelete_Delegates(t *testing.T) {
	// given
	mgr, repo, _ := newTestManager(t)
	token := "some-token"
	repo.EXPECT().Delete(mock.Anything, token).Return(nil)

	// when
	err := mgr.Delete(context.Background(), token)

	// then
	require.NoError(t, err)
}

func TestDelete_RepoError(t *testing.T) {
	// given
	mgr, repo, _ := newTestManager(t)
	token := "some-token"
	repo.EXPECT().Delete(mock.Anything, token).Return(errors.New("boom"))

	// when
	err := mgr.Delete(context.Background(), token)

	// then
	require.Error(t, err)
}

func TestDeleteAllForUser_Delegates(t *testing.T) {
	// given
	mgr, repo, _ := newTestManager(t)
	userID := uuid.New()
	repo.EXPECT().DeleteAllForUser(mock.Anything, userID).Return(nil)

	// when
	err := mgr.DeleteAllForUser(context.Background(), userID)

	// then
	require.NoError(t, err)
}

func TestDeleteAllForUser_RepoError(t *testing.T) {
	// given
	mgr, repo, _ := newTestManager(t)
	userID := uuid.New()
	repo.EXPECT().DeleteAllForUser(mock.Anything, userID).Return(errors.New("boom"))

	// when
	err := mgr.DeleteAllForUser(context.Background(), userID)

	// then
	require.Error(t, err)
}
