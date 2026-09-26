package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	fanficsvc "umineko_city_of_books/internal/fanfic"
	fanficparams "umineko_city_of_books/internal/fanfic/params"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	fanficCtlOutcome struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}

	fanficCtlWrite struct {
		name     string
		method   string
		route    string
		body     any
		expect   func(fs *fanficsvc.MockService, userID, targetID, newID uuid.UUID, err error)
		outcomes []fanficCtlOutcome
	}
)

func newFanficHarness(t *testing.T) (*testutil.Harness, *fanficsvc.MockService) {
	h := testutil.NewHarness(t)
	fs := fanficsvc.NewMockService(t)

	s := &Service{
		FanficService: fs,
		AuthSession:   h.SessionManager,
		AuthzService:  h.AuthzService,
	}

	for _, setup := range s.getAllFanficRoutes() {
		setup(h.App)
	}

	return h, fs
}

func fanficCtlResult[T any](value T, err error) T {
	if err != nil {
		var zero T
		return zero
	}

	return value
}

func fanficCtlGet(h *testutil.Harness, path string, authed bool) (*testutil.Request, uuid.UUID) {
	req := h.NewRequest("GET", path)
	if !authed {
		return req, uuid.Nil
	}

	viewerID := uuid.New()
	h.ExpectValidSession("valid-cookie", viewerID)

	return req.WithCookie("valid-cookie"), viewerID
}

func fanficCtlWriteRequest(h *testutil.Harness, ep fanficCtlWrite, id string) *testutil.Request {
	req := h.NewRequest(ep.method, strings.Replace(ep.route, ":id", id, 1)).WithCookie("valid-cookie")
	if ep.body != nil {
		req = req.WithJSONBody(ep.body)
	}

	return req
}

