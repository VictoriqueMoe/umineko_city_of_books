package comment_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
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
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/upload/uploadtest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testMediaLimit = 1024 * 1024
	quietWindow    = 100 * time.Millisecond
)

type fixture struct {
	repo        *repository.MockCommentRepository
	userRepo    *repository.MockUserRepository
	authz       *authz.MockService
	blockSvc    *block.MockService
	notifSvc    *notification.MockService
	uploadSvc   *upload.MockService
	recorder    *upload.MockMediaRecorder
	settingsSvc *settings.MockService
	notified    chan dto.NotifyParams
	svc         comment.Service
}

var (
	errBoom     = errors.New("boom")
	errDiskFull = errors.New("disk full")
	errMissing  = errors.Join(errors.New("no row"), dao.ErrNotFound)
)

func newFixture(t *testing.T) *fixture {
	f := &fixture{
		repo:        repository.NewMockCommentRepository(t),
		userRepo:    repository.NewMockUserRepository(t),
		authz:       authz.NewMockService(t),
		blockSvc:    block.NewMockService(t),
		notifSvc:    notification.NewMockService(t),
		uploadSvc:   upload.NewMockService(t),
		recorder:    upload.NewMockMediaRecorder(t),
		settingsSvc: settings.NewMockService(t),
		notified:    make(chan dto.NotifyParams, 16),
	}

	f.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			f.notified <- p
			return nil
		}).
		Maybe()

	mentionSvc := mention.NewService(f.userRepo, f.blockSvc, f.notifSvc)
	f.svc = comment.NewService(f.repo, f.userRepo, f.authz, f.blockSvc, f.notifSvc, mentionSvc, f.uploadSvc, f.recorder, f.settingsSvc, contentfilter.New())

	return f
}

func (f *fixture) expectActor(id uuid.UUID, name string) {
	f.userRepo.EXPECT().GetByID(mock.Anything, id).Return(&model.User{ID: id, DisplayName: name}, nil)
}

func (f *fixture) awaitNotifications(t *testing.T, want int) []dto.NotifyParams {
	t.Helper()

	got := make([]dto.NotifyParams, 0, want)
	for len(got) < want {
		select {
		case p := <-f.notified:
			got = append(got, p)
		case <-time.After(5 * time.Second):
			t.Fatalf("expected %d notifications, got %d", want, len(got))
		}
	}

	select {
	case p := <-f.notified:
		t.Fatalf("an unexpected %s notification went to %s", p.Type, p.RecipientID)
	case <-time.After(quietWindow):
	}

	return got
}

func (f *fixture) expectCreate(target spec.CommentTarget, parentID *uuid.UUID, userID uuid.UUID, body string, commentID uuid.UUID) {
	f.repo.EXPECT().
		CreateComment(mock.Anything, spec.NewComment{Target: target, ParentID: parentID, UserID: userID, Body: body}).
		Return(&model.CommentRow{ID: commentID}, nil).
		Once()
}

func recordInto(urlPath string) func(context.Context, string, string, string, int64, io.Reader, bool, media.AddFn, media.UpdateURLFn, media.UpdateURLFn) (*dto.PostMediaResponse, error) {
	return func(_ context.Context, _ string, _ string, filename string, _ int64, _ io.Reader, isSpoiler bool, addFn media.AddFn, _ media.UpdateURLFn, _ media.UpdateURLFn) (*dto.PostMediaResponse, error) {
		id, err := addFn(urlPath, media.MediaTypeImage, "", filename, 0)
		if err != nil {
			return nil, err
		}

		return &dto.PostMediaResponse{ID: int(id), MediaURL: urlPath, MediaType: media.MediaTypeImage, Filename: filename, IsSpoiler: isSpoiler}, nil
	}
}

func shipTarget(shipID uuid.UUID) spec.CommentTarget {
	return spec.CommentTarget{Kind: spec.CommentKindShip, ID: shipID}
}

