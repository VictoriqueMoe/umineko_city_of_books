package settings

import (
	"context"
	"errors"
	"sync"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordedChange struct {
	key   config.SiteSettingKey
	value string
}

type fakeListener struct {
	mu      sync.Mutex
	changes []recordedChange
}

func (f *fakeListener) OnSettingChanged(key config.SiteSettingKey, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.changes = append(f.changes, recordedChange{key: key, value: value})
}

func (f *fakeListener) snapshot() []recordedChange {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]recordedChange, len(f.changes))
	copy(out, f.changes)
	return out
}

type fakeBatchListener struct {
	mu          sync.Mutex
	changes     []recordedChange
	batches     [][]config.SiteSettingKey
	batchCalled int
}

func (f *fakeBatchListener) OnSettingChanged(key config.SiteSettingKey, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.changes = append(f.changes, recordedChange{key: key, value: value})
}

func (f *fakeBatchListener) OnSettingsBatchChanged(keys []config.SiteSettingKey) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.batchCalled++
	cp := make([]config.SiteSettingKey, len(keys))
	copy(cp, keys)
	f.batches = append(f.batches, cp)
}

func (f *fakeBatchListener) batchCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.batchCalled
}

func (f *fakeBatchListener) lastBatch() []config.SiteSettingKey {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.batches) == 0 {
		return nil
	}
	return f.batches[len(f.batches)-1]
}

func newTestService(t *testing.T) (*service, *repository.MockSettingsRepository) {
	repo := repository.NewMockSettingsRepository(t)
	svc := NewService(repo).(*service)
	return svc, repo
}

func primeValidCache(svc *service) {
	svc.cache.Store(config.SettingMaxBodySize.Key, "104857600")
	svc.cache.Store(config.SettingMaxImageSize.Key, "10485760")
	svc.cache.Store(config.SettingMaxVideoSize.Key, "52428800")
	svc.cache.Store(config.SettingMaxGeneralSize.Key, "52428800")
	svc.cache.Store(config.SettingMinPasswordLength.Key, "8")
	svc.cache.Store(config.SettingSessionDurationDays.Key, "30")
	svc.cache.Store(config.SettingMaxTheoriesPerDay.Key, "0")
	svc.cache.Store(config.SettingMaxResponsesPerDay.Key, "0")
	svc.cache.Store(config.SettingRegistrationType.Key, "open")
}

func validBaseSettings() map[string]string {
	out := make(map[string]string)
	for _, def := range config.AllSiteSettings {
		out[string(def.Key)] = def.Default
	}
	return out
}

func TestGet_ReturnsDefaultWhenCacheEmpty(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	got := svc.Get(context.Background(), config.SettingSiteName)

	// then
	assert.Equal(t, config.SettingSiteName.Default, got)
}

func TestGet_ReturnsCachedValue(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	svc.cache.Store(config.SettingSiteName.Key, "Cached Name")

	// when
	got := svc.Get(context.Background(), config.SettingSiteName)

	// then
	assert.Equal(t, "Cached Name", got)
}

func TestGetInt_ParsesCachedValue(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	svc.cache.Store(config.SettingMaxBodySize.Key, "4096")

	// when
	got := svc.GetInt(context.Background(), config.SettingMaxBodySize)

	// then
	assert.Equal(t, 4096, got)
}

func TestGetInt_ReturnsZeroOnParseFailure(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	svc.cache.Store(config.SettingMaxBodySize.Key, "not-a-number")

	// when
	got := svc.GetInt(context.Background(), config.SettingMaxBodySize)

	// then
	assert.Equal(t, 0, got)
}

func TestGetInt_UsesDefaultWhenMissing(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	got := svc.GetInt(context.Background(), config.SettingMaxBodySize)

	// then
	assert.Equal(t, 52428800, got)
}

func TestGetBool_TrueWhenValueIsTrue(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	svc.cache.Store(config.SettingMaintenanceMode.Key, "true")

	// when
	got := svc.GetBool(context.Background(), config.SettingMaintenanceMode)

	// then
	assert.True(t, got)
}

func TestGetBool_FalseForOtherValues(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"literal false", "false"},
		{"empty string", ""},
		{"garbage", "yes"},
		{"capitalised true", "True"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _ := newTestService(t)
			svc.cache.Store(config.SettingMaintenanceMode.Key, tc.value)

			// when
			got := svc.GetBool(context.Background(), config.SettingMaintenanceMode)

			// then
			assert.False(t, got)
		})
	}
}

func TestGetAll_ReturnsDefaultsWhenCacheEmpty(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	got := svc.GetAll(context.Background())

	// then
	assert.Len(t, got, len(config.AllSiteSettings))
	for _, def := range config.AllSiteSettings {
		assert.Equal(t, def.Default, got[def.Key])
	}
}

