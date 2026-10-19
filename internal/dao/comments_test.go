package dao_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	commentKindCase struct {
		kind     spec.CommentKind
		table    string
		mediaDir string
		owned    bool
		seed     func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget
	}
)

func commentKinds() []commentKindCase {
	return []commentKindCase{
		{kind: spec.CommentKindPost, table: "post_comments", mediaDir: "posts", owned: true, seed: func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget {
			return spec.CommentTarget{Kind: spec.CommentKindPost, ID: createPost(t, repos, ownerID, "general", "a theory")}
		}},
		{kind: spec.CommentKindArt, table: "art_comments", mediaDir: "art", owned: true, seed: func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget {
			return spec.CommentTarget{Kind: spec.CommentKindArt, ID: createArt(t, repos, ownerID, "general", "drawing", "Golden Land", nil, false)}
		}},
		{kind: spec.CommentKindAnnouncement, table: "announcement_comments", mediaDir: "announcements", owned: true, seed: func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget {
			return spec.CommentTarget{Kind: spec.CommentKindAnnouncement, ID: createAnnouncement(t, repos, ownerID, "Maintenance", "tonight")}
		}},
		{kind: spec.CommentKindMystery, table: "mystery_comments", mediaDir: "mysteries", owned: true, seed: func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget {
			return spec.CommentTarget{Kind: spec.CommentKindMystery, ID: createMystery(t, repos, ownerID, "The Locked Room", "medium", false)}
		}},
		{kind: spec.CommentKindShip, table: "ship_comments", mediaDir: "ships", owned: true, seed: func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget {
			return spec.CommentTarget{Kind: spec.CommentKindShip, ID: createShip(t, repos, ownerID, "BattlerBeato", makeChars())}
		}},
		{kind: spec.CommentKindOC, table: "oc_comments", mediaDir: "ocs", owned: true, seed: func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget {
			return spec.CommentTarget{Kind: spec.CommentKindOC, ID: createOC(t, repos, ownerID, "Linda", "umineko", "")}
		}},
		{kind: spec.CommentKindFanfic, table: "fanfic_comments", mediaDir: "fanfics", owned: true, seed: func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget {
			return spec.CommentTarget{Kind: spec.CommentKindFanfic, ID: createFanfic(t, repos, ownerID, "A Tea Party")}
		}},
		{kind: spec.CommentKindJournal, table: "journal_comments", mediaDir: "journals", owned: true, seed: func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget {
			return spec.CommentTarget{Kind: spec.CommentKindJournal, ID: createJournal(t, repos, ownerID, "Reading Log", "", "umineko")}
		}},
		{kind: spec.CommentKindJournalEntry, table: "journal_comments", mediaDir: "journals", owned: true, seed: func(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID) spec.CommentTarget {
			journalID := createJournal(t, repos, ownerID, "Reading Log", "", "umineko")
			journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 1, Body: "episode one", WordCount: 2})

			return spec.CommentTarget{Kind: spec.CommentKindJournalEntry, ID: journalID, EntryNumber: 1}
		}},
		{kind: spec.CommentKindSecret, table: "secret_comments", mediaDir: "secrets", seed: func(*testing.T, *repository.Repositories, uuid.UUID) spec.CommentTarget {
			return spec.CommentTarget{Kind: spec.CommentKindSecret, Key: "witchHunter"}
		}},
	}
}

func backdateComments(t *testing.T, repos *repository.Repositories, table string, ids ...uuid.UUID) {
	t.Helper()

	base := time.Now().UTC().Add(-time.Hour)
	for i, id := range ids {
		_, err := repos.DB().ExecContext(context.Background(), "UPDATE "+table+" SET created_at = $1 WHERE id = $2", base.Add(time.Duration(i)*time.Minute), id)
		require.NoError(t, err)
	}
}

