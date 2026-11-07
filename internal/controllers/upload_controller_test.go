package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newUploadHarness(t *testing.T) (*testutil.Harness, *upload.MockService) {
	h := testutil.NewHarness(t)
	us := upload.NewMockService(t)
	testutil.ExpectStaging(t, us)

	s := &Service{
		UploadService:   us,
		SettingsService: h.SettingsService,
		AuthSession:     h.SessionManager,
		AuthService:     h.AuthService,
		AuthzService:    h.AuthzService,
	}
	for _, setup := range s.getAllPendingUploadRoutes() {
		setup(h.App)
	}

	return h, us
}

func TestUploadRoutes_RequireASession(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{method: "POST", path: "/uploads/media"},
		{method: "POST", path: "/uploads/files"},
		{method: "GET", path: "/uploads?ids=" + uuid.NewString()},
		{method: "DELETE", path: "/uploads/" + uuid.NewString()},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			testutil.RunAuthFailureSuite(t, newUploadHarness, route.method, route.path, nil)
		})
	}
}

func TestBeginUpload_HandsTheFileToTheMatchingKind(t *testing.T) {
	routes := []struct {
		name   string
		path   string
		expect func(us *upload.MockService, userID uuid.UUID) *mock.Call
	}{
		{
			name: "media",
			path: "/uploads/media",
			expect: func(us *upload.MockService, userID uuid.UUID) *mock.Call {
				return us.EXPECT().BeginMediaUpload(mock.Anything, userID, testutil.FileUploadOf("image/png", "pic.png")).Call
			},
		},
		{
			name: "a document",
			path: "/uploads/files",
			expect: func(us *upload.MockService, userID uuid.UUID) *mock.Call {
				return us.EXPECT().BeginFileUpload(mock.Anything, userID, testutil.FileUploadOf("image/png", "pic.png")).Call
			},
		},
	}

	outcomes := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{name: "accepted", wantCode: http.StatusAccepted, wantBody: `"status":"processing"`},
		{name: "a refused file", err: upload.ErrInvalidFileType, wantCode: http.StatusBadRequest, wantBody: upload.ErrInvalidFileType.Error()},
		{name: "a server failure is not echoed back", err: errors.New("pq: connection refused"), wantCode: http.StatusInternalServerError, wantBody: "failed to accept the upload"},
	}

	for _, route := range routes {
		for _, tc := range outcomes {
			t.Run(route.name+" "+tc.name, func(t *testing.T) {
				// given
				h, us := newUploadHarness(t)
				userID := uuid.New()
				h.ExpectValidSession("valid-cookie", userID)
				var resp *dto.PendingUploadResponse
				if tc.err == nil {
					resp = &dto.PendingUploadResponse{ID: uuid.New(), MediaType: "image", Filename: "pic.png", Status: "processing"}
				}
				route.expect(us, userID).Return(resp, tc.err).Once()
				form, contentType := testutil.MediaForm(t, "file", nil)

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

func TestBeginUpload_NewAccountsCannotUpload(t *testing.T) {
	// given a strict upload mock that expects nothing
	h, _ := newUploadHarness(t)
	expectNewAccountSession(h, "new-cookie", uuid.New())
	form, contentType := testutil.MediaForm(t, "file", nil)

	// when
	status, body := h.NewRequest("POST", "/uploads/media").WithCookie("new-cookie").WithRawBody(form, contentType).Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "new accounts cannot post attachments")
}

func TestBeginUpload_AMissingFileIsABadRequest(t *testing.T) {
	// given a strict upload mock that expects nothing
	h, _ := newUploadHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())
	form, contentType := testutil.MediaForm(t, "media", nil)

	// when
	status, body := h.NewRequest("POST", "/uploads/media").WithCookie("valid-cookie").WithRawBody(form, contentType).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no file provided")
}

func TestUploadStatuses(t *testing.T) {
	first, second := uuid.New(), uuid.New()

	tests := []struct {
		name     string
		query    string
		expect   func(us *upload.MockService, userID uuid.UUID)
		wantCode int
		wantBody string
	}{
		{
			name:  "every listed upload is looked up for the caller",
			query: "?ids=" + first.String() + "," + second.String(),
			expect: func(us *upload.MockService, userID uuid.UUID) {
				us.EXPECT().UploadStatuses(mock.Anything, userID, []uuid.UUID{first, second}).
					Return([]dto.PendingUploadResponse{{ID: first, Status: "ready"}}, nil).Once()
			},
			wantCode: http.StatusOK,
			wantBody: `"status":"ready"`,
		},
		{
			name:     "a malformed id is refused before the service",
			query:    "?ids=" + first.String() + ",beatrice",
			wantCode: http.StatusBadRequest,
			wantBody: "invalid upload id",
		},
		{
			name:  "a failed lookup is not echoed back",
			query: "?ids=" + first.String(),
			expect: func(us *upload.MockService, userID uuid.UUID) {
				us.EXPECT().UploadStatuses(mock.Anything, userID, []uuid.UUID{first}).Return(nil, errors.New("pq: connection refused")).Once()
			},
			wantCode: http.StatusInternalServerError,
			wantBody: "failed to load upload statuses",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, us := newUploadHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			if tc.expect != nil {
				tc.expect(us, userID)
			}

			// when
			status, body := h.NewRequest("GET", "/uploads"+tc.query).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
			assert.NotContains(t, string(body), "pq:")
		})
	}
}

func TestRemoveUpload(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{name: "removed", wantCode: http.StatusNoContent},
		{name: "a failed removal", err: errors.New("pq: connection refused"), wantCode: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, us := newUploadHarness(t)
			userID, id := uuid.New(), uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			us.EXPECT().RemoveUpload(mock.Anything, userID, id).Return(tc.err).Once()

			// when
			status, body := h.NewRequest("DELETE", "/uploads/"+id.String()).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.NotContains(t, string(body), "pq:")
		})
	}
}
