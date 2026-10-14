package repository

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func newCachedBlockRepo(t *testing.T) (BlockRepository, *dao.MockBlockDAO, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	blockDAO := dao.NewMockBlockDAO(t)

	return NewBlockRepo(blockDAO, cache.NewManager(engines.NewValkeyWithClient(client))), blockDAO, client
}

func TestBlockRepo_GetBlockedIDs_CachesTheDaoResultPerUser(t *testing.T) {
	// given
	repo, blockDAO, client := newCachedBlockRepo(t)
	userID := uuid.New()
	blocked := []uuid.UUID{uuid.New()}
	blockDAO.EXPECT().GetBlockedIDs(mock.Anything, userID).Return(blocked, nil).Once()

	var commands [][]string
	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			commands = append(commands, cmd.Commands())

			if cmd.Commands()[0] == "GET" {
				return valkeymock.ErrorResult(valkey.Nil)
			}

			return valkeymock.Result(valkeymock.ValkeyString("OK"))
		}).
		Times(2)

	// when
	got, err := repo.GetBlockedIDs(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, blocked, got)
	require.Len(t, commands, 2)
	assert.Equal(t, []string{"GET", cache.UserBlockedIDs.Key(userID.String())}, commands[0])
	assert.Equal(t, "SET", commands[1][0])
	assert.Equal(t, cache.UserBlockedIDs.Key(userID.String()), commands[1][1])
}

func TestBlockRepo_WritesInvalidateBothUsersBlockedIDs(t *testing.T) {
	tests := []struct {
		name  string
		write func(repo BlockRepository, blockDAO *dao.MockBlockDAO, s spec.BlockSpec) error
	}{
		{
			name: "blocking someone",
			write: func(repo BlockRepository, blockDAO *dao.MockBlockDAO, s spec.BlockSpec) error {
				blockDAO.EXPECT().Block(mock.Anything, s).Return(nil).Once()

				return repo.Block(context.Background(), s)
			},
		},
		{
			name: "unblocking someone",
			write: func(repo BlockRepository, blockDAO *dao.MockBlockDAO, s spec.BlockSpec) error {
				blockDAO.EXPECT().Unblock(mock.Anything, s).Return(nil).Once()

				return repo.Unblock(context.Background(), s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, blockDAO, client := newCachedBlockRepo(t)
			s := spec.BlockSpec{BlockerID: uuid.New(), BlockedID: uuid.New()}

			var commands []string
			client.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
					commands = cmd.Commands()

					return valkeymock.Result(valkeymock.ValkeyInt64(2))
				}).
				Times(1)

			// when
			err := tc.write(repo, blockDAO, s)

			// then
			require.NoError(t, err)
			assert.Equal(t, []string{
				"DEL",
				cache.UserBlockedIDs.Key(s.BlockerID.String()),
				cache.UserBlockedIDs.Key(s.BlockedID.String()),
			}, commands)
		})
	}
}

func TestBlockRepo_FailedWriteKeepsTheCache(t *testing.T) {
	// given
	repo, blockDAO, client := newCachedBlockRepo(t)
	s := spec.BlockSpec{BlockerID: uuid.New(), BlockedID: uuid.New()}
	blockDAO.EXPECT().Block(mock.Anything, s).Return(errors.New("db down")).Once()
	client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	// when
	err := repo.Block(context.Background(), s)

	// then
	require.Error(t, err)
}