func createComment(t *testing.T, repos *repository.Repositories, target spec.CommentTarget, userID uuid.UUID, parentID *uuid.UUID, body string) uuid.UUID {
	t.Helper()

	created, err := repos.Comment.CreateComment(context.Background(), spec.NewComment{Target: target, ParentID: parentID, UserID: userID, Body: body})
	require.NoError(t, err)

	return created.ID
}

func addCommentMedia(t *testing.T, repos *repository.Repositories, kind spec.CommentKind, commentID uuid.UUID, mediaURL, thumbnailURL string) int64 {
	t.Helper()

	id, err := repos.Comment.AddCommentMedia(context.Background(), spec.NewCommentMedia{
		Kind:     kind,
		NewMedia: spec.NewMedia{TargetID: commentID, MediaURL: mediaURL, MediaType: "image", ThumbnailURL: thumbnailURL},
	})
	require.NoError(t, err)

	return id
}

func listComments(t *testing.T, repos *repository.Repositories, target spec.CommentTarget, viewerID uuid.UUID) ([]model.CommentRow, int) {
	t.Helper()

	rows, total, err := repos.Comment.GetComments(context.Background(), spec.CommentQuery{Target: target, ViewerID: viewerID, Limit: 10})
	require.NoError(t, err)

	return rows, total
}

func TestCommentDAO_CreatesAThreadAndKnowsWhereEachCommentLives(t *testing.T) {
	for _, kc := range commentKinds() {
		t.Run(string(kc.kind), func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			owner := daotest.CreateUser(t, repos)
			commenter := daotest.CreateUser(t, repos, daotest.WithDisplayName("Ange"))
			target := kc.seed(t, repos, owner.ID)

			// when
			parentID := createComment(t, repos, target, commenter.ID, nil, "parent")
			childID := createComment(t, repos, target, owner.ID, &parentID, "child")
			backdateComments(t, repos, kc.table, parentID, childID)

			// then
			rows, total := listComments(t, repos, target, commenter.ID)
			assert.Equal(t, 2, total)
			require.Len(t, rows, 2)
			assert.Equal(t, parentID, rows[0].ID)
			assert.Equal(t, "Ange", rows[0].AuthorDisplayName)
			assert.Equal(t, childID, rows[1].ID)
			require.NotNil(t, rows[1].ParentID)
			assert.Equal(t, parentID, *rows[1].ParentID)

			byID, err := repos.Comment.GetCommentByID(context.Background(), spec.CommentRef{Kind: kc.kind, CommentID: childID})
			require.NoError(t, err)
			require.NotNil(t, byID)
			assert.Equal(t, "child", byID.Body)

			located, err := repos.Comment.GetCommentTarget(context.Background(), spec.CommentRef{Kind: kc.kind, CommentID: childID})
			require.NoError(t, err)
			assert.Equal(t, target, located)

			authorID, err := repos.Comment.GetCommentAuthorID(context.Background(), spec.CommentRef{Kind: kc.kind, CommentID: parentID})
			require.NoError(t, err)
			assert.Equal(t, commenter.ID, authorID)

			ownerID, err := repos.Comment.GetTargetOwnerID(context.Background(), target)
			require.NoError(t, err)
			if kc.owned {
				assert.Equal(t, owner.ID, ownerID)
			} else {
				assert.Equal(t, uuid.Nil, ownerID, "a target nobody owns has no owner to notify")
			}

			mediaDir, err := repos.Comment.GetCommentMediaDir(kc.kind)
			require.NoError(t, err)
			assert.Equal(t, kc.mediaDir, mediaDir)
		})
	}
}

