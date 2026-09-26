package dao_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	journalMediaFixture struct {
		ownerID       uuid.UUID
		strangerID    uuid.UUID
		journalID     uuid.UUID
		entryID       uuid.UUID
		keptEntryID   uuid.UUID
		commentID     uuid.UUID
		keptCommentID uuid.UUID
	}
)

func createJournal(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title, _body, work string) uuid.UUID {
	t.Helper()
	created, err := repos.Journal.Create(context.Background(), spec.NewJournal{
		UserID: userID,
		Title:  title,
		Work:   work,
	})
	require.NoError(t, err)

	return created.ID
}

func createJournalComment(t *testing.T, repos *repository.Repositories, journalID, userID uuid.UUID, parentID *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Comments.Journal.CreateComment(context.Background(), spec.NewJournalComment{
		JournalID: journalID,
		ParentID:  parentID,
		UserID:    userID,
		Body:      body,
	})
	require.NoError(t, err)

	return created.ID
}

func journalCreateEntry(t *testing.T, repos *repository.Repositories, s spec.NewJournalEntry) uuid.UUID {
	t.Helper()
	created, err := repos.Journal.CreateEntry(context.Background(), s)
	require.NoError(t, err)

	return created.ID
}

func journalBackdate(t *testing.T, repos *repository.Repositories, journalID uuid.UUID, at time.Time) {
	t.Helper()
	_, err := repos.DB().ExecContext(context.Background(), `UPDATE journals SET created_at = $1, last_author_activity_at = $1 WHERE id = $2`, at, journalID)
	require.NoError(t, err)
}

func journalRootComments(t *testing.T, repos *repository.Repositories, journalID, viewerID uuid.UUID) ([]model.CommentRow, int) {
	t.Helper()
	comments, total, err := repos.Journal.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: journalID, ViewerID: viewerID, Limit: 10})
	require.NoError(t, err)

	return comments, total
}

func journalCommentIDs(comments []model.CommentRow) []uuid.UUID {
	var ids []uuid.UUID
	for _, comment := range comments {
		ids = append(ids, comment.ID)
	}

	return ids
}

func journalResponseIDs(journals []dto.JournalResponse) []uuid.UUID {
	var ids []uuid.UUID
	for _, journal := range journals {
		ids = append(ids, journal.ID)
	}

	return ids
}

func journalSeedMedia(t *testing.T, repos *repository.Repositories) journalMediaFixture {
	t.Helper()
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, owner.ID, "T", "", "general")
	entryID := journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 1, Body: "b", WordCount: 1})
	keptEntryID := journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 2, Body: "b", WordCount: 1})
	commentID := createJournalComment(t, repos, journalID, owner.ID, nil, "on journal")
	keptCommentID := createJournalComment(t, repos, journalID, owner.ID, nil, "kept")

	entryComment, err := repos.Comments.Journal.CreateComment(ctx, spec.NewJournalComment{JournalID: journalID, EntryID: &entryID, UserID: owner.ID, Body: "on entry"})
	require.NoError(t, err)

	replyID := createJournalComment(t, repos, journalID, owner.ID, &entryComment.ID, "reply")

	for _, media := range []spec.NewMedia{
		{TargetID: entryID, MediaURL: "/uploads/journal/entry.png", MediaType: "image", ThumbnailURL: "/uploads/journal/entry_thumb.png"},
		{TargetID: entryID, MediaURL: "/uploads/journal/entry_no_thumb.gif", MediaType: "image"},
		{TargetID: keptEntryID, MediaURL: "/uploads/journal/kept_entry.png", MediaType: "image"},
	} {
		_, err := repos.Journal.AddMedia(ctx, media)
		require.NoError(t, err)
	}

	for _, media := range []spec.NewMedia{
		{TargetID: commentID, MediaURL: "/uploads/journal/comment.png", MediaType: "image", ThumbnailURL: "/uploads/journal/comment_thumb.png"},
		{TargetID: keptCommentID, MediaURL: "/uploads/journal/kept_comment.png", MediaType: "image"},
		{TargetID: entryComment.ID, MediaURL: "/uploads/journal/entry_comment.png", MediaType: "image", ThumbnailURL: "/uploads/journal/entry_comment_thumb.png"},
		{TargetID: replyID, MediaURL: "/uploads/journal/reply.png", MediaType: "image"},
	} {
		_, err := repos.Journal.AddCommentMedia(ctx, media)
		require.NoError(t, err)
	}

	return journalMediaFixture{
		ownerID:       owner.ID,
		strangerID:    stranger.ID,
		journalID:     journalID,
		entryID:       entryID,
		keptEntryID:   keptEntryID,
		commentID:     commentID,
		keptCommentID: keptCommentID,
	}
}

