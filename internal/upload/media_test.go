package upload

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testMediaLimit = 1024 * 1024
)

var (
	errStoreFailed = errors.New("store failed")
)

func memoryUpload(contentType string, filename string, data []byte, isSpoiler bool) FileUpload {
	return FileUpload{
		ContentType: contentType,
		Filename:    filename,
		Size:        int64(len(data)),
		IsSpoiler:   isSpoiler,
		Open: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}
}

func newMediaService(t *testing.T, maxImageSize int, maxPixels int) (*service, *storage.MockService, string) {
	t.Helper()

	dir := t.TempDir()
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(dir).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(maxImageSize).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxVideoSize).Return(testMediaLimit).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxAudioSize).Return(testMediaLimit).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(maxPixels).Maybe()

	storageSvc := storage.NewMockService(t)
	storedFiles := repository.NewMockStoredFileRepository(t)
	storedFiles.EXPECT().ListByPrefix(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	return NewService(settingsSvc, storageSvc, storedFiles).(*service), storageSvc, dir
}

func TestStoreMedia_RefusesTheWholeBatchBeforeStoringAnything(t *testing.T) {
	png := mustPNGBytes()
	notAnImage := []byte("this is plainly not an image at all")

	tests := []struct {
		name         string
		files        []FileUpload
		maxImageSize int
		maxPixels    int
		wantErr      error
	}{
		{
			name:         "an image over the size limit is refused",
			files:        []FileUpload{memoryUpload("image/png", "one.png", png, false)},
			maxImageSize: 1,
			maxPixels:    bounds.FallbackMaxImagePixels,
			wantErr:      ErrFileTooLarge,
		},
		{
			name:         "bytes that are not an image are refused",
			files:        []FileUpload{memoryUpload("image/png", "one.png", notAnImage, false)},
			maxImageSize: testMediaLimit,
			maxPixels:    bounds.FallbackMaxImagePixels,
			wantErr:      ErrInvalidFileType,
		},
		{
			name:         "an image declared as audio is refused",
			files:        []FileUpload{memoryUpload("audio/mpeg", "one.mp3", png, false)},
			maxImageSize: testMediaLimit,
			maxPixels:    bounds.FallbackMaxImagePixels,
			wantErr:      ErrInvalidAudioType,
		},
		{
			name:         "an image declared as video is refused",
			files:        []FileUpload{memoryUpload("video/mp4", "one.mp4", png, false)},
			maxImageSize: testMediaLimit,
			maxPixels:    bounds.FallbackMaxImagePixels,
			wantErr:      ErrInvalidVideoType,
		},
		{
			name:         "an image past the pixel budget is refused",
			files:        []FileUpload{memoryUpload("image/png", "one.png", png, false)},
			maxImageSize: testMediaLimit,
			maxPixels:    3,
			wantErr:      bounds.ErrImageBounds,
		},
		{
			name: "a bad last file stops the good first file from being stored",
			files: []FileUpload{
				memoryUpload("image/png", "one.png", png, false),
				memoryUpload("image/png", "two.png", notAnImage, false),
			},
			maxImageSize: testMediaLimit,
			maxPixels:    bounds.FallbackMaxImagePixels,
			wantErr:      ErrInvalidFileType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given a strict storage mock that fails the test on any write
			svc, _, dir := newMediaService(t, tt.maxImageSize, tt.maxPixels)

			// when
			stored, err := svc.StoreMedia(t.Context(), "posts", tt.files)

			// then
			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, stored)
			assertStagingEmpty(t, dir)
		})
	}
}

func TestStoreMedia_StoresEachFileInDisplayOrder(t *testing.T) {
	// given two valid images, the second marked as a spoiler
	png := mustPNGBytes()
	svc, storageSvc, dir := newMediaService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	objects := captureStore(t, storageSvc)

	// when
	stored, err := svc.StoreMedia(t.Context(), "posts", []FileUpload{
		memoryUpload("image/png", "one.png", png, false),
		memoryUpload("image/png", "two.png", png, true),
	})

	// then each file is stored at its position with its own spoiler flag and dimensions
	require.NoError(t, err)
	require.Len(t, stored, 2)
	require.Len(t, *objects, 2)
	for i, m := range stored {
		assert.Equal(t, "/uploads/"+(*objects)[i].key, m.URL)
		assert.Equal(t, MediaTypeImage, m.MediaType)
		assert.Equal(t, i, m.SortOrder)
		assert.Equal(t, 2, m.Width)
		assert.Equal(t, 2, m.Height)
	}
	assert.Equal(t, "one.png", stored[0].Filename)
	assert.False(t, stored[0].IsSpoiler)
	assert.Equal(t, "two.png", stored[1].Filename)
	assert.True(t, stored[1].IsSpoiler)
	assertStagingEmpty(t, dir)
}

func TestStoreMedia_AFailedStoreDiscardsWhatWasAlreadyStored(t *testing.T) {
	// given two valid images where storage accepts the first and refuses the second
	png := mustPNGBytes()
	svc, storageSvc, dir := newMediaService(t, testMediaLimit, bounds.FallbackMaxImagePixels)

	var firstKey string
	storageSvc.EXPECT().Store(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string, _ string) error {
		firstKey = key
		return nil
	}).Once()
	storageSvc.EXPECT().Store(mock.Anything, mock.Anything, mock.Anything).Return(errStoreFailed).Once()

	var deletedKeys []string
	storageSvc.EXPECT().Delete(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, keys ...string) error {
		deletedKeys = keys
		return nil
	}).Once()

	// when
	stored, err := svc.StoreMedia(t.Context(), "posts", []FileUpload{
		memoryUpload("image/png", "one.png", png, false),
		memoryUpload("image/png", "two.png", png, false),
	})

	// then the failure is returned and the first object is removed again
	require.ErrorIs(t, err, errStoreFailed)
	assert.Nil(t, stored)
	assert.Equal(t, []string{firstKey}, deletedKeys)
	assertStagingEmpty(t, dir)
}

