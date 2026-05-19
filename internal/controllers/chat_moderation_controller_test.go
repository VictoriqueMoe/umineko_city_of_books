package controllers

import (
	"errors"
	"net/http"
	"testing"

	chatsvc "umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func modRoomID() uuid.UUID {
	return uuid.MustParse("44444444-4444-4444-4444-444444444444")
}

func modTargetID() uuid.UUID {
	return uuid.MustParse("55555555-5555-5555-5555-555555555555")
}

func modRuleID() uuid.UUID {
	return uuid.MustParse("66666666-6666-6666-6666-666666666666")
}

func TestListRoomBans_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET", "/chat/rooms/"+modRoomID().String()+"/bans", nil)
}

func TestListRoomBans_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListRoomBans(mock.Anything, userID, modRoomID()).
		Return([]dto.ChatRoomBanResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/"+modRoomID().String()+"/bans").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListRoomBans_InvalidRoomID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/not-a-uuid/bans").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestBanMember_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST",
		"/chat/rooms/"+modRoomID().String()+"/bans/"+modTargetID().String(),
		dto.BanMemberRequest{Reason: "spam"})
}

func TestBanMember_OK_WithBody(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().BanMember(mock.Anything, userID, modRoomID(), modTargetID(), "spam").Return(nil)

	// when
	status, _ := h.NewRequest("POST",
		"/chat/rooms/"+modRoomID().String()+"/bans/"+modTargetID().String()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.BanMemberRequest{Reason: "spam"}).Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestBanMember_OK_NoBody(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().BanMember(mock.Anything, userID, modRoomID(), modTargetID(), "").Return(nil)

	// when
	status, _ := h.NewRequest("POST",
		"/chat/rooms/"+modRoomID().String()+"/bans/"+modTargetID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestBanMember_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"room not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "room not found"},
		{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host"},
		{"cannot ban staff", chatsvc.ErrCannotBanStaff, http.StatusForbidden, "host and site staff"},
		{"system room", chatsvc.ErrSystemRoom, http.StatusForbidden, "managed automatically"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().BanMember(mock.Anything, userID, modRoomID(), modTargetID(), "spam").Return(tc.err)

			// when
			status, body := h.NewRequest("POST",
				"/chat/rooms/"+modRoomID().String()+"/bans/"+modTargetID().String()).
				WithCookie("valid-cookie").
				WithJSONBody(dto.BanMemberRequest{Reason: "spam"}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnbanMember_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE",
		"/chat/rooms/"+modRoomID().String()+"/bans/"+modTargetID().String(), nil)
}

func TestUnbanMember_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().UnbanMember(mock.Anything, userID, modRoomID(), modTargetID()).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE",
		"/chat/rooms/"+modRoomID().String()+"/bans/"+modTargetID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestUnbanMember_NotHost(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().UnbanMember(mock.Anything, userID, modRoomID(), modTargetID()).
		Return(chatsvc.ErrNotHost)

	// when
	status, body := h.NewRequest("DELETE",
		"/chat/rooms/"+modRoomID().String()+"/bans/"+modTargetID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "only the host")
}

func TestListRoomBannedWords_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET",
		"/chat/rooms/"+modRoomID().String()+"/banned-words", nil)
}

func TestListRoomBannedWords_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListRoomBannedWords(mock.Anything, userID, modRoomID()).
		Return([]dto.BannedWordRuleResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET",
		"/chat/rooms/"+modRoomID().String()+"/banned-words").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListRoomBannedWords_NotHost(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListRoomBannedWords(mock.Anything, userID, modRoomID()).
		Return(nil, chatsvc.ErrNotHost)

	// when
	status, body := h.NewRequest("GET",
		"/chat/rooms/"+modRoomID().String()+"/banned-words").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "only the host")
}

func TestCreateRoomBannedWord_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST",
		"/chat/rooms/"+modRoomID().String()+"/banned-words",
		dto.CreateBannedWordRequest{Pattern: "spam", MatchMode: "substring", Action: "delete"})
}

func TestCreateRoomBannedWord_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateBannedWordRequest{Pattern: "spam", MatchMode: "substring", Action: "delete"}
	chatMock.EXPECT().CreateRoomBannedWord(mock.Anything, userID, modRoomID(), req).
		Return(&dto.BannedWordRuleResponse{ID: modRuleID().String(), Pattern: "spam"}, nil)

	// when
	status, _ := h.NewRequest("POST",
		"/chat/rooms/"+modRoomID().String()+"/banned-words").
		WithCookie("valid-cookie").
		WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestCreateRoomBannedWord_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"room not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "not found"},
		{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host"},
		{"mod role required", chatsvc.ErrModRoleRequired, http.StatusForbidden, "admin permission required"},
		{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "pattern is required"},
		{"invalid mode", chatsvc.ErrInvalidBannedWordMode, http.StatusBadRequest, "invalid match mode"},
		{"invalid action", chatsvc.ErrInvalidBannedWordAction, http.StatusBadRequest, "invalid action"},
		{"invalid regex", chatsvc.ErrInvalidBannedWordRegex, http.StatusBadRequest, "invalid regex pattern"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateBannedWordRequest{Pattern: "spam", MatchMode: "substring", Action: "delete"}
			chatMock.EXPECT().CreateRoomBannedWord(mock.Anything, userID, modRoomID(), req).
				Return(nil, tc.err)

			// when
			status, body := h.NewRequest("POST",
				"/chat/rooms/"+modRoomID().String()+"/banned-words").
				WithCookie("valid-cookie").
				WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUpdateRoomBannedWord_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "PUT",
		"/chat/rooms/"+modRoomID().String()+"/banned-words/"+modRuleID().String(),
		dto.UpdateBannedWordRequest{})
}

func TestUpdateRoomBannedWord_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateBannedWordRequest{Pattern: "updated", MatchMode: "substring", Action: "delete"}
	chatMock.EXPECT().UpdateRoomBannedWord(mock.Anything, userID, modRoomID(), modRuleID(), req).
		Return(&dto.BannedWordRuleResponse{ID: modRuleID().String(), Pattern: "updated"}, nil)

	// when
	status, _ := h.NewRequest("PUT",
		"/chat/rooms/"+modRoomID().String()+"/banned-words/"+modRuleID().String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestUpdateRoomBannedWord_RuleNotInScope(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateBannedWordRequest{Pattern: "updated", MatchMode: "substring", Action: "delete"}
	chatMock.EXPECT().UpdateRoomBannedWord(mock.Anything, userID, modRoomID(), modRuleID(), req).
		Return(nil, chatsvc.ErrBannedWordRuleMismatch)

	// when
	status, body := h.NewRequest("PUT",
		"/chat/rooms/"+modRoomID().String()+"/banned-words/"+modRuleID().String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "rule not found for this scope")
}

func TestDeleteRoomBannedWord_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE",
		"/chat/rooms/"+modRoomID().String()+"/banned-words/"+modRuleID().String(), nil)
}

func TestDeleteRoomBannedWord_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().DeleteRoomBannedWord(mock.Anything, userID, modRoomID(), modRuleID()).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE",
		"/chat/rooms/"+modRoomID().String()+"/banned-words/"+modRuleID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestDeleteRoomBannedWord_NotHost(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().DeleteRoomBannedWord(mock.Anything, userID, modRoomID(), modRuleID()).
		Return(chatsvc.ErrNotHost)

	// when
	status, body := h.NewRequest("DELETE",
		"/chat/rooms/"+modRoomID().String()+"/banned-words/"+modRuleID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "only the host")
}
