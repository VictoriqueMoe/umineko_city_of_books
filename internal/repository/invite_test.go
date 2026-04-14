package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInviteRepository_CreateAndGetByCode(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	code := "invite-" + uuid.NewString()[:8]

	// when
	err := repos.Invite.Create(context.Background(), code, user.ID)

	// then
	require.NoError(t, err)
	inv, err := repos.Invite.GetByCode(context.Background(), code)
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, code, inv.Code)
	assert.Equal(t, user.ID, inv.CreatedBy)
	assert.Nil(t, inv.UsedBy)
	assert.Nil(t, inv.UsedAt)
	assert.NotEmpty(t, inv.CreatedAt)
}

func TestInviteRepository_GetByCode_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	inv, err := repos.Invite.GetByCode(context.Background(), "missing-code")

	// then
	require.NoError(t, err)
	assert.Nil(t, inv)
}

func TestInviteRepository_Create_DuplicateCodeFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	code := "dup-" + uuid.NewString()[:8]
	require.NoError(t, repos.Invite.Create(context.Background(), code, user.ID))

	// when
	err := repos.Invite.Create(context.Background(), code, user.ID)

	// then
	require.Error(t, err)
}

func TestInviteRepository_MarkUsed(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	creator := repotest.CreateUser(t, repos)
	consumer := repotest.CreateUser(t, repos)
	code := "use-" + uuid.NewString()[:8]
	require.NoError(t, repos.Invite.Create(context.Background(), code, creator.ID))

	// when
	err := repos.Invite.MarkUsed(context.Background(), code, consumer.ID)

	// then
	require.NoError(t, err)
	inv, err := repos.Invite.GetByCode(context.Background(), code)
	require.NoError(t, err)
	require.NotNil(t, inv)
	require.NotNil(t, inv.UsedBy)
	assert.Equal(t, consumer.ID, *inv.UsedBy)
	require.NotNil(t, inv.UsedAt)
	assert.NotEmpty(t, *inv.UsedAt)
}

func TestInviteRepository_MarkUsed_UnknownCodeNoError(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.Invite.MarkUsed(context.Background(), "no-such-code", user.ID)

	// then
	require.NoError(t, err)
}

func TestInviteRepository_List(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	codes := []string{"a-" + uuid.NewString()[:6], "b-" + uuid.NewString()[:6], "c-" + uuid.NewString()[:6]}
	for _, c := range codes {
		require.NoError(t, repos.Invite.Create(context.Background(), c, user.ID))
	}

	// when
	invites, total, err := repos.Invite.List(context.Background(), 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, invites, 3)
}

func TestInviteRepository_List_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 5; i++ {
		require.NoError(t, repos.Invite.Create(context.Background(), uuid.NewString(), user.ID))
	}

	// when
	page1, total, err := repos.Invite.List(context.Background(), 2, 0)
	require.NoError(t, err)
	page2, _, err := repos.Invite.List(context.Background(), 2, 2)
	require.NoError(t, err)

	// then
	assert.Equal(t, 5, total)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
	assert.NotEqual(t, page1[0].Code, page2[0].Code)
}

func TestInviteRepository_List_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	invites, total, err := repos.Invite.List(context.Background(), 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, invites)
}

func TestInviteRepository_Delete(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	code := "del-" + uuid.NewString()[:8]
	require.NoError(t, repos.Invite.Create(context.Background(), code, user.ID))

	// when
	err := repos.Invite.Delete(context.Background(), code)

	// then
	require.NoError(t, err)
	inv, err := repos.Invite.GetByCode(context.Background(), code)
	require.NoError(t, err)
	assert.Nil(t, inv)
}

func TestInviteRepository_Delete_UnknownCodeNoError(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Invite.Delete(context.Background(), "ghost")

	// then
	require.NoError(t, err)
}