func TestStoreMedia_AVideoKeepsItsStagedCopyUntilItIsDiscarded(t *testing.T) {
	// given
	svc, storageSvc, dir := newMediaService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	objects := captureStore(t, storageSvc)

	// when
	stored, err := svc.StoreMedia(t.Context(), "posts", []FileUpload{memoryUpload("video/mp4", "clip.mp4", mp4Magic, false)})

	// then the transcoder can still read the local copy, so nothing has to come back from storage
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, MediaTypeVideo, stored[0].MediaType)
	assert.Equal(t, "/uploads/"+(*objects)[0].key, stored[0].URL)
	data, err := os.ReadFile(stored[0].stagedVideo)
	require.NoError(t, err)
	assert.Equal(t, mp4Magic, data)

	// and when it is discarded
	storageSvc.EXPECT().Delete(mock.Anything, []string{(*objects)[0].key}).Return(nil).Once()
	svc.DiscardMedia(stored...)

	// then both the staged copy and the stored object are gone
	assertStagingEmpty(t, dir)
}

func TestTranscodeMedia_WithoutAProcessorKeepsTheRawVideoAndReleasesTheStagedCopy(t *testing.T) {
	// given a stored video on a service that has no media processor
	svc, storageSvc, dir := newMediaService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	captureStore(t, storageSvc)
	stored, err := svc.StoreMedia(t.Context(), "posts", []FileUpload{memoryUpload("video/mp4", "clip.mp4", mp4Magic, false)})
	require.NoError(t, err)

	// when
	svc.TranscodeMedia(t.Context(), "posts", stored, []int64{7}, nil, nil)

	// then the staged copy is gone and the strict storage mock proves the raw upload was left alone
	assertStagingEmpty(t, dir)
}

func TestStoreMedia_AVideoStoreFailureReleasesTheStagedCopy(t *testing.T) {
	// given
	svc, storageSvc, dir := newMediaService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	storageSvc.EXPECT().Store(mock.Anything, mock.Anything, mock.Anything).Return(errStoreFailed).Once()

	// when
	stored, err := svc.StoreMedia(t.Context(), "posts", []FileUpload{memoryUpload("video/mp4", "clip.mp4", mp4Magic, false)})

	// then
	require.ErrorIs(t, err, errStoreFailed)
	assert.Nil(t, stored)
	assertStagingEmpty(t, dir)
}

func TestStoredMedia_RowsAndResponsesKeepTheirOrder(t *testing.T) {
	// given
	stored := []StoredMedia{
		{URL: "/uploads/posts/a.webp", MediaType: MediaTypeImage, Filename: "a.png", SortOrder: 0, Width: 4, Height: 3},
		{URL: "/uploads/posts/b.mp4", MediaType: MediaTypeVideo, Filename: "b.mp4", SortOrder: 1, IsSpoiler: true},
	}

	// when
	rows := MediaRows(stored)
	responses := MediaResponses(stored, []int64{11, 12})

	// then rows wait for the repository to fill in their owner, responses carry the new row ids
	assert.Equal(t, []spec.NewMedia{
		{MediaURL: "/uploads/posts/a.webp", MediaType: MediaTypeImage, Filename: "a.png", SortOrder: 0},
		{MediaURL: "/uploads/posts/b.mp4", MediaType: MediaTypeVideo, Filename: "b.mp4", SortOrder: 1, IsSpoiler: true},
	}, rows)
	require.Len(t, responses, 2)
	assert.Equal(t, 11, responses[0].ID)
	assert.Equal(t, 4, responses[0].Width)
	assert.Equal(t, 12, responses[1].ID)
	assert.True(t, responses[1].IsSpoiler)
	assert.Equal(t, uuid.Nil, rows[1].TargetID)
}

func TestTranscodeDone_ARowRemovedMeanwhileDiscardsTheTranscodedOutput(t *testing.T) {
	tests := []struct {
		name      string
		updateErr error
	}{
		{name: "the row was rolled back or deleted", updateErr: fmt.Errorf("update media url 7: %w", daoerr.ErrNotFound)},
		{name: "the update failed for another reason", updateErr: errors.New("db down")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a finished transcode whose media row update fails
			svc, storageSvc, dir := newMediaService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
			output, err := svc.Stage("clip.mp4", strings.NewReader("transcoded"))
			require.NoError(t, err)

			var transcodedKey string
			storageSvc.EXPECT().Store(mock.Anything, mock.Anything, output).RunAndReturn(func(_ context.Context, key string, _ string) error {
				transcodedKey = key
				return nil
			}).Once()
			storageSvc.EXPECT().Delete(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, keys ...string) error {
				assert.Equal(t, []string{transcodedKey}, keys)
				return nil
			}).Once()

			var updated spec.MediaURLUpdate
			updateURL := func(_ context.Context, s spec.MediaURLUpdate, _ ...*sql.Tx) error {
				updated = s
				return tc.updateErr
			}
			done := svc.transcodeDone(context.Background(), "posts", "/uploads/posts/raw.mp4", "/uploads/.staging/upload-1/raw.mp4", 7, updateURL, nil)

			// when
			done(output)

			// then only the transcoded copy is deleted, the raw upload stays, and the output is released
			assert.Equal(t, spec.MediaURLUpdate{ID: 7, URL: "/uploads/" + transcodedKey}, updated)
			assertStagingEmpty(t, dir)
		})
	}
}
