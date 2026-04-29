package usersecret_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/secrets"
	"umineko_city_of_books/internal/usersecret"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testParentID = secrets.ID("test-unlock-parent")
	testChildID  = secrets.ID("test-unlock-child")
)

func registerTestSpecs(t *testing.T) (parentPhrase, childPhrase string) {
	t.Helper()
	parentPhrase = "the parent phrase"
	childPhrase = "the child phrase"
	parentSum := sha256.Sum256([]byte(parentPhrase))
	childSum := sha256.Sum256([]byte(childPhrase))
	secrets.Register(
		secrets.Spec{
			ID:           testParentID,
			ExpectedHash: hex.EncodeToString(parentSum[:]),
			Title:        "Test Hunt",
			Pieces: []secrets.Piece{
				{ID: testChildID, Letter: "X", Tile: 1},
			},
		},
		secrets.Spec{
			ID:           testChildID,
			ExpectedHash: hex.EncodeToString(childSum[:]),
			ParentID:     testParentID,
		},
	)
	return parentPhrase, childPhrase
}

func TestService_Unlock_UnknownSecret(t *testing.T) {
	// given
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)

	// when
	_, err := svc.Unlock(context.Background(), uuid.New(), "nonexistent", "phrase")

	// then
	assert.ErrorIs(t, err, usersecret.ErrInvalidRequest)
}

func TestService_Unlock_WrongPhrase(t *testing.T) {
	// given
	registerTestSpecs(t)
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)

	// when
	_, err := svc.Unlock(context.Background(), uuid.New(), string(testChildID), "wrong phrase")

	// then
	assert.ErrorIs(t, err, usersecret.ErrInvalidRequest)
}

func TestService_Unlock_HuntAlreadySolved(t *testing.T) {
	// given
	_, childPhrase := registerTestSpecs(t)
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)
	repo.EXPECT().IsSolvedByAnyone(mock.Anything, string(testParentID)).Return(true, nil)

	// when
	_, err := svc.Unlock(context.Background(), uuid.New(), string(testChildID), childPhrase)

	// then
	assert.ErrorIs(t, err, usersecret.ErrHuntAlreadySolved)
}

func TestService_Unlock_MissingPieces(t *testing.T) {
	// given
	parentPhrase, _ := registerTestSpecs(t)
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)
	userID := uuid.New()
	repo.EXPECT().IsSolvedByAnyone(mock.Anything, string(testParentID)).Return(false, nil)
	repo.EXPECT().ListForUser(mock.Anything, userID).Return([]string{}, nil)

	// when
	_, err := svc.Unlock(context.Background(), userID, string(testParentID), parentPhrase)

	// then
	assert.ErrorIs(t, err, usersecret.ErrInvalidRequest)
}

func TestService_Unlock_PieceSucceeds(t *testing.T) {
	// given
	_, childPhrase := registerTestSpecs(t)
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)
	userID := uuid.New()
	repo.EXPECT().IsSolvedByAnyone(mock.Anything, string(testParentID)).Return(false, nil)
	repo.EXPECT().Unlock(mock.Anything, userID, string(testChildID)).Return(nil)

	// when
	result, err := svc.Unlock(context.Background(), userID, string(testChildID), childPhrase)

	// then
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.HasParent)
	assert.False(t, result.IsParent)
	require.NotNil(t, result.Parent)
	assert.Equal(t, testParentID, result.Parent.ID)
}

func TestService_Unlock_ParentSucceedsWithAllPieces(t *testing.T) {
	// given
	parentPhrase, _ := registerTestSpecs(t)
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)
	userID := uuid.New()
	repo.EXPECT().IsSolvedByAnyone(mock.Anything, string(testParentID)).Return(false, nil)
	repo.EXPECT().ListForUser(mock.Anything, userID).Return([]string{string(testChildID)}, nil)
	repo.EXPECT().Unlock(mock.Anything, userID, string(testParentID)).Return(nil)

	// when
	result, err := svc.Unlock(context.Background(), userID, string(testParentID), parentPhrase)

	// then
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsParent)
}

func TestService_Unlock_DBError(t *testing.T) {
	// given
	_, childPhrase := registerTestSpecs(t)
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)
	userID := uuid.New()
	repo.EXPECT().IsSolvedByAnyone(mock.Anything, string(testParentID)).Return(false, nil)
	repo.EXPECT().Unlock(mock.Anything, userID, string(testChildID)).Return(errors.New("db down"))

	// when
	_, err := svc.Unlock(context.Background(), userID, string(testChildID), childPhrase)

	// then
	assert.Error(t, err)
	assert.NotErrorIs(t, err, usersecret.ErrInvalidRequest)
}

func TestService_ListForUser_Delegates(t *testing.T) {
	// given
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)
	userID := uuid.New()
	repo.EXPECT().ListForUser(mock.Anything, userID).Return([]string{"sec1"}, nil)

	// when
	got, err := svc.ListForUser(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"sec1"}, got)
}

func TestService_IsSolvedByAnyone_Delegates(t *testing.T) {
	// given
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)
	repo.EXPECT().IsSolvedByAnyone(mock.Anything, "sec").Return(true, nil)

	// when
	got, err := svc.IsSolvedByAnyone(context.Background(), "sec")

	// then
	require.NoError(t, err)
	assert.True(t, got)
}

func TestService_GetUserIDsWithSecret_Delegates(t *testing.T) {
	// given
	repo := repository.NewMockUserSecretRepository(t)
	svc := usersecret.NewService(repo)
	uid := uuid.New()
	repo.EXPECT().GetUserIDsWithSecret(mock.Anything, "sec").Return([]uuid.UUID{uid}, nil)

	// when
	got, err := svc.GetUserIDsWithSecret(context.Background(), "sec")

	// then
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{uid}, got)
}
