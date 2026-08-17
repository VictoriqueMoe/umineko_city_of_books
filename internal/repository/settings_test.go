package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

const (
	settingSiteName config.SiteSettingKey = "site_name"
	settingLogLevel config.SiteSettingKey = "log_level"
	settingStaleKey config.SiteSettingKey = "stale_key"
	settingDeadKey  config.SiteSettingKey = "dead_key"
)

func newCachedSettingsRepo(t *testing.T) (SettingsRepository, *MockSettingsDAO, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	dao := NewMockSettingsDAO(t)

	return NewSettingsRepo(nil, dao, cache.NewManager(engines.NewValkeyWithClient(client))), dao, client
}

func settingCacheKey(key config.SiteSettingKey) string {
	return cache.Setting.Key(string(key))
}

func captureSettingsCommands(client *valkeymock.Client, times int, reply func(cmd valkey.Completed) valkey.ValkeyResult) *[][]string {
	commands := new([][]string)

	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			*commands = append(*commands, cmd.Commands())

			return reply(cmd)
		}).
		Times(times)

	return commands
}

func settingsMissThenStore(cmd valkey.Completed) valkey.ValkeyResult {
	if cmd.Commands()[0] == "GET" {
		return valkeymock.ErrorResult(valkey.Nil)
	}

	return valkeymock.Result(valkeymock.ValkeyString("OK"))
}

func TestSettingsGet_CachesTheDaoResult(t *testing.T) {
	// given a cold cache
	repo, dao, client := newCachedSettingsRepo(t)
	dao.EXPECT().Get(mock.Anything, settingSiteName).Return("When They Cry", nil)
	commands := captureSettingsCommands(client, 2, settingsMissThenStore)

	// when
	got, err := repo.Get(context.Background(), settingSiteName)

	// then the miss falls through to the dao and the value is written back
	require.NoError(t, err)
	assert.Equal(t, "When They Cry", got)
	require.Len(t, *commands, 2)
	assert.Equal(t, []string{"GET", settingCacheKey(settingSiteName)}, (*commands)[0])
	assert.Equal(t, "SET", (*commands)[1][0])
	assert.Equal(t, settingCacheKey(settingSiteName), (*commands)[1][1])
}

func TestSettingsGet_ServesTheCacheWithoutTouchingTheDao(t *testing.T) {
	// given a warm cache, and a dao mock with no expectations so any call fails the test
	repo, _, client := newCachedSettingsRepo(t)
	captureSettingsCommands(client, 1, func(_ valkey.Completed) valkey.ValkeyResult {
		return valkeymock.Result(valkeymock.ValkeyString("Cached Name"))
	})

	// when
	got, err := repo.Get(context.Background(), settingSiteName)

	// then
	require.NoError(t, err)
	assert.Equal(t, "Cached Name", got)
}

func TestSettingsGet_DaoErrorSkipsTheCacheWrite(t *testing.T) {
	// given
	repo, dao, client := newCachedSettingsRepo(t)
	dao.EXPECT().Get(mock.Anything, settingSiteName).Return("", errors.New("db down"))
	commands := captureSettingsCommands(client, 1, settingsMissThenStore)

	// when
	_, err := repo.Get(context.Background(), settingSiteName)

	// then a failed read must never be cached
	require.Error(t, err)
	require.Len(t, *commands, 1)
	assert.Equal(t, "GET", (*commands)[0][0])
}

func TestSettingsSet_InvalidatesTheKey(t *testing.T) {
	// given
	repo, dao, client := newCachedSettingsRepo(t)
	updatedBy := uuid.New()
	dao.EXPECT().Set(mock.Anything, settingSiteName, "Rokkenjima", updatedBy).Return(nil)
	commands := captureSettingsCommands(client, 1, func(_ valkey.Completed) valkey.ValkeyResult {
		return valkeymock.Result(valkeymock.ValkeyInt64(1))
	})

	// when
	err := repo.Set(context.Background(), settingSiteName, "Rokkenjima", updatedBy)

	// then
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"DEL", settingCacheKey(settingSiteName)}}, *commands)
}

