package upload

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testMediaLimit = 1024 * 1024
	finishTimeout  = 5 * time.Second
)

var (
	errStoreFailed = errors.New("store failed")
)

func memoryUpload(contentType string, filename string, data []byte) FileUpload {
	return FileUpload{
		ContentType: contentType,
		Filename:    filename,
		Size:        int64(len(data)),
		Open: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}
}

func newPendingService(t *testing.T, maxImageSize int, maxPixels int) (*service, *storage.MockService, *repository.MockPendingUploadRepository, string) {
	t.Helper()

	dir := t.TempDir()
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(dir).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(maxImageSize).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxVideoSize).Return(testMediaLimit).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxAudioSize).Return(testMediaLimit).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(testMediaLimit).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(maxPixels).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxQueuedUploadsPerUser).Return(testQueueLimit).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxWaitingUploadsPerUser).Return(testWaitingLimit).Maybe()

	storageSvc := storage.NewMockService(t)
	storedFiles := repository.NewMockStoredFileRepository(t)
	storedFiles.EXPECT().ListByPrefix(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	pending := repository.NewMockPendingUploadRepository(t)

	return NewService(settingsSvc, storageSvc, storedFiles, pending, ws.NewHub()).(*service), storageSvc, pending, dir
}

func TestBeginMediaUpload_RefusesBeforeAnythingIsRecorded(t *testing.T) {
	png := mustPNGBytes()
	notAnImage := []byte("this is plainly not an image at all")

	tests := []struct {
		name         string
		file         FileUpload
		maxImageSize int
		maxPixels    int
		wantErr      error
	}{
		{name: "an image over the size limit is refused", file: memoryUpload("image/png", "one.png", png), maxImageSize: 1, maxPixels: bounds.FallbackMaxImagePixels, wantErr: ErrFileTooLarge},
		{name: "bytes that are not an image are refused", file: memoryUpload("image/png", "one.png", notAnImage), maxImageSize: testMediaLimit, maxPixels: bounds.FallbackMaxImagePixels, wantErr: ErrInvalidFileType},
		{name: "an image declared as audio is refused", file: memoryUpload("audio/mpeg", "one.mp3", png), maxImageSize: testMediaLimit, maxPixels: bounds.FallbackMaxImagePixels, wantErr: ErrInvalidAudioType},
		{name: "an image declared as video is refused", file: memoryUpload("video/mp4", "one.mp4", png), maxImageSize: testMediaLimit, maxPixels: bounds.FallbackMaxImagePixels, wantErr: ErrInvalidVideoType},
		{name: "an image past the pixel budget is refused", file: memoryUpload("image/png", "one.png", png), maxImageSize: testMediaLimit, maxPixels: 3, wantErr: bounds.ErrImageBounds},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given strict storage and pending mocks that fail the test on any write
			svc, _, _, dir := newPendingService(t, tt.maxImageSize, tt.maxPixels)

			// when
			got, err := svc.BeginMediaUpload(t.Context(), uuid.New(), tt.file)

			// then
			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, got)
			assertStagingEmpty(t, dir)
		})
	}
}

