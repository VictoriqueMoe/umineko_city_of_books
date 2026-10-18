package dao_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	postPollExpiresAt = time.Now().Add(24 * time.Hour).UTC().Format("2006-01-02 15:04:05")
)

func createPost(t *testing.T, repos *repository.Repositories, userID uuid.UUID, corner, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Post.Create(context.Background(), spec.NewPost{UserID: userID, Corner: corner, Body: body})
	require.NoError(t, err)

	return created.ID
}

func postAttachMedia(t *testing.T, add func(context.Context, spec.NewMedia, ...*sql.Tx) (int64, error), targetID uuid.UUID, mediaURL, thumbnailURL string) int64 {
	t.Helper()
	id, err := add(context.Background(), spec.NewMedia{TargetID: targetID, MediaURL: mediaURL, MediaType: "image", ThumbnailURL: thumbnailURL})
	require.NoError(t, err)

	return id
}

func postGet(t *testing.T, repos *repository.Repositories, postID, viewerID uuid.UUID) *model.PostRow {
	t.Helper()
	row, err := repos.Post.GetByID(context.Background(), spec.PostLookup{ID: postID, ViewerID: viewerID})
	require.NoError(t, err)

	return row
}

func postCreatePoll(t *testing.T, repos *repository.Repositories, postID uuid.UUID, options ...string) (uuid.UUID, []model.PollOptionRow) {
	t.Helper()
	ctx := context.Background()

	poll, err := repos.Post.CreatePollWithOptions(ctx, spec.NewPoll{PostID: postID, DurationSeconds: 86400, ExpiresAt: postPollExpiresAt, Options: options})
	require.NoError(t, err)

	_, opts, _, err := repos.Post.GetPollByPostID(ctx, spec.PostPollQuery{PostID: postID})
	require.NoError(t, err)
	require.Len(t, opts, len(options))

	return uuid.MustParse(poll.ID), opts
}

func postRowIDs(rows []model.PostRow) []uuid.UUID {
	var ids []uuid.UUID
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

func TestPostDAO_CreateAndGetByID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Poster"))

	// when
	id := createPost(t, repos, user.ID, "general", "hello world")

	// then
	row := postGet(t, repos, id, user.ID)
	require.NotNil(t, row)
	assert.Equal(t, "hello world", row.Body)
	assert.Equal(t, "general", row.Corner)
	assert.Equal(t, user.ID, row.UserID)
	assert.Equal(t, "Poster", row.AuthorDisplayName)
	assert.Nil(t, postGet(t, repos, uuid.New(), user.ID))
}

func TestPostDAO_GetSharedContentFields(t *testing.T) {
	tests := []struct {
		name     string
		shared   *model.SharedContentRef
		missing  bool
		wantID   *string
		wantType *string
	}{
		{name: "a post created with shared content reports its id and type", shared: &model.SharedContentRef{ID: "abc123", Type: "theory"}, wantID: new("abc123"), wantType: new("theory")},
		{name: "a post without shared content reports neither"},
		{name: "a missing post reports neither rather than an error", missing: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos)

			created, err := repos.Post.Create(ctx, spec.NewPost{UserID: user.ID, Corner: "general", Body: "shared", SharedContent: tc.shared})
			require.NoError(t, err)

			postID := created.ID
			if tc.missing {
				postID = uuid.New()
			}

			// when
			contentID, contentType, err := repos.Post.GetSharedContentFields(ctx, postID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantID, contentID)
			assert.Equal(t, tc.wantType, contentType)
		})
	}
}

func TestPostDAO_UpdatePost(t *testing.T) {
	tests := []struct {
		name     string
		byOther  bool
		asAdmin  bool
		missing  bool
		wantErr  bool
		wantBody string
	}{
		{name: "the owner edits their own post", wantBody: "edited"},
		{name: "another user cannot edit the post", byOther: true, wantErr: true, wantBody: "original"},
		{name: "an admin edits any post", byOther: true, asAdmin: true, wantBody: "edited"},
		{name: "an admin edit of a missing post fails", byOther: true, asAdmin: true, missing: true, wantErr: true, wantBody: "original"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			postID := createPost(t, repos, owner.ID, "general", "original")

			editorID := owner.ID
			if tc.byOther {
				editorID = other.ID
			}

			targetID := postID
			if tc.missing {
				targetID = uuid.New()
			}

			// when
			err := repos.Post.UpdatePost(context.Background(), spec.PostUpdate{ID: targetID, UserID: editorID, Body: "edited", AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantBody, postGet(t, repos, postID, owner.ID).Body)
		})
	}
}

