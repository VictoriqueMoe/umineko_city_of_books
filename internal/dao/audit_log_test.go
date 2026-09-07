package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogDAO_Create(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.AuditLog.Create(context.Background(), audit.NewEntry{
		ActorID:    user.ID,
		Action:     "user.ban",
		TargetType: audit.TargetUser,
		TargetID:   "target-1",
		Details:    "reason: spam",
	})

	// then
	require.NoError(t, err)
	entries, total, listErr := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Page: bounds.NewPage(10, 0)})
	require.NoError(t, listErr)
	assert.Equal(t, 1, total)
	require.Len(t, entries, 1)
	assert.Equal(t, user.ID, entries[0].ActorID)
	assert.Equal(t, user.DisplayName, entries[0].ActorName)
	assert.Equal(t, audit.Action("user.ban"), entries[0].Action)
	assert.Equal(t, audit.TargetUser, entries[0].TargetType)
	assert.Equal(t, "target-1", entries[0].TargetID)
	assert.Equal(t, "reason: spam", entries[0].Details)
	assert.NotEmpty(t, entries[0].CreatedAt)
	assert.NotZero(t, entries[0].ID)
}

func TestAuditLogDAO_Create_InvalidActor_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	bogus := uuid.New()

	// when
	err := repos.AuditLog.Create(context.Background(), audit.NewEntry{
		ActorID:    bogus,
		Action:     "user.ban",
		TargetType: audit.TargetUser,
		TargetID:   "target-1",
	})

	// then
	require.Error(t, err)
}

func TestAuditLogDAO_List_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	entries, total, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Page: bounds.NewPage(10, 0)})

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, entries)
}

func TestAuditLogDAO_List_FilterByAction(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	ctx := context.Background()
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: user.ID, Action: "user.ban", TargetType: audit.TargetUser, TargetID: "t1"}))
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: user.ID, Action: "user.ban", TargetType: audit.TargetUser, TargetID: "t2"}))
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: user.ID, Action: "user.unban", TargetType: audit.TargetUser, TargetID: "t3"}))

	// when
	entries, total, err := repos.AuditLog.List(ctx, spec.AuditLogListing{Action: "user.ban", Page: bounds.NewPage(10, 0)})

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, audit.Action("user.ban"), e.Action)
	}
}

func TestAuditLogDAO_ListForUser_ScopesToOneUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	actor := daotest.CreateUser(t, repos, daotest.WithUsername("actor"))
	subject := daotest.CreateUser(t, repos, daotest.WithUsername("subject"))
	other := daotest.CreateUser(t, repos, daotest.WithUsername("other"))
	ctx := context.Background()
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: actor.ID, Action: audit.ActionBanUser, TargetType: audit.TargetUser, TargetID: subject.ID.String()}))
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: actor.ID, Action: audit.ActionLockUser, TargetType: audit.TargetUser, TargetID: subject.ID.String(), Details: "spam"}))
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: actor.ID, Action: audit.ActionBanUser, TargetType: audit.TargetUser, TargetID: other.ID.String()}))
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: actor.ID, Action: audit.ActionUpdateSettings, TargetType: audit.TargetSettings, TargetID: subject.ID.String()}))

	// when
	entries, total, err := repos.AuditLog.ListForUser(ctx, spec.AuditLogUserListing{UserID: subject.ID, Page: bounds.NewPage(10, 0)})

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, audit.TargetUser, e.TargetType)
		assert.Equal(t, subject.ID.String(), e.TargetID)
	}
}

func TestAuditLogDAO_ListForUser_IncludesRoomScopedEvents(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	actor := daotest.CreateUser(t, repos, daotest.WithUsername("moderator"))
	subject := daotest.CreateUser(t, repos, daotest.WithUsername("offender"))
	other := daotest.CreateUser(t, repos, daotest.WithUsername("bystander"))
	roomID := uuid.New()
	ctx := context.Background()
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: actor.ID, Action: audit.ActionChatRoomBan, TargetType: audit.TargetChatRoom, TargetID: roomID.String(), Details: "reason=spam", SubjectID: subject.ID}))
	require.NoError(t, repos.AuditLog.CreateSystem(ctx, audit.NewEntry{Action: audit.ActionChatWordFilterKick, TargetType: audit.TargetChatRoom, TargetID: roomID.String(), Details: `pattern="x"`, SubjectID: subject.ID}))
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: actor.ID, Action: audit.ActionChatRoomBan, TargetType: audit.TargetChatRoom, TargetID: roomID.String(), SubjectID: other.ID}))

	// when
	entries, total, err := repos.AuditLog.ListForUser(ctx, spec.AuditLogUserListing{UserID: subject.ID, Page: bounds.NewPage(10, 0)})

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, audit.TargetChatRoom, e.TargetType)
		assert.Equal(t, roomID.String(), e.TargetID)
	}
}

func TestAuditLogDAO_ListForUser_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	entries, total, err := repos.AuditLog.ListForUser(context.Background(), spec.AuditLogUserListing{UserID: uuid.New(), Page: bounds.NewPage(10, 0)})

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, entries)
}

func TestAuditLogDAO_List_OrderedByCreatedAtDesc(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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
	entries, _, err := repos.AuditLog.List(ctx, spec.AuditLogListing{Page: bounds.NewPage(10, 0)})

	// then
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, audit.Action("a3"), entries[0].Action)
	assert.Equal(t, audit.Action("a2"), entries[1].Action)
	assert.Equal(t, audit.Action("a1"), entries[2].Action)
}

func TestAuditLogDAO_List_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	ctx := context.Background()
	for range 5 {
		require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: user.ID, Action: "act", TargetType: audit.TargetUser, TargetID: "t"}))
	}

	// when
	page1, total1, err1 := repos.AuditLog.List(ctx, spec.AuditLogListing{Page: bounds.NewPage(2, 0)})
	page2, total2, err2 := repos.AuditLog.List(ctx, spec.AuditLogListing{Page: bounds.NewPage(2, 2)})
	page3, total3, err3 := repos.AuditLog.List(ctx, spec.AuditLogListing{Page: bounds.NewPage(2, 4)})

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

func TestAuditLogDAO_List_JoinsActorDisplayName(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	alice := daotest.CreateUser(t, repos, daotest.WithDisplayName("Alice"))
	bob := daotest.CreateUser(t, repos, daotest.WithDisplayName("Bob"))
	ctx := context.Background()
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: alice.ID, Action: "act", TargetType: audit.TargetUser, TargetID: "t1"}))
	require.NoError(t, repos.AuditLog.Create(ctx, audit.NewEntry{ActorID: bob.ID, Action: "act", TargetType: audit.TargetUser, TargetID: "t2"}))

	// when
	entries, total, err := repos.AuditLog.List(ctx, spec.AuditLogListing{Page: bounds.NewPage(10, 0)})

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
