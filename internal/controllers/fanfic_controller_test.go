package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	fanficsvc "umineko_city_of_books/internal/fanfic"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func defaultFanficListParams() repository.FanficListParams {
	return repository.FanficListParams{
		Sort:   "updated",
		Limit:  25,
		Offset: 0,
	}
}

func TestListFanfics_Anonymous_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	expected := &dto.FanficListResponse{Total: 0, Limit: 25, Offset: 0}
	fs.EXPECT().ListFanfics(mock.Anything, uuid.Nil, defaultFanficListParams()).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/fanfics").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.FanficListResponse](t, body)
	assert.Equal(t, expected.Total, got.Total)
}

func TestListFanfics_CustomQuery_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	params := repository.FanficListParams{
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
	fs.EXPECT().ListFanfics(mock.Anything, uuid.Nil, params).Return(&dto.FanficListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/fanfics?sort=top&series=umineko&rating=teen&genre_a=romance&genre_b=mystery&language=en&status=complete&tag=fluff&char_a=beatrice&char_b=battler&char_c=rosa&char_d=maria&pairing=true&lemons=true&search=witch&limit=10&offset=5").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListFanfics_Authenticated_PassesViewerID(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().ListFanfics(mock.Anything, userID, defaultFanficListParams()).Return(&dto.FanficListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/fanfics").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListFanfics_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fs.EXPECT().ListFanfics(mock.Anything, uuid.Nil, defaultFanficListParams()).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/fanfics").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list fanfics")
}

func TestGetFanfic_Anonymous_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fanficID := uuid.New()
	expected := &dto.FanficDetailResponse{FanficResponse: dto.FanficResponse{ID: fanficID, Title: "The Golden Witch"}}
	fs.EXPECT().GetFanfic(mock.Anything, fanficID, uuid.Nil, mock.AnythingOfType("string")).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/fanfics/"+fanficID.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.FanficDetailResponse](t, body)
	assert.Equal(t, fanficID, got.ID)
}

func TestGetFanfic_Authenticated_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().GetFanfic(mock.Anything, fanficID, userID, mock.AnythingOfType("string")).
		Return(&dto.FanficDetailResponse{FanficResponse: dto.FanficResponse{ID: fanficID}}, nil)

	// when
	status, _ := h.NewRequest("GET", "/fanfics/"+fanficID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetFanfic_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)

	// when
	status, body := h.NewRequest("GET", "/fanfics/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetFanfic_NotFound(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fanficID := uuid.New()
	fs.EXPECT().GetFanfic(mock.Anything, fanficID, uuid.Nil, mock.AnythingOfType("string")).
		Return(nil, fanficsvc.ErrNotFound)

	// when
	status, _ := h.NewRequest("GET", "/fanfics/"+fanficID.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
}

func TestGetFanfic_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fanficID := uuid.New()
	fs.EXPECT().GetFanfic(mock.Anything, fanficID, uuid.Nil, mock.AnythingOfType("string")).
		Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/fanfics/"+fanficID.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestCreateFanfic_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "POST", "/fanfics", dto.CreateFanficRequest{Title: "x"})
}

func TestCreateFanfic_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateFanficRequest{Title: "The Golden Witch", Summary: "a tale", Rating: "teen"}
	fs.EXPECT().CreateFanfic(mock.Anything, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/fanfics").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateFanfic_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/fanfics").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateFanfic_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty title", fanficsvc.ErrEmptyTitle, http.StatusBadRequest},
		{"too many genres", fanficsvc.ErrTooManyGenres, http.StatusBadRequest},
		{"too many characters", fanficsvc.ErrTooManyCharacters, http.StatusBadRequest},
		{"too many tags", fanficsvc.ErrTooManyTags, http.StatusBadRequest},
		{"tag too long", fanficsvc.ErrTagTooLong, http.StatusBadRequest},
		{"invalid rating", fanficsvc.ErrInvalidRating, http.StatusBadRequest},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateFanficRequest{Title: "x"}
			fs.EXPECT().CreateFanfic(mock.Anything, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/fanfics").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUpdateFanfic_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "PUT", "/fanfics/"+uuid.NewString(), dto.UpdateFanficRequest{Title: "x"})
}