func TestGetAll_OverlaysCachedValues(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	svc.cache.Store(config.SettingSiteName.Key, "Overlay")
	svc.cache.Store(config.SettingMaintenanceMode.Key, "true")

	// when
	got := svc.GetAll(context.Background())

	// then
	assert.Equal(t, "Overlay", string(got[config.SettingSiteName.Key]))
	assert.Equal(t, "true", got[config.SettingMaintenanceMode.Key])
	assert.Equal(t, config.SettingBaseURL.Default, got[config.SettingBaseURL.Key])
}

func TestRefresh_RepoGetAllError(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().GetAll(mock.Anything).Return(nil, errors.New("db down"))

	// when
	err := svc.Refresh(context.Background())

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "db down")
}

func TestRefresh_SeedsMissingDefaults(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	existing := map[string]string{}
	for _, def := range config.AllSiteSettings {
		existing[string(def.Key)] = def.Default
	}
	delete(existing, string(config.SettingSiteName.Key))
	delete(existing, string(config.SettingBaseURL.Key))

	repo.EXPECT().GetAll(mock.Anything).Return(existing, nil)
	repo.EXPECT().SetMultiple(mock.Anything, mock.MatchedBy(func(m map[string]string) bool {
		if len(m) != 2 {
			return false
		}
		_, okName := m[string(config.SettingSiteName.Key)]
		_, okURL := m[string(config.SettingBaseURL.Key)]
		return okName && okURL
	}), uuid.Nil).Return(nil)

	// when
	err := svc.Refresh(context.Background())

	// then
	require.NoError(t, err)
	v, ok := svc.cache.Load(config.SettingSiteName.Key)
	require.True(t, ok)
	assert.Equal(t, config.SettingSiteName.Default, v.(string))
}

func TestRefresh_SeedErrorBubbles(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().GetAll(mock.Anything).Return(map[string]string{}, nil)
	repo.EXPECT().SetMultiple(mock.Anything, mock.Anything, uuid.Nil).Return(errors.New("seed failed"))

	// when
	err := svc.Refresh(context.Background())

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "seed failed")
}

func TestRefresh_DeletesStaleKeys(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	existing := validBaseSettings()
	existing["stale_key_1"] = "old"
	existing["stale_key_2"] = "older"

	repo.EXPECT().GetAll(mock.Anything).Return(existing, nil)
	repo.EXPECT().Delete(mock.Anything, "stale_key_1").Return(nil)
	repo.EXPECT().Delete(mock.Anything, "stale_key_2").Return(nil)

	// when
	err := svc.Refresh(context.Background())

	// then
	require.NoError(t, err)
	_, ok := svc.cache.Load(config.SiteSettingKey("stale_key_1"))
	assert.False(t, ok)
}

func TestRefresh_DeleteErrorLoggedNotFatal(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	existing := validBaseSettings()
	existing["stale"] = "v"

	repo.EXPECT().GetAll(mock.Anything).Return(existing, nil)
	repo.EXPECT().Delete(mock.Anything, "stale").Return(errors.New("delete failed"))

	// when
	err := svc.Refresh(context.Background())

	// then
	require.NoError(t, err)
}

func TestRefresh_PopulatesCacheFromRepo(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	existing := validBaseSettings()
	existing[string(config.SettingSiteName.Key)] = "Loaded Site"
	existing[string(config.SettingMaintenanceMode.Key)] = "true"

	repo.EXPECT().GetAll(mock.Anything).Return(existing, nil)

	// when
	err := svc.Refresh(context.Background())

	// then
	require.NoError(t, err)
	v, ok := svc.cache.Load(config.SettingSiteName.Key)
	require.True(t, ok)
	assert.Equal(t, "Loaded Site", v.(string))
	assert.True(t, svc.GetBool(context.Background(), config.SettingMaintenanceMode))
}

func TestSet_HappyPathUpdatesCacheAndNotifies(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(svc)
	listener := &fakeListener{}
	svc.Subscribe(listener)
	updatedBy := uuid.New()

	repo.EXPECT().Set(mock.Anything, string(config.SettingSiteName.Key), "New Name", updatedBy).Return(nil)

	// when
	err := svc.Set(context.Background(), config.SettingSiteName, "New Name", updatedBy)

	// then
	require.NoError(t, err)
	v, ok := svc.cache.Load(config.SettingSiteName.Key)
	require.True(t, ok)
	assert.Equal(t, "New Name", v.(string))
	changes := listener.snapshot()
	require.Len(t, changes, 1)
	assert.Equal(t, config.SettingSiteName.Key, changes[0].key)
	assert.Equal(t, "New Name", changes[0].value)
}

func TestSet_ValidationFailureSkipsRepo(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	primeValidCache(svc)
	updatedBy := uuid.New()

	// when
	err := svc.Set(context.Background(), config.SettingRegistrationType, "bogus", updatedBy)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registration type")
}

