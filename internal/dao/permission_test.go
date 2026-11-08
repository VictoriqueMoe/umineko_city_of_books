package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionDAO_AssignVanityRoleToUser_AndGetRolesForUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.VanityRole.Create(ctx, spec.NewVanityRole{ID: "vip", Label: "VIP", Color: "#fff", SortOrder: 3}))

	// when
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: user.ID, RoleID: "vip"}))
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: user.ID, RoleID: "system_top_gm"}))

	// then
	roles, err := repos.VanityRole.GetRolesForUser(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 2)
	assert.Equal(t, "system_top_gm", roles[0].ID)
	assert.Equal(t, "vip", roles[1].ID)
}

func TestPermissionDAO_AssignVanityRoleToUser_DuplicateIgnored(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	// when
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: user.ID, RoleID: "system_top_detective"}))
	err := repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: user.ID, RoleID: "system_top_detective"})

	// then
	require.NoError(t, err)
	roles, err := repos.VanityRole.GetRolesForUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, roles, 1)
}

func TestPermissionDAO_UnassignVanityRoleFromUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: user.ID, RoleID: "system_top_detective"}))
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: user.ID, RoleID: "system_top_gm"}))

	// when
	err := repos.Permission.UnassignVanityRoleFromUser(ctx, spec.VanityRoleAssignment{UserID: user.ID, RoleID: "system_top_detective"})

	// then
	require.NoError(t, err)
	roles, err := repos.VanityRole.GetRolesForUser(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "system_top_gm", roles[0].ID)
}

func TestPermissionDAO_GetAllVanityRoleAssignments(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: a.ID, RoleID: "system_top_detective"}))
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: a.ID, RoleID: "system_top_gm"}))
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: b.ID, RoleID: "system_top_gm"}))

	// when
	assignments, err := repos.Permission.GetAllVanityRoleAssignments(ctx)

	// then
	require.NoError(t, err)
	require.Len(t, assignments, 2)
	assert.ElementsMatch(t, []string{"system_top_detective", "system_top_gm"}, assignments[a.ID.String()])
	assert.ElementsMatch(t, []string{"system_top_gm"}, assignments[b.ID.String()])
}

func TestPermissionDAO_GetAllVanityRoleAssignments_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	assignments, err := repos.Permission.GetAllVanityRoleAssignments(context.Background())

	// then
	require.NoError(t, err)
	assert.Empty(t, assignments)
}

func TestPermissionDAO_MigratedModeratorRowsMatchTheDefaults(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	want := make([]string, 0)
	for _, perm := range authz.DefaultRolePermissions(authz.RoleModerator) {
		want = append(want, string(perm))
	}

	// when
	got, err := repos.Permission.GetRolePermissions(context.Background())

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, want, got[string(authz.RoleModerator)])
}
