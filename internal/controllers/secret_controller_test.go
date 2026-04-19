package controllers

import (
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	secretsvc "umineko_city_of_books/internal/secret"

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
		SecretSummary: dto.SecretSummary{ID: "witchHunter", Title: "The Witch's Epitaph"},
		Riddle:        "Uu~",
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
		CreateComment(mock.Anything, "witchHunter", userID, mock.MatchedBy(func(r dto.CreateSecretCommentRequest) bool { return r.Body == "hi" })).
		Return(commentID, nil)

	// when
	status, body := h.NewRequest("POST", "/secrets/witchHunter/comments").
		WithCookie("valid").
		WithJSONBody(dto.CreateSecretCommentRequest{Body: "hi"}).
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
		CreateComment(mock.Anything, "witchHunter", userID, mock.Anything).
		Return(uuid.Nil, secretsvc.ErrEmptyBody)

	// when
	status, body := h.NewRequest("POST", "/secrets/witchHunter/comments").
		WithCookie("valid").
		WithJSONBody(dto.CreateSecretCommentRequest{Body: "   "}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "empty")
}

func TestCreateSecretComment_NotFound(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	ms.EXPECT().
		CreateComment(mock.Anything, "nonsense", userID, mock.Anything).
		Return(uuid.Nil, secretsvc.ErrNotFound)

	// when
	status, _ := h.NewRequest("POST", "/secrets/nonsense/comments").
		WithCookie("valid").
		WithJSONBody(dto.CreateSecretCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
}

func TestDeleteSecretComment_OK(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid", userID)
	ms.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/secret-comments/"+commentID.String()).
		WithCookie("valid").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeSecretComment_OK(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid", userID)
	ms.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/secret-comments/"+commentID.String()+"/like").
		WithCookie("valid").
		WithJSONBody(map[string]any{}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeSecretComment_OK(t *testing.T) {
	// given
	h, ms := newSecretHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid", userID)
	ms.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/secret-comments/"+commentID.String()+"/like").
		WithCookie("valid").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestListSecrets_Unauthenticated_StillOK(t *testing.T) {
	// given — OptionalAuth should not require a cookie.
	h, ms := newSecretHarness(t)
	ms.EXPECT().List(mock.Anything, uuid.Nil).Return(&dto.SecretListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/secrets").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}