func TestCommentDAO_OnlyTheAuthorOrAnAdminCanChangeAComment(t *testing.T) {
	for _, kc := range commentKinds() {
		t.Run(string(kc.kind), func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			author := daotest.CreateUser(t, repos)
			stranger := daotest.CreateUser(t, repos)
			target := kc.seed(t, repos, author.ID)
			ref := spec.CommentRef{Kind: kc.kind, CommentID: createComment(t, repos, target, author.ID, nil, "original")}
			doomed := spec.CommentRef{Kind: kc.kind, CommentID: createComment(t, repos, target, author.ID, nil, "doomed")}

			// when
			strangerEdit := repos.Comment.UpdateComment(context.Background(), spec.CommentUpdate{CommentRef: ref, UserID: stranger.ID, Body: "vandalised"})
			ownEdit := repos.Comment.UpdateComment(context.Background(), spec.CommentUpdate{CommentRef: ref, UserID: author.ID, Body: "edited"})
			adminEdit := repos.Comment.UpdateComment(context.Background(), spec.CommentUpdate{CommentRef: ref, UserID: stranger.ID, Body: "moderated", AsAdmin: true})
			strangerDelete := repos.Comment.DeleteComment(context.Background(), spec.CommentDeletion{CommentRef: doomed, UserID: stranger.ID})
			adminDelete := repos.Comment.DeleteComment(context.Background(), spec.CommentDeletion{CommentRef: doomed, UserID: stranger.ID, AsAdmin: true})

			// then a stranger's attempt reads as not found, never as a database failure
			require.ErrorIs(t, strangerEdit, dao.ErrNotFound)
			require.NoError(t, ownEdit)
			require.NoError(t, adminEdit)
			require.ErrorIs(t, strangerDelete, dao.ErrNotFound)
			require.NoError(t, adminDelete)

			rows, total := listComments(t, repos, target, author.ID)
			assert.Equal(t, 1, total)
			require.Len(t, rows, 1)
			assert.Equal(t, "moderated", rows[0].Body)
			assert.NotNil(t, rows[0].UpdatedAt)
		})
	}
}

func TestCommentDAO_TheAuthorCanDeleteTheirOwnComment(t *testing.T) {
	for _, kc := range commentKinds() {
		t.Run(string(kc.kind), func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			author := daotest.CreateUser(t, repos)
			target := kc.seed(t, repos, author.ID)
			ref := spec.CommentRef{Kind: kc.kind, CommentID: createComment(t, repos, target, author.ID, nil, "regrettable")}

			// when
			err := repos.Comment.DeleteComment(context.Background(), spec.CommentDeletion{CommentRef: ref, UserID: author.ID})

			// then
			require.NoError(t, err)
			_, total := listComments(t, repos, target, author.ID)
			assert.Equal(t, 0, total)
		})
	}
}

func TestCommentDAO_ListingPagesInOrderAndHidesExcludedUsers(t *testing.T) {
	for _, kc := range commentKinds() {
		t.Run(string(kc.kind), func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			owner := daotest.CreateUser(t, repos)
			first := daotest.CreateUser(t, repos)
			second := daotest.CreateUser(t, repos)
			blocked := daotest.CreateUser(t, repos)
			moderator := daotest.CreateUser(t, repos)
			target := kc.seed(t, repos, owner.ID)
			firstID := createComment(t, repos, target, first.ID, nil, "first")
			secondID := createComment(t, repos, target, second.ID, nil, "second")
			unwelcomeID := createComment(t, repos, target, blocked.ID, nil, "unwelcome")
			backdateComments(t, repos, kc.table, firstID, secondID, unwelcomeID)
			require.NoError(t, repos.User.BanUser(context.Background(), spec.UserBan{UserID: second.ID, BannedBy: moderator.ID, Reason: "spam"}))

			// when
			excluded, excludedTotal, excludedErr := repos.Comment.GetComments(context.Background(), spec.CommentQuery{Target: target, ViewerID: first.ID, Limit: 10, ExcludeUserIDs: []uuid.UUID{blocked.ID}})
			page, pageTotal, pageErr := repos.Comment.GetComments(context.Background(), spec.CommentQuery{Target: target, ViewerID: first.ID, Limit: 1, Offset: 1})

			// then
			require.NoError(t, excludedErr)
			require.NoError(t, pageErr)
			assert.Equal(t, 2, excludedTotal)
			require.Len(t, excluded, 2)
			assert.Equal(t, []uuid.UUID{firstID, secondID}, []uuid.UUID{excluded[0].ID, excluded[1].ID})
			assert.False(t, excluded[0].AuthorBanned)
			assert.True(t, excluded[1].AuthorBanned, "a banned author's comment is flagged")
			assert.Equal(t, 3, pageTotal)
			require.Len(t, page, 1)
			assert.Equal(t, secondID, page[0].ID)
		})
	}
}

