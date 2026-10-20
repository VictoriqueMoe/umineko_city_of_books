package controllers

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	mysterysvc "umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	mysteryCtlOutcome struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}

	mysteryCtlRoute struct {
		method   string
		path     string
		body     any
		editPerm *bool
		expect   func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error)
	}

	mysteryCtlPagedCase struct {
		name     string
		query    string
		page     bounds.Page
		svcErr   error
		wantCode int
		wantBody string
	}
)

func newMysteryHarness(t *testing.T) (*testutil.Harness, *mysterysvc.MockService) {
	h := testutil.NewHarness(t)
	ms := mysterysvc.NewMockService(t)

	s := &Service{
		MysteryService:  ms,
		SettingsService: h.SettingsService,
		AuthSession:     h.SessionManager,
		AuthService:     h.AuthService,
		AuthzService:    h.AuthzService,
	}
	for _, setup := range s.getAllMysteryRoutes() {
		setup(h.App)
	}

	return h, ms
}

func mysteryCtlSignIn(t *testing.T, editPerm *bool) (*testutil.Harness, *mysterysvc.MockService, uuid.UUID) {
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	if editPerm != nil {
		h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(*editPerm)
	}

	return h, ms, userID
}

func mysteryCtlRequest(h *testutil.Harness, method, path string, body any) *testutil.Request {
	req := h.NewRequest(method, path)

	switch b := body.(type) {
	case nil:
	case string:
		req = req.WithRawBody(b, "application/json")
	default:
		req = req.WithJSONBody(b)
	}

	return req
}

func mysteryCtlRunOutcomes(t *testing.T, route mysteryCtlRoute, outcomes []mysteryCtlOutcome) {
	t.Helper()

	for _, tc := range outcomes {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, userID := mysteryCtlSignIn(t, route.editPerm)
			route.expect(ms, userID, tc.svcErr)

			// when
			status, body := mysteryCtlRequest(h, route.method, route.path, route.body).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func mysteryCtlRunPaged(t *testing.T, path string, expect func(ms *mysterysvc.MockService, page bounds.Page, svcErr error), cases []mysteryCtlPagedCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			expect(ms, tc.page, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", path+tc.query).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestMysteryRoutes_AuthFailures(t *testing.T) {
	id := uuid.NewString()
	routes := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"create mystery", "POST", "/mysteries", dto.CreateMysteryRequest{Title: "A Mystery"}},
		{"update mystery", "PUT", "/mysteries/" + id, dto.CreateMysteryRequest{Title: "x"}},
		{"delete mystery", "DELETE", "/mysteries/" + id, nil},
		{"create attempt", "POST", "/mysteries/" + id + "/attempts", dto.CreateAttemptRequest{Body: "b"}},
		{"delete attempt", "DELETE", "/mystery-attempts/" + id, nil},
		{"vote attempt", "POST", "/mystery-attempts/" + id + "/vote", dto.VoteRequest{Value: 1}},
		{"mark solved", "POST", "/mysteries/" + id + "/solve", map[string]string{"attempt_id": uuid.NewString()}},
		{"add clue", "POST", "/mysteries/" + id + "/clues", dto.CreateClueRequest{Body: "b"}},
		{"update clue", "PUT", "/mysteries/" + id + "/clues/1", map[string]string{"body": "x"}},
		{"delete clue", "DELETE", "/mysteries/" + id + "/clues/1", nil},
		{"toggle pause", "POST", "/mysteries/" + id + "/pause", map[string]bool{"paused": true}},
		{"toggle gm away", "POST", "/mysteries/" + id + "/away", map[string]bool{"away": true}},
		{"upload attachment", "POST", "/mysteries/" + id + "/attachments", nil},
		{"delete attachment", "DELETE", "/mysteries/" + id + "/attachments/1", nil},
		{"upload media", "POST", "/mysteries/" + id + "/media", nil},
		{"delete media", "DELETE", "/mysteries/" + id + "/media/1", nil},
		{"create comment", "POST", "/mysteries/" + id + "/comments", dto.CreateCommentRequest{Body: "hi"}},
	}

	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			testutil.RunAuthFailureSuite(t, newMysteryHarness, route.method, route.path, route.body)
		})
	}
}

