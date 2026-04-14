package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	roleAdmin     role.Role = "admin"
	roleModerator role.Role = "moderator"
	roleEditor    role.Role = "editor"
)

func TestRoleRepository_GetRole_NoRole(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	got, err := repos.Role.GetRole(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, role.Role(""), got)
}

func TestRoleRepository_SetAndGetRole(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.Role.SetRole(context.Background(), user.ID, roleAdmin)

	// then
	require.NoError(t, err)
	got, err := repos.Role.GetRole(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, roleAdmin, got)
}

func TestRoleRepository_SetRole_ReplacesExisting(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Role.SetRole(context.Background(), user.ID, roleAdmin))

	// when
	err := repos.Role.SetRole(context.Background(), user.ID, roleModerator)

	// then
	require.NoError(t, err)
	got, err := repos.Role.GetRole(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, roleModerator, got)
	hasOld, err := repos.Role.HasRole(context.Background(), user.ID, roleAdmin)
	require.NoError(t, err)
	assert.False(t, hasOld)
}

func TestRoleRepository_HasRole(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Role.SetRole(context.Background(), user.ID, roleAdmin))

	// when
	hasAdmin, errA := repos.Role.HasRole(context.Background(), user.ID, roleAdmin)
	hasModerator, errM := repos.Role.HasRole(context.Background(), user.ID, roleModerator)

	// then
	require.NoError(t, errA)
	require.NoError(t, errM)
	assert.True(t, hasAdmin)
	assert.False(t, hasModerator)
}

func TestRoleRepository_HasRole_NoRoleAssigned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	has, err := repos.Role.HasRole(context.Background(), user.ID, roleAdmin)

	// then
	require.NoError(t, err)
	assert.False(t, has)
}

func TestRoleRepository_RemoveRole(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Role.SetRole(context.Background(), user.ID, roleAdmin))

	// when
	err := repos.Role.RemoveRole(context.Background(), user.ID, roleAdmin)

	// then
	require.NoError(t, err)
	got, err := repos.Role.GetRole(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, role.Role(""), got)
}

func TestRoleRepository_RemoveRole_NotPresent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.Role.RemoveRole(context.Background(), user.ID, roleAdmin)

	// then
	require.NoError(t, err)
}

func TestRoleRepository_GetUsersByRoles_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	users, err := repos.Role.GetUsersByRoles(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, users)
}

func TestRoleRepository_GetUsersByRoles_SingleRole(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	admin := repotest.CreateUser(t, repos)
	mod := repotest.CreateUser(t, repos)
	plain := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Role.SetRole(context.Background(), admin.ID, roleAdmin))
	require.NoError(t, repos.Role.SetRole(context.Background(), mod.ID, roleModerator))

	// when
	users, err := repos.Role.GetUsersByRoles(context.Background(), []role.Role{roleAdmin})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{admin.ID}, users)
	assert.NotContains(t, users, mod.ID)
	assert.NotContains(t, users, plain.ID)
}

func TestRoleRepository_GetUsersByRoles_MultipleRoles(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	admin := repotest.CreateUser(t, repos)
	mod := repotest.CreateUser(t, repos)
	editor := repotest.CreateUser(t, repos)
	plain := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Role.SetRole(context.Background(), admin.ID, roleAdmin))
	require.NoError(t, repos.Role.SetRole(context.Background(), mod.ID, roleModerator))
	require.NoError(t, repos.Role.SetRole(context.Background(), editor.ID, roleEditor))

	// when
	users, err := repos.Role.GetUsersByRoles(context.Background(), []role.Role{roleAdmin, roleModerator})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{admin.ID, mod.ID}, users)
	assert.NotContains(t, users, editor.ID)
	assert.NotContains(t, users, plain.ID)
}
