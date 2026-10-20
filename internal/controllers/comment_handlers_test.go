package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/contentfilter"
	ctrlutils "umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	commentDeps struct {
		svc *comment.MockService
	}

	commentAction struct {
		name    string
		method  string
		suffix  string
		body    any
		failure string
		expect  func(svc *comment.MockService, ref spec.CommentRef, userID uuid.UUID, err error)
	}

	commentErrorCase struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}
)

var (
	commentActions = []commentAction{
		{name: "update", method: "PUT", body: dto.UpdateCommentRequest{Body: "edited"}, failure: "failed to update comment", expect: func(svc *comment.MockService, ref spec.CommentRef, userID uuid.UUID, err error) {
			svc.EXPECT().Update(mock.Anything, ref, userID, "edited").Return(err).Once()
		}},
		{name: "delete", method: "DELETE", failure: "failed to delete comment", expect: func(svc *comment.MockService, ref spec.CommentRef, userID uuid.UUID, err error) {
			svc.EXPECT().Delete(mock.Anything, ref, userID).Return(err).Once()
		}},
		{name: "like", method: "POST", suffix: "/like", failure: "failed to like comment", expect: func(svc *comment.MockService, ref spec.CommentRef, userID uuid.UUID, err error) {
			svc.EXPECT().Like(mock.Anything, ref, userID).Return(err).Once()
		}},
		{name: "unlike", method: "DELETE", suffix: "/like", failure: "failed to unlike comment", expect: func(svc *comment.MockService, ref spec.CommentRef, userID uuid.UUID, err error) {
			svc.EXPECT().Unlike(mock.Anything, ref, userID).Return(err).Once()
		}},
	}

	commentErrorCases = []commentErrorCase{
		{name: "a content filter rejection", err: &contentfilter.RejectedError{Rejection: contentfilter.Rejection{Rule: "slurs", Reason: "nope", Detail: "slur"}}, wantCode: http.StatusBadRequest, wantBody: "content_rejected"},
		{name: "an empty body", err: comment.ErrEmptyBody, wantCode: http.StatusBadRequest, wantBody: "comment body is required"},
		{name: "a missing target", err: comment.ErrTargetNotFound, wantCode: http.StatusNotFound, wantBody: "the content being commented on was not found"},
		{name: "a missing comment", err: comment.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "comment not found"},
		{name: "a missing row from the database", err: fmt.Errorf("delete comment: %w", daoerr.ErrNotFound), wantCode: http.StatusNotFound, wantBody: "comment not found"},
		{name: "someone else's comment", err: comment.ErrForbidden, wantCode: http.StatusForbidden, wantBody: "not allowed to change this comment"},
		{name: "not the comment author", err: authz.ErrNotCommentAuthor, wantCode: http.StatusForbidden, wantBody: "not the comment author"},
		{name: "a block between the users", err: block.ErrUserBlocked, wantCode: http.StatusForbidden, wantBody: "user is blocked"},
		{name: "an upload rejection", err: fmt.Errorf("%w: file size 9MB exceeds maximum 5MB", upload.ErrFileTooLarge), wantCode: http.StatusBadRequest, wantBody: "exceeds maximum 5MB"},
		{name: "an upload rejection whose rollback failed hides the rollback", err: errors.Join(upload.ErrInvalidFileType, errors.New("roll back comment: pq: connection refused")), wantCode: http.StatusBadRequest, wantBody: `{"error":"` + upload.ErrInvalidFileType.Error() + `"}`},
	}
)

func newCommentHarness(t *testing.T) (*testutil.Harness, commentDeps) {
	h := testutil.NewHarness(t)
	deps := commentDeps{
		svc: comment.NewMockService(t),
	}
	s := &Service{
		CommentService:  deps.svc,
		SettingsService: h.SettingsService,
		AuthSession:     h.SessionManager,
		AuthService:     h.AuthService,
		AuthzService:    h.AuthzService,
	}
	for _, setup := range s.getAllCommentRoutes() {
		setup(h.App)
	}

	return h, deps
}

