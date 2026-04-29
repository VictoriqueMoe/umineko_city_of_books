package vanityrole_test

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/vanityrole"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_List_Delegates(t *testing.T) {
	// given
	repo := repository.NewMockVanityRoleRepository(t)
	svc := vanityrole.NewService(repo)
	rows := []repository.VanityRoleRow{{ID: "r1", Label: "VIP"}}
	repo.EXPECT().List(mock.Anything).Return(rows, nil)

	// when
	got, err := svc.List(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, rows, got)
}

func TestService_List_PropagatesError(t *testing.T) {
	// given
	repo := repository.NewMockVanityRoleRepository(t)
	svc := vanityrole.NewService(repo)
	repo.EXPECT().List(mock.Anything).Return(nil, errors.New("boom"))

	// when
	_, err := svc.List(context.Background())

	// then
	assert.Error(t, err)
}

func TestService_GetAllAssignments_Delegates(t *testing.T) {
	// given
	repo := repository.NewMockVanityRoleRepository(t)
	svc := vanityrole.NewService(repo)
	assignments := map[string][]string{"u1": {"r1"}}
	repo.EXPECT().GetAllAssignments(mock.Anything).Return(assignments, nil)

	// when
	got, err := svc.GetAllAssignments(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, assignments, got)
}
