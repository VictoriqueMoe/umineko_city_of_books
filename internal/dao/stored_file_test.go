package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model"
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

	deleted, err := repos.StoredFile.DeleteAt(ctx, spec.StoredFileLocations{Backend: config.StorageBackendS3, Keys: []string{"posts/a.png"}, Locations: []string{"bucket/site/posts/a.png"}})
	require.NoError(t, err)
	assert.Equal(t, []string{"posts/a.png"}, deleted)
	row, err = repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/a.png"})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestStoredFileDAO_DeleteAtOnlyRemovesRowsStillAtTheReadLocation(t *testing.T) {
	// given one row still where it was read and one that has since moved to s3
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "posts/stays.png", Backend: config.StorageBackendLocal, Location: "posts/stays.png", Size: 1}))
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "posts/moved.png", Backend: config.StorageBackendS3, Location: "b/posts/moved.png", Size: 1}))

	// when
	deleted, err := repos.StoredFile.DeleteAt(ctx, spec.StoredFileLocations{
		Backend:   config.StorageBackendLocal,
		Keys:      []string{"posts/stays.png", "posts/moved.png"},
		Locations: []string{"posts/stays.png", "posts/moved.png"},
	})

	// then only the unchanged row is dropped and the moved row survives for a second pass
	require.NoError(t, err)
	assert.Equal(t, []string{"posts/stays.png"}, deleted)
	gone, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/stays.png"})
	require.NoError(t, err)
	assert.Nil(t, gone)
	moved, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/moved.png"})
	require.NoError(t, err)
	require.NotNil(t, moved)
	assert.Equal(t, config.StorageBackendS3, moved.Backend)
}

func TestStoredFileDAO_DeleteAtRejectsMismatchedSlices(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.StoredFile.DeleteAt(context.Background(), spec.StoredFileLocations{Backend: config.StorageBackendLocal, Keys: []string{"a/b.png"}})

	// then
	require.Error(t, err)
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

func TestStoredFileDAO_ListFallbackReturnsEveryFallbackRowInKeyOrderAndSkipsEverythingElse(t *testing.T) {
	// given fallback rows, a legacy local row, a backfilled row and an s3 row
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	for _, key := range []string{"posts/c.png", "posts/a.png", "avatars/b.webp", "posts/b.png"} {
		require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: key, Backend: config.StorageBackendLocal, Location: key, Size: 1, Fallback: true}))
	}
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "posts/legacy.png", Backend: config.StorageBackendLocal, Location: "posts/legacy.png", Size: 1}))
	_, err := repos.StoredFile.Backfill(ctx, spec.StoredFileBackfill{Backend: config.StorageBackendLocal, Keys: []string{"posts/disk.png"}, Locations: []string{"posts/disk.png"}, Sizes: []int64{1}})
	require.NoError(t, err)
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "posts/s3.png", Backend: config.StorageBackendS3, Location: "b/posts/s3.png", Size: 1}))

	// when
	rows, err := repos.StoredFile.ListFallback(ctx)
	require.NoError(t, err)

	// then every fallback row comes back exactly once and no legacy, backfilled or s3 row does
	var keys []string
	for _, row := range rows {
		keys = append(keys, row.Key)
		assert.True(t, row.Fallback)
	}
	assert.Equal(t, []string{"avatars/b.webp", "posts/a.png", "posts/b.png", "posts/c.png"}, keys)
}

func TestStoredFileDAO_MoveClearsTheFallbackMark(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png", Size: 1, Fallback: true}))
	row, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/a.png"})
	require.NoError(t, err)

	// when
	moved, err := repos.StoredFile.Move(ctx, spec.StoredFileMove{Key: row.Key, FromBackend: row.Backend, FromLocation: row.Location, Backend: config.StorageBackendS3, Location: "b/posts/a.png", Size: 1})

	// then
	require.NoError(t, err)
	assert.True(t, moved)
	after, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/a.png"})
	require.NoError(t, err)
	assert.False(t, after.Fallback)
	remaining, err := repos.StoredFile.ListFallback(ctx)
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestStoredFileDAO_MoveOnlyWhenTheRowIsUnchanged(t *testing.T) {
	tests := []struct {
		name      string
		move      spec.StoredFileMove
		wantMoved bool
		want      model.StoredFileRow
	}{
		{
			name:      "row still on the expected backend and location",
			move:      spec.StoredFileMove{Key: "posts/a.png", FromBackend: config.StorageBackendLocal, FromLocation: "posts/a.png", Backend: config.StorageBackendS3, Location: "b/posts/a.png", Size: 7},
			wantMoved: true,
			want:      model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendS3, Location: "b/posts/a.png", Size: 7},
		},
		{
			name: "row was re-stored elsewhere in the meantime",
			move: spec.StoredFileMove{Key: "posts/a.png", FromBackend: config.StorageBackendLocal, FromLocation: "posts/old.png", Backend: config.StorageBackendS3, Location: "b/posts/a.png", Size: 7},
			want: model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png", Size: 1},
		},
		{
			name: "row is not on the expected backend",
			move: spec.StoredFileMove{Key: "posts/a.png", FromBackend: config.StorageBackendS3, FromLocation: "posts/a.png", Backend: config.StorageBackendS3, Location: "b/posts/a.png", Size: 7},
			want: model.StoredFileRow{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png", Size: 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			require.NoError(t, repos.StoredFile.Upsert(ctx, spec.NewStoredFile{Key: "posts/a.png", Backend: config.StorageBackendLocal, Location: "posts/a.png", Size: 1}))
			before, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/a.png"})
			require.NoError(t, err)

			// when
			moved, err := repos.StoredFile.Move(ctx, tc.move)

			// then the original upload time survives a move
			require.NoError(t, err)
			assert.Equal(t, tc.wantMoved, moved)
			after, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/a.png"})
			require.NoError(t, err)
			require.NotNil(t, after)
			assert.Equal(t, tc.want.Backend, after.Backend)
			assert.Equal(t, tc.want.Location, after.Location)
			assert.Equal(t, tc.want.Size, after.Size)
			assert.True(t, before.CreatedAt.Equal(after.CreatedAt))
		})
	}
}

func TestStoredFileDAO_MoveOfADeletedRowIsANoOp(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	moved, err := repos.StoredFile.Move(ctx, spec.StoredFileMove{Key: "posts/gone.png", FromBackend: config.StorageBackendLocal, FromLocation: "posts/gone.png", Backend: config.StorageBackendS3, Location: "b/posts/gone.png", Size: 1})

	// then the move never resurrects the row
	require.NoError(t, err)
	assert.False(t, moved)
	row, err := repos.StoredFile.Get(ctx, spec.StoredFileKey{Key: "posts/gone.png"})
	require.NoError(t, err)
	assert.Nil(t, row)
}
