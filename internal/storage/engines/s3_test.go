package engines

import (
	"context"
	"net/http/httptest"
	"slices"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engine"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testBucket = "uploads"

type (
	fakeS3 struct {
		backend *s3mem.Backend
		url     string
	}

	s3Settings map[config.SiteSettingKey]string
)

func newFakeS3(t *testing.T, buckets ...string) *fakeS3 {
	t.Helper()
	backend := s3mem.New()
	for _, bucket := range append([]string{testBucket}, buckets...) {
		require.NoError(t, backend.CreateBucket(bucket))
	}
	server := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(server.Close)
	return &fakeS3{backend: backend, url: server.URL}
}

func (f *fakeS3) settings(prefix string) s3Settings {
	return s3Settings{
		config.SettingS3Endpoint.Key:        f.url,
		config.SettingS3Region.Key:          "us-east-1",
		config.SettingS3Bucket.Key:          testBucket,
		config.SettingS3Prefix.Key:          prefix,
		config.SettingS3AccessKeyID.Key:     "key",
		config.SettingS3SecretAccessKey.Key: "secret",
		config.SettingS3ForcePathStyle.Key:  "true",
	}
}

func settingsReturning(t *testing.T, values ...s3Settings) *settings.MockService {
	t.Helper()
	settingsSvc := settings.NewMockService(t)
	for _, value := range values {
		settingsSvc.EXPECT().GetAll(mock.Anything).Return(value).Once()
	}
	return settingsSvc
}

func configuredS3(t *testing.T, values s3Settings) *S3 {
	t.Helper()
	s3Engine := NewS3(settingsReturning(t, values))
	require.NoError(t, s3Engine.Reconfigure(context.Background()))
	require.True(t, s3Engine.Enabled())
	return s3Engine
}

func TestS3Config(t *testing.T) {
	tests := []struct {
		name         string
		values       s3Settings
		wantComplete bool
		wantValid    bool
		wantPrefix   string
	}{
		{name: "nothing set", values: s3Settings{}, wantValid: true},
		{name: "whitespace only fields are not complete", values: s3Settings{config.SettingS3Region.Key: " ", config.SettingS3Bucket.Key: "b", config.SettingS3AccessKeyID.Key: "k", config.SettingS3SecretAccessKey.Key: " "}, wantValid: true},
		{name: "complete without endpoint targets aws", values: s3Settings{config.SettingS3Region.Key: "eu-west-2", config.SettingS3Bucket.Key: "b", config.SettingS3AccessKeyID.Key: "k", config.SettingS3SecretAccessKey.Key: "s"}, wantComplete: true, wantValid: true},
		{name: "endpoint without scheme is invalid", values: s3Settings{config.SettingS3Endpoint.Key: "abc.r2.cloudflarestorage.com"}},
		{name: "endpoint with https is valid", values: s3Settings{config.SettingS3Endpoint.Key: "https://abc.r2.cloudflarestorage.com"}, wantValid: true},
		{name: "prefix is normalised", values: s3Settings{config.SettingS3Prefix.Key: " /site/uploads/ "}, wantValid: true, wantPrefix: "site/uploads/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			cfg := S3ConfigFrom(tc.values)

			// then
			assert.Equal(t, tc.wantComplete, cfg.Complete())
			assert.Equal(t, tc.wantPrefix, cfg.Prefix)
			if tc.wantValid {
				assert.NoError(t, cfg.Validate())
				return
			}
			assert.Error(t, cfg.Validate())
		})
	}
}

func TestS3_DisabledUntilConfigured(t *testing.T) {
	// given
	s3Engine := NewS3(settingsReturning(t, s3Settings{config.SettingS3Region.Key: "us-east-1", config.SettingS3Bucket.Key: testBucket}))

	// when
	err := s3Engine.Reconfigure(context.Background())

	// then
	require.NoError(t, err)
	assert.False(t, s3Engine.Enabled())
	_, headErr := s3Engine.Head(context.Background(), testBucket+"/posts/a.txt")
	assert.ErrorIs(t, headErr, engine.ErrDisabled)
	assert.ErrorIs(t, s3Engine.Delete(context.Background(), testBucket+"/posts/a.txt"), engine.ErrDisabled)
}

