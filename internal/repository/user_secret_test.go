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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func TestUserSecretRepository_WritesSurviveAFailedInvalidation(t *testing.T) {
	unlock := spec.SecretUnlock{UserID: uuid.New(), SecretID: "witch_hunt"}
	cases := []struct {
		name   string
		expect func(secretDAO *dao.MockUserSecretDAO)
		call   func(repo UserSecretRepository) error
	}{
		{
			name: "an unlock",
			expect: func(secretDAO *dao.MockUserSecretDAO) {
				secretDAO.EXPECT().Unlock(mock.Anything, unlock).Return(nil)
			},
			call: func(repo UserSecretRepository) error {
				return repo.Unlock(context.Background(), unlock)
			},
		},
		{
			name: "a secret deletion",
			expect: func(secretDAO *dao.MockUserSecretDAO) {
				secretDAO.EXPECT().DeleteSecrets(mock.Anything, []string{"witch_hunt"}).Return(nil)
			},
			call: func(repo UserSecretRepository) error {
				return repo.DeleteSecrets(context.Background(), []string{"witch_hunt"})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a write that commits while the cache is unreachable
			client := valkeymock.NewClient(gomock.NewController(t))
			secretDAO := dao.NewMockUserSecretDAO(t)
			repo := NewUserSecretRepo(secretDAO, cache.NewManager(engines.NewValkeyWithClient(client)))
			tc.expect(secretDAO)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(valkeymock.ErrorResult(errors.New("valkey down"))).Times(1)

			// when
			err := tc.call(repo)

			// then the committed write is not reported as failed
			require.NoError(t, err)
		})
	}
}
