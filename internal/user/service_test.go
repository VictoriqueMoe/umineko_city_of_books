package user

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (
	*service,
	*repository.MockUserRepository,
	*repository.MockRoleRepository,
	*authz.MockService,
) {
	userRepo := repository.NewMockUserRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	authzSvc := authz.NewMockService(t)
	svc := NewService(userRepo, roleRepo, authzSvc).(*service)
	return svc, userRepo, roleRepo, authzSvc
}

func TestCreate_FirstUserAssignsSuperAdmin(t *testing.T) {
	// given
	svc, userRepo, roleRepo, _ := newTestService(t)
	userID := uuid.New()
	created := &model.User{ID: userID, Username: "alice", DisplayName: "Alice"}
	userRepo.EXPECT().Count(mock.Anything).Return(0, nil)
	userRepo.EXPECT().Create(mock.Anything, "alice", "alice@example.com", "pw", "Alice").Return(created, nil)
	roleRepo.EXPECT().SetRole(mock.Anything, userID, authz.RoleSuperAdmin).Return(nil)

	// when
	got, err := svc.Create(context.Background(), "alice", "alice@example.com", "pw", "Alice")

	// then
	require.NoError(t, err)
	assert.Equal(t, userID, got.ID)
	assert.Equal(t, "alice", got.Username)
}

func TestCreate_FirstUserSetRoleErrorSwallowed(t *testing.T) {
	// given
	svc, userRepo, roleRepo, _ := newTestService(t)
	userID := uuid.New()
	created := &model.User{ID: userID, Username: "alice", DisplayName: "Alice"}
	userRepo.EXPECT().Count(mock.Anything).Return(0, nil)
	userRepo.EXPECT().Create(mock.Anything, "alice", "alice@example.com", "pw", "Alice").Return(created, nil)
	roleRepo.EXPECT().SetRole(mock.Anything, userID, authz.RoleSuperAdmin).Return(errors.New("boom"))

	// when
	got, err := svc.Create(context.Background(), "alice", "alice@example.com", "pw", "Alice")

	// then
	require.NoError(t, err)
	assert.Equal(t, userID, got.ID)
}

func TestListStaff_OK_FiltersBannedAndSorts(t *testing.T) {
	// given
	svc, userRepo, roleRepo, _ := newTestService(t)
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{authz.RoleSuperAdmin, authz.RoleAdmin}).Return(ids, nil)
	userRepo.EXPECT().GetByIDs(mock.Anything, ids).Return([]model.User{
		{ID: ids[0], Username: "bob", DisplayName: "Bob", Role: string(authz.RoleAdmin)},
		{ID: ids[1], Username: "zelda", DisplayName: "Zelda", Role: string(authz.RoleSuperAdmin)},
		{ID: ids[2], Username: "anna", DisplayName: "Anna", Role: string(authz.RoleSuperAdmin)},
		{ID: ids[3], Username: "evil", DisplayName: "Evil", Role: string(authz.RoleAdmin), BannedAt: new("2026-01-01")},
	}, nil)

	// when
	staff, err := svc.ListStaff(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, staff, 3)
	assert.Equal(t, "Anna", staff[0].DisplayName)
	assert.Equal(t, "Zelda", staff[1].DisplayName)
	assert.Equal(t, "Bob", staff[2].DisplayName)
}

func TestListStaff_NoStaff(t *testing.T) {
	// given
	svc, _, roleRepo, _ := newTestService(t)
	roleRepo.EXPECT().GetUsersByRoles(mock.Anything, []role.Role{authz.RoleSuperAdmin, authz.RoleAdmin}).Return(nil, nil)

	// when
	staff, err := svc.ListStaff(context.Background())

	// then
	require.NoError(t, err)
	assert.Empty(t, staff)
}

func TestListStaff_RoleRepoError(t *testing.T) {
	// given
	svc, _, roleRepo, _ := newTestService(t)
	roleRepo.EXPECT().GetUsersByRoles(mock.Anything, mock.Anything).Return(nil, errors.New("boom"))

	// when
	_, err := svc.ListStaff(context.Background())

	// then
	require.Error(t, err)
}