func TestJournalDAO_Create(t *testing.T) {
	cases := []struct {
		name     string
		work     string
		wantWork string
	}{
		{name: "an empty work falls back to general", work: "", wantWork: "general"},
		{name: "a named work is kept", work: "umineko", wantWork: "umineko"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Author"))

			// when
			id := createJournal(t, repos, user.ID, "My Title", "", tc.work)

			// then
			got, err := repos.Journal.GetByID(ctx, spec.JournalLookup{ID: id, ViewerID: user.ID})
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, id, got.ID)
			assert.Equal(t, "My Title", got.Title)
			assert.Equal(t, tc.wantWork, got.Work)
			assert.Equal(t, user.ID, got.Author.ID)
			assert.Equal(t, "Author", got.Author.DisplayName)
			assert.False(t, got.IsArchived)
			assert.Nil(t, got.ArchivedAt)
			assert.Equal(t, 0, got.FollowerCount)
			assert.Equal(t, 0, got.CommentCount)
			assert.False(t, got.IsFollowing)

			authorID, err := repos.Journal.GetAuthorID(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, user.ID, authorID)

			title, err := repos.Journal.GetTitle(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, "My Title", title)
		})
	}
}

func TestJournalDAO_UnknownIDs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	missing := uuid.New()

	// when
	journal, journalErr := repos.Journal.GetByID(ctx, spec.JournalLookup{ID: missing})
	_, authorErr := repos.Journal.GetAuthorID(ctx, missing)
	_, titleErr := repos.Journal.GetTitle(ctx, missing)
	_, commentJournalErr := repos.Journal.GetCommentEntityID(ctx, missing)
	_, commentAuthorErr := repos.Journal.GetCommentAuthorID(ctx, missing)
	_, entryAuthorErr := repos.Journal.GetEntryAuthorID(ctx, missing)
	_, entryJournalErr := repos.Journal.GetEntryJournalID(ctx, missing)

	// then
	require.NoError(t, journalErr)
	assert.Nil(t, journal)
	assert.ErrorIs(t, authorErr, dao.ErrNotFound)
	assert.ErrorIs(t, titleErr, dao.ErrNotFound)
	assert.Error(t, commentJournalErr)
	assert.ErrorIs(t, commentAuthorErr, dao.ErrNotFound)
	assert.ErrorIs(t, entryAuthorErr, dao.ErrNotFound, "a missing entry must read as not found, not as a database failure")
	assert.ErrorIs(t, entryJournalErr, dao.ErrNotFound)
}

func TestJournalDAO_GetByID_CountsCommentsAndSurfacesTheLatestEntry(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, user.ID, "Title", "", "general")
	createJournalComment(t, repos, journalID, user.ID, nil, "one")
	createJournalComment(t, repos, journalID, user.ID, nil, "two")
	journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 1, Body: "first body", WordCount: 2})
	journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 2, Title: new("Latest"), Body: "newest body", WordCount: 2})

	// when
	got, err := repos.Journal.GetByID(context.Background(), spec.JournalLookup{ID: journalID})

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 2, got.CommentCount)
	require.NotNil(t, got.LatestEntryNumber)
	assert.Equal(t, 2, *got.LatestEntryNumber)
	require.NotNil(t, got.LatestEntryTitle)
	assert.Equal(t, "Latest", *got.LatestEntryTitle)
	assert.Equal(t, "newest body", got.LatestEntryExcerpt)
	assert.Equal(t, 2, got.EntryCount)
}

