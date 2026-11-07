package mystery

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAttachFile_Refusals(t *testing.T) {
	cases := []struct {
		name   string
		expect func(m *testMocks, mid, userID, uploadID uuid.UUID)
		want   error
	}{
		{
			name: "a missing mystery is not found",
			expect: func(m *testMocks, mid, _, _ uuid.UUID) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errMissingRow)
			},
			want: ErrNotFound,
		},
		{
			name: "only the author or an editor may attach",
			expect: func(m *testMocks, mid, userID, _ uuid.UUID) {
				stubNotAuthor(m, mid, userID)
			},
			want: ErrNotAuthor,
		},
		{
			name: "a file named like one already attached is refused",
			expect: func(m *testMocks, mid, userID, uploadID uuid.UUID) {
				stubAuthor(m, mid, userID)
				m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return([]dto.MysteryAttachment{{FileName: "f.txt"}}, nil)
				m.uploadSvc.EXPECT().UploadStatuses(mock.Anything, userID, []uuid.UUID{uploadID}).Return([]dto.PendingUploadResponse{{ID: uploadID, Filename: "f.txt"}}, nil)
			},
			want: ErrDuplicateAttachment,
		},
		{
			name: "an upload that cannot be claimed is surfaced",
			expect: func(m *testMocks, mid, userID, uploadID uuid.UUID) {
				stubAuthor(m, mid, userID)
				m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
				m.uploadSvc.EXPECT().UploadStatuses(mock.Anything, userID, []uuid.UUID{uploadID}).Return(nil, nil)
				m.repo.EXPECT().
					AttachFile(mock.Anything, spec.MediaAttachment{TargetID: mid, Uploads: upload.FileClaim(userID, []dto.UploadRef{{ID: uploadID}})}).
					Return(nil, daoerr.ErrUploadUnavailable)
			},
			want: daoerr.ErrUploadUnavailable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			mid, userID, uploadID := uuid.New(), uuid.New(), uuid.New()
			tc.expect(m, mid, userID, uploadID)

			// when
			got, err := svc.AttachFile(context.Background(), mid, userID, dto.UploadRef{ID: uploadID})

			// then
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, got)
		})
	}
}

func TestAttachFile_OK(t *testing.T) {
	// given a finished upload whose name is new to the mystery
	svc, m := newTestService(t)
	mid, userID, uploadID := uuid.New(), uuid.New(), uuid.New()
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return([]dto.MysteryAttachment{{FileName: "other.txt"}}, nil)
	m.uploadSvc.EXPECT().UploadStatuses(mock.Anything, userID, []uuid.UUID{uploadID}).Return([]dto.PendingUploadResponse{{ID: uploadID, Filename: "f.txt"}}, nil)
	m.repo.EXPECT().
		AttachFile(mock.Anything, spec.MediaAttachment{TargetID: mid, Uploads: upload.FileClaim(userID, []dto.UploadRef{{ID: uploadID}})}).
		Return(&dto.MysteryAttachment{ID: 42, FileURL: "/uploads/attachments/x.txt", FileName: "f.txt", FileSize: 10}, nil)

	// when
	got, err := svc.AttachFile(context.Background(), mid, userID, dto.UploadRef{ID: uploadID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 42, got.ID)
	assert.Equal(t, "/uploads/attachments/x.txt", got.FileURL)
}

func TestDeleteAttachment_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errMissingRow)

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
	m.repo.EXPECT().DeleteAttachment(mock.Anything, spec.MysteryAttachmentDeletion{ID: 1, MysteryID: mid}).Return(errors.New("boom"))

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
	m.repo.EXPECT().DeleteAttachment(mock.Anything, spec.MysteryAttachmentDeletion{ID: 1, MysteryID: mid}).Return(nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/mystery-attachments/abc/f.txt"}).Return()

	// when
	err := svc.DeleteAttachment(context.Background(), 1, mid, userID)

	// then
	require.NoError(t, err)
}

