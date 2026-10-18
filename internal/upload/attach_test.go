package upload

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func recordAndTouchUpdates(urlPath string) func(context.Context, string, string, string, int64, io.Reader, bool, media.AddFn, media.UpdateURLFn, media.UpdateURLFn) (*dto.PostMediaResponse, error) {
	return func(ctx context.Context, _ string, _ string, filename string, _ int64, _ io.Reader, isSpoiler bool, addFn media.AddFn, updateURL media.UpdateURLFn, updateThumb media.UpdateURLFn) (*dto.PostMediaResponse, error) {
		id, err := addFn(urlPath, media.MediaTypeImage, "", filename, 0)
		if err != nil {
			return nil, err
		}

		if err := updateURL(ctx, spec.MediaURLUpdate{ID: id, URL: urlPath + ".mp4"}); err != nil {
			return nil, err
		}

		if err := updateThumb(ctx, spec.MediaURLUpdate{ID: id, URL: urlPath + ".jpg"}); err != nil {
			return nil, err
		}

		return &dto.PostMediaResponse{ID: int(id), MediaURL: urlPath, MediaType: media.MediaTypeImage, Filename: filename, IsSpoiler: isSpoiler}, nil
	}
}

func TestSaveToEntity_RecordsEveryFileAgainstTheEntityThroughTheEntityMediaStore(t *testing.T) {
	// given
	recorder := NewMockMediaRecorder(t)
	deleter := NewMockService(t)
	store := repository.NewMockPostRepository(t)
	postID := uuid.New()
	validated := ValidatedMedia{files: []FileUpload{memoryUpload("image/png", "one.png", mustPNGBytes(), false)}}

	recorder.EXPECT().
		SaveAndRecord(mock.Anything, "posts", "image/png", "one.png", mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(recordAndTouchUpdates("/uploads/posts/one.png")).
		Once()
	store.EXPECT().
		AddMedia(mock.Anything, spec.NewMedia{TargetID: postID, MediaURL: "/uploads/posts/one.png", MediaType: media.MediaTypeImage, Filename: "one.png"}).
		Return(int64(9), nil).
		Once()
	store.EXPECT().UpdateMediaURL(mock.Anything, spec.MediaURLUpdate{ID: 9, URL: "/uploads/posts/one.png.mp4"}).Return(nil).Once()
	store.EXPECT().UpdateMediaThumbnail(mock.Anything, spec.MediaURLUpdate{ID: 9, URL: "/uploads/posts/one.png.jpg"}).Return(nil).Once()

	// when
	err := validated.SaveToEntity(context.Background(), recorder, deleter, "posts", store, postID)

	// then
	require.NoError(t, err)
}

func TestDiscardOwned(t *testing.T) {
	cause := errors.New("store failed")
	rollbackErr := errors.New("db down")

	tests := []struct {
		name        string
		paths       []string
		rollbackErr error
		hangUp      bool
		wantErrs    []error
		wantText    string
	}{
		{name: "the entity and every stored object are removed", paths: []string{"/uploads/posts/one.png"}, wantErrs: []error{cause}},
		{name: "an entity with no stored objects is still removed", wantErrs: []error{cause}},
		{name: "a failed rollback is reported alongside the cause", rollbackErr: rollbackErr, wantErrs: []error{cause, rollbackErr}, wantText: "roll back post"},
		{name: "a client hanging up still lets the rollback run", paths: []string{"/uploads/posts/one.png"}, hangUp: true, wantErrs: []error{cause}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			store := repository.NewMockPostRepository(t)
			deleter := NewMockService(t)
			deletion := spec.OwnedDeletion{ID: uuid.New(), UserID: uuid.New()}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.hangUp {
				cancel()
			}

			var rollbackCtxErr error
			store.EXPECT().
				DeleteWithMedia(mock.Anything, deletion).
				Run(func(ctx context.Context, _ spec.OwnedDeletion, _ ...*sql.Tx) { rollbackCtxErr = ctx.Err() }).
				Return(tc.paths, tc.rollbackErr).
				Once()
			if len(tc.paths) == 0 {
				deleter.EXPECT().Delete().Return().Once()
			} else {
				deleter.EXPECT().Delete(tc.paths).Return().Once()
			}

			// when
			err := DiscardOwned(ctx, store, deleter, "post", deletion, cause)

			// then
			for _, want := range tc.wantErrs {
				require.ErrorIs(t, err, want)
			}
			if tc.wantText != "" {
				assert.Contains(t, err.Error(), tc.wantText)
			}
			require.NoError(t, rollbackCtxErr)
		})
	}
}

func TestValidateAttachmentFile(t *testing.T) {
	pdf := []byte("%PDF-1.4\n%case notes\n")

	tests := []struct {
		name    string
		file    FileUpload
		maxSize int64
		wantErr error
	}{
		{name: "a pdf within the limit passes", file: memoryUpload("application/pdf", "notes.pdf", pdf, false), maxSize: 1024},
		{name: "a file over the limit is refused before it is read", file: memoryUpload("application/pdf", "notes.pdf", pdf, false), maxSize: 1, wantErr: ErrFileTooLarge},
		{name: "html is refused whatever it claims to be", file: memoryUpload("application/pdf", "page.pdf", []byte("<!DOCTYPE html><html><body>hi</body></html>"), false), maxSize: 1024, wantErr: ErrInvalidAttachmentType},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			err := ValidateAttachmentFile(tc.file, tc.maxSize)

			// then
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
