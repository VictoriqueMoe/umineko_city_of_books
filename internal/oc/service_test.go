package oc

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/sidebar"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	errMissingRow = errors.Join(errors.New("no row"), dao.ErrNotFound)
)

type testMocks struct {
	ocRepo      *repository.MockOCRepository
	commentSvc  *comment.MockService
	userRepo    *repository.MockUserRepository
	auditRepo   *repository.MockAuditLogRepository
	authz       *authz.MockService
	blockSvc    *block.MockService
	notifSvc    *notification.MockService
	uploadSvc   *upload.MockService
	settingsSvc *settings.MockService
	activity    *sidebar.MockActivityBumper
}

func newTestService(t *testing.T) (*service, *testMocks) {
	ocRepo := repository.NewMockOCRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := media.NewProcessor(1)
	commentSvc := comment.NewMockService(t)
	activity := sidebar.NewMockActivityBumper(t)
	mentionSvc := mention.NewService(userRepo, blockSvc, notifSvc)

	svc := NewService(ocRepo, userRepo, auditRepo, authzSvc, blockSvc, notifSvc, mentionSvc, commentSvc, uploadSvc, mediaProc, settingsSvc, nil, contentfilter.New(), nil, activity).(*service)
	return svc, &testMocks{
		ocRepo:      ocRepo,
		commentSvc:  commentSvc,
		userRepo:    userRepo,
		auditRepo:   auditRepo,
		authz:       authzSvc,
		blockSvc:    blockSvc,
		notifSvc:    notifSvc,
		uploadSvc:   uploadSvc,
		settingsSvc: settingsSvc,
		activity:    activity,
	}
}

func TestCreateOC_EmptyNameRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateOCRequest{Name: "   ", Series: "umineko"}

	// when
	_, err := svc.CreateOC(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyName)
}

func TestCreateOC_InvalidSeriesRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateOCRequest{Name: "Linda", Series: "rosegunsdays"}

	// when
	_, err := svc.CreateOC(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrInvalidSeries)
}

func TestCreateOC_CustomSeriesRequiresName(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateOCRequest{Name: "Linda", Series: "custom", CustomSeriesName: "  "}

	// when
	_, err := svc.CreateOC(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyCustomSeries)
}

func TestCreateOC_DuplicateNameRejected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateOCRequest{Name: "Linda", Series: "umineko"}
	m.ocRepo.EXPECT().HasOC(mock.Anything, spec.OCNameLookup{UserID: userID, Name: "Linda"}).Return(true, nil)

	// when
	_, err := svc.CreateOC(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrDuplicateName)
}

func TestCreateOC_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	createdID := uuid.New()
	req := dto.CreateOCRequest{Name: "  Linda  ", Description: "  bio ", Series: "umineko"}
	m.ocRepo.EXPECT().HasOC(mock.Anything, spec.OCNameLookup{UserID: userID, Name: "Linda"}).Return(false, nil)
	m.ocRepo.EXPECT().
		Create(mock.Anything, spec.NewOC{UserID: userID, Name: "Linda", Description: "bio", Series: "umineko"}).
		Return(&model.OCRow{ID: createdID}, nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, spec.OCByID{ID: createdID, ViewerID: userID}).Return(nil, nil).Maybe()
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyOCs).Return().Once()

	// when
	id, err := svc.CreateOC(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateOC_CustomSeriesOK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	createdID := uuid.New()
	req := dto.CreateOCRequest{Name: "Linda", Description: "bio", Series: "custom", CustomSeriesName: " Higanbana "}
	m.ocRepo.EXPECT().HasOC(mock.Anything, spec.OCNameLookup{UserID: userID, Name: "Linda"}).Return(false, nil)
	m.ocRepo.EXPECT().
		Create(mock.Anything, spec.NewOC{UserID: userID, Name: "Linda", Description: "bio", Series: "custom", CustomSeriesName: "Higanbana"}).
		Return(&model.OCRow{ID: createdID}, nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, spec.OCByID{ID: createdID, ViewerID: userID}).Return(nil, nil).Maybe()
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyOCs).Return().Once()

	// when
	_, err := svc.CreateOC(context.Background(), userID, req)

	// then
	require.NoError(t, err)
}