func TestMysteryRoutes_RejectedBeforeService(t *testing.T) {
	id := uuid.NewString()
	granted, denied := new(true), new(false)
	cases := []struct {
		name      string
		method    string
		path      string
		body      any
		anonymous bool
		editPerm  *bool
		wantCode  int
		wantBody  string
	}{
		{name: "get mystery with a malformed id", method: "GET", path: "/mysteries/not-a-uuid", anonymous: true, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "list user mysteries with a malformed id", method: "GET", path: "/users/not-a-uuid/mysteries", anonymous: true, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "update mystery with a malformed id", method: "PUT", path: "/mysteries/not-a-uuid", body: dto.CreateMysteryRequest{Title: "x"}, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "delete mystery with a malformed id", method: "DELETE", path: "/mysteries/not-a-uuid", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "create attempt with a malformed id", method: "POST", path: "/mysteries/not-a-uuid/attempts", body: dto.CreateAttemptRequest{Body: "b"}, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "delete attempt with a malformed id", method: "DELETE", path: "/mystery-attempts/not-a-uuid", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "vote attempt with a malformed id", method: "POST", path: "/mystery-attempts/not-a-uuid/vote", body: dto.VoteRequest{Value: 1}, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "mark solved with a malformed id", method: "POST", path: "/mysteries/not-a-uuid/solve", body: map[string]string{"attempt_id": id}, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "add clue with a malformed id", method: "POST", path: "/mysteries/not-a-uuid/clues", body: dto.CreateClueRequest{Body: "b"}, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "update clue with a malformed mystery id", method: "PUT", path: "/mysteries/not-a-uuid/clues/1", body: map[string]string{"body": "x"}, editPerm: granted, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "delete clue with a malformed mystery id", method: "DELETE", path: "/mysteries/not-a-uuid/clues/1", editPerm: granted, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "toggle pause with a malformed id", method: "POST", path: "/mysteries/not-a-uuid/pause", body: map[string]bool{"paused": true}, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "toggle gm away with a malformed id", method: "POST", path: "/mysteries/not-a-uuid/away", body: map[string]bool{"away": true}, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "upload attachment with a malformed id", method: "POST", path: "/mysteries/not-a-uuid/attachments", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "delete attachment with a malformed mystery id", method: "DELETE", path: "/mysteries/not-a-uuid/attachments/1", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "upload media with a malformed id", method: "POST", path: "/mysteries/not-a-uuid/media", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "delete media with a malformed mystery id", method: "DELETE", path: "/mysteries/not-a-uuid/media/1", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "create comment with a malformed id", method: "POST", path: "/mysteries/not-a-uuid/comments", body: dto.CreateCommentRequest{Body: "hi"}, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "delete attachment with a non-numeric attachment id", method: "DELETE", path: "/mysteries/" + id + "/attachments/not-an-int", wantCode: http.StatusBadRequest, wantBody: "invalid attachment id"},
		{name: "delete media with a non-numeric media id", method: "DELETE", path: "/mysteries/" + id + "/media/not-an-int", wantCode: http.StatusBadRequest, wantBody: "invalid media id"},
		{name: "delete clue with a non-numeric clue id", method: "DELETE", path: "/mysteries/" + id + "/clues/not-an-int", editPerm: granted, wantCode: http.StatusBadRequest, wantBody: "invalid clue id"},
		{name: "update clue with a non-numeric clue id", method: "PUT", path: "/mysteries/" + id + "/clues/not-an-int", body: map[string]string{"body": "x"}, editPerm: granted, wantCode: http.StatusBadRequest, wantBody: "invalid clue id"},
		{name: "create mystery with malformed json", method: "POST", path: "/mysteries", body: "not json", wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "update mystery with malformed json", method: "PUT", path: "/mysteries/" + id, body: "not json", wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "create attempt with malformed json", method: "POST", path: "/mysteries/" + id + "/attempts", body: "not json", wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "vote attempt with malformed json", method: "POST", path: "/mystery-attempts/" + id + "/vote", body: "not json", wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "mark solved with malformed json", method: "POST", path: "/mysteries/" + id + "/solve", body: "not json", wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "add clue with malformed json", method: "POST", path: "/mysteries/" + id + "/clues", body: "not json", wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "update clue with malformed json", method: "PUT", path: "/mysteries/" + id + "/clues/1", body: "not json", editPerm: granted, wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "toggle pause with malformed json", method: "POST", path: "/mysteries/" + id + "/pause", body: "not json", wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "toggle gm away with malformed json", method: "POST", path: "/mysteries/" + id + "/away", body: "not json", wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "create comment with malformed json", method: "POST", path: "/mysteries/" + id + "/comments", body: "not json", wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "mark solved without an attempt id", method: "POST", path: "/mysteries/" + id + "/solve", body: map[string]string{}, wantCode: http.StatusBadRequest, wantBody: "attempt_id is required"},
		{name: "upload attachment without a file (multipart happy path deliberately not covered)", method: "POST", path: "/mysteries/" + id + "/attachments", wantCode: http.StatusBadRequest, wantBody: "no file provided"},
		{name: "upload media without a file (multipart happy path deliberately not covered)", method: "POST", path: "/mysteries/" + id + "/media", wantCode: http.StatusBadRequest, wantBody: "no media file provided"},
		{name: "update clue without the edit-any-theory permission", method: "PUT", path: "/mysteries/" + id + "/clues/1", body: map[string]string{"body": "x"}, editPerm: denied, wantCode: http.StatusForbidden, wantBody: "insufficient permissions"},
		{name: "delete clue without the edit-any-theory permission", method: "DELETE", path: "/mysteries/" + id + "/clues/1", editPerm: denied, wantCode: http.StatusForbidden, wantBody: "insufficient permissions"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _, _ := mysteryCtlSignIn(t, tc.editPerm)
			req := mysteryCtlRequest(h, tc.method, tc.path, tc.body)
			if !tc.anonymous {
				req = req.WithCookie("valid-cookie")
			}

			// when
			status, body := req.Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetMystery(t *testing.T) {
	mysteryID := uuid.New()
	viewerID := uuid.New()
	found := `{"id":"` + mysteryID.String() + `","title":"Legend of the Gold"`
	cases := []struct {
		name     string
		cookie   string
		viewer   uuid.UUID
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"anonymous viewer", "", uuid.Nil, nil, http.StatusOK, found},
		{"signed-in viewer is passed to the service", "valid-cookie", viewerID, nil, http.StatusOK, found},
		{"optional auth falls through to uuid.Nil when the cookie is junk", "bogus", uuid.Nil, nil, http.StatusOK, found},
		{"not found", "", uuid.Nil, mysterysvc.ErrNotFound, http.StatusNotFound, "mystery not found"},
		{"internal error", "", uuid.Nil, errors.New("boom"), http.StatusInternalServerError, "failed to get mystery"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			h.ExpectValidSession("valid-cookie", viewerID)
			h.ExpectInvalidSession("bogus")
			ms.EXPECT().GetMystery(mock.Anything, mysteryID, tc.viewer).Return(&dto.MysteryDetailResponse{ID: mysteryID, Title: "Legend of the Gold"}, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/mysteries/"+mysteryID.String()).WithCookie(tc.cookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListMysteries(t *testing.T) {
	viewerID := uuid.New()
	cases := []struct {
		name     string
		query    string
		cookie   string
		viewer   uuid.UUID
		sort     string
		solved   any
		page     bounds.Page
		svcErr   error
		wantCode int
		wantBody string
	}{
		{name: "anonymous request uses the defaults", sort: "new", solved: (*bool)(nil), page: bounds.NewPage(20, 0), wantCode: http.StatusOK, wantBody: `"total":3`},
		{
			name:  "solved=true filters to solved mysteries",
			query: "?solved=true",
			sort:  "new",
			solved: mock.MatchedBy(func(p *bool) bool {
				return p != nil && *p
			}),
			page:     bounds.NewPage(20, 0),
			wantCode: http.StatusOK,
			wantBody: `"total":3`,
		},
		{
			name:  "sort, solved=false and paging are passed through",
			query: "?sort=top&solved=false&limit=50&offset=10",
			sort:  "top",
			solved: mock.MatchedBy(func(p *bool) bool {
				return p != nil && !*p
			}),
			page:     bounds.NewPage(50, 10),
			wantCode: http.StatusOK,
			wantBody: `"total":3`,
		},
		{name: "signed-in viewer is passed to the service", cookie: "valid-cookie", viewer: viewerID, sort: "new", solved: (*bool)(nil), page: bounds.NewPage(20, 0), wantCode: http.StatusOK, wantBody: `"total":3`},
		{name: "internal error", sort: "new", solved: (*bool)(nil), page: bounds.NewPage(20, 0), svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to list mysteries"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			h.ExpectValidSession("valid-cookie", viewerID)
			ms.EXPECT().ListMysteries(mock.Anything, tc.sort, tc.solved, tc.viewer, tc.page).Return(&dto.MysteryListResponse{Total: 3}, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/mysteries"+tc.query).WithCookie(tc.cookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestMysteryLeaderboard(t *testing.T) {
	expect := func(ms *mysterysvc.MockService, page bounds.Page, svcErr error) {
		ms.EXPECT().GetLeaderboard(mock.Anything, page).Return(&dto.MysteryLeaderboardResponse{}, svcErr)
	}

	mysteryCtlRunPaged(t, "/mysteries/leaderboard", expect, []mysteryCtlPagedCase{
		{"defaults to 20 entries", "", bounds.NewPage(20, 0), nil, http.StatusOK, `"entries"`},
		{"honours a custom limit", "?limit=5", bounds.NewPage(5, 0), nil, http.StatusOK, `"entries"`},
		{"internal error", "", bounds.NewPage(20, 0), errors.New("boom"), http.StatusInternalServerError, "failed to load leaderboard"},
	})
}

func TestGMLeaderboard(t *testing.T) {
	expect := func(ms *mysterysvc.MockService, page bounds.Page, svcErr error) {
		ms.EXPECT().GetGMLeaderboard(mock.Anything, page).Return(&dto.GMLeaderboardResponse{}, svcErr)
	}

	mysteryCtlRunPaged(t, "/mysteries/gm-leaderboard", expect, []mysteryCtlPagedCase{
		{"defaults to 20 entries", "", bounds.NewPage(20, 0), nil, http.StatusOK, `"entries"`},
		{"internal error", "", bounds.NewPage(20, 0), errors.New("boom"), http.StatusInternalServerError, "failed to load gm leaderboard"},
	})
}

func TestListUserMysteries(t *testing.T) {
	targetUserID := uuid.New()
	expect := func(ms *mysterysvc.MockService, page bounds.Page, svcErr error) {
		ms.EXPECT().ListByUser(mock.Anything, targetUserID, page).Return(&dto.MysteryListResponse{Total: 3}, svcErr)
	}

	mysteryCtlRunPaged(t, "/users/"+targetUserID.String()+"/mysteries", expect, []mysteryCtlPagedCase{
		{"defaults to the first page of 20", "", bounds.NewPage(20, 0), nil, http.StatusOK, `"total":3`},
		{"honours custom paging", "?limit=5&offset=10", bounds.NewPage(5, 10), nil, http.StatusOK, `"total":3`},
		{"internal error", "", bounds.NewPage(20, 0), errors.New("boom"), http.StatusInternalServerError, "failed to list user mysteries"},
	})
}

func TestCreateMystery(t *testing.T) {
	mysteryID := uuid.New()
	cases := []struct {
		name     string
		req      dto.CreateMysteryRequest
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"created", dto.CreateMysteryRequest{Title: "A Mystery", Body: "body", Difficulty: "medium"}, nil, http.StatusCreated, `{"id":"` + mysteryID.String() + `"}`},
		{"empty title", dto.CreateMysteryRequest{Title: ""}, mysterysvc.ErrEmptyTitle, http.StatusBadRequest, mysterysvc.ErrEmptyTitle.Error()},
		{"two attachments sharing a name", dto.CreateMysteryRequest{Title: "A Mystery"}, fmt.Errorf("%w: %q", mysterysvc.ErrDuplicateAttachment, "notes.pdf"), http.StatusBadRequest, "already attached"},
		{"a refused file", dto.CreateMysteryRequest{Title: "A Mystery"}, upload.ErrInvalidAttachmentType, http.StatusBadRequest, upload.ErrInvalidAttachmentType.Error()},
		{"a server failure", dto.CreateMysteryRequest{Title: "A Mystery"}, errors.New("pq: connection refused"), http.StatusInternalServerError, "failed to create mystery"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given an old client posting plain json with no files
			h, ms, userID := mysteryCtlSignIn(t, nil)
			ms.EXPECT().CreateMystery(mock.Anything, userID, tc.req, []upload.FileUpload(nil), []upload.FileUpload(nil)).Return(mysteryID, tc.svcErr)

			// when
			status, body := mysteryCtlRequest(h, "POST", "/mysteries", tc.req).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
			assert.NotContains(t, string(body), "pq:")
		})
	}
}

func mysteryCtlCreateForm(t *testing.T, payload string) (string, string) {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("payload", payload))
	require.NoError(t, w.WriteField("spoiler_indexes", "1"))

	parts := []struct {
		field       string
		filename    string
		contentType string
	}{
		{field: "media", filename: "one.png", contentType: "image/png"},
		{field: "media", filename: "two.png", contentType: "image/png"},
		{field: "attachments", filename: "notes.pdf", contentType: "application/pdf"},
	}
	for _, p := range parts {
		part, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": {`form-data; name="` + p.field + `"; filename="` + p.filename + `"`},
			"Content-Type":        {p.contentType},
		})
		require.NoError(t, err)

		_, err = part.Write([]byte("content of " + p.filename))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())

	return buf.String(), w.FormDataContentType()
}

func TestCreateMystery_MultipartCarriesThePayloadMediaAndAttachments(t *testing.T) {
	// given a form carrying the json payload, two images with the second spoilered, and one attachment
	h, ms, userID := mysteryCtlSignIn(t, nil)
	mysteryID := uuid.New()
	form, contentType := mysteryCtlCreateForm(t, `{"title":"A Mystery","body":"body","difficulty":"hard"}`)

	twoImagesSecondSpoilered := mock.MatchedBy(func(files []upload.FileUpload) bool {
		return len(files) == 2 &&
			files[0].Filename == "one.png" && !files[0].IsSpoiler &&
			files[1].Filename == "two.png" && files[1].IsSpoiler
	})
	oneAttachment := mock.MatchedBy(func(files []upload.FileUpload) bool {
		return len(files) == 1 && files[0].Filename == "notes.pdf" && files[0].ContentType == "application/pdf" && !files[0].IsSpoiler
	})
	ms.EXPECT().
		CreateMystery(mock.Anything, userID, dto.CreateMysteryRequest{Title: "A Mystery", Body: "body", Difficulty: "hard"}, twoImagesSecondSpoilered, oneAttachment).
		Return(mysteryID, nil)

	// when
	status, body := h.NewRequest("POST", "/mysteries").WithCookie("valid-cookie").WithRawBody(form, contentType).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	assert.Contains(t, string(body), `{"id":"`+mysteryID.String()+`"}`)
}

func TestMysteryCreateRoutes_NewAccountsCannotAttachFiles(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		field string
	}{
		{name: "media on a mystery", path: "/mysteries", field: "media"},
		{name: "an attachment on a mystery", path: "/mysteries", field: "attachments"},
		{name: "media on a comment", path: "/mysteries/" + uuid.NewString() + "/comments", field: "media"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newMysteryHarness(t)
			userID := uuid.New()
			h.AuthService.EXPECT().
				Authenticate(mock.Anything, "new-cookie").
				Return(model.SessionAuthState{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil).
				Maybe()
			h.SessionRepo.EXPECT().GetUserID(mock.Anything, "new-cookie").Return(userID, time.Now().Add(time.Hour), nil).Maybe()
			h.AuthzService.EXPECT().IsBanned(mock.Anything, userID).Return(false).Maybe()
			h.AuthzService.EXPECT().IsRestrictedNewAccount(mock.Anything, userID).Return(true)
			form, contentType := testutil.MediaForm(t, tc.field, map[string]string{"payload": `{"title":"A Mystery","body":"body"}`})

			// when
			status, body := h.NewRequest("POST", tc.path).
				WithCookie("new-cookie").
				WithRawBody(form, contentType).
				Do()

			// then
			require.Equal(t, http.StatusForbidden, status)
			assert.Contains(t, string(body), "new accounts cannot post attachments")
		})
	}
}

func TestUpdateMystery(t *testing.T) {
	mysteryID := uuid.New()
	req := dto.CreateMysteryRequest{Title: "Updated", Body: "body"}
	route := mysteryCtlRoute{
		method: "PUT",
		path:   "/mysteries/" + mysteryID.String(),
		body:   req,
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().UpdateMystery(mock.Anything, mysteryID, userID, req).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"updated", nil, http.StatusOK, `{"status":"ok"}`},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, "mystery not found"},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to update mystery"},
	})
}

