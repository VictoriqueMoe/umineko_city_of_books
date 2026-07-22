package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
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

func TestRoleDAO_GetRole_NoRole(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	got, err := repos.Role.GetRole(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, role.Role(""), got)
}

func TestRoleDAO_SetAndGetRole(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.Role.SetRole(context.Background(), user.ID, roleAdmin)

	// then
	require.NoError(t, err)
	got, err := repos.Role.GetRole(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, roleAdmin, got)
}

func TestRoleDAO_GetRoles(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	c := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Role.SetRole(context.Background(), a.ID, roleAdmin))
	require.NoError(t, repos.Role.SetRole(context.Background(), b.ID, roleModerator))

	// when
	got, err := repos.Role.GetRoles(context.Background(), []uuid.UUID{a.ID, b.ID, c.ID})

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, roleAdmin, got[a.ID])
	assert.Equal(t, roleModerator, got[b.ID])
	_, hasC := got[c.ID]
	assert.False(t, hasC, "user c has no role and should be absent from the map")
}

func TestRoleDAO_GetRoles_EmptyInput(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.Role.GetRoles(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRoleDAO_SetRole_ReplacesExisting(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestRoleDAO_HasRole(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestRoleDAO_HasRole_NoRoleAssigned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	has, err := repos.Role.HasRole(context.Background(), user.ID, roleAdmin)

	// then
	require.NoError(t, err)
	assert.False(t, has)
}

func TestRoleDAO_RemoveRole(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Role.SetRole(context.Background(), user.ID, roleAdmin))

	// when
	err := repos.Role.RemoveRole(context.Background(), user.ID, roleAdmin)

	// then
	require.NoError(t, err)
	got, err := repos.Role.GetRole(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, role.Role(""), got)
}

func TestRoleDAO_RemoveRole_NotPresent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.Role.RemoveRole(context.Background(), user.ID, roleAdmin)

	// then
	require.NoError(t, err)
}

func TestRoleDAO_GetUsersByRoles_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	users, err := repos.Role.GetUsersByRoles(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, users)
}

func TestRoleDAO_GetUsersByRoles_SingleRole(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	admin := daotest.CreateUser(t, repos)
	mod := daotest.CreateUser(t, repos)
	plain := daotest.CreateUser(t, repos)
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

func TestRoleDAO_GetUsersByRoles_MultipleRoles(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	admin := daotest.CreateUser(t, repos)
	mod := daotest.CreateUser(t, repos)
	editor := daotest.CreateUser(t, repos)
	plain := daotest.CreateUser(t, repos)
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