func fanficCtlWriteEndpoints() []fanficCtlWrite {
	boom := errors.New("boom")
	createFanfic := dto.CreateFanficRequest{Title: "The Golden Witch", Summary: "a tale", Rating: "teen"}
	updateFanfic := dto.UpdateFanficRequest{Title: "Updated"}
	createChapter := dto.CreateChapterRequest{Title: "Ch1", Body: "body"}
	updateChapter := dto.UpdateChapterRequest{Title: "Updated", Body: "body"}
	createComment := dto.CreateCommentRequest{Body: "nice"}
	updateComment := dto.UpdateCommentRequest{Body: "edited"}

	created := fanficCtlOutcome{"created", nil, http.StatusCreated, ""}
	noContent := fanficCtlOutcome{"no content", nil, http.StatusNoContent, ""}
	notAuthor := fanficCtlOutcome{"not author", fanficsvc.ErrNotAuthor, http.StatusForbidden, fanficsvc.ErrNotAuthor.Error()}
	emptyBody := fanficCtlOutcome{"empty body", fanficsvc.ErrEmptyBody, http.StatusBadRequest, fanficsvc.ErrEmptyBody.Error()}
	blocked := fanficCtlOutcome{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"}
	fanficValidation := []fanficCtlOutcome{
		{"empty title", fanficsvc.ErrEmptyTitle, http.StatusBadRequest, fanficsvc.ErrEmptyTitle.Error()},
		{"too many genres", fanficsvc.ErrTooManyGenres, http.StatusBadRequest, fanficsvc.ErrTooManyGenres.Error()},
		{"too many tags", fanficsvc.ErrTooManyTags, http.StatusBadRequest, fanficsvc.ErrTooManyTags.Error()},
		{"tag too long", fanficsvc.ErrTagTooLong, http.StatusBadRequest, fanficsvc.ErrTagTooLong.Error()},
		{"invalid rating", fanficsvc.ErrInvalidRating, http.StatusBadRequest, fanficsvc.ErrInvalidRating.Error()},
	}

	return []fanficCtlWrite{
		{
			name: "create fanfic", method: "POST", route: "/fanfics", body: createFanfic,
			expect: func(fs *fanficsvc.MockService, userID, _, newID uuid.UUID, err error) {
				fs.EXPECT().CreateFanfic(mock.Anything, userID, createFanfic).Return(newID, err)
			},
			outcomes: append([]fanficCtlOutcome{
				created,
				{"internal", boom, http.StatusInternalServerError, "failed to create fanfic"},
			}, fanficValidation...),
		},
		{
			name: "update fanfic", method: "PUT", route: "/fanfics/:id", body: updateFanfic,
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().UpdateFanfic(mock.Anything, targetID, userID, updateFanfic).Return(err)
			},
			outcomes: append([]fanficCtlOutcome{
				noContent,
				notAuthor,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "fanfic not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to update fanfic"},
			}, fanficValidation...),
		},
		{
			name: "delete fanfic", method: "DELETE", route: "/fanfics/:id",
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().DeleteFanfic(mock.Anything, targetID, userID).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				notAuthor,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "fanfic not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to delete fanfic"},
			},
		},
		{name: "upload cover", method: "POST", route: "/fanfics/:id/cover"},
		{
			name: "delete cover", method: "DELETE", route: "/fanfics/:id/cover",
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().RemoveCoverImage(mock.Anything, targetID, userID).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				notAuthor,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "fanfic not found"},
				{"a server failure is a 500, not a 400 carrying the database error", errors.New("pq: connection refused"), http.StatusInternalServerError, "failed to remove the cover"},
			},
		},
		{
			name: "create chapter", method: "POST", route: "/fanfics/:id/chapters", body: createChapter,
			expect: func(fs *fanficsvc.MockService, userID, targetID, newID uuid.UUID, err error) {
				fs.EXPECT().CreateChapter(mock.Anything, targetID, userID, createChapter).Return(newID, err)
			},
			outcomes: []fanficCtlOutcome{
				created,
				notAuthor,
				emptyBody,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "fanfic not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to create chapter"},
			},
		},
		{
			name: "update chapter", method: "PUT", route: "/fanfic-chapters/:id", body: updateChapter,
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().UpdateChapter(mock.Anything, targetID, userID, updateChapter).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				notAuthor,
				emptyBody,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "chapter not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to update chapter"},
			},
		},
		{
			name: "delete chapter", method: "DELETE", route: "/fanfic-chapters/:id",
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().DeleteChapter(mock.Anything, targetID, userID).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				notAuthor,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "chapter not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to delete chapter"},
			},
		},
		{
			name: "favourite", method: "POST", route: "/fanfics/:id/favourite",
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().Favourite(mock.Anything, userID, targetID).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				blocked,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "fanfic not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to favourite fanfic"},
			},
		},
		{
			name: "unfavourite", method: "DELETE", route: "/fanfics/:id/favourite",
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().Unfavourite(mock.Anything, userID, targetID).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				{"internal", boom, http.StatusInternalServerError, "failed to unfavourite fanfic"},
			},
		},
		{
			name: "create comment", method: "POST", route: "/fanfics/:id/comments", body: createComment,
			expect: func(fs *fanficsvc.MockService, userID, targetID, newID uuid.UUID, err error) {
				fs.EXPECT().CreateComment(mock.Anything, targetID, userID, createComment).Return(newID, err)
			},
			outcomes: []fanficCtlOutcome{
				created,
				blocked,
				emptyBody,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "fanfic not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to create comment"},
			},
		},
		{
			name: "update comment", method: "PUT", route: "/fanfic-comments/:id", body: updateComment,
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().UpdateComment(mock.Anything, targetID, userID, updateComment).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				emptyBody,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "comment not found"},
				{"not owned", errors.Join(errors.New("comment not found or not owned"), dao.ErrNotFound), http.StatusForbidden, "cannot update this comment"},
				{"internal", boom, http.StatusInternalServerError, "failed to update comment"},
			},
		},
		{
			name: "delete comment", method: "DELETE", route: "/fanfic-comments/:id",
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().DeleteComment(mock.Anything, targetID, userID).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				{"internal", boom, http.StatusInternalServerError, "failed to delete comment"},
			},
		},
		{
			name: "like comment", method: "POST", route: "/fanfic-comments/:id/like",
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().LikeComment(mock.Anything, userID, targetID).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				blocked,
				{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "comment not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to like comment"},
			},
		},
		{
			name: "unlike comment", method: "DELETE", route: "/fanfic-comments/:id/like",
			expect: func(fs *fanficsvc.MockService, userID, targetID, _ uuid.UUID, err error) {
				fs.EXPECT().UnlikeComment(mock.Anything, userID, targetID).Return(err)
			},
			outcomes: []fanficCtlOutcome{
				noContent,
				{"internal", boom, http.StatusInternalServerError, "failed to unlike comment"},
			},
		},
		{name: "upload comment media", method: "POST", route: "/fanfic-comments/:id/media"},
	}
}