func TestJournalDAO_List(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	fan := daotest.CreateUser(t, repos)
	base := time.Now().UTC().Add(-time.Hour)
	stale := createJournal(t, repos, author.ID, "Stale", "", "general")
	magic := createJournal(t, repos, author.ID, "Witches and Magic", "", "umineko")
	unrelated := createJournal(t, repos, author.ID, "Unrelated", "", "higurashi")
	hidden := createJournal(t, repos, blocked.ID, "Hidden", "", "general")

	journalBackdate(t, repos, stale, base.Add(-48*time.Hour))
	for i, id := range []uuid.UUID{magic, unrelated, hidden} {
		journalBackdate(t, repos, id, base.Add(time.Duration(i+1)*time.Minute))
	}

	archived, err := repos.Journal.ArchiveStale(ctx, base)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{stale}, archived)

	for _, follow := range []spec.JournalFollow{
		{UserID: fan.ID, JournalID: unrelated},
		{UserID: blocked.ID, JournalID: unrelated},
		{UserID: fan.ID, JournalID: magic},
	} {
		require.NoError(t, repos.Journal.Follow(ctx, follow))
	}

	cases := []struct {
		name      string
		query     spec.JournalQuery
		want      []uuid.UUID
		wantTotal int
	}{
		{name: "sorts newest first by default and leaves archived journals out", query: spec.JournalQuery{Sort: "new", Limit: 20}, want: []uuid.UUID{hidden, unrelated, magic}, wantTotal: 3},
		{name: "sorts oldest first", query: spec.JournalQuery{Sort: "old", Limit: 20}, want: []uuid.UUID{magic, unrelated, hidden}, wantTotal: 3},
		{name: "sorts by follower count", query: spec.JournalQuery{Sort: "most_followed", Limit: 20}, want: []uuid.UUID{unrelated, magic, hidden}, wantTotal: 3},
		{name: "includes archived journals on request", query: spec.JournalQuery{Sort: "new", Limit: 20, IncludeArchived: true}, want: []uuid.UUID{hidden, unrelated, magic, stale}, wantTotal: 4},
		{name: "filters by work", query: spec.JournalQuery{Sort: "new", Limit: 20, Work: "umineko"}, want: []uuid.UUID{magic}, wantTotal: 1},
		{name: "filters by author", query: spec.JournalQuery{Sort: "new", Limit: 20, AuthorID: author.ID}, want: []uuid.UUID{unrelated, magic}, wantTotal: 2},
		{name: "searches titles case-insensitively", query: spec.JournalQuery{Sort: "new", Limit: 20, Search: "magic"}, want: []uuid.UUID{magic}, wantTotal: 1},
		{name: "excludes blocked users", query: spec.JournalQuery{Sort: "new", Limit: 20, ExcludeUserIDs: []uuid.UUID{blocked.ID}}, want: []uuid.UUID{unrelated, magic}, wantTotal: 2},
		{name: "pages while the total still counts every match", query: spec.JournalQuery{Sort: "new", Limit: 1, Offset: 1}, want: []uuid.UUID{unrelated}, wantTotal: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			journals, total, err := repos.Journal.List(ctx, tc.query)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)
			assert.Equal(t, tc.want, journalResponseIDs(journals))
			for _, journal := range journals {
				assert.Equal(t, journal.ID == stale, journal.IsArchived)
			}
		})
	}
}

func TestJournalDAO_List_LatestEntryExcerpt(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "a long body is clamped to 300 characters plus an ellipsis", body: strings.Repeat("a", 400), want: strings.Repeat("a", 300) + "..."},
		{name: "a multi-byte body is clipped on rune boundaries", body: strings.Repeat("雛見沢", 200), want: strings.Repeat("雛見沢", 100) + "..."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			user := daotest.CreateUser(t, repos)
			journalID := createJournal(t, repos, user.ID, "T", "", "general")
			journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 1, Body: tc.body, WordCount: 1})

			// when
			journals, _, err := repos.Journal.List(context.Background(), spec.JournalQuery{Sort: "new", Limit: 20})

			// then
			require.NoError(t, err)
			require.Len(t, journals, 1)
			assert.Equal(t, tc.want, journals[0].LatestEntryExcerpt)
		})
	}
}

func TestJournalDAO_Update(t *testing.T) {
	cases := []struct {
		name         string
		byOwner      bool
		asAdmin      bool
		wantErr      bool
		wantTitle    string
		wantWork     string
		wantArchived bool
	}{
		{name: "the owner retitles it and moves it to another work, which also unarchives it", byOwner: true, wantTitle: "New", wantWork: "higurashi", wantArchived: false},
		{name: "a stranger is refused and nothing changes", wantErr: true, wantTitle: "Old", wantWork: "general", wantArchived: true},
		{name: "an admin retitles someone else's journal without reviving it", asAdmin: true, wantTitle: "New", wantWork: "higurashi", wantArchived: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			stranger := daotest.CreateUser(t, repos)
			id := createJournal(t, repos, owner.ID, "Old", "", "general")

			_, err := repos.Journal.ArchiveStale(ctx, time.Now().Add(time.Hour))
			require.NoError(t, err)

			actor := stranger.ID
			if tc.byOwner {
				actor = owner.ID
			}

			// when
			err = repos.Journal.Update(ctx, spec.JournalUpdate{ID: id, UserID: actor, Title: "New", Work: "higurashi", AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			got, err := repos.Journal.GetByID(ctx, spec.JournalLookup{ID: id})
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.wantTitle, got.Title)
			assert.Equal(t, tc.wantWork, got.Work)
			assert.Equal(t, !tc.wantErr, got.UpdatedAt != nil)

			isArchived, err := repos.Journal.IsArchived(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, tc.wantArchived, isArchived)
		})
	}
}