func TestCreate_RefusesBeforeWritingAnything(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		files    func(t *testing.T) []upload.FileUpload
		ownerErr error
		blocked  bool
		blockErr error
		want     error
	}{
		{name: "a blank body with no files is refused", body: "   ", want: comment.ErrEmptyBody},
		{name: "a file that is not really an image is refused", body: "hi", files: func(*testing.T) []upload.FileUpload {
			return []upload.FileUpload{uploadtest.MemoryUpload("image/png", "fake.png", []byte("this is plainly not an image"), false)}
		}, want: upload.ErrInvalidFileType},
		{name: "a missing target is reported as such, not as a missing comment", body: "hi", ownerErr: errMissing, want: comment.ErrTargetNotFound},
		{name: "a failed owner lookup is surfaced", body: "hi", ownerErr: errBoom, want: errBoom},
		{name: "a block either way refuses the comment", body: "hi", blocked: true, want: block.ErrUserBlocked},
		{name: "a failed block lookup refuses the comment", body: "hi", blockErr: errBoom, want: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newFixture(t)
			shipID, userID, ownerID := uuid.New(), uuid.New(), uuid.New()
			uploadtest.ExpectMediaLimits(f.settingsSvc, testMediaLimit)

			var files []upload.FileUpload
			if tc.files != nil {
				files = tc.files(t)
			}

			validInput := tc.want != comment.ErrEmptyBody && tc.want != upload.ErrInvalidFileType
			if validInput {
				f.repo.EXPECT().GetTargetOwnerID(mock.Anything, shipTarget(shipID)).Return(ownerID, tc.ownerErr)
			}
			if validInput && tc.ownerErr == nil {
				f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, ownerID).Return(tc.blocked, tc.blockErr)
			}

			// when
			id, err := f.svc.Create(context.Background(), shipTarget(shipID), userID, dto.CreateCommentRequest{Body: tc.body}, files)

			// then
			require.ErrorIs(t, err, tc.want)
			assert.Equal(t, uuid.Nil, id)
			f.repo.AssertNotCalled(t, "CreateComment", mock.Anything, mock.Anything)
		})
	}
}

func TestCreate_AFailedWriteIsSurfaced(t *testing.T) {
	// given
	f := newFixture(t)
	shipID, userID, ownerID := uuid.New(), uuid.New(), uuid.New()
	f.repo.EXPECT().GetTargetOwnerID(mock.Anything, shipTarget(shipID)).Return(ownerID, nil)
	f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, ownerID).Return(false, nil)
	f.repo.EXPECT().GetCommentMediaDir(spec.CommentKindShip).Return("ships", nil)
	f.repo.EXPECT().CreateComment(mock.Anything, spec.NewComment{Target: shipTarget(shipID), UserID: userID, Body: "hi"}).Return(nil, errBoom)

	// when
	id, err := f.svc.Create(context.Background(), shipTarget(shipID), userID, dto.CreateCommentRequest{Body: "  hi  "}, nil)

	// then
	require.ErrorIs(t, err, errBoom)
	assert.Equal(t, uuid.Nil, id)
}