func TestSet_RepoErrorBubblesAndSkipsCache(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(svc)
	listener := &fakeListener{}
	svc.Subscribe(listener)
	updatedBy := uuid.New()

	repo.EXPECT().Set(mock.Anything, string(config.SettingSiteName.Key), "Attempt", updatedBy).Return(errors.New("db down"))

	// when
	err := svc.Set(context.Background(), config.SettingSiteName, "Attempt", updatedBy)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "db down")
	_, ok := svc.cache.Load(config.SettingSiteName.Key)
	assert.False(t, ok)
	assert.Empty(t, listener.snapshot())
}

func TestSetMultiple_HappyPathNotifiesEachAndBatch(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(svc)
	listener := &fakeBatchListener{}
	svc.Subscribe(listener)
	updatedBy := uuid.New()

	values := map[config.SiteSettingKey]string{
		config.SettingSiteName.Key:        "Multi Name",
		config.SettingMaintenanceMode.Key: "true",
	}

	repo.EXPECT().SetMultiple(mock.Anything, mock.MatchedBy(func(m map[string]string) bool {
		return m[string(config.SettingSiteName.Key)] == "Multi Name" &&
			m[string(config.SettingMaintenanceMode.Key)] == "true" &&
			len(m) == 2
	}), updatedBy).Return(nil)

	// when
	err := svc.SetMultiple(context.Background(), values, updatedBy)

	// then
	require.NoError(t, err)
	v1, _ := svc.cache.Load(config.SettingSiteName.Key)
	v2, _ := svc.cache.Load(config.SettingMaintenanceMode.Key)
	assert.Equal(t, "Multi Name", v1.(string))
	assert.Equal(t, "true", v2.(string))

	listener.mu.Lock()
	gotChanges := len(listener.changes)
	listener.mu.Unlock()
	assert.Equal(t, 2, gotChanges)

	assert.Equal(t, 1, listener.batchCount())
	last := listener.lastBatch()
	assert.ElementsMatch(t, []config.SiteSettingKey{
		config.SettingSiteName.Key,
		config.SettingMaintenanceMode.Key,
	}, last)
}

func TestSetMultiple_UnknownKeyRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	primeValidCache(svc)
	updatedBy := uuid.New()
	values := map[config.SiteSettingKey]string{
		config.SiteSettingKey("not_a_real_key"): "v",
	}

	// when
	err := svc.SetMultiple(context.Background(), values, updatedBy)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown setting")
}

func TestSetMultiple_ValidationFailureSkipsRepo(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	primeValidCache(svc)
	updatedBy := uuid.New()
	values := map[config.SiteSettingKey]string{
		config.SettingMaxBodySize.Key: "0",
	}

	// when
	err := svc.SetMultiple(context.Background(), values, updatedBy)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max body size")
}

func TestSetMultiple_RepoErrorBubbles(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(svc)
	listener := &fakeBatchListener{}
	svc.Subscribe(listener)
	updatedBy := uuid.New()
	values := map[config.SiteSettingKey]string{
		config.SettingSiteName.Key: "X",
	}

	repo.EXPECT().SetMultiple(mock.Anything, mock.Anything, updatedBy).Return(errors.New("boom"))

	// when
	err := svc.SetMultiple(context.Background(), values, updatedBy)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "boom")
	_, ok := svc.cache.Load(config.SettingSiteName.Key)
	assert.False(t, ok)
	assert.Equal(t, 0, listener.batchCount())
}

func TestSubscribe_MultipleListenersAllNotified(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(svc)
	l1 := &fakeListener{}
	l2 := &fakeListener{}
	svc.Subscribe(l1)
	svc.Subscribe(l2)
	updatedBy := uuid.New()

	repo.EXPECT().Set(mock.Anything, string(config.SettingSiteName.Key), "Hello", updatedBy).Return(nil)

	// when
	err := svc.Set(context.Background(), config.SettingSiteName, "Hello", updatedBy)

	// then
	require.NoError(t, err)
	assert.Len(t, l1.snapshot(), 1)
	assert.Len(t, l2.snapshot(), 1)
}

func TestSubscribe_NonBatchListenerDoesNotReceiveBatch(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(svc)
	plain := &fakeListener{}
	batch := &fakeBatchListener{}
	svc.Subscribe(plain)
	svc.Subscribe(batch)
	updatedBy := uuid.New()

	values := map[config.SiteSettingKey]string{
		config.SettingSiteName.Key: "Batched",
	}

	repo.EXPECT().SetMultiple(mock.Anything, mock.Anything, updatedBy).Return(nil)

	// when
	err := svc.SetMultiple(context.Background(), values, updatedBy)

	// then
	require.NoError(t, err)
	assert.Len(t, plain.snapshot(), 1)
	assert.Equal(t, 1, batch.batchCount())
}
