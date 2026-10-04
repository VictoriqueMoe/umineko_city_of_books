package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoredFileDAO_UpsertGetAndDelete(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png", Size: 10}))
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "posts/a.png", Backend: config.StorageBackendS3, Location: "bucket/site/posts/a.png", Size: 12}))

	// then the second store moved the file to s3 and remembers where
	row, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/a.png"})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, config.StorageBackendS3, row.Backend)
	assert.Equal(t, "bucket/site/posts/a.png", row.Location)
	assert.Equal(t, int64(12), row.Size)
	assert.False(t, row.CreatedAt.IsZero())

	require.NoError(t, repos.StoredFile.Delete(ctx, spec.StoredFileKeys{Keys: []string{"posts/a.png"}}))
	row, err = repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/a.png"})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestStoredFileDAO_BackfillNeverOverwritesExistingRows(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "posts/moved.png", Backend: config.StorageBackendS3, Location: "bucket/posts/moved.png", Size: 1}))

	// when
	inserted, err := repos.StoredFile.Backfill(ctx, spec.StoredFileBackfill{
		Backend:   config.StorageBackendLocal,
		Keys:      []string{"posts/moved.png", "posts/legacy.png", "avatars/old.webp"},
		Locations: []string{"posts/moved.png", "posts/legacy.png", "avatars/old.webp"},
		Sizes:     []int64{1, 2, 3},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, int64(2), inserted)

	moved, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/moved.png"})
	require.NoError(t, err)
	assert.Equal(t, config.StorageBackendS3, moved.Backend)
	assert.Equal(t, "bucket/posts/moved.png", moved.Location)

	legacy, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/legacy.png"})
	require.NoError(t, err)
	assert.Equal(t, config.StorageBackendLocal, legacy.Backend)
	assert.Equal(t, "posts/legacy.png", legacy.Location)
	assert.Equal(t, int64(2), legacy.Size)

	again, err := repos.StoredFile.Backfill(ctx, spec.StoredFileBackfill{Backend: config.StorageBackendLocal, Keys: []string{"posts/legacy.png"}, Locations: []string{"posts/legacy.png"}, Sizes: []int64{2}})
	require.NoError(t, err)
	assert.Equal(t, int64(0), again)
}

func TestStoredFileDAO_BackfillRejectsMismatchedSlices(t *testing.T) {
	tests := []struct {
		name     string
		backfill spec.StoredFileBackfill
	}{
		{name: "missing sizes", backfill: spec.StoredFileBackfill{Backend: config.StorageBackendLocal, Keys: []string{"a/b.png"}, Locations: []string{"a/b.png"}}},
		{name: "missing locations", backfill: spec.StoredFileBackfill{Backend: config.StorageBackendLocal, Keys: []string{"a/b.png"}, Sizes: []int64{1}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)

			// when
			_, err := repos.StoredFile.Backfill(context.Background(), tc.backfill)

			// then
			require.Error(t, err)
		})
	}
}

func TestStoredFileDAO_ListByKeysAndPrefix(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	for _, key := range []string{"avatars/abc_1.webp", "avatars/abc_2.webp", "avatars/xyz_1.webp", "posts/abc_1.png", "avatars/a%c_1.webp"} {
		require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: key, Backend: config.StorageBackendLocal, Location: key, Size: 1}))
	}

	// when
	byPrefix, err := repos.StoredFile.ListByPrefix(ctx, spec.StoredFilePrefix{Prefix: "avatars/abc_"})
	require.NoError(t, err)
	byKeys, err := repos.StoredFile.ListByKeys(ctx, spec.StoredFileKeys{Keys: []string{"posts/abc_1.png", "missing/key.png"}})
	require.NoError(t, err)
	everything, err := repos.StoredFile.ListByPrefix(ctx, spec.StoredFilePrefix{})
	require.NoError(t, err)

	// then prefix matching is literal, so % is not a wildcard
	prefixKeys := make([]string, 0, len(byPrefix))
	for _, row := range byPrefix {
		prefixKeys = append(prefixKeys, row.Key)
	}
	assert.Equal(t, []string{"avatars/abc_1.webp", "avatars/abc_2.webp"}, prefixKeys)
	require.Len(t, byKeys, 1)
	assert.Equal(t, "posts/abc_1.png", byKeys[0].Key)
	assert.Equal(t, "posts/abc_1.png", byKeys[0].Location)
	assert.Len(t, everything, 5)
}

func TestStoredFileDAO_CountByBackend(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "a/1.png", Backend: config.StorageBackendS3, Location: "b/a/1.png", Size: 1}))
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "a/2.png", Backend: config.StorageBackendS3, Location: "b/a/2.png", Size: 1}))
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "a/3.png", Backend: config.StorageBackendLocal, Location: "a/3.png", Size: 1}))

	// when
	s3Count, err := repos.StoredFile.CountByBackend(ctx, spec.StoredFileBackend{Backend: config.StorageBackendS3})
	require.NoError(t, err)
	localCount, err := repos.StoredFile.CountByBackend(ctx, spec.StoredFileBackend{Backend: config.StorageBackendLocal})
	require.NoError(t, err)

	// then
	assert.Equal(t, int64(2), s3Count)
	assert.Equal(t, int64(1), localCount)
}
