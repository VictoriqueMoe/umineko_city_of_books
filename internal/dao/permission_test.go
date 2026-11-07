package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dao/daotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionDAO_MigratedModeratorRowsMatchTheDefaults(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	want := make([]string, 0)
	for _, perm := range authz.DefaultRolePermissions(authz.RoleModerator) {
		want = append(want, string(perm))
	}

	// when
	got, err := repos.Permission.GetRolePermissions(context.Background())

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, want, got[string(authz.RoleModerator)])
}
