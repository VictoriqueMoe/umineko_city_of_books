package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	giphysvc "umineko_city_of_books/internal/giphy"
	giphyfavourite "umineko_city_of_books/internal/giphy/favourite"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newGiphyHarness(t *testing.T) (*testutil.Harness, *giphysvc.MockService) {
	h := testutil.NewHarness(t)
	gs := giphysvc.NewMockService(t)

	s := &Service{
		GiphyService: gs,
		AuthSession:  h.SessionManager,
		AuthzService: h.AuthzService,
	}
	for _, setup := range s.getAllGiphyRoutes() {
		setup(h.App)
	}
	return h, gs
}

func TestGiphySearch_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newGiphyHarness, "GET", "/giphy/search?q=cats", nil)
}

func TestGiphySearch_OK(t *testing.T) {
	h, gs := newGiphyHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	expected := &giphysvc.Response{
		Data: []giphysvc.Gif{{ID: "abc", Title: "cat"}},
	}
	gs.EXPECT().Search(mock.Anything, "cats", 0, 24).Return(expected, nil)

	status, body := h.NewRequest("GET", "/giphy/search?q=cats").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[giphysvc.Response](t, body)
	require.Len(t, got.Data, 1)
	assert.Equal(t, "abc", got.Data[0].ID)
}

func TestGiphySearch_PassesOffsetAndLimit(t *testing.T) {
	h, gs := newGiphyHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	gs.EXPECT().Search(mock.Anything, "cats", 50, 10).Return(&giphysvc.Response{}, nil)

	status, _ := h.NewRequest("GET", "/giphy/search?q=cats&offset=50&limit=10").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusOK, status)
}

func TestGiphySearch_MissingQuery(t *testing.T) {
	h, _ := newGiphyHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	status, body := h.NewRequest("GET", "/giphy/search").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "query is required")
}

func TestGiphySearch_Disabled(t *testing.T) {
	h, gs := newGiphyHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	gs.EXPECT().Search(mock.Anything, "cats", 0, 24).Return(nil, giphysvc.ErrDisabled)

	status, body := h.NewRequest("GET", "/giphy/search?q=cats").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "not configured")
}

func TestGiphySearch_UpstreamError(t *testing.T) {
	h, gs := newGiphyHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	gs.EXPECT().Search(mock.Anything, "cats", 0, 24).Return(nil, errors.New("boom"))

	status, body := h.NewRequest("GET", "/giphy/search?q=cats").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "gif search failed")
}

func TestGiphyTrending_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newGiphyHarness, "GET", "/giphy/trending", nil)
}

func TestGiphyTrending_OK(t *testing.T) {
	h, gs := newGiphyHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	expected := &giphysvc.Response{Data: []giphysvc.Gif{{ID: "t1"}}}
	gs.EXPECT().Trending(mock.Anything, 0, 24).Return(expected, nil)

	status, body := h.NewRequest("GET", "/giphy/trending").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[giphysvc.Response](t, body)
	require.Len(t, got.Data, 1)
	assert.Equal(t, "t1", got.Data[0].ID)
}

func TestGiphyTrending_Disabled(t *testing.T) {
	h, gs := newGiphyHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	gs.EXPECT().Trending(mock.Anything, 0, 24).Return(nil, giphysvc.ErrDisabled)

	status, body := h.NewRequest("GET", "/giphy/trending").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "not configured")
}

func TestGiphyTrending_UpstreamError(t *testing.T) {
	h, gs := newGiphyHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	gs.EXPECT().Trending(mock.Anything, 0, 24).Return(nil, errors.New("boom"))

	status, body := h.NewRequest("GET", "/giphy/trending").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "gif trending failed")
}

func newGiphyFavouriteHarness(t *testing.T) (*testutil.Harness, *giphyfavourite.MockService) {
	h := testutil.NewHarness(t)
	fs := giphyfavourite.NewMockService(t)

	s := &Service{
		GiphyFavouriteService: fs,
		AuthSession:           h.SessionManager,
		AuthzService:          h.AuthzService,
	}
	for _, setup := range s.getAllGiphyRoutes() {
		setup(h.App)
	}
	return h, fs
}

func TestGiphyFavouritesList_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newGiphyFavouriteHarness, "GET", "/giphy/favourites", nil)
}

