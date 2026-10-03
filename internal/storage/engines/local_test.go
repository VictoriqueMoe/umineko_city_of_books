package engines

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestLocal(t *testing.T) (*Local, string) {
	t.Helper()
	dir := t.TempDir()
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(dir).Maybe()
	return NewLocal(settingsSvc), dir
}

func TestLocal_PutFileThenGetAndHead(t *testing.T) {
	// given
	local, dir := newTestLocal(t)
	ctx := context.Background()
	src := writeLocalSource(t, "0123456789")

	// when
	location, err := local.PutFile(ctx, "posts/a.txt", src)

	// then
	require.NoError(t, err)
	assert.Equal(t, "posts/a.txt", location)
	_, statErr := os.Stat(src)
	assert.NoError(t, statErr, "PutFile must not consume the source file")

	data, err := os.ReadFile(filepath.Join(dir, "posts", "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "0123456789", string(data))

	info, err := local.Head(ctx, location)
	require.NoError(t, err)
	assert.Equal(t, "posts/a.txt", info.Key)
	assert.Equal(t, location, info.Location)
	assert.Equal(t, int64(10), info.Size)
	assert.Equal(t, config.StorageBackendLocal, info.Backend)

	body, err := local.Get(ctx, location, nil)
	require.NoError(t, err)
	assert.Equal(t, "0123456789", readAll(t, body))

	ranged, err := local.Get(ctx, location, &engine.ByteRange{Start: 3, End: 5})
	require.NoError(t, err)
	assert.Equal(t, "345", readAll(t, ranged))
}

func TestLocal_PutFileOverwritesAndLeavesNoTempFiles(t *testing.T) {
	// given
	local, dir := newTestLocal(t)
	mustPut(t, local, "posts/a.txt", "old")

	// when
	mustPut(t, local, "posts/a.txt", "new")

	// then
	data, err := os.ReadFile(filepath.Join(dir, "posts", "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
	entries, err := os.ReadDir(filepath.Join(dir, "posts"))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLocal_MissingObjects(t *testing.T) {
	// given
	local, dir := newTestLocal(t)
	ctx := context.Background()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "posts", "folder"), 0755))

	tests := []struct {
		name     string
		location string
	}{
		{name: "missing file", location: "posts/missing.txt"},
		{name: "missing directory", location: "nowhere/missing.txt"},
		{name: "directory is not an object", location: "posts/folder"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			_, headErr := local.Head(ctx, tc.location)
			_, getErr := local.Get(ctx, tc.location, nil)

			// then
			assert.ErrorIs(t, headErr, engine.ErrNotFound)
			assert.ErrorIs(t, getErr, engine.ErrNotFound)
		})
	}
}

func TestLocal_RejectsInvalidKeys(t *testing.T) {
	// given
	local, _ := newTestLocal(t)
	ctx := context.Background()
	src := writeLocalSource(t, "x")

	// when
	_, putErr := local.PutFile(ctx, "../escape.txt", src)
	_, headErr := local.Head(ctx, "../escape.txt")
	_, getErr := local.Get(ctx, ".staging/x", nil)
	deleteErr := local.Delete(ctx, "a/../../b")

	// then
	assert.ErrorIs(t, putErr, engine.ErrInvalidKey)
	assert.ErrorIs(t, headErr, engine.ErrInvalidKey)
	assert.ErrorIs(t, getErr, engine.ErrInvalidKey)
	assert.ErrorIs(t, deleteErr, engine.ErrInvalidKey)
}

func TestLocal_DeleteIgnoresMissing(t *testing.T) {
	// given
	local, dir := newTestLocal(t)
	location := mustPut(t, local, "posts/a.txt", "x")

	// when
	err := local.Delete(context.Background(), location, "posts/never-existed.txt")

	// then
	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(dir, "posts", "a.txt"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestLocal_ListSkipsHiddenEntriesAndFiltersPrefix(t *testing.T) {
	// given
	local, dir := newTestLocal(t)
	ctx := context.Background()
	for _, key := range []string{"avatars/abc_1.webp", "avatars/abc_2.webp", "avatars/xyz_1.webp", "posts/p.png"} {
		mustPut(t, local, key, key)
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".staging", "upload-1"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".staging", "upload-1", "staged.webp"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "avatars", ".abc_3.webp.tmp-1"), []byte("x"), 0644))

	tests := []struct {
		name   string
		prefix string
		want   []string
	}{
		{name: "everything", prefix: "", want: []string{"avatars/abc_1.webp", "avatars/abc_2.webp", "avatars/xyz_1.webp", "posts/p.png"}},
		{name: "one directory", prefix: "posts/", want: []string{"posts/p.png"}},
		{name: "filename prefix", prefix: "avatars/abc_", want: []string{"avatars/abc_1.webp", "avatars/abc_2.webp"}},
		{name: "missing directory", prefix: "nothing/here_", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			var got []string
			for info, err := range local.List(ctx, tc.prefix) {
				require.NoError(t, err)
				assert.Equal(t, config.StorageBackendLocal, info.Backend)
				assert.Equal(t, info.Key, info.Location)
				got = append(got, info.Key)
			}

			// then
			slices.Sort(got)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLocal_ListContinuesPastUnreadableDirectory(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a non-root unix user, chmod cannot lock a directory on Windows or for root")
	}

	// given
	local, dir := newTestLocal(t)
	ctx := context.Background()
	for _, key := range []string{"art/a.png", "lost/b.png", "posts/c.png"} {
		mustPut(t, local, key, key)
	}
	locked := filepath.Join(dir, "lost")
	require.NoError(t, os.Chmod(locked, 0))
	t.Cleanup(func() {
		_ = os.Chmod(locked, 0755)
	})

	// when
	var keys []string
	var errs []error
	for info, err := range local.List(ctx, "") {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		keys = append(keys, info.Key)
	}

	// then the unreadable directory is reported and everything after it is still listed
	slices.Sort(keys)
	assert.Equal(t, []string{"art/a.png", "posts/c.png"}, keys)
	assert.Len(t, errs, 1)
}

func TestLocal_ListStopsWhenConsumerBreaks(t *testing.T) {
	// given
	local, _ := newTestLocal(t)
	for _, key := range []string{"posts/a.png", "posts/b.png", "posts/c.png"} {
		mustPut(t, local, key, key)
	}

	// when
	seen := 0
	for _, err := range local.List(context.Background(), "") {
		require.NoError(t, err)
		seen++
		break
	}

	// then
	assert.Equal(t, 1, seen)
}
