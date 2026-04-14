package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertTheory(t *testing.T, db *sql.DB, userID uuid.UUID, createdAt string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	created := createdAt
	if created == "" {
		created = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	_, err := db.Exec(
		`INSERT INTO theories (id, user_id, title, body, episode, created_at, updated_at) VALUES (?, ?, ?, ?, 0, ?, ?)`,
		id, userID, "title", "body", created, created,
	)
	require.NoError(t, err)
	return id
}

func insertResponse(t *testing.T, db *sql.DB, theoryID, userID uuid.UUID, createdAt string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	created := createdAt
	if created == "" {
		created = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	_, err := db.Exec(
		`INSERT INTO responses (id, theory_id, user_id, side, body, created_at) VALUES (?, ?, ?, 'with_love', 'b', ?)`,
		id, theoryID, userID, created,
	)
	require.NoError(t, err)
	return id
}

func insertPost(t *testing.T, db *sql.DB, userID uuid.UUID, corner, createdAt string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	created := createdAt
	if created == "" {
		created = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	_, err := db.Exec(
		`INSERT INTO posts (id, user_id, body, corner, created_at) VALUES (?, ?, 'b', ?, ?)`,
		id, userID, corner, created,
	)
	require.NoError(t, err)
	return id
}

func insertPostComment(t *testing.T, db *sql.DB, postID, userID uuid.UUID) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO post_comments (id, post_id, user_id, body) VALUES (?, ?, ?, 'b')`,
		uuid.New(), postID, userID,
	)
	require.NoError(t, err)
}

func insertTheoryVote(t *testing.T, db *sql.DB, userID, theoryID uuid.UUID) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO theory_votes (user_id, theory_id, value) VALUES (?, ?, 1)`,
		userID, theoryID,
	)
	require.NoError(t, err)
}

func insertResponseVote(t *testing.T, db *sql.DB, userID, responseID uuid.UUID) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO response_votes (user_id, response_id, value) VALUES (?, ?, 1)`,
		userID, responseID,
	)
	require.NoError(t, err)
}

func TestStatsRepository_GetOverview_EmptyDB(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	stats, err := repos.Stats.GetOverview(context.Background())

	// then
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalUsers)
	assert.Equal(t, 0, stats.TotalTheories)
	assert.Equal(t, 0, stats.TotalResponses)
	assert.Equal(t, 0, stats.TotalVotes)
	assert.Equal(t, 0, stats.TotalPosts)
	assert.Equal(t, 0, stats.TotalComments)
	assert.Equal(t, 0, stats.NewUsers24h)
	assert.Equal(t, 0, stats.NewTheories7d)
	assert.Equal(t, 0, stats.NewPosts30d)
	assert.NotNil(t, stats.PostsByCorner)
	assert.Empty(t, stats.PostsByCorner)
}

func TestStatsRepository_GetOverview_TotalsAreCorrect(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	db := repos.DB()
	alice := repotest.CreateUser(t, repos)
	bob := repotest.CreateUser(t, repos)
	theoryA := insertTheory(t, db, alice.ID, "")
	theoryB := insertTheory(t, db, bob.ID, "")
	respA := insertResponse(t, db, theoryA, bob.ID, "")
	respB := insertResponse(t, db, theoryB, alice.ID, "")
	insertResponse(t, db, theoryA, alice.ID, "")
	postA := insertPost(t, db, alice.ID, "general", "")
	postB := insertPost(t, db, bob.ID, "general", "")
	insertPostComment(t, db, postA, bob.ID)
	insertPostComment(t, db, postB, alice.ID)
	insertPostComment(t, db, postA, alice.ID)
	insertTheoryVote(t, db, alice.ID, theoryB)
	insertTheoryVote(t, db, bob.ID, theoryA)
	insertResponseVote(t, db, alice.ID, respA)
	insertResponseVote(t, db, bob.ID, respB)

	// when
	stats, err := repos.Stats.GetOverview(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalUsers)
	assert.Equal(t, 2, stats.TotalTheories)
	assert.Equal(t, 3, stats.TotalResponses)
	assert.Equal(t, 4, stats.TotalVotes)
	assert.Equal(t, 2, stats.TotalPosts)
	assert.Equal(t, 3, stats.TotalComments)
}

func TestStatsRepository_GetOverview_TimeRangeFilters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	db := repos.DB()
	user := repotest.CreateUser(t, repos)
	now := time.Now().UTC()
	recent := now.Add(-1 * time.Hour).Format("2006-01-02 15:04:05")
	withinWeek := now.Add(-3 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	withinMonth := now.Add(-15 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	old := now.Add(-60 * 24 * time.Hour).Format("2006-01-02 15:04:05")

	insertTheory(t, db, user.ID, recent)
	insertTheory(t, db, user.ID, withinWeek)
	insertTheory(t, db, user.ID, withinMonth)
	insertTheory(t, db, user.ID, old)

	theoryForResp := insertTheory(t, db, user.ID, recent)
	insertResponse(t, db, theoryForResp, user.ID, recent)
	insertResponse(t, db, theoryForResp, user.ID, withinWeek)
	insertResponse(t, db, theoryForResp, user.ID, old)

	insertPost(t, db, user.ID, "general", recent)
	insertPost(t, db, user.ID, "general", withinMonth)
	insertPost(t, db, user.ID, "general", old)

	// when
	stats, err := repos.Stats.GetOverview(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, stats.NewTheories24h)
	assert.Equal(t, 3, stats.NewTheories7d)
	assert.Equal(t, 4, stats.NewTheories30d)
	assert.Equal(t, 1, stats.NewResponses24h)
	assert.Equal(t, 2, stats.NewResponses7d)
	assert.Equal(t, 2, stats.NewResponses30d)
	assert.Equal(t, 1, stats.NewPosts24h)
	assert.Equal(t, 1, stats.NewPosts7d)
	assert.Equal(t, 2, stats.NewPosts30d)
}

func TestStatsRepository_GetOverview_PostsByCorner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	db := repos.DB()
	user := repotest.CreateUser(t, repos)
	insertPost(t, db, user.ID, "general", "")
	insertPost(t, db, user.ID, "general", "")
	insertPost(t, db, user.ID, "art", "")
	insertPost(t, db, user.ID, "fanfiction", "")
	insertPost(t, db, user.ID, "fanfiction", "")
	insertPost(t, db, user.ID, "fanfiction", "")

	// when
	stats, err := repos.Stats.GetOverview(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 6, stats.TotalPosts)
	assert.Equal(t, 2, stats.PostsByCorner["general"])
	assert.Equal(t, 1, stats.PostsByCorner["art"])
	assert.Equal(t, 3, stats.PostsByCorner["fanfiction"])
}

func TestStatsRepository_GetOverview_CountsUsersOnly(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos)
	repotest.CreateUser(t, repos)
	repotest.CreateUser(t, repos)

	// when
	stats, err := repos.Stats.GetOverview(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalUsers)
	assert.Equal(t, 3, stats.NewUsers24h)
	assert.Equal(t, 3, stats.NewUsers7d)
	assert.Equal(t, 3, stats.NewUsers30d)
}

func TestStatsRepository_GetMostActiveUsers_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	users, err := repos.Stats.GetMostActiveUsers(context.Background(), 10)

	// then
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestStatsRepository_GetMostActiveUsers_OrdersByActionCount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	db := repos.DB()
	alice := repotest.CreateUser(t, repos, repotest.WithDisplayName("Alice"))
	bob := repotest.CreateUser(t, repos, repotest.WithDisplayName("Bob"))
	carol := repotest.CreateUser(t, repos, repotest.WithDisplayName("Carol"))

	tA := insertTheory(t, db, alice.ID, "")
	insertTheory(t, db, alice.ID, "")
	insertResponse(t, db, tA, alice.ID, "")
	postA := insertPost(t, db, alice.ID, "general", "")
	insertPostComment(t, db, postA, alice.ID)

	tB := insertTheory(t, db, bob.ID, "")
	insertResponse(t, db, tB, bob.ID, "")

	insertPost(t, db, carol.ID, "general", "")

	// when
	users, err := repos.Stats.GetMostActiveUsers(context.Background(), 10)

	// then
	require.NoError(t, err)
	require.Len(t, users, 3)
	assert.Equal(t, alice.ID, users[0].ID)
	assert.Equal(t, 5, users[0].ActionCount)
	assert.Equal(t, bob.ID, users[1].ID)
	assert.Equal(t, 2, users[1].ActionCount)
	assert.Equal(t, carol.ID, users[2].ID)
	assert.Equal(t, 1, users[2].ActionCount)
}

func TestStatsRepository_GetMostActiveUsers_RespectsLimit(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	db := repos.DB()
	for i := 0; i < 5; i++ {
		u := repotest.CreateUser(t, repos)
		insertTheory(t, db, u.ID, "")
	}

	// when
	users, err := repos.Stats.GetMostActiveUsers(context.Background(), 2)

	// then
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestStatsRepository_GetMostActiveUsers_OmitsUsersWithNoActions(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	db := repos.DB()
	active := repotest.CreateUser(t, repos, repotest.WithDisplayName("Active"))
	repotest.CreateUser(t, repos, repotest.WithDisplayName("Idle"))
	insertTheory(t, db, active.ID, "")

	// when
	users, err := repos.Stats.GetMostActiveUsers(context.Background(), 10)

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, active.ID, users[0].ID)
	assert.Equal(t, "Active", users[0].DisplayName)
}

func TestStatsRepository_GetMostActiveUsers_PopulatesUserFields(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	db := repos.DB()
	user := repotest.CreateUser(t, repos,
		repotest.WithUsername("acehunter"),
		repotest.WithDisplayName("Ace Hunter"),
	)
	_, err := db.Exec(`UPDATE users SET avatar_url = ? WHERE id = ?`, "https://example.com/a.png", user.ID)
	require.NoError(t, err)
	insertTheory(t, db, user.ID, "")

	// when
	users, err := repos.Stats.GetMostActiveUsers(context.Background(), 5)

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, user.ID, users[0].ID)
	assert.Equal(t, "acehunter", users[0].Username)
	assert.Equal(t, "Ace Hunter", users[0].DisplayName)
	assert.Equal(t, "https://example.com/a.png", users[0].AvatarURL)
	assert.Equal(t, 1, users[0].ActionCount)
}
