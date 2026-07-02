package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/stream"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newStreamHarness(t *testing.T) (*testutil.Harness, *stream.MockService) {
	h := testutil.NewHarness(t)
	ss := stream.NewMockService(t)

	s := &Service{
		StreamService: ss,
		AuthSession:   h.SessionManager,
		AuthzService:  h.AuthzService,
	}
	for _, setup := range s.getAllStreamRoutes() {
		setup(h.App)
	}
	return h, ss
}

func TestUpdateStreamTitle_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newStreamHarness, "PATCH", "/streams/"+uuid.NewString(), dto.UpdateStreamTitleRequest{Title: "x"})
}

func TestUpdateStreamTitle_OK(t *testing.T) {
	// given
	h, ss := newStreamHarness(t)
	userID := uuid.New()
	streamID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	want := &dto.LiveStreamResponse{ID: streamID, Title: "New title"}
	ss.EXPECT().UpdateTitle(mock.Anything, userID, streamID, "New title").Return(want, nil)

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
			name:     "internal error",
			err:      errors.New("boom"),
			wantCode: http.StatusInternalServerError,
			wantBody: "stream request failed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ss := newStreamHarness(t)
			userID := uuid.New()
			streamID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ss.EXPECT().UpdateTitle(mock.Anything, userID, streamID, "New title").Return(nil, tc.err)

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
