package overlay

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type overlayMocks struct {
	repo     *repository.MockOverlayTokenRepository
	settings *settings.MockService
	hub      *ws.Hub
}

func newTestOverlayService(t *testing.T) (Service, *overlayMocks) {
	repo := repository.NewMockOverlayTokenRepository(t)
	settingsSvc := settings.NewMockService(t)
	hub := ws.NewHub()
	return NewService(repo, hub, settingsSvc), &overlayMocks{repo: repo, settings: settingsSvc, hub: hub}
}

func TestToken_ReturnsExisting(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().GetByUser(mock.Anything, userID).Return("existing_tok", nil)

	// when
	tok, err := svc.Token(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "existing_tok", tok)
}

func TestToken_GeneratesWhenMissing(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().GetByUser(mock.Anything, userID).Return("", nil)
	m.repo.EXPECT().Upsert(mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)

	// when
	tok, err := svc.Token(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

func TestToken_RepoError(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().GetByUser(mock.Anything, userID).Return("", errors.New("db down"))

	// when
	_, err := svc.Token(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestResetToken_GeneratesUniqueTokens(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().Upsert(mock.Anything, userID, mock.AnythingOfType("string")).Return(nil).Twice()

	// when
	tok1, err1 := svc.ResetToken(context.Background(), userID)
	tok2, err2 := svc.ResetToken(context.Background(), userID)

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEmpty(t, tok1)
	assert.NotEqual(t, tok1, tok2)
}

func TestResetToken_UpsertError(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().Upsert(mock.Anything, userID, mock.Anything).Return(errors.New("boom"))

	// when
	_, err := svc.ResetToken(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestValidate_EmptyTokenReturnsNil(t *testing.T) {
	// given
	svc, _ := newTestOverlayService(t)

	// when
	id, err := svc.Validate(context.Background(), "")

	// then
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, id)
}

func TestValidate_LooksUpToken(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().GetUserByToken(mock.Anything, "tok").Return(userID, nil)

	// when
	id, err := svc.Validate(context.Background(), "tok")

	// then
	require.NoError(t, err)
	assert.Equal(t, userID, id)
}

func TestConnectURL_TransformsHTTPSToWSS(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().GetByUser(mock.Anything, userID).Return("tok_abc", nil)
	m.settings.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://umineko.example/")

	// when
	url, err := svc.ConnectURL(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "wss://umineko.example/api/v1/overlay?token=tok_abc", url)
}

func TestConnectURL_TransformsHTTPToWS(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().GetByUser(mock.Anything, userID).Return("tok_abc", nil)
	m.settings.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://localhost:8080")

	// when
	url, err := svc.ConnectURL(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "ws://localhost:8080/api/v1/overlay?token=tok_abc", url)
}

func TestConnectURL_EscapesToken(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().GetByUser(mock.Anything, userID).Return("a+b/c", nil)
	m.settings.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://x")

	// when
	url, err := svc.ConnectURL(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "wss://x/api/v1/overlay?token=a%2Bb%2Fc", url)
}

func TestBuildSEF_RendersTemplate(t *testing.T) {
	// given
	svc, m := newTestOverlayService(t)
	userID := uuid.New()
	m.repo.EXPECT().GetByUser(mock.Anything, userID).Return("tok_xyz", nil)
	m.settings.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://umineko.example")
	m.settings.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("Umineko DB")

	// when
	sef, err := svc.BuildSEF(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Contains(t, sef, "Umineko DB Overlay")
	assert.Contains(t, sef, "wss://umineko.example/api/v1/overlay?token=tok_xyz")
	assert.Contains(t, sef, "overlay_event")
}

func TestTestFire_NotConnected(t *testing.T) {
	// given
	svc, _ := newTestOverlayService(t)

	// when
	err := svc.TestFire(uuid.New())

	// then
	require.ErrorIs(t, err, ErrNotConnected)
}

func TestIsConnected_FalseWhenOffline(t *testing.T) {
	// given
	svc, _ := newTestOverlayService(t)

	// when
	connected := svc.IsConnected(uuid.New())

	// then
	assert.False(t, connected)
}

func TestDispatchNotification_IgnoresUnmappedType(t *testing.T) {
	// given
	svc, _ := newTestOverlayService(t)

	// when
	svc.DispatchNotification(uuid.New(), dto.NotificationResponse{Type: dto.NotifChatMessage})

	// then: an unmapped type returns before touching the hub, so nothing happens
}

func TestDispatchNotification_OfflineIsNoop(t *testing.T) {
	// given
	svc, _ := newTestOverlayService(t)

	// when
	svc.DispatchNotification(uuid.New(), dto.NotificationResponse{
		Type:  dto.NotifPostLiked,
		Actor: dto.UserResponse{Username: "beato"},
	})

	// then: a mapped type with an offline recipient sends nothing and does not panic
}

func TestGenerateToken_UniqueAndURLSafe(t *testing.T) {
	// given / when
	a, err1 := generateToken()
	b, err2 := generateToken()

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEmpty(t, a)
	assert.NotEqual(t, a, b)
	assert.NotContains(t, a, "=")
	assert.NotContains(t, a, "+")
	assert.NotContains(t, a, "/")
}