func TestDeleteMystery(t *testing.T) {
	mysteryID := uuid.New()
	route := mysteryCtlRoute{
		method: "DELETE",
		path:   "/mysteries/" + mysteryID.String(),
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().DeleteMystery(mock.Anything, mysteryID, userID).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"deleted", nil, http.StatusOK, `{"status":"ok"}`},
		{"not owned", errors.Join(errors.New("mystery not found or not owned"), daoerr.ErrNotFound), http.StatusForbidden, "cannot delete this mystery"},
		{"a server failure is a 500, not a 403", errors.New("boom"), http.StatusInternalServerError, "failed to delete mystery"},
	})
}

func TestCreateAttempt(t *testing.T) {
	mysteryID := uuid.New()
	attemptID := uuid.New()
	req := dto.CreateAttemptRequest{Body: "my guess"}
	route := mysteryCtlRoute{
		method: "POST",
		path:   "/mysteries/" + mysteryID.String() + "/attempts",
		body:   req,
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().CreateAttempt(mock.Anything, mysteryID, userID, req).Return(attemptID, svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"created", nil, http.StatusCreated, `{"id":"` + attemptID.String() + `"}`},
		{"empty body", mysterysvc.ErrEmptyBody, http.StatusBadRequest, mysterysvc.ErrEmptyBody.Error()},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, "mystery not found"},
		{"already solved", mysterysvc.ErrAlreadySolved, http.StatusForbidden, mysterysvc.ErrAlreadySolved.Error()},
		{"cannot reply", mysterysvc.ErrCannotReply, http.StatusForbidden, mysterysvc.ErrCannotReply.Error()},
		{"paused", mysterysvc.ErrMysteryPaused, http.StatusForbidden, mysterysvc.ErrMysteryPaused.Error()},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, block.ErrUserBlocked.Error()},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create attempt"},
	})
}

