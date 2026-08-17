package settings

import (
	"context"
	"errors"
	"slices"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*service, *repository.MockSettingsRepository) {
	repo := repository.NewMockSettingsRepository(t)
	svc := NewService(repo).(*service)
	return svc, repo
}

func batchOfKeys(want ...config.SiteSettingKey) any {
	return mock.MatchedBy(func(got []config.SiteSettingKey) bool {
		if len(got) != len(want) {
			return false
		}

		for _, key := range want {
			if !slices.Contains(got, key) {
				return false
			}
		}

		return true
	})
}

func primeValidCache(repo *repository.MockSettingsRepository) {
	m := validBaseSettings()
	m[config.SettingMaxBodySize.Key] = "104857600"
	m[config.SettingMaxImageSize.Key] = "10485760"
	m[config.SettingMaxVideoSize.Key] = "52428800"
	m[config.SettingMaxGeneralSize.Key] = "52428800"
	m[config.SettingMinPasswordLength.Key] = "8"
	m[config.SettingSessionDurationDays.Key] = "30"
	m[config.SettingMaxTheoriesPerDay.Key] = "0"
	m[config.SettingMaxResponsesPerDay.Key] = "0"
	m[config.SettingRegistrationType.Key] = "open"
	repo.EXPECT().GetAll(mock.Anything).Return(m, nil).Maybe()
}

func validBaseSettings() map[config.SiteSettingKey]string {
	out := make(map[config.SiteSettingKey]string)
	for _, def := range config.AllSiteSettings {
		out[def.Key] = def.Default
	}
	return out
}

func TestGet_ReturnsDefaultWhenCacheEmpty(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().Get(mock.Anything, config.SettingSiteName.Key).Return("", errors.New("not found"))

	// when
	got := svc.Get(context.Background(), config.SettingSiteName)

	// then
	assert.Equal(t, config.SettingSiteName.Default, got)
}

func TestGet_ReturnsCachedValue(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().Get(mock.Anything, config.SettingSiteName.Key).Return("Cached Name", nil)

	// when
	got := svc.Get(context.Background(), config.SettingSiteName)

	// then
	assert.Equal(t, "Cached Name", got)
}

func TestGetInt_ParsesCachedValue(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().Get(mock.Anything, config.SettingMaxBodySize.Key).Return("4096", nil)

	// when
	got := svc.GetInt(context.Background(), config.SettingMaxBodySize)

	// then
	assert.Equal(t, 4096, got)
}

func TestGetInt_ReturnsZeroOnParseFailure(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().Get(mock.Anything, config.SettingMaxBodySize.Key).Return("not-a-number", nil)

	// when
	got := svc.GetInt(context.Background(), config.SettingMaxBodySize)

	// then
	assert.Equal(t, 0, got)
}

func TestGetInt_UsesDefaultWhenMissing(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().Get(mock.Anything, config.SettingMaxBodySize.Key).Return("", errors.New("not found"))

	// when
	got := svc.GetInt(context.Background(), config.SettingMaxBodySize)

	// then
	assert.Equal(t, 52428800, got)
}

func TestGetBool_TrueWhenValueIsTrue(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().Get(mock.Anything, config.SettingMaintenanceMode.Key).Return("true", nil)

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
			svc, repo := newTestService(t)
			repo.EXPECT().Get(mock.Anything, config.SettingMaintenanceMode.Key).Return(tc.value, nil)

			// when
			got := svc.GetBool(context.Background(), config.SettingMaintenanceMode)

			// then
			assert.False(t, got)
		})
	}
}

func TestGetAll_ReturnsDefaultsWhenCacheEmpty(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{}, nil)

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
	svc, repo := newTestService(t)
	repo.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{
		config.SettingSiteName.Key:        "Overlay",
		config.SettingMaintenanceMode.Key: "true",
	}, nil)

	// when
	got := svc.GetAll(context.Background())

	// then
	assert.Equal(t, "Overlay", got[config.SettingSiteName.Key])
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
	existing := map[config.SiteSettingKey]string{}
	for _, def := range config.AllSiteSettings {
		existing[def.Key] = def.Default
	}
	delete(existing, config.SettingSiteName.Key)
	delete(existing, config.SettingBaseURL.Key)

	repo.EXPECT().GetAll(mock.Anything).Return(existing, nil)
	repo.EXPECT().Reconcile(mock.Anything, mock.MatchedBy(func(spec repository.SettingsReconcile) bool {
		if len(spec.Missing) != 2 || len(spec.Stale) != 0 || spec.UpdatedBy != uuid.Nil {
			return false
		}
		_, okName := spec.Missing[config.SettingSiteName.Key]
		_, okURL := spec.Missing[config.SettingBaseURL.Key]
		return okName && okURL
	})).Return(nil)

	// when
	err := svc.Refresh(context.Background())

	// then
	require.NoError(t, err)
}

