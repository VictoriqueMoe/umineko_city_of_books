package post

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
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
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	testMocks struct {
		postRepo     *repository.MockPostRepository
		postComments *dao.MockCommentDAO[uuid.UUID]
		userRepo     *repository.MockUserRepository
		roleRepo     *repository.MockRoleRepository
		auditRepo    *repository.MockAuditLogRepository
		authz        *authz.MockService
		blockSvc     *block.MockService
		notifSvc     *notification.MockService
		uploadSvc    *upload.MockService
		settingsSvc  *settings.MockService
		hub          *ws.Hub
	}

	authorGuard struct {
		name      string
		lookupErr error
		blocked   bool
		blockErr  error
		wantErr   error
		wantMsg   string
	}
)

var (
	uploadPostMedia = func(svc *service, ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
		_, err := svc.UploadPostMedia(ctx, postID, userID, "image/png", "photo.png", 10, strings.NewReader("x"), false)

		return err
	}

	uploadCommentMedia = func(svc *service, ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error {
		_, err := svc.UploadCommentMedia(ctx, commentID, userID, "image/png", "photo.png", 10, strings.NewReader("x"), false)

		return err
	}
)

func newTestService(t *testing.T) (*service, *testMocks) {
	t.Helper()

	postRepo := repository.NewMockPostRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := &media.Processor{}
	hub := ws.NewHub()

	postComments := dao.NewMockCommentDAO[uuid.UUID](t)
	mentionSvc := mention.NewService(userRepo, blockSvc, notifSvc, dao.CommentDAOs{
		ByID: map[string]dao.CommentDAO[uuid.UUID]{string(mention.KindPostComment): postComments},
	})

	svc := NewService(postRepo, userRepo, roleRepo, auditRepo, authzSvc, blockSvc, notifSvc, mentionSvc, uploadSvc, mediaProc, settingsSvc, hub, contentfilter.New(), nil, nil).(*service)

	return svc, &testMocks{
		postRepo:     postRepo,
		postComments: postComments,
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		auditRepo:    auditRepo,
		authz:        authzSvc,
		blockSvc:     blockSvc,
		notifSvc:     notifSvc,
		uploadSvc:    uploadSvc,
		settingsSvc:  settingsSvc,
		hub:          hub,
	}
}

func expectBackgroundSocial(m *testMocks) {
	m.postRepo.EXPECT().IncrementShareCount(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.postRepo.EXPECT().DecrementShareCount(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("ignored")).Maybe()
	m.postRepo.EXPECT().GetSharedContentAuthor(mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("ignored")).Maybe()
	m.postRepo.EXPECT().GetSharedContentPreviews(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("ignored")).Maybe()
	m.postRepo.EXPECT().GetCommentEntityID(mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("ignored")).Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("ignored")).Maybe()
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, mock.Anything).Return(nil, errors.New("ignored")).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, mock.Anything).Return("http://base").Maybe()
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	m.roleRepo.EXPECT().GetUsersByRoles(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
}

func expectAuthorLookup(m *testMocks, ofComment bool, targetID uuid.UUID, authorID uuid.UUID, err error) {
	if ofComment {
		m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, targetID).Return(authorID, err)

		return
	}

	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, targetID).Return(authorID, err)
}

func expectUnlinked(m *testMocks, paths []string) {
	if len(paths) == 0 {
		m.uploadSvc.EXPECT().Delete().Return()

		return
	}

	m.uploadSvc.EXPECT().Delete(paths).Return()
}

func expectPostListBatch(m *testMocks, rows []model.PostRow, viewerID uuid.UUID) {
	postIDs := make([]uuid.UUID, 0, len(rows))
	postIDStrs := make([]string, 0, len(rows))
	for _, r := range rows {
		postIDs = append(postIDs, r.ID)
		postIDStrs = append(postIDStrs, r.ID.String())
	}

	m.postRepo.EXPECT().GetMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetPollsByPostIDs(mock.Anything, spec.PostPollBatchQuery{PostIDs: postIDs, ViewerID: viewerID}).Return(nil, nil, nil, nil)
	m.postRepo.EXPECT().GetShareCountsBatch(mock.Anything, spec.SharedContentBatchRef{ContentIDs: postIDStrs, ContentType: "post"}).Return(nil, nil)
	m.postRepo.EXPECT().GetSharedContentPreviews(mock.Anything, mock.Anything).Return(nil, nil)
}

func matchNewPost(want spec.NewPost) any {
	return mock.MatchedBy(func(got spec.NewPost) bool {
		if got.Poll != nil {
			poll := *got.Poll
			poll.ExpiresAt = ""
			got.Poll = &poll
		}

		return assert.ObjectsAreEqual(want, got)
	})
}

func pollPostRequest(durationSeconds int, labels ...string) dto.CreatePostRequest {
	options := make([]dto.PollOptionInput, 0, len(labels))
	for _, label := range labels {
		options = append(options, dto.PollOptionInput{Label: label})
	}

	return dto.CreatePostRequest{Corner: "general", Body: "hello", Poll: &dto.CreatePollInput{Options: options, DurationSeconds: durationSeconds}}
}

func pollExpiringIn(d time.Duration) *model.PollRow {
	return &model.PollRow{ID: uuid.New().String(), ExpiresAt: time.Now().Add(d).UTC().Format(time.RFC3339)}
}

func postDeleteSpec(id uuid.UUID, userID uuid.UUID, authorID uuid.UUID, asAdmin bool, mediaCount int) spec.PostDelete {
	action := audit.ActionPostDelete
	if authorID != userID {
		action = audit.ActionPostDeleteAdmin
	}

	return spec.PostDelete{
		ID:      id,
		UserID:  userID,
		AsAdmin: asAdmin,
		Audit: audit.NewEntry{
			ActorID:    userID,
			Action:     action,
			TargetType: audit.TargetPost,
			TargetID:   id.String(),
			Details:    fmt.Sprintf("media=%d", mediaCount),
			SubjectID:  authorID,
		},
	}
}

