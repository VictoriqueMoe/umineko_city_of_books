package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	giphysvc "umineko_city_of_books/internal/giphy"

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
