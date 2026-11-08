package repository

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/model/spec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newVanityRoleRepo(t *testing.T) (VanityRoleRepository, *dao.MockVanityRoleDAO, *MockPermissionRepository) {
	t.Helper()

	vanityDAO := dao.NewMockVanityRoleDAO(t)
	permissions := NewMockPermissionRepository(t)

	return NewVanityRoleRepo(vanityDAO, permissions), vanityDAO, permissions
}

func TestVanityRoleRepository_DeleteInvalidatesEveryHoldersCachedRoles(t *testing.T) {
	// given a role held by two users, whose assignments cascade away with it
	repo, vanityDAO, permissions := newVanityRoleRepo(t)
	permissions.EXPECT().GetAllVanityRoleAssignments(mock.Anything).Return(map[string][]string{
		"holder-a": {"vanity-a", "vanity-b"},
		"other":    {"vanity-b"},
		"holder-b": {"vanity-a"},
	}, nil).Once()
	vanityDAO.EXPECT().Delete(mock.Anything, "vanity-a").Return(nil).Once()

	var invalidated []string
	permissions.EXPECT().InvalidateVanityAssignments(mock.Anything, mock.Anything).Run(func(_ context.Context, userIDs ...string) {
		invalidated = userIDs
	}).Once()
	permissions.EXPECT().InvalidateVanityRolePermissions(mock.Anything).Once()

	// when
	err := repo.Delete(context.Background(), "vanity-a")

	// then only the holders lose their cached vanity role ids
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"holder-a", "holder-b"}, invalidated)
}

func TestVanityRoleRepository_DeleteAbortsWhenHoldersCannotBeRead(t *testing.T) {
	// given
	repo, _, permissions := newVanityRoleRepo(t)
	readErr := errors.New("db down")
	permissions.EXPECT().GetAllVanityRoleAssignments(mock.Anything).Return(nil, readErr).Once()

	// when
	err := repo.Delete(context.Background(), "vanity-a")

	// then the role is never deleted, so no holder is left with a stale cache
	require.ErrorIs(t, err, readErr)
}

func TestVanityRoleRepository_FailedDeleteKeepsTheCache(t *testing.T) {
	// given
	repo, vanityDAO, permissions := newVanityRoleRepo(t)
	deleteErr := errors.New("db down")
	permissions.EXPECT().GetAllVanityRoleAssignments(mock.Anything).Return(map[string][]string{"holder-a": {"vanity-a"}}, nil).Once()
	vanityDAO.EXPECT().Delete(mock.Anything, "vanity-a").Return(deleteErr).Once()

	// when
	err := repo.Delete(context.Background(), "vanity-a")

	// then
	require.ErrorIs(t, err, deleteErr)
}

func TestVanityRoleRepository_RoleEditsRefreshTheAssignmentsView(t *testing.T) {
	tests := []struct {
		name  string
		write func(repo VanityRoleRepository, vanityDAO *dao.MockVanityRoleDAO) error
	}{
		{
			name: "creating a role",
			write: func(repo VanityRoleRepository, vanityDAO *dao.MockVanityRoleDAO) error {
				s := spec.NewVanityRole{ID: "vanity-a", Label: "A"}
				vanityDAO.EXPECT().Create(mock.Anything, s).Return(nil).Once()

				return repo.Create(context.Background(), s)
			},
		},
		{
			name: "updating a role",
			write: func(repo VanityRoleRepository, vanityDAO *dao.MockVanityRoleDAO) error {
				s := spec.VanityRoleUpdate{ID: "vanity-a", Label: "B"}
				vanityDAO.EXPECT().Update(mock.Anything, s).Return(nil).Once()

				return repo.Update(context.Background(), s)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, vanityDAO, permissions := newVanityRoleRepo(t)
			permissions.EXPECT().InvalidateVanityAssignments(mock.Anything).Once()

			// when
			err := tc.write(repo, vanityDAO)

			// then
			require.NoError(t, err)
		})
	}
}