func TestCreateOC_RepoErrorBubbles(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateOCRequest{Name: "Linda", Series: "umineko"}
	m.ocRepo.EXPECT().HasOC(mock.Anything, spec.OCNameLookup{UserID: userID, Name: "Linda"}).Return(false, nil)
	m.ocRepo.EXPECT().
		Create(mock.Anything, spec.NewOC{UserID: userID, Name: "Linda", Series: "umineko"}).
		Return(nil, errors.New("db down"))

	// when
	_, err := svc.CreateOC(context.Background(), userID, req)

	// then
	require.Error(t, err)
}

func TestCreateOC_MentionOnTheDescriptionNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	ocID := uuid.New()
	mentionedID := uuid.New()
	req := dto.CreateOCRequest{Name: "Linda", Description: "designed with @alice", Series: "umineko"}

	m.ocRepo.EXPECT().HasOC(mock.Anything, spec.OCNameLookup{UserID: userID, Name: "Linda"}).Return(false, nil)
	m.ocRepo.EXPECT().
		Create(mock.Anything, spec.NewOC{UserID: userID, Name: "Linda", Description: "designed with @alice", Series: "umineko"}).
		Return(&model.OCRow{ID: ocID}, nil)
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyOCs).Return().Once()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Battler"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)

	var wg sync.WaitGroup
	wg.Add(1)

	var mentioned dto.NotifyParams
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			mentioned = p
			wg.Done()

			return nil
		})

	// when
	_, err := svc.CreateOC(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, dto.NotifMention, mentioned.Type)
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, ocID, mentioned.ReferenceID)
	assert.Equal(t, "oc", mentioned.ReferenceType)
	assert.Equal(t, "/oc/"+ocID.String(), mentioned.EmailLink)
}

func TestGetOC_NotFoundError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	m.ocRepo.EXPECT().GetByID(mock.Anything, spec.OCByID{ID: id, ViewerID: uuid.Nil}).Return(nil, nil)

	// when
	_, err := svc.GetOC(context.Background(), id, uuid.Nil)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetOC_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	authorID := uuid.New()
	row := &model.OCRow{ID: id, UserID: authorID, Name: "Linda", Series: "umineko"}
	m.ocRepo.EXPECT().GetByID(mock.Anything, spec.OCByID{ID: id, ViewerID: uuid.Nil}).Return(row, nil)
	thread := []dto.CommentResponse{{ID: uuid.New(), Body: "she would win"}}
	m.ocRepo.EXPECT().GetGallery(mock.Anything, id).Return(nil, nil)
	m.commentSvc.EXPECT().Thread(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindOC, ID: id}, uuid.Nil).Return(thread, nil).Once()

	// when
	got, err := svc.GetOC(context.Background(), id, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "Linda", got.Name)
	assert.Equal(t, thread, got.Comments)
}

func TestUpdateOC_EmptyNameRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.UpdateOCRequest{Name: "  ", Series: "umineko"}

	// when
	err := svc.UpdateOC(context.Background(), uuid.New(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyName)
}

func TestUpdateOC_InvalidSeriesRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.UpdateOCRequest{Name: "Linda", Series: "bad"}

	// when
	err := svc.UpdateOC(context.Background(), uuid.New(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrInvalidSeries)
}

func TestUpdateOC_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	req := dto.UpdateOCRequest{Name: "Linda", Description: "bio", Series: "umineko"}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
	m.ocRepo.EXPECT().Update(mock.Anything, spec.OCUpdate{ID: id, UserID: userID, Name: "Linda", Description: "bio", Series: "umineko"}).Return(nil)
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, spec.OCByID{ID: id, ViewerID: userID}).Return(nil, nil).Maybe()

	// when
	err := svc.UpdateOC(context.Background(), id, userID, req)

	// then
	require.NoError(t, err)
}

