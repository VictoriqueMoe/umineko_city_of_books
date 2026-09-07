package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadDAO_GetAllReferencedFiles_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	files, err := repos.Upload.GetAllReferencedFiles()

	// then
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestUploadDAO_GetAllReferencedFiles_FindsAvatar(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	avatarURL := "/uploads/avatars/test-avatar.png"
	require.NoError(t, repos.User.UpdateAvatarURL(context.Background(), spec.UserAvatarUpdate{UserID: user.ID, AvatarURL: avatarURL}))

	// when
	files, err := repos.Upload.GetAllReferencedFiles()

	// then
	require.NoError(t, err)
	assert.Contains(t, files, avatarURL)
}

func TestUploadDAO_GetAllReferencedFiles_IgnoresNonUploadURLs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.UpdateAvatarURL(context.Background(), spec.UserAvatarUpdate{UserID: user.ID, AvatarURL: "https://external.example.com/image.png"}))

	// when
	files, err := repos.Upload.GetAllReferencedFiles()

	// then
	require.NoError(t, err)
	for _, f := range files {
		assert.NotEqual(t, "https://external.example.com/image.png", f)
	}
}

func TestUploadDAO_GetAllReferencedFiles_FindsAcrossColumns(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	avatarURL := "/uploads/avatars/multi-a.png"
	bannerURL := "/uploads/banners/multi-b.png"
	require.NoError(t, repos.User.UpdateAvatarURL(context.Background(), spec.UserAvatarUpdate{UserID: user.ID, AvatarURL: avatarURL}))
	require.NoError(t, repos.User.UpdateBannerURL(context.Background(), spec.UserBannerUpdate{UserID: user.ID, BannerURL: bannerURL}))

	// when
	files, err := repos.Upload.GetAllReferencedFiles()

	// then
	require.NoError(t, err)
	assert.Contains(t, files, avatarURL)
	assert.Contains(t, files, bannerURL)
}

func TestUploadDAO_GetAllReferencedFiles_DistinctPerColumn(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	sharedURL := "/uploads/avatars/shared.png"
	require.NoError(t, repos.User.UpdateAvatarURL(context.Background(), spec.UserAvatarUpdate{UserID: userA.ID, AvatarURL: sharedURL}))
	require.NoError(t, repos.User.UpdateAvatarURL(context.Background(), spec.UserAvatarUpdate{UserID: userB.ID, AvatarURL: sharedURL}))

	// when
	files, err := repos.Upload.GetAllReferencedFiles()

	// then
	require.NoError(t, err)
	count := 0
	for _, f := range files {
		if f == sharedURL {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestUploadDAO_GetAllReferencedFiles_MultipleUsersDistinctURLs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	urlA := "/uploads/avatars/user-a.png"
	urlB := "/uploads/avatars/user-b.png"
	require.NoError(t, repos.User.UpdateAvatarURL(context.Background(), spec.UserAvatarUpdate{UserID: userA.ID, AvatarURL: urlA}))
	require.NoError(t, repos.User.UpdateAvatarURL(context.Background(), spec.UserAvatarUpdate{UserID: userB.ID, AvatarURL: urlB}))

	// when
	files, err := repos.Upload.GetAllReferencedFiles()

	// then
	require.NoError(t, err)
	assert.Contains(t, files, urlA)
	assert.Contains(t, files, urlB)
}