func TestUploadFanficCover(t *testing.T) {
	cases := []fanficCtlOutcome{
		{"not found", fanficsvc.ErrNotFound, http.StatusNotFound, "fanfic not found"},
		{"not author", fanficsvc.ErrNotAuthor, http.StatusForbidden, fanficsvc.ErrNotAuthor.Error()},
		{"too large", fmt.Errorf("%w: file size 9MB exceeds maximum 5MB", upload.ErrFileTooLarge), http.StatusBadRequest, "exceeds maximum 5MB"},
		{"a server failure is a 500, not a 400 carrying the database error", errors.New("pq: connection refused"), http.StatusInternalServerError, "failed to upload the cover"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			fanficID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			form, contentType := testutil.MediaForm(t, "image", nil)
			fs.EXPECT().UploadCoverImage(mock.Anything, fanficID, userID, "image/png", mock.AnythingOfType("int64"), mock.Anything).Return("", tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/fanfics/"+fanficID.String()+"/cover").WithCookie("valid-cookie").WithRawBody(form, contentType).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
			assert.NotContains(t, string(body), "pq:")
		})
	}
}

func TestFanficWrites_AuthFailures(t *testing.T) {
	for _, ep := range fanficCtlWriteEndpoints() {
		t.Run(ep.name, func(t *testing.T) {
			testutil.RunAuthFailureSuite(t, newFanficHarness, ep.method, strings.Replace(ep.route, ":id", uuid.NewString(), 1), ep.body)
		})
	}
}

func TestFanficWrites_InvalidID(t *testing.T) {
	for _, ep := range fanficCtlWriteEndpoints() {
		if !strings.Contains(ep.route, ":id") {
			continue
		}

		t.Run(ep.name, func(t *testing.T) {
			// given
			h, _ := newFanficHarness(t)
			h.ExpectValidSession("valid-cookie", uuid.New())

			// when
			status, body := fanficCtlWriteRequest(h, ep, "not-a-uuid").Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), "invalid id")
		})
	}
}

func TestFanficWrites_BadJSON_BadRequest(t *testing.T) {
	for _, ep := range fanficCtlWriteEndpoints() {
		if ep.body == nil {
			continue
		}

		t.Run(ep.name, func(t *testing.T) {
			// given
			h, _ := newFanficHarness(t)
			h.ExpectValidSession("valid-cookie", uuid.New())

			// when
			status, body := h.NewRequest(ep.method, strings.Replace(ep.route, ":id", uuid.NewString(), 1)).
				WithCookie("valid-cookie").
				WithRawBody("not json", "application/json").
				Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), "invalid request body")
		})
	}
}

func TestFanficWrites_ServiceOutcomes(t *testing.T) {
	for _, ep := range fanficCtlWriteEndpoints() {
		for _, oc := range ep.outcomes {
			t.Run(ep.name+"/"+oc.name, func(t *testing.T) {
				// given
				h, fs := newFanficHarness(t)
				userID := uuid.New()
				targetID := uuid.New()
				newID := fanficCtlResult(uuid.New(), oc.svcErr)
				h.ExpectValidSession("valid-cookie", userID)
				ep.expect(fs, userID, targetID, newID, oc.svcErr)

				// when
				status, body := fanficCtlWriteRequest(h, ep, targetID.String()).Do()

				// then
				require.Equal(t, oc.wantCode, status)
				assert.Contains(t, string(body), oc.wantBody)
				if oc.wantCode == http.StatusCreated {
					resp := testutil.UnmarshalJSON[map[string]string](t, body)
					assert.Equal(t, newID.String(), resp["id"])
				}
			})
		}
	}
}

