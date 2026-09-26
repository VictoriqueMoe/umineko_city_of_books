package dao_test

import (
	"context"
	"database/sql"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	mysteryMediaSurface struct {
		name      string
		onComment bool
		add       func(repository.MysteryRepository, context.Context, spec.NewMedia, ...*sql.Tx) (int64, error)
		get       func(repository.MysteryRepository, context.Context, uuid.UUID, ...*sql.Tx) ([]model.PostMediaRow, error)
	}

	mysteryClueFixture struct {
		repos     *repository.Repositories
		mysteryID uuid.UUID
		oldClueID int
		playerID  uuid.UUID
	}

	mysteryVote struct {
		voter int
		value int
	}

	mysteryLikeAction func(repository.MysteryRepository, context.Context, spec.CommentLike, ...*sql.Tx) error
)

var (
	mysteryEntityMedia = mysteryMediaSurface{
		name:      "mystery media",
		onComment: false,
		add:       repository.MysteryRepository.AddMedia,
		get:       repository.MysteryRepository.GetMedia,
	}
	mysteryCommentMedia = mysteryMediaSurface{
		name:      "comment media",
		onComment: true,
		add:       repository.MysteryRepository.AddCommentMedia,
		get:       repository.MysteryRepository.GetCommentMedia,
	}
)

func createMystery(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title string, difficulty string, freeForAll bool) uuid.UUID {
	t.Helper()
	created, err := repos.Mystery.Create(context.Background(), spec.NewMystery{
		UserID:             userID,
		Title:              title,
		Body:               "body",
		Difficulty:         difficulty,
		FreeForAll:         freeForAll,
		KeepOpenAfterSolve: false,
		Knox:               dto.DefaultKnoxContract(),
	})
	require.NoError(t, err)

	return created.ID
}

func createAttempt(t *testing.T, repos *repository.Repositories, mysteryID, userID uuid.UUID, parent *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Mystery.CreateAttempt(context.Background(), spec.NewMysteryAttempt{
		MysteryID: mysteryID,
		UserID:    userID,
		ParentID:  parent,
		Body:      body,
	})
	require.NoError(t, err)

	return created.ID
}

func createMysteryComment(t *testing.T, repos *repository.Repositories, mysteryID uuid.UUID, parent *uuid.UUID, userID uuid.UUID, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Comments.ByID[string(mention.KindMysteryComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{
		TargetID: mysteryID,
		ParentID: parent,
		UserID:   userID,
		Body:     body,
	})
	require.NoError(t, err)

	return created.ID
}

func mysteryByGM(t *testing.T) (*repository.Repositories, *model.User, uuid.UUID) {
	t.Helper()
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)

	return repos, gm, createMystery(t, repos, gm.ID, "T", "easy", false)
}

func mysteryGet(t *testing.T, repos *repository.Repositories, id uuid.UUID) *model.MysteryRow {
	t.Helper()
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)

	return row
}

func mysteryBackdate(t *testing.T, repos *repository.Repositories, mysteryID uuid.UUID, createdAt string) {
	t.Helper()
	_, err := repos.DB().ExecContext(context.Background(), `UPDATE mysteries SET created_at = $1 WHERE id = $2`, createdAt, mysteryID)
	require.NoError(t, err)
}

func mysteryRowIDs(rows []model.MysteryRow) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

func mysteryAddClue(t *testing.T, repos *repository.Repositories, mysteryID uuid.UUID, clue spec.NewClue) int {
	t.Helper()
	created, err := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{MysteryID: mysteryID, NewClue: clue})
	require.NoError(t, err)

	return created.ID
}

func mysteryWithClues(t *testing.T) mysteryClueFixture {
	t.Helper()
	repos, _, id := mysteryByGM(t)
	player := daotest.CreateUser(t, repos)

	mysteryAddClue(t, repos, id, spec.NewClue{Body: "other", TruthType: "blue", SortOrder: 1})
	oldID := mysteryAddClue(t, repos, id, spec.NewClue{Body: "old", TruthType: "red", SortOrder: 0})
	mysteryAddClue(t, repos, id, spec.NewClue{Body: "private", TruthType: "red", SortOrder: 2, PlayerID: &player.ID})

	return mysteryClueFixture{repos: repos, mysteryID: id, oldClueID: oldID, playerID: player.ID}
}

func mysteryClueBodies(t *testing.T, repos *repository.Repositories, mysteryID uuid.UUID) []string {
	t.Helper()
	clues, err := repos.Mystery.GetClues(context.Background(), mysteryID)
	require.NoError(t, err)

	var bodies []string
	for _, clue := range clues {
		bodies = append(bodies, clue.Body)
	}

	return bodies
}

func mysterySolve(t *testing.T, repos *repository.Repositories, mysteryID, attemptID uuid.UUID) {
	t.Helper()
	err := repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: mysteryID, AttemptID: attemptID, LockMystery: true})
	require.NoError(t, err)
}

func mysterySolvedBy(t *testing.T, repos *repository.Repositories, gmID, solverID uuid.UUID, difficulty string) {
	t.Helper()
	id := createMystery(t, repos, gmID, difficulty, difficulty, false)
	mysterySolve(t, repos, id, createAttempt(t, repos, id, solverID, nil, "answer"))
}

func mysteryAttempts(t *testing.T, repos *repository.Repositories, mysteryID, viewerID uuid.UUID) map[uuid.UUID]model.MysteryAttemptRow {
	t.Helper()
	rows, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: mysteryID, ViewerID: viewerID})
	require.NoError(t, err)

	byID := make(map[uuid.UUID]model.MysteryAttemptRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	require.Len(t, byID, len(rows))

	return byID
}

func mysteryComments(t *testing.T, repos *repository.Repositories, mysteryID, viewerID uuid.UUID) map[uuid.UUID]model.CommentRow {
	t.Helper()
	rows, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: mysteryID, ViewerID: viewerID, Limit: 500, Offset: 0})
	require.NoError(t, err)

	byID := make(map[uuid.UUID]model.CommentRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	require.Len(t, byID, len(rows))

	return byID
}

func mysteryAddCommentMedia(t *testing.T, repos *repository.Repositories, commentID uuid.UUID, mediaURL, thumbnailURL string) int64 {
	t.Helper()
	id, err := repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: mediaURL, MediaType: "image", ThumbnailURL: thumbnailURL})
	require.NoError(t, err)

	return id
}

