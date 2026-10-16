package controllers

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/stream"
	usersvc "umineko_city_of_books/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	streamDeps struct {
		stream *stream.MockService
		users  *usersvc.MockService
	}
)

func newStreamHarness(t *testing.T) (*testutil.Harness, streamDeps) {
	h := testutil.NewHarness(t)
	deps := streamDeps{
		stream: stream.NewMockService(t),
		users:  usersvc.NewMockService(t),
	}

	s := &Service{
		StreamService: deps.stream,
		UserService:   deps.users,
		AuthSession:   h.SessionManager,
		AuthService:   h.AuthService,
		AuthzService:  h.AuthzService,
		RateLimiter:   middleware.NewRateLimiter(cache.NewManager(engines.NewInMemory(0))),
	}
	for _, setup := range s.getAllStreamRoutes() {
		setup(h.App)
	}
	return h, deps
}

func TestUpdateStreamTitle_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newStreamHarness, "PATCH", "/streams/"+uuid.NewString(), dto.UpdateStreamTitleRequest{Title: "x"})
}

func TestUpdateStreamTitle_OK(t *testing.T) {
	// given
	h, deps := newStreamHarness(t)
	userID := uuid.New()
	streamID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	want := &dto.LiveStreamResponse{ID: streamID, Title: "New title"}
	deps.stream.EXPECT().UpdateTitle(mock.Anything, userID, streamID, "New title").Return(want, nil)

	// when
	status, body := h.NewRequest("PATCH", "/streams/"+streamID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateStreamTitleRequest{Title: "New title"}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.LiveStreamResponse](t, body)
	assert.Equal(t, streamID, got.ID)
	assert.Equal(t, "New title", got.Title)
}