func postCommentDeleteSpec(id uuid.UUID, userID uuid.UUID, authorID uuid.UUID, asAdmin bool) spec.PostCommentDelete {
	action := audit.ActionPostCommentDelete
	if authorID != userID {
		action = audit.ActionPostCommentDeleteAdmin
	}

	return spec.PostCommentDelete{
		CommentDeletion: spec.CommentDeletion{
			CommentID: id,
			UserID:    userID,
			AsAdmin:   asAdmin,
		},
		Audit: audit.NewEntry{
			ActorID:    userID,
			Action:     action,
			TargetType: audit.TargetPostComment,
			TargetID:   id.String(),
			SubjectID:  authorID,
		},
	}
}

func TestBlankBodyIsRejected(t *testing.T) {
	cases := []struct {
		name string
		call func(svc *service) error
	}{
		{
			name: "CreatePost without a share",
			call: func(svc *service) error {
				_, err := svc.CreatePost(context.Background(), uuid.New(), dto.CreatePostRequest{Corner: "general", Body: "   "})

				return err
			},
		},
		{
			name: "UpdatePost",
			call: func(svc *service) error {
				return svc.UpdatePost(context.Background(), uuid.New(), uuid.New(), dto.UpdatePostRequest{Body: "  "})
			},
		},
		{
			name: "CreateComment",
			call: func(svc *service) error {
				_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), dto.CreateCommentRequest{Body: "  "})

				return err
			},
		},
		{
			name: "UpdateComment",
			call: func(svc *service) error {
				return svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), dto.UpdateCommentRequest{Body: " "})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _ := newTestService(t)

			// when
			err := tc.call(svc)

			// then
			require.ErrorIs(t, err, ErrEmptyBody)
		})
	}
}

func TestCreatePost_Rejected(t *testing.T) {
	countErr := errors.New("db down")
	hello := dto.CreatePostRequest{Corner: "general", Body: "hello"}

	cases := []struct {
		name        string
		req         dto.CreatePostRequest
		postsPerDay *int
		postsToday  *int
		countErr    error
		wantErr     error
	}{
		{name: "an unknown share type", req: dto.CreatePostRequest{Corner: "general", Body: "hello", SharedContentID: "abc", SharedContentType: "bogus"}, wantErr: ErrInvalidShareType},
		{name: "a failed daily post count", req: hello, postsPerDay: new(5), postsToday: new(0), countErr: countErr, wantErr: countErr},
		{name: "the daily post limit", req: hello, postsPerDay: new(3), postsToday: new(3), wantErr: ErrRateLimited},
		{name: "a poll with a single option", req: pollPostRequest(3600, "only"), postsPerDay: new(0), wantErr: ErrInvalidPoll},
		{name: "a poll with a blank option label", req: pollPostRequest(3600, "a", "   "), postsPerDay: new(0), wantErr: ErrInvalidPoll},
		{name: "a poll with an unsupported duration", req: pollPostRequest(123, "a", "b"), postsPerDay: new(0), wantErr: ErrInvalidDuration},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()

			if tc.postsPerDay != nil {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(*tc.postsPerDay)
			}
			if tc.postsToday != nil {
				m.postRepo.EXPECT().CountUserPostsToday(mock.Anything, userID).Return(*tc.postsToday, tc.countErr)
			}

			// when
			_, err := svc.CreatePost(context.Background(), userID, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestCreatePost_RejectedByContentFilter(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	rule := contentfilter.NewMockRule(t)
	rule.EXPECT().Check(mock.Anything, []string{"https://giphy.com/gifs/abc123"}).Return(&contentfilter.Rejection{Rule: "test_reject", Reason: "nope", Detail: "xyz"}, nil)
	svc.contentFilter = contentfilter.New(rule)

	// when
	_, err := svc.CreatePost(context.Background(), uuid.New(), dto.CreatePostRequest{Corner: "general", Body: "https://giphy.com/gifs/abc123"})

	// then
	var rej *contentfilter.RejectedError
	require.ErrorAs(t, err, &rej)
	assert.Equal(t, "xyz", rej.Rejection.Detail)
}

func TestCreatePost(t *testing.T) {
	createErr := errors.New("boom")
	sharedID := uuid.New().String()
	hello := spec.NewPost{Corner: "general", Body: "hello"}
	helloWithPoll := spec.NewPost{Corner: "general", Body: "hello", Poll: &spec.NewPoll{DurationSeconds: 3600, Options: []string{"a", "b"}}}

	cases := []struct {
		name      string
		req       dto.CreatePostRequest
		want      spec.NewPost
		createErr error
	}{
		{name: "an empty corner defaults to general", req: dto.CreatePostRequest{Body: "hello"}, want: hello},
		{name: "a poll is created with the post", req: pollPostRequest(3600, "a", "b"), want: helloWithPoll},
		{name: "a share needs no body", req: dto.CreatePostRequest{SharedContentID: sharedID, SharedContentType: "theory"}, want: spec.NewPost{Corner: "general", SharedContent: &model.SharedContentRef{ID: sharedID, Type: "theory"}}},
		{name: "a create failure is surfaced", req: dto.CreatePostRequest{Corner: "general", Body: "hello"}, want: hello, createErr: createErr},
		{name: "a create failure with a poll is surfaced", req: pollPostRequest(3600, "a", "b"), want: helloWithPoll, createErr: createErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			postID := uuid.New()
			want := tc.want
			want.UserID = userID

			var created *model.PostRow
			if tc.createErr == nil {
				created = &model.PostRow{ID: postID}
			}

			m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
			m.postRepo.EXPECT().CreateWithDetails(mock.Anything, matchNewPost(want)).Return(created, tc.createErr)
			if tc.createErr == nil {
				expectBackgroundSocial(m)
			}

			// when
			id, err := svc.CreatePost(context.Background(), userID, tc.req)

			// then
			if tc.createErr != nil {
				require.ErrorIs(t, err, tc.createErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, postID, id)
		})
	}
}

func TestCreatePost_MentionNotifiesTheNamedUserOutsideSuggestions(t *testing.T) {
	tests := []struct {
		name       string
		corner     string
		wantNotify bool
	}{
		{name: "the general corner pings the named user", corner: "general", wantNotify: true},
		{name: "the suggestions corner pings nobody", corner: "suggestions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			postID := uuid.New()
			mentionedID := uuid.New()

			m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
			m.postRepo.EXPECT().
				CreateWithDetails(mock.Anything, spec.NewPost{UserID: userID, Corner: tt.corner, Body: "look at this @alice"}).
				Return(&model.PostRow{ID: postID}, nil)

			var wg sync.WaitGroup
			var mentioned dto.NotifyParams

			if tt.wantNotify {
				wg.Add(1)
				m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Battler"}, nil)
				m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)
				m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
					RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
						mentioned = p
						wg.Done()

						return nil
					})
			} else {
				expectBackgroundSocial(m)
			}

			// when
			id, err := svc.CreatePost(context.Background(), userID, dto.CreatePostRequest{Corner: tt.corner, Body: "look at this @alice"})

			// then
			require.NoError(t, err)
			assert.Equal(t, postID, id)

			if !tt.wantNotify {
				m.notifSvc.AssertNotCalled(t, "Notify", mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
					return p.Type == dto.NotifMention
				}))

				return
			}

			wg.Wait()
			assert.Equal(t, dto.NotifMention, mentioned.Type)
			assert.Equal(t, mentionedID, mentioned.RecipientID)
			assert.Equal(t, postID, mentioned.ReferenceID)
			assert.Equal(t, "post", mentioned.ReferenceType)
			assert.Equal(t, "/game-board/"+postID.String(), mentioned.EmailLink)
		})
	}
}