func mysteryAddUploads(t *testing.T, repos *repository.Repositories, mysteryID, commenterID uuid.UUID, thumbnails bool) {
	t.Helper()
	ctx := context.Background()

	boardThumbnail := ""
	replyThumbnail := ""
	if thumbnails {
		boardThumbnail = "/uploads/mystery/board_thumb.png"
		replyThumbnail = "/uploads/mystery/reply_thumb.png"
	}

	_, err := repos.Mystery.AddMedia(ctx, spec.NewMedia{TargetID: mysteryID, MediaURL: "/uploads/mystery/board.png", MediaType: "image", ThumbnailURL: boardThumbnail})
	require.NoError(t, err)

	_, err = repos.Mystery.AddAttachment(ctx, spec.NewMysteryAttachment{MysteryID: mysteryID, FileURL: "/uploads/mystery/case.pdf", FileName: "case.pdf", FileSize: 42})
	require.NoError(t, err)

	commentID := createMysteryComment(t, repos, mysteryID, nil, commenterID, "a clue")
	mysteryAddCommentMedia(t, repos, commentID, "/uploads/mystery/reply.png", replyThumbnail)
}

func mysteryCheckErr(t *testing.T, wantErr bool, err error) {
	t.Helper()
	if wantErr {
		require.Error(t, err)

		return
	}

	require.NoError(t, err)
}

func (s mysteryMediaSurface) target(t *testing.T, repos *repository.Repositories, mysteryID, userID uuid.UUID) uuid.UUID {
	t.Helper()
	if !s.onComment {
		return mysteryID
	}

	return createMysteryComment(t, repos, mysteryID, nil, userID, "x")
}

func TestMysteryDAO_Create(t *testing.T) {
	cases := []struct {
		name       string
		freeForAll bool
	}{
		{name: "a regular mystery reads back unsolved with its author through every lookup", freeForAll: false},
		{name: "a free-for-all mystery keeps its flag", freeForAll: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Author Name"))

			// when
			created, err := repos.Mystery.Create(ctx, spec.NewMystery{
				UserID:     user.ID,
				Title:      "The Murder",
				Body:       "Who did it?",
				Difficulty: "hard",
				FreeForAll: tc.freeForAll,
				Knox:       dto.DefaultKnoxContract(),
			})

			// then
			require.NoError(t, err)

			row := mysteryGet(t, repos, created.ID)
			require.NotNil(t, row)
			assert.Equal(t, "The Murder", row.Title)
			assert.Equal(t, "Who did it?", row.Body)
			assert.Equal(t, "hard", row.Difficulty)
			assert.Equal(t, tc.freeForAll, row.FreeForAll)
			assert.False(t, row.Solved)
			assert.Equal(t, user.Username, row.AuthorUsername)
			assert.Equal(t, "Author Name", row.AuthorDisplayName)

			authorID, err := repos.Mystery.GetAuthorID(ctx, created.ID)
			require.NoError(t, err)
			assert.Equal(t, user.ID, authorID)
		})
	}
}

func TestMysteryDAO_GetByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	row, err := repos.Mystery.GetByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestMysteryDAO_UnknownIDsReadAsNotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	missing := uuid.New()

	// when
	_, authorErr := repos.Mystery.GetAuthorID(ctx, missing)
	_, solvedErr := repos.Mystery.IsSolved(ctx, missing)
	_, pausedErr := repos.Mystery.IsPaused(ctx, missing)
	_, attemptAuthorErr := repos.Mystery.GetAttemptAuthorID(ctx, missing)
	_, attemptMysteryErr := repos.Mystery.GetAttemptMysteryID(ctx, missing)

	// then
	assert.ErrorIs(t, authorErr, dao.ErrNotFound)
	assert.ErrorIs(t, solvedErr, dao.ErrNotFound)
	assert.ErrorIs(t, pausedErr, dao.ErrNotFound)
	assert.ErrorIs(t, attemptAuthorErr, dao.ErrNotFound)
	assert.ErrorIs(t, attemptMysteryErr, dao.ErrNotFound)
}

func TestMysteryDAO_Update(t *testing.T) {
	cases := []struct {
		name           string
		byOwner        bool
		wantErr        bool
		wantTitle      string
		wantBody       string
		wantDifficulty string
	}{
		{name: "the owner can change the title, body and difficulty", byOwner: true, wantErr: false, wantTitle: "New", wantBody: "new body", wantDifficulty: "hard"},
		{name: "a stranger cannot edit someone else's mystery", byOwner: false, wantErr: true, wantTitle: "T", wantBody: "body", wantDifficulty: "easy"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, owner, id := mysteryByGM(t)

			editor := owner
			if !tc.byOwner {
				editor = daotest.CreateUser(t, repos)
			}

			// when
			err := repos.Mystery.Update(context.Background(), spec.MysteryOwnerUpdate{ID: id, UserID: editor.ID, Title: "New", Body: "new body", Difficulty: "hard"})

			// then
			mysteryCheckErr(t, tc.wantErr, err)

			row := mysteryGet(t, repos, id)
			require.NotNil(t, row)
			assert.Equal(t, tc.wantTitle, row.Title)
			assert.Equal(t, tc.wantBody, row.Body)
			assert.Equal(t, tc.wantDifficulty, row.Difficulty)
		})
	}
}

func TestMysteryDAO_UpdateAsAdmin(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)

	// when
	err := repos.Mystery.UpdateAsAdmin(context.Background(), spec.MysteryUpdate{
		ID:         id,
		Title:      "Admin Title",
		Body:       "Admin Body",
		Difficulty: "nightmare",
		FreeForAll: true,
		Knox:       dto.DefaultKnoxContract(),
	})

	// then
	require.NoError(t, err)

	row := mysteryGet(t, repos, id)
	require.NotNil(t, row)
	assert.Equal(t, "Admin Title", row.Title)
	assert.Equal(t, "Admin Body", row.Body)
	assert.Equal(t, "nightmare", row.Difficulty)
	assert.True(t, row.FreeForAll)
}

func TestMysteryRepo_CreateWithClues(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.Mystery.CreateWithClues(context.Background(), spec.NewMysteryWithClues{
		NewMystery: spec.NewMystery{
			UserID:     user.ID,
			Title:      "Locked Room",
			Body:       "How?",
			Difficulty: "hard",
			Knox:       dto.DefaultKnoxContract(),
		},
		Clues: []spec.NewClue{
			{Body: "first", TruthType: "red", SortOrder: 0},
			{Body: "second", TruthType: "blue", SortOrder: 1},
		},
	})

	// then
	require.NoError(t, err)

	row := mysteryGet(t, repos, created.ID)
	require.NotNil(t, row)
	assert.Equal(t, "Locked Room", row.Title)
	assert.Equal(t, []string{"first", "second"}, mysteryClueBodies(t, repos, created.ID))
}