func TestUpdateFanfic_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/fanfics/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateFanficRequest{Title: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateFanfic_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/fanfics/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateFanfic_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateFanficRequest{Title: "Updated"}
	fs.EXPECT().UpdateFanfic(mock.Anything, fanficID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/fanfics/"+fanficID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateFanfic_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty title", fanficsvc.ErrEmptyTitle, http.StatusBadRequest},
		{"too many genres", fanficsvc.ErrTooManyGenres, http.StatusBadRequest},
		{"too many characters", fanficsvc.ErrTooManyCharacters, http.StatusBadRequest},
		{"too many tags", fanficsvc.ErrTooManyTags, http.StatusBadRequest},
		{"tag too long", fanficsvc.ErrTagTooLong, http.StatusBadRequest},
		{"invalid rating", fanficsvc.ErrInvalidRating, http.StatusBadRequest},
		{"not author", fanficsvc.ErrNotAuthor, http.StatusForbidden},
		{"not found", fanficsvc.ErrNotFound, http.StatusNotFound},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			fanficID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.UpdateFanficRequest{Title: "x"}
			fs.EXPECT().UpdateFanfic(mock.Anything, fanficID, userID, req).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("PUT", "/fanfics/"+fanficID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestDeleteFanfic_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "DELETE", "/fanfics/"+uuid.NewString(), nil)
}

func TestDeleteFanfic_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/fanfics/not-a-uuid").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteFanfic_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().DeleteFanfic(mock.Anything, fanficID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/fanfics/"+fanficID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteFanfic_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().DeleteFanfic(mock.Anything, fanficID, userID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/fanfics/"+fanficID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestUploadFanficCover_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "POST", "/fanfics/"+uuid.NewString()+"/cover", nil)
}

func TestUploadFanficCover_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/fanfics/not-a-uuid/cover").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadFanficCover_NoFile_BadRequest(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/fanfics/"+uuid.NewString()+"/cover").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no image file provided")
}

func TestDeleteFanficCover_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "DELETE", "/fanfics/"+uuid.NewString()+"/cover", nil)
}

func TestDeleteFanficCover_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/fanfics/not-a-uuid/cover").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteFanficCover_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().RemoveCoverImage(mock.Anything, fanficID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/fanfics/"+fanficID.String()+"/cover").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteFanficCover_ServiceError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().RemoveCoverImage(mock.Anything, fanficID, userID).Return(errors.New("not author"))

	// when
	status, body := h.NewRequest("DELETE", "/fanfics/"+fanficID.String()+"/cover").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "not author")
}

func TestGetFanficChapter_Anonymous_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fanficID := uuid.New()
	expected := &dto.FanficChapterResponse{ID: uuid.New(), ChapterNum: 1, Title: "Prologue"}
	fs.EXPECT().GetChapter(mock.Anything, fanficID, 1, uuid.Nil).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/fanfics/"+fanficID.String()+"/chapters/1").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.FanficChapterResponse](t, body)
	assert.Equal(t, 1, got.ChapterNum)
}

func TestGetFanficChapter_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)

	// when
	status, body := h.NewRequest("GET", "/fanfics/not-a-uuid/chapters/1").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetFanficChapter_InvalidChapterNumber(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	fanficID := uuid.New()

	// when
	status, body := h.NewRequest("GET", "/fanfics/"+fanficID.String()+"/chapters/0").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid chapter number")
}

func TestGetFanficChapter_NotFound(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fanficID := uuid.New()
	fs.EXPECT().GetChapter(mock.Anything, fanficID, 2, uuid.Nil).Return(nil, fanficsvc.ErrNotFound)

	// when
	status, _ := h.NewRequest("GET", "/fanfics/"+fanficID.String()+"/chapters/2").Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
}

func TestGetFanficChapter_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fanficID := uuid.New()
	fs.EXPECT().GetChapter(mock.Anything, fanficID, 1, uuid.Nil).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/fanfics/"+fanficID.String()+"/chapters/1").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestCreateFanficChapter_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "POST", "/fanfics/"+uuid.NewString()+"/chapters", dto.CreateChapterRequest{Body: "b"})
}

