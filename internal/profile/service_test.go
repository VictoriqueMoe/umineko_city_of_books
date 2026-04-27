package profile

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (
	*service,
	*repository.MockUserRepository,
	*repository.MockTheoryRepository,
	*authz.MockService,
	*upload.MockService,
	*settings.MockService,
) {
	userRepo := repository.NewMockUserRepository(t)
	userSecretRepo := repository.NewMockUserSecretRepository(t)
	userSecretRepo.EXPECT().ListForUser(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	theoryRepo := repository.NewMockTheoryRepository(t)
	authzSvc := authz.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	svc := NewService(userRepo, userSecretRepo, theoryRepo, authzSvc, uploadSvc, settingsSvc, contentfilter.New()).(*service)
	return svc, userRepo, theoryRepo, authzSvc, uploadSvc, settingsSvc
}

func TestGetProfile_OK(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	viewerID := uuid.New()
	user := &model.User{ID: userID, Username: "alice", DisplayName: "Alice"}
	stats := &model.UserStats{TheoryCount: 5}
	userRepo.EXPECT().GetProfileByUsername(mock.Anything, "alice").Return(user, stats, nil)

	// when
	got, err := svc.GetProfile(context.Background(), "alice", viewerID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "alice", got.Username)
	assert.Equal(t, 5, got.Stats.TheoryCount)
}

func TestGetProfile_SelfViewIncludesPrivate(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	user := &model.User{
		ID:                 userID,
		Username:           "alice",
		DOB:                "2000-04-15",
		DOBPublic:          false,
		Email:              "a@x.com",
		EmailNotifications: true,
		HomePage:           "home",
	}
	stats := &model.UserStats{}
	userRepo.EXPECT().GetProfileByUsername(mock.Anything, "alice").Return(user, stats, nil)

	// when
	got, err := svc.GetProfile(context.Background(), "alice", userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "2000-04-15", got.DOB)
	assert.False(t, got.DOBPublic)
	assert.Equal(t, "a@x.com", got.Email)
	assert.True(t, got.EmailNotifications)
	assert.Equal(t, "home", got.HomePage)
}

func TestGetProfile_NonSelfHidesPrivateDOB(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	user := &model.User{
		ID:        uuid.New(),
		Username:  "alice",
		DOB:       "2000-04-15",
		DOBPublic: false,
	}
	stats := &model.UserStats{}
	userRepo.EXPECT().GetProfileByUsername(mock.Anything, "alice").Return(user, stats, nil)

	// when
	got, err := svc.GetProfile(context.Background(), "alice", uuid.New())

	// then
	require.NoError(t, err)
	assert.Empty(t, got.DOB)
	assert.False(t, got.DOBPublic)
}

func TestGetProfile_NotFound(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userRepo.EXPECT().GetProfileByUsername(mock.Anything, "ghost").Return(nil, nil, nil)

	// when
	_, err := svc.GetProfile(context.Background(), "ghost", uuid.New())

	// then
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestGetProfile_RepoError(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userRepo.EXPECT().GetProfileByUsername(mock.Anything, "alice").Return(nil, nil, errors.New("db down"))

	// when
	_, err := svc.GetProfile(context.Background(), "alice", uuid.New())

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get profile")
}

func TestUpdateProfile_OK(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	req := dto.UpdateProfileRequest{DisplayName: "New Name"}
	userRepo.EXPECT().UpdateProfile(mock.Anything, userID, req).Return(nil)

	// when
	err := svc.UpdateProfile(context.Background(), userID, req)

	// then
	require.NoError(t, err)
}

func TestUpdateProfile_InvalidDOBFormat(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	req := dto.UpdateProfileRequest{DisplayName: "New Name", DOB: "15-04-2000"}

	// when
	err := svc.UpdateProfile(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrInvalidDOB)
	userRepo.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateProfile_FutureDOB(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	future := time.Now().UTC().AddDate(1, 0, 0).Format("2006-01-02")
	req := dto.UpdateProfileRequest{DisplayName: "New Name", DOB: future}

	// when
	err := svc.UpdateProfile(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrFutureDOB)
	userRepo.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateProfile_RepoError(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	req := dto.UpdateProfileRequest{DisplayName: "New Name"}
	userRepo.EXPECT().UpdateProfile(mock.Anything, userID, req).Return(errors.New("boom"))

	// when
	err := svc.UpdateProfile(context.Background(), userID, req)

	// then
	require.Error(t, err)
}

func TestUploadAvatar_OK(t *testing.T) {
	// given
	svc, userRepo, _, _, uploadSvc, settingsSvc := newTestService(t)
	userID := uuid.New()
	reader := strings.NewReader("img")
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1024)
	uploadSvc.EXPECT().SaveImage(mock.Anything, "avatars", userID, int64(3), int64(1024), reader).Return("/avatars/a.png", nil)
	userRepo.EXPECT().UpdateAvatarURL(mock.Anything, userID, "/avatars/a.png").Return(nil)

	// when
	got, err := svc.UploadAvatar(context.Background(), userID, "image/png", 3, reader)

	// then
	require.NoError(t, err)
	assert.Equal(t, "/avatars/a.png", got)
}

func TestUploadAvatar_UploadError(t *testing.T) {
	// given
	svc, _, _, _, uploadSvc, settingsSvc := newTestService(t)
	userID := uuid.New()
	reader := strings.NewReader("img")
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1024)
	uploadSvc.EXPECT().SaveImage(mock.Anything, "avatars", userID, int64(3), int64(1024), reader).Return("", errors.New("too big"))

	// when
	got, err := svc.UploadAvatar(context.Background(), userID, "image/png", 3, reader)

	// then
	require.Error(t, err)
	assert.Empty(t, got)
}

func TestUploadAvatar_UpdateRepoError(t *testing.T) {
	// given
	svc, userRepo, _, _, uploadSvc, settingsSvc := newTestService(t)
	userID := uuid.New()
	reader := strings.NewReader("img")
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1024)
	uploadSvc.EXPECT().SaveImage(mock.Anything, "avatars", userID, int64(3), int64(1024), reader).Return("/avatars/a.png", nil)
	userRepo.EXPECT().UpdateAvatarURL(mock.Anything, userID, "/avatars/a.png").Return(errors.New("db down"))

	// when
	got, err := svc.UploadAvatar(context.Background(), userID, "image/png", 3, reader)

	// then
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), "update avatar url")
}