func TestSettingsSet_DaoErrorSkipsInvalidation(t *testing.T) {
	// given
	repo, dao, client := newCachedSettingsRepo(t)
	dao.EXPECT().Set(mock.Anything, settingSiteName, "Rokkenjima", uuid.Nil).Return(errors.New("db down"))
	client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	// when
	err := repo.Set(context.Background(), settingSiteName, "Rokkenjima", uuid.Nil)

	// then a failed write must leave the cached value alone
	require.Error(t, err)
}

func TestSettingsSetMultiple_InvalidatesEveryKey(t *testing.T) {
	// given
	repo, dao, client := newCachedSettingsRepo(t)
	values := map[config.SiteSettingKey]string{settingSiteName: "Rokkenjima", settingLogLevel: "debug"}
	dao.EXPECT().SetMultiple(mock.Anything, values, uuid.Nil).Return(nil)
	commands := captureSettingsCommands(client, 1, func(_ valkey.Completed) valkey.ValkeyResult {
		return valkeymock.Result(valkeymock.ValkeyInt64(2))
	})

	// when
	err := repo.SetMultiple(context.Background(), values, uuid.Nil)

	// then every written key is dropped in one call, in whatever order the map yielded
	require.NoError(t, err)
	require.Len(t, *commands, 1)
	assert.Equal(t, "DEL", (*commands)[0][0])
	assert.ElementsMatch(t, []string{settingCacheKey(settingSiteName), settingCacheKey(settingLogLevel)}, (*commands)[0][1:])
}

func TestSettingsDelete_InvalidatesTheKey(t *testing.T) {
	// given
	repo, dao, client := newCachedSettingsRepo(t)
	dao.EXPECT().Delete(mock.Anything, settingStaleKey).Return(nil)
	commands := captureSettingsCommands(client, 1, func(_ valkey.Completed) valkey.ValkeyResult {
		return valkeymock.Result(valkeymock.ValkeyInt64(1))
	})

	// when
	err := repo.Delete(context.Background(), settingStaleKey)

	// then
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"DEL", settingCacheKey(settingStaleKey)}}, *commands)
}

func TestSettingsReconcile_InvalidatesSeededAndStaleKeys(t *testing.T) {
	// given a caller-supplied transaction, so the repo joins it instead of opening one
	repo, dao, client := newCachedSettingsRepo(t)
	tx := new(sql.Tx)
	missing := map[config.SiteSettingKey]string{settingSiteName: "When They Cry"}
	dao.EXPECT().SetMultiple(mock.Anything, missing, uuid.Nil, []*sql.Tx{tx}).Return(nil)
	dao.EXPECT().Delete(mock.Anything, settingDeadKey, []*sql.Tx{tx}).Return(nil)
	commands := captureSettingsCommands(client, 1, func(_ valkey.Completed) valkey.ValkeyResult {
		return valkeymock.Result(valkeymock.ValkeyInt64(2))
	})

	// when
	err := repo.Reconcile(context.Background(), SettingsReconcile{Missing: missing, Stale: []config.SiteSettingKey{settingDeadKey}}, tx)

	// then both the seeded default and the removed key are dropped from the cache
	require.NoError(t, err)
	require.Len(t, *commands, 1)
	assert.Equal(t, "DEL", (*commands)[0][0])
	assert.ElementsMatch(t, []string{settingCacheKey(settingSiteName), settingCacheKey(settingDeadKey)}, (*commands)[0][1:])
}

func TestSettingsReconcile_NothingToDoSkipsInvalidation(t *testing.T) {
	// given a reconcile that finds the stored settings already correct
	repo, _, client := newCachedSettingsRepo(t)
	client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	// when
	err := repo.Reconcile(context.Background(), SettingsReconcile{}, new(sql.Tx))

	// then
	require.NoError(t, err)
}
