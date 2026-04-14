package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportRepository_Create(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	id, err := repos.Report.Create(context.Background(), user.ID, "post", "post-1", "ctx-1", "spam")

	// then
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
}

func TestReportRepository_GetByID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Reporter"))
	id, err := repos.Report.Create(context.Background(), user.ID, "post", "post-1", "ctx-1", "abusive")
	require.NoError(t, err)

	// when
	row, err := repos.Report.GetByID(context.Background(), int(id))

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, int(id), row.ID)
	assert.Equal(t, user.ID, row.ReporterID)
	assert.Equal(t, "Reporter", row.ReporterName)
	assert.Equal(t, "post", row.TargetType)
	assert.Equal(t, "post-1", row.TargetID)
	assert.Equal(t, "ctx-1", row.ContextID)
	assert.Equal(t, "abusive", row.Reason)
	assert.Equal(t, "open", row.Status)
	assert.Nil(t, row.ResolvedByID)
	assert.Equal(t, "", row.ResolvedByName)
	assert.NotEmpty(t, row.CreatedAt)
}

func TestReportRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	row, err := repos.Report.GetByID(context.Background(), 999)

	// then
	require.Error(t, err)
	assert.Nil(t, row)
}

func TestReportRepository_List_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	rows, total, err := repos.Report.List(context.Background(), "", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}

func TestReportRepository_List_All(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	_, err1 := repos.Report.Create(ctx, user.ID, "post", "p1", "", "spam")
	_, err2 := repos.Report.Create(ctx, user.ID, "comment", "c1", "", "abuse")
	require.NoError(t, err1)
	require.NoError(t, err2)

	// when
	rows, total, err := repos.Report.List(ctx, "", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
}

func TestReportRepository_List_FilterByStatus(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	resolver := repotest.CreateUser(t, repos)
	ctx := context.Background()
	idOpen, err := repos.Report.Create(ctx, user.ID, "post", "p1", "", "spam")
	require.NoError(t, err)
	idResolved, err := repos.Report.Create(ctx, user.ID, "post", "p2", "", "spam")
	require.NoError(t, err)
	require.NoError(t, repos.Report.Resolve(ctx, int(idResolved), resolver.ID, "handled"))

	// when
	openRows, openTotal, openErr := repos.Report.List(ctx, "open", 10, 0)
	resolvedRows, resolvedTotal, resolvedErr := repos.Report.List(ctx, "resolved", 10, 0)

	// then
	require.NoError(t, openErr)
	require.NoError(t, resolvedErr)
	assert.Equal(t, 1, openTotal)
	require.Len(t, openRows, 1)
	assert.Equal(t, int(idOpen), openRows[0].ID)
	assert.Equal(t, 1, resolvedTotal)
	require.Len(t, resolvedRows, 1)
	assert.Equal(t, int(idResolved), resolvedRows[0].ID)
}

func TestReportRepository_List_OrderedByCreatedAtDesc(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	ids := make([]int64, 3)
	for i := 0; i < 3; i++ {
		id, err := repos.Report.Create(ctx, user.ID, "post", "p", "", "r")
		require.NoError(t, err)
		ids[i] = id
	}

	// when
	rows, _, err := repos.Report.List(ctx, "", 10, 0)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 3)
	gotIDs := []int{rows[0].ID, rows[1].ID, rows[2].ID}
	assert.Contains(t, gotIDs, int(ids[0]))
	assert.Contains(t, gotIDs, int(ids[1]))
	assert.Contains(t, gotIDs, int(ids[2]))
	for i := 0; i < len(rows)-1; i++ {
		assert.GreaterOrEqual(t, rows[i].CreatedAt, rows[i+1].CreatedAt)
	}
}

func TestReportRepository_List_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := repos.Report.Create(ctx, user.ID, "post", "p", "", "r")
		require.NoError(t, err)
	}

	// when
	page1, total1, err1 := repos.Report.List(ctx, "", 2, 0)
	page2, total2, err2 := repos.Report.List(ctx, "", 2, 2)
	page3, total3, err3 := repos.Report.List(ctx, "", 2, 4)

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)
	assert.Equal(t, 5, total1)
	assert.Equal(t, 5, total2)
	assert.Equal(t, 5, total3)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
	assert.Len(t, page3, 1)
	seen := map[int]bool{}
	for _, r := range append(append(page1, page2...), page3...) {
		assert.False(t, seen[r.ID])
		seen[r.ID] = true
	}
}

func TestReportRepository_Resolve(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	resolver := repotest.CreateUser(t, repos, repotest.WithDisplayName("Mod"))
	ctx := context.Background()
	id, err := repos.Report.Create(ctx, user.ID, "post", "p1", "", "spam")
	require.NoError(t, err)

	// when
	err = repos.Report.Resolve(ctx, int(id), resolver.ID, "warned user")

	// then
	require.NoError(t, err)
	row, getErr := repos.Report.GetByID(ctx, int(id))
	require.NoError(t, getErr)
	assert.Equal(t, "resolved", row.Status)
	require.NotNil(t, row.ResolvedByID)
	assert.Equal(t, resolver.ID, *row.ResolvedByID)
	assert.Equal(t, "Mod", row.ResolvedByName)
}

func TestReportRepository_Resolve_NonExistent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	resolver := repotest.CreateUser(t, repos)

	// when
	err := repos.Report.Resolve(context.Background(), 9999, resolver.ID, "comment")

	// then
	require.NoError(t, err)
}
