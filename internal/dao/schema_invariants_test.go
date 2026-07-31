package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_PostCommentParentCannotCrossPosts(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)

	postA, postB := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{postA, postB} {
		_, err := repos.DB().ExecContext(ctx,
			`INSERT INTO posts (id, user_id, body) VALUES ($1, $2, 'body')`, id, author.ID)
		require.NoError(t, err)
	}

	parent := uuid.New()
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO post_comments (id, post_id, user_id, body) VALUES ($1, $2, $3, 'parent')`,
		parent, postA, author.ID)
	require.NoError(t, err)

	// when
	_, err = repos.DB().ExecContext(ctx,
		`INSERT INTO post_comments (id, post_id, user_id, body, parent_id) VALUES ($1, $2, $3, 'reply', $4)`,
		uuid.New(), postB, author.ID, parent)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "post_comments_parent_same_post_fkey")
}

func TestSchema_PostCommentParentAllowedWithinSamePost(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)

	post := uuid.New()
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO posts (id, user_id, body) VALUES ($1, $2, 'body')`, post, author.ID)
	require.NoError(t, err)

	parent := uuid.New()
	_, err = repos.DB().ExecContext(ctx,
		`INSERT INTO post_comments (id, post_id, user_id, body) VALUES ($1, $2, $3, 'parent')`,
		parent, post, author.ID)
	require.NoError(t, err)

	// when
	_, err = repos.DB().ExecContext(ctx,
		`INSERT INTO post_comments (id, post_id, user_id, body, parent_id) VALUES ($1, $2, $3, 'reply', $4)`,
		uuid.New(), post, author.ID, parent)

	// then
	require.NoError(t, err)
}

func TestSchema_PollVoteCannotUseAnotherPollsOption(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos, daotest.WithUsername("voter"), daotest.WithEmail("voter@example.com"))

	pollA, pollB := uuid.New(), uuid.New()
	var foreignOption int64
	for i, pollID := range []uuid.UUID{pollA, pollB} {
		postID := uuid.New()
		_, err := repos.DB().ExecContext(ctx,
			`INSERT INTO posts (id, user_id, body) VALUES ($1, $2, 'body')`, postID, author.ID)
		require.NoError(t, err)

		_, err = repos.DB().ExecContext(ctx,
			`INSERT INTO post_polls (id, post_id, duration_seconds, expires_at) VALUES ($1, $2, 3600, NOW() + INTERVAL '1 hour')`,
			pollID, postID)
		require.NoError(t, err)

		var optionID int64
		require.NoError(t, repos.DB().QueryRowContext(ctx,
			`INSERT INTO post_poll_options (poll_id, label) VALUES ($1, 'opt') RETURNING id`, pollID,
		).Scan(&optionID))
		if i == 1 {
			foreignOption = optionID
		}
	}

	// when
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO post_poll_votes (poll_id, user_id, option_id) VALUES ($1, $2, $3)`,
		pollA, voter.ID, foreignOption)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "post_poll_votes_option_same_poll_fkey")
}

func TestSchema_DeletingWatchPartyHostNoLongerBlocksAccountDeletion(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)

	roomID := uuid.New()
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO chat_rooms (id, name, type, created_by) VALUES ($1, 'room', 'group', $2)`, roomID, host.ID)
	require.NoError(t, err)

	sessionID := uuid.New()
	_, err = repos.DB().ExecContext(ctx,
		`INSERT INTO chat_watch_party_sessions (id, room_id, started_by, controller_id, hyperbeam_session_id, embed_url, title, type)
		 VALUES ($1, $2, $3, $3, 'hb', 'https://embed', 'party', 'hyperbeam')`,
		sessionID, roomID, host.ID)
	require.NoError(t, err)

	// when
	_, err = repos.DB().ExecContext(ctx, `DELETE FROM users WHERE id = $1`, host.ID)

	// then
	require.NoError(t, err)

	var startedBy, controllerID *uuid.UUID
	require.NoError(t, repos.DB().QueryRowContext(ctx,
		`SELECT started_by, controller_id FROM chat_watch_party_sessions WHERE id = $1`, sessionID,
	).Scan(&startedBy, &controllerID))
	assert.Nil(t, startedBy)
	assert.Nil(t, controllerID)
}

func TestSchema_DeletingRoomCreatorPreservesRoom(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	creator := daotest.CreateUser(t, repos)

	roomID := uuid.New()
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO chat_rooms (id, name, type, created_by) VALUES ($1, 'group room', 'group', $2)`, roomID, creator.ID)
	require.NoError(t, err)

	// when
	_, err = repos.DB().ExecContext(ctx, `DELETE FROM users WHERE id = $1`, creator.ID)
	require.NoError(t, err)

	// then
	var createdBy *uuid.UUID
	require.NoError(t, repos.DB().QueryRowContext(ctx,
		`SELECT created_by FROM chat_rooms WHERE id = $1`, roomID,
	).Scan(&createdBy))
	assert.Nil(t, createdBy)
}

func TestSchema_DeletingAnnouncementAuthorPreservesAnnouncement(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)

	id := uuid.New()
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO announcements (id, title, body, author_id) VALUES ($1, 'title', 'body', $2)`, id, author.ID)
	require.NoError(t, err)

	// when
	_, err = repos.DB().ExecContext(ctx, `DELETE FROM users WHERE id = $1`, author.ID)
	require.NoError(t, err)

	// then
	row, err := repos.Announcement.GetByID(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "title", row.Title)
	assert.Equal(t, uuid.Nil, row.AuthorID)
	assert.Equal(t, "", row.AuthorUsername)
}

func TestSchema_LiveStreamStatusRejectsTypos(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	streamer := daotest.CreateUser(t, repos)

	// when
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO live_streams (user_id, title, status) VALUES ($1, 'stream', 'offine')`, streamer.ID)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "live_streams_status_check")
}

func TestSchema_FanficLanguageMustExistInLookupTable(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)

	// when
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO fanfics (id, user_id, title, series, language) VALUES ($1, $2, 'title', 'Umineko', 'Klingon')`,
		uuid.New(), author.ID)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fanfics_language_fkey")
}

func TestSchema_UnsolvedMysteryCannotHaveWinner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	gm := daotest.CreateUser(t, repos)

	// when
	_, err := repos.DB().ExecContext(ctx,
		`INSERT INTO mysteries (id, user_id, title, body, solved, winner_id) VALUES ($1, $2, 'title', 'body', FALSE, $2)`,
		uuid.New(), gm.ID)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mysteries_unsolved_state_check")
}
