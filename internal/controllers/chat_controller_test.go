package controllers

import (
	"errors"
	"net/http"
	"testing"

	chatsvc "umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newChatHarness(t *testing.T) (*testutil.Harness, *chatsvc.MockService) {
	h := testutil.NewHarness(t)
	chatMock := chatsvc.NewMockService(t)

	s := &Service{
		ChatService:     chatMock,
		SettingsService: h.SettingsService,
		AuthSession:     h.SessionManager,
		AuthzService:    h.AuthzService,
		Hub:             ws.NewHub(),
	}
	for _, setup := range s.getAllChatRoutes() {
		setup(h.App)
	}
	return h, chatMock
}

func chatFactory(t *testing.T) (*testutil.Harness, *chatsvc.MockService) {
	return newChatHarness(t)
}

func TestResolveDM_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET", "/chat/dm/"+uuid.NewString()+"/resolve", nil)
}

func TestResolveDM_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	recipientID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ResolveDMRoom(mock.Anything, userID, recipientID).
		Return(&dto.ResolveDMResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/dm/"+recipientID.String()+"/resolve").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestResolveDM_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("GET", "/chat/dm/not-a-uuid/resolve").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid userID")
}

func TestResolveDM_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"blocked", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot message"},
		{"dms disabled", chatsvc.ErrDmsDisabled, http.StatusForbidden, "DMs disabled"},
		{"user not found", chatsvc.ErrUserNotFound, http.StatusNotFound, "user not found"},
		{"cannot dm self", chatsvc.ErrCannotDMSelf, http.StatusBadRequest, "cannot DM yourself"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "chat operation failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			recipientID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().ResolveDMRoom(mock.Anything, userID, recipientID).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("GET", "/chat/dm/"+recipientID.String()+"/resolve").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestSendFirstDM_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST", "/chat/dm/"+uuid.NewString()+"/messages",
		dto.SendMessageRequest{Body: "hi"})
}

