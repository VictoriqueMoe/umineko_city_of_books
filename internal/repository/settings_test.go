package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsRepository_SetAndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.Settings.Set(context.Background(), "site_name", "Umineko", user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.Settings.Get(context.Background(), "site_name")
	require.NoError(t, err)
	assert.Equal(t, "Umineko", got)
}

func TestSettingsRepository_Set_WithNilUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Settings.Set(context.Background(), "anon_key", "anon_value", uuid.Nil)

	// then
	require.NoError(t, err)
	got, err := repos.Settings.Get(context.Background(), "anon_key")
	require.NoError(t, err)
	assert.Equal(t, "anon_value", got)
}

func TestSettingsRepository_Set_Upsert(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Settings.Set(context.Background(), "theme", "light", user.ID))

	// when
	err := repos.Settings.Set(context.Background(), "theme", "dark", user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.Settings.Get(context.Background(), "theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", got)
}

func TestSettingsRepository_Get_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Settings.Get(context.Background(), "missing_key")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing_key")
}

func TestSettingsRepository_GetAll_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.Settings.GetAll(context.Background())

	// then
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSettingsRepository_GetAll(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Settings.Set(context.Background(), "alpha", "1", user.ID))
	require.NoError(t, repos.Settings.Set(context.Background(), "beta", "2", user.ID))
	require.NoError(t, repos.Settings.Set(context.Background(), "gamma", "3", uuid.Nil))

	// when
	got, err := repos.Settings.GetAll(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"alpha": "1",
		"beta":  "2",
		"gamma": "3",
	}, got)
}

func TestSettingsRepository_SetMultiple(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	settings := map[string]string{
		"colour":   "blue",
		"language": "en-GB",
		"timezone": "UTC",
	}

	// when
	err := repos.Settings.SetMultiple(context.Background(), settings, user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.Settings.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, settings, got)
}

func TestSettingsRepository_SetMultiple_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.Settings.SetMultiple(context.Background(), map[string]string{}, user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.Settings.GetAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSettingsRepository_SetMultiple_Upsert(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Settings.Set(context.Background(), "colour", "red", user.ID))
	require.NoError(t, repos.Settings.Set(context.Background(), "extra", "keep", user.ID))

	// when
	err := repos.Settings.SetMultiple(context.Background(), map[string]string{
		"colour": "green",
		"size":   "large",
	}, user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.Settings.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"colour": "green",
		"extra":  "keep",
		"size":   "large",
	}, got)
}

func TestSettingsRepository_SetMultiple_NilUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Settings.SetMultiple(context.Background(), map[string]string{
		"a": "1",
		"b": "2",
	}, uuid.Nil)

	// then
	require.NoError(t, err)
	got, err := repos.Settings.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, got)
}

func TestSettingsRepository_Delete(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Settings.Set(context.Background(), "to_delete", "value", user.ID))
	require.NoError(t, repos.Settings.Set(context.Background(), "to_keep", "value", user.ID))

	// when
	err := repos.Settings.Delete(context.Background(), "to_delete")

	// then
	require.NoError(t, err)
	_, getErr := repos.Settings.Get(context.Background(), "to_delete")
	assert.Error(t, getErr)
	kept, err := repos.Settings.Get(context.Background(), "to_keep")
	require.NoError(t, err)
	assert.Equal(t, "value", kept)
}

func TestSettingsRepository_Delete_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Settings.Delete(context.Background(), "never_existed")

	// then
	assert.NoError(t, err)
}
