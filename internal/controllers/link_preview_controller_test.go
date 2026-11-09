package controllers

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/linkpreview"
	"umineko_city_of_books/internal/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newLinkPreviewHarness(t *testing.T) (*testutil.Harness, *linkpreview.MockService) {
	h := testutil.NewHarness(t)
	previews := linkpreview.NewMockService(t)

	s := &Service{
		LinkPreviewService: previews,
		AuthSession:        h.SessionManager,
		AuthService:        h.AuthService,
		AuthzService:       h.AuthzService,
		RateLimiter:        middleware.NewRateLimiter(cache.NewManager(engines.NewInMemory(0)), h.SettingsService),
	}
	for _, setup := range s.getAllLinkPreviewRoutes() {
		setup(h.App)
	}

	return h, previews
}

func TestLinkPreview_ABusyFetcherIsAServiceUnavailable(t *testing.T) {
	// given
	h, previews := newLinkPreviewHarness(t)
	target := "https://example.com/umineko"
	previews.EXPECT().Resolve(mock.Anything, target).Return(dto.LinkPreviewResponse{}, linkpreview.ErrBusy).Once()

	// when
	status, body := h.NewRequest(http.MethodGet, "/link-preview?url="+url.QueryEscape(target)).Do()

	// then
	require.Equal(t, http.StatusServiceUnavailable, status)
	assert.Contains(t, string(body), "try again shortly")
}

func TestLinkPreview_RateLimitByClientIP(t *testing.T) {
	// given
	h, previews := newLinkPreviewHarness(t)
	target := "https://example.com/umineko"
	path := "/link-preview?url=" + url.QueryEscape(target)
	previews.EXPECT().Resolve(mock.Anything, target).
		Return(dto.LinkPreviewResponse{URL: target, Type: "link"}, nil).
		Times(testutil.HarnessIntSetting)

	// when
	allowedStatuses := make([]int, 0, testutil.HarnessIntSetting)
	for range testutil.HarnessIntSetting {
		status, _ := h.NewRequest(http.MethodGet, path).Do()
		allowedStatuses = append(allowedStatuses, status)
	}
	overflow := h.NewRequest(http.MethodGet, path).DoResponse()

	// then the request over budget never reaches the service
	for _, status := range allowedStatuses {
		assert.Equal(t, http.StatusOK, status)
	}
	require.Equal(t, http.StatusTooManyRequests, overflow.StatusCode)
	h.SettingsService.AssertCalled(t, "GetInt", mock.Anything, config.SettingRateLimitLinkPreview)

	retryAfter, err := strconv.Atoi(overflow.Header.Get("Retry-After"))
	require.NoError(t, err)
	assert.Positive(t, retryAfter)
	assert.LessOrEqual(t, retryAfter, 60)

	body, err := io.ReadAll(overflow.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"error":"too many requests, please try again later"}`, string(body))
}
