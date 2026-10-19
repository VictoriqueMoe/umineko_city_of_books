package mystery

import (
	"context"
	"errors"
	"io"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/upload/uploadtest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

const (
	testMediaLimit = 1024 * 1024
)

var (
	errMissingRow  = errors.Join(errors.New("no row"), dao.ErrNotFound)
	errStoreFailed = errors.New("store failed")
)

func stubAuthor(m *testMocks, id, authorID uuid.UUID) {
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
}

func stubNotAuthor(m *testMocks, id, actorID uuid.UUID) {
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, actorID, authz.PermEditAnyTheory).Return(false)
}

func stubActor(m *testMocks, userID uuid.UUID, displayName string) {
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: displayName}, nil)
}

func stubMentionOf(m *testMocks, actorID, mentionedID uuid.UUID, username string) {
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{username}).Return([]model.User{{ID: mentionedID}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, actorID, mentionedID).Return(false, nil)
}

func pdfUpload(filename string) upload.FileUpload {
	return uploadtest.MemoryUpload("application/pdf", filename, []byte("%PDF-1.4\n%case notes\n"), false)
}

func expectAttachmentLimit(m *testMocks, maxSize int) {
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(maxSize).Maybe()
}

func expectUnlinked(m *testMocks, paths []string) {
	if len(paths) == 0 {
		m.uploadSvc.EXPECT().Delete().Return()

		return
	}

	m.uploadSvc.EXPECT().Delete(paths).Return()
}

func storeAs(urlPath string) func(context.Context, string, string, string, int64, io.Reader, bool, media.AddFn, media.UpdateURLFn, media.UpdateURLFn) (*dto.PostMediaResponse, error) {
	return func(_ context.Context, _ string, _ string, filename string, _ int64, _ io.Reader, isSpoiler bool, addFn media.AddFn, _ media.UpdateURLFn, _ media.UpdateURLFn) (*dto.PostMediaResponse, error) {
		id, err := addFn(urlPath, media.MediaTypeImage, "", filename, 0)
		if err != nil {
			return nil, err
		}

		return &dto.PostMediaResponse{ID: int(id), MediaURL: urlPath, MediaType: media.MediaTypeImage, Filename: filename, IsSpoiler: isSpoiler}, nil
	}
}

func expectStored(recorder *upload.MockMediaRecorder, f upload.FileUpload, urlPath string) {
	recorder.EXPECT().
		SaveAndRecord(mock.Anything, "mysteries", f.ContentType, f.Filename, f.Size, mock.Anything, f.IsSpoiler, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(storeAs(urlPath)).
		Once()
}
