package storage

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/storage/engines"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func completeS3Settings(backend config.StorageBackend) map[config.SiteSettingKey]string {
	return map[config.SiteSettingKey]string{
		config.SettingStorageBackend.Key:    string(backend),
		config.SettingS3Endpoint.Key:        "https://s3.example.com",
		config.SettingS3Region.Key:          "auto",
		config.SettingS3Bucket.Key:          "uploads",
		config.SettingS3AccessKeyID.Key:     "key",
		config.SettingS3SecretAccessKey.Key: "secret",
	}
}

func TestSettingsValidator(t *testing.T) {
	tests := []struct {
		name       string
		merged     map[config.SiteSettingKey]string
		changed    []config.SiteSettingKey
		storedInS3 int64
		countsRows bool
		probeErr   error
		wantProbe  bool
		wantErr    string
	}{
		{
			name:    "unrelated change does nothing",
			merged:  completeS3Settings(config.StorageBackendS3),
			changed: []config.SiteSettingKey{config.SettingSiteName.Key},
		},
		{
			name:      "switching to s3 probes the bucket",
			merged:    completeS3Settings(config.StorageBackendS3),
			changed:   []config.SiteSettingKey{config.SettingStorageBackend.Key},
			wantProbe: true,
		},
		{
			name:      "an unreachable bucket refuses the save",
			merged:    completeS3Settings(config.StorageBackendS3),
			changed:   []config.SiteSettingKey{config.SettingStorageBackend.Key},
			wantProbe: true,
			probeErr:  errors.New("SignatureDoesNotMatch"),
			wantErr:   "cannot use these S3 settings: SignatureDoesNotMatch",
		},
		{
			name:    "switching back to local does not need s3 to answer",
			merged:  completeS3Settings(config.StorageBackendLocal),
			changed: []config.SiteSettingKey{config.SettingStorageBackend.Key},
		},
		{
			name:       "changing a bucket with existing s3 files is allowed and probed",
			merged:     completeS3Settings(config.StorageBackendS3),
			changed:    []config.SiteSettingKey{config.SettingS3Bucket.Key},
			countsRows: true,
			storedInS3: 4,
			wantProbe:  true,
		},
		{
			name:       "changing the endpoint while files live in s3 is refused",
			merged:     completeS3Settings(config.StorageBackendS3),
			changed:    []config.SiteSettingKey{config.SettingS3Endpoint.Key},
			countsRows: true,
			storedInS3: 4,
			wantErr:    "4 uploads are stored on the current S3 endpoint",
		},
		{
			name: "clearing s3 while files live there is refused",
			merged: map[config.SiteSettingKey]string{
				config.SettingStorageBackend.Key: string(config.StorageBackendLocal),
				config.SettingS3Region.Key:       "auto",
			},
			changed:    []config.SiteSettingKey{config.SettingS3SecretAccessKey.Key},
			countsRows: true,
			storedInS3: 2,
			wantErr:    "2 uploads are stored in S3",
		},
		{
			name: "clearing s3 with nothing stored there is allowed",
			merged: map[config.SiteSettingKey]string{
				config.SettingStorageBackend.Key: string(config.StorageBackendLocal),
			},
			changed:    []config.SiteSettingKey{config.SettingS3SecretAccessKey.Key},
			countsRows: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo := repository.NewMockStoredFileRepository(t)
			if tc.countsRows {
				repo.EXPECT().CountByBackend(mock.Anything, spec.StoredFileBackend{Backend: config.StorageBackendS3}).Return(tc.storedInS3, nil)
			}
			probed := false
			probe := func(_ context.Context, cfg engines.S3Config) error {
				probed = true
				assert.Equal(t, "uploads", cfg.Bucket)
				return tc.probeErr
			}

			// when
			err := SettingsValidator(repo, probe)(context.Background(), tc.merged, tc.changed)

			// then
			assert.Equal(t, tc.wantProbe, probed)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