func TestMysteryFiles_AFailedExistingFilesLookupIsSurfaced(t *testing.T) {
	cases := []struct {
		name   string
		expect func(m *testMocks, mid uuid.UUID, boom error)
		call   func(s *service, mid, userID uuid.UUID) error
	}{
		{
			name: "an attachment upload is refused instead of skipping the duplicate name check",
			expect: func(m *testMocks, mid uuid.UUID, boom error) {
				m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, boom)
			},
			call: func(s *service, mid, userID uuid.UUID) error {
				_, err := s.AttachFile(context.Background(), mid, userID, dto.UploadRef{ID: uuid.New()})
				return err
			},
		},
		{
			name: "an attachment delete is refused instead of leaving its file behind",
			expect: func(m *testMocks, mid uuid.UUID, boom error) {
				m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, boom)
			},
			call: func(s *service, mid, userID uuid.UUID) error {
				return s.DeleteAttachment(context.Background(), 1, mid, userID)
			},
		},
		{
			name: "a media upload is refused instead of reusing a sort position",
			expect: func(m *testMocks, mid uuid.UUID, boom error) {
				m.repo.EXPECT().GetMedia(mock.Anything, mid).Return(nil, boom)
			},
			call: func(s *service, mid, userID uuid.UUID) error {
				_, err := s.AttachMedia(context.Background(), mid, userID, dto.UploadRef{ID: uuid.New()})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			mid := uuid.New()
			userID := uuid.New()
			boom := errors.New("boom")
			stubAuthor(m, mid, userID)
			tc.expect(m, mid, boom)

			// when
			err := tc.call(svc, mid, userID)

			// then
			require.ErrorIs(t, err, boom)
		})
	}
}

func TestAttachMedia_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errMissingRow)

	// when
	_, err := svc.AttachMedia(context.Background(), mid, userID, dto.UploadRef{ID: uuid.New()})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestAttachMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	stubNotAuthor(m, mid, userID)

	// when
	_, err := svc.AttachMedia(context.Background(), mid, userID, dto.UploadRef{ID: uuid.New()})

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestAttachMedia_PlacesTheUploadAfterTheExistingMedia(t *testing.T) {
	// given a mystery with two pictures and a finished upload marked as a spoiler
	svc, m := newTestService(t)
	mid, userID := uuid.New(), uuid.New()
	ref := dto.UploadRef{ID: uuid.New(), IsSpoiler: true}
	stubAuthor(m, mid, userID)
	m.repo.EXPECT().GetMedia(mock.Anything, mid).Return([]model.PostMediaRow{{ID: 1}, {ID: 2}}, nil)
	m.repo.EXPECT().
		AttachMedia(mock.Anything, spec.MediaAttachment{TargetID: mid, SortOrder: 2, Uploads: upload.MediaClaim(userID, []dto.UploadRef{ref})}).
		Return(&model.PostMediaRow{ID: 9, PostID: mid, MediaURL: "/uploads/media/x.webp", MediaType: upload.MediaTypeImage, SortOrder: 2, IsSpoiler: true}, nil)

	// when
	got, err := svc.AttachMedia(context.Background(), mid, userID, ref)

	// then
	require.NoError(t, err)
	assert.Equal(t, dto.PostMediaResponse{ID: 9, MediaURL: "/uploads/media/x.webp", MediaType: upload.MediaTypeImage, SortOrder: 2, IsSpoiler: true}, *got)
}

func TestDeleteMedia_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errMissingRow)

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
	m.repo.EXPECT().DeleteMedia(mock.Anything, spec.MediaDeletion{ID: 1, TargetID: mid}).Return("", errors.New("boom"))

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
	m.repo.EXPECT().DeleteMedia(mock.Anything, spec.MediaDeletion{ID: 1, TargetID: mid}).Return("/uploads/mysteries/x.png", nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/mysteries/x.png"}).Return()

	// when
	err := svc.DeleteMedia(context.Background(), 1, mid, userID)

	// then
	require.NoError(t, err)
}