func TestFanficUploads_NoFile_BadRequest(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		wantBody string
	}{
		{"cover", "/fanfics/" + uuid.NewString() + "/cover", "no image file provided"},
		{"comment media", "/fanfic-comments/" + uuid.NewString() + "/media", "no media file provided"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newFanficHarness(t)
			h.ExpectValidSession("valid-cookie", uuid.New())

			// when
			status, body := h.NewRequest("POST", tc.path).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestFanficReads_InvalidPathParam_BadRequest(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		wantBody string
	}{
		{"get fanfic", "/fanfics/not-a-uuid", "invalid id"},
		{"get chapter", "/fanfics/not-a-uuid/chapters/1", "invalid id"},
		{"get chapter zero", "/fanfics/" + uuid.NewString() + "/chapters/0", "invalid chapter number"},
		{"list user fanfics", "/users/not-a-uuid/fanfics", "invalid id"},
		{"list user favourites", "/users/not-a-uuid/fanfic-favourites", "invalid id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newFanficHarness(t)

			// when
			status, body := h.NewRequest("GET", tc.path).Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListFanfics(t *testing.T) {
	defaults := fanficparams.ListParams{Sort: "updated", Limit: 25, Offset: 0}
	custom := fanficparams.ListParams{
		Sort:       "top",
		Series:     "umineko",
		Rating:     "teen",
		GenreA:     "romance",
		GenreB:     "mystery",
		Language:   "en",
		Status:     "complete",
		Tag:        "fluff",
		CharacterA: "beatrice",
		CharacterB: "battler",
		CharacterC: "rosa",
		CharacterD: "maria",
		IsPairing:  true,
		ShowLemons: true,
		Search:     "witch",
		Limit:      10,
		Offset:     5,
	}
	customQuery := "?sort=top&series=umineko&rating=teen&genre_a=romance&genre_b=mystery&language=en&status=complete&tag=fluff&char_a=beatrice&char_b=battler&char_c=rosa&char_d=maria&pairing=true&lemons=true&search=witch&limit=10&offset=5"
	cases := []struct {
		name     string
		query    string
		authed   bool
		params   fanficparams.ListParams
		svcErr   error
		wantCode int
		errBody  string
	}{
		{name: "anonymous uses the default params", params: defaults, wantCode: http.StatusOK},
		{name: "authenticated passes the viewer id", authed: true, params: defaults, wantCode: http.StatusOK},
		{name: "every query parameter is mapped", query: customQuery, params: custom, wantCode: http.StatusOK},
		{name: "internal error", params: defaults, svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, errBody: "failed to list fanfics"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			req, viewerID := fanficCtlGet(h, "/fanfics"+tc.query, tc.authed)
			fs.EXPECT().ListFanfics(mock.Anything, viewerID, tc.params).
				Return(fanficCtlResult(&dto.FanficListResponse{Total: 3}, tc.svcErr), tc.svcErr)

			// when
			status, body := req.Do()

			// then
			require.Equal(t, tc.wantCode, status)
			if tc.svcErr != nil {
				assert.Contains(t, string(body), tc.errBody)
			} else {
				got := testutil.UnmarshalJSON[dto.FanficListResponse](t, body)
				assert.Equal(t, 3, got.Total)
			}
		})
	}
}

func TestGetFanfic(t *testing.T) {
	fanficID := uuid.New()
	cases := []struct {
		name     string
		authed   bool
		svcErr   error
		wantCode int
		errBody  string
	}{
		{name: "anonymous", wantCode: http.StatusOK},
		{name: "authenticated passes the viewer id", authed: true, wantCode: http.StatusOK},
		{name: "not found", svcErr: fanficsvc.ErrNotFound, wantCode: http.StatusNotFound, errBody: "fanfic not found"},
		{name: "internal error", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, errBody: "failed to get fanfic"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			req, viewerID := fanficCtlGet(h, "/fanfics/"+fanficID.String(), tc.authed)
			detail := &dto.FanficDetailResponse{FanficResponse: dto.FanficResponse{ID: fanficID, Title: "The Golden Witch"}}
			fs.EXPECT().GetFanfic(mock.Anything, fanficID, viewerID, mock.AnythingOfType("string")).
				Return(fanficCtlResult(detail, tc.svcErr), tc.svcErr)

			// when
			status, body := req.Do()

			// then
			require.Equal(t, tc.wantCode, status)
			if tc.svcErr != nil {
				assert.Contains(t, string(body), tc.errBody)
			} else {
				got := testutil.UnmarshalJSON[dto.FanficDetailResponse](t, body)
				assert.Equal(t, fanficID, got.ID)
			}
		})
	}
}

func TestGetFanficChapter(t *testing.T) {
	cases := []struct {
		name     string
		number   int
		svcErr   error
		wantCode int
		errBody  string
	}{
		{name: "anonymous", number: 1, wantCode: http.StatusOK},
		{name: "not found", number: 2, svcErr: fanficsvc.ErrNotFound, wantCode: http.StatusNotFound, errBody: "chapter not found"},
		{name: "internal error", number: 1, svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, errBody: "failed to get chapter"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			fanficID := uuid.New()
			chapter := &dto.FanficChapterResponse{ID: uuid.New(), ChapterNum: tc.number, Title: "Prologue"}
			fs.EXPECT().GetChapter(mock.Anything, fanficID, tc.number, uuid.Nil).
				Return(fanficCtlResult(chapter, tc.svcErr), tc.svcErr)

			// when
			status, body := h.NewRequest("GET", fmt.Sprintf("/fanfics/%s/chapters/%d", fanficID, tc.number)).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			if tc.svcErr != nil {
				assert.Contains(t, string(body), tc.errBody)
			} else {
				got := testutil.UnmarshalJSON[dto.FanficChapterResponse](t, body)
				assert.Equal(t, tc.number, got.ChapterNum)
			}
		})
	}
}

func TestListUserFanficsAndFavourites(t *testing.T) {
	endpoints := []struct {
		name    string
		suffix  string
		errBody string
		expect  func(fs *fanficsvc.MockService, targetID, viewerID uuid.UUID, page bounds.Page, resp *dto.FanficListResponse, err error)
	}{
		{
			name: "fanfics", suffix: "/fanfics", errBody: "failed to list user fanfics",
			expect: func(fs *fanficsvc.MockService, targetID, viewerID uuid.UUID, page bounds.Page, resp *dto.FanficListResponse, err error) {
				fs.EXPECT().ListFanficsByUser(mock.Anything, targetID, viewerID, page).Return(resp, err)
			},
		},
		{
			name: "favourites", suffix: "/fanfic-favourites", errBody: "failed to list favourites",
			expect: func(fs *fanficsvc.MockService, targetID, viewerID uuid.UUID, page bounds.Page, resp *dto.FanficListResponse, err error) {
				fs.EXPECT().ListFavourites(mock.Anything, targetID, viewerID, page).Return(resp, err)
			},
		},
	}

	scenarios := []struct {
		name     string
		query    string
		authed   bool
		page     bounds.Page
		svcErr   error
		wantCode int
	}{
		{name: "anonymous uses the default page", page: bounds.NewPage(20, 0), wantCode: http.StatusOK},
		{name: "authenticated passes the viewer id", authed: true, page: bounds.NewPage(20, 0), wantCode: http.StatusOK},
		{name: "custom paging", query: "?limit=5&offset=10", page: bounds.NewPage(5, 10), wantCode: http.StatusOK},
		{name: "internal error", page: bounds.NewPage(20, 0), svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError},
	}

	for _, ep := range endpoints {
		for _, sc := range scenarios {
			t.Run(ep.name+"/"+sc.name, func(t *testing.T) {
				// given
				h, fs := newFanficHarness(t)
				targetID := uuid.New()
				req, viewerID := fanficCtlGet(h, "/users/"+targetID.String()+ep.suffix+sc.query, sc.authed)
				ep.expect(fs, targetID, viewerID, sc.page, fanficCtlResult(&dto.FanficListResponse{}, sc.svcErr), sc.svcErr)

				// when
				status, body := req.Do()

				// then
				require.Equal(t, sc.wantCode, status)
				if sc.svcErr != nil {
					assert.Contains(t, string(body), ep.errBody)
				}
			})
		}
	}
}

func TestFanficLookups(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		key     string
		result  []string
		errBody string
		expect  func(fs *fanficsvc.MockService, result []string, err error)
	}{
		{
			name: "languages", path: "/fanfic-languages", key: "languages", result: []string{"en", "ja"}, errBody: "failed to get languages",
			expect: func(fs *fanficsvc.MockService, result []string, err error) {
				fs.EXPECT().GetLanguages(mock.Anything).Return(result, err)
			},
		},
		{
			name: "series", path: "/fanfic-series", key: "series", result: []string{"umineko", "higurashi"}, errBody: "failed to get series",
			expect: func(fs *fanficsvc.MockService, result []string, err error) {
				fs.EXPECT().GetSeries(mock.Anything).Return(result, err)
			},
		},
		{
			name: "oc characters", path: "/fanfic-oc-characters?q=bea", key: "characters", result: []string{"beatrice"}, errBody: "failed to search characters",
			expect: func(fs *fanficsvc.MockService, result []string, err error) {
				fs.EXPECT().SearchOCCharacters(mock.Anything, "bea").Return(result, err)
			},
		},
		{
			name: "oc characters without a query", path: "/fanfic-oc-characters", key: "characters", result: []string{}, errBody: "failed to search characters",
			expect: func(fs *fanficsvc.MockService, result []string, err error) {
				fs.EXPECT().SearchOCCharacters(mock.Anything, "").Return(result, err)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/ok", func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			tc.expect(fs, tc.result, nil)

			// when
			status, body := h.NewRequest("GET", tc.path).Do()

			// then
			require.Equal(t, http.StatusOK, status)
			got := testutil.UnmarshalJSON[map[string][]string](t, body)
			assert.Equal(t, tc.result, got[tc.key])
		})

		t.Run(tc.name+"/internal error", func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			tc.expect(fs, nil, errors.New("boom"))

			// when
			status, body := h.NewRequest("GET", tc.path).Do()

			// then
			require.Equal(t, http.StatusInternalServerError, status)
			assert.Contains(t, string(body), tc.errBody)
		})
	}
}