func TestMysteryRepo_UpdateWithClues_ReplacesPublicCluesOnly(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)
	player := daotest.CreateUser(t, repos)
	mysteryAddClue(t, repos, id, spec.NewClue{Body: "old public", TruthType: "red", SortOrder: 0})
	mysteryAddClue(t, repos, id, spec.NewClue{Body: "private", TruthType: "red", SortOrder: 1, PlayerID: &player.ID})

	// when
	err := repos.Mystery.UpdateWithClues(context.Background(), spec.MysteryUpdateWithClues{
		MysteryUpdate: spec.MysteryUpdate{
			ID:         id,
			Title:      "New Title",
			Body:       "New Body",
			Difficulty: "nightmare",
			FreeForAll: true,
			Knox:       dto.DefaultKnoxContract(),
		},
		Clues: []spec.NewClue{{Body: "new public", TruthType: "blue", SortOrder: 0}},
	})

	// then
	require.NoError(t, err)

	row := mysteryGet(t, repos, id)
	require.NotNil(t, row)
	assert.Equal(t, "New Title", row.Title)
	assert.Equal(t, "nightmare", row.Difficulty)
	assert.True(t, row.FreeForAll)
	assert.Equal(t, []string{"new public", "private"}, mysteryClueBodies(t, repos, id))
}

func TestMysteryRepo_UpdateWithClues_ClueFailureRollsBackTheUpdate(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)

	// when
	err := repos.Mystery.UpdateWithClues(context.Background(), spec.MysteryUpdateWithClues{
		MysteryUpdate: spec.MysteryUpdate{
			ID:         id,
			Title:      "New Title",
			Body:       "New Body",
			Difficulty: "nightmare",
			Knox:       dto.DefaultKnoxContract(),
		},
		Clues: []spec.NewClue{{Body: "doomed", TruthType: "red", SortOrder: -1}},
	})

	// then
	require.Error(t, err)

	row := mysteryGet(t, repos, id)
	require.NotNil(t, row)
	assert.Equal(t, "T", row.Title)
	assert.Equal(t, "easy", row.Difficulty)
}

func TestMysteryDAO_Delete(t *testing.T) {
	cases := []struct {
		name     string
		byOwner  bool
		wantErr  bool
		wantKept bool
	}{
		{name: "the owner can delete their mystery", byOwner: true, wantErr: false, wantKept: false},
		{name: "a stranger cannot delete someone else's mystery", byOwner: false, wantErr: true, wantKept: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, owner, id := mysteryByGM(t)

			deleter := owner
			if !tc.byOwner {
				deleter = daotest.CreateUser(t, repos)
			}

			// when
			err := repos.Mystery.Delete(context.Background(), spec.OwnedDeletion{ID: id, UserID: deleter.ID})

			// then
			mysteryCheckErr(t, tc.wantErr, err)
			assert.Equal(t, tc.wantKept, mysteryGet(t, repos, id) != nil)
		})
	}
}

func TestMysteryDAO_DeleteAsAdmin(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)

	// when
	err := repos.Mystery.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Nil(t, mysteryGet(t, repos, id))
}

func TestMysteryDAO_List(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	gm := daotest.CreateUser(t, repos)
	rival := daotest.CreateUser(t, repos)
	oldest := createMystery(t, repos, gm.ID, "oldest", "easy", false)
	middle := createMystery(t, repos, gm.ID, "middle", "easy", false)
	newest := createMystery(t, repos, rival.ID, "newest", "easy", false)
	mysteryBackdate(t, repos, oldest, "2024-01-01 00:00:00")
	mysteryBackdate(t, repos, middle, "2024-01-02 00:00:00")
	mysteryBackdate(t, repos, newest, "2024-01-03 00:00:00")
	mysterySolve(t, repos, oldest, createAttempt(t, repos, oldest, rival.ID, nil, "answer"))

	cases := []struct {
		name      string
		filter    spec.MysteryListFilter
		want      []uuid.UUID
		wantTotal int
	}{
		{name: "the new sort lists newest first", filter: spec.MysteryListFilter{Sort: "new", Limit: 10}, want: []uuid.UUID{newest, middle, oldest}, wantTotal: 3},
		{name: "the old sort lists oldest first", filter: spec.MysteryListFilter{Sort: "old", Limit: 10}, want: []uuid.UUID{oldest, middle, newest}, wantTotal: 3},
		{name: "a first page stops at the limit while the total counts every mystery", filter: spec.MysteryListFilter{Sort: "new", Limit: 2, Offset: 0}, want: []uuid.UUID{newest, middle}, wantTotal: 3},
		{name: "a later page starts at the offset", filter: spec.MysteryListFilter{Sort: "new", Limit: 2, Offset: 2}, want: []uuid.UUID{oldest}, wantTotal: 3},
		{name: "the solved filter keeps only solved mysteries", filter: spec.MysteryListFilter{Sort: "new", Solved: new(true), Limit: 10}, want: []uuid.UUID{oldest}, wantTotal: 1},
		{name: "the unsolved filter keeps only unsolved mysteries", filter: spec.MysteryListFilter{Sort: "new", Solved: new(false), Limit: 10}, want: []uuid.UUID{newest, middle}, wantTotal: 2},
		{name: "excluded authors drop out of the rows and the total", filter: spec.MysteryListFilter{Sort: "new", Limit: 10, ExcludeUserIDs: []uuid.UUID{gm.ID}}, want: []uuid.UUID{newest}, wantTotal: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			rows, total, err := repos.Mystery.List(ctx, tc.filter)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, mysteryRowIDs(rows))
			assert.Equal(t, tc.wantTotal, total)
		})
	}
}

func TestMysteryDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	older := createMystery(t, repos, user.ID, "older", "easy", false)
	newer := createMystery(t, repos, user.ID, "newer", "easy", false)
	theirs := createMystery(t, repos, other.ID, "theirs", "easy", false)
	mysteryBackdate(t, repos, older, "2024-01-01 00:00:00")
	mysteryBackdate(t, repos, newer, "2024-01-02 00:00:00")
	mysteryBackdate(t, repos, theirs, "2024-01-03 00:00:00")

	cases := []struct {
		name      string
		limit     int
		offset    int
		want      []uuid.UUID
		wantTotal int
	}{
		{name: "lists only the user's own mysteries, newest first", limit: 10, offset: 0, want: []uuid.UUID{newer, older}, wantTotal: 2},
		{name: "a page from an offset still totals every one of the user's mysteries", limit: 1, offset: 1, want: []uuid.UUID{older}, wantTotal: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			rows, total, err := repos.Mystery.ListByUser(ctx, spec.MysteryUserListFilter{UserID: user.ID, Limit: tc.limit, Offset: tc.offset})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, mysteryRowIDs(rows))
			assert.Equal(t, tc.wantTotal, total)
		})
	}
}

