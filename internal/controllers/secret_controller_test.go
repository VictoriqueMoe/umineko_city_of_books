package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/controllers/utils"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	secretsvc "umineko_city_of_books/internal/secret"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newSecretHarness(t *testing.T) (*testutil.Harness, *secretsvc.MockService) {
	h := testutil.NewHarness(t)
	ms := secretsvc.NewMockService(t)

	s := &Service{
		SecretService: ms,
		AuthSession:   h.SessionManager,
		AuthService:   h.AuthService,
		AuthzService:  h.AuthzService,
	}
	for _, setup := range s.getAllSecretRoutes() {
		setup(h.App)
	}
	return h, ms
}

func TestListSecrets_OK(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	ms.EXPECT().List(mock.Anything, uuid.Nil).Return(&dto.SecretListResponse{
		Secrets: []dto.SecretSummary{{ID: "witchHunter", Title: "The Witch's Epitaph"}},
	}, nil)

	// when
	status, body := h.NewRequest("GET", "/secrets").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.SecretListResponse](t, body)
	require.Len(t, got.Secrets, 1)
	assert.Equal(t, "witchHunter", got.Secrets[0].ID)
}

func TestGetSecret_OK(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	ms.EXPECT().Get(mock.Anything, "witchHunter", uuid.Nil).Return(&dto.SecretDetailResponse{
		ID: "witchHunter", Title: "The Witch's Epitaph",
		Riddle: "Uu~",
	}, nil)

	// when
	status, body := h.NewRequest("GET", "/secrets/witchHunter").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.SecretDetailResponse](t, body)
	assert.Equal(t, "witchHunter", got.ID)
	assert.Equal(t, "Uu~", got.Riddle)
}

func TestGetSecret_NotFound(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	ms.EXPECT().Get(mock.Anything, "nonsense", uuid.Nil).Return(nil, secretsvc.ErrNotFound)

	// when
	status, _ := h.NewRequest("GET", "/secrets/nonsense").Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
}

func TestCreateSecretComment_OK(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid", userID)
	ms.EXPECT().
		CreateComment(mock.Anything, "witchHunter", userID, dto.CreateCommentRequest{Body: "hi"}, []upload.FileUpload(nil)).
		Return(commentID, nil)

	// when
	status, body := h.NewRequest("POST", "/secrets/witchHunter/comments").
		WithCookie("valid").
		WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	assert.Contains(t, string(body), commentID.String())
}

func TestCreateSecretComment_EmptyBody(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	ms.EXPECT().
		CreateComment(mock.Anything, "witchHunter", userID, mock.Anything, []upload.FileUpload(nil)).
		Return(uuid.Nil, comment.ErrEmptyBody)

	// when
	status, body := h.NewRequest("POST", "/secrets/witchHunter/comments").
		WithCookie("valid").
		WithJSONBody(dto.CreateCommentRequest{Body: "   "}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), comment.ErrEmptyBody.Error())
}

func TestCreateSecretComment_NotFound(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	ms.EXPECT().
		CreateComment(mock.Anything, "nonsense", userID, mock.Anything, []upload.FileUpload(nil)).
		Return(uuid.Nil, secretsvc.ErrNotFound)

	// when
	status, _ := h.NewRequest("POST", "/secrets/nonsense/comments").
		WithCookie("valid").
		WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
}

func TestCreateSecretComment_ServiceOutcomes(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"refused upload", upload.ErrFileTooLarge, http.StatusBadRequest, upload.ErrFileTooLarge.Error()},
		{"server failure", errors.New("pq: connection refused"), http.StatusInternalServerError, `{"error":"failed to create comment"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newSecretHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid", userID)
			ms.EXPECT().
				CreateComment(mock.Anything, "witchHunter", userID, dto.CreateCommentRequest{Body: "hi"}, []upload.FileUpload(nil)).
				Return(uuid.Nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/secrets/witchHunter/comments").
				WithCookie("valid").
				WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestCreateSecretComment_MultipartPassesTheMediaThrough(t *testing.T) {
	// given a multipart comment with no text and one spoilered image
	h, ms := newSecretHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid", userID)
	ms.EXPECT().
		CreateComment(mock.Anything, "witchHunter", userID, dto.CreateCommentRequest{}, mock.MatchedBy(func(files []upload.FileUpload) bool {
			return len(files) == 1 && files[0].Filename == "pic.png" && files[0].ContentType == "image/png" && files[0].IsSpoiler
		})).
		Return(commentID, nil)
	form, contentType := testutil.MediaForm(t, utils.MediaField, map[string]string{
		utils.PayloadField:        `{"body":""}`,
		utils.SpoilerIndexesField: "0",
	})

	// when
	status, body := h.NewRequest("POST", "/secrets/witchHunter/comments").
		WithCookie("valid").
		WithRawBody(form, contentType).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	assert.Contains(t, string(body), commentID.String())
}

func TestListSecrets_Unauthenticated_StillOK(t *testing.T) {
	// given: OptionalAuth should not require a cookie.
	h, ms := newSecretHarness(t)
	ms.EXPECT().List(mock.Anything, uuid.Nil).Return(&dto.SecretListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/secrets").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}