func TestSendFirstDM_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	recipientID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().SendDMMessage(mock.Anything, userID, recipientID, "hi").
		Return(&dto.SendDMResponse{}, nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/dm/"+recipientID.String()+"/messages").
		WithCookie("valid-cookie").
		WithJSONBody(dto.SendMessageRequest{Body: "hi"}).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestSendFirstDM_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/dm/not-a-uuid/messages").
		WithCookie("valid-cookie").
		WithJSONBody(dto.SendMessageRequest{Body: "hi"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid userID")
}

func TestSendFirstDM_BadJSON(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/dm/"+uuid.NewString()+"/messages").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestSendFirstDM_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "message body is required"},
		{"blocked", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot message"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "chat operation failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			recipientID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().SendDMMessage(mock.Anything, userID, recipientID, "hi").Return(nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/dm/"+recipientID.String()+"/messages").
				WithCookie("valid-cookie").
				WithJSONBody(dto.SendMessageRequest{Body: "hi"}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestCreateGroupRoom_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST", "/chat/rooms",
		dto.CreateGroupRoomRequest{Name: "room"})
}

func TestCreateGroupRoom_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateGroupRoomRequest{Name: "room"}
	chatMock.EXPECT().CreateGroupRoom(mock.Anything, userID, req).
		Return(&dto.ChatRoomResponse{ID: uuid.New(), Name: "room"}, nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/rooms").
		WithCookie("valid-cookie").
		WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestCreateGroupRoom_BadJSON(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestCreateGroupRoom_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "room name is required"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create group room"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateGroupRoomRequest{Name: ""}
			chatMock.EXPECT().CreateGroupRoom(mock.Anything, userID, req).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/rooms").
				WithCookie("valid-cookie").
				WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListRooms_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET", "/chat/rooms", nil)
}

func TestListRooms_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListRooms(mock.Anything, userID).
		Return(&dto.ChatRoomListResponse{Rooms: []dto.ChatRoomResponse{}, Total: 0}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListRooms_InternalError(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListRooms(mock.Anything, userID).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/chat/rooms").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list rooms")
}

func TestListMyGroupRooms_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET", "/chat/rooms/mine", nil)
}

func TestListMyGroupRooms_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListUserGroupRooms(mock.Anything, userID, "foo", true, "tag", "admin", 10, 5).
		Return(&dto.ChatRoomListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/mine?search=foo&rp=true&tag=tag&role=admin&limit=10&offset=5").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListMyGroupRooms_InternalError(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListUserGroupRooms(mock.Anything, userID, "", false, "", "", 20, 0).
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/chat/rooms/mine").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list rooms")
}

func TestListPublicRooms_OK_Anonymous(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	chatMock.EXPECT().ListPublicRooms(mock.Anything, "", false, "", uuid.Nil, 20, 0).
		Return(&dto.ChatRoomListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/public").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListPublicRooms_OK_Authenticated(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListPublicRooms(mock.Anything, "foo", true, "tag", userID, 5, 2).
		Return(&dto.ChatRoomListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/public?search=foo&rp=true&tag=tag&limit=5&offset=2").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListPublicRooms_InternalError(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	chatMock.EXPECT().ListPublicRooms(mock.Anything, "", false, "", uuid.Nil, 20, 0).
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/chat/rooms/public").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list public rooms")
}

func TestJoinRoom_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST", "/chat/rooms/"+uuid.NewString()+"/join", nil)
}

func TestJoinRoom_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().JoinRoom(mock.Anything, roomID, userID).
		Return(&dto.ChatRoomResponse{ID: roomID}, nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/join").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestJoinRoom_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/not-a-uuid/join").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestJoinRoom_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"room not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "room not found"},
		{"not group room", chatsvc.ErrNotGroupRoom, http.StatusBadRequest, "not a group room"},
		{"not public", chatsvc.ErrNotPublic, http.StatusForbidden, "not public"},
		{"room full", chatsvc.ErrRoomFull, http.StatusConflict, "room is full"},
		{"blocked", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot join this room"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to join room"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().JoinRoom(mock.Anything, roomID, userID).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/join").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestLeaveRoom_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST", "/chat/rooms/"+uuid.NewString()+"/leave", nil)
}

func TestLeaveRoom_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().LeaveRoom(mock.Anything, roomID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/leave").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestLeaveRoom_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/not-a-uuid/leave").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestLeaveRoom_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"cannot leave as host", chatsvc.ErrCannotLeaveAsHost, http.StatusForbidden, "host cannot leave"},
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to leave room"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().LeaveRoom(mock.Anything, roomID, userID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/leave").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetRoomMembers_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET", "/chat/rooms/"+uuid.NewString()+"/members", nil)
}

func TestGetRoomMembers_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GetMembers(mock.Anything, userID, roomID).
		Return([]dto.ChatRoomMemberResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/"+roomID.String()+"/members").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetRoomMembers_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("GET", "/chat/rooms/not-a-uuid/members").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestGetRoomMembers_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to get members"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().GetMembers(mock.Anything, userID, roomID).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("GET", "/chat/rooms/"+roomID.String()+"/members").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestKickMember_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE",
		"/chat/rooms/"+uuid.NewString()+"/members/"+uuid.NewString(), nil)
}

func TestKickMember_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().KickMember(mock.Anything, userID, roomID, targetID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestKickMember_InvalidRoomID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/rooms/not-a-uuid/members/"+uuid.NewString()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestKickMember_InvalidUserID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/rooms/"+uuid.NewString()+"/members/not-a-uuid").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid userID")
}

func TestKickMember_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host can kick"},
		{"cannot kick host", chatsvc.ErrCannotKickHost, http.StatusBadRequest, "cannot kick the host"},
		{"not member", chatsvc.ErrNotMember, http.StatusNotFound, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to kick member"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().KickMember(mock.Anything, userID, roomID, targetID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()).
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestSetRoomMute_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "PUT", "/chat/rooms/"+uuid.NewString()+"/mute",
		map[string]bool{"muted": true})
}

func TestSetRoomMute_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().SetRoomMuted(mock.Anything, roomID, userID, true).Return(nil)

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/"+roomID.String()+"/mute").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"muted": true}).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]bool](t, body)
	assert.True(t, got["muted"])
}