func TestMysteryDAO_GetClues(t *testing.T) {
	// given
	fx := mysteryWithClues(t)
	ctx := context.Background()

	// when
	clues, err := fx.repos.Mystery.GetClues(ctx, fx.mysteryID)

	// then
	require.NoError(t, err)
	require.Len(t, clues, 3)
	assert.Equal(t, "old", clues[0].Body)
	assert.Nil(t, clues[0].PlayerID)
	assert.Equal(t, "other", clues[1].Body)
	assert.Nil(t, clues[1].PlayerID)
	assert.Equal(t, "private", clues[2].Body)
	require.NotNil(t, clues[2].PlayerID)
	assert.Equal(t, fx.playerID, *clues[2].PlayerID)

	count, err := fx.repos.Mystery.CountClues(ctx, fx.mysteryID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	row := mysteryGet(t, fx.repos, fx.mysteryID)
	require.NotNil(t, row)
	assert.Equal(t, 3, row.ClueCount)
}

func TestMysteryDAO_DeleteClues_SkipsPrivate(t *testing.T) {
	// given
	fx := mysteryWithClues(t)

	// when
	err := fx.repos.Mystery.DeleteClues(context.Background(), fx.mysteryID)

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"private"}, mysteryClueBodies(t, fx.repos, fx.mysteryID))
}

func TestMysteryDAO_DeleteClue(t *testing.T) {
	// given
	fx := mysteryWithClues(t)

	// when
	err := fx.repos.Mystery.DeleteClue(context.Background(), fx.oldClueID)

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"other", "private"}, mysteryClueBodies(t, fx.repos, fx.mysteryID))
}

func TestMysteryDAO_UpdateClue(t *testing.T) {
	// given
	fx := mysteryWithClues(t)

	// when
	err := fx.repos.Mystery.UpdateClue(context.Background(), spec.MysteryClueUpdate{ClueID: fx.oldClueID, Body: "new"})

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"new", "other", "private"}, mysteryClueBodies(t, fx.repos, fx.mysteryID))
}

func TestMysteryDAO_CreateAttempt(t *testing.T) {
	// given
	repos, gm, id := mysteryByGM(t)
	ctx := context.Background()
	player := daotest.CreateUser(t, repos)

	// when
	rootID := createAttempt(t, repos, id, player.ID, nil, "the answer")
	replyID := createAttempt(t, repos, id, gm.ID, &rootID, "reply")

	// then
	attempts := mysteryAttempts(t, repos, id, player.ID)
	require.Len(t, attempts, 2)
	assert.Equal(t, "the answer", attempts[rootID].Body)
	assert.False(t, attempts[rootID].IsWinner)
	assert.Nil(t, attempts[rootID].ParentID)
	require.NotNil(t, attempts[replyID].ParentID)
	assert.Equal(t, rootID, *attempts[replyID].ParentID)

	authorID, err := repos.Mystery.GetAttemptAuthorID(ctx, rootID)
	require.NoError(t, err)
	assert.Equal(t, player.ID, authorID)

	mysteryID, err := repos.Mystery.GetAttemptMysteryID(ctx, rootID)
	require.NoError(t, err)
	assert.Equal(t, id, mysteryID)
}

func TestMysteryDAO_AttemptCounts(t *testing.T) {
	// given
	repos, gm, id := mysteryByGM(t)
	ctx := context.Background()
	p1 := daotest.CreateUser(t, repos)
	p2 := daotest.CreateUser(t, repos)
	createAttempt(t, repos, id, p1.ID, nil, "a")
	createAttempt(t, repos, id, p2.ID, nil, "b")
	createAttempt(t, repos, id, gm.ID, nil, "gm")
	parentID := createAttempt(t, repos, id, p1.ID, nil, "parent")
	createAttempt(t, repos, id, p1.ID, &parentID, "reply")

	// when
	row := mysteryGet(t, repos, id)

	// then
	require.NotNil(t, row)
	assert.Equal(t, 3, row.AttemptCount)

	count, err := repos.Mystery.CountAttempts(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	playerIDs, err := repos.Mystery.GetPlayerIDs(ctx, id)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{p1.ID, p2.ID}, playerIDs)
}

func TestMysteryDAO_DeleteAttempt(t *testing.T) {
	cases := []struct {
		name          string
		byAuthor      bool
		wantErr       bool
		wantRemaining int
	}{
		{name: "the author can delete their attempt", byAuthor: true, wantErr: false, wantRemaining: 0},
		{name: "a stranger cannot delete someone else's attempt", byAuthor: false, wantErr: true, wantRemaining: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, _, id := mysteryByGM(t)
			player := daotest.CreateUser(t, repos)
			attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

			deleter := player
			if !tc.byAuthor {
				deleter = daotest.CreateUser(t, repos)
			}

			// when
			err := repos.Mystery.DeleteAttempt(context.Background(), spec.MysteryAttemptDeletion{ID: attemptID, UserID: deleter.ID})

			// then
			mysteryCheckErr(t, tc.wantErr, err)
			assert.Len(t, mysteryAttempts(t, repos, id, player.ID), tc.wantRemaining)
		})
	}
}

func TestMysteryDAO_DeleteAttemptAsAdmin(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)
	player := daotest.CreateUser(t, repos)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.DeleteAttemptAsAdmin(context.Background(), attemptID)

	// then
	require.NoError(t, err)
	assert.Empty(t, mysteryAttempts(t, repos, id, player.ID))
}