func TestDeleteAttempt(t *testing.T) {
	attemptID := uuid.New()
	route := mysteryCtlRoute{
		method: "DELETE",
		path:   "/mystery-attempts/" + attemptID.String(),
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().DeleteAttempt(mock.Anything, attemptID, userID).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"deleted", nil, http.StatusOK, `{"status":"ok"}`},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, "attempt not found"},
		{"not owned", errors.Join(errors.New("attempt not found or not owned"), daoerr.ErrNotFound), http.StatusForbidden, "cannot delete this attempt"},
		{"a server failure is a 500, not a 403", errors.New("boom"), http.StatusInternalServerError, "failed to delete attempt"},
	})
}

func TestVoteAttempt(t *testing.T) {
	attemptID := uuid.New()
	cases := []struct {
		name     string
		value    int
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"voted", 1, nil, http.StatusOK, `{"status":"ok"}`},
		{"not found", 1, mysterysvc.ErrNotFound, http.StatusNotFound, "attempt not found"},
		{"invalid vote", 42, mysterysvc.ErrInvalidVote, http.StatusBadRequest, mysterysvc.ErrInvalidVote.Error()},
		{"blocked", 1, block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal", 1, errors.New("boom"), http.StatusInternalServerError, "failed to vote"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, userID := mysteryCtlSignIn(t, nil)
			ms.EXPECT().VoteAttempt(mock.Anything, attemptID, userID, tc.value).Return(tc.svcErr)

			// when
			status, body := mysteryCtlRequest(h, "POST", "/mystery-attempts/"+attemptID.String()+"/vote", dto.VoteRequest{Value: tc.value}).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestMarkSolved(t *testing.T) {
	mysteryID := uuid.New()
	attemptID := uuid.New()
	route := mysteryCtlRoute{
		method: "POST",
		path:   "/mysteries/" + mysteryID.String() + "/solve",
		body:   map[string]string{"attempt_id": attemptID.String()},
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().MarkSolved(mock.Anything, mysteryID, userID, attemptID).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"solved", nil, http.StatusOK, `{"status":"ok"}`},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, mysterysvc.ErrNotFound.Error()},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
		{"an attempt from another mystery", mysterysvc.ErrAttemptNotOnMystery, http.StatusBadRequest, mysterysvc.ErrAttemptNotOnMystery.Error()},
		{"the game master's own attempt", mysterysvc.ErrOwnAttempt, http.StatusBadRequest, mysterysvc.ErrOwnAttempt.Error()},
		{"a player who already won", mysterysvc.ErrAlreadyWon, http.StatusBadRequest, mysterysvc.ErrAlreadyWon.Error()},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to mark as solved"},
	})
}

