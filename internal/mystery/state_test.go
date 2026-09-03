package mystery

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSetPaused_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.SetPaused(context.Background(), mid, userID, true)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSetPaused_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubNotAuthor(m, mid, userID)

	// when
	err := svc.SetPaused(context.Background(), mid, userID, true)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestSetPaused_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().SetPaused(mock.Anything, mid, true).Return(errors.New("boom"))

	// when
	err := svc.SetPaused(context.Background(), mid, userID, true)

	// then
	require.Error(t, err)
}

func TestSetPaused_OK_Pause(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().SetPaused(mock.Anything, mid, true).Return(nil)
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()

	// when
	err := svc.SetPaused(context.Background(), mid, userID, true)

	// then
	require.NoError(t, err)
}

func TestSetPaused_OK_Unpause_NotifiesPlayers(t *testing.T) {
	synctest.Test(t, testSetPausedOKUnpauseNotifiesPlayers)
}

func testSetPausedOKUnpauseNotifiesPlayers(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	player := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().SetPaused(mock.Anything, mid, false).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return([]uuid.UUID{player}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.Type == dto.NotifMysteryUnpaused && p.RecipientID == player
	})).Run(func(_ context.Context, _ dto.NotifyParams) { wg.Done() }).Return(nil).Maybe()

	// when
	err := svc.SetPaused(context.Background(), mid, userID, false)

	// then
	require.NoError(t, err)
	wg.Wait()
}

func TestSetGmAway_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, true)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSetGmAway_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubNotAuthor(m, mid, userID)

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, true)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestSetGmAway_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().SetGmAway(mock.Anything, mid, true).Return(errors.New("boom"))

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, true)

	// then
	require.Error(t, err)
}

func TestSetGmAway_OK_Away(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().SetGmAway(mock.Anything, mid, true).Return(nil)
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, true)

	// then
	require.NoError(t, err)
}

func TestSetGmAway_OK_Back_NotifiesPlayers(t *testing.T) {
	synctest.Test(t, testSetGmAwayOKBackNotifiesPlayers)
}

func testSetGmAwayOKBackNotifiesPlayers(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	player := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().SetGmAway(mock.Anything, mid, false).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return([]uuid.UUID{player}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.Type == dto.NotifMysteryGmBack && p.RecipientID == player
	})).Run(func(_ context.Context, _ dto.NotifyParams) { wg.Done() }).Return(nil).Maybe()

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, false)

	// then
	require.NoError(t, err)
	wg.Wait()
}
