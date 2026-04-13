package follow

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (
	*service,
	*repository.MockFollowRepository,
	*repository.MockUserRepository,
	*block.MockService,
	*notification.MockService,
	*settings.MockService,
) {
	followRepo := repository.NewMockFollowRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	svc := NewService(followRepo, userRepo, blockSvc, notifSvc, settingsSvc).(*service)
	return svc, followRepo, userRepo, blockSvc, notifSvc, settingsSvc
}

func TestFollow_CannotFollowSelf(t *testing.T) {
	// given
	svc, _, _, _, _, _ := newTestService(t)
	userID := uuid.New()

	// when
	err := svc.Follow(context.Background(), userID, userID)

	// then
	require.ErrorIs(t, err, ErrCannotFollowSelf)
}

func TestFollow_BlockedEitherDirection(t *testing.T) {
	// given
	svc, _, _, blockSvc, _, _ := newTestService(t)
	follower := uuid.New()
	target := uuid.New()
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, follower, target).Return(true, nil)

	// when
	err := svc.Follow(context.Background(), follower, target)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestFollow_RepoErrorBubbles(t *testing.T) {
	// given
	svc, followRepo, _, blockSvc, _, _ := newTestService(t)
	follower := uuid.New()
	target := uuid.New()
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, follower, target).Return(false, nil)
	followRepo.EXPECT().Follow(mock.Anything, follower, target).Return(errors.New("db down"))

	// when
	err := svc.Follow(context.Background(), follower, target)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "follow")
	assert.Contains(t, err.Error(), "db down")
}

func TestFollow_OK_SendsNotification(t *testing.T) {
	// given
	svc, followRepo, userRepo, blockSvc, notifSvc, settingsSvc := newTestService(t)
	follower := uuid.New()
	target := uuid.New()
	user := &model.User{
		ID:          follower,
		Username:    "alice",
		DisplayName: "Alice",
	}
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, follower, target).Return(false, nil)
	followRepo.EXPECT().Follow(mock.Anything, follower, target).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	userRepo.EXPECT().GetByID(mock.Anything, follower).Return(user, nil)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://example.test")
	notifSvc.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == target &&
			p.Type == dto.NotifNewFollower &&
			p.ReferenceID == follower &&
			p.ActorID == follower &&
			p.ReferenceType == "user"
	})).Run(func(_ context.Context, _ dto.NotifyParams) {
		wg.Done()
	}).Return(nil)

	// when
	err := svc.Follow(context.Background(), follower, target)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestFollow_OK_UserLookupErrorSwallowed(t *testing.T) {
	// given
	svc, followRepo, userRepo, blockSvc, _, _ := newTestService(t)
	follower := uuid.New()
	target := uuid.New()
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, follower, target).Return(false, nil)
	followRepo.EXPECT().Follow(mock.Anything, follower, target).Return(nil)

	done := make(chan struct{})
	userRepo.EXPECT().GetByID(mock.Anything, follower).
		Run(func(_ context.Context, _ uuid.UUID) { close(done) }).
		Return(nil, errors.New("missing"))

	// when
	err := svc.Follow(context.Background(), follower, target)

	// then
	require.NoError(t, err)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("goroutine did not run")
	}
}

func TestFollow_OK_NilUserSwallowed(t *testing.T) {
	// given
	svc, followRepo, userRepo, blockSvc, _, _ := newTestService(t)
	follower := uuid.New()
	target := uuid.New()
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, follower, target).Return(false, nil)
	followRepo.EXPECT().Follow(mock.Anything, follower, target).Return(nil)

	done := make(chan struct{})
	userRepo.EXPECT().GetByID(mock.Anything, follower).
		Run(func(_ context.Context, _ uuid.UUID) { close(done) }).
		Return(nil, nil)

	// when
	err := svc.Follow(context.Background(), follower, target)

	// then
	require.NoError(t, err)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("goroutine did not run")
	}
}

func TestUnfollow_Delegates(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	follower := uuid.New()
	target := uuid.New()
	followRepo.EXPECT().Unfollow(mock.Anything, follower, target).Return(nil)

	// when
	err := svc.Unfollow(context.Background(), follower, target)

	// then
	require.NoError(t, err)
}

func TestUnfollow_RepoError(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	follower := uuid.New()
	target := uuid.New()
	followRepo.EXPECT().Unfollow(mock.Anything, follower, target).Return(errors.New("boom"))

	// when
	err := svc.Unfollow(context.Background(), follower, target)

	// then
	require.Error(t, err)
}

func TestIsFollowing_Delegates(t *testing.T) {
	cases := []struct {
		name string
		ret  bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, followRepo, _, _, _, _ := newTestService(t)
			follower := uuid.New()
			target := uuid.New()
			followRepo.EXPECT().IsFollowing(mock.Anything, follower, target).Return(tc.ret, nil)

			// when
			got, err := svc.IsFollowing(context.Background(), follower, target)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.ret, got)
		})
	}
}

func TestIsFollowing_RepoError(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	follower := uuid.New()
	target := uuid.New()
	followRepo.EXPECT().IsFollowing(mock.Anything, follower, target).Return(false, errors.New("boom"))

	// when
	_, err := svc.IsFollowing(context.Background(), follower, target)

	// then
	require.Error(t, err)
}

func TestGetFollowStats_FollowerCountError(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	followRepo.EXPECT().GetFollowerCount(mock.Anything, userID).Return(0, errors.New("boom"))

	// when
	_, err := svc.GetFollowStats(context.Background(), userID, uuid.Nil)

	// then
	require.Error(t, err)
}

