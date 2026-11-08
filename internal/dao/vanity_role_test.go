package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVanityRoleDAO_List_SeedsSystemRoles(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	roles, err := repos.VanityRole.List(context.Background())

	// then
	require.NoError(t, err)

	ids := make([]string, len(roles))
	for i := range roles {
		ids[i] = roles[i].ID
		assert.True(t, roles[i].IsSystem, "every seeded role must be a system role")
	}

	assert.Equal(t, []string{
		"system_top_detective",
		"system_top_gm",
		"system_top_chess",
		"system_top_checkers",
		"system_top_othello",
		"system_top_minesweeper",
		"bot",
		"system_witch_hunter",
	}, ids)
	assert.Equal(t, 0, roles[0].SortOrder)
}

func TestVanityRoleDAO_Create_AndGetByID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	err := repos.VanityRole.Create(ctx, spec.NewVanityRole{ID: "vip", Label: "VIP", Color: "#ff00ff", SortOrder: 5})

	// then
	require.NoError(t, err)
	row, err := repos.VanityRole.GetByID(ctx, "vip")
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "vip", row.ID)
	assert.Equal(t, "VIP", row.Label)
	assert.Equal(t, "#ff00ff", row.Color)
	assert.False(t, row.IsSystem)
	assert.Equal(t, 5, row.SortOrder)
}

func TestVanityRoleDAO_GetByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	row, err := repos.VanityRole.GetByID(context.Background(), "missing")

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestVanityRoleDAO_Update(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	require.NoError(t, repos.VanityRole.Create(ctx, spec.NewVanityRole{ID: "mod", Label: "Mod", Color: "#111111", SortOrder: 1}))

	// when
	err := repos.VanityRole.Update(ctx, spec.VanityRoleUpdate{ID: "mod", Label: "Moderator", Color: "#222222", SortOrder: 9})

	// then
	require.NoError(t, err)
	row, err := repos.VanityRole.GetByID(ctx, "mod")
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Moderator", row.Label)
	assert.Equal(t, "#222222", row.Color)
	assert.Equal(t, 9, row.SortOrder)
}

func TestVanityRoleDAO_Delete_NonSystem(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	require.NoError(t, repos.VanityRole.Create(ctx, spec.NewVanityRole{ID: "temp", Label: "Temp", Color: "#000000", SortOrder: 0}))

	// when
	err := repos.VanityRole.Delete(ctx, "temp")

	// then
	require.NoError(t, err)
	row, err := repos.VanityRole.GetByID(ctx, "temp")
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestVanityRoleDAO_Delete_SystemIsProtected(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	err := repos.VanityRole.Delete(ctx, "system_top_detective")

	// then
	require.NoError(t, err)
	row, err := repos.VanityRole.GetByID(ctx, "system_top_detective")
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsSystem)
}

func TestVanityRoleDAO_List_OrdersBySortOrderThenLabel(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	require.NoError(t, repos.VanityRole.Create(ctx, spec.NewVanityRole{ID: "b", Label: "Beta", Color: "#000000", SortOrder: 10}))
	require.NoError(t, repos.VanityRole.Create(ctx, spec.NewVanityRole{ID: "a", Label: "Alpha", Color: "#000000", SortOrder: 10}))
	require.NoError(t, repos.VanityRole.Create(ctx, spec.NewVanityRole{ID: "c", Label: "Charlie", Color: "#000000", SortOrder: 5}))

	// when
	roles, err := repos.VanityRole.List(ctx)

	// then
	require.NoError(t, err)

	ids := make([]string, len(roles))
	for i := range roles {
		ids[i] = roles[i].ID
	}

	assert.Equal(t, []string{
		"system_top_detective",
		"system_top_gm",
		"system_top_chess",
		"system_top_checkers",
		"system_top_othello",
		"c",
		"system_top_minesweeper",
		"a",
		"b",
		"bot",
		"system_witch_hunter",
	}, ids, "roles sort by sort_order then label")
}

func TestVanityRoleDAO_GetRolesForUser_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	roles, err := repos.VanityRole.GetRolesForUser(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Empty(t, roles)
}

func TestVanityRoleDAO_GetUsersForRole_PaginatesAndOrders(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice"), daotest.WithDisplayName("Alice"))
	bob := daotest.CreateUser(t, repos, daotest.WithUsername("bob"), daotest.WithDisplayName("Bob"))
	carol := daotest.CreateUser(t, repos, daotest.WithUsername("carol"), daotest.WithDisplayName("Carol"))
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: alice.ID, RoleID: "system_top_detective"}))
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: bob.ID, RoleID: "system_top_detective"}))
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: carol.ID, RoleID: "system_top_detective"}))

	// when
	page1, total, err := repos.VanityRole.GetUsersForRole(ctx, spec.VanityRoleUserQuery{RoleID: "system_top_detective", Limit: 2, Offset: 0})

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, page1, 2)
	assert.Equal(t, "Alice", page1[0].DisplayName)
	assert.Equal(t, "Bob", page1[1].DisplayName)

	page2, _, err := repos.VanityRole.GetUsersForRole(ctx, spec.VanityRoleUserQuery{RoleID: "system_top_detective", Limit: 2, Offset: 2})
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, "Carol", page2[0].DisplayName)
}

func TestVanityRoleDAO_GetUsersForRole_SearchFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice123"), daotest.WithDisplayName("Alice"))
	bob := daotest.CreateUser(t, repos, daotest.WithUsername("bob"), daotest.WithDisplayName("Bobson"))
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: alice.ID, RoleID: "system_top_gm"}))
	require.NoError(t, repos.Permission.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: bob.ID, RoleID: "system_top_gm"}))

	// when
	users, total, err := repos.VanityRole.GetUsersForRole(ctx, spec.VanityRoleUserQuery{RoleID: "system_top_gm", Search: "ali", Limit: 10, Offset: 0})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, users, 1)
	assert.Equal(t, alice.ID, users[0].UserID)
	assert.Equal(t, "alice123", users[0].Username)
}

func TestVanityRoleDAO_GetUsersForRole_NoResults(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	users, total, err := repos.VanityRole.GetUsersForRole(context.Background(), spec.VanityRoleUserQuery{RoleID: "system_top_gm", Limit: 10, Offset: 0})

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, users)
}

func TestExcludeVanityRoleIDs_Empty(t *testing.T) {
	// given
	var ids []string

	// when
	clause, args := repository.ExcludeVanityRoleIDs(ids, 1)

	// then
	assert.Empty(t, clause)
	assert.Nil(t, args)
}

func TestExcludeVanityRoleIDs_BuildsClause(t *testing.T) {
	// given
	ids := []string{"a", "b", "c"}

	// when
	clause, args := repository.ExcludeVanityRoleIDs(ids, 1)

	// then
	assert.Equal(t, " AND id NOT IN ($1, $2, $3)", clause)
	assert.Equal(t, []any{"a", "b", "c"}, args)
}