func TestMysteryDAO_VoteAttempt(t *testing.T) {
	cases := []struct {
		name         string
		votes        []mysteryVote
		wantScore    int
		wantUserVote int
	}{
		{name: "an upvote scores one and is the viewer's own vote", votes: []mysteryVote{{voter: 0, value: 1}}, wantScore: 1, wantUserVote: 1},
		{name: "votes from several users are summed", votes: []mysteryVote{{voter: 0, value: 1}, {voter: 1, value: 1}, {voter: 2, value: -1}}, wantScore: 1, wantUserVote: 1},
		{name: "voting again replaces the earlier vote", votes: []mysteryVote{{voter: 0, value: 1}, {voter: 0, value: -1}}, wantScore: -1, wantUserVote: -1},
		{name: "a zero vote clears the earlier vote", votes: []mysteryVote{{voter: 0, value: 1}, {voter: 0, value: 0}}, wantScore: 0, wantUserVote: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, _, id := mysteryByGM(t)
			ctx := context.Background()
			player := daotest.CreateUser(t, repos)
			voters := []uuid.UUID{daotest.CreateUser(t, repos).ID, daotest.CreateUser(t, repos).ID, daotest.CreateUser(t, repos).ID}
			attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

			// when
			for _, vote := range tc.votes {
				require.NoError(t, repos.Mystery.VoteAttempt(ctx, spec.Vote{UserID: voters[vote.voter], TargetID: attemptID, Value: vote.value}))
			}

			// then
			attempts := mysteryAttempts(t, repos, id, voters[0])
			require.Len(t, attempts, 1)
			assert.Equal(t, tc.wantScore, attempts[attemptID].VoteScore)
			assert.Equal(t, tc.wantUserVote, attempts[attemptID].UserVote)
		})
	}
}

func TestMysteryDAO_MarkSolved(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)
	ctx := context.Background()
	player := daotest.CreateUser(t, repos)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	solvedBefore, err := repos.Mystery.IsSolved(ctx, id)
	require.NoError(t, err)

	// when
	err = repos.Mystery.MarkSolved(ctx, spec.MysterySolve{MysteryID: id, AttemptID: attemptID, LockMystery: true})

	// then
	require.NoError(t, err)
	assert.False(t, solvedBefore)

	solvedAfter, err := repos.Mystery.IsSolved(ctx, id)
	require.NoError(t, err)
	assert.True(t, solvedAfter)

	row := mysteryGet(t, repos, id)
	require.NotNil(t, row)
	assert.True(t, row.Solved)
	require.NotNil(t, row.WinnerID)
	assert.Equal(t, player.ID, *row.WinnerID)

	attempts := mysteryAttempts(t, repos, id, player.ID)
	require.Len(t, attempts, 1)
	assert.True(t, attempts[attemptID].IsWinner)
}

func TestMysteryDAO_MarkSolved_PreservesPreviousWinner(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)
	p1 := daotest.CreateUser(t, repos)
	p2 := daotest.CreateUser(t, repos)
	a1 := createAttempt(t, repos, id, p1.ID, nil, "first")
	a2 := createAttempt(t, repos, id, p2.ID, nil, "second")
	mysterySolve(t, repos, id, a1)

	// when
	err := repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: id, AttemptID: a2, LockMystery: true})

	// then
	require.NoError(t, err)

	attempts := mysteryAttempts(t, repos, id, p2.ID)
	require.Len(t, attempts, 2)
	assert.True(t, attempts[a1].IsWinner)
	assert.True(t, attempts[a2].IsWinner)
}

func TestMysteryDAO_MarkSolved_MismatchMysteryFails(t *testing.T) {
	// given
	repos, gm, m1 := mysteryByGM(t)
	ctx := context.Background()
	player := daotest.CreateUser(t, repos)
	m2 := createMystery(t, repos, gm.ID, "T2", "easy", false)
	attemptID := createAttempt(t, repos, m1, player.ID, nil, "a")

	// when
	err := repos.Mystery.MarkSolved(ctx, spec.MysterySolve{MysteryID: m2, AttemptID: attemptID, LockMystery: true})

	// then
	require.Error(t, err)

	solved, err := repos.Mystery.IsSolved(ctx, m2)
	require.NoError(t, err)
	assert.False(t, solved)
}

func TestMysteryDAO_SetPaused(t *testing.T) {
	cases := []struct {
		name       string
		steps      []bool
		wantPaused bool
	}{
		{name: "pausing marks the mystery paused and stamps when", steps: []bool{true}, wantPaused: true},
		{name: "unpausing clears the flag and the stamp", steps: []bool{true, false}, wantPaused: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, _, id := mysteryByGM(t)
			ctx := context.Background()

			// when
			for _, paused := range tc.steps {
				require.NoError(t, repos.Mystery.SetPaused(ctx, spec.MysteryPauseUpdate{MysteryID: id, Paused: paused}))
			}

			// then
			paused, err := repos.Mystery.IsPaused(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPaused, paused)

			row := mysteryGet(t, repos, id)
			require.NotNil(t, row)
			assert.Equal(t, tc.wantPaused, row.PausedAt != nil)
		})
	}
}

func TestMysteryDAO_SetGmAway(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)
	ctx := context.Background()

	// when
	require.NoError(t, repos.Mystery.SetGmAway(ctx, spec.MysteryGmAwayUpdate{MysteryID: id, Away: true}))
	awayRow := mysteryGet(t, repos, id)
	require.NoError(t, repos.Mystery.SetGmAway(ctx, spec.MysteryGmAwayUpdate{MysteryID: id, Away: false}))
	backRow := mysteryGet(t, repos, id)

	// then
	require.NotNil(t, awayRow)
	require.NotNil(t, backRow)
	assert.True(t, awayRow.GmAway)
	assert.False(t, backRow.GmAway)
}

func TestMysteryDAO_GetLeaderboard(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	gm := daotest.CreateUser(t, repos)
	first := daotest.CreateUser(t, repos, daotest.WithDisplayName("Zelda"))
	second := daotest.CreateUser(t, repos, daotest.WithDisplayName("Mallory"))
	third := daotest.CreateUser(t, repos, daotest.WithDisplayName("Alice"))
	mysterySolvedBy(t, repos, gm.ID, first.ID, "nightmare")
	mysterySolvedBy(t, repos, gm.ID, first.ID, "easy")
	mysterySolvedBy(t, repos, gm.ID, second.ID, "easy")
	mysterySolvedBy(t, repos, gm.ID, second.ID, "hard")
	mysterySolvedBy(t, repos, gm.ID, third.ID, "easy")

	// when
	entries, err := repos.Mystery.GetLeaderboard(ctx, 10)

	// then
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, first.ID, entries[0].UserID)
	assert.Equal(t, 10, entries[0].Score)
	assert.Equal(t, 1, entries[0].NightmareSolved)
	assert.Equal(t, 1, entries[0].EasySolved)
	assert.Equal(t, second.ID, entries[1].UserID)
	assert.Equal(t, 8, entries[1].Score)
	assert.Equal(t, 1, entries[1].EasySolved)
	assert.Equal(t, 1, entries[1].HardSolved)
	assert.Equal(t, third.ID, entries[2].UserID)
	assert.Equal(t, 2, entries[2].Score)

	topIDs, err := repos.Mystery.GetTopDetectiveIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{first.ID.String()}, topIDs)
}

