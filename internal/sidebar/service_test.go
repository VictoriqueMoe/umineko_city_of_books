package sidebar_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/sidebar"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_ListVisited_DelegatesAndWraps(t *testing.T) {
	// given
	repo := repository.NewMockSidebarLastVisitedRepository(t)
	svc := sidebar.NewService(repo)
	userID := uuid.New()
	repo.EXPECT().ListForUser(mock.Anything, userID).Return(map[string]string{
		"mysteries": "2026-04-24T10:00:00Z",
	}, nil)

	// when
	resp, err := svc.ListVisited(context.Background(), userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &dto.SidebarLastVisitedResponse{
		Visited: map[string]string{"mysteries": "2026-04-24T10:00:00Z"},
	}, resp)
}

func TestService_ListVisited_PropagatesError(t *testing.T) {
	// given
	repo := repository.NewMockSidebarLastVisitedRepository(t)
	svc := sidebar.NewService(repo)
	userID := uuid.New()
	repo.EXPECT().ListForUser(mock.Anything, userID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.ListVisited(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestService_MarkVisited_RejectsEmptyKey(t *testing.T) {
	// given
	repo := repository.NewMockSidebarLastVisitedRepository(t)
	svc := sidebar.NewService(repo)

	// when
	err := svc.MarkVisited(context.Background(), uuid.New(), "")

	// then
	assert.ErrorIs(t, err, sidebar.ErrEmptyKey)
}

func TestService_MarkVisited_RejectsTooLongKey(t *testing.T) {
	// given
	repo := repository.NewMockSidebarLastVisitedRepository(t)
	svc := sidebar.NewService(repo)

	// when
	err := svc.MarkVisited(context.Background(), uuid.New(), strings.Repeat("a", 101))

	// then
	assert.ErrorIs(t, err, sidebar.ErrKeyTooLong)
}

func TestService_MarkVisited_AcceptsBoundaryLength(t *testing.T) {
	// given
	repo := repository.NewMockSidebarLastVisitedRepository(t)
	svc := sidebar.NewService(repo)
	userID := uuid.New()
	key := strings.Repeat("a", 100)
	repo.EXPECT().Upsert(mock.Anything, userID, key).Return(nil)

	// when
	err := svc.MarkVisited(context.Background(), userID, key)

	// then
	assert.NoError(t, err)
}

func TestService_MarkVisited_DelegatesToRepo(t *testing.T) {
	// given
	repo := repository.NewMockSidebarLastVisitedRepository(t)
	svc := sidebar.NewService(repo)
	userID := uuid.New()
	repo.EXPECT().Upsert(mock.Anything, userID, "rooms").Return(nil)

	// when
	err := svc.MarkVisited(context.Background(), userID, "rooms")

	// then
	assert.NoError(t, err)
}
