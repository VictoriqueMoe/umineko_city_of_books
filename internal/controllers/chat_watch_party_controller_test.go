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

func wpRoomID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func wpSessionID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func wpTargetID() uuid.UUID {
	return uuid.MustParse("33333333-3333-3333-3333-333333333333")
}

func TestListWatchParties_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET", "/chat/rooms/"+wpRoomID().String()+"/watch-parties", nil)
}

func TestListWatchParties_InvalidRoomID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("GET", "/chat/rooms/not-a-uuid/watch-parties").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid")
}

func TestListWatchParties_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListWatchParties(mock.Anything, wpRoomID(), userID).
		Return(&dto.WatchPartyListResponse{Enabled: true}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/"+wpRoomID().String()+"/watch-parties").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestStartWatchParty_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST", "/chat/rooms/"+wpRoomID().String()+"/watch-parties", dto.StartWatchPartyRequest{})
}

func TestStartWatchParty_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().StartWatchParty(mock.Anything, wpRoomID(), userID, "https://example.com", "EU", "Movie").
		Return(&dto.StartWatchPartyResponse{EmbedURL: "https://hb/embed"}, nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/rooms/"+wpRoomID().String()+"/watch-parties").
		WithCookie("valid-cookie").
		WithJSONBody(dto.StartWatchPartyRequest{StartURL: "https://example.com", Region: "EU", Title: "Movie"}).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestStartWatchParty_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"disabled", chatsvc.ErrWatchPartyDisabled, http.StatusServiceUnavailable, "not configured"},
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"wrong room type", chatsvc.ErrWatchPartyWrongRoomType, http.StatusBadRequest, "group chat rooms"},
		{"room not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "room not found"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "watch party request failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().StartWatchParty(mock.Anything, wpRoomID(), userID, "", "", "").
				Return(nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/rooms/"+wpRoomID().String()+"/watch-parties").
				WithCookie("valid-cookie").
				WithJSONBody(dto.StartWatchPartyRequest{}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestJoinWatchParty_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST", "/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/join", nil)
}

func TestJoinWatchParty_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().JoinWatchParty(mock.Anything, wpRoomID(), wpSessionID(), userID).
		Return(&dto.JoinWatchPartyResponse{EmbedURL: "https://hb/embed"}, nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/join").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestJoinWatchParty_NotActive(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().JoinWatchParty(mock.Anything, wpRoomID(), wpSessionID(), userID).
		Return(nil, chatsvc.ErrWatchPartyNotActive)

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/join").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "no such active watch party")
}

func TestLeaveWatchParty_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE", "/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/participants/me", nil)
}

func TestLeaveWatchParty_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().LeaveWatchParty(mock.Anything, wpRoomID(), wpSessionID(), userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/participants/me").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGrantWatchPartyControl_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "PATCH",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/participants/"+wpTargetID().String(),
		nil)
}

func TestGrantWatchPartyControl_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GrantWatchPartyControl(mock.Anything, wpRoomID(), wpSessionID(), userID, wpTargetID()).Return(nil)

	// when
	status, _ := h.NewRequest("PATCH",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/participants/"+wpTargetID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGrantWatchPartyControl_NotController(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GrantWatchPartyControl(mock.Anything, wpRoomID(), wpSessionID(), userID, wpTargetID()).
		Return(chatsvc.ErrWatchPartyNotController)

	// when
	status, body := h.NewRequest("PATCH",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/participants/"+wpTargetID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "only the watch party controller")
}

func TestGrantWatchPartyControl_NotParticipant(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GrantWatchPartyControl(mock.Anything, wpRoomID(), wpSessionID(), userID, wpTargetID()).
		Return(chatsvc.ErrWatchPartyNotParticipant)

	// when
	status, body := h.NewRequest("PATCH",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/participants/"+wpTargetID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "not a participant")
}

func TestEndWatchParty_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE", "/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String(), nil)
}

func TestEndWatchParty_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().EndWatchParty(mock.Anything, wpRoomID(), wpSessionID(), userID, "controller_ended").Return(nil)

	// when
	status, _ := h.NewRequest("DELETE",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestEndWatchParty_NotController(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().EndWatchParty(mock.Anything, wpRoomID(), wpSessionID(), userID, "controller_ended").
		Return(chatsvc.ErrWatchPartyNotController)

	// when
	status, body := h.NewRequest("DELETE",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "only the watch party controller")
}

func TestListWatchPartyMessages_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/messages", nil)
}

func TestListWatchPartyMessages_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GetWatchPartyMessages(mock.Anything, wpRoomID(), wpSessionID(), userID).
		Return(&dto.WatchPartyMessagesResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/messages").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestSendWatchPartyMessage_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/messages",
		dto.SendWatchPartyMessageRequest{Body: "hi"})
}

func TestSendWatchPartyMessage_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().SendWatchPartyMessage(mock.Anything, wpRoomID(), wpSessionID(), userID, "hello").
		Return(&dto.WatchPartyMessage{Body: "hello"}, nil)

	// when
	status, _ := h.NewRequest("POST",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/messages").
		WithCookie("valid-cookie").
		WithJSONBody(dto.SendWatchPartyMessageRequest{Body: "hello"}).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestSendWatchPartyMessage_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"empty body", chatsvc.ErrWatchPartyMessageEmpty, http.StatusBadRequest, "message body is required"},
		{"too long", chatsvc.ErrWatchPartyMessageTooLong, http.StatusBadRequest, "too long"},
		{"not participant", chatsvc.ErrWatchPartyNotParticipant, http.StatusForbidden, "not a participant"},
		{"not active", chatsvc.ErrWatchPartyNotActive, http.StatusNotFound, "no such active watch party"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().SendWatchPartyMessage(mock.Anything, wpRoomID(), wpSessionID(), userID, "hi").
				Return(nil, tc.err)

			// when
			status, body := h.NewRequest("POST",
				"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/messages").
				WithCookie("valid-cookie").
				WithJSONBody(dto.SendWatchPartyMessageRequest{Body: "hi"}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestIdentifyWatchPartyParticipant_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/identify",
		dto.IdentifyWatchPartyParticipantRequest{Identifier: "id-abc"})
}

func TestIdentifyWatchPartyParticipant_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().IdentifyWatchPartyParticipant(mock.Anything, wpRoomID(), wpSessionID(), userID, "id-abc").
		Return(nil)

	// when
	status, _ := h.NewRequest("POST",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/identify").
		WithCookie("valid-cookie").
		WithJSONBody(dto.IdentifyWatchPartyParticipantRequest{Identifier: "id-abc"}).Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestIdentifyWatchPartyParticipant_NotParticipant(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().IdentifyWatchPartyParticipant(mock.Anything, wpRoomID(), wpSessionID(), userID, "id-abc").
		Return(chatsvc.ErrWatchPartyNotParticipant)

	// when
	status, body := h.NewRequest("POST",
		"/chat/rooms/"+wpRoomID().String()+"/watch-parties/"+wpSessionID().String()+"/identify").
		WithCookie("valid-cookie").
		WithJSONBody(dto.IdentifyWatchPartyParticipantRequest{Identifier: "id-abc"}).Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "not a participant")
}
