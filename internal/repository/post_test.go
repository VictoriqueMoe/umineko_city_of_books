package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

type (
	postWriterCase struct {
		name   string
		expect func(postDAO *dao.MockPostDAO, comments *dao.MockCommentDAO, audit *MockAuditLogRepository, err error)
		call   func(repo PostRepository) error
	}
)

func newCachedPostRepo(t *testing.T) (PostRepository, *dao.MockPostDAO, *dao.MockCommentDAO, *MockAuditLogRepository, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	postDAO := dao.NewMockPostDAO(t)
	comments := dao.NewMockCommentDAO(t)
	audit := NewMockAuditLogRepository(t)

	return NewPostRepo(nil, postDAO, comments, audit, cache.NewManager(engines.NewValkeyWithClient(client))), postDAO, comments, audit, client
}

func cachedCornerCounts(payload string) func(cmd valkey.Completed) valkey.ValkeyResult {
	return func(_ valkey.Completed) valkey.ValkeyResult {
		return valkeymock.Result(valkeymock.ValkeyString(payload))
	}
}

func TestPostRepository_GetCornerCountsIsCached(t *testing.T) {
	cases := []struct {
		name         string
		warm         bool
		wantCommands int
	}{
		{name: "a cold cache loads from the dao and stores the counts", wantCommands: 2},
		{name: "a warm cache serves the counts without the dao", warm: true, wantCommands: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a strict dao mock that only expects a load on a cold cache
			repo, postDAO, _, _, client := newCachedPostRepo(t)
			reply := settingsMissThenStore
			if tc.warm {
				reply = cachedCornerCounts(`{"general":4,"suggestions":2}`)
			} else {
				postDAO.EXPECT().GetCornerCounts(mock.Anything).Return(map[string]int{"general": 4, "suggestions": 2}, nil).Once()
			}
			commands := captureSettingsCommands(client, tc.wantCommands, reply)

			// when
			got, err := repo.GetCornerCounts(context.Background())

			// then
			require.NoError(t, err)
			assert.Equal(t, map[string]int{"general": 4, "suggestions": 2}, got)
			assert.Equal(t, []string{"GET", cache.PostCornerCounts.Key().String()}, (*commands)[0])
		})
	}
}

func postCornerCountWriters(postID, userID uuid.UUID) []postWriterCase {
	return []postWriterCase{
		{
			name: "create",
			expect: func(postDAO *dao.MockPostDAO, _ *dao.MockCommentDAO, _ *MockAuditLogRepository, err error) {
				var created *model.PostRow
				if err == nil {
					created = &model.PostRow{ID: postID}
				}

				postDAO.EXPECT().Create(mock.Anything, spec.NewPost{UserID: userID, Corner: "general", Body: "b"}, mock.Anything).Return(created, err)
			},
			call: func(repo PostRepository) error {
				_, err := repo.CreateWithDetails(context.Background(), spec.NewPost{UserID: userID, Corner: "general", Body: "b"}, new(sql.Tx))

				return err
			},
		},
		{
			name: "delete",
			expect: func(postDAO *dao.MockPostDAO, comments *dao.MockCommentDAO, audit *MockAuditLogRepository, err error) {
				postDAO.EXPECT().GetSharedContentFields(mock.Anything, postID, mock.Anything).Return(nil, nil, nil)
				postDAO.EXPECT().CollectMediaPaths(mock.Anything, postID, mock.Anything).Return(nil, nil)
				comments.EXPECT().CollectCommentMediaPaths(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindPost, ID: postID}, mock.Anything).Return(nil, nil)
				postDAO.EXPECT().Delete(mock.Anything, spec.OwnedDeletion{ID: postID, UserID: userID}, mock.Anything).Return(err)
				if err == nil {
					audit.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				}
			},
			call: func(repo PostRepository) error {
				_, _, err := repo.DeleteWithSharedContent(context.Background(), spec.PostDelete{ID: postID, UserID: userID}, new(sql.Tx))

				return err
			},
		},
		{
			name: "delete with media",
			expect: func(postDAO *dao.MockPostDAO, comments *dao.MockCommentDAO, _ *MockAuditLogRepository, err error) {
				postDAO.EXPECT().CollectMediaPaths(mock.Anything, postID, mock.Anything).Return(nil, nil)
				comments.EXPECT().CollectCommentMediaPaths(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindPost, ID: postID}, mock.Anything).Return(nil, nil)
				postDAO.EXPECT().Delete(mock.Anything, spec.OwnedDeletion{ID: postID, UserID: userID}, mock.Anything).Return(err)
			},
			call: func(repo PostRepository) error {
				_, err := repo.DeleteWithMedia(context.Background(), spec.OwnedDeletion{ID: postID, UserID: userID}, new(sql.Tx))

				return err
			},
		},
	}
}