func TestBeginMediaUpload_RecordsTheUploadAndFinishesItInTheBackground(t *testing.T) {
	// given a valid picture
	svc, storageSvc, pending, _ := newPendingService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	objects := captureStore(t, storageSvc)
	userID := uuid.New()

	var recorded spec.NewPendingUpload
	pending.EXPECT().Insert(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, s spec.NewPendingUpload, _ ...*sql.Tx) error {
		recorded = s
		return nil
	}).Once()

	ready := make(chan spec.ProcessedUpload, 1)
	pending.EXPECT().MarkReady(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, s spec.ProcessedUpload, _ ...*sql.Tx) (bool, error) {
		ready <- s
		return true, nil
	}).Once()

	// when
	got, err := svc.BeginMediaUpload(t.Context(), userID, memoryUpload("image/png", "one.png", mustPNGBytes()))

	// then the upload is recorded as processing before the request returns
	require.NoError(t, err)
	assert.Equal(t, dto.PendingUploadResponse{ID: recorded.ID, MediaType: MediaTypeImage, Filename: "one.png", Status: string(model.PendingUploadProcessing)}, *got)
	assert.Equal(t, spec.NewPendingUpload{ID: recorded.ID, UserID: userID, MediaType: MediaTypeImage, Filename: "one.png", FileSize: int64(len(mustPNGBytes()))}, recorded)

	// and the stored file is recorded once the background work is done
	select {
	case processed := <-ready:
		require.Len(t, *objects, 1)
		assert.Equal(t, spec.ProcessedUpload{ID: recorded.ID, MediaURL: "/uploads/" + (*objects)[0].key, Width: 2, Height: 2}, processed)
		assert.True(t, strings.HasPrefix((*objects)[0].key, mediaSubDir+"/"))
		assert.Zero(t, admittedUploads(&svc.userQueues, userID), "a processed upload must give its place back")
	case <-time.After(finishTimeout):
		t.Fatal("the upload was never marked ready")
	}
}

func TestBeginMediaUpload_RefusesBeforeStagingWhenTheBacklogIsFull(t *testing.T) {
	// given a user whose running and waiting places are all taken, and strict storage and pending mocks
	svc, _, _, dir := newPendingService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	userID := uuid.New()
	fillBacklog(t, &svc.userQueues, userID)

	// when
	got, err := svc.BeginMediaUpload(t.Context(), userID, memoryUpload("image/png", "one.png", mustPNGBytes()))

	// then
	require.ErrorIs(t, err, ErrTooManyUploads)
	assert.Nil(t, got)
	assertStagingEmpty(t, dir)
}

func TestBeginMediaUpload_AFailedRecordReleasesTheStagedCopy(t *testing.T) {
	// given a database that refuses the record
	svc, _, pending, dir := newPendingService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	pending.EXPECT().Insert(mock.Anything, mock.Anything).Return(errStoreFailed).Once()
	userID := uuid.New()

	// when
	got, err := svc.BeginMediaUpload(t.Context(), userID, memoryUpload("image/png", "one.png", mustPNGBytes()))

	// then
	require.ErrorIs(t, err, errStoreFailed)
	assert.Nil(t, got)
	assertStagingEmpty(t, dir)
	assert.Zero(t, admittedUploads(&svc.userQueues, userID), "a failed upload must give its place back")
}

func TestBeginFileUpload_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		file    FileUpload
		wantErr error
	}{
		{name: "a file over the limit is refused before it is read", file: memoryUpload("application/pdf", "notes.pdf", bytes.Repeat([]byte("x"), testMediaLimit+1)), wantErr: ErrFileTooLarge},
		{name: "html is refused whatever it claims to be", file: memoryUpload("application/pdf", "page.pdf", []byte("<!DOCTYPE html><html><body>hi</body></html>")), wantErr: ErrInvalidAttachmentType},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given strict storage and pending mocks
			svc, _, _, dir := newPendingService(t, testMediaLimit, bounds.FallbackMaxImagePixels)

			// when
			got, err := svc.BeginFileUpload(t.Context(), uuid.New(), tc.file)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
			assertStagingEmpty(t, dir)
		})
	}
}

