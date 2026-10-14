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
		expect func(postDAO *dao.MockPostDAO, audit *MockAuditLogRepository, err error)
		call   func(repo PostRepository) error
	}
)

func newCachedPostRepo(t *testing.T) (PostRepository, *dao.MockPostDAO, *MockAuditLogRepository, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	postDAO := dao.NewMockPostDAO(t)
	audit := NewMockAuditLogRepository(t)

	return NewPostRepo(nil, postDAO, audit, cache.NewManager(engines.NewValkeyWithClient(client))), postDAO, audit, client
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
			repo, postDAO, _, client := newCachedPostRepo(t)
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
			assert.Equal(t, []string{"GET", cache.PostCornerCounts.Key()}, (*commands)[0])
		})
	}
}

func postCornerCountWriters(postID, userID uuid.UUID) []postWriterCase {
	return []postWriterCase{
		{
			name: "create",
			expect: func(postDAO *dao.MockPostDAO, _ *MockAuditLogRepository, err error) {
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
			expect: func(postDAO *dao.MockPostDAO, audit *MockAuditLogRepository, err error) {
				postDAO.EXPECT().GetSharedContentFields(mock.Anything, postID, mock.Anything).Return(nil, nil, nil)
				postDAO.EXPECT().CollectMediaPaths(mock.Anything, postID, mock.Anything).Return(nil, nil)
				postDAO.EXPECT().CollectCommentMediaPaths(mock.Anything, postID, mock.Anything).Return(nil, nil)
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
	}
}

func TestPostRepository_CornerCountWritersInvalidateTheCount(t *testing.T) {
	postID, userID := uuid.New(), uuid.New()

	for _, tc := range postCornerCountWriters(postID, userID) {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, postDAO, audit, client := newCachedPostRepo(t)
			tc.expect(postDAO, audit, nil)

			var commands []string
			captureDel(client, &commands)

			// when
			err := tc.call(repo)

			// then
			require.NoError(t, err)
			assert.Equal(t, []string{"DEL", cache.PostCornerCounts.Key()}, commands)
		})
	}
}

func TestPostRepository_CornerCountWritersSkipInvalidationOnDaoError(t *testing.T) {
	postID, userID := uuid.New(), uuid.New()

	for _, tc := range postCornerCountWriters(postID, userID) {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, postDAO, audit, client := newCachedPostRepo(t)
			tc.expect(postDAO, audit, errors.New("db down"))
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
			repo, postDAO, audit, client := newCachedPostRepo(t)
			tc.expect(postDAO, audit, nil)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(valkeymock.ErrorResult(errors.New("valkey down"))).Times(1)

			// when
			err := tc.call(repo)

			// then the stale count ages out on its TTL instead of failing the write
			require.NoError(t, err)
		})
	}
}
