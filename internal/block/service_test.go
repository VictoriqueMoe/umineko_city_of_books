package block

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (
	*service,
	*repository.MockBlockRepository,
	*repository.MockFollowRepository,
	*authz.MockService,
) {
	blockRepo := repository.NewMockBlockRepository(t)
	followRepo := repository.NewMockFollowRepository(t)
	authzSvc := authz.NewMockService(t)
	svc := NewService(blockRepo, followRepo, authzSvc).(*service)
	return svc, blockRepo, followRepo, authzSvc
}

func TestBlock_CannotBlockSelf(t *testing.T) {
	// given
	svc, _, _, _ := newTestService(t)
	userID := uuid.New()

	// when
	err := svc.Block(context.Background(), userID, userID)

	// then
	require.ErrorIs(t, err, ErrCannotBlockSelf)
}

func TestBlock_StaffTargetsRejected(t *testing.T) {
	cases := []struct {
		name     string
		role     role.Role
		wantErr  error
		wantCall bool
	}{
		{"super admin", authz.RoleSuperAdmin, ErrCannotBlockStaff, false},
		{"admin", authz.RoleAdmin, ErrCannotBlockStaff, false},
		{"moderator", authz.RoleModerator, ErrCannotBlockStaff, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _, _, authzSvc := newTestService(t)
			blocker := uuid.New()
			target := uuid.New()
			authzSvc.EXPECT().GetRole(mock.Anything, target).Return(tc.role, nil)

			// when
			err := svc.Block(context.Background(), blocker, target)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestBlock_RoleLookupFailsBubblesError(t *testing.T) {
	// given
	svc, _, _, authzSvc := newTestService(t)
	blocker := uuid.New()
	target := uuid.New()
	authzSvc.EXPECT().GetRole(mock.Anything, target).Return("", errors.New("db down"))

	// when
	err := svc.Block(context.Background(), blocker, target)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check target role")
}

func TestBlock_OK_UnfollowsBothDirections(t *testing.T) {
	// given
	svc, blockRepo, followRepo, authzSvc := newTestService(t)
	blocker := uuid.New()
	target := uuid.New()
	authzSvc.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	blockRepo.EXPECT().Block(mock.Anything, blocker, target).Return(nil)
	followRepo.EXPECT().Unfollow(mock.Anything, blocker, target).Return(nil)
	followRepo.EXPECT().Unfollow(mock.Anything, target, blocker).Return(nil)

	// when
	err := svc.Block(context.Background(), blocker, target)

	// then
	require.NoError(t, err)
}

func TestBlock_UnfollowErrorsSwallowed(t *testing.T) {
	// given
	svc, blockRepo, followRepo, authzSvc := newTestService(t)
	blocker := uuid.New()
	target := uuid.New()
	authzSvc.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	blockRepo.EXPECT().Block(mock.Anything, blocker, target).Return(nil)
	followRepo.EXPECT().Unfollow(mock.Anything, blocker, target).Return(errors.New("boom"))
	followRepo.EXPECT().Unfollow(mock.Anything, target, blocker).Return(errors.New("boom"))

	// when
	err := svc.Block(context.Background(), blocker, target)

	// then
	require.NoError(t, err)
}

func TestBlock_RepoErrorBubbles(t *testing.T) {
	// given
	svc, blockRepo, _, authzSvc := newTestService(t)
	blocker := uuid.New()
	target := uuid.New()
	authzSvc.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	blockRepo.EXPECT().Block(mock.Anything, blocker, target).Return(errors.New("db down"))

	// when
	err := svc.Block(context.Background(), blocker, target)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "db down")
}

func TestUnblock_OK(t *testing.T) {
	// given
	svc, blockRepo, _, _ := newTestService(t)
	blocker := uuid.New()
	target := uuid.New()
	blockRepo.EXPECT().Unblock(mock.Anything, blocker, target).Return(nil)

	// when
	err := svc.Unblock(context.Background(), blocker, target)

	// then
	require.NoError(t, err)
}

func TestUnblock_RepoError(t *testing.T) {
	// given
	svc, blockRepo, _, _ := newTestService(t)
	blocker := uuid.New()
	target := uuid.New()
	blockRepo.EXPECT().Unblock(mock.Anything, blocker, target).Return(errors.New("boom"))

	// when
	err := svc.Unblock(context.Background(), blocker, target)

	// then
	require.Error(t, err)
}

func TestIsBlocked_Delegates(t *testing.T) {
	// given
	svc, blockRepo, _, _ := newTestService(t)
	blocker := uuid.New()
	target := uuid.New()
	blockRepo.EXPECT().IsBlocked(mock.Anything, blocker, target).Return(true, nil)

	// when
	got, err := svc.IsBlocked(context.Background(), blocker, target)

	// then
	require.NoError(t, err)
	assert.True(t, got)
}

func TestIsBlocked_RepoError(t *testing.T) {
	// given
	svc, blockRepo, _, _ := newTestService(t)
	blocker := uuid.New()
	target := uuid.New()
	blockRepo.EXPECT().IsBlocked(mock.Anything, blocker, target).Return(false, errors.New("boom"))

	// when
	_, err := svc.IsBlocked(context.Background(), blocker, target)

	// then
	require.Error(t, err)
}

func TestIsBlockedEither_Delegates(t *testing.T) {
	// given
	svc, blockRepo, _, _ := newTestService(t)
	a := uuid.New()
	b := uuid.New()
	blockRepo.EXPECT().IsBlockedEither(mock.Anything, a, b).Return(true, nil)

	// when
	got, err := svc.IsBlockedEither(context.Background(), a, b)

	// then
	require.NoError(t, err)
	assert.True(t, got)
}

func TestGetBlockedIDs_NilUserReturnsNil(t *testing.T) {
	// given
	svc, _, _, _ := newTestService(t)

	// when
	got, err := svc.GetBlockedIDs(context.Background(), uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetBlockedIDs_Delegates(t *testing.T) {
	// given
	svc, blockRepo, _, _ := newTestService(t)
	userID := uuid.New()
	want := []uuid.UUID{uuid.New(), uuid.New()}
	blockRepo.EXPECT().GetBlockedIDs(mock.Anything, userID).Return(want, nil)

	// when
	got, err := svc.GetBlockedIDs(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetBlockedIDs_RepoError(t *testing.T) {
	// given
	svc, blockRepo, _, _ := newTestService(t)
	userID := uuid.New()
	blockRepo.EXPECT().GetBlockedIDs(mock.Anything, userID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetBlockedIDs(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestGetBlockedUsers_Delegates(t *testing.T) {
	// given
	svc, blockRepo, _, _ := newTestService(t)
	blocker := uuid.New()
	want := []repository.BlockedUser{
		{ID: uuid.New(), Username: "alice"},
		{ID: uuid.New(), Username: "bob"},
	}
	blockRepo.EXPECT().GetBlockedUsers(mock.Anything, blocker).Return(want, nil)

	// when
	got, err := svc.GetBlockedUsers(context.Background(), blocker)

	// then
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetBlockedUsers_RepoError(t *testing.T) {
	// given
	svc, blockRepo, _, _ := newTestService(t)
	blocker := uuid.New()
	blockRepo.EXPECT().GetBlockedUsers(mock.Anything, blocker).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetBlockedUsers(context.Background(), blocker)

	// then
	require.Error(t, err)
}