func TestGetPost_LookupFailure(t *testing.T) {
	getErr := errors.New("boom")

	cases := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "a repo failure is surfaced", getErr: getErr, wantErr: getErr},
		{name: "a missing post is not found", wantErr: ErrNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			viewer := uuid.New()
			m.postRepo.EXPECT().GetByID(mock.Anything, spec.PostLookup{ID: id, ViewerID: viewer}).Return(nil, tc.getErr)

			// when
			_, err := svc.GetPost(context.Background(), id, viewer, "")

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestGetPost(t *testing.T) {
	cases := []struct {
		name          string
		anonymous     bool
		viewHash      string
		shareCount    int
		wantViewCount int
	}{
		{name: "a signed in viewer with a new view hash bumps the view count", viewHash: "hash", wantViewCount: 1},
		{name: "an anonymous viewer without a view hash records no view and is never blocked", anonymous: true, shareCount: 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			authorID := uuid.New()
			viewer := uuid.New()
			if tc.anonymous {
				viewer = uuid.Nil
			}

			m.postRepo.EXPECT().GetByID(mock.Anything, spec.PostLookup{ID: id, ViewerID: viewer}).Return(&model.PostRow{ID: id, UserID: authorID}, nil)
			if tc.viewHash != "" {
				m.postRepo.EXPECT().RecordView(mock.Anything, spec.ViewRecord{TargetID: id, ViewerHash: tc.viewHash}).Return(true, nil)
			}
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
			m.postRepo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil)
			m.postRepo.EXPECT().GetComments(mock.Anything, spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: viewer, Limit: 500}).Return(nil, 0, nil)
			m.postRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
			m.postRepo.EXPECT().GetLikedBy(mock.Anything, spec.LikedByQuery{TargetID: id}).Return(nil, nil)
			if !tc.anonymous {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewer, authorID).Return(false, nil)
			}
			m.postRepo.EXPECT().GetPollByPostID(mock.Anything, spec.PostPollQuery{PostID: id, ViewerID: viewer}).Return(nil, nil, nil, nil)
			m.postRepo.EXPECT().GetShareCount(mock.Anything, model.SharedContentRef{ID: id.String(), Type: "post"}).Return(tc.shareCount, nil)

			// when
			got, err := svc.GetPost(context.Background(), id, viewer, tc.viewHash)

			// then
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.wantViewCount, got.ViewCount)
			assert.Equal(t, tc.shareCount, got.ShareCount)
			assert.False(t, got.ViewerBlocked)
		})
	}
}

func TestGetPost_SurfacesAFailedLookup(t *testing.T) {
	boom := errors.New("boom")

	cases := []struct {
		name   string
		failAt int
	}{
		{name: "the viewer's block list", failAt: 0},
		{name: "the post media", failAt: 1},
		{name: "the comments", failAt: 2},
		{name: "the comment media", failAt: 3},
		{name: "the likes", failAt: 4},
		{name: "the block check against the author", failAt: 5},
		{name: "the poll", failAt: 6},
		{name: "the shared content preview", failAt: 7},
		{name: "the share count", failAt: 8},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced, not rendered as empty", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			authorID := uuid.New()
			viewer := uuid.New()
			sharedID := uuid.New().String()
			sharedType := "art"
			m.postRepo.EXPECT().GetByID(mock.Anything, spec.PostLookup{ID: id, ViewerID: viewer}).Return(&model.PostRow{ID: id, UserID: authorID, SharedContentID: &sharedID, SharedContentType: &sharedType}, nil)

			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, postFailAt(tc.failAt, 0, boom))
			if tc.failAt >= 1 {
				m.postRepo.EXPECT().GetMedia(mock.Anything, id).Return(nil, postFailAt(tc.failAt, 1, boom))
			}
			if tc.failAt >= 2 {
				m.postRepo.EXPECT().GetComments(mock.Anything, mock.Anything).Return(nil, 0, postFailAt(tc.failAt, 2, boom))
			}
			if tc.failAt >= 3 {
				m.postRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, postFailAt(tc.failAt, 3, boom))
			}
			if tc.failAt >= 4 {
				m.postRepo.EXPECT().GetLikedBy(mock.Anything, mock.Anything).Return(nil, postFailAt(tc.failAt, 4, boom))
			}
			if tc.failAt >= 5 {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewer, authorID).Return(false, postFailAt(tc.failAt, 5, boom))
			}
			if tc.failAt >= 6 {
				m.postRepo.EXPECT().GetPollByPostID(mock.Anything, spec.PostPollQuery{PostID: id, ViewerID: viewer}).Return(nil, nil, nil, postFailAt(tc.failAt, 6, boom))
			}
			if tc.failAt >= 7 {
				m.postRepo.EXPECT().GetSharedContentPreviews(mock.Anything, []model.SharedContentRef{{ID: sharedID, Type: sharedType}}).Return(nil, postFailAt(tc.failAt, 7, boom))
			}
			if tc.failAt >= 8 {
				m.postRepo.EXPECT().GetShareCount(mock.Anything, model.SharedContentRef{ID: id.String(), Type: "post"}).Return(0, postFailAt(tc.failAt, 8, boom))
			}

			// when
			got, err := svc.GetPost(context.Background(), id, viewer, "")

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestGetPost_AFailedViewRecordStillRendersThePost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	m.postRepo.EXPECT().GetByID(mock.Anything, spec.PostLookup{ID: id, ViewerID: viewer}).Return(&model.PostRow{ID: id, UserID: uuid.New()}, nil)
	m.postRepo.EXPECT().RecordView(mock.Anything, spec.ViewRecord{TargetID: id, ViewerHash: "hash"}).Return(false, errors.New("boom"))
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.postRepo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil)
	m.postRepo.EXPECT().GetComments(mock.Anything, mock.Anything).Return(nil, 0, nil)
	m.postRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetLikedBy(mock.Anything, mock.Anything).Return(nil, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, mock.Anything).Return(nil, nil, nil, nil)
	m.postRepo.EXPECT().GetShareCount(mock.Anything, mock.Anything).Return(0, nil)

	// when
	got, err := svc.GetPost(context.Background(), id, viewer, "hash")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 0, got.ViewCount, "an unrecorded view is not counted")
}