func TestUploadBanner_OK(t *testing.T) {
	// given
	svc, userRepo, _, _, uploadSvc, settingsSvc := newTestService(t)
	userID := uuid.New()
	reader := strings.NewReader("img")
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(2048)
	uploadSvc.EXPECT().SaveImage(mock.Anything, "banners", userID, int64(3), int64(2048), reader).Return("/banners/b.jpg", nil)
	userRepo.EXPECT().UpdateBannerURL(mock.Anything, userID, "/banners/b.jpg").Return(nil)

	// when
	got, err := svc.UploadBanner(context.Background(), userID, "image/jpeg", 3, reader)

	// then
	require.NoError(t, err)
	assert.Equal(t, "/banners/b.jpg", got)
}

func TestUploadBanner_UploadError(t *testing.T) {
	// given
	svc, _, _, _, uploadSvc, settingsSvc := newTestService(t)
	userID := uuid.New()
	reader := strings.NewReader("img")
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(2048)
	uploadSvc.EXPECT().SaveImage(mock.Anything, "banners", userID, int64(3), int64(2048), reader).Return("", errors.New("bad type"))

	// when
	got, err := svc.UploadBanner(context.Background(), userID, "image/jpeg", 3, reader)

	// then
	require.Error(t, err)
	assert.Empty(t, got)
}

func TestUploadBanner_UpdateRepoError(t *testing.T) {
	// given
	svc, userRepo, _, _, uploadSvc, settingsSvc := newTestService(t)
	userID := uuid.New()
	reader := strings.NewReader("img")
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(2048)
	uploadSvc.EXPECT().SaveImage(mock.Anything, "banners", userID, int64(3), int64(2048), reader).Return("/banners/b.jpg", nil)
	userRepo.EXPECT().UpdateBannerURL(mock.Anything, userID, "/banners/b.jpg").Return(errors.New("db down"))

	// when
	got, err := svc.UploadBanner(context.Background(), userID, "image/jpeg", 3, reader)

	// then
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), "update banner url")
}

