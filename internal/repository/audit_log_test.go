package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogRepository_Create(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.AuditLog.Create(context.Background(), user.ID, "user.ban", "user", "target-1", "reason: spam")

	// then
	require.NoError(t, err)
	entries, total, listErr := repos.AuditLog.List(context.Background(), "", 10, 0)
	require.NoError(t, listErr)
	assert.Equal(t, 1, total)
	require.Len(t, entries, 1)
	assert.Equal(t, user.ID, entries[0].ActorID)
	assert.Equal(t, user.DisplayName, entries[0].ActorName)
	assert.Equal(t, "user.ban", entries[0].Action)
	assert.Equal(t, "user", entries[0].TargetType)
	assert.Equal(t, "target-1", entries[0].TargetID)
	assert.Equal(t, "reason: spam", entries[0].Details)
	assert.NotEmpty(t, entries[0].CreatedAt)
	assert.NotZero(t, entries[0].ID)
}

func TestAuditLogRepository_Create_InvalidActor_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	bogus := uuid.New()

	// when
	err := repos.AuditLog.Create(context.Background(), bogus, "user.ban", "user", "target-1", "")

	// then
	require.Error(t, err)
}

func TestAuditLogRepository_List_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	entries, total, err := repos.AuditLog.List(context.Background(), "", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, entries)
}

func TestAuditLogRepository_List_FilterByAction(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	require.NoError(t, repos.AuditLog.Create(ctx, user.ID, "user.ban", "user", "t1", ""))
	require.NoError(t, repos.AuditLog.Create(ctx, user.ID, "user.ban", "user", "t2", ""))
	require.NoError(t, repos.AuditLog.Create(ctx, user.ID, "user.unban", "user", "t3", ""))

	// when
	entries, total, err := repos.AuditLog.List(ctx, "user.ban", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, "user.ban", e.Action)
	}
}

func TestAuditLogRepository_List_OrderedByCreatedAtDesc(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO audit_log (actor_id, action, target_type, target_id, details, created_at) VALUES
		($1, 'a1', 'user', 't1', '', '2026-01-01 10:00:00'),
		($2, 'a2', 'user', 't2', '', '2026-01-02 10:00:00'),
		($3, 'a3', 'user', 't3', '', '2026-01-03 10:00:00')`,
		user.ID, user.ID, user.ID,
	)
	require.NoError(t, err)

	// when
	entries, _, err := repos.AuditLog.List(ctx, "", 10, 0)

	// then
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, "a3", entries[0].Action)
	assert.Equal(t, "a2", entries[1].Action)
	assert.Equal(t, "a1", entries[2].Action)
}

func TestAuditLogRepository_List_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		require.NoError(t, repos.AuditLog.Create(ctx, user.ID, "act", "user", "t", ""))
	}

	// when
	page1, total1, err1 := repos.AuditLog.List(ctx, "", 2, 0)
	page2, total2, err2 := repos.AuditLog.List(ctx, "", 2, 2)
	page3, total3, err3 := repos.AuditLog.List(ctx, "", 2, 4)

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
	for _, e := range append(append(page1, page2...), page3...) {
		assert.False(t, seen[e.ID], "duplicate entry id %d across pages", e.ID)
		seen[e.ID] = true
	}
}

func TestAuditLogRepository_List_JoinsActorDisplayName(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	alice := repotest.CreateUser(t, repos, repotest.WithDisplayName("Alice"))
	bob := repotest.CreateUser(t, repos, repotest.WithDisplayName("Bob"))
	ctx := context.Background()
	require.NoError(t, repos.AuditLog.Create(ctx, alice.ID, "act", "user", "t1", ""))
	require.NoError(t, repos.AuditLog.Create(ctx, bob.ID, "act", "user", "t2", ""))

	// when
	entries, total, err := repos.AuditLog.List(ctx, "", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	byName := map[string]string{}
	for _, e := range entries {
		byName[e.ActorName] = e.ActorID.String()
	}
	assert.Equal(t, alice.ID.String(), byName["Alice"])
	assert.Equal(t, bob.ID.String(), byName["Bob"])
}