func postFailAt(failAt int, step int, err error) error {
	if failAt == step {
		return err
	}

	return nil
}

func TestUpdatePost(t *testing.T) {
	updateErr := errors.New("boom")

	cases := []struct {
		name      string
		asAdmin   bool
		updateErr error
	}{
		{name: "an editor of any post updates as admin", asAdmin: true},
		{name: "an admin update failure is surfaced", asAdmin: true, updateErr: updateErr},
		{name: "an owner updates without admin rights"},
		{name: "an owner update failure is surfaced", updateErr: updateErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(tc.asAdmin)
			m.postRepo.EXPECT().UpdateWithDetails(mock.Anything, spec.PostUpdate{ID: id, UserID: userID, Body: "body", AsAdmin: tc.asAdmin}).Return(tc.updateErr)
			if tc.updateErr == nil {
				expectBackgroundSocial(m)
			}

			// when
			err := svc.UpdatePost(context.Background(), id, userID, dto.UpdatePostRequest{Body: " body "})

			// then
			require.ErrorIs(t, err, tc.updateErr)
		})
	}
}

func TestDeletePost(t *testing.T) {
	deleteErr := errors.New("boom")

	cases := []struct {
		name         string
		byAuthor     bool
		canDeleteAny bool
		mediaCount   int
		shared       *model.SharedContentRef
		paths        []string
		deleteErr    error
	}{
		{name: "an admin deletes someone else's post as an admin action", canDeleteAny: true, mediaCount: 2},
		{name: "an owner deletes their own post", byAuthor: true},
		{name: "a moderator deleting their own post is not an admin action", byAuthor: true, canDeleteAny: true},
		{name: "a shared post decrements the shared content's share count", byAuthor: true, shared: &model.SharedContentRef{ID: "shared-abc", Type: "theory"}},
		{name: "the media paths the repo returns are unlinked", byAuthor: true, mediaCount: 2, paths: []string{"/uploads/posts/a.webp", "/uploads/posts/a_thumb.webp", "/uploads/posts/c.webp"}},
		{name: "a repo failure unlinks nothing", byAuthor: true, mediaCount: 1, paths: []string{"/uploads/posts/a.webp"}, deleteErr: deleteErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				id := uuid.New()
				userID := uuid.New()
				authorID := uuid.New()
				if tc.byAuthor {
					authorID = userID
				}

				m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, id).Return(authorID, nil)
				m.postRepo.EXPECT().GetMedia(mock.Anything, id).Return(make([]model.PostMediaRow, tc.mediaCount), nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(tc.canDeleteAny)
				m.postRepo.EXPECT().DeleteWithSharedContent(mock.Anything, postDeleteSpec(id, userID, authorID, tc.canDeleteAny, tc.mediaCount)).Return(tc.shared, tc.paths, tc.deleteErr)
				if tc.deleteErr == nil {
					expectUnlinked(m, tc.paths)
				}
				if tc.shared != nil {
					m.postRepo.EXPECT().DecrementShareCount(mock.Anything, *tc.shared).Return(nil)
				}

				// when
				err := svc.DeletePost(context.Background(), id, userID)
				synctest.Wait()

				// then
				require.ErrorIs(t, err, tc.deleteErr)
				if tc.deleteErr != nil {
					m.uploadSvc.AssertNotCalled(t, "Delete", mock.Anything)
				}
			})
		})
	}
}

func TestListFeed(t *testing.T) {
	listErr := errors.New("boom")

	cases := []struct {
		name      string
		tab       string
		anonymous bool
		search    string
		limit     int
		resolved  string
		rows      []model.PostRow
		listErr   error
	}{
		{name: "the following tab lists followed authors", tab: "following", limit: 10, rows: []model.PostRow{{ID: uuid.New()}}},
		{name: "the all tab passes the search and resolved filters through", tab: "all", anonymous: true, search: "q", limit: 5, resolved: "resolved"},
		{name: "a list failure is surfaced", tab: "all", limit: 10, listErr: listErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewer := uuid.New()
			if tc.anonymous {
				viewer = uuid.Nil
			}

			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
			if tc.tab == "following" {
				m.postRepo.EXPECT().ListByFollowing(mock.Anything, spec.PostFollowingFeedQuery{UserID: viewer, Corner: "general", Sort: "new", Limit: tc.limit}).Return(tc.rows, len(tc.rows), tc.listErr)
			} else {
				m.postRepo.EXPECT().ListAll(mock.Anything, spec.PostFeedQuery{ViewerID: viewer, Corner: "general", Search: tc.search, Sort: "new", Limit: tc.limit, ResolvedFilter: tc.resolved}).Return(tc.rows, len(tc.rows), tc.listErr)
			}
			if tc.listErr == nil {
				expectPostListBatch(m, tc.rows, viewer)
			}

			// when
			got, err := svc.ListFeed(context.Background(), tc.tab, viewer, "", tc.search, "new", 0, bounds.NewPage(tc.limit, 0), tc.resolved)

			// then
			if tc.listErr != nil {
				require.ErrorIs(t, err, tc.listErr)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, len(tc.rows), got.Total)
		})
	}
}