func TestPostDAO_DeleteWithSharedContent(t *testing.T) {
	everyPath := []string{
		"/uploads/posts/a.webp",
		"/uploads/posts/a_thumb.webp",
		"/uploads/posts/b.webp",
		"/uploads/posts/c.webp",
		"/uploads/posts/c_thumb.webp",
		"/uploads/posts/d.webp",
	}

	tests := []struct {
		name      string
		byOther   bool
		asAdmin   bool
		action    audit.Action
		withMedia bool
		wantErr   bool
		wantPaths []string
	}{
		{name: "an owner delete returns every uploaded path of the post and its comment tree", action: audit.ActionPostDelete, withMedia: true, wantPaths: everyPath},
		{name: "an admin delete returns every uploaded path of the post and its comment tree", byOther: true, asAdmin: true, action: audit.ActionPostDeleteAdmin, withMedia: true, wantPaths: everyPath},
		{name: "an owner delete of a post with no media returns no paths", action: audit.ActionPostDelete},
		{name: "a failed delete by a stranger returns no paths and keeps the media", byOther: true, action: audit.ActionPostDeleteAdmin, withMedia: true, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			postID := createPost(t, repos, owner.ID, "general", "body")
			survivorID := createPost(t, repos, owner.ID, "general", "survivor")
			survivorCommentID := createComment(t, repos, spec.CommentTarget{Kind: spec.CommentKindPost, ID: survivorID}, owner.ID, nil, "untouched")
			addCommentMedia(t, repos, spec.CommentKindPost, survivorCommentID, "/uploads/posts/keep.webp", "")

			if tc.withMedia {
				postAttachMedia(t, repos.Post.AddMedia, postID, "/uploads/posts/a.webp", "/uploads/posts/a_thumb.webp")
				postAttachMedia(t, repos.Post.AddMedia, postID, "/uploads/posts/b.webp", "")
				target := spec.CommentTarget{Kind: spec.CommentKindPost, ID: postID}
				commentID := createComment(t, repos, target, owner.ID, nil, "comment")
				replyID := createComment(t, repos, target, owner.ID, &commentID, "reply")
				addCommentMedia(t, repos, spec.CommentKindPost, commentID, "/uploads/posts/c.webp", "/uploads/posts/c_thumb.webp")
				addCommentMedia(t, repos, spec.CommentKindPost, replyID, "/uploads/posts/d.webp", "")
			}

			actorID := owner.ID
			if tc.byOther {
				actorID = other.ID
			}

			// when
			shared, paths, err := repos.Post.DeleteWithSharedContent(ctx, spec.PostDelete{
				ID:      postID,
				UserID:  actorID,
				AsAdmin: tc.asAdmin,
				Audit: audit.NewEntry{
					ActorID:    actorID,
					Action:     tc.action,
					TargetType: audit.TargetPost,
					TargetID:   postID.String(),
					SubjectID:  owner.ID,
				},
			})

			// then
			assert.Nil(t, shared)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, paths)
				assert.NotNil(t, postGet(t, repos, postID, owner.ID))

				media, err := repos.Post.GetMedia(ctx, postID)
				require.NoError(t, err)
				assert.Len(t, media, 2)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tc.wantPaths, paths)
				assert.Nil(t, postGet(t, repos, postID, owner.ID))
			}
		})
	}
}