func TestMysteryDAO_GetGMLeaderboard(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	gm := daotest.CreateUser(t, repos, daotest.WithDisplayName("Ruler"))
	player := daotest.CreateUser(t, repos)
	mysterySolvedBy(t, repos, gm.ID, player.ID, "hard")
	createMystery(t, repos, gm.ID, "still open", "easy", false)

	// when
	entries, err := repos.Mystery.GetGMLeaderboard(ctx, 10)

	// then
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, gm.ID, entries[0].UserID)
	assert.Equal(t, 1, entries[0].MysteryCount)
	assert.Equal(t, 1, entries[0].PlayerCount)
	assert.Equal(t, 7, entries[0].Score)

	topIDs, err := repos.Mystery.GetTopGMIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{gm.ID.String()}, topIDs)
}

func TestMysteryDAO_CreateComment(t *testing.T) {
	// given
	repos, gm, id := mysteryByGM(t)
	ctx := context.Background()
	commenter := daotest.CreateUser(t, repos)

	// when
	parentID := createMysteryComment(t, repos, id, nil, commenter.ID, "nice mystery")
	replyID := createMysteryComment(t, repos, id, &parentID, gm.ID, "reply")

	// then
	comments := mysteryComments(t, repos, id, commenter.ID)
	require.Len(t, comments, 2)
	assert.Equal(t, "nice mystery", comments[parentID].Body)
	assert.Nil(t, comments[parentID].ParentID)
	require.NotNil(t, comments[replyID].ParentID)
	assert.Equal(t, parentID, *comments[replyID].ParentID)

	entityID, err := repos.Mystery.GetCommentEntityID(ctx, parentID)
	require.NoError(t, err)
	assert.Equal(t, id, entityID)

	authorID, err := repos.Mystery.GetCommentAuthorID(ctx, parentID)
	require.NoError(t, err)
	assert.Equal(t, commenter.ID, authorID)
}

func TestMysteryDAO_GetComments_ExcludeUsers(t *testing.T) {
	// given
	repos, gm, id := mysteryByGM(t)
	c1 := daotest.CreateUser(t, repos)
	c2 := daotest.CreateUser(t, repos)
	createMysteryComment(t, repos, id, nil, c1.ID, "A")
	keepID := createMysteryComment(t, repos, id, nil, c2.ID, "B")

	// when
	comments, total, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID:       id,
		ViewerID:       gm.ID,
		Limit:          500,
		Offset:         0,
		ExcludeUserIDs: []uuid.UUID{c1.ID},
	})

	// then
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, keepID, comments[0].ID)
	assert.Equal(t, 1, total)
}

func TestMysteryDAO_UpdateComment(t *testing.T) {
	cases := []struct {
		name     string
		byAuthor bool
		asAdmin  bool
		wantErr  bool
		wantBody string
	}{
		{name: "the author can edit their comment and it is stamped as edited", byAuthor: true, asAdmin: false, wantErr: false, wantBody: "new body"},
		{name: "a stranger cannot edit someone else's comment", byAuthor: false, asAdmin: false, wantErr: true, wantBody: "old"},
		{name: "an admin can edit anyone's comment", byAuthor: false, asAdmin: true, wantErr: false, wantBody: "new body"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, _, id := mysteryByGM(t)
			commenter := daotest.CreateUser(t, repos)
			commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "old")

			editor := commenter
			if !tc.byAuthor {
				editor = daotest.CreateUser(t, repos)
			}

			// when
			err := repos.Mystery.UpdateComment(context.Background(), spec.CommentUpdate{CommentID: commentID, UserID: editor.ID, Body: "new body", AsAdmin: tc.asAdmin})

			// then
			mysteryCheckErr(t, tc.wantErr, err)

			comments := mysteryComments(t, repos, id, commenter.ID)
			require.Len(t, comments, 1)
			assert.Equal(t, tc.wantBody, comments[commentID].Body)
			assert.Equal(t, !tc.wantErr, comments[commentID].UpdatedAt != nil)
		})
	}
}

func TestMysteryRepo_DeleteCommentWithAudit(t *testing.T) {
	cases := []struct {
		name       string
		asAdmin    bool
		wantAction audit.Action
	}{
		{name: "the owner deleting their own comment", asAdmin: false, wantAction: audit.ActionMysteryCommentDelete},
		{name: "a moderator deleting someone else's comment", asAdmin: true, wantAction: audit.ActionMysteryCommentDeleteAdmin},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, gm, id := mysteryByGM(t)
			ctx := context.Background()
			commenter := daotest.CreateUser(t, repos)
			commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "a clue")
			mysteryAddCommentMedia(t, repos, commentID, "/uploads/mystery/reply.png", "/uploads/mystery/reply_thumb.png")
			keptID := createMysteryComment(t, repos, id, nil, commenter.ID, "another clue")
			mysteryAddCommentMedia(t, repos, keptID, "/uploads/mystery/keep.png", "")

			actor := commenter
			if tc.asAdmin {
				actor = gm
			}

			// when
			paths, err := repos.Mystery.DeleteCommentWithAudit(ctx, spec.CommentDeletion{
				CommentID: commentID,
				UserID:    actor.ID,
				AsAdmin:   tc.asAdmin,
			})

			// then
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{"/uploads/mystery/reply.png", "/uploads/mystery/reply_thumb.png"}, paths)

			comments := mysteryComments(t, repos, id, commenter.ID)
			assert.Len(t, comments, 1)
			assert.Contains(t, comments, keptID)

			media, err := repos.Mystery.GetCommentMedia(ctx, keptID)
			require.NoError(t, err)
			require.Len(t, media, 1)
			assert.Equal(t, "/uploads/mystery/keep.png", media[0].MediaURL)

			entries, total, err := repos.AuditLog.List(ctx, spec.AuditLogListing{Action: tc.wantAction, Page: bounds.NewPage(10, 0)})
			require.NoError(t, err)
			assert.Equal(t, 1, total)
			require.Len(t, entries, 1)
			assert.Equal(t, actor.ID, entries[0].ActorID)
			assert.Equal(t, audit.TargetMysteryComment, entries[0].TargetType)
			assert.Equal(t, commentID.String(), entries[0].TargetID)
		})
	}
}