func TestListUserPosts(t *testing.T) {
	cases := []struct {
		name    string
		listErr error
	}{
		{name: "the repo total is passed through"},
		{name: "a list failure is surfaced", listErr: errors.New("boom")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			target := uuid.New()
			viewer := uuid.New()
			m.postRepo.EXPECT().ListByUser(mock.Anything, spec.PostUserPage{UserID: target, ViewerID: viewer, Limit: 10}).Return(nil, 2, tc.listErr)
			if tc.listErr == nil {
				expectPostListBatch(m, nil, viewer)
			}

			// when
			got, err := svc.ListUserPosts(context.Background(), target, viewer, bounds.NewPage(10, 0))

			// then
			if tc.listErr != nil {
				require.ErrorIs(t, err, tc.listErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, 2, got.Total)
		})
	}
}

func TestListFeed_AFailedBlockListIsSurfacedBeforeAnythingIsListed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	boom := errors.New("boom")
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, boom)

	// when
	got, err := svc.ListFeed(context.Background(), "all", viewer, "", "", "new", 0, bounds.NewPage(10, 0), "")

	// then
	require.ErrorIs(t, err, boom, "listing without the block list would show blocked users' posts")
	assert.Nil(t, got)
}

func TestListUserPosts_SurfacesAFailedBatchLookup(t *testing.T) {
	boom := errors.New("boom")

	cases := []struct {
		name   string
		failAt int
	}{
		{name: "the media", failAt: 0},
		{name: "the polls", failAt: 1},
		{name: "the shared content previews", failAt: 2},
		{name: "the share counts", failAt: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced, not rendered as empty", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			target := uuid.New()
			viewer := uuid.New()
			m.postRepo.EXPECT().ListByUser(mock.Anything, mock.Anything).Return([]model.PostRow{{ID: uuid.New()}}, 1, nil)

			m.postRepo.EXPECT().GetMediaBatch(mock.Anything, mock.Anything).Return(nil, postFailAt(tc.failAt, 0, boom))
			if tc.failAt >= 1 {
				m.postRepo.EXPECT().GetPollsByPostIDs(mock.Anything, mock.Anything).Return(nil, nil, nil, postFailAt(tc.failAt, 1, boom))
			}
			if tc.failAt >= 2 {
				m.postRepo.EXPECT().GetSharedContentPreviews(mock.Anything, mock.Anything).Return(nil, postFailAt(tc.failAt, 2, boom))
			}
			if tc.failAt >= 3 {
				m.postRepo.EXPECT().GetShareCountsBatch(mock.Anything, mock.Anything).Return(nil, postFailAt(tc.failAt, 3, boom))
			}

			// when
			got, err := svc.ListUserPosts(context.Background(), target, viewer, bounds.NewPage(10, 0))

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestAuthorGuards(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	blockErr := errors.New("block lookup failed")
	lookupFailureSurfaced := authorGuard{name: "surfaces an author lookup failure", lookupErr: lookupErr, wantErr: lookupErr}
	vanishedIsNotFound := authorGuard{name: "maps a vanished target to not found", lookupErr: fmt.Errorf("get author: %w", dao.ErrNotFound), wantErr: ErrNotFound}
	blockedEitherWay := authorGuard{name: "is refused across a block", blocked: true, wantErr: block.ErrUserBlocked}
	blockLookupFails := authorGuard{name: "is refused when the block lookup fails", blockErr: blockErr, wantErr: blockErr}
	notPostAuthor := authorGuard{name: "refuses anyone but the post author", wantMsg: "not the post author"}

	cases := []struct {
		name      string
		ofComment bool
		call      func(svc *service, ctx context.Context, targetID uuid.UUID, userID uuid.UUID) error
		guards    []authorGuard
	}{
		{
			name:   "DeletePost",
			call:   (*service).DeletePost,
			guards: []authorGuard{lookupFailureSurfaced, vanishedIsNotFound},
		},
		{
			name:   "UploadPostMedia",
			call:   uploadPostMedia,
			guards: []authorGuard{vanishedIsNotFound, notPostAuthor},
		},
		{
			name: "DeletePostMedia",
			call: func(svc *service, ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
				return svc.DeletePostMedia(ctx, postID, 1, userID)
			},
			guards: []authorGuard{vanishedIsNotFound, notPostAuthor},
		},
		{
			name:      "UploadCommentMedia",
			ofComment: true,
			call:      uploadCommentMedia,
			guards: []authorGuard{
				lookupFailureSurfaced,
				vanishedIsNotFound,
				{name: "refuses anyone but the comment author", wantMsg: "not the comment author"},
			},
		},
		{
			name: "LikePost",
			call: func(svc *service, ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
				return svc.LikePost(ctx, userID, postID)
			},
			guards: []authorGuard{lookupFailureSurfaced, vanishedIsNotFound, blockedEitherWay, blockLookupFails},
		},
		{
			name: "CreateComment",
			call: func(svc *service, ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
				_, err := svc.CreateComment(ctx, postID, userID, dto.CreateCommentRequest{Body: "hi"})

				return err
			},
			guards: []authorGuard{lookupFailureSurfaced, vanishedIsNotFound, blockedEitherWay, blockLookupFails},
		},
		{
			name:      "UpdateComment",
			ofComment: true,
			call: func(svc *service, ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error {
				return svc.UpdateComment(ctx, commentID, userID, dto.UpdateCommentRequest{Body: "body"})
			},
			guards: []authorGuard{lookupFailureSurfaced, vanishedIsNotFound},
		},
		{
			name:      "DeleteComment",
			ofComment: true,
			call:      (*service).DeleteComment,
			guards:    []authorGuard{lookupFailureSurfaced, vanishedIsNotFound},
		},
		{
			name:      "LikeComment",
			ofComment: true,
			call: func(svc *service, ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error {
				return svc.LikeComment(ctx, userID, commentID)
			},
			guards: []authorGuard{lookupFailureSurfaced, vanishedIsNotFound, blockedEitherWay, blockLookupFails},
		},
	}

	for _, tc := range cases {
		for _, guard := range tc.guards {
			t.Run(tc.name+" "+guard.name, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				targetID := uuid.New()
				userID := uuid.New()
				authorID := uuid.New()
				if guard.lookupErr != nil {
					authorID = uuid.Nil
				}

				expectAuthorLookup(m, tc.ofComment, targetID, authorID, guard.lookupErr)
				if guard.blocked || guard.blockErr != nil {
					m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(guard.blocked, guard.blockErr)
				}

				// when
				err := tc.call(svc, context.Background(), targetID, userID)

				// then
				if guard.wantMsg != "" {
					require.ErrorContains(t, err, guard.wantMsg)
				} else {
					require.ErrorIs(t, err, guard.wantErr)
				}

				if !errors.Is(guard.wantErr, ErrNotFound) {
					require.NotErrorIs(t, err, ErrNotFound)
				}
			})
		}
	}
}

func TestUploadMedia_SaveFailureIsSurfaced(t *testing.T) {
	cases := []struct {
		name      string
		ofComment bool
		upload    func(svc *service, ctx context.Context, targetID uuid.UUID, userID uuid.UUID) error
	}{
		{name: "post media", upload: uploadPostMedia},
		{name: "comment media", ofComment: true, upload: uploadCommentMedia},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			targetID := uuid.New()
			userID := uuid.New()
			saveErr := errors.New("upload fail")
			expectAuthorLookup(m, tc.ofComment, targetID, userID, nil)
			m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
			m.uploadSvc.EXPECT().SaveImage(mock.Anything, "posts", mock.Anything, int64(10), int64(1000), mock.Anything).Return("", saveErr)

			// when
			err := tc.upload(svc, context.Background(), targetID, userID)

			// then
			require.ErrorIs(t, err, saveErr)
		})
	}
}

func TestDeletePostMedia(t *testing.T) {
	cases := []struct {
		name      string
		mediaURL  string
		deleteErr error
	}{
		{name: "the removed file is unlinked", mediaURL: "/uploads/posts/a.webp"},
		{name: "a repo failure is surfaced", deleteErr: errors.New("boom")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			postID := uuid.New()
			userID := uuid.New()
			m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(userID, nil)
			m.postRepo.EXPECT().DeleteMedia(mock.Anything, spec.MediaDeletion{ID: 1, TargetID: postID}).Return(tc.mediaURL, tc.deleteErr)
			if tc.deleteErr == nil {
				m.uploadSvc.EXPECT().Delete([]string{tc.mediaURL}).Return()
			}

			// when
			err := svc.DeletePostMedia(context.Background(), postID, 1, userID)

			// then
			require.ErrorIs(t, err, tc.deleteErr)
		})
	}
}

func TestLikeAndUnlike(t *testing.T) {
	cases := []struct {
		name  string
		given func(m *testMocks, userID uuid.UUID, targetID uuid.UUID, repoErr error)
		call  func(svc *service, ctx context.Context, userID uuid.UUID, targetID uuid.UUID) error
	}{
		{
			name: "LikePost",
			given: func(m *testMocks, userID uuid.UUID, postID uuid.UUID, repoErr error) {
				authorID := uuid.New()
				m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
				m.postRepo.EXPECT().Like(mock.Anything, spec.Like{UserID: userID, TargetID: postID}).Return(repoErr)
				if repoErr == nil {
					expectBackgroundSocial(m)
				}
			},
			call: (*service).LikePost,
		},
		{
			name: "UnlikePost",
			given: func(m *testMocks, userID uuid.UUID, postID uuid.UUID, repoErr error) {
				m.postRepo.EXPECT().Unlike(mock.Anything, spec.Like{UserID: userID, TargetID: postID}).Return(repoErr)
			},
			call: (*service).UnlikePost,
		},
		{
			name: "LikeComment",
			given: func(m *testMocks, userID uuid.UUID, commentID uuid.UUID, repoErr error) {
				authorID := uuid.New()
				m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
				m.postRepo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: commentID}).Return(repoErr)
				if repoErr == nil {
					expectBackgroundSocial(m)
				}
			},
			call: (*service).LikeComment,
		},
		{
			name: "UnlikeComment",
			given: func(m *testMocks, userID uuid.UUID, commentID uuid.UUID, repoErr error) {
				m.postRepo.EXPECT().UnlikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: commentID}).Return(repoErr)
			},
			call: (*service).UnlikeComment,
		},
	}

	outcomes := []struct {
		name    string
		repoErr error
	}{
		{name: "succeeds"},
		{name: "surfaces a repo failure", repoErr: errors.New("boom")},
	}

	for _, tc := range cases {
		for _, outcome := range outcomes {
			t.Run(tc.name+" "+outcome.name, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				userID := uuid.New()
				targetID := uuid.New()
				tc.given(m, userID, targetID, outcome.repoErr)

				// when
				err := tc.call(svc, context.Background(), userID, targetID)

				// then
				require.ErrorIs(t, err, outcome.repoErr)
			})
		}
	}
}