func TestUpdateOC_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	adminID := uuid.New()
	req := dto.UpdateOCRequest{Name: "Linda", Series: "umineko"}
	m.authz.EXPECT().Can(mock.Anything, adminID, authz.PermEditAnyPost).Return(true)
	ownerID := uuid.New()
	m.ocRepo.EXPECT().Update(mock.Anything, spec.OCUpdate{ID: id, UserID: adminID, Name: "Linda", Series: "umineko", AsAdmin: true}).Return(nil)
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(ownerID, nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, spec.OCByID{ID: id, ViewerID: ownerID}).Return(nil, nil).Maybe()

	// when
	err := svc.UpdateOC(context.Background(), id, adminID, req)

	// then
	require.NoError(t, err)
}

func TestDeleteOC_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.ocRepo.EXPECT().
		DeleteOC(mock.Anything, spec.OCDeletion{ID: id, UserID: userID}).
		Return([]string{"/uploads/ocs/portrait.png", "/uploads/ocs/portrait_thumb.png"}, nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/ocs/portrait.png", "/uploads/ocs/portrait_thumb.png"}).Return()
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionOCDelete,
		TargetType: audit.TargetOC,
		TargetID:   id.String(),
		SubjectID:  userID,
	}).Return(nil)

	// when
	err := svc.DeleteOC(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteOC_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	adminID := uuid.New()
	ownerID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(ownerID, nil)
	m.authz.EXPECT().Can(mock.Anything, adminID, authz.PermDeleteAnyPost).Return(true)
	m.ocRepo.EXPECT().
		DeleteOC(mock.Anything, spec.OCDeletion{ID: id, UserID: adminID, AsAdmin: true}).
		Return([]string{"/uploads/ocs/gallery.png"}, nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/ocs/gallery.png"}).Return()
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    adminID,
		Action:     audit.ActionOCDeleteAdmin,
		TargetType: audit.TargetOC,
		TargetID:   id.String(),
		SubjectID:  ownerID,
	}).Return(nil)

	// when
	err := svc.DeleteOC(context.Background(), id, adminID)

	// then
	require.NoError(t, err)
}

func TestDeleteOC_ModeratorDeletingOwnOCRecordsOwnerAction(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(true)
	m.ocRepo.EXPECT().
		DeleteOC(mock.Anything, spec.OCDeletion{ID: id, UserID: userID, AsAdmin: true}).
		Return([]string{"/uploads/ocs/mine.png"}, nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/ocs/mine.png"}).Return()
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionOCDelete,
		TargetType: audit.TargetOC,
		TargetID:   id.String(),
		SubjectID:  userID,
	}).Return(nil)

	// when
	err := svc.DeleteOC(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteOC_RepoErrorSkipsUnlink(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.ocRepo.EXPECT().
		DeleteOC(mock.Anything, spec.OCDeletion{ID: id, UserID: userID}).
		Return(nil, errors.New("boom"))

	// when
	err := svc.DeleteOC(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestVote_BlockedReturnsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.Vote(context.Background(), userID, id, 1)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestVote_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.ocRepo.EXPECT().Vote(mock.Anything, spec.Vote{UserID: userID, TargetID: id, Value: 1}).Return(nil)

	// when
	err := svc.Vote(context.Background(), userID, id, 1)

	// then
	require.NoError(t, err)
}

func TestToggleFavourite_AddsFavourite(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, spec.OCByID{ID: id, ViewerID: userID}).Return(&model.OCRow{ID: id, Name: "Linda", UserFavourited: false}, nil)
	m.ocRepo.EXPECT().Favourite(mock.Anything, spec.Like{UserID: userID, TargetID: id}).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("ignored")).Maybe()

	// when
	favourited, err := svc.ToggleFavourite(context.Background(), userID, id)

	// then
	require.NoError(t, err)
	assert.True(t, favourited)
}

func TestToggleFavourite_RemovesFavourite(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, spec.OCByID{ID: id, ViewerID: userID}).Return(&model.OCRow{ID: id, Name: "Linda", UserFavourited: true}, nil)
	m.ocRepo.EXPECT().Unfavourite(mock.Anything, spec.Like{UserID: userID, TargetID: id}).Return(nil)

	// when
	favourited, err := svc.ToggleFavourite(context.Background(), userID, id)

	// then
	require.NoError(t, err)
	assert.False(t, favourited)
}

