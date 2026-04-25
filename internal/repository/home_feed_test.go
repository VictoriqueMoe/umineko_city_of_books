package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeFeedRepository_ListRecentActivity_ReturnsAllKinds(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Author"))

	createPost(t, repos, user.ID, "general", "post body")
	createTheory(t, repos, user.ID, "theory title")
	createJournal(t, repos, user.ID, "journal title", "journal body", "umineko")
	createArt(t, repos, user.ID, "general", "drawing", "art title", nil, false)

	rows, err := repos.HomeFeed.ListRecentActivity(ctx, 10)
	require.NoError(t, err)

	kinds := make(map[string]int)
	for i := 0; i < len(rows); i++ {
		kinds[rows[i].Kind]++
	}
	assert.Equal(t, 1, kinds["post"])
	assert.Equal(t, 1, kinds["theory"])
	assert.Equal(t, 1, kinds["journal"])
	assert.Equal(t, 1, kinds["art"])

	for i := 0; i < len(rows); i++ {
		assert.Equal(t, user.ID, rows[i].AuthorID)
		assert.Equal(t, "Author", rows[i].DisplayName)
	}
}

func TestHomeFeedRepository_ListRecentActivity_OrderedByCreatedAtDesc(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	first := createPost(t, repos, user.ID, "general", "first")
	second := createPost(t, repos, user.ID, "general", "second")

	_, err := repos.DB().ExecContext(ctx,
		`UPDATE posts SET created_at = NOW() - INTERVAL '2 hours' WHERE id = $1`, first,
	)
	require.NoError(t, err)
	_, err = repos.DB().ExecContext(ctx,
		`UPDATE posts SET created_at = NOW() - INTERVAL '1 hour' WHERE id = $1`, second,
	)
	require.NoError(t, err)

	rows, err := repos.HomeFeed.ListRecentActivity(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, second, rows[0].ID)
	assert.Equal(t, first, rows[1].ID)
}

func TestHomeFeedRepository_ListRecentActivity_Limit(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	for i := 0; i < 5; i++ {
		createPost(t, repos, user.ID, "general", "body")
	}

	rows, err := repos.HomeFeed.ListRecentActivity(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, rows, 3)
}

func TestHomeFeedRepository_ListRecentActivity_ExcludesBannedAuthors(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	good := repotest.CreateUser(t, repos, repotest.WithDisplayName("Good"))
	bad := repotest.CreateUser(t, repos, repotest.WithDisplayName("Bad"))
	mod := repotest.CreateUser(t, repos)

	createPost(t, repos, good.ID, "general", "visible")
	createPost(t, repos, bad.ID, "general", "hidden")
	require.NoError(t, repos.User.BanUser(ctx, bad.ID, mod.ID, "spam"))

	rows, err := repos.HomeFeed.ListRecentActivity(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, good.ID, rows[0].AuthorID)
}

func TestHomeFeedRepository_ListRecentActivity_ExcludesArchivedJournals(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	id := createJournal(t, repos, user.ID, "live", "body", "umineko")
	createJournal(t, repos, user.ID, "archived", "body", "umineko")

	_, err := repos.DB().ExecContext(ctx,
		`UPDATE journals SET archived_at = NOW() WHERE title = 'archived'`,
	)
	require.NoError(t, err)

	rows, err := repos.HomeFeed.ListRecentActivity(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, id, rows[0].ID)
}

func TestHomeFeedRepository_ListRecentMembers_OrderedAndLimited(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()

	first := repotest.CreateUser(t, repos, repotest.WithDisplayName("First"))
	second := repotest.CreateUser(t, repos, repotest.WithDisplayName("Second"))
	third := repotest.CreateUser(t, repos, repotest.WithDisplayName("Third"))

	_, err := repos.DB().ExecContext(ctx,
		`UPDATE users SET created_at = NOW() - INTERVAL '3 days' WHERE id = $1`, first.ID)
	require.NoError(t, err)
	_, err = repos.DB().ExecContext(ctx,
		`UPDATE users SET created_at = NOW() - INTERVAL '2 days' WHERE id = $1`, second.ID)
	require.NoError(t, err)
	_, err = repos.DB().ExecContext(ctx,
		`UPDATE users SET created_at = NOW() - INTERVAL '1 day' WHERE id = $1`, third.ID)
	require.NoError(t, err)

	rows, err := repos.HomeFeed.ListRecentMembers(ctx, 2)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, third.ID, rows[0].ID)
	assert.Equal(t, second.ID, rows[1].ID)
}

func TestHomeFeedRepository_ListRecentMembers_ExcludesBanned(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()

	good := repotest.CreateUser(t, repos, repotest.WithDisplayName("Good"))
	bad := repotest.CreateUser(t, repos)
	mod := repotest.CreateUser(t, repos)
	require.NoError(t, repos.User.BanUser(ctx, bad.ID, mod.ID, "x"))

	rows, err := repos.HomeFeed.ListRecentMembers(ctx, 10)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)
	for i := 0; i < len(rows); i++ {
		ids[rows[i].ID] = true
	}
	assert.True(t, ids[good.ID])
	assert.True(t, ids[mod.ID])
	assert.False(t, ids[bad.ID])
}

