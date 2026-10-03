package media

import (
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	stagedVideo = "/uploads/.staging/upload-1/clip.mkv"
	storedVideo = "/uploads/posts/clip.mkv"
)

func newTestUploader(t *testing.T, sizeSetting *config.SiteSettingDef) (*Uploader, *mockuploadSvc) {
	t.Helper()
	uploadSvc := newMockuploadSvc(t)
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetInt(mock.Anything, sizeSetting).Return(1 << 20)

	processor := NewProcessor(1)
	require.NoError(t, processor.Shutdown(context.Background()))

	return NewUploader(uploadSvc, settingsSvc, processor), uploadSvc
}

func addRow(id int64, err error) AddFn {
	return func(string, string, string, string, int) (int64, error) {
		return id, err
	}
}

func TestSaveAndRecord_VideoHandsItsStagedCopyToTheTranscoder(t *testing.T) {
	// given a processor that refuses work, so the job's error path runs straight away
	uploader, uploadSvc := newTestUploader(t, config.SettingMaxVideoSize)
	uploadSvc.EXPECT().StageMedia(mock.Anything, MediaTypeVideo, "posts", mock.Anything, int64(10), int64(1<<20), mock.Anything).Return(stagedVideo, nil)
	uploadSvc.EXPECT().Store(mock.Anything, "posts", stagedVideo).Return(storedVideo, nil)
	uploadSvc.EXPECT().Discard(stagedVideo).Once()

	// when
	resp, err := uploader.SaveAndRecord(context.Background(), "posts", "video/x-matroska", "clip.mkv", 10, strings.NewReader("x"), false, addRow(7, nil), nil, nil)

	// then the transcoder received the staged file itself, nothing was downloaded back from storage
	require.NoError(t, err)
	assert.Equal(t, storedVideo, resp.MediaURL)
	assert.Equal(t, MediaTypeVideo, resp.MediaType)
	assert.Equal(t, 7, resp.ID)
}

func TestSaveAndRecord_VideoCleansUpWhenTheRowCannotBeAdded(t *testing.T) {
	// given
	uploader, uploadSvc := newTestUploader(t, config.SettingMaxVideoSize)
	uploadSvc.EXPECT().StageMedia(mock.Anything, MediaTypeVideo, "posts", mock.Anything, int64(10), int64(1<<20), mock.Anything).Return(stagedVideo, nil)
	uploadSvc.EXPECT().Store(mock.Anything, "posts", stagedVideo).Return(storedVideo, nil)
	uploadSvc.EXPECT().Discard(stagedVideo).Once()
	uploadSvc.EXPECT().Delete([]string{storedVideo}).Once()

	// when
	_, err := uploader.SaveAndRecord(context.Background(), "posts", "video/mp4", "clip.mp4", 10, strings.NewReader("x"), false, addRow(0, errors.New("db down")), nil, nil)

	// then
	require.EqualError(t, err, "db down")
}

func TestSaveAndRecord_VideoStoreFailureReleasesTheStagedCopy(t *testing.T) {
	// given
	uploader, uploadSvc := newTestUploader(t, config.SettingMaxVideoSize)
	uploadSvc.EXPECT().StageMedia(mock.Anything, MediaTypeVideo, "posts", mock.Anything, int64(10), int64(1<<20), mock.Anything).Return(stagedVideo, nil)
	uploadSvc.EXPECT().Store(mock.Anything, "posts", stagedVideo).Return("", errors.New("bucket gone"))
	uploadSvc.EXPECT().Discard(stagedVideo).Once()

	// when
	_, err := uploader.SaveAndRecord(context.Background(), "posts", "video/mp4", "clip.mp4", 10, strings.NewReader("x"), false, addRow(1, nil), nil, nil)

	// then no row was added for a file that was never stored
	require.EqualError(t, err, "bucket gone")
}

func TestSaveAndRecord_ImageAndAudioAreSavedDirectly(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		sizeSetting *config.SiteSettingDef
		mediaType   string
	}{
		{name: "image", contentType: "image/png", sizeSetting: config.SettingMaxImageSize, mediaType: MediaTypeImage},
		{name: "audio", contentType: "audio/mpeg", sizeSetting: config.SettingMaxAudioSize, mediaType: MediaTypeAudio},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			uploader, uploadSvc := newTestUploader(t, tc.sizeSetting)
			if tc.mediaType == MediaTypeImage {
				uploadSvc.EXPECT().SaveImage(mock.Anything, "posts", mock.Anything, int64(10), int64(1<<20), mock.Anything).Return("/uploads/posts/a.webp", nil)
			} else {
				uploadSvc.EXPECT().SaveAudio(mock.Anything, "posts", mock.Anything, int64(10), int64(1<<20), mock.Anything).Return("/uploads/posts/a.webp", nil)
			}

			// when
			resp, err := uploader.SaveAndRecord(context.Background(), "posts", tc.contentType, "a", 10, strings.NewReader("x"), false, addRow(3, nil), nil, nil)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.mediaType, resp.MediaType)
		})
	}
}