func TestOCLookupFailures(t *testing.T) {
	boom := errors.New("boom")
	ctx := context.Background()
	cases := []struct {
		name   string
		lookup func(m *testMocks, id uuid.UUID, err error)
		call   func(s *service, id, userID uuid.UUID) error
	}{
		{name: "delete", lookup: expectOCAuthor, call: func(s *service, id, userID uuid.UUID) error { return s.DeleteOC(ctx, id, userID) }},
		{name: "upload the portrait", lookup: expectOCAuthor, call: func(s *service, id, userID uuid.UUID) error {
			_, err := s.UploadOCImage(ctx, id, userID, "image/png", 3, strings.NewReader("img"))
			return err
		}},
		{name: "add a gallery image", lookup: expectOCAuthor, call: func(s *service, id, userID uuid.UUID) error {
			_, err := s.AddGalleryImage(ctx, id, userID, "c", "image/png", 3, strings.NewReader("img"))
			return err
		}},
		{name: "update a gallery image", lookup: expectOCAuthor, call: func(s *service, id, userID uuid.UUID) error {
			return s.UpdateGalleryImage(ctx, id, 1, userID, dto.UpdateOCImageRequest{})
		}},
		{name: "delete a gallery image", lookup: expectOCAuthor, call: func(s *service, id, userID uuid.UUID) error { return s.DeleteGalleryImage(ctx, id, 1, userID) }},
		{name: "vote", lookup: expectOCAuthor, call: func(s *service, id, userID uuid.UUID) error { return s.Vote(ctx, userID, id, 1) }},
		{name: "favourite", lookup: expectOCAuthor, call: func(s *service, id, userID uuid.UUID) error {
			_, err := s.ToggleFavourite(ctx, userID, id)
			return err
		}},
	}
	outcomes := []struct {
		name      string
		lookupErr error
		wantErr   error
	}{
		{name: "a missing row is not found", lookupErr: errMissingRow, wantErr: ErrNotFound},
		{name: "a failed lookup is surfaced, not reported as not found", lookupErr: boom, wantErr: boom},
	}

	for _, tc := range cases {
		for _, outcome := range outcomes {
			t.Run(tc.name+": "+outcome.name, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				id := uuid.New()
				tc.lookup(m, id, outcome.lookupErr)

				// when
				err := tc.call(svc, id, uuid.New())

				// then
				require.ErrorIs(t, err, outcome.wantErr)
			})
		}
	}
}

func TestOCImageWrites_ANonOwnerIsRefusedWithTheSentinel(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		call func(s *service, id, userID uuid.UUID) error
	}{
		{name: "upload the portrait", call: func(s *service, id, userID uuid.UUID) error {
			_, err := s.UploadOCImage(ctx, id, userID, "image/png", 3, strings.NewReader("img"))
			return err
		}},
		{name: "add a gallery image", call: func(s *service, id, userID uuid.UUID) error {
			_, err := s.AddGalleryImage(ctx, id, userID, "c", "image/png", 3, strings.NewReader("img"))
			return err
		}},
		{name: "update a gallery image", call: func(s *service, id, userID uuid.UUID) error {
			return s.UpdateGalleryImage(ctx, id, 1, userID, dto.UpdateOCImageRequest{})
		}},
		{name: "delete a gallery image", call: func(s *service, id, userID uuid.UUID) error { return s.DeleteGalleryImage(ctx, id, 1, userID) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)

			// when
			err := tc.call(svc, id, userID)

			// then
			require.ErrorIs(t, err, ErrNotOwner)
		})
	}
}

func expectOCAuthor(m *testMocks, id uuid.UUID, err error) {
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.Nil, err)
}

func TestOCBlockCheckFailuresRefuseTheAction(t *testing.T) {
	boom := errors.New("boom")
	ctx := context.Background()
	cases := []struct {
		name   string
		lookup func(m *testMocks, id, authorID uuid.UUID)
		call   func(s *service, id, userID uuid.UUID) error
	}{
		{name: "vote", lookup: expectOCOwnedBy, call: func(s *service, id, userID uuid.UUID) error { return s.Vote(ctx, userID, id, 1) }},
		{name: "favourite", lookup: expectOCOwnedBy, call: func(s *service, id, userID uuid.UUID) error {
			_, err := s.ToggleFavourite(ctx, userID, id)
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			authorID := uuid.New()
			tc.lookup(m, id, authorID)
			m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, boom)

			// when
			err := tc.call(svc, id, userID)

			// then
			require.ErrorIs(t, err, boom)
		})
	}
}