func TestBeginFileUpload_StoresTheDocumentUnderItsSniffedExtension(t *testing.T) {
	// given a pdf
	svc, storageSvc, pending, _ := newPendingService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	objects := captureStore(t, storageSvc)
	userID := uuid.New()
	pending.EXPECT().Insert(mock.Anything, mock.MatchedBy(func(s spec.NewPendingUpload) bool {
		return s.UserID == userID && s.MediaType == MediaTypeFile && s.Filename == "notes.pdf"
	})).Return(nil).Once()

	ready := make(chan spec.ProcessedUpload, 1)
	pending.EXPECT().MarkReady(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, s spec.ProcessedUpload, _ ...*sql.Tx) (bool, error) {
		ready <- s
		return true, nil
	}).Once()

	// when
	got, err := svc.BeginFileUpload(t.Context(), userID, memoryUpload("application/pdf", "notes.pdf", pdfMagic))

	// then
	require.NoError(t, err)
	assert.Equal(t, MediaTypeFile, got.MediaType)
	select {
	case processed := <-ready:
		require.Len(t, *objects, 1)
		assert.True(t, strings.HasPrefix((*objects)[0].key, filesSubDir+"/"))
		assert.True(t, strings.HasSuffix((*objects)[0].key, ".pdf"))
		assert.Equal(t, pdfMagic, (*objects)[0].data)
		assert.Equal(t, "/uploads/"+(*objects)[0].key, processed.MediaURL)
	case <-time.After(finishTimeout):
		t.Fatal("the upload was never marked ready")
	}
}