func TestPostDAO_ListAll(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	apple := createPost(t, repos, user.ID, "general", "apple pie")
	banana := createPost(t, repos, user.ID, "general", "banana bread")
	blockedPost := createPost(t, repos, blocked.ID, "general", "blocked")
	open := createPost(t, repos, user.ID, "suggestions", "open suggestion")
	done := createPost(t, repos, user.ID, "suggestions", "done suggestion")
	archived := createPost(t, repos, user.ID, "suggestions", "archived suggestion")
	require.NoError(t, repos.Post.ResolveSuggestion(ctx, spec.SuggestionResolution{PostID: done, ResolvedBy: user.ID, Status: "done"}))
	require.NoError(t, repos.Post.ResolveSuggestion(ctx, spec.SuggestionResolution{PostID: archived, ResolvedBy: user.ID, Status: "archived"}))

	generalFeed, _, err := repos.Post.ListAll(ctx, spec.PostFeedQuery{ViewerID: user.ID, Corner: "general", Sort: "new", Limit: 10})
	require.NoError(t, err)

	generalOrder := postRowIDs(generalFeed)
	require.Len(t, generalOrder, 3)

	tests := []struct {
		name      string
		query     spec.PostFeedQuery
		wantTotal int
		want      []uuid.UUID
	}{
		{name: "a corner lists only its own posts", query: spec.PostFeedQuery{Corner: "general", Sort: "new", Limit: 10}, wantTotal: 3, want: []uuid.UUID{apple, banana, blockedPost}},
		{name: "no sort falls back to relevance over the same posts", query: spec.PostFeedQuery{Corner: "general", Limit: 10}, wantTotal: 3, want: []uuid.UUID{apple, banana, blockedPost}},
		{name: "search matches the body", query: spec.PostFeedQuery{Corner: "general", Search: "apple", Sort: "new", Limit: 10}, wantTotal: 1, want: []uuid.UUID{apple}},
		{name: "excluded users' posts are hidden from the page and the total", query: spec.PostFeedQuery{Corner: "general", Sort: "new", Limit: 10, ExcludeUserIDs: []uuid.UUID{blocked.ID}}, wantTotal: 2, want: []uuid.UUID{apple, banana}},
		{name: "the open filter hides resolved suggestions", query: spec.PostFeedQuery{Corner: "suggestions", Sort: "new", Limit: 10, ResolvedFilter: "open"}, wantTotal: 1, want: []uuid.UUID{open}},
		{name: "the done filter lists only done suggestions", query: spec.PostFeedQuery{Corner: "suggestions", Sort: "new", Limit: 10, ResolvedFilter: "done"}, wantTotal: 1, want: []uuid.UUID{done}},
		{name: "the archived filter lists only archived suggestions", query: spec.PostFeedQuery{Corner: "suggestions", Sort: "new", Limit: 10, ResolvedFilter: "archived"}, wantTotal: 1, want: []uuid.UUID{archived}},
		{name: "a limit caps the page while the total counts every match", query: spec.PostFeedQuery{Corner: "general", Sort: "new", Limit: 2}, wantTotal: 3, want: generalOrder[:2]},
		{name: "an offset skips exactly the posts the previous page returned", query: spec.PostFeedQuery{Corner: "general", Sort: "new", Limit: 2, Offset: 2}, wantTotal: 3, want: generalOrder[2:]},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			tc.query.ViewerID = user.ID
			posts, total, err := repos.Post.ListAll(ctx, tc.query)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)
			assert.ElementsMatch(t, tc.want, postRowIDs(posts))
		})
	}
}

func TestPostDAO_ListAll_Sort(t *testing.T) {
	tests := []struct {
		name string
		sort string
	}{
		{name: "likes puts the most liked post first", sort: "likes"},
		{name: "comments puts the most commented post first", sort: "comments"},
		{name: "views puts the most viewed post first", sort: "views"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos)
			boosted := createPost(t, repos, user.ID, "general", "boosted")
			other := createPost(t, repos, user.ID, "general", "plain")

			switch tc.sort {
			case "likes":
				require.NoError(t, repos.Post.Like(ctx, spec.Like{UserID: user.ID, TargetID: boosted}))
			case "comments":
				createComment(t, repos, spec.CommentTarget{Kind: spec.CommentKindPost, ID: boosted}, user.ID, nil, "c1")
			case "views":
				_, err := repos.Post.RecordView(ctx, spec.ViewRecord{TargetID: boosted, ViewerHash: "hash1"})
				require.NoError(t, err)
			}

			// when
			posts, _, err := repos.Post.ListAll(ctx, spec.PostFeedQuery{ViewerID: user.ID, Corner: "general", Sort: tc.sort, Limit: 10})

			// then
			require.NoError(t, err)
			assert.Equal(t, []uuid.UUID{boosted, other}, postRowIDs(posts))
		})
	}
}

func postBackdate(t *testing.T, repos *repository.Repositories, postID uuid.UUID, age time.Duration) {
	t.Helper()

	_, err := repos.DB().ExecContext(context.Background(), "UPDATE posts SET created_at = NOW() - make_interval(secs => $1) WHERE id = $2", age.Seconds(), postID)
	require.NoError(t, err)

	_, err = repos.DB().ExecContext(context.Background(), "SELECT refresh_post_engagement($1)", postID)
	require.NoError(t, err)
}

