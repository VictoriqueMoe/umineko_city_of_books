package controllers

import (
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type searchDeps struct {
	svc *search.MockService
}

func newSearchHarness(t *testing.T) (*testutil.Harness, searchDeps) {
	h := testutil.NewHarness(t)
	deps := searchDeps{
		svc: search.NewMockService(t),
	}
	s := &Service{
		SearchService:   deps.svc,
		SettingsService: h.SettingsService,
		AuthSession:     h.SessionManager,
		AuthzService:    h.AuthzService,
	}
	for _, setup := range s.getAllSearchRoutes() {
		setup(h.App)
	}
	return h, deps
}

func TestSearchController_EmptyQuery_ReturnsEmpty(t *testing.T) {
	// given
	h, _ := newSearchHarness(t)

	// when
	status, body := h.NewRequest(http.MethodGet, "/search?q=").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
	parsed := testutil.UnmarshalJSON[map[string]any](t, body)
	assert.Equal(t, float64(0), parsed["total"])
	results, _ := parsed["results"].([]any)
	assert.Empty(t, results)
}

func TestSearchController_FullSearch_PassesParamsAndShapesResponse(t *testing.T) {
	// given
	h, deps := newSearchHarness(t)
	parentID := "parent-uuid"
	deps.svc.EXPECT().ParseTypes("theory").
		Return([]repository.SearchEntityType{repository.SearchEntityTheory})
	deps.svc.EXPECT().
		Search(mock.Anything, "beatrice", []repository.SearchEntityType{repository.SearchEntityTheory}, 10, 5).
		Return([]search.Result{
			{
				SearchResult: repository.SearchResult{
					EntityType:        repository.SearchEntityPostComment,
					ID:                "comment-id",
					ParentID:          &parentID,
					Title:             "On a post",
					Snippet:           "matched <mark>beatrice</mark>",
					AuthorUsername:    "loliduck",
					AuthorDisplayName: "Loli",
					CreatedAt:         "2026-04-28T00:00:00Z",
				},
				URL: "/game-board/parent-uuid#comment-comment-id",
			},
		}, 1, nil)

	// when
	status, body := h.NewRequest(http.MethodGet, "/search?q=beatrice&types=theory&limit=10&offset=5").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
	parsed := testutil.UnmarshalJSON[map[string]any](t, body)
	assert.Equal(t, float64(1), parsed["total"])
	results, _ := parsed["results"].([]any)
	require.Len(t, results, 1)
	first := results[0].(map[string]any)
	assert.Equal(t, "post_comment", first["type"])
	assert.Equal(t, "/game-board/parent-uuid#comment-comment-id", first["url"])
	author := first["author"].(map[string]any)
	assert.Equal(t, "loliduck", author["username"])
}

func TestSearchController_QuickSearch_DelegatesToService(t *testing.T) {
	// given
	h, deps := newSearchHarness(t)
	deps.svc.EXPECT().QuickSearch(mock.Anything, "kinzo", 5).
		Return([]search.Result{
			{
				SearchResult: repository.SearchResult{EntityType: repository.SearchEntityTheory, ID: "theory-id", Title: "Kinzo"},
				URL:          "/theory/theory-id",
			},
		}, nil)

	// when
	status, body := h.NewRequest(http.MethodGet, "/search/quick?q=kinzo&perType=5").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
	parsed := testutil.UnmarshalJSON[map[string]any](t, body)
	results := parsed["results"].([]any)
	require.Len(t, results, 1)
	first := results[0].(map[string]any)
	assert.Equal(t, "/theory/theory-id", first["url"])
}

func TestSearchController_QuickSearch_EmptyQ_NoCall(t *testing.T) {
	// given
	h, _ := newSearchHarness(t)

	// when
	status, body := h.NewRequest(http.MethodGet, "/search/quick?q=").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
	parsed := testutil.UnmarshalJSON[map[string]any](t, body)
	results, _ := parsed["results"].([]any)
	assert.Empty(t, results)
}