func TestCommentDAO_LikesAreIdempotentAndReversible(t *testing.T) {
	for _, kc := range commentKinds() {
		t.Run(string(kc.kind), func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			author := daotest.CreateUser(t, repos)
			liker := daotest.CreateUser(t, repos)
			target := kc.seed(t, repos, author.ID)
			like := spec.CommentLike{CommentRef: spec.CommentRef{Kind: kc.kind, CommentID: createComment(t, repos, target, author.ID, nil, "likeable")}, UserID: liker.ID}

			// when
			require.NoError(t, repos.Comment.LikeComment(context.Background(), like))
			require.NoError(t, repos.Comment.LikeComment(context.Background(), like))
			liked, _ := listComments(t, repos, target, liker.ID)
			require.NoError(t, repos.Comment.UnlikeComment(context.Background(), like))
			unliked, _ := listComments(t, repos, target, liker.ID)

			// then
			require.Len(t, liked, 1)
			assert.Equal(t, 1, liked[0].LikeCount)
			assert.True(t, liked[0].UserLiked)
			require.Len(t, unliked, 1)
			assert.Equal(t, 0, unliked[0].LikeCount)
			assert.False(t, unliked[0].UserLiked)
		})
	}
}

func TestCommentDAO_MediaIsRecordedReadAndRewrittenPerComment(t *testing.T) {
	for _, kc := range commentKinds() {
		t.Run(string(kc.kind), func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			author := daotest.CreateUser(t, repos)
			target := kc.seed(t, repos, author.ID)
			withTwo := createComment(t, repos, target, author.ID, nil, "two pictures")
			withOne := createComment(t, repos, target, author.ID, nil, "one picture")
			withNone := createComment(t, repos, target, author.ID, nil, "words only")
			first := addCommentMedia(t, repos, kc.kind, withTwo, "/u/one.png", "/u/one-thumb.png")
			addCommentMedia(t, repos, kc.kind, withTwo, "/u/two.png", "")
			addCommentMedia(t, repos, kc.kind, withOne, "/u/three.png", "")

			// when
			require.NoError(t, repos.Comment.UpdateCommentMediaURL(context.Background(), spec.CommentMediaURLUpdate{Kind: kc.kind, MediaURLUpdate: spec.MediaURLUpdate{ID: first, URL: "/u/one.webm"}}))
			require.NoError(t, repos.Comment.UpdateCommentMediaThumbnail(context.Background(), spec.CommentMediaURLUpdate{Kind: kc.kind, MediaURLUpdate: spec.MediaURLUpdate{ID: first, URL: "/u/one-new-thumb.png"}}))
			missingURL := repos.Comment.UpdateCommentMediaURL(context.Background(), spec.CommentMediaURLUpdate{Kind: kc.kind, MediaURLUpdate: spec.MediaURLUpdate{ID: first + 1000, URL: "/u/ghost.png"}})
			batch, batchErr := repos.Comment.GetCommentMediaBatch(context.Background(), spec.CommentMediaBatch{Kind: kc.kind, CommentIDs: []uuid.UUID{withTwo, withOne, withNone}})
			single, singleErr := repos.Comment.GetCommentMedia(context.Background(), spec.CommentRef{Kind: kc.kind, CommentID: withTwo})
			empty, emptyErr := repos.Comment.GetCommentMediaBatch(context.Background(), spec.CommentMediaBatch{Kind: kc.kind})

			// then
			require.ErrorIs(t, missingURL, dao.ErrNotFound)
			require.NoError(t, errors.Join(batchErr, singleErr, emptyErr))
			require.Len(t, batch[withTwo], 2)
			assert.Equal(t, "/u/one.webm", batch[withTwo][0].MediaURL)
			assert.Equal(t, "/u/one-new-thumb.png", batch[withTwo][0].ThumbnailURL)
			assert.Equal(t, "/u/two.png", batch[withTwo][1].MediaURL)
			require.Len(t, batch[withOne], 1)
			assert.NotContains(t, batch, withNone)
			assert.Len(t, single, 2)
			assert.Nil(t, empty)
		})
	}
}