func TestPostDAO_ListAll_RelevanceFollowsEngagement(t *testing.T) {
	tests := []struct {
		name   string
		engage func(t *testing.T, repos *repository.Repositories, older, fan uuid.UUID)
		older  bool
	}{
		{
			name:   "without engagement the newer post ranks first",
			engage: func(t *testing.T, repos *repository.Repositories, older, fan uuid.UUID) {},
		},
		{
			name: "a like lifts the older post above the newer one",
			engage: func(t *testing.T, repos *repository.Repositories, older, fan uuid.UUID) {
				require.NoError(t, repos.Post.Like(context.Background(), spec.Like{UserID: fan, TargetID: older}))
			},
			older: true,
		},
		{
			name: "a comment lifts the older post above the newer one",
			engage: func(t *testing.T, repos *repository.Repositories, older, fan uuid.UUID) {
				createComment(t, repos, spec.CommentTarget{Kind: spec.CommentKindPost, ID: older}, fan, nil, "great post")
			},
			older: true,
		},
		{
			name: "removing the like drops the older post back down",
			engage: func(t *testing.T, repos *repository.Repositories, older, fan uuid.UUID) {
				ctx := context.Background()
				require.NoError(t, repos.Post.Like(ctx, spec.Like{UserID: fan, TargetID: older}))
				require.NoError(t, repos.Post.Unlike(ctx, spec.Like{UserID: fan, TargetID: older}))
			},
		},
		{
			name: "deleting the comment drops the older post back down",
			engage: func(t *testing.T, repos *repository.Repositories, older, fan uuid.UUID) {
				target := spec.CommentTarget{Kind: spec.CommentKindPost, ID: older}
				commentID := createComment(t, repos, target, fan, nil, "great post")
				ref := spec.CommentRef{Kind: spec.CommentKindPost, CommentID: commentID}
				require.NoError(t, repos.Comment.DeleteComment(context.Background(), spec.CommentDeletion{CommentRef: ref, UserID: fan}))
			},
		},
		{
			name: "deleting the liker's account drops the older post back down",
			engage: func(t *testing.T, repos *repository.Repositories, older, fan uuid.UUID) {
				ctx := context.Background()
				require.NoError(t, repos.Post.Like(ctx, spec.Like{UserID: fan, TargetID: older}))
				require.NoError(t, repos.User.DeleteAccount(ctx, fan))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			author := daotest.CreateUser(t, repos)
			fan := daotest.CreateUser(t, repos)
			older := createPost(t, repos, author.ID, "general", "older")
			newer := createPost(t, repos, author.ID, "general", "newer")
			postBackdate(t, repos, older, time.Hour)
			tc.engage(t, repos, older, fan.ID)

			// when
			posts, _, err := repos.Post.ListAll(ctx, spec.PostFeedQuery{ViewerID: author.ID, Corner: "general", Limit: 10})

			// then
			require.NoError(t, err)
			want := []uuid.UUID{newer, older}
			if tc.older {
				want = []uuid.UUID{older, newer}
			}
			assert.Equal(t, want, postRowIDs(posts))
		})
	}
}

func TestPostDAO_ListByFollowing(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	viewer := daotest.CreateUser(t, repos)
	followed := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(ctx, spec.FollowSpec{FollowerID: viewer.ID, FollowingID: followed.ID}))
	own := createPost(t, repos, viewer.ID, "general", "self")
	followedPost := createPost(t, repos, followed.ID, "general", "followed")
	createPost(t, repos, stranger.ID, "general", "stranger")

	tests := []struct {
		name  string
		query spec.PostFollowingFeedQuery
		want  []uuid.UUID
	}{
		{name: "the viewer's own and followed users' posts are listed", query: spec.PostFollowingFeedQuery{Sort: "new"}, want: []uuid.UUID{own, followedPost}},
		{name: "no sort falls back to relevance over the same posts", query: spec.PostFollowingFeedQuery{}, want: []uuid.UUID{own, followedPost}},
		{name: "excluded users' posts are hidden from the page and the total", query: spec.PostFollowingFeedQuery{Sort: "new", ExcludeUserIDs: []uuid.UUID{followed.ID}}, want: []uuid.UUID{own}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			tc.query.UserID = viewer.ID
			tc.query.Corner = "general"
			tc.query.Limit = 10
			posts, total, err := repos.Post.ListByFollowing(ctx, tc.query)

			// then
			require.NoError(t, err)
			assert.Equal(t, len(tc.want), total)
			assert.ElementsMatch(t, tc.want, postRowIDs(posts))
		})
	}
}

func TestPostDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	target := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)

	var own []uuid.UUID
	for range 4 {
		own = append(own, createPost(t, repos, target.ID, "general", "mine"))
	}

	createPost(t, repos, other.ID, "general", "not mine")

	everyPost, _, err := repos.Post.ListByUser(ctx, spec.PostUserPage{UserID: target.ID, ViewerID: target.ID, Limit: 10})
	require.NoError(t, err)

	order := postRowIDs(everyPost)
	require.Len(t, order, 4)

	tests := []struct {
		name   string
		limit  int
		offset int
		want   []uuid.UUID
	}{
		{name: "only the user's posts are listed", limit: 10, want: own},
		{name: "a limit caps the page while the total counts every post", limit: 2, offset: 1, want: order[1:3]},
		{name: "an offset near the end returns only the remaining posts", limit: 2, offset: 3, want: order[3:]},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			posts, total, err := repos.Post.ListByUser(ctx, spec.PostUserPage{UserID: target.ID, ViewerID: target.ID, Limit: tc.limit, Offset: tc.offset})

			// then
			require.NoError(t, err)
			assert.Equal(t, 4, total)
			assert.ElementsMatch(t, tc.want, postRowIDs(posts))
		})
	}
}

func TestPostDAO_Media(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	first := createPost(t, repos, user.ID, "general", "a")
	second := createPost(t, repos, user.ID, "general", "b")

	// when
	firstMediaID := postAttachMedia(t, repos.Post.AddMedia, first, "/m.jpg", "/t.jpg")
	postAttachMedia(t, repos.Post.AddMedia, second, "/b.jpg", "")
	require.NoError(t, repos.Post.UpdateMediaURL(ctx, spec.MediaURLUpdate{ID: firstMediaID, URL: "/new.jpg"}))
	require.NoError(t, repos.Post.UpdateMediaThumbnail(ctx, spec.MediaURLUpdate{ID: firstMediaID, URL: "/thumb.jpg"}))

	// then
	assert.Greater(t, firstMediaID, int64(0))

	untouched, err := repos.Post.GetMedia(ctx, second)
	require.NoError(t, err)
	require.Len(t, untouched, 1)
	assert.Equal(t, "/b.jpg", untouched[0].MediaURL)
	assert.Equal(t, "image", untouched[0].MediaType)

	updated, err := repos.Post.GetMedia(ctx, first)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, "/new.jpg", updated[0].MediaURL)
	assert.Equal(t, "/thumb.jpg", updated[0].ThumbnailURL)

	batch, err := repos.Post.GetMediaBatch(ctx, []uuid.UUID{first, second})
	require.NoError(t, err)
	assert.Len(t, batch[first], 1)
	assert.Len(t, batch[second], 1)

	none, err := repos.Post.GetMediaBatch(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestPostDAO_DeleteMedia(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	mediaID := postAttachMedia(t, repos.Post.AddMedia, postID, "/m.jpg", "")

	// when
	_, missingErr := repos.Post.DeleteMedia(ctx, spec.MediaDeletion{ID: 99999, TargetID: postID})
	url, err := repos.Post.DeleteMedia(ctx, spec.MediaDeletion{ID: mediaID, TargetID: postID})

	// then
	require.Error(t, missingErr)
	require.NoError(t, err)
	assert.Equal(t, "/m.jpg", url)

	media, err := repos.Post.GetMedia(ctx, postID)
	require.NoError(t, err)
	assert.Empty(t, media)
}

func TestPostDAO_Like(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos, daotest.WithDisplayName("Liker"))
	otherLiker := daotest.CreateUser(t, repos)
	postID := createPost(t, repos, author.ID, "general", "b")
	require.NoError(t, repos.Post.Like(ctx, spec.Like{UserID: liker.ID, TargetID: postID}))
	require.NoError(t, repos.Post.Like(ctx, spec.Like{UserID: otherLiker.ID, TargetID: postID}))

	// when
	err := repos.Post.Like(ctx, spec.Like{UserID: liker.ID, TargetID: postID})

	// then
	require.NoError(t, err)

	row := postGet(t, repos, postID, liker.ID)
	assert.Equal(t, 2, row.LikeCount)
	assert.True(t, row.UserLiked)

	everyone, err := repos.Post.GetLikedBy(ctx, spec.LikedByQuery{TargetID: postID})
	require.NoError(t, err)
	require.Len(t, everyone, 2)
	assert.ElementsMatch(t, []uuid.UUID{liker.ID, otherLiker.ID}, []uuid.UUID{everyone[0].ID, everyone[1].ID})

	filtered, err := repos.Post.GetLikedBy(ctx, spec.LikedByQuery{TargetID: postID, ExcludeUserIDs: []uuid.UUID{otherLiker.ID}})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, liker.ID, filtered[0].ID)
	assert.Equal(t, "Liker", filtered[0].DisplayName)
}