func TestS3_ReconfigureReportsUnreachableBucket(t *testing.T) {
	// given
	fake := newFakeS3(t)
	values := fake.settings("")
	values[config.SettingS3Bucket.Key] = "missing-bucket"
	s3Engine := NewS3(settingsReturning(t, values))

	// when
	err := s3Engine.Reconfigure(context.Background())

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing-bucket")
}

func TestS3_ProbeRejectsBadConfiguration(t *testing.T) {
	// given
	fake := newFakeS3(t)
	missingBucket := S3ConfigFrom(fake.settings(""))
	missingBucket.Bucket = "missing-bucket"
	noScheme := S3ConfigFrom(fake.settings(""))
	noScheme.Endpoint = "abc.r2.cloudflarestorage.com"

	// when
	okErr := ProbeS3(context.Background(), S3ConfigFrom(fake.settings("")))
	bucketErr := ProbeS3(context.Background(), missingBucket)
	schemeErr := ProbeS3(context.Background(), noScheme)

	// then
	assert.NoError(t, okErr)
	assert.ErrorContains(t, bucketErr, "missing-bucket")
	assert.ErrorContains(t, schemeErr, "full http(s) URL")
}

func TestS3_RoundTrip(t *testing.T) {
	prefixes := []struct {
		name         string
		prefix       string
		wantLocation string
	}{
		{name: "no prefix", prefix: "", wantLocation: testBucket + "/posts/a.txt"},
		{name: "prefix", prefix: "/cityofbooks/", wantLocation: testBucket + "/cityofbooks/posts/a.txt"},
	}

	for _, tc := range prefixes {
		t.Run(tc.name, func(t *testing.T) {
			// given
			fake := newFakeS3(t)
			s3Engine := configuredS3(t, fake.settings(tc.prefix))
			ctx := context.Background()

			// when
			location := mustPut(t, s3Engine, "posts/a.txt", "0123456789")

			// then
			assert.Equal(t, tc.wantLocation, location)

			info, err := s3Engine.Head(ctx, location)
			require.NoError(t, err)
			assert.Equal(t, location, info.Location)
			assert.Equal(t, int64(10), info.Size)
			assert.Equal(t, config.StorageBackendS3, info.Backend)

			body, err := s3Engine.Get(ctx, location, nil)
			require.NoError(t, err)
			assert.Equal(t, "0123456789", readAll(t, body))

			ranged, err := s3Engine.Get(ctx, location, &engine.ByteRange{Start: 2, End: 4})
			require.NoError(t, err)
			assert.Equal(t, "234", readAll(t, ranged))

			var listed []engine.ObjectInfo
			for object, err := range s3Engine.List(ctx, "posts/") {
				require.NoError(t, err)
				listed = append(listed, object)
			}
			require.Len(t, listed, 1)
			assert.Equal(t, "posts/a.txt", listed[0].Key)
			assert.Equal(t, location, listed[0].Location)

			require.NoError(t, s3Engine.Delete(ctx, location, testBucket+"/posts/never-existed.txt"))
			_, err = s3Engine.Head(ctx, location)
			assert.ErrorIs(t, err, engine.ErrNotFound)
		})
	}
}

func TestS3_OldObjectsStayReachableAfterBucketAndPrefixChange(t *testing.T) {
	// given
	fake := newFakeS3(t, "new-bucket")
	moved := fake.settings("new-prefix")
	moved[config.SettingS3Bucket.Key] = "new-bucket"
	s3Engine := NewS3(settingsReturning(t, fake.settings("old-prefix"), moved))
	require.NoError(t, s3Engine.Reconfigure(context.Background()))
	ctx := context.Background()
	oldLocation := mustPut(t, s3Engine, "posts/old.txt", "old")

	// when
	require.NoError(t, s3Engine.Reconfigure(ctx))
	newLocation := mustPut(t, s3Engine, "posts/new.txt", "new")

	// then
	assert.Equal(t, testBucket+"/old-prefix/posts/old.txt", oldLocation)
	assert.Equal(t, "new-bucket/new-prefix/posts/new.txt", newLocation)

	body, err := s3Engine.Get(ctx, oldLocation, nil)
	require.NoError(t, err)
	assert.Equal(t, "old", readAll(t, body))

	require.NoError(t, s3Engine.Delete(ctx, oldLocation, newLocation))
	_, rawErr := fake.backend.HeadObject(testBucket, "old-prefix/posts/old.txt")
	assert.Error(t, rawErr, "deleting must reach the object where it was originally written")
}

