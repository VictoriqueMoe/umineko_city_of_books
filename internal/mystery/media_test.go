package mystery

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUploadAttachment_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadAttachment_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubNotAuthor(m, mid, userID)

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestUploadAttachment_FileTooBig(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(5)
	m.uploadSvc.EXPECT().SaveAttachment(mock.Anything, mock.Anything, int64(999), int64(5), mock.Anything).Return("", errors.New("too big"))

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 999, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadAttachment_DuplicateName(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return([]dto.MysteryAttachment{{FileName: "f.txt"}}, nil)

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadAttachment_SaveAttachmentError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(1024 * 1024)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
	m.uploadSvc.EXPECT().SaveAttachment(mock.Anything, mock.Anything, int64(10), int64(1024*1024), mock.Anything).Return("", errors.New("boom"))

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadAttachment_AddAttachmentError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(1024 * 1024)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
	m.uploadSvc.EXPECT().SaveAttachment(mock.Anything, mock.Anything, int64(10), int64(1024*1024), mock.Anything).Return("/uploads/x", nil)
	m.repo.EXPECT().AddAttachment(mock.Anything, mid, "/uploads/x", "f.txt", 10).Return(int64(0), errors.New("boom"))

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadAttachment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(1024 * 1024)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
	m.uploadSvc.EXPECT().SaveAttachment(mock.Anything, mock.Anything, int64(10), int64(1024*1024), mock.Anything).Return("/uploads/x", nil)
	m.repo.EXPECT().AddAttachment(mock.Anything, mid, "/uploads/x", "f.txt", 10).Return(int64(42), nil)

	// when
	got, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, 42, got.ID)
	assert.Equal(t, "/uploads/x", got.FileURL)
}

func TestDeleteAttachment_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.DeleteAttachment(context.Background(), 1, mid, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteAttachment_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubNotAuthor(m, mid, userID)

	// when
	err := svc.DeleteAttachment(context.Background(), 1, mid, userID)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestDeleteAttachment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
	m.repo.EXPECT().DeleteAttachment(mock.Anything, int64(1), mid).Return(errors.New("boom"))

	// when
	err := svc.DeleteAttachment(context.Background(), 1, mid, userID)

	// then
	require.Error(t, err)
}

func TestDeleteAttachment_OK_DeletesFile(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	attachments := []dto.MysteryAttachment{{ID: 1, FileURL: "/uploads/mystery-attachments/abc/f.txt"}}
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(attachments, nil)
	m.repo.EXPECT().DeleteAttachment(mock.Anything, int64(1), mid).Return(nil)
	m.uploadSvc.EXPECT().GetUploadDir().Return("/tmp/nonexistent-dir")

	// when
	err := svc.DeleteAttachment(context.Background(), 1, mid, userID)

	// then
	require.NoError(t, err)
}

func TestUploadMedia_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.UploadMedia(context.Background(), mid, userID, "image/png", "photo.png", 10, bytes.NewReader(nil), false)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubNotAuthor(m, mid, userID)

	// when
	_, err := svc.UploadMedia(context.Background(), mid, userID, "image/png", "photo.png", 10, bytes.NewReader(nil), false)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestDeleteMedia_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.DeleteMedia(context.Background(), 1, mid, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubNotAuthor(m, mid, userID)

	// when
	err := svc.DeleteMedia(context.Background(), 1, mid, userID)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestDeleteMedia_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().DeleteMedia(mock.Anything, int64(1), mid).Return("", errors.New("boom"))

	// when
	err := svc.DeleteMedia(context.Background(), 1, mid, userID)

	// then
	require.Error(t, err)
}

func TestDeleteMedia_OK_DeletesFile(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().DeleteMedia(mock.Anything, int64(1), mid).Return("/uploads/mysteries/x.png", nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/mysteries/x.png"}).Return()

	// when
	err := svc.DeleteMedia(context.Background(), 1, mid, userID)

	// then
	require.NoError(t, err)
}
