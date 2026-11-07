package mystery

import (
	"errors"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/daoerr"
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
	errMissingRow  = errors.Join(errors.New("no row"), daoerr.ErrNotFound)
	errStoreFailed = errors.New("store failed")
)

func stubAuthor(m *testMocks, id, authorID uuid.UUID) {
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
}

func stubNotAuthor(m *testMocks, id, actorID uuid.UUID) {
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, actorID, authz.PermEditAnyWork).Return(false)
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

func storedMysteryFile(filename string, sortOrder int, isSpoiler bool) upload.StoredMedia {
	return upload.StoredMedia{URL: "/uploads/mysteries/" + filename, MediaType: upload.MediaTypeImage, Filename: filename, SortOrder: sortOrder, IsSpoiler: isSpoiler}
}