func TestCreateComment(t *testing.T) {
	cases := []struct {
		name      string
		reply     bool
		createErr error
	}{
		{name: "a reply is created under its parent", reply: true},
		{name: "a create failure is surfaced", createErr: errors.New("boom")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			postID := uuid.New()
			userID := uuid.New()
			authorID := uuid.New()
			commentID := uuid.New()

			var parentID *uuid.UUID
			if tc.reply {
				parentID = new(uuid.New())
			}

			var created *model.CommentRow
			if tc.createErr == nil {
				created = &model.CommentRow{ID: commentID}
			}

			m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
			m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
			m.postComments.EXPECT().CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{TargetID: postID, ParentID: parentID, UserID: userID, Body: "hi"}).Return(created, tc.createErr)
			if tc.createErr == nil {
				expectBackgroundSocial(m)
			}

			// when
			id, err := svc.CreateComment(context.Background(), postID, userID, dto.CreateCommentRequest{Body: "hi", ParentID: parentID})

			// then
			if tc.createErr != nil {
				require.ErrorIs(t, err, tc.createErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, commentID, id)
		})
	}
}

func TestCreateComment_MentionNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	commentID := uuid.New()
	mentionedID := uuid.New()

	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.postComments.EXPECT().
		CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{TargetID: postID, ParentID: nil, UserID: userID, Body: "look at this @alice"}).
		Return(&model.CommentRow{ID: commentID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Battler"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)

	var wg sync.WaitGroup
	wg.Add(2)

	var mentioned dto.NotifyParams
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			if p.Type == dto.NotifMention {
				mentioned = p
			}
			wg.Done()

			return nil
		})

	// when
	id, err := svc.CreateComment(context.Background(), postID, userID, dto.CreateCommentRequest{Body: "look at this @alice"})

	// then
	require.NoError(t, err)
	assert.Equal(t, commentID, id)
	wg.Wait()
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, postID, mentioned.ReferenceID)
	assert.Equal(t, "post_comment:"+commentID.String(), mentioned.ReferenceType)
	assert.Equal(t, "/game-board/"+postID.String()+"#comment-"+commentID.String(), mentioned.EmailLink)
}