func TestRefresh_SeedErrorBubbles(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	repo.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{}, nil)
	repo.EXPECT().Reconcile(mock.Anything, mock.Anything).Return(errors.New("seed failed"))

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
	repo.EXPECT().Reconcile(mock.Anything, mock.MatchedBy(func(spec repository.SettingsReconcile) bool {
		return len(spec.Missing) == 0 &&
			slices.Contains(spec.Stale, "stale_key_1") &&
			slices.Contains(spec.Stale, "stale_key_2") &&
			len(spec.Stale) == 2
	})).Return(nil)

	// when
	err := svc.Refresh(context.Background())

	// then
	require.NoError(t, err)
}

func TestRefresh_StaleDeleteErrorBubbles(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	existing := validBaseSettings()
	existing["stale"] = "v"

	repo.EXPECT().GetAll(mock.Anything).Return(existing, nil)
	repo.EXPECT().Reconcile(mock.Anything, mock.Anything).Return(errors.New("delete failed"))

	// when
	err := svc.Refresh(context.Background())

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "delete failed")
}

func TestRefresh_PopulatesCacheFromRepo(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	existing := validBaseSettings()
	existing[config.SettingSiteName.Key] = "Loaded Site"
	existing[config.SettingMaintenanceMode.Key] = "true"

	repo.EXPECT().GetAll(mock.Anything).Return(existing, nil)
	repo.EXPECT().Get(mock.Anything, config.SettingMaintenanceMode.Key).Return("true", nil)

	// when
	err := svc.Refresh(context.Background())

	// then
	require.NoError(t, err)
	assert.True(t, svc.GetBool(context.Background(), config.SettingMaintenanceMode))
}

func TestSet_HappyPathUpdatesCacheAndNotifies(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(repo)
	listener := NewMockListener(t)
	listener.EXPECT().OnSettingChanged(config.SettingSiteName.Key, "New Name").Once()
	svc.Subscribe(listener)
	updatedBy := uuid.New()

	repo.EXPECT().Set(mock.Anything, config.SettingSiteName.Key, "New Name", updatedBy).Return(nil)

	// when
	err := svc.Set(context.Background(), config.SettingSiteName, "New Name", updatedBy)

	// then
	require.NoError(t, err)
	listener.AssertExpectations(t)
}

func TestSet_ValidationFailureSkipsRepo(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(repo)
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
	primeValidCache(repo)
	listener := NewMockListener(t)
	svc.Subscribe(listener)
	updatedBy := uuid.New()

	repo.EXPECT().Set(mock.Anything, config.SettingSiteName.Key, "Attempt", updatedBy).Return(errors.New("db down"))

	// when
	err := svc.Set(context.Background(), config.SettingSiteName, "Attempt", updatedBy)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, "db down")
	listener.AssertNotCalled(t, "OnSettingChanged", mock.Anything, mock.Anything)
}

func TestSetMultiple_HappyPathNotifiesEachAndBatch(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(repo)
	plain := NewMockListener(t)
	plain.EXPECT().OnSettingChanged(config.SettingSiteName.Key, "Multi Name").Once()
	plain.EXPECT().OnSettingChanged(config.SettingMaintenanceMode.Key, "true").Once()

	listener := NewMockBatchListener(t)
	listener.EXPECT().OnSettingsBatchChanged(batchOfKeys(config.SettingSiteName.Key, config.SettingMaintenanceMode.Key)).Once()

	svc.Subscribe(plain)
	svc.SubscribeBatch(listener)
	updatedBy := uuid.New()

	values := map[config.SiteSettingKey]string{
		config.SettingSiteName.Key:        "Multi Name",
		config.SettingMaintenanceMode.Key: "true",
	}

	repo.EXPECT().SetMultiple(mock.Anything, mock.MatchedBy(func(m map[config.SiteSettingKey]string) bool {
		return m[config.SettingSiteName.Key] == "Multi Name" &&
			m[config.SettingMaintenanceMode.Key] == "true" &&
			len(m) == 2
	}), updatedBy).Return(nil)

	// when
	err := svc.SetMultiple(context.Background(), values, updatedBy)

	// then
	require.NoError(t, err)
	plain.AssertExpectations(t)
	listener.AssertExpectations(t)
	listener.AssertNumberOfCalls(t, "OnSettingsBatchChanged", 1)
}