func TestCreate_NotifiesTheOwnerAndTheParentAuthorOnceEach(t *testing.T) {
	ownerID, parentAuthorID, actorID := uuid.New(), uuid.New(), uuid.New()

	cases := []struct {
		name         string
		kind         spec.CommentKind
		actorID      uuid.UUID
		parentAuthor *uuid.UUID
		want         map[uuid.UUID]dto.NotificationType
		wantAction   map[uuid.UUID]string
	}{
		{
			name:       "a top-level comment tells the owner",
			kind:       spec.CommentKindShip,
			actorID:    actorID,
			want:       map[uuid.UUID]dto.NotificationType{ownerID: dto.NotifShipCommented},
			wantAction: map[uuid.UUID]string{ownerID: "commented on your ship"},
		},
		{
			name:         "a reply tells the parent's author and the owner",
			kind:         spec.CommentKindShip,
			actorID:      actorID,
			parentAuthor: &parentAuthorID,
			want:         map[uuid.UUID]dto.NotificationType{parentAuthorID: dto.NotifShipCommentReply, ownerID: dto.NotifShipCommented},
			wantAction:   map[uuid.UUID]string{parentAuthorID: "replied to your comment", ownerID: "commented on your ship"},
		},
		{
			name:         "a reply to the owner's own comment tells the owner once, as a reply",
			kind:         spec.CommentKindShip,
			actorID:      actorID,
			parentAuthor: &ownerID,
			want:         map[uuid.UUID]dto.NotificationType{ownerID: dto.NotifShipCommentReply},
			wantAction:   map[uuid.UUID]string{ownerID: "replied to your comment"},
		},
		{
			name:    "the owner commenting on their own content tells nobody",
			kind:    spec.CommentKindShip,
			actorID: ownerID,
			want:    map[uuid.UUID]dto.NotificationType{},
		},
		{
			name:         "replying to yourself tells only the owner",
			kind:         spec.CommentKindShip,
			actorID:      actorID,
			parentAuthor: &actorID,
			want:         map[uuid.UUID]dto.NotificationType{ownerID: dto.NotifShipCommented},
			wantAction:   map[uuid.UUID]string{ownerID: "commented on your ship"},
		},
		{
			name:       "a short family name is spelt as an acronym",
			kind:       spec.CommentKindOC,
			actorID:    actorID,
			want:       map[uuid.UUID]dto.NotificationType{ownerID: dto.NotifOCCommented},
			wantAction: map[uuid.UUID]string{ownerID: "commented on your OC"},
		},
		{
			name:       "a mystery's game master is told with its own type",
			kind:       spec.CommentKindMystery,
			actorID:    actorID,
			want:       map[uuid.UUID]dto.NotificationType{ownerID: dto.NotifMysteryCommented},
			wantAction: map[uuid.UUID]string{ownerID: "commented on your mystery"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newFixture(t)
			entityID, commentID := uuid.New(), uuid.New()
			target := spec.CommentTarget{Kind: tc.kind, ID: entityID}

			var parentID *uuid.UUID
			if tc.parentAuthor != nil {
				parentID = new(uuid.New())
				f.repo.EXPECT().GetCommentAuthorID(mock.Anything, spec.CommentRef{Kind: tc.kind, CommentID: *parentID}).Return(*tc.parentAuthor, nil)
			}

			f.repo.EXPECT().GetTargetOwnerID(mock.Anything, target).Return(ownerID, nil)
			if tc.actorID != ownerID {
				f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, tc.actorID, ownerID).Return(false, nil)
			}
			if tc.actorID == ownerID {
				f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, ownerID, ownerID).Return(false, nil)
			}
			f.repo.EXPECT().GetCommentMediaDir(tc.kind).Return("dir", nil)
			f.expectCreate(target, parentID, tc.actorID, "nice", commentID)
			f.expectActor(tc.actorID, "Battler")

			// when
			id, err := f.svc.Create(context.Background(), target, tc.actorID, dto.CreateCommentRequest{Body: "nice", ParentID: parentID}, nil)

			// then
			require.NoError(t, err)
			assert.Equal(t, commentID, id)

			got := f.awaitNotifications(t, len(tc.want))
			for _, p := range got {
				assert.Equal(t, tc.want[p.RecipientID], p.Type, "recipient %s", p.RecipientID)
				assert.Equal(t, tc.wantAction[p.RecipientID], p.EmailAction)
				assert.Equal(t, tc.actorID, p.ActorID)
				assert.Equal(t, "Battler", p.EmailActor)
			}
		})
	}
}

func TestCreate_TheNotificationLinksToTheNewComment(t *testing.T) {
	// given
	f := newFixture(t)
	shipID, userID, ownerID, commentID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.repo.EXPECT().GetTargetOwnerID(mock.Anything, shipTarget(shipID)).Return(ownerID, nil)
	f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, ownerID).Return(false, nil)
	f.repo.EXPECT().GetCommentMediaDir(spec.CommentKindShip).Return("ships", nil)
	f.expectCreate(shipTarget(shipID), nil, userID, "hi", commentID)
	f.expectActor(userID, "Battler")

	// when
	_, err := f.svc.Create(context.Background(), shipTarget(shipID), userID, dto.CreateCommentRequest{Body: "hi"}, nil)

	// then
	require.NoError(t, err)
	got := f.awaitNotifications(t, 1)
	assert.Equal(t, shipID, got[0].ReferenceID)
	assert.Equal(t, "ship_comment:"+commentID.String(), got[0].ReferenceType)
	assert.Equal(t, "/ships/"+shipID.String()+"#comment-"+commentID.String(), got[0].EmailLink)
}