func TestAddClue(t *testing.T) {
	mysteryID := uuid.New()
	req := dto.CreateClueRequest{Body: "clue", TruthType: "red"}
	route := mysteryCtlRoute{
		method: "POST",
		path:   "/mysteries/" + mysteryID.String() + "/clues",
		body:   req,
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().AddClue(mock.Anything, mysteryID, userID, req).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"created", nil, http.StatusCreated, `{"status":"ok"}`},
		{"empty body", mysterysvc.ErrEmptyBody, http.StatusBadRequest, mysterysvc.ErrEmptyBody.Error()},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, "mystery not found"},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to add clue"},
	})
}

func TestUpdateMysteryClue(t *testing.T) {
	mysteryID := uuid.New()
	cases := []struct {
		name     string
		body     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"updated", "updated body", nil, http.StatusOK, `{"status":"ok"}`},
		{"not found", "x", mysterysvc.ErrNotFound, http.StatusNotFound, "clue not found"},
		{"not author", "x", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
		{"empty body", "", mysterysvc.ErrEmptyBody, http.StatusBadRequest, mysterysvc.ErrEmptyBody.Error()},
		{"internal", "x", errors.New("boom"), http.StatusInternalServerError, "failed to update clue"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, userID := mysteryCtlSignIn(t, new(true))
			ms.EXPECT().UpdateClue(mock.Anything, mysteryID, 5, userID, tc.body).Return(tc.svcErr)

			// when
			status, body := mysteryCtlRequest(h, "PUT", "/mysteries/"+mysteryID.String()+"/clues/5", map[string]string{"body": tc.body}).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeleteMysteryClue(t *testing.T) {
	mysteryID := uuid.New()
	route := mysteryCtlRoute{
		method:   "DELETE",
		path:     "/mysteries/" + mysteryID.String() + "/clues/7",
		editPerm: new(true),
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().DeleteClue(mock.Anything, mysteryID, 7, userID).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"deleted", nil, http.StatusNoContent, ""},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, "clue not found"},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to delete clue"},
	})
}