func TestPostRepository_DeleteWithMediaReturnsEveryStoredFileWithoutAuditing(t *testing.T) {
	// given a post carrying its own media and a comment's media, with one path shared by both
	repo, postDAO, comments, _, client := newCachedPostRepo(t)
	postID, userID := uuid.New(), uuid.New()
	postDAO.EXPECT().CollectMediaPaths(mock.Anything, postID, mock.Anything).Return([]string{"/uploads/posts/a.webp", "/uploads/posts/a_thumb.webp"}, nil)
	comments.EXPECT().CollectCommentMediaPaths(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindPost, ID: postID}, mock.Anything).Return([]string{"/uploads/posts/c.webp", "/uploads/posts/a.webp"}, nil)
	postDAO.EXPECT().Delete(mock.Anything, spec.OwnedDeletion{ID: postID, UserID: userID}, mock.Anything).Return(nil)

	var commands []string
	captureDel(client, &commands)

	// when
	paths, err := repo.DeleteWithMedia(context.Background(), spec.OwnedDeletion{ID: postID, UserID: userID}, new(sql.Tx))

	// then every stored file is handed back once, the count is invalidated, and the strict audit mock records no row for a rollback
	require.NoError(t, err)
	assert.Equal(t, []string{"/uploads/posts/a.webp", "/uploads/posts/a_thumb.webp", "/uploads/posts/c.webp"}, paths)
	assert.Equal(t, []string{"DEL", cache.PostCornerCounts.Key().String()}, commands)
}

func TestPostRepository_CornerCountWritersInvalidateTheCount(t *testing.T) {
	postID, userID := uuid.New(), uuid.New()

	for _, tc := range postCornerCountWriters(postID, userID) {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, postDAO, comments, audit, client := newCachedPostRepo(t)
			tc.expect(postDAO, comments, audit, nil)

			var commands []string
			captureDel(client, &commands)

			// when
			err := tc.call(repo)

			// then
			require.NoError(t, err)
			assert.Equal(t, []string{"DEL", cache.PostCornerCounts.Key().String()}, commands)
		})
	}
}

func TestPostRepository_CornerCountWritersSkipInvalidationOnDaoError(t *testing.T) {
	postID, userID := uuid.New(), uuid.New()

	for _, tc := range postCornerCountWriters(postID, userID) {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, postDAO, comments, audit, client := newCachedPostRepo(t)
			tc.expect(postDAO, comments, audit, errors.New("db down"))
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

			// when
			err := tc.call(repo)

			// then
			require.Error(t, err)
		})
	}
}

func TestPostRepository_AFailedInvalidationDoesNotFailACommittedWrite(t *testing.T) {
	postID, userID := uuid.New(), uuid.New()

	for _, tc := range postCornerCountWriters(postID, userID) {
		t.Run(tc.name, func(t *testing.T) {
			// given a write that commits while the cache is unreachable
			repo, postDAO, comments, audit, client := newCachedPostRepo(t)
			tc.expect(postDAO, comments, audit, nil)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(valkeymock.ErrorResult(errors.New("valkey down"))).Times(1)

			// when
			err := tc.call(repo)

			// then the stale count ages out on its TTL instead of failing the write
			require.NoError(t, err)
		})
	}
}