func TestChangePassword_TooShort(t *testing.T) {
	// given
	svc, _, _, _, _, settingsSvc := newTestService(t)
	userID := uuid.New()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMinPasswordLength).Return(8)

	// when
	err := svc.ChangePassword(context.Background(), userID, dto.ChangePasswordRequest{OldPassword: "old", NewPassword: "short"})

	// then
	require.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestChangePassword_MinLenZeroSkipsValidation(t *testing.T) {
	// given
	svc, userRepo, _, _, _, settingsSvc := newTestService(t)
	userID := uuid.New()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMinPasswordLength).Return(0)
	userRepo.EXPECT().ChangePassword(mock.Anything, userID, "old", "x").Return(nil)

	// when
	err := svc.ChangePassword(context.Background(), userID, dto.ChangePasswordRequest{OldPassword: "old", NewPassword: "x"})

	// then
	require.NoError(t, err)
}

func TestChangePassword_OK(t *testing.T) {
	// given
	svc, userRepo, _, _, _, settingsSvc := newTestService(t)
	userID := uuid.New()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMinPasswordLength).Return(4)
	userRepo.EXPECT().ChangePassword(mock.Anything, userID, "oldpass", "newpass").Return(nil)

	// when
	err := svc.ChangePassword(context.Background(), userID, dto.ChangePasswordRequest{OldPassword: "oldpass", NewPassword: "newpass"})

	// then
	require.NoError(t, err)
}

func TestChangePassword_RepoError(t *testing.T) {
	// given
	svc, userRepo, _, _, _, settingsSvc := newTestService(t)
	userID := uuid.New()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMinPasswordLength).Return(4)
	userRepo.EXPECT().ChangePassword(mock.Anything, userID, "oldpass", "newpass").Return(errors.New("wrong old"))

	// when
	err := svc.ChangePassword(context.Background(), userID, dto.ChangePasswordRequest{OldPassword: "oldpass", NewPassword: "newpass"})

	// then
	require.Error(t, err)
}

func TestDeleteAccount_OK_CleansUpUploads(t *testing.T) {
	// given
	svc, userRepo, _, _, uploadSvc, _ := newTestService(t)
	userID := uuid.New()
	user := &model.User{ID: userID, AvatarURL: "/avatars/a.png", BannerURL: "/banners/b.jpg"}
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	userRepo.EXPECT().DeleteAccount(mock.Anything, userID, "pw").Return(nil)
	uploadSvc.EXPECT().Delete("/avatars/a.png").Return(nil)
	uploadSvc.EXPECT().Delete("/banners/b.jpg").Return(nil)

	// when
	err := svc.DeleteAccount(context.Background(), userID, dto.DeleteAccountRequest{Password: "pw"})

	// then
	require.NoError(t, err)
}

func TestDeleteAccount_UploadDeleteErrorsSwallowed(t *testing.T) {
	// given
	svc, userRepo, _, _, uploadSvc, _ := newTestService(t)
	userID := uuid.New()
	user := &model.User{ID: userID, AvatarURL: "/avatars/a.png", BannerURL: "/banners/b.jpg"}
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	userRepo.EXPECT().DeleteAccount(mock.Anything, userID, "pw").Return(nil)
	uploadSvc.EXPECT().Delete("/avatars/a.png").Return(errors.New("gone"))
	uploadSvc.EXPECT().Delete("/banners/b.jpg").Return(errors.New("gone"))

	// when
	err := svc.DeleteAccount(context.Background(), userID, dto.DeleteAccountRequest{Password: "pw"})

	// then
	require.NoError(t, err)
}

func TestDeleteAccount_NilUserSkipsCleanup(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, nil)
	userRepo.EXPECT().DeleteAccount(mock.Anything, userID, "pw").Return(nil)

	// when
	err := svc.DeleteAccount(context.Background(), userID, dto.DeleteAccountRequest{Password: "pw"})

	// then
	require.NoError(t, err)
}

func TestDeleteAccount_GetByIDError(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("db down"))

	// when
	err := svc.DeleteAccount(context.Background(), userID, dto.DeleteAccountRequest{Password: "pw"})

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get user for cleanup")
}

func TestDeleteAccount_DeleteRepoError(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userID := uuid.New()
	user := &model.User{ID: userID}
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	userRepo.EXPECT().DeleteAccount(mock.Anything, userID, "pw").Return(errors.New("wrong password"))

	// when
	err := svc.DeleteAccount(context.Background(), userID, dto.DeleteAccountRequest{Password: "pw"})

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "wrong password")
}