func TestSetMultiple_UnknownKeyRejected(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(repo)
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
	svc, repo := newTestService(t)
	primeValidCache(repo)
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
	primeValidCache(repo)
	plain := NewMockListener(t)
	listener := NewMockBatchListener(t)
	svc.Subscribe(plain)
	svc.SubscribeBatch(listener)
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
	plain.AssertNotCalled(t, "OnSettingChanged", mock.Anything, mock.Anything)
	listener.AssertNotCalled(t, "OnSettingsBatchChanged", mock.Anything)
}

func TestSubscribe_MultipleListenersAllNotified(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(repo)
	l1 := NewMockListener(t)
	l2 := NewMockListener(t)
	l1.EXPECT().OnSettingChanged(config.SettingSiteName.Key, "Hello").Once()
	l2.EXPECT().OnSettingChanged(config.SettingSiteName.Key, "Hello").Once()
	svc.Subscribe(l1)
	svc.Subscribe(l2)
	updatedBy := uuid.New()

	repo.EXPECT().Set(mock.Anything, config.SettingSiteName.Key, "Hello", updatedBy).Return(nil)

	// when
	err := svc.Set(context.Background(), config.SettingSiteName, "Hello", updatedBy)

	// then
	require.NoError(t, err)
	l1.AssertExpectations(t)
	l2.AssertExpectations(t)
}

func TestSubscribe_NonBatchListenerDoesNotReceiveBatch(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(repo)
	plain := NewMockListener(t)
	plain.EXPECT().OnSettingChanged(config.SettingSiteName.Key, "Batched").Once()

	batch := NewMockBatchListener(t)
	batch.EXPECT().OnSettingsBatchChanged(batchOfKeys(config.SettingSiteName.Key)).Once()

	svc.Subscribe(plain)
	svc.SubscribeBatch(batch)
	updatedBy := uuid.New()

	values := map[config.SiteSettingKey]string{
		config.SettingSiteName.Key: "Batched",
	}

	repo.EXPECT().SetMultiple(mock.Anything, mock.Anything, updatedBy).Return(nil)

	// when
	err := svc.SetMultiple(context.Background(), values, updatedBy)

	// then
	require.NoError(t, err)
	plain.AssertExpectations(t)
	plain.AssertNumberOfCalls(t, "OnSettingChanged", 1)
	batch.AssertExpectations(t)
	batch.AssertNumberOfCalls(t, "OnSettingsBatchChanged", 1)
}

func TestSet_RejectsValueThatDoesNotMatchSettingType(t *testing.T) {
	tests := []struct {
		name    string
		setting *config.SiteSettingDef
		value   string
		wantErr string
	}{
		{
			name:    "text in a numeric setting",
			setting: config.SettingMaxChatRoomMembers,
			value:   "abc",
			wantErr: "must be a whole number",
		},
		{
			name:    "cleared numeric setting",
			setting: config.SettingMaxChatRoomMembers,
			value:   "",
			wantErr: "must be a whole number",
		},
		{
			name:    "yes in a boolean setting",
			setting: config.SettingDMsEnabled,
			value:   "yes",
			wantErr: "must be true or false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, repo := newTestService(t)
			primeValidCache(repo)
			updatedBy := uuid.New()

			// when
			err := svc.Set(context.Background(), tt.setting, tt.value, updatedBy)

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSet_RegisteredValidatorBlocksWrite(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(repo)
	updatedBy := uuid.New()

	svc.RegisterValidator(config.SettingValkeyURL, func(_ context.Context, _ string) error {
		return errors.New("cannot reach valkey")
	})

	// when
	err := svc.Set(context.Background(), config.SettingValkeyURL, "redis://nothing-here:6379", updatedBy)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reach valkey")
}

func TestSet_ValidatorSkippedWhenValueUnchanged(t *testing.T) {
	// given
	svc, repo := newTestService(t)
	primeValidCache(repo)
	updatedBy := uuid.New()

	ran := false
	svc.RegisterValidator(config.SettingValkeyURL, func(_ context.Context, _ string) error {
		ran = true
		return errors.New("should not run")
	})

	repo.EXPECT().Set(mock.Anything, config.SettingValkeyURL.Key, config.SettingValkeyURL.Default, updatedBy).Return(nil)

	// when
	err := svc.Set(context.Background(), config.SettingValkeyURL, config.SettingValkeyURL.Default, updatedBy)

	// then
	require.NoError(t, err)
	assert.False(t, ran)
}