func TestCreate_MentionNotifiesTheNamedUser(t *testing.T) {
	// given
	f := newFixture(t)
	shipID, userID, commentID, mentionedID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.repo.EXPECT().GetTargetOwnerID(mock.Anything, shipTarget(shipID)).Return(userID, nil)
	f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, nil)
	f.repo.EXPECT().GetCommentMediaDir(spec.CommentKindShip).Return("ships", nil)
	f.expectCreate(shipTarget(shipID), nil, userID, "look at this @alice", commentID)
	f.expectActor(userID, "Battler")
	f.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
	f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)

	// when
	_, err := f.svc.Create(context.Background(), shipTarget(shipID), userID, dto.CreateCommentRequest{Body: "look at this @alice"}, nil)

	// then
	require.NoError(t, err)
	got := f.awaitNotifications(t, 1)
	assert.Equal(t, dto.NotifMention, got[0].Type)
	assert.Equal(t, mentionedID, got[0].RecipientID)
	assert.Equal(t, "ship_comment:"+commentID.String(), got[0].ReferenceType)
	assert.Equal(t, "/ships/"+shipID.String()+"#comment-"+commentID.String(), got[0].EmailLink)
}

func TestCreate_SavesItsMediaInDisplayOrderIntoTheKindsDirectory(t *testing.T) {
	// given a media-only comment with two images, the second marked as a spoiler
	f := newFixture(t)
	shipID, userID, commentID := uuid.New(), uuid.New(), uuid.New()
	img := uploadtest.PNG(t)
	uploadtest.ExpectMediaLimits(f.settingsSvc, testMediaLimit)

	f.repo.EXPECT().GetTargetOwnerID(mock.Anything, shipTarget(shipID)).Return(userID, nil)
	f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, nil)
	f.repo.EXPECT().GetCommentMediaDir(spec.CommentKindShip).Return("ships", nil)
	f.expectCreate(shipTarget(shipID), nil, userID, "", commentID)
	f.recorder.EXPECT().
		SaveAndRecord(mock.Anything, "ships", "image/png", "one.png", mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(recordInto("/uploads/ships/one.png")).
		Once()
	f.recorder.EXPECT().
		SaveAndRecord(mock.Anything, "ships", "image/png", "two.png", mock.Anything, mock.Anything, true, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(recordInto("/uploads/ships/two.png")).
		Once()
	f.repo.EXPECT().
		AddCommentMedia(mock.Anything, spec.NewCommentMedia{Kind: spec.CommentKindShip, NewMedia: spec.NewMedia{TargetID: commentID, MediaURL: "/uploads/ships/one.png", MediaType: media.MediaTypeImage, Filename: "one.png", SortOrder: 0}}).
		Return(int64(1), nil).
		Once()
	f.repo.EXPECT().
		AddCommentMedia(mock.Anything, spec.NewCommentMedia{Kind: spec.CommentKindShip, NewMedia: spec.NewMedia{TargetID: commentID, MediaURL: "/uploads/ships/two.png", MediaType: media.MediaTypeImage, Filename: "two.png", SortOrder: 1, IsSpoiler: true}}).
		Return(int64(2), nil).
		Once()
	f.expectActor(userID, "Battler")

	// when
	id, err := f.svc.Create(context.Background(), shipTarget(shipID), userID, dto.CreateCommentRequest{Body: "   "}, []upload.FileUpload{
		uploadtest.MemoryUpload("image/png", "one.png", img, false),
		uploadtest.MemoryUpload("image/png", "two.png", img, true),
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, commentID, id)
	f.awaitNotifications(t, 0)
}

func TestCreate_AFailedMediaSaveRollsTheCommentBackAndTellsNobody(t *testing.T) {
	cases := []struct {
		name        string
		rollbackErr error
		hangUp      bool
		wantErrs    []error
	}{
		{name: "the comment and its first image are removed", wantErrs: []error{errDiskFull}},
		{name: "a failed rollback is reported alongside the save error", rollbackErr: errBoom, wantErrs: []error{errDiskFull, errBoom}},
		{name: "a client hanging up still lets the rollback run", hangUp: true, wantErrs: []error{errDiskFull}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a comment mentioning someone whose second image fails to store
			f := newFixture(t)
			shipID, userID, ownerID, commentID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			img := uploadtest.PNG(t)
			uploadtest.ExpectMediaLimits(f.settingsSvc, testMediaLimit)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			f.repo.EXPECT().GetTargetOwnerID(mock.Anything, shipTarget(shipID)).Return(ownerID, nil)
			f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, ownerID).Return(false, nil)
			f.repo.EXPECT().GetCommentMediaDir(spec.CommentKindShip).Return("ships", nil)
			f.expectCreate(shipTarget(shipID), nil, userID, "look at this @alice", commentID)
			f.recorder.EXPECT().
				SaveAndRecord(mock.Anything, "ships", "image/png", "one.png", mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(recordInto("/uploads/ships/one.png")).
				Once()
			f.recorder.EXPECT().
				SaveAndRecord(mock.Anything, "ships", "image/png", "two.png", mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(context.Context, string, string, string, int64, io.Reader, bool, media.AddFn, media.UpdateURLFn, media.UpdateURLFn) (*dto.PostMediaResponse, error) {
					if tc.hangUp {
						cancel()
					}

					return nil, errDiskFull
				}).
				Once()
			f.repo.EXPECT().AddCommentMedia(mock.Anything, mock.Anything).Return(int64(1), nil).Once()
			f.uploadSvc.EXPECT().Delete([]string{"/uploads/ships/one.png"}).Return().Once()

			var rollbackCtxErr error
			f.repo.EXPECT().
				DeleteComment(mock.Anything, spec.CommentDeletion{CommentRef: spec.CommentRef{Kind: spec.CommentKindShip, CommentID: commentID}, UserID: userID, AsAdmin: true}).
				RunAndReturn(func(ctx context.Context, _ spec.CommentDeletion, _ ...*sql.Tx) error {
					rollbackCtxErr = ctx.Err()
					return tc.rollbackErr
				}).
				Once()

			// when
			id, err := f.svc.Create(ctx, shipTarget(shipID), userID, dto.CreateCommentRequest{Body: "look at this @alice"}, []upload.FileUpload{
				uploadtest.MemoryUpload("image/png", "one.png", img, false),
				uploadtest.MemoryUpload("image/png", "two.png", img, false),
			})

			// then
			for _, want := range tc.wantErrs {
				require.ErrorIs(t, err, want)
			}
			assert.Equal(t, uuid.Nil, id)
			require.NoError(t, rollbackCtxErr, "the rollback must not inherit the request's cancellation")
			f.userRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
			f.awaitNotifications(t, 0)
		})
	}
}

func TestUpdate(t *testing.T) {
	authorID, strangerID := uuid.New(), uuid.New()

	cases := []struct {
		name      string
		actorID   uuid.UUID
		body      string
		lookupErr error
		canEdit   bool
		writeErr  error
		want      error
		wantAdmin bool
	}{
		{name: "a blank body is refused", actorID: authorID, body: "   ", want: comment.ErrEmptyBody},
		{name: "a missing comment is not found", actorID: authorID, body: "hi", lookupErr: errMissing, want: comment.ErrNotFound},
		{name: "a failed author lookup is surfaced, not reported as not found", actorID: authorID, body: "hi", lookupErr: errBoom, want: errBoom},
		{name: "the author edits without an audit row", actorID: authorID, body: "  hi  "},
		{name: "the author's failed write bubbles", actorID: authorID, body: "hi", writeErr: errBoom, want: errBoom},
		{name: "a stranger without the permission is refused", actorID: strangerID, body: "hi", want: comment.ErrForbidden},
		{name: "a moderator's edit of someone else's comment is always audited", actorID: strangerID, body: "hi", canEdit: true, wantAdmin: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newFixture(t)
			shipID, commentID := uuid.New(), uuid.New()
			ref := spec.CommentRef{Kind: spec.CommentKindShip, CommentID: commentID}

			reachesLookup := tc.want != comment.ErrEmptyBody
			if reachesLookup {
				f.repo.EXPECT().GetCommentAuthorID(mock.Anything, ref).Return(authorID, tc.lookupErr)
			}
			if reachesLookup && tc.lookupErr == nil && tc.actorID != authorID {
				f.authz.EXPECT().Can(mock.Anything, tc.actorID, authz.PermEditAnyComment).Return(tc.canEdit)
			}
			if reachesLookup && tc.lookupErr == nil && tc.actorID == authorID {
				f.repo.EXPECT().UpdateComment(mock.Anything, spec.CommentUpdate{CommentRef: ref, UserID: authorID, Body: "hi"}).Return(tc.writeErr)
			}
			if tc.wantAdmin {
				f.repo.EXPECT().UpdateCommentWithAudit(mock.Anything, spec.AuditedCommentUpdate{
					CommentUpdate: spec.CommentUpdate{CommentRef: ref, UserID: strangerID, Body: "hi", AsAdmin: true},
					Audit: audit.NewEntry{
						ActorID:    strangerID,
						Action:     audit.ActionShipCommentUpdateAdmin,
						TargetType: audit.TargetShipComment,
						TargetID:   commentID.String(),
						SubjectID:  authorID,
					},
				}).Return(nil)
				f.repo.EXPECT().GetCommentTarget(mock.Anything, ref).Return(shipTarget(shipID), nil)
				f.expectActor(strangerID, "Erika")
			}

			// when
			err := f.svc.Update(context.Background(), ref, tc.actorID, tc.body)

			// then
			require.ErrorIs(t, err, tc.want)

			if !tc.wantAdmin {
				f.awaitNotifications(t, 0)
				return
			}

			edited := f.awaitNotifications(t, 1)[0]
			assert.Equal(t, dto.NotifContentEdited, edited.Type)
			assert.Equal(t, authorID, edited.RecipientID)
			assert.Equal(t, "ship_comment:"+commentID.String(), edited.ReferenceType)
			assert.Equal(t, "/ships/"+shipID.String()+"#comment-"+commentID.String(), edited.EmailLink)
		})
	}
}