func TestJournalDAO_DeleteWithMedia(t *testing.T) {
	seeded := []string{
		"/uploads/journal/entry.png",
		"/uploads/journal/entry_thumb.png",
		"/uploads/journal/entry_no_thumb.gif",
		"/uploads/journal/kept_entry.png",
		"/uploads/journal/comment.png",
		"/uploads/journal/comment_thumb.png",
		"/uploads/journal/kept_comment.png",
		"/uploads/journal/entry_comment.png",
		"/uploads/journal/entry_comment_thumb.png",
		"/uploads/journal/reply.png",
	}

	cases := []struct {
		name      string
		bare      bool
		byOwner   bool
		asAdmin   bool
		wantErr   bool
		wantPaths []string
	}{
		{name: "the owner deletes it and gets back every entry and comment media path", byOwner: true, wantPaths: seeded},
		{name: "the owner deletes a journal with nothing attached and gets back no paths", bare: true, byOwner: true, wantPaths: nil},
		{name: "a stranger is refused and the journal survives", wantErr: true},
		{name: "an admin deletes someone else's journal and gets back every media path", asAdmin: true, wantPaths: seeded},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			f := journalSeedMedia(t, repos)

			target := f.journalID
			if tc.bare {
				target = createJournal(t, repos, f.ownerID, "Bare", "", "general")
			}

			actor := f.strangerID
			if tc.byOwner {
				actor = f.ownerID
			}

			// when
			paths, err := repos.Journal.DeleteWithMedia(ctx, spec.JournalDeletion{ID: target, UserID: actor, AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, paths)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tc.wantPaths, paths)
				assert.NotContains(t, paths, "")
			}

			got, err := repos.Journal.GetByID(ctx, spec.JournalLookup{ID: target})
			require.NoError(t, err)
			assert.Equal(t, tc.wantErr, got != nil)
		})
	}
}

func TestJournalDAO_CountUserJournalsToday(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	createJournal(t, repos, user.ID, "A", "", "general")
	createJournal(t, repos, user.ID, "B", "", "general")
	createJournal(t, repos, other.ID, "C", "", "general")

	// when
	count, err := repos.Journal.CountUserJournalsToday(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestJournalDAO_UpdateLastAuthorActivity_Unarchives(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "", "general")

	_, err := repos.Journal.ArchiveStale(ctx, time.Now().Add(time.Hour))
	require.NoError(t, err)

	// when
	err = repos.Journal.UpdateLastAuthorActivity(ctx, id)

	// then
	require.NoError(t, err)

	archived, err := repos.Journal.IsArchived(ctx, id)
	require.NoError(t, err)
	assert.False(t, archived)
}

func TestJournalDAO_ArchiveStale(t *testing.T) {
	cases := []struct {
		name         string
		pauses       []bool
		cutoff       time.Duration
		wantArchived bool
	}{
		{name: "archives a journal idle since before the cutoff and returns its id", cutoff: time.Hour, wantArchived: true},
		{name: "leaves a journal active since the cutoff alone", cutoff: -time.Hour, wantArchived: false},
		{name: "skips a journal old enough to archive that its author has paused, when a cutoff that would otherwise catch it runs, because pausing exists precisely to survive the sweep", pauses: []bool{true}, cutoff: time.Hour, wantArchived: false},
		{name: "catches a paused journal once its author resumes it, because the pause must not linger and shield it forever", pauses: []bool{true, false}, cutoff: time.Hour, wantArchived: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos)
			id := createJournal(t, repos, user.ID, "T", "", "general")

			for _, paused := range tc.pauses {
				require.NoError(t, repos.Journal.SetPaused(ctx, spec.JournalPause{ID: id, UserID: user.ID, Paused: paused}))
			}

			// when
			ids, err := repos.Journal.ArchiveStale(ctx, time.Now().Add(tc.cutoff))

			// then
			require.NoError(t, err)
			if tc.wantArchived {
				assert.Equal(t, []uuid.UUID{id}, ids)
			} else {
				assert.Empty(t, ids)
			}

			archived, err := repos.Journal.IsArchived(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, tc.wantArchived, archived)
		})
	}
}