func TestGetActivity_OK(t *testing.T) {
	// given
	svc, userRepo, theoryRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	user := &model.User{ID: userID, Username: "alice"}
	items := []dto.ActivityItem{{Type: "theory", TheoryTitle: "Foo"}}
	userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(user, nil)
	theoryRepo.EXPECT().GetRecentActivityByUser(mock.Anything, userID, 10, 0).Return(items, 1, nil)

	// when
	got, err := svc.GetActivity(context.Background(), "alice", 10, 0)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, items, got.Items)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 0, got.Offset)
}

func TestGetActivity_UserNotFound(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userRepo.EXPECT().GetByUsername(mock.Anything, "ghost").Return(nil, nil)

	// when
	_, err := svc.GetActivity(context.Background(), "ghost", 10, 0)

	// then
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestGetActivity_GetByUsernameError(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(nil, errors.New("db down"))

	// when
	_, err := svc.GetActivity(context.Background(), "alice", 10, 0)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get user")
}

func TestGetActivity_TheoryRepoError(t *testing.T) {
	// given
	svc, userRepo, theoryRepo, _, _, _ := newTestService(t)
	userID := uuid.New()
	user := &model.User{ID: userID, Username: "alice"}
	userRepo.EXPECT().GetByUsername(mock.Anything, "alice").Return(user, nil)
	theoryRepo.EXPECT().GetRecentActivityByUser(mock.Anything, userID, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.GetActivity(context.Background(), "alice", 10, 0)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get activity")
}

func TestListPublicUsers_OK(t *testing.T) {
	// given
	svc, userRepo, _, authzSvc, _, _ := newTestService(t)
	id1 := uuid.New()
	id2 := uuid.New()
	users := []model.User{
		{ID: id1, Username: "alice", DisplayName: "Alice", AvatarURL: "/a.png"},
		{ID: id2, Username: "bob", DisplayName: "Bob"},
	}
	userRepo.EXPECT().ListPublic(mock.Anything).Return(users, nil)
	authzSvc.EXPECT().GetRoles(mock.Anything, mock.Anything).Return(map[uuid.UUID]role.Role{
		id1: authz.RoleAdmin,
	}, nil)

	// when
	got, err := svc.ListPublicUsers(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "alice", got[0].Username)
	assert.Equal(t, authz.RoleAdmin, got[0].Role)
	assert.Equal(t, "bob", got[1].Username)
	assert.Equal(t, role.Role(""), got[1].Role)
}

func TestListPublicUsers_EmptyList(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userRepo.EXPECT().ListPublic(mock.Anything).Return([]model.User{}, nil)

	// when
	got, err := svc.ListPublicUsers(context.Background())

	// then
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestListPublicUsers_RepoError(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userRepo.EXPECT().ListPublic(mock.Anything).Return(nil, errors.New("db down"))

	// when
	_, err := svc.ListPublicUsers(context.Background())

	// then
	require.Error(t, err)
}

func TestSearchUsers_OK(t *testing.T) {
	// given
	svc, userRepo, _, authzSvc, _, _ := newTestService(t)
	id1 := uuid.New()
	users := []model.User{{ID: id1, Username: "alice", DisplayName: "Alice"}}
	userRepo.EXPECT().SearchByName(mock.Anything, "ali", 5).Return(users, nil)
	authzSvc.EXPECT().GetRoles(mock.Anything, mock.Anything).Return(map[uuid.UUID]role.Role{
		id1: authz.RoleModerator,
	}, nil)

	// when
	got, err := svc.SearchUsers(context.Background(), "ali", 5)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "alice", got[0].Username)
	assert.Equal(t, authz.RoleModerator, got[0].Role)
}

func TestSearchUsers_RepoError(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userRepo.EXPECT().SearchByName(mock.Anything, "ali", 5).Return(nil, errors.New("db down"))

	// when
	_, err := svc.SearchUsers(context.Background(), "ali", 5)

	// then
	require.Error(t, err)
}

func TestSearchUsers_EmptyResult(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userRepo.EXPECT().SearchByName(mock.Anything, "zzz", 5).Return([]model.User{}, nil)

	// when
	got, err := svc.SearchUsers(context.Background(), "zzz", 5)

	// then
	require.NoError(t, err)
	assert.Empty(t, got)
}