func TestDelete(t *testing.T) {
	authorID, strangerID := uuid.New(), uuid.New()
	paths := []string{"/uploads/ships/c.png", "/uploads/ships/c-thumb.png"}

	cases := []struct {
		name       string
		actorID    uuid.UUID
		lookupErr  error
		canDelete  bool
		writeErr   error
		want       error
		wantAction audit.Action
	}{
		{name: "a missing comment is not found", actorID: authorID, lookupErr: errMissing, want: comment.ErrNotFound},
		{name: "the author's delete is audited as their own", actorID: authorID, wantAction: audit.ActionShipCommentDelete},
		{name: "a stranger without the permission is refused", actorID: strangerID, want: comment.ErrForbidden},
		{name: "a moderator's delete is audited as an admin action", actorID: strangerID, canDelete: true, wantAction: audit.ActionShipCommentDeleteAdmin},
		{name: "a failed delete keeps the files", actorID: authorID, writeErr: errBoom, want: errBoom, wantAction: audit.ActionShipCommentDelete},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newFixture(t)
			commentID := uuid.New()
			ref := spec.CommentRef{Kind: spec.CommentKindShip, CommentID: commentID}
			f.repo.EXPECT().GetCommentAuthorID(mock.Anything, ref).Return(authorID, tc.lookupErr)
			if tc.lookupErr == nil && tc.actorID != authorID {
				f.authz.EXPECT().Can(mock.Anything, tc.actorID, authz.PermDeleteAnyComment).Return(tc.canDelete)
			}
			if tc.wantAction != "" {
				f.repo.EXPECT().DeleteCommentWithAudit(mock.Anything, spec.AuditedCommentDeletion{
					CommentDeletion: spec.CommentDeletion{CommentRef: ref, UserID: tc.actorID, AsAdmin: tc.actorID != authorID},
					Audit: audit.NewEntry{
						ActorID:    tc.actorID,
						Action:     tc.wantAction,
						TargetType: audit.TargetShipComment,
						TargetID:   commentID.String(),
						SubjectID:  authorID,
					},
				}).Return(paths, tc.writeErr)
			}
			if tc.wantAction != "" && tc.writeErr == nil {
				f.uploadSvc.EXPECT().Delete(paths).Return().Once()
			}

			// when
			err := f.svc.Delete(context.Background(), ref, tc.actorID)

			// then
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestLike(t *testing.T) {
	cases := []struct {
		name      string
		selfLike  bool
		lookupErr error
		blocked   bool
		blockErr  error
		likeErr   error
		want      error
		notifies  bool
	}{
		{name: "a missing comment is not found", lookupErr: errMissing, want: comment.ErrNotFound},
		{name: "a block either way refuses the like", blocked: true, want: block.ErrUserBlocked},
		{name: "a failed block lookup refuses the like", blockErr: errBoom, want: errBoom},
		{name: "a failed like bubbles and tells nobody", likeErr: errBoom, want: errBoom},
		{name: "liking your own comment tells nobody", selfLike: true},
		{name: "liking someone else's comment always tells its author", notifies: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newFixture(t)
			shipID, commentID, userID, authorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			if tc.selfLike {
				authorID = userID
			}
			ref := spec.CommentRef{Kind: spec.CommentKindShip, CommentID: commentID}
			f.repo.EXPECT().GetCommentAuthorID(mock.Anything, ref).Return(authorID, tc.lookupErr)
			if tc.lookupErr == nil {
				f.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(tc.blocked, tc.blockErr)
			}
			if tc.lookupErr == nil && !tc.blocked && tc.blockErr == nil {
				f.repo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{CommentRef: ref, UserID: userID}).Return(tc.likeErr)
			}
			if tc.notifies {
				f.repo.EXPECT().GetCommentTarget(mock.Anything, ref).Return(shipTarget(shipID), nil)
				f.expectActor(userID, "Battler")
			}

			// when
			err := f.svc.Like(context.Background(), ref, userID)

			// then
			require.ErrorIs(t, err, tc.want)
			if !tc.notifies {
				f.awaitNotifications(t, 0)
				return
			}

			liked := f.awaitNotifications(t, 1)[0]
			assert.Equal(t, dto.NotifCommentLiked, liked.Type)
			assert.Equal(t, authorID, liked.RecipientID)
			assert.Equal(t, "liked your comment", liked.EmailAction)
			assert.Equal(t, "ship_comment:"+commentID.String(), liked.ReferenceType)
			assert.Equal(t, "/ships/"+shipID.String()+"#comment-"+commentID.String(), liked.EmailLink)
		})
	}
}