func TestUpdateStreamTitle_InvalidID(t *testing.T) {
	// given
	h, _ := newStreamHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("PATCH", "/streams/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateStreamTitleRequest{Title: "New title"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateStreamTitle_BadJSON(t *testing.T) {
	// given
	h, _ := newStreamHarness(t)
	userID := uuid.New()
	streamID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("PATCH", "/streams/"+streamID.String()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestUpdateStreamTitle_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{
			name:     "title required",
			err:      stream.ErrTitleRequired,
			wantCode: http.StatusBadRequest,
			wantBody: "a stream title is required",
		},
		{
			name:     "stream not found",
			err:      stream.ErrStreamNotFound,
			wantCode: http.StatusNotFound,
			wantBody: "stream not found",
		},
		{
			name:     "not owner",
			err:      stream.ErrNotOwner,
			wantCode: http.StatusForbidden,
			wantBody: "you do not own this stream",
		},
		{
			name:     "internal error does not leak the cause",
			err:      errors.New("boom"),
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"stream request failed"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newStreamHarness(t)
			userID := uuid.New()
			streamID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.stream.EXPECT().UpdateTitle(mock.Anything, userID, streamID, "New title").Return(nil, tc.err)

			// when
			status, body := h.NewRequest("PATCH", "/streams/"+streamID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(dto.UpdateStreamTitleRequest{Title: "New title"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetStreamByUsername(t *testing.T) {
	streamID := uuid.New()

	cases := []struct {
		name       string
		username   string
		serviceErr error
		want       *dto.LiveStreamResponse
		wantStatus int
		wantBody   string
	}{
		{
			name:       "a live streamer resolves from their stable url",
			username:   "Featherine",
			want:       &dto.LiveStreamResponse{ID: streamID, Title: "Ciconia blind run", StreamerUsername: "Featherine"},
			wantStatus: http.StatusOK,
			wantBody:   "Ciconia blind run",
		},
		{
			name:       "an offline or unknown streamer is a 404",
			username:   "nobody",
			serviceErr: stream.ErrStreamNotFound,
			wantStatus: http.StatusNotFound,
			wantBody:   "stream not found",
		},
		{
			name:       "a username is never parsed as a stream id",
			username:   "not-a-uuid",
			serviceErr: stream.ErrStreamNotFound,
			wantStatus: http.StatusNotFound,
			wantBody:   "stream not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newStreamHarness(t)
			deps.stream.EXPECT().GetByUsername(mock.Anything, tc.username).Return(tc.want, tc.serviceErr)

			// when
			status, body := h.NewRequest(http.MethodGet, "/streams/user/"+tc.username).Do()

			// then
			require.Equal(t, tc.wantStatus, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestStreamRoutes_RateLimitByClientIP(t *testing.T) {
	userID := uuid.New()
	streamID := uuid.New()
	thumbnailBody, thumbnailContentType := testutil.MediaForm(t, "thumbnail", nil)

	cases := []struct {
		name    string
		budget  int
		setup   func(h *testutil.Harness, deps streamDeps, budget int)
		request func(h *testutil.Harness) *testutil.Request
	}{
		{
			name:   "an anonymous viewer token mint",
			budget: middleware.StreamViewerTokensPerMinute,
			setup: func(_ *testutil.Harness, deps streamDeps, budget int) {
				deps.stream.EXPECT().MintViewerToken(mock.Anything, streamID, (*dto.StreamViewer)(nil)).
					Return("token", "wss://livekit.example", nil).
					Times(budget)
			},
			request: func(h *testutil.Harness) *testutil.Request {
				return h.NewRequest(http.MethodPost, "/streams/"+streamID.String()+"/token")
			},
		},
		{
			name:   "a thumbnail upload",
			budget: middleware.StreamThumbnailsPerMinute,
			setup: func(h *testutil.Harness, deps streamDeps, budget int) {
				h.ExpectValidSession("valid-cookie", userID)
				deps.stream.EXPECT().SaveThumbnail(mock.Anything, userID, streamID, int64(len("payload")), mock.Anything).
					Return(nil).
					Times(budget)
			},
			request: func(h *testutil.Harness) *testutil.Request {
				return h.NewRequest(http.MethodPost, "/streams/"+streamID.String()+"/thumbnail").
					WithCookie("valid-cookie").
					WithRawBody(thumbnailBody, thumbnailContentType)
			},
		},
		{
			name:   "a stream credentials reset",
			budget: middleware.StreamCredentialsResetsPerMinute,
			setup: func(h *testutil.Harness, deps streamDeps, budget int) {
				h.ExpectValidSession("valid-cookie", userID)
				deps.users.EXPECT().GetByID(mock.Anything, userID).
					Return(&dto.UserResponse{ID: userID, Username: "beato", DisplayName: "Beatrice"}, nil).
					Times(budget)
				deps.stream.EXPECT().ResetCredentials(mock.Anything, userID, "Beatrice").
					Return(&dto.StreamCredentialsResponse{StreamKey: "key"}, nil).
					Times(budget)
			},
			request: func(h *testutil.Harness) *testutil.Request {
				return h.NewRequest(http.MethodPost, "/streams/credentials/reset").
					WithCookie("valid-cookie")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newStreamHarness(t)
			tc.setup(h, deps, tc.budget)

			// when
			allowedStatuses := make([]int, 0, tc.budget)
			for range tc.budget {
				status, _ := tc.request(h).Do()
				allowedStatuses = append(allowedStatuses, status)
			}
			overflow := tc.request(h).DoResponse()

			// then the request over budget never reaches the service
			for _, status := range allowedStatuses {
				assert.Equal(t, http.StatusOK, status)
			}
			require.Equal(t, http.StatusTooManyRequests, overflow.StatusCode)

			retryAfter, err := strconv.Atoi(overflow.Header.Get("Retry-After"))
			require.NoError(t, err)
			assert.Positive(t, retryAfter)
			assert.LessOrEqual(t, retryAfter, 60)

			body, err := io.ReadAll(overflow.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `{"error":"too many requests, please try again later"}`, string(body))
		})
	}
}