func TestJournalDAO_SetPaused_RefusesAJournalTheUserDoesNotOwn(t *testing.T) {
	// given a journal belonging to somebody else
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "Theirs", "", "general")

	// when
	err := repos.Journal.SetPaused(context.Background(), spec.JournalPause{ID: journalID, UserID: stranger.ID, Paused: true})

	// then ownership is enforced in the statement itself, not only in the service above it
	require.Error(t, err)
}

func TestJournalDAO_FollowAndUnfollow(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	follower := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "", "general")
	follow := spec.JournalFollow{UserID: follower.ID, JournalID: id}
	otherFollow := spec.JournalFollow{UserID: other.ID, JournalID: id}

	// when
	require.NoError(t, repos.Journal.Follow(ctx, follow))
	require.NoError(t, repos.Journal.Follow(ctx, follow))
	require.NoError(t, repos.Journal.Follow(ctx, otherFollow))

	isFollowerAfter, err := repos.Journal.IsFollower(ctx, follow)
	require.NoError(t, err)
	countAfter, err := repos.Journal.GetFollowerCount(ctx, id)
	require.NoError(t, err)
	idsAfter, err := repos.Journal.GetFollowerIDs(ctx, id)
	require.NoError(t, err)
	viewed, err := repos.Journal.GetByID(ctx, spec.JournalLookup{ID: id, ViewerID: follower.ID})
	require.NoError(t, err)

	require.NoError(t, repos.Journal.Unfollow(ctx, follow))
	require.NoError(t, repos.Journal.Unfollow(ctx, otherFollow))

	isFollowerFinal, err := repos.Journal.IsFollower(ctx, follow)
	require.NoError(t, err)
	countFinal, err := repos.Journal.GetFollowerCount(ctx, id)
	require.NoError(t, err)
	idsFinal, err := repos.Journal.GetFollowerIDs(ctx, id)
	require.NoError(t, err)

	// then
	assert.True(t, isFollowerAfter)
	assert.Equal(t, 2, countAfter)
	assert.ElementsMatch(t, []uuid.UUID{follower.ID, other.ID}, idsAfter)
	require.NotNil(t, viewed)
	assert.True(t, viewed.IsFollowing)
	assert.Equal(t, 2, viewed.FollowerCount)
	assert.False(t, isFollowerFinal)
	assert.Equal(t, 0, countFinal)
	assert.Empty(t, idsFinal)
}

func TestJournalDAO_ListFollowedByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	follower := daotest.CreateUser(t, repos)
	a := createJournal(t, repos, author.ID, "A", "", "general")
	b := createJournal(t, repos, author.ID, "B", "", "general")
	createJournal(t, repos, author.ID, "C", "", "general")

	for _, id := range []uuid.UUID{a, b} {
		require.NoError(t, repos.Journal.Follow(ctx, spec.JournalFollow{UserID: follower.ID, JournalID: id}))
	}

	// when
	journals, total, err := repos.Journal.ListFollowedByUser(ctx, spec.JournalFollowedQuery{FollowerID: follower.ID, ViewerID: follower.ID, Limit: 10})

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.ElementsMatch(t, []uuid.UUID{a, b}, journalResponseIDs(journals))
	for _, journal := range journals {
		assert.True(t, journal.IsFollowing)
	}
}

func TestJournalDAO_CreateComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "", "general")

	// when
	rootID := createJournalComment(t, repos, journalID, commenter.ID, nil, "hello")
	replyID := createJournalComment(t, repos, journalID, commenter.ID, &rootID, "child")

	// then
	comments, total := journalRootComments(t, repos, journalID, commenter.ID)
	assert.Equal(t, 2, total)
	require.ElementsMatch(t, []uuid.UUID{rootID, replyID}, journalCommentIDs(comments))
	for _, comment := range comments {
		if comment.ID == rootID {
			assert.Equal(t, "hello", comment.Body)
			assert.Nil(t, comment.ParentID)
		} else {
			require.NotNil(t, comment.ParentID)
			assert.Equal(t, rootID, *comment.ParentID)
		}
	}

	gotJournalID, err := repos.Journal.GetCommentEntityID(ctx, rootID)
	require.NoError(t, err)
	assert.Equal(t, journalID, gotJournalID)

	gotAuthorID, err := repos.Journal.GetCommentAuthorID(ctx, rootID)
	require.NoError(t, err)
	assert.Equal(t, commenter.ID, gotAuthorID)
}