func TestPostDAO_Unlike(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	postID := createPost(t, repos, author.ID, "general", "b")
	require.NoError(t, repos.Post.Like(ctx, spec.Like{UserID: liker.ID, TargetID: postID}))

	// when
	err := repos.Post.Unlike(ctx, spec.Like{UserID: liker.ID, TargetID: postID})

	// then
	require.NoError(t, err)

	row := postGet(t, repos, postID, liker.ID)
	assert.Equal(t, 0, row.LikeCount)
	assert.False(t, row.UserLiked)
}

func TestPostDAO_RecordView_NewAndDuplicate(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")

	// when
	first, err := repos.Post.RecordView(ctx, spec.ViewRecord{TargetID: postID, ViewerHash: "hash1"})
	second, err2 := repos.Post.RecordView(ctx, spec.ViewRecord{TargetID: postID, ViewerHash: "hash1"})

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.True(t, first)
	assert.False(t, second)
	assert.Equal(t, 1, postGet(t, repos, postID, user.ID).ViewCount)
}

func TestPostDAO_AuthorAndEntityLookups(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")

	tests := []struct {
		name    string
		lookup  func(repository.PostRepository, context.Context, uuid.UUID, ...*sql.Tx) (uuid.UUID, error)
		id      uuid.UUID
		want    uuid.UUID
		wantErr bool
	}{
		{name: "a post's author", lookup: repository.PostRepository.GetPostAuthorID, id: postID, want: user.ID},
		{name: "a missing post's author is an error", lookup: repository.PostRepository.GetPostAuthorID, id: uuid.New(), wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := tc.lookup(repos.Post, ctx, tc.id)

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPostDAO_GetSharedContentAuthor(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "body")

	tests := []struct {
		name    string
		ref     model.SharedContentRef
		want    uuid.UUID
		wantErr bool
	}{
		{name: "a shared post resolves to its author", ref: model.SharedContentRef{ID: postID.String(), Type: "post"}, want: user.ID},
		{name: "shared content of an unknown type is an error", ref: model.SharedContentRef{ID: uuid.New().String(), Type: "nonsense"}, wantErr: true},
		{name: "a missing shared post is an error", ref: model.SharedContentRef{ID: uuid.New().String(), Type: "post"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := repos.Post.GetSharedContentAuthor(ctx, tc.ref)

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPostDAO_ResolveSuggestion(t *testing.T) {
	tests := []struct {
		name      string
		statuses  []string
		unresolve bool
		want      string
	}{
		{name: "resolving records the status", statuses: []string{"done"}, want: "done"},
		{name: "resolving again replaces the status", statuses: []string{"done", "archived"}, want: "archived"},
		{name: "unresolving clears the status", statuses: []string{"done"}, unresolve: true, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos)
			postID := createPost(t, repos, user.ID, "suggestions", "idea")

			// when
			for _, status := range tc.statuses {
				require.NoError(t, repos.Post.ResolveSuggestion(ctx, spec.SuggestionResolution{PostID: postID, ResolvedBy: user.ID, Status: status}))
			}

			if tc.unresolve {
				require.NoError(t, repos.Post.UnresolveSuggestion(ctx, postID))
			}

			// then
			assert.Equal(t, tc.want, postGet(t, repos, postID, user.ID).ResolvedStatus)
		})
	}
}

func TestPostDAO_CountUserPostsTodayAndCornerCounts(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	poster := daotest.CreateUser(t, repos)
	lurker := daotest.CreateUser(t, repos)
	createPost(t, repos, poster.ID, "general", "a")
	createPost(t, repos, poster.ID, "general", "b")
	createPost(t, repos, poster.ID, "suggestions", "c")

	// when
	posterToday, err := repos.Post.CountUserPostsToday(ctx, poster.ID)
	require.NoError(t, err)

	lurkerToday, err := repos.Post.CountUserPostsToday(ctx, lurker.ID)
	require.NoError(t, err)

	corners, err := repos.Post.GetCornerCounts(ctx)
	require.NoError(t, err)

	// then
	assert.Equal(t, 3, posterToday)
	assert.Equal(t, 0, lurkerToday)
	assert.Equal(t, map[string]int{"general": 2, "suggestions": 1}, corners)
}

func TestPostDAO_ShareCount(t *testing.T) {
	tests := []struct {
		name   string
		deltas []int
		want   int
	}{
		{name: "content never shared counts zero", want: 0},
		{name: "an increment starts the count at one", deltas: []int{1}, want: 1},
		{name: "increments accumulate", deltas: []int{1, 1}, want: 2},
		{name: "a decrement subtracts one", deltas: []int{1, 1, -1}, want: 1},
		{name: "decrements clamp at zero", deltas: []int{1, -1, -1}, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			ref := model.SharedContentRef{ID: "abc", Type: "post"}

			// when
			for _, delta := range tc.deltas {
				if delta > 0 {
					require.NoError(t, repos.Post.IncrementShareCount(ctx, ref))
				} else {
					require.NoError(t, repos.Post.DecrementShareCount(ctx, ref))
				}
			}

			count, err := repos.Post.GetShareCount(ctx, ref)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, count)
		})
	}
}

func TestPostDAO_GetShareCountsBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	for _, id := range []string{"a", "b", "b"} {
		require.NoError(t, repos.Post.IncrementShareCount(ctx, model.SharedContentRef{ID: id, Type: "post"}))
	}

	// when
	counts, err := repos.Post.GetShareCountsBatch(ctx, spec.SharedContentBatchRef{ContentIDs: []string{"a", "b", "c"}, ContentType: "post"})
	require.NoError(t, err)

	none, err := repos.Post.GetShareCountsBatch(ctx, spec.SharedContentBatchRef{ContentType: "post"})
	require.NoError(t, err)

	// then
	assert.Equal(t, map[string]int{"a": 1, "b": 2}, counts)
	assert.Nil(t, none)
}

func TestPostDAO_CreatePollWithOptions_GetPollByPostID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "body")
	noPollPost := createPost(t, repos, user.ID, "general", "no poll")

	// when
	created, err := repos.Post.CreatePollWithOptions(ctx, spec.NewPoll{PostID: postID, DurationSeconds: 86400, ExpiresAt: postPollExpiresAt, Options: []string{"yes", "no"}})

	// then
	require.NoError(t, err)

	poll, opts, voted, err := repos.Post.GetPollByPostID(ctx, spec.PostPollQuery{PostID: postID, ViewerID: user.ID})
	require.NoError(t, err)
	require.NotNil(t, poll)
	assert.Equal(t, created.ID, poll.ID)
	assert.Len(t, opts, 2)
	assert.Nil(t, voted)

	poll, opts, voted, err = repos.Post.GetPollByPostID(ctx, spec.PostPollQuery{PostID: noPollPost, ViewerID: user.ID})
	require.NoError(t, err)
	assert.Nil(t, poll)
	assert.Nil(t, opts)
	assert.Nil(t, voted)

	// when the viewer's own vote cannot be read
	_, err = repos.DB().ExecContext(ctx, `ALTER TABLE post_poll_votes RENAME COLUMN user_id TO user_id_gone`)
	require.NoError(t, err)
	_, _, _, err = repos.Post.GetPollByPostID(ctx, spec.PostPollQuery{PostID: postID, ViewerID: user.ID})

	// then the failure surfaces instead of reading as "not voted"
	require.Error(t, err)
}