func TestCreateFanficChapter_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/fanfics/not-a-uuid/chapters").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateChapterRequest{Body: "b"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateFanficChapter_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/fanfics/"+uuid.NewString()+"/chapters").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateFanficChapter_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateChapterRequest{Title: "Ch1", Body: "body"}
	fs.EXPECT().CreateChapter(mock.Anything, fanficID, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/fanfics/"+fanficID.String()+"/chapters").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateFanficChapter_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"not author", fanficsvc.ErrNotAuthor, http.StatusForbidden},
		{"empty body", fanficsvc.ErrEmptyBody, http.StatusBadRequest},
		{"not found", fanficsvc.ErrNotFound, http.StatusNotFound},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			fanficID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateChapterRequest{Body: "x"}
			fs.EXPECT().CreateChapter(mock.Anything, fanficID, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/fanfics/"+fanficID.String()+"/chapters").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUpdateFanficChapter_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "PUT", "/fanfic-chapters/"+uuid.NewString(), dto.UpdateChapterRequest{Body: "b"})
}

func TestUpdateFanficChapter_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/fanfic-chapters/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateChapterRequest{Body: "b"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateFanficChapter_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/fanfic-chapters/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateFanficChapter_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	chapterID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateChapterRequest{Title: "Updated", Body: "body"}
	fs.EXPECT().UpdateChapter(mock.Anything, chapterID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/fanfic-chapters/"+chapterID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateFanficChapter_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"not author", fanficsvc.ErrNotAuthor, http.StatusForbidden},
		{"empty body", fanficsvc.ErrEmptyBody, http.StatusBadRequest},
		{"not found", fanficsvc.ErrNotFound, http.StatusNotFound},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			chapterID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.UpdateChapterRequest{Body: "x"}
			fs.EXPECT().UpdateChapter(mock.Anything, chapterID, userID, req).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("PUT", "/fanfic-chapters/"+chapterID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestDeleteFanficChapter_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "DELETE", "/fanfic-chapters/"+uuid.NewString(), nil)
}

func TestDeleteFanficChapter_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/fanfic-chapters/not-a-uuid").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteFanficChapter_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	chapterID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().DeleteChapter(mock.Anything, chapterID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/fanfic-chapters/"+chapterID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteFanficChapter_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"not author", fanficsvc.ErrNotAuthor, http.StatusForbidden},
		{"not found", fanficsvc.ErrNotFound, http.StatusNotFound},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			chapterID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			fs.EXPECT().DeleteChapter(mock.Anything, chapterID, userID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("DELETE", "/fanfic-chapters/"+chapterID.String()).
				WithCookie("valid-cookie").
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestFavouriteFanfic_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "POST", "/fanfics/"+uuid.NewString()+"/favourite", nil)
}

func TestFavouriteFanfic_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/fanfics/not-a-uuid/favourite").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestFavouriteFanfic_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().Favourite(mock.Anything, userID, fanficID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/fanfics/"+fanficID.String()+"/favourite").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestFavouriteFanfic_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			fanficID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			fs.EXPECT().Favourite(mock.Anything, userID, fanficID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/fanfics/"+fanficID.String()+"/favourite").
				WithCookie("valid-cookie").
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUnfavouriteFanfic_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "DELETE", "/fanfics/"+uuid.NewString()+"/favourite", nil)
}

func TestUnfavouriteFanfic_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/fanfics/not-a-uuid/favourite").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnfavouriteFanfic_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().Unfavourite(mock.Anything, userID, fanficID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/fanfics/"+fanficID.String()+"/favourite").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnfavouriteFanfic_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().Unfavourite(mock.Anything, userID, fanficID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/fanfics/"+fanficID.String()+"/favourite").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestCreateFanficComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "POST", "/fanfics/"+uuid.NewString()+"/comments", dto.CreateCommentRequest{Body: "b"})
}

func TestCreateFanficComment_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/fanfics/not-a-uuid/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "b"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateFanficComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/fanfics/"+uuid.NewString()+"/comments").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateFanficComment_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	fanficID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateCommentRequest{Body: "nice"}
	fs.EXPECT().CreateComment(mock.Anything, fanficID, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/fanfics/"+fanficID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateFanficComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden},
		{"empty body", fanficsvc.ErrEmptyBody, http.StatusBadRequest},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			fanficID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateCommentRequest{Body: "x"}
			fs.EXPECT().CreateComment(mock.Anything, fanficID, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/fanfics/"+fanficID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUpdateFanficComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "PUT", "/fanfic-comments/"+uuid.NewString(), dto.UpdateCommentRequest{Body: "b"})
}

func TestUpdateFanficComment_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/fanfic-comments/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "b"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateFanficComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/fanfic-comments/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateFanficComment_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateCommentRequest{Body: "edited"}
	fs.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/fanfic-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateFanficComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty body", fanficsvc.ErrEmptyBody, http.StatusBadRequest},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.UpdateCommentRequest{Body: "x"}
			fs.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("PUT", "/fanfic-comments/"+commentID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestDeleteFanficComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "DELETE", "/fanfic-comments/"+uuid.NewString(), nil)
}

func TestDeleteFanficComment_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/fanfic-comments/not-a-uuid").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteFanficComment_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/fanfic-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteFanficComment_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/fanfic-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestLikeFanficComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "POST", "/fanfic-comments/"+uuid.NewString()+"/like", nil)
}

func TestLikeFanficComment_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/fanfic-comments/not-a-uuid/like").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestLikeFanficComment_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/fanfic-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeFanficComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, fs := newFanficHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			fs.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/fanfic-comments/"+commentID.String()+"/like").
				WithCookie("valid-cookie").
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUnlikeFanficComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "DELETE", "/fanfic-comments/"+uuid.NewString()+"/like", nil)
}