func commentRequest(h *testutil.Harness, action commentAction, path string) *testutil.Request {
	req := h.NewRequest(action.method, path+action.suffix).WithCookie("valid-cookie")
	if action.body != nil {
		req = req.WithJSONBody(action.body)
	}

	return req
}

func expectNewAccountSession(h *testutil.Harness, cookie string, userID uuid.UUID) {
	h.AuthService.EXPECT().
		Authenticate(mock.Anything, cookie).
		Return(model.SessionAuthState{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil).
		Maybe()
	h.SessionRepo.EXPECT().GetUserID(mock.Anything, cookie).Return(userID, time.Now().Add(time.Hour), nil).Maybe()
	h.AuthzService.EXPECT().IsBanned(mock.Anything, userID).Return(false).Maybe()
	h.AuthzService.EXPECT().IsRestrictedNewAccount(mock.Anything, userID).Return(true)
}

func declaredCommentKinds(t *testing.T) []spec.CommentKind {
	t.Helper()

	_, self, _, ok := runtime.Caller(0)
	require.True(t, ok)
	specDir := filepath.Join(filepath.Dir(self), "..", "model", "spec")

	entries, err := os.ReadDir(specDir)
	require.NoError(t, err)

	var kinds []spec.CommentKind
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(specDir, entry.Name()), nil, 0)
		require.NoError(t, err)

		for _, decl := range file.Decls {
			gen, isGen := decl.(*ast.GenDecl)
			if !isGen || gen.Tok != token.CONST {
				continue
			}

			for _, s := range gen.Specs {
				value := s.(*ast.ValueSpec)
				typ, isIdent := value.Type.(*ast.Ident)
				if !isIdent || typ.Name != "CommentKind" {
					continue
				}

				for _, v := range value.Values {
					kind, err := strconv.Unquote(v.(*ast.BasicLit).Value)
					require.NoError(t, err)
					kinds = append(kinds, spec.CommentKind(kind))
				}
			}
		}
	}
	require.NotEmpty(t, kinds)

	return kinds
}

func TestCommentRoutes_EveryDeclaredCommentKindIsWired(t *testing.T) {
	// given every CommentKind constant in spec, the DAO's registry and the route table
	routed := make(map[spec.CommentKind]bool, len(commentRoutes))
	for _, route := range commentRoutes {
		routed[route.kind] = true
	}
	comments := dao.NewComments(nil)

	for _, kind := range declaredCommentKinds(t) {
		t.Run(string(kind), func(t *testing.T) {
			// when
			_, err := comments.GetCommentMediaDir(kind)

			// then
			require.NoError(t, err, "register a querier for this kind in dao.NewComments")

			if kind == spec.CommentKindJournalEntry {
				assert.True(t, routed[spec.CommentKindJournal], "journal entry comments share the journal_comments table, so the journal routes reach them")

				return
			}

			assert.True(t, routed[kind], "add this kind to commentRoutes so its comments can be edited, deleted, liked and given media")
		})
	}
}

func TestCommentRoutes_EveryKindReachesTheServiceWithItsOwnKind(t *testing.T) {
	for _, route := range commentRoutes {
		for _, action := range commentActions {
			t.Run(string(route.kind)+" "+action.name, func(t *testing.T) {
				// given
				h, deps := newCommentHarness(t)
				userID := uuid.New()
				commentID := uuid.New()
				h.ExpectValidSession("valid-cookie", userID)
				action.expect(deps.svc, spec.CommentRef{Kind: route.kind, CommentID: commentID}, userID, nil)

				// when
				status, body := commentRequest(h, action, route.path+"/"+commentID.String()).Do()

				// then
				require.Equal(t, http.StatusNoContent, status, string(body))
			})
		}
	}
}