func TestGiphyFavouritesList_OK(t *testing.T) {
	h, fs := newGiphyFavouriteHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	favs := []giphyfavourite.Favourite{
		{GiphyID: "a", URL: "urlA", Title: "A"},
		{GiphyID: "b", URL: "urlB", Title: "B"},
	}
	fs.EXPECT().List(mock.Anything, userID, 50, 0).Return(favs, 2, nil)

	status, body := h.NewRequest("GET", "/giphy/favourites").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[struct {
		Data  []giphyfavourite.Favourite `json:"data"`
		Total int                        `json:"total"`
	}](t, body)
	assert.Equal(t, 2, got.Total)
	require.Len(t, got.Data, 2)
	assert.Equal(t, "a", got.Data[0].GiphyID)
}

func TestGiphyFavouritesList_PassesOffsetAndLimit(t *testing.T) {
	h, fs := newGiphyFavouriteHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().List(mock.Anything, userID, 10, 40).Return([]giphyfavourite.Favourite{}, 0, nil)

	status, _ := h.NewRequest("GET", "/giphy/favourites?offset=40&limit=10").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusOK, status)
}

func TestGiphyFavouritesList_UpstreamError(t *testing.T) {
	h, fs := newGiphyFavouriteHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	fs.EXPECT().List(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, 0, errors.New("boom"))

	status, body := h.NewRequest("GET", "/giphy/favourites").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load favourites")
}

func TestGiphyFavouritesAdd_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newGiphyFavouriteHarness, "POST", "/giphy/favourites", map[string]any{
		"giphy_id": "abc",
		"url":      "https://media.giphy.com/abc.gif",
	})
}

func TestGiphyFavouritesAdd_OK(t *testing.T) {
	h, fs := newGiphyFavouriteHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().Add(mock.Anything, userID, giphyfavourite.Favourite{
		GiphyID:    "abc",
		URL:        "https://media.giphy.com/abc.gif",
		Title:      "cat",
		PreviewURL: "https://media.giphy.com/abc-preview.gif",
		Width:      100,
		Height:     50,
	}).Return(nil)

	status, _ := h.NewRequest("POST", "/giphy/favourites").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{
			"giphy_id":    "abc",
			"url":         "https://media.giphy.com/abc.gif",
			"title":       "cat",
			"preview_url": "https://media.giphy.com/abc-preview.gif",
			"width":       100,
			"height":      50,
		}).
		Do()

	require.Equal(t, http.StatusNoContent, status)
}

func TestGiphyFavouritesAdd_MissingFields(t *testing.T) {
	h, _ := newGiphyFavouriteHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	status, body := h.NewRequest("POST", "/giphy/favourites").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"giphy_id": "abc"}).
		Do()

	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "giphy_id and url are required")
}

func TestGiphyFavouritesAdd_TrimsWhitespace(t *testing.T) {
	h, _ := newGiphyFavouriteHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	status, body := h.NewRequest("POST", "/giphy/favourites").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"giphy_id": "   ", "url": "  "}).
		Do()

	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "giphy_id and url are required")
}

func TestGiphyFavouritesAdd_UpstreamError(t *testing.T) {
	h, fs := newGiphyFavouriteHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	fs.EXPECT().Add(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("boom"))

	status, body := h.NewRequest("POST", "/giphy/favourites").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"giphy_id": "abc", "url": "https://x"}).
		Do()

	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to add favourite")
}

func TestGiphyFavouritesRemove_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newGiphyFavouriteHarness, "DELETE", "/giphy/favourites/abc", nil)
}

func TestGiphyFavouritesRemove_OK(t *testing.T) {
	h, fs := newGiphyFavouriteHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	fs.EXPECT().Remove(mock.Anything, userID, "abc").Return(nil)

	status, _ := h.NewRequest("DELETE", "/giphy/favourites/abc").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusNoContent, status)
}

func TestGiphyFavouritesRemove_UpstreamError(t *testing.T) {
	h, fs := newGiphyFavouriteHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	fs.EXPECT().Remove(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("boom"))

	status, body := h.NewRequest("DELETE", "/giphy/favourites/abc").
		WithCookie("valid-cookie").
		Do()

	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to remove favourite")
}