func TestUpdateComment(t *testing.T) {
	updateErr := errors.New("boom")

	cases := []struct {
		name       string
		byAuthor   bool
		canEditAny bool
		updateErr  error
		wantAudit  bool
	}{
		{name: "an admin editing someone else's comment is audited", canEditAny: true, wantAudit: true},
		{name: "a moderator editing their own comment is not audited", byAuthor: true, canEditAny: true},
		{name: "an admin update failure is surfaced", canEditAny: true, updateErr: updateErr},
		{name: "an owner edits their own comment", byAuthor: true},
		{name: "an owner update failure is surfaced", byAuthor: true, updateErr: updateErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			authorID := uuid.New()
			if tc.byAuthor {
				authorID = userID
			}

			m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(authorID, nil)
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(tc.canEditAny)
			m.postRepo.EXPECT().UpdateCommentWithDetails(mock.Anything, spec.CommentUpdate{CommentID: id, UserID: userID, Body: "body", AsAdmin: tc.canEditAny}).Return(tc.updateErr)
			if tc.wantAudit {
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    userID,
					Action:     audit.ActionPostCommentUpdateAdmin,
					TargetType: audit.TargetPostComment,
					TargetID:   id.String(),
					SubjectID:  authorID,
				}).Return(nil)
			}
			if tc.updateErr == nil {
				expectBackgroundSocial(m)
			}

			// when
			err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "body"})

			// then
			require.ErrorIs(t, err, tc.updateErr)
			if !tc.wantAudit {
				m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestDeleteComment(t *testing.T) {
	cases := []struct {
		name         string
		byAuthor     bool
		canDeleteAny bool
		paths        []string
		deleteErr    error
	}{
		{name: "an admin deletes someone else's comment as an admin action", canDeleteAny: true},
		{name: "a moderator deleting their own comment is not an admin action", byAuthor: true, canDeleteAny: true},
		{name: "an owner deletes their own comment", byAuthor: true},
		{name: "the media paths the repo returns are unlinked", byAuthor: true, paths: []string{"/uploads/posts/c.webp", "/uploads/posts/c_thumb.webp"}},
		{name: "a repo failure unlinks nothing", byAuthor: true, paths: []string{"/uploads/posts/c.webp"}, deleteErr: errors.New("boom")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			authorID := uuid.New()
			if tc.byAuthor {
				authorID = userID
			}

			m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(authorID, nil)
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(tc.canDeleteAny)
			m.postRepo.EXPECT().DeleteCommentWithAudit(mock.Anything, postCommentDeleteSpec(id, userID, authorID, tc.canDeleteAny)).Return(tc.paths, tc.deleteErr)
			if tc.deleteErr == nil {
				expectUnlinked(m, tc.paths)
			}

			// when
			err := svc.DeleteComment(context.Background(), id, userID)

			// then
			require.ErrorIs(t, err, tc.deleteErr)
			if tc.deleteErr != nil {
				m.uploadSvc.AssertNotCalled(t, "Delete", mock.Anything)
			}
		})
	}
}

func TestGetCornerCounts(t *testing.T) {
	cases := []struct {
		name    string
		counts  map[string]int
		repoErr error
	}{
		{name: "the repo counts are returned", counts: map[string]int{"general": 3}},
		{name: "a repo failure is surfaced", repoErr: errors.New("boom")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.postRepo.EXPECT().GetCornerCounts(mock.Anything).Return(tc.counts, tc.repoErr)

			// when
			got, err := svc.GetCornerCounts(context.Background())

			// then
			require.ErrorIs(t, err, tc.repoErr)
			assert.Equal(t, tc.counts, got)
		})
	}
}

func TestVotePoll_Refused(t *testing.T) {
	pollErr := errors.New("boom")
	authorErr := errors.New("author lookup failed")
	blockErr := errors.New("block lookup failed")

	cases := []struct {
		name        string
		poll        *model.PollRow
		votedOption *int
		pollErr     error
		authorErr   error
		blocked     bool
		blockErr    error
		vote        int
		wantErr     error
	}{
		{name: "a poll lookup failure is surfaced", pollErr: pollErr, vote: 1, wantErr: pollErr},
		{name: "a post without a poll is not found", vote: 1, wantErr: ErrNotFound},
		{name: "an author lookup failure is surfaced", poll: pollExpiringIn(time.Hour), authorErr: authorErr, vote: 1, wantErr: authorErr},
		{name: "a post that vanished before the vote is not found", poll: pollExpiringIn(time.Hour), authorErr: fmt.Errorf("get author: %w", dao.ErrNotFound), vote: 1, wantErr: ErrNotFound},
		{name: "a voter blocked either way cannot vote", poll: pollExpiringIn(time.Hour), blocked: true, vote: 1, wantErr: block.ErrUserBlocked},
		{name: "a failed block lookup refuses the vote", poll: pollExpiringIn(time.Hour), blockErr: blockErr, vote: 1, wantErr: blockErr},
		{name: "a second vote is refused", poll: pollExpiringIn(time.Hour), votedOption: new(1), vote: 1, wantErr: ErrAlreadyVoted},
		{name: "an expired poll is closed", poll: pollExpiringIn(-time.Hour), vote: 1, wantErr: ErrPollExpired},
		{name: "an unknown option is refused", poll: pollExpiringIn(time.Hour), vote: 999, wantErr: ErrInvalidOption},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			postID := uuid.New()
			userID := uuid.New()
			authorID := uuid.New()
			options := []model.PollOptionRow{{ID: 1}, {ID: 2}}

			m.postRepo.EXPECT().GetPollByPostID(mock.Anything, spec.PostPollQuery{PostID: postID, ViewerID: userID}).Return(tc.poll, options, tc.votedOption, tc.pollErr).Once()
			if tc.poll != nil {
				m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, tc.authorErr)
			}
			if tc.poll != nil && tc.authorErr == nil {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(tc.blocked, tc.blockErr)
			}

			// when
			got, err := svc.VotePoll(context.Background(), postID, userID, tc.vote)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
			m.postRepo.AssertNotCalled(t, "VotePoll", mock.Anything, mock.Anything)
		})
	}
}