func TestCommentRoutes_RequireASession(t *testing.T) {
	for _, route := range commentRoutes {
		for _, action := range commentActions {
			t.Run(string(route.kind)+" "+action.name, func(t *testing.T) {
				testutil.RunAuthFailureSuite(t, newCommentHarness, action.method, route.path+"/"+uuid.NewString()+action.suffix, action.body)
			})
		}

		t.Run(string(route.kind)+" media", func(t *testing.T) {
			testutil.RunAuthFailureSuite(t, newCommentHarness, "POST", route.path+"/"+uuid.NewString()+"/media", nil)
		})

		if route.createPath == "" {
			continue
		}

		t.Run(string(route.kind)+" create", func(t *testing.T) {
			testutil.RunAuthFailureSuite(t, newCommentHarness, "POST", strings.Replace(route.createPath, ":id", uuid.NewString(), 1), dto.CreateCommentRequest{Body: "hi"})
		})
	}
}

func TestCommentRoutes_AMalformedIDIsRefusedBeforeTheService(t *testing.T) {
	actions := slices.Concat(commentActions, []commentAction{
		{name: "media", method: "POST", suffix: "/media"},
	})

	for _, action := range actions {
		t.Run(action.name, func(t *testing.T) {
			// given a strict service mock that expects nothing
			h, _ := newCommentHarness(t)
			h.ExpectValidSession("valid-cookie", uuid.New())

			// when
			status, body := commentRequest(h, action, "/comments/not-a-uuid").Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), "invalid id")
		})
	}

	t.Run("create", func(t *testing.T) {
		// given a strict service mock that expects nothing
		h, _ := newCommentHarness(t)
		h.ExpectValidSession("valid-cookie", uuid.New())

		// when
		status, body := h.NewRequest("POST", "/ships/not-a-uuid/comments").
			WithCookie("valid-cookie").
			WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
			Do()

		// then
		require.Equal(t, http.StatusBadRequest, status)
		assert.Contains(t, string(body), "invalid id")
	})
}