func TestSetRoomMute_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/not-a-uuid/mute").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"muted": true}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestSetRoomMute_BadJSON(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/"+uuid.NewString()+"/mute").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request")
}

func TestSetRoomMute_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to set mute"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().SetRoomMuted(mock.Anything, roomID, userID, false).Return(tc.err)

			// when
			status, body := h.NewRequest("PUT", "/chat/rooms/"+roomID.String()+"/mute").
				WithCookie("valid-cookie").
				WithJSONBody(map[string]bool{"muted": false}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetMessages_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET", "/chat/rooms/"+uuid.NewString()+"/messages", nil)
}

func TestGetMessages_OK_Default(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GetMessages(mock.Anything, userID, roomID, 50, 0).
		Return(&dto.ChatMessageListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/"+roomID.String()+"/messages").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetMessages_OK_WithPaging(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GetMessages(mock.Anything, userID, roomID, 10, 20).
		Return(&dto.ChatMessageListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/"+roomID.String()+"/messages?limit=10&offset=20").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetMessages_OK_BeforeCursor(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GetMessagesBefore(mock.Anything, userID, roomID, "2024-01-01T00:00:00Z", 50).
		Return(&dto.ChatMessageListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/"+roomID.String()+"/messages?before=2024-01-01T00:00:00Z").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetMessages_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("GET", "/chat/rooms/not-a-uuid/messages").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestGetMessages_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to get messages"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().GetMessages(mock.Anything, userID, roomID, 50, 0).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("GET", "/chat/rooms/"+roomID.String()+"/messages").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestSendMessage_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST", "/chat/rooms/"+uuid.NewString()+"/messages",
		dto.SendMessageRequest{Body: "hi"})
}

func TestSendMessage_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.SendMessageRequest{Body: "hi"}
	chatMock.EXPECT().SendMessage(mock.Anything, userID, roomID, req).
		Return(&dto.ChatMessageResponse{ID: uuid.New()}, nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/messages").
		WithCookie("valid-cookie").
		WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestSendMessage_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/not-a-uuid/messages").
		WithCookie("valid-cookie").
		WithJSONBody(dto.SendMessageRequest{Body: "hi"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestSendMessage_BadJSON(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/"+uuid.NewString()+"/messages").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestSendMessage_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"blocked", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot message"},
		{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "message body is required"},
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to send message"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.SendMessageRequest{Body: "hi"}
			chatMock.EXPECT().SendMessage(mock.Anything, userID, roomID, req).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/messages").
				WithCookie("valid-cookie").
				WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeleteChat_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE", "/chat/rooms/"+uuid.NewString(), nil)
}

func TestDeleteChat_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().DeleteChat(mock.Anything, roomID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestDeleteChat_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/rooms/not-a-uuid").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestDeleteChat_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to delete chat"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().DeleteChat(mock.Anything, roomID, userID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()).
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestChatUnreadCount_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET", "/chat/unread-count", nil)
}

func TestChatUnreadCount_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GetUnreadCount(mock.Anything, userID).Return(7, nil)

	// when
	status, body := h.NewRequest("GET", "/chat/unread-count").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]int](t, body)
	assert.Equal(t, 7, got["count"])
}

func TestChatUnreadCount_InternalError(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().GetUnreadCount(mock.Anything, userID).Return(0, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/chat/unread-count").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get unread count")
}

func TestMarkRoomRead_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST", "/chat/rooms/"+uuid.NewString()+"/read", nil)
}

func TestMarkRoomRead_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().MarkRead(mock.Anything, roomID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/read").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestMarkRoomRead_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/not-a-uuid/read").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestMarkRoomRead_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to mark room read"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().MarkRead(mock.Anything, roomID, userID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/read").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUploadChatMessageMedia_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST",
		"/chat/messages/"+uuid.NewString()+"/media", nil)
}

func TestUploadChatMessageMedia_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/messages/not-a-uuid/media").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid messageID")
}

func TestUploadChatMessageMedia_MissingFile(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/chat/messages/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").
		WithRawBody("", "multipart/form-data; boundary=----xxx").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}