func TestToggleMysteryPause(t *testing.T) {
	mysteryID := uuid.New()
	route := mysteryCtlRoute{
		method: "POST",
		path:   "/mysteries/" + mysteryID.String() + "/pause",
		body:   map[string]bool{"paused": true},
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().SetPaused(mock.Anything, mysteryID, userID, true).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"paused", nil, http.StatusOK, `{"status":"ok"}`},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, mysterysvc.ErrNotFound.Error()},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to toggle pause"},
	})
}

func TestToggleMysteryGmAway(t *testing.T) {
	mysteryID := uuid.New()
	route := mysteryCtlRoute{
		method: "POST",
		path:   "/mysteries/" + mysteryID.String() + "/away",
		body:   map[string]bool{"away": true},
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().SetGmAway(mock.Anything, mysteryID, userID, true).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"away", nil, http.StatusOK, `{"status":"ok"}`},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, mysterysvc.ErrNotFound.Error()},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to toggle away"},
	})
}

func TestUploadMysteryFiles(t *testing.T) {
	mysteryID := uuid.New()
	shared := []mysteryCtlOutcome{
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, "mystery not found"},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
		{"too large", fmt.Errorf("%w: file size 9MB exceeds maximum 5MB", upload.ErrFileTooLarge), http.StatusBadRequest, "exceeds maximum 5MB"},
		{"a server failure is a 500, not a 400 carrying the database error", errors.New("pq: connection refused"), http.StatusInternalServerError, "failed to upload"},
	}
	routes := []struct {
		name     string
		path     string
		field    string
		expect   func(ms *mysterysvc.MockService, userID uuid.UUID, err error)
		outcomes []mysteryCtlOutcome
	}{
		{
			name:  "attachment",
			path:  "/mysteries/" + mysteryID.String() + "/attachments",
			field: "file",
			expect: func(ms *mysterysvc.MockService, userID uuid.UUID, err error) {
				ms.EXPECT().UploadAttachment(mock.Anything, mysteryID, userID, "pic.png", mock.AnythingOfType("int64"), mock.Anything).Return(nil, err)
			},
			outcomes: append([]mysteryCtlOutcome{
				{"a duplicate name", fmt.Errorf("%w: %q", mysterysvc.ErrDuplicateAttachment, "pic.png"), http.StatusBadRequest, "already attached"},
			}, shared...),
		},
		{
			name:  "media",
			path:  "/mysteries/" + mysteryID.String() + "/media",
			field: "media",
			expect: func(ms *mysterysvc.MockService, userID uuid.UUID, err error) {
				ms.EXPECT().UploadMedia(mock.Anything, mysteryID, userID, "image/png", "pic.png", mock.AnythingOfType("int64"), mock.Anything, false).Return(nil, err)
			},
			outcomes: shared,
		},
	}

	for _, route := range routes {
		for _, tc := range route.outcomes {
			t.Run(route.name+": "+tc.name, func(t *testing.T) {
				// given
				h, ms, userID := mysteryCtlSignIn(t, nil)
				form, contentType := testutil.MediaForm(t, route.field, nil)
				route.expect(ms, userID, tc.svcErr)

				// when
				status, body := h.NewRequest("POST", route.path).WithCookie("valid-cookie").WithRawBody(form, contentType).Do()

				// then
				require.Equal(t, tc.wantCode, status)
				assert.Contains(t, string(body), tc.wantBody)
				assert.NotContains(t, string(body), "pq:")
			})
		}
	}
}