func TestCommentDAO_MediaPathsCoverExactlyWhatIsBeingRemoved(t *testing.T) {
	for _, kc := range commentKinds() {
		t.Run(string(kc.kind), func(t *testing.T) {
			// given a thread with media, a sibling with media, and a comment on a second target of the same kind
			repos := daotest.NewRepos(t)
			author := daotest.CreateUser(t, repos)
			target := kc.seed(t, repos, author.ID)
			doomed := createComment(t, repos, target, author.ID, nil, "doomed")
			reply := createComment(t, repos, target, author.ID, &doomed, "reply")
			kept := createComment(t, repos, target, author.ID, nil, "kept")
			addCommentMedia(t, repos, kc.kind, doomed, "/u/doomed.png", "/u/doomed-thumb.png")
			addCommentMedia(t, repos, kc.kind, reply, "/u/reply.png", "")
			addCommentMedia(t, repos, kc.kind, kept, "/u/kept.png", "")

			if kc.owned {
				otherOwner := daotest.CreateUser(t, repos)
				elsewhere := createComment(t, repos, kc.seed(t, repos, otherOwner.ID), author.ID, nil, "elsewhere")
				addCommentMedia(t, repos, kc.kind, elsewhere, "/u/elsewhere.png", "")
			}

			// when
			single, singleErr := repos.Comment.CollectSingleCommentMediaPaths(context.Background(), spec.CommentRef{Kind: kc.kind, CommentID: doomed})
			all, allErr := repos.Comment.CollectCommentMediaPaths(context.Background(), target)

			// then a comment's paths take its replies with it, and a target's paths stop at the target
			require.NoError(t, errors.Join(singleErr, allErr))
			assert.ElementsMatch(t, []string{"/u/doomed.png", "/u/doomed-thumb.png", "/u/reply.png"}, single)
			assert.ElementsMatch(t, []string{"/u/doomed.png", "/u/doomed-thumb.png", "/u/reply.png", "/u/kept.png"}, all)
		})
	}
}

func TestCommentDAO_UnknownIDsReadAsNotFound(t *testing.T) {
	for _, kc := range commentKinds() {
		t.Run(string(kc.kind), func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			missing := spec.CommentRef{Kind: kc.kind, CommentID: uuid.New()}

			// when
			row, rowErr := repos.Comment.GetCommentByID(context.Background(), missing)
			_, targetErr := repos.Comment.GetCommentTarget(context.Background(), missing)
			_, authorErr := repos.Comment.GetCommentAuthorID(context.Background(), missing)

			// then
			require.NoError(t, rowErr)
			assert.Nil(t, row)
			assert.ErrorIs(t, targetErr, dao.ErrNotFound)
			assert.ErrorIs(t, authorErr, dao.ErrNotFound)
		})
	}
}

func TestCommentDAO_AMissingOwnedTargetReadsAsNotFound(t *testing.T) {
	for _, kc := range commentKinds() {
		if !kc.owned || kc.kind == spec.CommentKindJournalEntry {
			continue
		}

		t.Run(string(kc.kind), func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)

			// when
			_, err := repos.Comment.GetTargetOwnerID(context.Background(), spec.CommentTarget{Kind: kc.kind, ID: uuid.New()})

			// then
			assert.ErrorIs(t, err, dao.ErrNotFound)
		})
	}
}