func TestListStaff_UserRepoError(t *testing.T) {
	// given
	svc, userRepo, roleRepo, _ := newTestService(t)
	ids := []uuid.UUID{uuid.New()}
	roleRepo.EXPECT().GetUsersByRoles(mock.Anything, mock.Anything).Return(ids, nil)
	userRepo.EXPECT().GetByIDs(mock.Anything, ids).Return(nil, errors.New("boom"))

	// when
	_, err := svc.ListStaff(context.Background())

	// then
	require.Error(t, err)
}

func TestCreate_SubsequentUserNoRoleAssigned(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userID := uuid.New()
	created := &model.User{ID: userID, Username: "bob", DisplayName: "Bob"}
	userRepo.EXPECT().Count(mock.Anything).Return(5, nil)
	userRepo.EXPECT().Create(mock.Anything, "bob", "bob@example.com", "pw", "Bob").Return(created, nil)

	// when
	got, err := svc.Create(context.Background(), "bob", "bob@example.com", "pw", "Bob")

	// then
	require.NoError(t, err)
	assert.Equal(t, "bob", got.Username)
}

func TestCreate_CountErrorBubbles(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userRepo.EXPECT().Count(mock.Anything).Return(0, errors.New("db down"))

	// when
	_, err := svc.Create(context.Background(), "alice", "alice@example.com", "pw", "Alice")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count users")
}

func TestCreate_CreateErrorBubbles(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userRepo.EXPECT().Count(mock.Anything).Return(3, nil)
	userRepo.EXPECT().Create(mock.Anything, "alice", "alice@example.com", "pw", "Alice").Return(nil, errors.New("dup"))

	// when
	_, err := svc.Create(context.Background(), "alice", "alice@example.com", "pw", "Alice")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create user")
}

func TestGetByID_OK(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userID := uuid.New()
	found := &model.User{ID: userID, Username: "alice", DisplayName: "Alice"}
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(found, nil)

	// when
	got, err := svc.GetByID(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, userID, got.ID)
	assert.Equal(t, "alice", got.Username)
}

func TestGetByID_NotFound(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userID := uuid.New()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, nil)

	// when
	_, err := svc.GetByID(context.Background(), userID)

	// then
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestGetByID_RepoError(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userID := uuid.New()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetByID(context.Background(), userID)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get user")
}

func TestValidateCredentials_OK(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userID := uuid.New()
	found := &model.User{ID: userID, Username: "alice", DisplayName: "Alice"}
	userRepo.EXPECT().ValidatePassword(mock.Anything, "alice", "pw").Return(found, nil)

	// when
	got, err := svc.ValidateCredentials(context.Background(), "alice", "pw")

	// then
	require.NoError(t, err)
	assert.Equal(t, userID, got.ID)
}

func TestValidateCredentials_InvalidReturnsErr(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userRepo.EXPECT().ValidatePassword(mock.Anything, "alice", "wrong").Return(nil, nil)

	// when
	_, err := svc.ValidateCredentials(context.Background(), "alice", "wrong")

	// then
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestValidateCredentials_RepoError(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userRepo.EXPECT().ValidatePassword(mock.Anything, "alice", "pw").Return(nil, errors.New("boom"))

	// when
	_, err := svc.ValidateCredentials(context.Background(), "alice", "pw")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate credentials")
}

func TestCheckUsernameAvailable_Available(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userRepo.EXPECT().ExistsByUsername(mock.Anything, "alice").Return(false, nil)

	// when
	err := svc.CheckUsernameAvailable(context.Background(), "alice")

	// then
	require.NoError(t, err)
}

func TestCheckUsernameAvailable_Taken(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userRepo.EXPECT().ExistsByUsername(mock.Anything, "alice").Return(true, nil)

	// when
	err := svc.CheckUsernameAvailable(context.Background(), "alice")

	// then
	require.ErrorIs(t, err, ErrUsernameTaken)
}

func TestCheckUsernameAvailable_RepoError(t *testing.T) {
	// given
	svc, userRepo, _, _ := newTestService(t)
	userRepo.EXPECT().ExistsByUsername(mock.Anything, "alice").Return(false, errors.New("boom"))

	// when
	err := svc.CheckUsernameAvailable(context.Background(), "alice")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check username")
}