func TestFinish(t *testing.T) {
	processed := spec.ProcessedUpload{MediaURL: "/uploads/media/a.webp", ThumbnailURL: "/uploads/media/thumb_a.webp"}

	tests := []struct {
		name       string
		processErr error
		markErr    error
		marked     bool
		expect     func(storageSvc *storage.MockService, pending *repository.MockPendingUploadRepository, pendingUpload spec.NewPendingUpload)
	}{
		{
			name:   "a processed upload is marked ready and keeps its files",
			marked: true,
		},
		{
			name:       "a failed process marks the upload failed",
			processErr: errStoreFailed,
			expect: func(_ *storage.MockService, pending *repository.MockPendingUploadRepository, p spec.NewPendingUpload) {
				pending.EXPECT().MarkFailed(mock.Anything, spec.UserUpload{ID: p.ID, UserID: p.UserID}).Return(true, nil).Once()
			},
		},
		{
			name: "an upload removed while it was processing has its files deleted",
			expect: func(storageSvc *storage.MockService, _ *repository.MockPendingUploadRepository, _ spec.NewPendingUpload) {
				storageSvc.EXPECT().Delete(mock.Anything, []string{"media/a.webp"}).Return(nil).Once()
				storageSvc.EXPECT().Delete(mock.Anything, []string{"media/thumb_a.webp"}).Return(nil).Once()
			},
		},
		{
			name:    "a failed ready record deletes the files and marks the upload failed",
			markErr: errStoreFailed,
			expect: func(storageSvc *storage.MockService, pending *repository.MockPendingUploadRepository, p spec.NewPendingUpload) {
				storageSvc.EXPECT().Delete(mock.Anything, []string{"media/a.webp"}).Return(nil).Once()
				storageSvc.EXPECT().Delete(mock.Anything, []string{"media/thumb_a.webp"}).Return(nil).Once()
				pending.EXPECT().MarkFailed(mock.Anything, spec.UserUpload{ID: p.ID, UserID: p.UserID}).Return(true, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a staged upload
			svc, storageSvc, pending, dir := newPendingService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
			staged, err := svc.Stage("a.png", strings.NewReader("x"))
			require.NoError(t, err)
			pendingUpload := spec.NewPendingUpload{ID: uuid.New(), UserID: uuid.New(), MediaType: MediaTypeImage, Filename: "a.png"}

			if tc.processErr == nil {
				ready := processed
				ready.ID = pendingUpload.ID
				pending.EXPECT().MarkReady(mock.Anything, ready).Return(tc.marked, tc.markErr).Once()
			}
			if tc.expect != nil {
				tc.expect(storageSvc, pending, pendingUpload)
			}

			// when
			svc.finish(context.Background(), pendingUpload, staged, func(context.Context) (spec.ProcessedUpload, error) {
				return processed, tc.processErr
			})

			// then the staged copy is always released, and the strict mocks prove which files were kept
			assertStagingEmpty(t, dir)
		})
	}
}

func TestStoreVideo_WithoutAProcessorStoresTheRawFileOnce(t *testing.T) {
	// given a staged video on a service that has no media processor
	svc, storageSvc, _, _ := newPendingService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	objects := captureStore(t, storageSvc)
	staged, err := svc.Stage("clip.mp4", bytes.NewReader(mp4Magic))
	require.NoError(t, err)

	// when
	got, err := svc.processMedia(t.Context(), MediaTypeVideo, staged)

	// then the raw file goes to storage once, with no thumbnail
	require.NoError(t, err)
	require.Len(t, *objects, 1)
	assert.Equal(t, mp4Magic, (*objects)[0].data)
	assert.Equal(t, spec.ProcessedUpload{MediaURL: "/uploads/" + (*objects)[0].key}, got)
}

func TestRemoveUpload_DeletesTheRowAndItsFiles(t *testing.T) {
	// given an upload that finished with a thumbnail
	svc, storageSvc, pending, _ := newPendingService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	userID, id := uuid.New(), uuid.New()
	pending.EXPECT().Delete(mock.Anything, spec.UserUpload{ID: id, UserID: userID}).
		Return([]model.PendingUploadRow{{ID: id, MediaURL: "/uploads/media/a.mp4", ThumbnailURL: "/uploads/media/thumb_a.webp"}}, nil).Once()
	storageSvc.EXPECT().Delete(mock.Anything, []string{"media/a.mp4"}).Return(nil).Once()
	storageSvc.EXPECT().Delete(mock.Anything, []string{"media/thumb_a.webp"}).Return(nil).Once()

	// when
	err := svc.RemoveUpload(t.Context(), userID, id)

	// then
	require.NoError(t, err)
}

func TestExpireUploads_DeletesEveryUploadOlderThanItsLifetime(t *testing.T) {
	// given two expired uploads, one still processing
	svc, storageSvc, pending, _ := newPendingService(t, testMediaLimit, bounds.FallbackMaxImagePixels)
	before := time.Now()
	pending.EXPECT().DeleteExpired(mock.Anything, mock.MatchedBy(func(cutoff time.Time) bool {
		return !cutoff.After(time.Now().Add(-PendingUploadLifetime)) && !cutoff.Before(before.Add(-PendingUploadLifetime))
	})).Return([]model.PendingUploadRow{{MediaURL: "/uploads/media/a.webp"}, {}}, nil).Once()
	storageSvc.EXPECT().Delete(mock.Anything, []string{"media/a.webp"}).Return(nil).Once()

	// when
	n, err := svc.ExpireUploads(t.Context())

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestClaims(t *testing.T) {
	userID := uuid.New()
	refs := []dto.UploadRef{{ID: uuid.New()}, {ID: uuid.New(), IsSpoiler: true}}

	tests := []struct {
		name string
		got  spec.ClaimedUploads
		want spec.ClaimedUploads
	}{
		{
			name: "a media claim accepts only pictures, videos and audio, in the order they were attached",
			got:  MediaClaim(userID, refs),
			want: spec.ClaimedUploads{
				UserID:     userID,
				Uploads:    []spec.UploadRef{{ID: refs[0].ID}, {ID: refs[1].ID, IsSpoiler: true}},
				MediaTypes: []string{MediaTypeImage, MediaTypeVideo, MediaTypeAudio},
				MaxAge:     PendingUploadLifetime,
			},
		},
		{
			name: "a file claim accepts only documents",
			got:  FileClaim(userID, refs[:1]),
			want: spec.ClaimedUploads{UserID: userID, Uploads: []spec.UploadRef{{ID: refs[0].ID}}, MediaTypes: []string{MediaTypeFile}, MaxAge: PendingUploadLifetime},
		},
		{
			name: "a claim with nothing attached claims nothing",
			got:  MediaClaim(userID, nil),
			want: spec.ClaimedUploads{UserID: userID, MediaTypes: []string{MediaTypeImage, MediaTypeVideo, MediaTypeAudio}, MaxAge: PendingUploadLifetime},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given the refs a client sent

			// when the service builds its claim

			// then
			assert.Equal(t, tc.want, tc.got)
		})
	}
}
