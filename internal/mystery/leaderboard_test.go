package mystery

import (
	"context"
	"errors"
	"strings"
	"testing"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetLeaderboard_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().GetLeaderboard(mock.Anything, 10).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetLeaderboard(context.Background(), bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
}

func TestGetLeaderboard_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	entries := []repository.LeaderboardEntry{{UserID: uuid.New(), Username: "u", Score: 5}}
	m.repo.EXPECT().GetLeaderboard(mock.Anything, 10).Return(entries, nil)

	// when
	got, err := svc.GetLeaderboard(context.Background(), bounds.NewPage(10, 0))

	// then
	require.NoError(t, err)
	require.Len(t, got.Entries, 1)
	assert.Equal(t, 5, got.Entries[0].Score)
}

func TestGetTopDetectiveIDs_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expected := []string{"a", "b"}
	m.repo.EXPECT().GetTopDetectiveIDs(mock.Anything).Return(expected, nil)

	// when
	got, err := svc.GetTopDetectiveIDs(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestGetGMLeaderboard_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().GetGMLeaderboard(mock.Anything, 5).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetGMLeaderboard(context.Background(), bounds.NewPage(5, 0))

	// then
	require.Error(t, err)
}

func TestGetGMLeaderboard_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	entries := []repository.GMLeaderboardEntry{{UserID: uuid.New(), Score: 7, MysteryCount: 2, PlayerCount: 4}}
	m.repo.EXPECT().GetGMLeaderboard(mock.Anything, 5).Return(entries, nil)

	// when
	got, err := svc.GetGMLeaderboard(context.Background(), bounds.NewPage(5, 0))

	// then
	require.NoError(t, err)
	require.Len(t, got.Entries, 1)
	assert.Equal(t, 7, got.Entries[0].Score)
}

func TestGetTopGMIDs_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().GetTopGMIDs(mock.Anything).Return([]string{"x"}, nil)

	// when
	got, err := svc.GetTopGMIDs(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"x"}, got)
}

func TestListByUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.repo.EXPECT().ListByUser(mock.Anything, userID, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListByUser(context.Background(), userID, bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
}

func TestListByUser_TruncatesLongBody(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	var body strings.Builder
	for range 300 {
		body.WriteString("x")
	}
	rows := []repository.MysteryRow{{ID: uuid.New(), Body: body.String()}}
	m.repo.EXPECT().ListByUser(mock.Anything, userID, 10, 0).Return(rows, 1, nil)

	// when
	got, err := svc.ListByUser(context.Background(), userID, bounds.NewPage(10, 0))

	// then
	require.NoError(t, err)
	assert.Equal(t, 203, len(got.Mysteries[0].Body))
	assert.Equal(t, 1, got.Total)
}