func TestMysteryRepo_DeleteCommentWithAudit_NotOwnedLeavesEverythingInPlace(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)
	ctx := context.Background()
	commenter := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "a clue")
	mysteryAddCommentMedia(t, repos, commentID, "/uploads/mystery/reply.png", "/uploads/mystery/reply_thumb.png")

	// when
	paths, err := repos.Mystery.DeleteCommentWithAudit(ctx, spec.CommentDeletion{CommentID: commentID, UserID: stranger.ID})

	// then
	require.Error(t, err)
	assert.Empty(t, paths)

	comments := mysteryComments(t, repos, id, commenter.ID)
	assert.Len(t, comments, 1)
	assert.Contains(t, comments, commentID)

	media, err := repos.Mystery.GetCommentMedia(ctx, commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/uploads/mystery/reply.png", media[0].MediaURL)
	assert.Equal(t, "/uploads/mystery/reply_thumb.png", media[0].ThumbnailURL)

	_, total, err := repos.AuditLog.List(ctx, spec.AuditLogListing{Action: "", Page: bounds.NewPage(10, 0)})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestMysteryDAO_LikeComment(t *testing.T) {
	like := repository.MysteryRepository.LikeComment
	unlike := repository.MysteryRepository.UnlikeComment

	cases := []struct {
		name      string
		actions   []mysteryLikeAction
		wantCount int
		wantLiked bool
	}{
		{name: "a like is counted and marked as the viewer's own", actions: []mysteryLikeAction{like}, wantCount: 1, wantLiked: true},
		{name: "liking twice is idempotent", actions: []mysteryLikeAction{like, like}, wantCount: 1, wantLiked: true},
		{name: "unliking removes the like", actions: []mysteryLikeAction{like, unlike}, wantCount: 0, wantLiked: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, _, id := mysteryByGM(t)
			ctx := context.Background()
			commenter := daotest.CreateUser(t, repos)
			liker := daotest.CreateUser(t, repos)
			commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

			// when
			for _, act := range tc.actions {
				require.NoError(t, act(repos.Mystery, ctx, spec.CommentLike{UserID: liker.ID, CommentID: commentID}))
			}

			// then
			comments := mysteryComments(t, repos, id, liker.ID)
			require.Len(t, comments, 1)
			assert.Equal(t, tc.wantCount, comments[commentID].LikeCount)
			assert.Equal(t, tc.wantLiked, comments[commentID].UserLiked)
		})
	}
}

func TestMysteryDAO_Media(t *testing.T) {
	cases := []mysteryMediaSurface{mysteryEntityMedia, mysteryCommentMedia}

	for _, tc := range cases {
		t.Run(tc.name+" reads back in the order it was added", func(t *testing.T) {
			// given
			repos, gm, id := mysteryByGM(t)
			ctx := context.Background()
			targetID := tc.target(t, repos, id, gm.ID)

			// when
			firstID, errFirst := tc.add(repos.Mystery, ctx, spec.NewMedia{TargetID: targetID, MediaURL: "/a.png", MediaType: "image", ThumbnailURL: "/t.png"})
			_, errSecond := tc.add(repos.Mystery, ctx, spec.NewMedia{TargetID: targetID, MediaURL: "/b.png", MediaType: "image"})

			// then
			require.NoError(t, errFirst)
			require.NoError(t, errSecond)
			assert.NotZero(t, firstID)

			media, err := tc.get(repos.Mystery, ctx, targetID)
			require.NoError(t, err)
			require.Len(t, media, 2)
			assert.Equal(t, "/a.png", media[0].MediaURL)
			assert.Equal(t, "image", media[0].MediaType)
			assert.Equal(t, "/t.png", media[0].ThumbnailURL)
			assert.Equal(t, 0, media[0].SortOrder)
			assert.Equal(t, "/b.png", media[1].MediaURL)
			assert.Equal(t, 1, media[1].SortOrder)
		})
	}
}

func TestMysteryDAO_UpdateMedia(t *testing.T) {
	cases := []struct {
		name          string
		surface       mysteryMediaSurface
		update        func(repository.MysteryRepository, context.Context, spec.MediaURLUpdate, ...*sql.Tx) error
		wantURL       string
		wantThumbnail string
	}{
		{name: "UpdateMediaURL swaps a mystery image's url and keeps its thumbnail", surface: mysteryEntityMedia, update: repository.MysteryRepository.UpdateMediaURL, wantURL: "/new.png", wantThumbnail: "/old.png"},
		{name: "UpdateMediaThumbnail swaps a mystery image's thumbnail and keeps its url", surface: mysteryEntityMedia, update: repository.MysteryRepository.UpdateMediaThumbnail, wantURL: "/a.png", wantThumbnail: "/new.png"},
		{name: "UpdateCommentMediaURL swaps a comment image's url and keeps its thumbnail", surface: mysteryCommentMedia, update: repository.MysteryRepository.UpdateCommentMediaURL, wantURL: "/new.png", wantThumbnail: "/old.png"},
		{name: "UpdateCommentMediaThumbnail swaps a comment image's thumbnail and keeps its url", surface: mysteryCommentMedia, update: repository.MysteryRepository.UpdateCommentMediaThumbnail, wantURL: "/a.png", wantThumbnail: "/new.png"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, gm, id := mysteryByGM(t)
			ctx := context.Background()
			targetID := tc.surface.target(t, repos, id, gm.ID)
			mediaID, err := tc.surface.add(repos.Mystery, ctx, spec.NewMedia{TargetID: targetID, MediaURL: "/a.png", MediaType: "image", ThumbnailURL: "/old.png"})
			require.NoError(t, err)

			// when
			err = tc.update(repos.Mystery, ctx, spec.MediaURLUpdate{ID: mediaID, URL: "/new.png"})

			// then
			require.NoError(t, err)

			media, err := tc.surface.get(repos.Mystery, ctx, targetID)
			require.NoError(t, err)
			require.Len(t, media, 1)
			assert.Equal(t, tc.wantURL, media[0].MediaURL)
			assert.Equal(t, tc.wantThumbnail, media[0].ThumbnailURL)
		})
	}
}