func TestS3_MissingObjectIsNotFound(t *testing.T) {
	// given
	s3Engine := configuredS3(t, newFakeS3(t).settings(""))
	ctx := context.Background()

	// when
	_, headErr := s3Engine.Head(ctx, testBucket+"/posts/missing.txt")
	_, getErr := s3Engine.Get(ctx, testBucket+"/posts/missing.txt", nil)

	// then
	assert.ErrorIs(t, headErr, engine.ErrNotFound)
	assert.ErrorIs(t, getErr, engine.ErrNotFound)
}

func TestS3_ListFiltersByKeyPrefix(t *testing.T) {
	// given
	s3Engine := configuredS3(t, newFakeS3(t).settings("site"))
	for _, key := range []string{"avatars/abc_1.webp", "avatars/abc_2.webp", "avatars/xyz_1.webp"} {
		mustPut(t, s3Engine, key, key)
	}

	// when
	var got []string
	for object, err := range s3Engine.List(context.Background(), "avatars/abc_") {
		require.NoError(t, err)
		got = append(got, object.Key)
	}

	// then
	slices.Sort(got)
	assert.Equal(t, []string{"avatars/abc_1.webp", "avatars/abc_2.webp"}, got)
}

func TestS3_RejectsInvalidKeysAndLocations(t *testing.T) {
	// given
	s3Engine := configuredS3(t, newFakeS3(t).settings(""))
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{name: "put with traversal key", call: func() error {
			_, err := s3Engine.PutFile(ctx, "../escape.txt", writeLocalSource(t, "x"))
			return err
		}},
		{name: "get hidden object key", call: func() error {
			_, err := s3Engine.Get(ctx, testBucket+"/.staging/x", nil)
			return err
		}},
		{name: "head without bucket", call: func() error {
			_, err := s3Engine.Head(ctx, "/posts/a.txt")
			return err
		}},
		{name: "head without object key", call: func() error {
			_, err := s3Engine.Head(ctx, testBucket)
			return err
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			err := tc.call()

			// then
			assert.ErrorIs(t, err, engine.ErrInvalidKey)
		})
	}
}

func TestS3_ReconfigureSkipsUnchangedSettings(t *testing.T) {
	// given
	fake := newFakeS3(t)
	s3Engine := NewS3(settingsReturning(t, fake.settings(""), fake.settings("")))
	require.NoError(t, s3Engine.Reconfigure(context.Background()))
	before := s3Engine.current()

	// when
	err := s3Engine.Reconfigure(context.Background())

	// then the client and its connection pool are kept
	require.NoError(t, err)
	assert.Same(t, before, s3Engine.current())
}

func TestS3_OnSettingsBatchChangedOnlyReactsToS3Keys(t *testing.T) {
	// given
	fake := newFakeS3(t)
	settingsSvc := settings.NewMockService(t)
	s3Engine := NewS3(settingsSvc)

	// when an unrelated batch arrives
	s3Engine.OnSettingsBatchChanged([]config.SiteSettingKey{config.SettingSMTPHost.Key})

	// then nothing was read from settings and the engine stays off
	assert.False(t, s3Engine.Enabled())

	// when an s3 key changes
	settingsSvc.EXPECT().GetAll(mock.Anything).Return(fake.settings("")).Once()
	s3Engine.OnSettingsBatchChanged([]config.SiteSettingKey{config.SettingS3Bucket.Key})

	// then
	assert.True(t, s3Engine.Enabled())
}