func TestVotePoll_AMalformedPollIDIsAnErrorNotAVoteForNobody(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	poll := pollExpiringIn(time.Hour)
	poll.ID = "not-a-uuid"
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, spec.PostPollQuery{PostID: postID, ViewerID: userID}).Return(poll, []model.PollOptionRow{{ID: 1}}, nil, nil)
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)

	// when
	got, err := svc.VotePoll(context.Background(), postID, userID, 1)

	// then
	require.Error(t, err)
	assert.Nil(t, got)
	m.postRepo.AssertNotCalled(t, "VotePoll", mock.Anything, mock.Anything)
}

func TestVotePoll(t *testing.T) {
	boom := errors.New("boom")
	lockedErr := errors.New("already voted table is locked")

	cases := []struct {
		name       string
		voteErr    error
		refreshErr error
		wantErr    error
	}{
		{name: "an unblocked voter is recorded and gets the refreshed poll"},
		{name: "a vote failure is surfaced", voteErr: boom, wantErr: boom},
		{name: "a duplicate vote caught by the database is already voted", voteErr: fmt.Errorf("vote: %w", dao.ErrAlreadyVoted), wantErr: ErrAlreadyVoted},
		{name: "an unrelated failure whose text mentions a vote is not mistaken for a duplicate", voteErr: lockedErr, wantErr: lockedErr},
		{name: "a refresh failure after voting is surfaced", refreshErr: boom, wantErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			postID := uuid.New()
			userID := uuid.New()
			authorID := uuid.New()
			poll := pollExpiringIn(time.Hour)
			pollID := uuid.MustParse(poll.ID)
			options := []model.PollOptionRow{{ID: 1}}
			query := spec.PostPollQuery{PostID: postID, ViewerID: userID}

			m.postRepo.EXPECT().GetPollByPostID(mock.Anything, query).Return(poll, options, nil, nil).Once()
			m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
			m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
			m.postRepo.EXPECT().VotePoll(mock.Anything, spec.PostPollVote{PollID: pollID, UserID: userID, OptionID: 1}).Return(tc.voteErr)
			if tc.voteErr == nil {
				m.postRepo.EXPECT().GetPollByPostID(mock.Anything, query).Return(poll, options, new(1), tc.refreshErr).Once()
			}

			// when
			got, err := svc.VotePoll(context.Background(), postID, userID, 1)

			// then
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestResolveSuggestion(t *testing.T) {
	cases := []struct {
		name           string
		allowed        bool
		status         string
		wantStatus     string
		repoErr        error
		notifiesAuthor bool
	}{
		{name: "a user without the permission is refused", status: "done"},
		{name: "an unknown status is normalised to done", allowed: true, status: "not-a-status", wantStatus: "done", notifiesAuthor: true},
		{name: "an archived suggestion stays archived", allowed: true, status: "archived", wantStatus: "archived"},
		{name: "a repo failure is surfaced", allowed: true, status: "done", wantStatus: "done", repoErr: errors.New("boom")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			postID := uuid.New()
			userID := uuid.New()
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermResolveSuggestion).Return(tc.allowed)
			if tc.allowed {
				m.postRepo.EXPECT().ResolveSuggestion(mock.Anything, spec.SuggestionResolution{PostID: postID, ResolvedBy: userID, Status: tc.wantStatus}).Return(tc.repoErr)
			}
			if tc.notifiesAuthor {
				expectBackgroundSocial(m)
			}

			// when
			err := svc.ResolveSuggestion(context.Background(), postID, userID, tc.status)

			// then
			if !tc.allowed {
				require.ErrorContains(t, err, "not authorised")

				return
			}

			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestUnresolveSuggestion(t *testing.T) {
	cases := []struct {
		name    string
		allowed bool
		repoErr error
	}{
		{name: "a user without the permission is refused"},
		{name: "a resolver reopens the suggestion", allowed: true},
		{name: "a repo failure is surfaced", allowed: true, repoErr: errors.New("boom")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			postID := uuid.New()
			userID := uuid.New()
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermResolveSuggestion).Return(tc.allowed)
			if tc.allowed {
				m.postRepo.EXPECT().UnresolveSuggestion(mock.Anything, postID).Return(tc.repoErr)
			}

			// when
			err := svc.UnresolveSuggestion(context.Background(), postID, userID)

			// then
			if !tc.allowed {
				require.ErrorContains(t, err, "not authorised")

				return
			}

			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestGetShareCount(t *testing.T) {
	cases := []struct {
		name    string
		count   int
		repoErr error
	}{
		{name: "the repo count is returned", count: 7},
		{name: "a repo failure is surfaced", repoErr: errors.New("boom")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.postRepo.EXPECT().GetShareCount(mock.Anything, model.SharedContentRef{ID: "abc", Type: "post"}).Return(tc.count, tc.repoErr)

			// when
			got, err := svc.GetShareCount(context.Background(), "abc", "post")

			// then
			require.ErrorIs(t, err, tc.repoErr)
			assert.Equal(t, tc.count, got)
		})
	}
}