func TestJournalDAO_UpdateComment(t *testing.T) {
	cases := []struct {
		name     string
		byOwner  bool
		asAdmin  bool
		wantErr  bool
		wantBody string
	}{
		{name: "the author edits their own comment", byOwner: true, wantBody: "new"},
		{name: "a stranger is refused and the comment is untouched", wantErr: true, wantBody: "old"},
		{name: "an admin edits someone else's comment", asAdmin: true, wantBody: "new"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			author := daotest.CreateUser(t, repos)
			stranger := daotest.CreateUser(t, repos)
			journalID := createJournal(t, repos, author.ID, "T", "", "general")
			commentID := createJournalComment(t, repos, journalID, author.ID, nil, "old")

			actor := stranger.ID
			if tc.byOwner {
				actor = author.ID
			}

			// when
			err := repos.Journal.UpdateComment(ctx, spec.CommentUpdate{CommentID: commentID, UserID: actor, Body: "new", AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			comments, _ := journalRootComments(t, repos, journalID, author.ID)
			require.Len(t, comments, 1)
			assert.Equal(t, tc.wantBody, comments[0].Body)
			assert.Equal(t, !tc.wantErr, comments[0].UpdatedAt != nil)
		})
	}
}

func TestJournalDAO_DeleteCommentWithAudit(t *testing.T) {
	cases := []struct {
		name        string
		byOwner     bool
		asAdmin     bool
		wantErr     bool
		auditAction audit.Action
		wantAudited int
	}{
		{name: "the author deletes their own comment, gets back only its media paths, and it is audited", byOwner: true, auditAction: audit.ActionJournalCommentDelete, wantAudited: 1},
		{name: "a stranger is refused, the comment survives, and nothing is audited", wantErr: true, auditAction: audit.ActionJournalCommentDelete, wantAudited: 0},
		{name: "an admin deletes someone else's comment and it is audited as an admin action", asAdmin: true, auditAction: audit.ActionJournalCommentDeleteAdmin, wantAudited: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			f := journalSeedMedia(t, repos)

			actor := f.strangerID
			if tc.byOwner {
				actor = f.ownerID
			}

			// when
			paths, err := repos.Journal.DeleteCommentWithAudit(ctx, spec.CommentDeletion{CommentID: f.commentID, UserID: actor, AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.ErrorIs(t, err, dao.ErrNotFound)
				assert.Nil(t, paths)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"/uploads/journal/comment.png", "/uploads/journal/comment_thumb.png"}, paths)
				assert.NotContains(t, paths, "")
			}

			comments, _ := journalRootComments(t, repos, f.journalID, f.ownerID)
			assert.Equal(t, tc.wantErr, slices.Contains(journalCommentIDs(comments), f.commentID))

			keptMedia, err := repos.Journal.GetCommentMediaBatch(ctx, []uuid.UUID{f.keptCommentID})
			require.NoError(t, err)
			require.Len(t, keptMedia[f.keptCommentID], 1)
			assert.Equal(t, "/uploads/journal/kept_comment.png", keptMedia[f.keptCommentID][0].MediaURL)

			entries, total, err := repos.AuditLog.List(ctx, spec.AuditLogListing{Action: tc.auditAction, Page: bounds.NewPage(10, 0)})
			require.NoError(t, err)
			assert.Equal(t, tc.wantAudited, total)
			require.Len(t, entries, tc.wantAudited)
			for _, entry := range entries {
				assert.Equal(t, actor, entry.ActorID)
				assert.Equal(t, audit.TargetJournalComment, entry.TargetType)
				assert.Equal(t, f.commentID.String(), entry.TargetID)
			}
		})
	}
}

func TestJournalDAO_GetComments_PaginationOrderingAndExclusion(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	commenterA := daotest.CreateUser(t, repos, daotest.WithDisplayName("A"))
	commenterB := daotest.CreateUser(t, repos, daotest.WithDisplayName("B"))
	blocked := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "", "general")
	first := createJournalComment(t, repos, journalID, commenterA.ID, nil, "first")
	second := createJournalComment(t, repos, journalID, commenterB.ID, nil, "second")
	third := createJournalComment(t, repos, journalID, blocked.ID, nil, "blocked-comment")

	base := time.Now().UTC().Add(-time.Hour)
	for i, id := range []uuid.UUID{first, second, third} {
		_, err := repos.DB().ExecContext(ctx, `UPDATE journal_comments SET created_at = $1 WHERE id = $2`, base.Add(time.Duration(i)*time.Minute), id)
		require.NoError(t, err)
	}

	// when
	all, total, err := repos.Journal.GetComments(ctx, spec.CommentQuery[uuid.UUID]{TargetID: journalID, ViewerID: commenterA.ID, Limit: 10})
	require.NoError(t, err)
	excluded, exclTotal, err := repos.Journal.GetComments(ctx, spec.CommentQuery[uuid.UUID]{TargetID: journalID, ViewerID: commenterA.ID, Limit: 10, ExcludeUserIDs: []uuid.UUID{blocked.ID}})
	require.NoError(t, err)
	page, _, err := repos.Journal.GetComments(ctx, spec.CommentQuery[uuid.UUID]{TargetID: journalID, ViewerID: commenterA.ID, Limit: 1, Offset: 1})
	require.NoError(t, err)

	// then
	assert.Equal(t, 3, total)
	require.Equal(t, []uuid.UUID{first, second, third}, journalCommentIDs(all))
	assert.Equal(t, "A", all[0].AuthorDisplayName)
	assert.Equal(t, 2, exclTotal)
	assert.Equal(t, []uuid.UUID{first, second}, journalCommentIDs(excluded))
	assert.Equal(t, []uuid.UUID{second}, journalCommentIDs(page))
}