func TestDeleteMysteryAttachment(t *testing.T) {
	mysteryID := uuid.New()
	route := mysteryCtlRoute{
		method: "DELETE",
		path:   "/mysteries/" + mysteryID.String() + "/attachments/42",
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().DeleteAttachment(mock.Anything, int64(42), mysteryID, userID).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"deleted", nil, http.StatusNoContent, ""},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, "mystery not found"},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to delete attachment"},
	})
}

func TestDeleteMysteryMedia(t *testing.T) {
	mysteryID := uuid.New()
	route := mysteryCtlRoute{
		method: "DELETE",
		path:   "/mysteries/" + mysteryID.String() + "/media/42",
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().DeleteMedia(mock.Anything, int64(42), mysteryID, userID).Return(svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"deleted", nil, http.StatusNoContent, ""},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, "mystery not found"},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden, mysterysvc.ErrNotAuthor.Error()},
	})
}

func TestCreateMysteryComment(t *testing.T) {
	mysteryID := uuid.New()
	commentID := uuid.New()
	req := dto.CreateCommentRequest{Body: "hello"}
	route := mysteryCtlRoute{
		method: "POST",
		path:   "/mysteries/" + mysteryID.String() + "/comments",
		body:   req,
		expect: func(ms *mysterysvc.MockService, userID uuid.UUID, svcErr error) {
			ms.EXPECT().CreateComment(mock.Anything, mysteryID, userID, req, []upload.FileUpload(nil)).Return(commentID, svcErr)
		},
	}

	mysteryCtlRunOutcomes(t, route, []mysteryCtlOutcome{
		{"created", nil, http.StatusCreated, `{"id":"` + commentID.String() + `"}`},
		{"empty body", comment.ErrEmptyBody, http.StatusBadRequest, comment.ErrEmptyBody.Error()},
		{"not solved", mysterysvc.ErrNotSolved, http.StatusForbidden, mysterysvc.ErrNotSolved.Error()},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound, "mystery not found"},
		{"a refused file", fmt.Errorf("%w: file size 9MB exceeds maximum 5MB", upload.ErrFileTooLarge), http.StatusBadRequest, "exceeds maximum 5MB"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create comment"},
	})
}

func TestCreateMysteryComment_MultipartCarriesABlankBodyAndItsMedia(t *testing.T) {
	// given a form carrying a blank comment and one image marked as a spoiler
	h, ms, userID := mysteryCtlSignIn(t, nil)
	mysteryID := uuid.New()
	commentID := uuid.New()
	form, contentType := testutil.MediaForm(t, "media", map[string]string{"payload": `{"body":""}`, "spoiler_indexes": "0"})
	oneSpoileredPic := mock.MatchedBy(func(files []upload.FileUpload) bool {
		return len(files) == 1 && files[0].Filename == "pic.png" && files[0].IsSpoiler
	})
	ms.EXPECT().CreateComment(mock.Anything, mysteryID, userID, dto.CreateCommentRequest{}, oneSpoileredPic).Return(commentID, nil)

	// when
	status, body := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/comments").WithCookie("valid-cookie").WithRawBody(form, contentType).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	assert.Contains(t, string(body), `{"id":"`+commentID.String()+`"}`)
}