func TestUnlikeFanficComment_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/fanfic-comments/not-a-uuid/like").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnlikeFanficComment_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/fanfic-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeFanficComment_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/fanfic-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestUploadFanficCommentMedia_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newFanficHarness, "POST", "/fanfic-comments/"+uuid.NewString()+"/media", nil)
}

func TestUploadFanficCommentMedia_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/fanfic-comments/not-a-uuid/media").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadFanficCommentMedia_NoFile_BadRequest(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/fanfic-comments/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}

func TestGetFanficLanguages_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fs.EXPECT().GetLanguages(mock.Anything).Return([]string{"en", "ja"}, nil)

	// when
	status, body := h.NewRequest("GET", "/fanfic-languages").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string][]string](t, body)
	assert.Equal(t, []string{"en", "ja"}, got["languages"])
}

func TestGetFanficLanguages_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fs.EXPECT().GetLanguages(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/fanfic-languages").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get languages")
}

func TestGetFanficSeries_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fs.EXPECT().GetSeries(mock.Anything).Return([]string{"umineko", "higurashi"}, nil)

	// when
	status, body := h.NewRequest("GET", "/fanfic-series").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string][]string](t, body)
	assert.Equal(t, []string{"umineko", "higurashi"}, got["series"])
}

func TestGetFanficSeries_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fs.EXPECT().GetSeries(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/fanfic-series").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get series")
}

func TestSearchOCCharacters_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fs.EXPECT().SearchOCCharacters(mock.Anything, "bea").Return([]string{"beatrice"}, nil)

	// when
	status, body := h.NewRequest("GET", "/fanfic-oc-characters?q=bea").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string][]string](t, body)
	assert.Equal(t, []string{"beatrice"}, got["characters"])
}

func TestSearchOCCharacters_EmptyQuery_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fs.EXPECT().SearchOCCharacters(mock.Anything, "").Return([]string{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/fanfic-oc-characters").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestSearchOCCharacters_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	fs.EXPECT().SearchOCCharacters(mock.Anything, "").Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/fanfic-oc-characters").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to search characters")
}

func TestListUserFanfics_Anonymous_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	targetUserID := uuid.New()
	fs.EXPECT().ListFanficsByUser(mock.Anything, targetUserID, uuid.Nil, 20, 0).Return(&dto.FanficListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/fanfics").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserFanfics_Authenticated_PassesViewerID(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	viewerID := uuid.New()
	targetUserID := uuid.New()
	h.ExpectValidSession("valid-cookie", viewerID)
	fs.EXPECT().ListFanficsByUser(mock.Anything, targetUserID, viewerID, 20, 0).Return(&dto.FanficListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/fanfics").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserFanfics_CustomPaging(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	targetUserID := uuid.New()
	fs.EXPECT().ListFanficsByUser(mock.Anything, targetUserID, uuid.Nil, 5, 10).Return(&dto.FanficListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/fanfics?limit=5&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserFanfics_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/fanfics").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestListUserFanfics_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	targetUserID := uuid.New()
	fs.EXPECT().ListFanficsByUser(mock.Anything, targetUserID, uuid.Nil, 20, 0).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+targetUserID.String()+"/fanfics").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list user fanfics")
}

func TestListUserFanficFavourites_Anonymous_OK(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	targetUserID := uuid.New()
	fs.EXPECT().ListFavourites(mock.Anything, targetUserID, uuid.Nil, 20, 0).Return(&dto.FanficListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/fanfic-favourites").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserFanficFavourites_Authenticated_PassesViewerID(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	viewerID := uuid.New()
	targetUserID := uuid.New()
	h.ExpectValidSession("valid-cookie", viewerID)
	fs.EXPECT().ListFavourites(mock.Anything, targetUserID, viewerID, 20, 0).Return(&dto.FanficListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/fanfic-favourites").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserFanficFavourites_CustomPaging(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	targetUserID := uuid.New()
	fs.EXPECT().ListFavourites(mock.Anything, targetUserID, uuid.Nil, 5, 10).Return(&dto.FanficListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/fanfic-favourites?limit=5&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserFanficFavourites_InvalidID(t *testing.T) {
	// given
	h, _ := newFanficHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/fanfic-favourites").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestListUserFanficFavourites_InternalError(t *testing.T) {
	// given
	h, fs := newFanficHarness(t)
	targetUserID := uuid.New()
	fs.EXPECT().ListFavourites(mock.Anything, targetUserID, uuid.Nil, 20, 0).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+targetUserID.String()+"/fanfic-favourites").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list favourites")
}
