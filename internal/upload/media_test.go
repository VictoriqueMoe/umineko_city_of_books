package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/settings"

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

func mediaSettings(t *testing.T, maxImageSize int, maxPixels int) settings.Service {
	t.Helper()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(maxImageSize).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxVideoSize).Return(testMediaLimit).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxAudioSize).Return(testMediaLimit).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(maxPixels).Maybe()

	return settingsSvc
}

func recordingTarget(records *[]MediaRecord) MediaTarget {
	return MediaTarget{
		SubDir: "posts",
		Record: func(r MediaRecord) (int64, error) {
			*records = append(*records, r)

			return int64(len(*records)), nil
		},
	}
}

func recordAndReturn(urlPath string) func(context.Context, string, string, string, int64, io.Reader, bool, media.AddFn, media.UpdateURLFn, media.UpdateURLFn) (*dto.PostMediaResponse, error) {
	return func(_ context.Context, _ string, _ string, filename string, _ int64, _ io.Reader, isSpoiler bool, addFn media.AddFn, _ media.UpdateURLFn, _ media.UpdateURLFn) (*dto.PostMediaResponse, error) {
		id, err := addFn(urlPath, media.MediaTypeImage, "", filename, 0)
		if err != nil {
			return nil, err
		}

		return &dto.PostMediaResponse{ID: int(id), MediaURL: urlPath, MediaType: media.MediaTypeImage, Filename: filename, IsSpoiler: isSpoiler}, nil
	}
}

func TestValidateMedia(t *testing.T) {
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
			name:         "a well-formed png passes",
			files:        []FileUpload{memoryUpload("image/png", "one.png", png, false)},
			maxImageSize: testMediaLimit,
			maxPixels:    bounds.FallbackMaxImagePixels,
		},
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
			name: "one bad file rejects the whole batch",
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
			// given
			settingsSvc := mediaSettings(t, tt.maxImageSize, tt.maxPixels)

			// when
			_, err := ValidateMedia(t.Context(), settingsSvc, tt.files)

			// then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidatedMediaSave_RefusedBatchWritesNothing(t *testing.T) {
	// given a batch whose second file is over the limit, and stores that fail on any call
	png := mustPNGBytes()
	settingsSvc := mediaSettings(t, len(png), bounds.FallbackMaxImagePixels)
	recorder := NewMockMediaRecorder(t)
	deleter := NewMockService(t)

	files := []FileUpload{
		memoryUpload("image/png", "one.png", png, false),
		memoryUpload("image/png", "two.png", append(bytes.Clone(png), 0), false),
	}

	var records []MediaRecord

	// when the batch is validated and whatever came back is saved
	validated, err := ValidateMedia(t.Context(), settingsSvc, files)
	saved, saveErr := validated.Save(t.Context(), recorder, deleter, recordingTarget(&records))

	// then the refusal is reported and nothing was stored or recorded
	require.ErrorIs(t, err, ErrFileTooLarge)
	require.NoError(t, saveErr)
	assert.Nil(t, saved)
	assert.Empty(t, records)
}

func TestValidatedMediaSave_RecordsEachFileInDisplayOrder(t *testing.T) {
	// given two valid images, the second marked as a spoiler
	png := mustPNGBytes()
	settingsSvc := mediaSettings(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	recorder := NewMockMediaRecorder(t)
	deleter := NewMockService(t)
	targetID := uuid.New()

	recorder.EXPECT().
		SaveAndRecord(mock.Anything, "posts", "image/png", "one.png", int64(len(png)), mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(recordAndReturn("/uploads/posts/one.png")).
		Once()
	recorder.EXPECT().
		SaveAndRecord(mock.Anything, "posts", "image/png", "two.png", int64(len(png)), mock.Anything, true, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(recordAndReturn("/uploads/posts/two.png")).
		Once()

	validated, err := ValidateMedia(t.Context(), settingsSvc, []FileUpload{
		memoryUpload("image/png", "one.png", png, false),
		memoryUpload("image/png", "two.png", png, true),
	})
	require.NoError(t, err)

	var records []MediaRecord

	// when the batch is saved
	saved, err := validated.Save(t.Context(), recorder, deleter, recordingTarget(&records))

	// then each file is recorded at its position with its own spoiler flag and dimensions
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, spec.NewMedia{
		TargetID:  targetID,
		MediaURL:  "/uploads/posts/one.png",
		MediaType: media.MediaTypeImage,
		Filename:  "one.png",
		SortOrder: 0,
	}, records[0].NewMedia(targetID))
	assert.Equal(t, spec.NewMedia{
		TargetID:  targetID,
		MediaURL:  "/uploads/posts/two.png",
		MediaType: media.MediaTypeImage,
		Filename:  "two.png",
		SortOrder: 1,
		IsSpoiler: true,
	}, records[1].NewMedia(targetID))
	assert.Equal(t, 2, records[1].Width)
	assert.Equal(t, 2, records[1].Height)

	require.Len(t, saved, 2)
	assert.Equal(t, "/uploads/posts/one.png", saved[0].MediaURL)
	assert.Equal(t, 0, saved[0].SortOrder)
	assert.False(t, saved[0].IsSpoiler)
	assert.Equal(t, "/uploads/posts/two.png", saved[1].MediaURL)
	assert.Equal(t, 1, saved[1].SortOrder)
	assert.True(t, saved[1].IsSpoiler)
	assert.Equal(t, 2, saved[1].Width)
	assert.Equal(t, 2, saved[1].Height)
}

func TestValidatedMediaSave_FailureDiscardsWhatWasStored(t *testing.T) {
	tests := []struct {
		name        string
		failOn      string
		wantDeleted []string
	}{
		{
			name:   "a first-file failure has nothing to discard",
			failOn: "one.png",
		},
		{
			name:        "a second-file failure discards the first stored object",
			failOn:      "two.png",
			wantDeleted: []string{"/uploads/posts/one.png"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given two valid images where the store fails on one of them
			png := mustPNGBytes()
			settingsSvc := mediaSettings(t, testMediaLimit, bounds.FallbackMaxImagePixels)
			recorder := NewMockMediaRecorder(t)
			deleter := NewMockService(t)

			for _, name := range []string{"one.png", "two.png"} {
				call := recorder.EXPECT().
					SaveAndRecord(mock.Anything, "posts", "image/png", name, mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything)
				if name == tt.failOn {
					call.Return(nil, errStoreFailed).Once()
					break
				}

				call.RunAndReturn(recordAndReturn("/uploads/posts/" + name)).Once()
			}

			if tt.wantDeleted != nil {
				deleter.EXPECT().Delete(tt.wantDeleted).Return().Once()
			}

			validated, err := ValidateMedia(t.Context(), settingsSvc, []FileUpload{
				memoryUpload("image/png", "one.png", png, false),
				memoryUpload("image/png", "two.png", png, false),
			})
			require.NoError(t, err)

			var records []MediaRecord

			// when the batch is saved
			saved, err := validated.Save(t.Context(), recorder, deleter, recordingTarget(&records))

			// then the store failure is returned and only the objects already written are discarded
			require.ErrorIs(t, err, errStoreFailed)
			assert.Nil(t, saved)
		})
	}
}