func TestPostDAO_VotePoll(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	pollID, opts := postCreatePoll(t, repos, postID, "yes", "no")

	// when
	err := repos.Post.VotePoll(ctx, spec.PostPollVote{PollID: pollID, UserID: user.ID, OptionID: opts[0].ID})
	duplicateErr := repos.Post.VotePoll(ctx, spec.PostPollVote{PollID: pollID, UserID: user.ID, OptionID: opts[1].ID})

	// then
	require.NoError(t, err)
	require.ErrorIs(t, duplicateErr, dao.ErrAlreadyVoted)

	_, counted, voted, err := repos.Post.GetPollByPostID(ctx, spec.PostPollQuery{PostID: postID, ViewerID: user.ID})
	require.NoError(t, err)
	require.NotNil(t, voted)
	assert.Equal(t, opts[0].ID, *voted)

	total := 0
	for _, option := range counted {
		total += option.VoteCount
	}

	assert.Equal(t, 1, total)
}

func TestPostDAO_GetPollsByPostIDs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	voted := createPost(t, repos, user.ID, "general", "a")
	unvoted := createPost(t, repos, user.ID, "general", "b")
	pollID, opts := postCreatePoll(t, repos, voted, "a", "b")
	postCreatePoll(t, repos, unvoted, "c", "d")
	require.NoError(t, repos.Post.VotePoll(ctx, spec.PostPollVote{PollID: pollID, UserID: user.ID, OptionID: opts[1].ID}))

	// when
	polls, options, votes, err := repos.Post.GetPollsByPostIDs(ctx, spec.PostPollBatchQuery{PostIDs: []uuid.UUID{voted, unvoted}, ViewerID: user.ID})
	require.NoError(t, err)

	nonePolls, noneOptions, noneVotes, err := repos.Post.GetPollsByPostIDs(ctx, spec.PostPollBatchQuery{ViewerID: user.ID})
	require.NoError(t, err)

	// then
	assert.Len(t, polls, 2)
	assert.Len(t, options[voted], 2)
	assert.Len(t, options[unvoted], 2)
	require.Len(t, votes, 1)
	require.NotNil(t, votes[voted])
	assert.Equal(t, opts[1].ID, *votes[voted])

	assert.Nil(t, nonePolls)
	assert.Nil(t, noneOptions)
	assert.Nil(t, noneVotes)
}