func TestJournalDAO_LikeAndUnlikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "", "general")
	like := spec.CommentLike{UserID: liker.ID, CommentID: createJournalComment(t, repos, journalID, author.ID, nil, "x")}

	// when
	require.NoError(t, repos.Journal.LikeComment(ctx, like))
	require.NoError(t, repos.Journal.LikeComment(ctx, like))
	liked, _ := journalRootComments(t, repos, journalID, liker.ID)

	require.NoError(t, repos.Journal.UnlikeComment(ctx, like))
	unliked, _ := journalRootComments(t, repos, journalID, liker.ID)

	// then
	require.Len(t, liked, 1)
	assert.Equal(t, 1, liked[0].LikeCount)
	assert.True(t, liked[0].UserLiked)
	require.Len(t, unliked, 1)
	assert.Equal(t, 0, unliked[0].LikeCount)
	assert.False(t, unliked[0].UserLiked)
}

func TestJournalDAO_CommentMedia(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "", "general")
	commentA := createJournalComment(t, repos, journalID, author.ID, nil, "a")
	commentB := createJournalComment(t, repos, journalID, author.ID, nil, "b")
	commentC := createJournalComment(t, repos, journalID, author.ID, nil, "c")
	commentD := createJournalComment(t, repos, journalID, author.ID, nil, "d")

	// when
	idA0, err := repos.Journal.AddCommentMedia(ctx, spec.NewMedia{TargetID: commentA, MediaURL: "url-a-0", MediaType: "image", ThumbnailURL: "thumb-a-0"})
	require.NoError(t, err)
	idA1, err := repos.Journal.AddCommentMedia(ctx, spec.NewMedia{TargetID: commentA, MediaURL: "url-a-1", MediaType: "image", ThumbnailURL: "thumb-a-1"})
	require.NoError(t, err)
	idB, err := repos.Journal.AddCommentMedia(ctx, spec.NewMedia{TargetID: commentB, MediaURL: "url-b", MediaType: "video", ThumbnailURL: "thumb-b"})
	require.NoError(t, err)
	idD, err := repos.Journal.AddCommentMedia(ctx, spec.NewMedia{TargetID: commentD, MediaURL: "old-url", MediaType: "image", ThumbnailURL: "old-thumb"})
	require.NoError(t, err)

	require.NoError(t, repos.Journal.UpdateCommentMediaURL(ctx, spec.MediaURLUpdate{ID: idD, URL: "new-url"}))
	require.NoError(t, repos.Journal.UpdateCommentMediaThumbnail(ctx, spec.MediaURLUpdate{ID: idD, URL: "new-thumb"}))

	batch, err := repos.Journal.GetCommentMediaBatch(ctx, []uuid.UUID{commentA, commentB, commentC, commentD})
	require.NoError(t, err)
	empty, err := repos.Journal.GetCommentMediaBatch(ctx, nil)
	require.NoError(t, err)

	// then
	assert.Greater(t, idA0, int64(0))
	assert.Greater(t, idA1, int64(0))
	assert.Greater(t, idB, int64(0))
	require.Len(t, batch[commentA], 2)
	assert.Equal(t, "url-a-0", batch[commentA][0].MediaURL)
	assert.Equal(t, "url-a-1", batch[commentA][1].MediaURL)
	require.Len(t, batch[commentB], 1)
	assert.Equal(t, "url-b", batch[commentB][0].MediaURL)
	assert.Equal(t, "video", batch[commentB][0].MediaType)
	assert.NotContains(t, batch, commentC)
	require.Len(t, batch[commentD], 1)
	assert.Equal(t, "new-url", batch[commentD][0].MediaURL)
	assert.Equal(t, "new-thumb", batch[commentD][0].ThumbnailURL)
	assert.Nil(t, empty)
}