func TestCommentDAO_AnUnknownKindIsRefused(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	unknown := spec.CommentKind("teapot_comment")

	// when
	_, createErr := repos.Comment.CreateComment(context.Background(), spec.NewComment{Target: spec.CommentTarget{Kind: unknown, ID: uuid.New()}, UserID: uuid.New(), Body: "hi"})
	_, dirErr := repos.Comment.GetCommentMediaDir(unknown)

	// then
	assert.ErrorIs(t, createErr, dao.ErrUnknownCommentKind)
	assert.ErrorIs(t, dirErr, dao.ErrUnknownCommentKind)
}

func TestCommentDAO_JournalEntryCommentsStayOnTheirOwnEntry(t *testing.T) {
	// given a journal with two entries, each with its own comment, and a comment on the journal itself
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, owner.ID, "Reading Log", "", "umineko")
	journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 1, Body: "one", WordCount: 1})
	journalCreateEntry(t, repos, spec.NewJournalEntry{JournalID: journalID, EntryNumber: 2, Body: "two", WordCount: 1})
	root := spec.CommentTarget{Kind: spec.CommentKindJournal, ID: journalID}
	entryOne := spec.CommentTarget{Kind: spec.CommentKindJournalEntry, ID: journalID, EntryNumber: 1}
	entryTwo := spec.CommentTarget{Kind: spec.CommentKindJournalEntry, ID: journalID, EntryNumber: 2}
	rootID := createComment(t, repos, root, owner.ID, nil, "on the journal")
	oneID := createComment(t, repos, entryOne, owner.ID, nil, "on entry one")
	twoID := createComment(t, repos, entryTwo, owner.ID, nil, "on entry two")

	// when
	rootRows, _ := listComments(t, repos, root, owner.ID)
	oneRows, _ := listComments(t, repos, entryOne, owner.ID)
	twoRows, _ := listComments(t, repos, entryTwo, owner.ID)
	oneTarget, err := repos.Comment.GetCommentTarget(context.Background(), spec.CommentRef{Kind: spec.CommentKindJournal, CommentID: oneID})

	// then
	require.NoError(t, err)
	require.Len(t, rootRows, 1)
	assert.Equal(t, rootID, rootRows[0].ID)
	require.Len(t, oneRows, 1)
	assert.Equal(t, oneID, oneRows[0].ID)
	require.NotNil(t, oneRows[0].EntryID)
	require.Len(t, twoRows, 1)
	assert.Equal(t, twoID, twoRows[0].ID)
	assert.Equal(t, entryOne, oneTarget, "a journal comment's own row says which entry it sits on")
}

func TestCommentRepository_AnAdminEditIsAuditedInTheSameWrite(t *testing.T) {
	cases := []struct {
		name      string
		asAdmin   bool
		wantErr   error
		wantAudit bool
		wantBody  string
	}{
		{name: "an admin edit lands with its audit row", asAdmin: true, wantAudit: true, wantBody: "moderated"},
		{name: "a refused edit leaves the comment and writes no audit row", wantErr: dao.ErrNotFound, wantBody: "original"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			author := daotest.CreateUser(t, repos)
			moderator := daotest.CreateUser(t, repos)
			target := spec.CommentTarget{Kind: spec.CommentKindShip, ID: createShip(t, repos, author.ID, "BattlerBeato", makeChars())}
			ref := spec.CommentRef{Kind: spec.CommentKindShip, CommentID: createComment(t, repos, target, author.ID, nil, "original")}

			// when
			err := repos.Comment.UpdateCommentWithAudit(context.Background(), spec.AuditedCommentUpdate{
				CommentUpdate: spec.CommentUpdate{CommentRef: ref, UserID: moderator.ID, Body: "moderated", AsAdmin: tc.asAdmin},
				Audit:         audit.NewEntry{ActorID: moderator.ID, Action: audit.ActionShipCommentUpdateAdmin, TargetType: audit.TargetShipComment, TargetID: ref.CommentID.String(), SubjectID: author.ID},
			})

			// then
			require.ErrorIs(t, err, tc.wantErr)
			rows, _ := listComments(t, repos, target, author.ID)
			require.Len(t, rows, 1)
			assert.Equal(t, tc.wantBody, rows[0].Body)

			entries, _, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Action: audit.ActionShipCommentUpdateAdmin, Page: bounds.NewPage(10, 0)})
			require.NoError(t, err)
			if !tc.wantAudit {
				assert.Empty(t, entries)
				return
			}

			require.Len(t, entries, 1)
			assert.Equal(t, moderator.ID, entries[0].ActorID)
			assert.Equal(t, audit.TargetShipComment, entries[0].TargetType)
			assert.Equal(t, ref.CommentID.String(), entries[0].TargetID)
			require.NotNil(t, entries[0].SubjectID)
			assert.Equal(t, author.ID, *entries[0].SubjectID)
		})
	}
}