func TestGetSharedContentPreviews_ResolvesPostsAndFlagsMissingOrUnknownRefsDeleted(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "shared body")
	postAttachMedia(t, repos.Post.AddMedia, postID, "/m.jpg", "")
	missingID := uuid.New().String()

	// when
	result, err := repos.Post.GetSharedContentPreviews(context.Background(), []model.SharedContentRef{
		{ID: postID.String(), Type: "post"},
		{ID: missingID, Type: "post"},
		{ID: "xyz", Type: "nonsense"},
	})
	require.NoError(t, err)
	empty, err := repos.Post.GetSharedContentPreviews(context.Background(), nil)
	require.NoError(t, err)

	// then
	assert.Len(t, result, 3)

	post := result["post:"+postID.String()]
	require.NotNil(t, post)
	assert.Equal(t, "post", post.ContentType)
	assert.Equal(t, "shared body", post.Body)
	assert.False(t, post.Deleted)
	assert.Len(t, post.Media, 1)

	missing := result["post:"+missingID]
	require.NotNil(t, missing)
	assert.True(t, missing.Deleted)
	assert.Equal(t, "/game-board/"+missingID, missing.URL)

	unknown := result["nonsense:xyz"]
	require.NotNil(t, unknown)
	assert.True(t, unknown.Deleted)
	assert.Equal(t, "/", unknown.URL)

	assert.Empty(t, empty)
}

func TestGetSharedContentPreviews_DraftFanficFlaggedDeleted(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	created, err := repos.Fanfic.CreateWithDetails(context.Background(), spec.NewFanficWithDetails{
		UserID:   user.ID,
		Title:    "Secret Draft",
		Summary:  "unpublished summary",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "draft",
	})
	require.NoError(t, err)

	// when
	result, err := repos.Post.GetSharedContentPreviews(context.Background(), []model.SharedContentRef{
		{ID: created.ID.String(), Type: "fanfic"},
	})

	// then
	require.NoError(t, err)
	preview := result["fanfic:"+created.ID.String()]
	require.NotNil(t, preview)
	assert.True(t, preview.Deleted)
	assert.NotContains(t, preview.Body, "unpublished summary")
	assert.NotEqual(t, "Secret Draft", preview.Title)
}

func TestGetSharedContentPreviews_SurfacesAFailedLookupInsteadOfCallingTheContentDeleted(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		sabotage    string
	}{
		{name: "post", contentType: "post", sabotage: `DROP TABLE posts CASCADE`},
		{name: "post media", contentType: "post", sabotage: `DROP TABLE post_media CASCADE`},
		{name: "art", contentType: "art", sabotage: `DROP TABLE art CASCADE`},
		{name: "ship", contentType: "ship", sabotage: `DROP TABLE ships CASCADE`},
		{name: "mystery", contentType: "mystery", sabotage: `DROP TABLE mysteries CASCADE`},
		{name: "theory", contentType: "theory", sabotage: `DROP TABLE theories CASCADE`},
		{name: "fanfic", contentType: "fanfic", sabotage: `DROP TABLE fanfics CASCADE`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			_, err := repos.DB().ExecContext(ctx, tc.sabotage)
			require.NoError(t, err)

			// when
			_, err = repos.Post.GetSharedContentPreviews(ctx, []model.SharedContentRef{{ID: uuid.New().String(), Type: tc.contentType}})

			// then
			require.Error(t, err, "a failed lookup must not be shown to users as deleted content")
		})
	}
}