func TestCommentRoutes_ServiceErrorsMapToResponses(t *testing.T) {
	for _, action := range commentActions {
		cases := slices.Concat(commentErrorCases, []commentErrorCase{{
			name:     "a server failure is not echoed back",
			err:      errors.New("pq: connection refused"),
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"` + action.failure + `"}`,
		}})

		for _, tc := range cases {
			t.Run(action.name+" "+tc.name, func(t *testing.T) {
				// given
				h, deps := newCommentHarness(t)
				userID := uuid.New()
				commentID := uuid.New()
				h.ExpectValidSession("valid-cookie", userID)
				action.expect(deps.svc, spec.CommentRef{Kind: spec.CommentKindPost, CommentID: commentID}, userID, tc.err)

				// when
				status, body := commentRequest(h, action, "/comments/"+commentID.String()).Do()

				// then
				require.Equal(t, tc.wantCode, status)
				assert.Contains(t, string(body), tc.wantBody)
			})
		}
	}
}

func TestCreateComment_OwnedRoutesCreateOnTheirOwnKind(t *testing.T) {
	for _, route := range commentRoutes {
		if route.createPath == "" {
			continue
		}

		t.Run(string(route.kind), func(t *testing.T) {
			// given
			h, deps := newCommentHarness(t)
			userID := uuid.New()
			entityID := uuid.New()
			newID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.svc.EXPECT().Create(mock.Anything, spec.CommentTarget{Kind: route.kind, ID: entityID}, userID, dto.CreateCommentRequest{Body: "hello"}, []upload.FileUpload(nil)).Return(newID, nil).Once()

			// when
			status, body := h.NewRequest("POST", strings.Replace(route.createPath, ":id", entityID.String(), 1)).
				WithCookie("valid-cookie").
				WithJSONBody(dto.CreateCommentRequest{Body: "hello"}).
				Do()

			// then
			require.Equal(t, http.StatusCreated, status)
			got := testutil.UnmarshalJSON[map[string]string](t, body)
			assert.Equal(t, newID.String(), got["id"])
		})
	}
}

func TestCreateComment_MultipartCarriesThePayloadAndFiles(t *testing.T) {
	// given
	h, deps := newCommentHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	parentID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	payload, err := json.Marshal(dto.CreateCommentRequest{ParentID: &parentID})
	require.NoError(t, err)
	form, contentType := testutil.MediaForm(t, ctrlutils.MediaField, map[string]string{
		ctrlutils.PayloadField:        string(payload),
		ctrlutils.SpoilerIndexesField: "0",
	})

	deps.svc.EXPECT().
		Create(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindArt, ID: artID}, userID, dto.CreateCommentRequest{ParentID: &parentID}, mock.MatchedBy(func(files []upload.FileUpload) bool {
			return len(files) == 1 && files[0].Filename == "pic.png" && files[0].ContentType == "image/png" && files[0].IsSpoiler
		})).
		Return(newID, nil).
		Once()

	// when
	status, body := h.NewRequest("POST", "/art/"+artID.String()+"/comments").
		WithCookie("valid-cookie").
		WithRawBody(form, contentType).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), got["id"])
}

func TestCommentRoutes_MalformedBodiesAreRefusedBeforeTheService(t *testing.T) {
	cases := []struct {
		name        string
		method      string
		path        string
		body        string
		contentType string
		wantBody    string
	}{
		{name: "an edit that is not json", method: "PUT", path: "/art-comments/" + uuid.NewString(), body: "not json", contentType: "application/json", wantBody: "invalid request body"},
		{name: "a create that is not json", method: "POST", path: "/art/" + uuid.NewString() + "/comments", body: "not json", contentType: "application/json", wantBody: "invalid request body"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a strict service mock that expects nothing
			h, _ := newCommentHarness(t)
			h.ExpectValidSession("valid-cookie", uuid.New())

			// when
			status, body := h.NewRequest(tc.method, tc.path).
				WithCookie("valid-cookie").
				WithRawBody(tc.body, tc.contentType).
				Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestCreateComment_MalformedMultipartIsRefusedBeforeTheService(t *testing.T) {
	cases := []struct {
		name     string
		values   map[string]string
		wantBody string
	}{
		{name: "a malformed payload", values: map[string]string{ctrlutils.PayloadField: "not json"}, wantBody: "invalid request body"},
		{name: "malformed spoiler indexes", values: map[string]string{ctrlutils.PayloadField: `{"body":"hi"}`, ctrlutils.SpoilerIndexesField: "first"}, wantBody: "invalid spoiler_indexes"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a strict service mock that expects nothing
			h, _ := newCommentHarness(t)
			h.ExpectValidSession("valid-cookie", uuid.New())
			form, contentType := testutil.MediaForm(t, ctrlutils.MediaField, tc.values)

			// when
			status, body := h.NewRequest("POST", "/announcements/"+uuid.NewString()+"/comments").
				WithCookie("valid-cookie").
				WithRawBody(form, contentType).
				Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestCreateComment_NewAccountsCannotAttachFiles(t *testing.T) {
	// given a strict service mock that expects nothing
	h, _ := newCommentHarness(t)
	expectNewAccountSession(h, "new-cookie", uuid.New())
	form, contentType := testutil.MediaForm(t, ctrlutils.MediaField, map[string]string{ctrlutils.PayloadField: `{"body":"hi"}`})

	// when
	status, body := h.NewRequest("POST", "/ships/"+uuid.NewString()+"/comments").
		WithCookie("new-cookie").
		WithRawBody(form, contentType).
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "new accounts cannot post attachments")
}

func TestCreateComment_ServiceErrorsMapToResponses(t *testing.T) {
	cases := slices.Concat(commentErrorCases, []commentErrorCase{{
		name:     "a server failure is not echoed back",
		err:      errors.New("pq: connection refused"),
		wantCode: http.StatusInternalServerError,
		wantBody: `{"error":"failed to create comment"}`,
	}})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newCommentHarness(t)
			userID := uuid.New()
			ocID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.svc.EXPECT().Create(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindOC, ID: ocID}, userID, dto.CreateCommentRequest{Body: "hi"}, []upload.FileUpload(nil)).Return(uuid.Nil, tc.err).Once()

			// when
			status, body := h.NewRequest("POST", "/ocs/"+ocID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUploadCommentMedia_EveryKindReachesTheServiceWithItsOwnKind(t *testing.T) {
	for _, route := range commentRoutes {
		t.Run(string(route.kind), func(t *testing.T) {
			// given
			h, deps := newCommentHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			form, contentType := testutil.MediaForm(t, "media", nil)
			deps.svc.EXPECT().
				UploadMedia(mock.Anything, spec.CommentRef{Kind: route.kind, CommentID: commentID}, userID, "image/png", "pic.png", mock.AnythingOfType("int64"), mock.Anything, false).
				Return(&dto.PostMediaResponse{ID: 7, MediaURL: "/uploads/pic.webp", MediaType: "image", SortOrder: 2}, nil).
				Once()

			// when
			status, body := h.NewRequest("POST", route.path+"/"+commentID.String()+"/media").
				WithCookie("valid-cookie").
				WithRawBody(form, contentType).
				Do()

			// then
			require.Equal(t, http.StatusCreated, status)
			got := testutil.UnmarshalJSON[dto.PostMediaResponse](t, body)
			assert.Equal(t, dto.PostMediaResponse{ID: 7, MediaURL: "/uploads/pic.webp", MediaType: "image", SortOrder: 2}, got)
		})
	}
}

func TestUploadCommentMedia_TheSpoilerFlagReachesTheService(t *testing.T) {
	// given
	h, deps := newCommentHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	form, contentType := testutil.MediaForm(t, "media", map[string]string{"is_spoiler": "true"})
	deps.svc.EXPECT().
		UploadMedia(mock.Anything, spec.CommentRef{Kind: spec.CommentKindMystery, CommentID: commentID}, userID, "image/png", "pic.png", mock.AnythingOfType("int64"), mock.Anything, true).
		Return(&dto.PostMediaResponse{ID: 1, IsSpoiler: true}, nil).
		Once()

	// when
	status, _ := h.NewRequest("POST", "/mystery-comments/"+commentID.String()+"/media").
		WithCookie("valid-cookie").
		WithRawBody(form, contentType).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestUploadCommentMedia_AMissingFileIsABadRequest(t *testing.T) {
	// given a strict service mock that expects nothing
	h, _ := newCommentHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/fanfic-comments/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").
		WithRawBody("", "multipart/form-data; boundary=----xxx").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}

func TestUploadCommentMedia_NewAccountsCannotUpload(t *testing.T) {
	// given a strict service mock that expects nothing
	h, _ := newCommentHarness(t)
	expectNewAccountSession(h, "new-cookie", uuid.New())
	form, contentType := testutil.MediaForm(t, "media", nil)

	// when
	status, body := h.NewRequest("POST", "/journal-comments/"+uuid.NewString()+"/media").
		WithCookie("new-cookie").
		WithRawBody(form, contentType).
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "new accounts cannot post attachments")
}

func TestUploadCommentMedia_ServiceErrorsMapToResponses(t *testing.T) {
	cases := slices.Concat(commentErrorCases, []commentErrorCase{{
		name:     "a server failure is not echoed back",
		err:      errors.New("pq: connection refused"),
		wantCode: http.StatusInternalServerError,
		wantBody: `{"error":"failed to upload media"}`,
	}})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newCommentHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			form, contentType := testutil.MediaForm(t, "media", nil)
			deps.svc.EXPECT().
				UploadMedia(mock.Anything, spec.CommentRef{Kind: spec.CommentKindSecret, CommentID: commentID}, userID, "image/png", "pic.png", mock.AnythingOfType("int64"), mock.Anything, false).
				Return(nil, tc.err).
				Once()

			// when
			status, body := h.NewRequest("POST", "/secret-comments/"+commentID.String()+"/media").
				WithCookie("valid-cookie").
				WithRawBody(form, contentType).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}