func TestMysteryDAO_GetCommentMediaBatch(t *testing.T) {
	// given
	repos, gm, id := mysteryByGM(t)
	ctx := context.Background()
	c1 := createMysteryComment(t, repos, id, nil, gm.ID, "a")
	c2 := createMysteryComment(t, repos, id, nil, gm.ID, "b")
	m1 := mysteryAddCommentMedia(t, repos, c1, "/c1.png", "")
	m2 := mysteryAddCommentMedia(t, repos, c2, "/c2.png", "")

	cases := []struct {
		name string
		ids  []uuid.UUID
		want map[uuid.UUID][]model.PostMediaRow
	}{
		{
			name: "each requested comment maps to its own media",
			ids:  []uuid.UUID{c1, c2},
			want: map[uuid.UUID][]model.PostMediaRow{
				c1: {{ID: int(m1), PostID: c1, MediaURL: "/c1.png", MediaType: "image"}},
				c2: {{ID: int(m2), PostID: c2, MediaURL: "/c2.png", MediaType: "image"}},
			},
		},
		{name: "an empty id list short-circuits to nil", ids: nil, want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			result, err := repos.Mystery.GetCommentMediaBatch(ctx, tc.ids)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestMysteryDAO_AddAttachment_AndGet(t *testing.T) {
	// given
	repos, _, id := mysteryByGM(t)
	ctx := context.Background()

	// when
	attID, err := repos.Mystery.AddAttachment(ctx, spec.NewMysteryAttachment{
		MysteryID: id,
		FileURL:   "/file.pdf",
		FileName:  "file.pdf",
		FileSize:  1234,
	})

	// then
	require.NoError(t, err)
	assert.NotZero(t, attID)

	atts, err := repos.Mystery.GetAttachments(ctx, id)
	require.NoError(t, err)
	require.Len(t, atts, 1)
	assert.Equal(t, "/file.pdf", atts[0].FileURL)
	assert.Equal(t, "file.pdf", atts[0].FileName)
	assert.Equal(t, 1234, atts[0].FileSize)
}

func TestMysteryDAO_DeleteAttachment(t *testing.T) {
	cases := []struct {
		name          string
		viaOther      bool
		wantErr       bool
		wantRemaining int
	}{
		{name: "an attachment is deleted through its own mystery", viaOther: false, wantErr: false, wantRemaining: 0},
		{name: "an attachment cannot be deleted through another mystery", viaOther: true, wantErr: true, wantRemaining: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, gm, own := mysteryByGM(t)
			ctx := context.Background()
			other := createMystery(t, repos, gm.ID, "B", "easy", false)
			attID, err := repos.Mystery.AddAttachment(ctx, spec.NewMysteryAttachment{MysteryID: own, FileURL: "/f.pdf", FileName: "f.pdf", FileSize: 1})
			require.NoError(t, err)

			via := own
			if tc.viaOther {
				via = other
			}

			// when
			err = repos.Mystery.DeleteAttachment(ctx, spec.MysteryAttachmentDeletion{ID: attID, MysteryID: via})

			// then
			mysteryCheckErr(t, tc.wantErr, err)

			atts, err := repos.Mystery.GetAttachments(ctx, own)
			require.NoError(t, err)
			assert.Len(t, atts, tc.wantRemaining)
		})
	}
}

func TestMysteryDAO_DeleteMedia(t *testing.T) {
	cases := []struct {
		name          string
		viaOther      bool
		wantErr       bool
		wantURL       string
		wantRemaining int
	}{
		{name: "a media item is deleted through its own mystery and its url handed back", viaOther: false, wantErr: false, wantURL: "/x.png", wantRemaining: 0},
		{name: "a media item cannot be deleted through another mystery", viaOther: true, wantErr: true, wantURL: "", wantRemaining: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, gm, own := mysteryByGM(t)
			ctx := context.Background()
			other := createMystery(t, repos, gm.ID, "Other", "easy", false)
			mediaID, err := repos.Mystery.AddMedia(ctx, spec.NewMedia{TargetID: own, MediaURL: "/x.png", MediaType: "image"})
			require.NoError(t, err)

			via := own
			if tc.viaOther {
				via = other
			}

			// when
			url, err := repos.Mystery.DeleteMedia(ctx, spec.MediaDeletion{ID: mediaID, TargetID: via})

			// then
			mysteryCheckErr(t, tc.wantErr, err)
			assert.Equal(t, tc.wantURL, url)

			media, err := repos.Mystery.GetMedia(ctx, own)
			require.NoError(t, err)
			assert.Len(t, media, tc.wantRemaining)
		})
	}
}

func TestMysteryRepo_DeleteWithFiles(t *testing.T) {
	everyPath := []string{
		"/uploads/mystery/board.png",
		"/uploads/mystery/board_thumb.png",
		"/uploads/mystery/case.pdf",
		"/uploads/mystery/reply.png",
		"/uploads/mystery/reply_thumb.png",
	}

	cases := []struct {
		name       string
		byOwner    bool
		asAdmin    bool
		uploads    bool
		thumbnails bool
		wantErr    bool
		wantPaths  []string
	}{
		{name: "the owner gets back every uploaded path: media, thumbnails, attachments and comment media", byOwner: true, uploads: true, thumbnails: true, wantPaths: everyPath},
		{name: "an admin deleting someone else's mystery gets back every uploaded path", asAdmin: true, uploads: true, thumbnails: true, wantPaths: everyPath},
		{name: "blank thumbnails are skipped", byOwner: true, uploads: true, thumbnails: false, wantPaths: []string{"/uploads/mystery/board.png", "/uploads/mystery/case.pdf", "/uploads/mystery/reply.png"}},
		{name: "a mystery with no uploads returns no paths", byOwner: true, uploads: false, wantPaths: nil},
		{name: "a stranger deletes nothing and gets no paths", uploads: true, thumbnails: true, wantErr: true, wantPaths: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos, gm, id := mysteryByGM(t)
			ctx := context.Background()
			commenter := daotest.CreateUser(t, repos)

			if tc.uploads {
				mysteryAddUploads(t, repos, id, commenter.ID, tc.thumbnails)
			}

			deleter := gm
			if !tc.byOwner {
				deleter = daotest.CreateUser(t, repos)
			}

			// when
			paths, err := repos.Mystery.DeleteWithFiles(ctx, spec.MysteryDelete{ID: id, UserID: deleter.ID, AsAdmin: tc.asAdmin})

			// then
			mysteryCheckErr(t, tc.wantErr, err)
			assert.ElementsMatch(t, tc.wantPaths, paths)

			row := mysteryGet(t, repos, id)
			media, err := repos.Mystery.GetMedia(ctx, id)
			require.NoError(t, err)

			if tc.wantErr {
				require.NotNil(t, row)
				require.Len(t, media, 1)
				assert.Equal(t, "/uploads/mystery/board.png", media[0].MediaURL)
			} else {
				assert.Nil(t, row)
				assert.Empty(t, media)
			}
		})
	}
}