func TestCommentRepository_ADeleteReturnsThatCommentsFilesAndIsAudited(t *testing.T) {
	cases := []struct {
		name        string
		byAuthor    bool
		forgedActor bool
		wantErr     error
		wantPaths   []string
		wantLeft    int
		wantAudit   bool
	}{
		{name: "the author's delete removes the thread, returns its files and is audited", byAuthor: true, wantPaths: []string{"/u/doomed.png", "/u/doomed-thumb.png", "/u/reply.png"}, wantLeft: 1, wantAudit: true},
		{name: "a stranger's delete leaves everything in place and writes no audit row", wantErr: dao.ErrNotFound, wantLeft: 3},
		{name: "an audit row that cannot be written rolls the delete back", byAuthor: true, forgedActor: true, wantLeft: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			author := daotest.CreateUser(t, repos)
			stranger := daotest.CreateUser(t, repos)
			target := spec.CommentTarget{Kind: spec.CommentKindFanfic, ID: createFanfic(t, repos, author.ID, "A Tea Party")}
			doomed := createComment(t, repos, target, author.ID, nil, "doomed")
			reply := createComment(t, repos, target, author.ID, &doomed, "reply")
			kept := createComment(t, repos, target, author.ID, nil, "kept")
			addCommentMedia(t, repos, spec.CommentKindFanfic, doomed, "/u/doomed.png", "/u/doomed-thumb.png")
			addCommentMedia(t, repos, spec.CommentKindFanfic, reply, "/u/reply.png", "")
			addCommentMedia(t, repos, spec.CommentKindFanfic, kept, "/u/kept.png", "")

			actorID := stranger.ID
			if tc.byAuthor {
				actorID = author.ID
			}
			auditActor := actorID
			if tc.forgedActor {
				auditActor = uuid.New()
			}
			ref := spec.CommentRef{Kind: spec.CommentKindFanfic, CommentID: doomed}

			// when
			paths, err := repos.Comment.DeleteCommentWithAudit(context.Background(), spec.AuditedCommentDeletion{
				CommentDeletion: spec.CommentDeletion{CommentRef: ref, UserID: actorID},
				Audit:           audit.NewEntry{ActorID: auditActor, Action: audit.ActionFanficCommentDelete, TargetType: audit.TargetFanficComment, TargetID: doomed.String(), SubjectID: author.ID},
			})

			// then
			if tc.forgedActor {
				require.Error(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
			assert.ElementsMatch(t, tc.wantPaths, paths)
			_, left := listComments(t, repos, target, author.ID)
			assert.Equal(t, tc.wantLeft, left)

			entries, _, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Action: audit.ActionFanficCommentDelete, Page: bounds.NewPage(10, 0)})
			require.NoError(t, err)
			if tc.wantAudit {
				require.Len(t, entries, 1)
				assert.Equal(t, doomed.String(), entries[0].TargetID)
				return
			}

			assert.Empty(t, entries)
		})
	}
}