func TestJournalDAO_Entries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, user.ID, "Title", "", "general")

	next, err := repos.Journal.GetNextEntryNumber(ctx, journalID)
	require.NoError(t, err)
	assert.Equal(t, 1, next)

	firstID := journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 1, Title: new("Day 1"), Body: "the body", WordCount: 2})
	for _, number := range []int{2, 3} {
		journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: number, Body: "b", WordCount: 1})
	}

	// when
	first, err := repos.Journal.GetEntry(ctx, spec.JournalEntryLookup{JournalID: journalID, EntryNumber: 1})
	require.NoError(t, err)
	middle, err := repos.Journal.GetEntry(ctx, spec.JournalEntryLookup{JournalID: journalID, EntryNumber: 2})
	require.NoError(t, err)
	last, err := repos.Journal.GetEntry(ctx, spec.JournalEntryLookup{JournalID: journalID, EntryNumber: 3})
	require.NoError(t, err)
	entries, err := repos.Journal.ListEntries(ctx, journalID)
	require.NoError(t, err)
	next, err = repos.Journal.GetNextEntryNumber(ctx, journalID)
	require.NoError(t, err)

	// then
	require.NotNil(t, first)
	assert.Equal(t, firstID, first.ID)
	assert.Equal(t, 1, first.EntryNumber)
	require.NotNil(t, first.Title)
	assert.Equal(t, "Day 1", *first.Title)
	assert.Equal(t, "the body", first.Body)
	assert.False(t, first.HasPrev)
	assert.True(t, first.HasNext)
	require.NotNil(t, middle)
	assert.True(t, middle.HasPrev)
	assert.True(t, middle.HasNext)
	require.NotNil(t, last)
	assert.True(t, last.HasPrev)
	assert.False(t, last.HasNext)

	var listed []int
	for _, entry := range entries {
		listed = append(listed, entry.EntryNumber)
	}
	assert.Equal(t, []int{3, 2, 1}, listed)
	assert.Equal(t, 4, next)
}

func TestJournalDAO_EntryComments_ScopedSeparately(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, user.ID, "Title", "", "general")
	entryID := journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 1, Body: "b", WordCount: 1})

	// when
	topLevelComment := createJournalComment(t, repos, journalID, user.ID, nil, "on journal")
	entryComment, err := repos.Comments.Journal.CreateComment(ctx, spec.NewJournalComment{JournalID: journalID, EntryID: &entryID, UserID: user.ID, Body: "on entry"})
	require.NoError(t, err)

	jrComments, _ := journalRootComments(t, repos, journalID, uuid.Nil)
	enComments, _, err := repos.Journal.GetEntryComments(ctx, spec.CommentQuery[uuid.UUID]{TargetID: entryID, Limit: 100})
	require.NoError(t, err)

	// then: title-page query only returns the journal-level comment
	require.Len(t, jrComments, 1)
	assert.Equal(t, topLevelComment, jrComments[0].ID)
	// entry query only returns the entry-scoped comment
	require.Len(t, enComments, 1)
	assert.Equal(t, entryComment.ID, enComments[0].ID)
	require.NotNil(t, enComments[0].EntryID)
	assert.Equal(t, entryID, *enComments[0].EntryID)
}

func TestJournalDAO_DeleteEntry_ReturnsOnlyThatEntrysMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	f := journalSeedMedia(t, repos)

	// when
	paths, err := repos.Journal.DeleteEntryWithMedia(ctx, f.entryID)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/journal/entry.png",
		"/uploads/journal/entry_thumb.png",
		"/uploads/journal/entry_no_thumb.gif",
		"/uploads/journal/entry_comment.png",
		"/uploads/journal/entry_comment_thumb.png",
		"/uploads/journal/reply.png",
	}, paths)
	assert.NotContains(t, paths, "")

	keptEntryMedia, err := repos.Journal.GetMediaBatch(ctx, []uuid.UUID{f.keptEntryID})
	require.NoError(t, err)
	require.Len(t, keptEntryMedia[f.keptEntryID], 1)
	assert.Equal(t, "/uploads/journal/kept_entry.png", keptEntryMedia[f.keptEntryID][0].MediaURL)

	keptCommentMedia, err := repos.Journal.GetCommentMediaBatch(ctx, []uuid.UUID{f.keptCommentID})
	require.NoError(t, err)
	require.Len(t, keptCommentMedia[f.keptCommentID], 1)
	assert.Equal(t, "/uploads/journal/kept_comment.png", keptCommentMedia[f.keptCommentID][0].MediaURL)
}