func expectOCOwnedBy(m *testMocks, id, authorID uuid.UUID) {
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
}

func TestGetOC_AFailedReadIsSurfacedInsteadOfRenderingAnEmptySection(t *testing.T) {
	steps := []string{"gallery", "comments", "viewer block"}

	for _, failAt := range steps {
		t.Run("the "+failAt+" read failing", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			viewerID := uuid.New()
			boom := errors.New("boom")
			errAt := func(step string) error {
				if step == failAt {
					return boom
				}

				return nil
			}
			m.ocRepo.EXPECT().GetByID(mock.Anything, spec.OCByID{ID: id, ViewerID: viewerID}).Return(&model.OCRow{ID: id, UserID: uuid.New()}, nil)
			m.ocRepo.EXPECT().GetGallery(mock.Anything, id).Return(nil, errAt("gallery")).Maybe()
			m.commentSvc.EXPECT().Thread(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindOC, ID: id}, viewerID).Return(nil, errAt("comments")).Maybe()
			m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewerID, mock.Anything).Return(false, errAt("viewer block")).Maybe()

			// when
			got, err := svc.GetOC(context.Background(), id, viewerID)

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestListOCs_AFailedReadIsSurfaced(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name       string
		blockErr   error
		galleryErr error
	}{
		{name: "the viewer's block list", blockErr: boom},
		{name: "the gallery previews", galleryErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing to load", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewerID := uuid.New()
			ocID := uuid.New()
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, tc.blockErr)
			if tc.blockErr == nil {
				m.ocRepo.EXPECT().List(mock.Anything, mock.Anything).Return([]model.OCRow{{ID: ocID}}, 1, nil)
				m.ocRepo.EXPECT().GetGalleryBatch(mock.Anything, []uuid.UUID{ocID}).Return(nil, tc.galleryErr)
			}

			// when
			got, err := svc.ListOCs(context.Background(), viewerID, "new", false, "", "", uuid.Nil, bounds.NewPage(10, 0))

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestAddGalleryImage_AFailedGalleryLookupRefusesBeforeAnythingIsSaved(t *testing.T) {
	// given
	svc, m := newTestService(t)
	ocID := uuid.New()
	userID := uuid.New()
	boom := errors.New("boom")
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, ocID).Return(userID, nil)
	m.ocRepo.EXPECT().GetGallery(mock.Anything, ocID).Return(nil, boom)

	// when
	_, err := svc.AddGalleryImage(context.Background(), ocID, userID, "c", "image/png", 3, strings.NewReader("img"))

	// then no file is left behind for a gallery row that was never written
	require.ErrorIs(t, err, boom)
	m.uploadSvc.AssertNotCalled(t, "SaveImage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestValidateSeries_AllVariants(t *testing.T) {
	// given
	cases := []struct {
		input    string
		custom   string
		wantErr  error
		wantSer  string
		wantName string
	}{
		{input: "umineko", wantSer: "umineko"},
		{input: "HIGURASHI", wantSer: "higurashi"},
		{input: " ciconia  ", wantSer: "ciconia"},
		{input: "custom", custom: "Higanbana", wantSer: "custom", wantName: "Higanbana"},
		{input: "custom", custom: "  ", wantErr: ErrEmptyCustomSeries},
		{input: "rosegunsdays", wantErr: ErrInvalidSeries},
	}

	// when / then
	for _, c := range cases {
		gotSer, gotName, err := validateSeries(c.input, c.custom)
		if c.wantErr != nil {
			require.ErrorIs(t, err, c.wantErr, "case %q", c.input)
			continue
		}
		require.NoError(t, err, "case %q", c.input)
		assert.Equal(t, c.wantSer, gotSer)
		assert.Equal(t, c.wantName, gotName)
		assert.Equal(t, strings.ToLower(strings.TrimSpace(c.input)), gotSer)
	}
}