func TestGetFollowStats_FollowingCountError(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	followRepo.EXPECT().GetFollowerCount(mock.Anything, userID).Return(3, nil)
	followRepo.EXPECT().GetFollowingCount(mock.Anything, userID).Return(0, errors.New("boom"))

	// when
	_, err := svc.GetFollowStats(context.Background(), userID, uuid.Nil)

	// then
	require.Error(t, err)
}

func TestGetFollowStats_NilViewerSkipsRelations(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	followRepo.EXPECT().GetFollowerCount(mock.Anything, userID).Return(5, nil)
	followRepo.EXPECT().GetFollowingCount(mock.Anything, userID).Return(7, nil)

	// when
	got, err := svc.GetFollowStats(context.Background(), userID, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 5, got.FollowerCount)
	assert.Equal(t, 7, got.FollowingCount)
	assert.False(t, got.IsFollowing)
	assert.False(t, got.FollowsYou)
}

func TestGetFollowStats_SameViewerSkipsRelations(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	followRepo.EXPECT().GetFollowerCount(mock.Anything, userID).Return(1, nil)
	followRepo.EXPECT().GetFollowingCount(mock.Anything, userID).Return(2, nil)

	// when
	got, err := svc.GetFollowStats(context.Background(), userID, userID)

	// then
	require.NoError(t, err)
	assert.False(t, got.IsFollowing)
	assert.False(t, got.FollowsYou)
}

func TestGetFollowStats_WithViewerPopulatesRelations(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	viewer := uuid.New()
	followRepo.EXPECT().GetFollowerCount(mock.Anything, userID).Return(10, nil)
	followRepo.EXPECT().GetFollowingCount(mock.Anything, userID).Return(20, nil)
	followRepo.EXPECT().IsFollowing(mock.Anything, viewer, userID).Return(true, nil)
	followRepo.EXPECT().IsFollowing(mock.Anything, userID, viewer).Return(false, nil)

	// when
	got, err := svc.GetFollowStats(context.Background(), userID, viewer)

	// then
	require.NoError(t, err)
	assert.Equal(t, 10, got.FollowerCount)
	assert.Equal(t, 20, got.FollowingCount)
	assert.True(t, got.IsFollowing)
	assert.False(t, got.FollowsYou)
}

func TestGetFollowStats_RelationErrorsSwallowed(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	viewer := uuid.New()
	followRepo.EXPECT().GetFollowerCount(mock.Anything, userID).Return(0, nil)
	followRepo.EXPECT().GetFollowingCount(mock.Anything, userID).Return(0, nil)
	followRepo.EXPECT().IsFollowing(mock.Anything, viewer, userID).Return(false, errors.New("boom"))
	followRepo.EXPECT().IsFollowing(mock.Anything, userID, viewer).Return(false, errors.New("boom"))

	// when
	got, err := svc.GetFollowStats(context.Background(), userID, viewer)

	// then
	require.NoError(t, err)
	assert.False(t, got.IsFollowing)
	assert.False(t, got.FollowsYou)
}

func TestGetFollowers_OK(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	u1 := repository.FollowUser{ID: uuid.New(), Username: "alice", DisplayName: "Alice", AvatarURL: "a.png", Role: "user"}
	u2 := repository.FollowUser{ID: uuid.New(), Username: "bob", DisplayName: "Bob", AvatarURL: "b.png", Role: "admin"}
	followRepo.EXPECT().GetFollowers(mock.Anything, userID, 10, 0).Return([]repository.FollowUser{u1, u2}, 2, nil)

	// when
	got, total, err := svc.GetFollowers(context.Background(), userID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, got, 2)
	assert.Equal(t, u1.ID, got[0].ID)
	assert.Equal(t, "alice", got[0].Username)
	assert.Equal(t, role.Role("user"), got[0].Role)
	assert.Equal(t, role.Role("admin"), got[1].Role)
}

func TestGetFollowers_RepoError(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	followRepo.EXPECT().GetFollowers(mock.Anything, userID, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, _, err := svc.GetFollowers(context.Background(), userID, 10, 0)

	// then
	require.Error(t, err)
}

func TestGetFollowing_OK(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	u := repository.FollowUser{ID: uuid.New(), Username: "carol", DisplayName: "Carol", Role: "moderator"}
	followRepo.EXPECT().GetFollowing(mock.Anything, userID, 5, 10).Return([]repository.FollowUser{u}, 1, nil)

	// when
	got, total, err := svc.GetFollowing(context.Background(), userID, 5, 10)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, got, 1)
	assert.Equal(t, u.ID, got[0].ID)
	assert.Equal(t, role.Role("moderator"), got[0].Role)
}

func TestGetFollowing_RepoError(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	followRepo.EXPECT().GetFollowing(mock.Anything, userID, 5, 10).Return(nil, 0, errors.New("boom"))

	// when
	_, _, err := svc.GetFollowing(context.Background(), userID, 5, 10)

	// then
	require.Error(t, err)
}

func TestGetMutualFollowers_OK(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	u1 := repository.FollowUser{ID: uuid.New(), Username: "alice"}
	u2 := repository.FollowUser{ID: uuid.New(), Username: "bob"}
	followRepo.EXPECT().GetMutualFollowers(mock.Anything, userID).Return([]repository.FollowUser{u1, u2}, nil)

	// when
	got, err := svc.GetMutualFollowers(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, u1.ID, got[0].ID)
	assert.Equal(t, u2.ID, got[1].ID)
}

func TestGetMutualFollowers_RepoError(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	followRepo.EXPECT().GetMutualFollowers(mock.Anything, userID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetMutualFollowers(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestGetMutualFollowers_Empty(t *testing.T) {
	// given
	svc, followRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	followRepo.EXPECT().GetMutualFollowers(mock.Anything, userID).Return(nil, nil)

	// when
	got, err := svc.GetMutualFollowers(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Empty(t, got)
}

func waitOrFail(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("timed out waiting for goroutine")
	}
}