func TestUnlike(t *testing.T) {
	cases := []struct {
		name    string
		repoErr error
	}{
		{name: "the like is removed"},
		{name: "a failed unlike bubbles", repoErr: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			f := newFixture(t)
			ref := spec.CommentRef{Kind: spec.CommentKindShip, CommentID: uuid.New()}
			userID := uuid.New()
			f.repo.EXPECT().UnlikeComment(mock.Anything, spec.CommentLike{CommentRef: ref, UserID: userID}).Return(tc.repoErr)

			// when
			err := f.svc.Unlike(context.Background(), ref, userID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestUploadMedia(t *testing.T) {
	cases := []struct {
		name        string
		stranger    bool
		lookupErr   error
		existingErr error
		saveErr     error
		want        error
	}{
		{name: "a missing comment is not found", lookupErr: errMissing, want: comment.ErrNotFound},
		{name: "a failed author lookup is surfaced, not reported as not found", lookupErr: errBoom, want: errBoom},
		{name: "only the comment's author may attach media", stranger: true, want: authz.ErrNotCommentAuthor},
		{name: "a failed existing media lookup is surfaced instead of reusing a sort position", existingErr: errBoom, want: errBoom},
		{name: "a failed save bubbles", saveErr: errDiskFull, want: errDiskFull},
		{name: "the file lands in the kind's directory after the comment's existing media"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a comment that already carries two files
			f := newFixture(t)
			commentID, userID := uuid.New(), uuid.New()
			authorID := userID
			if tc.stranger {
				authorID = uuid.New()
			}
			ref := spec.CommentRef{Kind: spec.CommentKindShip, CommentID: commentID}
			f.repo.EXPECT().GetCommentAuthorID(mock.Anything, ref).Return(authorID, tc.lookupErr)

			reachesLookup := tc.lookupErr == nil && !tc.stranger
			if reachesLookup {
				f.repo.EXPECT().GetCommentMediaDir(spec.CommentKindShip).Return("ships", nil)
				f.repo.EXPECT().GetCommentMedia(mock.Anything, ref).Return([]model.PostMediaRow{{ID: 1}, {ID: 2}}, tc.existingErr)
			}

			reachesSave := reachesLookup && tc.existingErr == nil
			if reachesSave && tc.saveErr != nil {
				f.recorder.EXPECT().
					SaveAndRecord(mock.Anything, "ships", "image/png", "photo.png", int64(3), mock.Anything, true, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tc.saveErr)
			}
			if reachesSave && tc.saveErr == nil {
				f.recorder.EXPECT().
					SaveAndRecord(mock.Anything, "ships", "image/png", "photo.png", int64(3), mock.Anything, true, mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(recordInto("/uploads/ships/photo.png"))
				f.repo.EXPECT().
					AddCommentMedia(mock.Anything, spec.NewCommentMedia{Kind: spec.CommentKindShip, NewMedia: spec.NewMedia{TargetID: commentID, MediaURL: "/uploads/ships/photo.png", MediaType: media.MediaTypeImage, Filename: "photo.png", SortOrder: 2, IsSpoiler: true}}).
					Return(int64(42), nil)
			}

			// when
			resp, err := f.svc.UploadMedia(context.Background(), ref, userID, "image/png", "photo.png", 3, strings.NewReader("img"), true)

			// then
			require.ErrorIs(t, err, tc.want)
			if tc.want != nil {
				assert.Nil(t, resp)
				return
			}

			require.NotNil(t, resp)
			assert.Equal(t, 42, resp.ID)
			assert.Equal(t, "/uploads/ships/photo.png", resp.MediaURL)
			assert.Equal(t, 2, resp.SortOrder)
		})
	}
}

func TestThread_BuildsTheTreeWithoutBlockedUsersAndFlagsTheOwner(t *testing.T) {
	// given
	f := newFixture(t)
	shipID, viewerID, ownerID, otherID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	parentID, replyID := uuid.New(), uuid.New()
	blocked := []uuid.UUID{uuid.New()}

	f.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(blocked, nil)
	f.repo.EXPECT().GetTargetOwnerID(mock.Anything, shipTarget(shipID)).Return(ownerID, nil)
	f.repo.EXPECT().GetComments(mock.Anything, spec.CommentQuery{Target: shipTarget(shipID), ViewerID: viewerID, Limit: 500, ExcludeUserIDs: blocked}).
		Return([]model.CommentRow{
			{ID: parentID, UserID: otherID, Body: "parent", AuthorBanned: true},
			{ID: replyID, ParentID: &parentID, UserID: ownerID, Body: "reply"},
		}, 2, nil)
	f.repo.EXPECT().GetCommentMediaBatch(mock.Anything, spec.CommentMediaBatch{Kind: spec.CommentKindShip, CommentIDs: []uuid.UUID{parentID, replyID}}).
		Return(map[uuid.UUID][]model.PostMediaRow{replyID: {{ID: 7, MediaURL: "/uploads/ships/r.png", MediaType: media.MediaTypeImage}}}, nil)

	// when
	got, err := f.svc.Thread(context.Background(), shipTarget(shipID), viewerID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, parentID, got[0].ID)
	assert.False(t, got[0].IsAuthor)
	assert.True(t, got[0].Author.Banned)
	require.Len(t, got[0].Replies, 1)
	assert.Equal(t, replyID, got[0].Replies[0].ID)
	assert.True(t, got[0].Replies[0].IsAuthor, "the owner's own comment is flagged")
	require.Len(t, got[0].Replies[0].Media, 1)
	assert.Equal(t, "/uploads/ships/r.png", got[0].Replies[0].Media[0].MediaURL)
}

func TestThread_AFailedReadIsSurfacedInsteadOfRenderingAnEmptyThread(t *testing.T) {
	steps := []string{"blocked users", "owner", "comments", "comment media"}

	for _, failAt := range steps {
		t.Run("the "+failAt+" read failing", func(t *testing.T) {
			// given
			f := newFixture(t)
			shipID, viewerID := uuid.New(), uuid.New()
			errAt := func(step string) error {
				if step == failAt {
					return errBoom
				}

				return nil
			}
			f.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, errAt("blocked users")).Maybe()
			f.repo.EXPECT().GetTargetOwnerID(mock.Anything, mock.Anything).Return(uuid.New(), errAt("owner")).Maybe()
			f.repo.EXPECT().GetComments(mock.Anything, mock.Anything).Return([]model.CommentRow{{ID: uuid.New()}}, 1, errAt("comments")).Maybe()
			f.repo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, errAt("comment media")).Maybe()

			// when
			got, err := f.svc.Thread(context.Background(), shipTarget(shipID), viewerID)

			// then
			require.ErrorIs(t, err, errBoom)
			assert.Nil(t, got)
		})
	}
}