func TestHomeFeedRepository_ListCornerActivity24h_GroupsAndCountsUniquePosters(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()

	userA := repotest.CreateUser(t, repos)
	userB := repotest.CreateUser(t, repos)

	createPost(t, repos, userA.ID, "umineko", "p1")
	createPost(t, repos, userA.ID, "umineko", "p2")
	createPost(t, repos, userB.ID, "umineko", "p3")
	createPost(t, repos, userA.ID, "higurashi", "p4")

	rows, err := repos.HomeFeed.ListCornerActivity24h(ctx)
	require.NoError(t, err)

	byCorner := make(map[string]int)
	uniqueByCorner := make(map[string]int)
	for i := 0; i < len(rows); i++ {
		byCorner[rows[i].Corner] = rows[i].PostCount
		uniqueByCorner[rows[i].Corner] = rows[i].UniquePosters
	}
	assert.Equal(t, 3, byCorner["umineko"])
	assert.Equal(t, 1, byCorner["higurashi"])
	assert.Equal(t, 2, uniqueByCorner["umineko"])
	assert.Equal(t, 1, uniqueByCorner["higurashi"])
}

func TestHomeFeedRepository_ListCornerActivity24h_ExcludesOlderThan24h(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	createPost(t, repos, user.ID, "umineko", "recent")
	old := createPost(t, repos, user.ID, "umineko", "old")

	_, err := repos.DB().ExecContext(ctx,
		`UPDATE posts SET created_at = NOW() - INTERVAL '2 days' WHERE id = $1`, old)
	require.NoError(t, err)

	rows, err := repos.HomeFeed.ListCornerActivity24h(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "umineko", rows[0].Corner)
	assert.Equal(t, 1, rows[0].PostCount)
}

func TestHomeFeedRepository_ListCornerActivity24h_ExcludesBanned(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	good := repotest.CreateUser(t, repos)
	bad := repotest.CreateUser(t, repos)
	mod := repotest.CreateUser(t, repos)

	createPost(t, repos, good.ID, "umineko", "keep")
	createPost(t, repos, bad.ID, "umineko", "drop")
	require.NoError(t, repos.User.BanUser(ctx, bad.ID, mod.ID, "x"))

	rows, err := repos.HomeFeed.ListCornerActivity24h(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].PostCount)
	assert.Equal(t, 1, rows[0].UniquePosters)
}

func TestHomeFeedRepository_ListSidebarActivity_AggregatesByKey(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	createPost(t, repos, user.ID, "umineko", "p")
	createArt(t, repos, user.ID, "higurashi", "drawing", "art", nil, false)
	createTheory(t, repos, user.ID, "th")
	createMystery(t, repos, user.ID, "m", "easy", false)
	createShip(t, repos, user.ID, "ship", makeChars())
	createFanfic(t, repos, user.ID, "ff")
	createJournal(t, repos, user.ID, "j", "body", "umineko")
	unlockSecretFor(t, repos, user.ID, testSecretID)
	createSecretComment(t, repos, testSecretID, nil, user.ID, "s")

	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "d", "group", true, false, user.ID))

	rows, err := repos.HomeFeed.ListSidebarActivity(ctx)
	require.NoError(t, err)

	keys := make(map[string]string)
	for i := 0; i < len(rows); i++ {
		keys[rows[i].Key] = rows[i].LatestAt
	}
	assert.Contains(t, keys, "game_board_umineko")
	assert.Contains(t, keys, "gallery_higurashi")
	assert.Contains(t, keys, "theories_umineko")
	assert.Contains(t, keys, "mysteries")
	assert.Contains(t, keys, "ships")
	assert.Contains(t, keys, "fanfiction")
	assert.Contains(t, keys, "journals")
	assert.Contains(t, keys, "secrets")
	assert.Contains(t, keys, "rooms")
}

func TestHomeFeedRepository_ListSidebarActivity_EmptyDB(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()

	rows, err := repos.HomeFeed.ListSidebarActivity(ctx)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestHomeFeedRepository_ListPublicRooms_ReturnsPublicGroupsOnly(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	publicID := uuid.New()
	privateID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, publicID, "Public", "", "group", true, false, user.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, privateID, "Private", "", "group", false, false, user.ID))

	rows, err := repos.HomeFeed.ListPublicRooms(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, publicID, rows[0].ID)
	assert.Equal(t, "Public", rows[0].Name)
}

func TestHomeFeedRepository_ListPublicRooms_OrderedByLastMessage(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	older := uuid.New()
	newer := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, older, "Older", "", "group", true, false, user.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, newer, "Newer", "", "group", true, false, user.ID))

	_, err := repos.DB().ExecContext(ctx,
		`UPDATE chat_rooms SET last_message_at = NOW() - INTERVAL '2 hours' WHERE id = $1`, older)
	require.NoError(t, err)
	_, err = repos.DB().ExecContext(ctx,
		`UPDATE chat_rooms SET last_message_at = NOW() - INTERVAL '1 hour' WHERE id = $1`, newer)
	require.NoError(t, err)

	rows, err := repos.HomeFeed.ListPublicRooms(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, newer, rows[0].ID)
	assert.Equal(t, older, rows[1].ID)
}

func TestHomeFeedRepository_ListPublicRooms_LimitApplies(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	for i := 0; i < 4; i++ {
		require.NoError(t, repos.Chat.CreateRoom(ctx, uuid.New(), "R", "", "group", true, false, user.ID))
	}

	rows, err := repos.HomeFeed.ListPublicRooms(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
}

func TestHomeFeedRepository_ListPublicRooms_IncludesMemberCount(t *testing.T) {
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	joiner := repotest.CreateUser(t, repos)

	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, joiner.ID))

	rows, err := repos.HomeFeed.ListPublicRooms(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 2, rows[0].MemberCount)
}
