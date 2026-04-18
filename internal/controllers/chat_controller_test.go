package controllers

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"

	chatsvc "umineko_city_of_books/internal/chat"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func buildAvatarMultipart(t *testing.T, fieldName, fileName, contentType, content string) (string, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if fieldName != "" {
		header := make(map[string][]string)
		header["Content-Disposition"] = []string{`form-data; name="` + fieldName + `"; filename="` + fileName + `"`}
		if contentType != "" {
			header["Content-Type"] = []string{contentType}
		}
		part, err := w.CreatePart(header)
		require.NoError(t, err)
		_, err = io.WriteString(part, content)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.String(), w.FormDataContentType()
}

func buildSendMessageMultipart(t *testing.T, body string) (string, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("body", body))
	require.NoError(t, w.Close())
	return buf.String(), w.FormDataContentType()
}

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
	chatMock.EXPECT().SendDMMessage(mock.Anything, userID, recipientID, "hi", []chatsvc.FileUpload(nil)).
		Return(&dto.SendDMResponse{}, nil)

	// when
	rawBody, contentType := buildSendMessageMultipart(t, "hi")
	status, _ := h.NewRequest("POST", "/chat/dm/"+recipientID.String()+"/messages").
		WithCookie("valid-cookie").
		WithRawBody(rawBody, contentType).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestSendFirstDM_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	rawBody, contentType := buildSendMessageMultipart(t, "hi")
	status, body := h.NewRequest("POST", "/chat/dm/not-a-uuid/messages").
		WithCookie("valid-cookie").
		WithRawBody(rawBody, contentType).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid userID")
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
			chatMock.EXPECT().SendDMMessage(mock.Anything, userID, recipientID, "hi", []chatsvc.FileUpload(nil)).Return(nil, tc.err)

			// when
			rawBody, contentType := buildSendMessageMultipart(t, "hi")
			status, body := h.NewRequest("POST", "/chat/dm/"+recipientID.String()+"/messages").
				WithCookie("valid-cookie").
				WithRawBody(rawBody, contentType).Do()

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
	chatMock.EXPECT().JoinRoom(mock.Anything, roomID, userID, false).
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
			chatMock.EXPECT().JoinRoom(mock.Anything, roomID, userID, false).Return(nil, tc.err)

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

func TestInviteMembers_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST",
		"/chat/rooms/"+uuid.NewString()+"/members",
		dto.InviteMembersRequest{UserIDs: []uuid.UUID{uuid.New()}})
}

func TestInviteMembers_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.InviteMembersRequest{UserIDs: []uuid.UUID{targetID}}
	chatMock.EXPECT().InviteMembers(mock.Anything, userID, roomID, req.UserIDs).
		Return(&dto.InviteMembersResponse{InvitedCount: 1, SkippedCount: 0}, nil)

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/members").
		WithCookie("valid-cookie").
		WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), `"invited_count":1`)
}

func TestInviteMembers_InvalidRoomID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/not-a-uuid/members").
		WithCookie("valid-cookie").
		WithJSONBody(dto.InviteMembersRequest{UserIDs: []uuid.UUID{uuid.New()}}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestInviteMembers_EmptyUserIDs(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/"+uuid.NewString()+"/members").
		WithCookie("valid-cookie").
		WithJSONBody(dto.InviteMembersRequest{UserIDs: []uuid.UUID{}}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "user_ids is required")
}

func TestInviteMembers_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"room not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "room not found"},
		{"not group room", chatsvc.ErrNotGroupRoom, http.StatusBadRequest, "only group rooms"},
		{"system room", chatsvc.ErrSystemRoom, http.StatusForbidden, "managed automatically"},
		{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host can invite"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to invite members"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.InviteMembersRequest{UserIDs: []uuid.UUID{targetID}}
			chatMock.EXPECT().InviteMembers(mock.Anything, userID, roomID, req.UserIDs).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/members").
				WithCookie("valid-cookie").
				WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestSetMemberTimeout_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "PUT",
		"/chat/rooms/"+uuid.NewString()+"/members/"+uuid.NewString()+"/timeout",
		dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"})
}

func TestSetMemberTimeout_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", actorID)
	req := dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"}
	chatMock.EXPECT().SetMemberTimeout(mock.Anything, roomID, actorID, targetID, req).Return(&dto.ChatRoomMemberResponse{}, nil)

	// when
	status, _ := h.NewRequest("PUT", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()+"/timeout").
		WithCookie("valid-cookie").
		WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestSetMemberTimeout_InvalidRoomID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/not-a-uuid/members/"+uuid.NewString()+"/timeout").
		WithCookie("valid-cookie").
		WithJSONBody(dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestSetMemberTimeout_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only room hosts and site moderators"},
		{"not member", chatsvc.ErrNotMember, http.StatusNotFound, "user is not a member"},
		{"cannot timeout host", chatsvc.ErrCannotKickHost, http.StatusBadRequest, "cannot timeout the host"},
		{"target immune", chatsvc.ErrTargetImmune, http.StatusForbidden, "cannot be timed out"},
		{"invalid duration", chatsvc.ErrInvalidTimeoutDuration, http.StatusBadRequest, "invalid timeout duration"},
		{"locked by staff", chatsvc.ErrTimeoutLockedByStaff, http.StatusForbidden, "can only be changed by site moderators"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to set timeout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			actorID := uuid.New()
			roomID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", actorID)
			req := dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"}
			chatMock.EXPECT().SetMemberTimeout(mock.Anything, roomID, actorID, targetID, req).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("PUT", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()+"/timeout").
				WithCookie("valid-cookie").
				WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestClearMemberTimeout_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE",
		"/chat/rooms/"+uuid.NewString()+"/members/"+uuid.NewString()+"/timeout", nil)
}

func TestClearMemberTimeout_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", actorID)
	chatMock.EXPECT().ClearMemberTimeout(mock.Anything, roomID, actorID, targetID).Return(&dto.ChatRoomMemberResponse{}, nil)

	// when
	status, _ := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()+"/timeout").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestClearMemberTimeout_InvalidUserID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/rooms/"+uuid.NewString()+"/members/not-a-uuid/timeout").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid userID")
}

func TestClearMemberTimeout_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only room hosts and site moderators"},
		{"not member", chatsvc.ErrNotMember, http.StatusNotFound, "user is not a member"},
		{"locked by staff", chatsvc.ErrTimeoutLockedByStaff, http.StatusForbidden, "can only be removed by site moderators"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to clear timeout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			actorID := uuid.New()
			roomID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", actorID)
			chatMock.EXPECT().ClearMemberTimeout(mock.Anything, roomID, actorID, targetID).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()+"/timeout").
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
	chatMock.EXPECT().SendMessage(mock.Anything, userID, roomID, req, []chatsvc.FileUpload(nil)).
		Return(&dto.ChatMessageResponse{ID: uuid.New()}, nil)

	// when
	rawBody, contentType := buildSendMessageMultipart(t, "hi")
	status, _ := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/messages").
		WithCookie("valid-cookie").
		WithRawBody(rawBody, contentType).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestSendMessage_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	rawBody, contentType := buildSendMessageMultipart(t, "hi")
	status, body := h.NewRequest("POST", "/chat/rooms/not-a-uuid/messages").
		WithCookie("valid-cookie").
		WithRawBody(rawBody, contentType).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestSendMessage_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"blocked", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot message"},
		{"timed out", chatsvc.ErrTimedOut, http.StatusForbidden, chatsvc.ErrTimedOut.Error()},
		{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "message body is required"},
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"invalid file type", upload.ErrInvalidFileType, http.StatusBadRequest, upload.ErrInvalidFileType.Error()},
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
			chatMock.EXPECT().SendMessage(mock.Anything, userID, roomID, req, []chatsvc.FileUpload(nil)).Return(nil, tc.err)

			// when
			rawBody, contentType := buildSendMessageMultipart(t, "hi")
			status, body := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/messages").
				WithCookie("valid-cookie").
				WithRawBody(rawBody, contentType).Do()

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

func TestSetRoomNickname_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "PUT", "/chat/rooms/"+uuid.NewString()+"/me",
		dto.UpdateMemberProfileRequest{Nickname: "nick"})
}

func TestSetRoomNickname_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().SetRoomNickname(mock.Anything, roomID, userID, "nick").Return(&dto.ChatRoomMemberResponse{
		User:     dto.UserResponse{ID: userID},
		Nickname: "nick",
	}, nil)

	// when
	status, respBody := h.NewRequest("PUT", "/chat/rooms/"+roomID.String()+"/me").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateMemberProfileRequest{Nickname: "nick"}).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.ChatRoomMemberResponse](t, respBody)
	assert.Equal(t, "nick", got.Nickname)
}

func TestSetRoomNickname_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/not-a-uuid/me").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateMemberProfileRequest{Nickname: "nick"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestSetRoomNickname_BadJSON(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/"+uuid.NewString()+"/me").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request")
}

func TestSetRoomNickname_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to update nickname"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().SetRoomNickname(mock.Anything, roomID, userID, "nick").Return(nil, tc.err)

			// when
			status, body := h.NewRequest("PUT", "/chat/rooms/"+roomID.String()+"/me").
				WithCookie("valid-cookie").
				WithJSONBody(dto.UpdateMemberProfileRequest{Nickname: "nick"}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestSetRoomAvatar_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST",
		"/chat/rooms/"+uuid.NewString()+"/me/avatar", nil)
}

func TestSetRoomAvatar_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	body, ct := buildAvatarMultipart(t, "avatar", "avatar.png", "image/png", "pngdata")
	chatMock.EXPECT().SetRoomAvatar(mock.Anything, roomID, userID, "image/png", int64(len("pngdata")), mock.Anything).
		Return(&dto.ChatRoomMemberResponse{
			User:            dto.UserResponse{ID: userID},
			MemberAvatarURL: "https://cdn/avatar.png",
		}, nil)

	// when
	status, respBody := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/me/avatar").
		WithCookie("valid-cookie").
		WithRawBody(body, ct).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.ChatRoomMemberResponse](t, respBody)
	assert.Equal(t, "https://cdn/avatar.png", got.MemberAvatarURL)
}

func TestSetRoomAvatar_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/not-a-uuid/me/avatar").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestSetRoomAvatar_MissingFile(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/rooms/"+uuid.NewString()+"/me/avatar").
		WithCookie("valid-cookie").
		WithRawBody("", "multipart/form-data; boundary=----xxx").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "avatar file is required")
}

func TestSetRoomAvatar_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"file too large", upload.ErrFileTooLarge, http.StatusBadRequest, "50MB"},
		{"invalid file type", upload.ErrInvalidFileType, http.StatusBadRequest, "PNG"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to upload avatar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			body, ct := buildAvatarMultipart(t, "avatar", "avatar.png", "image/png", "pngdata")
			chatMock.EXPECT().SetRoomAvatar(mock.Anything, roomID, userID, "image/png", int64(len("pngdata")), mock.Anything).
				Return(nil, tc.err)

			// when
			status, respBody := h.NewRequest("POST", "/chat/rooms/"+roomID.String()+"/me/avatar").
				WithCookie("valid-cookie").
				WithRawBody(body, ct).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(respBody), tc.wantBody)
		})
	}
}

func TestClearRoomAvatar_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE",
		"/chat/rooms/"+uuid.NewString()+"/me/avatar", nil)
}

func TestClearRoomAvatar_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ClearRoomAvatar(mock.Anything, roomID, userID).Return(&dto.ChatRoomMemberResponse{
		User:            dto.UserResponse{ID: userID},
		MemberAvatarURL: "",
	}, nil)

	// when
	status, respBody := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()+"/me/avatar").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.ChatRoomMemberResponse](t, respBody)
	assert.Equal(t, "", got.MemberAvatarURL)
}

func TestClearRoomAvatar_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/rooms/not-a-uuid/me/avatar").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestClearRoomAvatar_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to clear avatar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().ClearRoomAvatar(mock.Anything, roomID, userID).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()+"/me/avatar").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestPinMessage_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST",
		"/chat/messages/"+uuid.NewString()+"/pin", nil)
}

func TestPinMessage_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	messageID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().PinMessage(mock.Anything, messageID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/messages/"+messageID.String()+"/pin").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestPinMessage_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/messages/not-a-uuid/pin").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid messageID")
}

func TestPinMessage_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host can pin"},
		{"message not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to pin message"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			messageID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().PinMessage(mock.Anything, messageID, userID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/messages/"+messageID.String()+"/pin").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnpinMessage_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE",
		"/chat/messages/"+uuid.NewString()+"/pin", nil)
}

func TestUnpinMessage_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	messageID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().UnpinMessage(mock.Anything, messageID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/chat/messages/"+messageID.String()+"/pin").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestUnpinMessage_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/messages/not-a-uuid/pin").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid messageID")
}

func TestUnpinMessage_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host can unpin"},
		{"not pinned", chatsvc.ErrMessageNotPinned, http.StatusBadRequest, "not pinned"},
		{"message not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to unpin message"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			messageID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().UnpinMessage(mock.Anything, messageID, userID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/chat/messages/"+messageID.String()+"/pin").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListPinnedMessages_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "GET",
		"/chat/rooms/"+uuid.NewString()+"/pins", nil)
}

func TestListPinnedMessages_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	roomID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().ListPinnedMessages(mock.Anything, roomID, userID).
		Return(&dto.ChatMessageListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/chat/rooms/"+roomID.String()+"/pins").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListPinnedMessages_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("GET", "/chat/rooms/not-a-uuid/pins").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestListPinnedMessages_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to list pinned messages"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			roomID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().ListPinnedMessages(mock.Anything, roomID, userID).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("GET", "/chat/rooms/"+roomID.String()+"/pins").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestAddReaction_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "POST",
		"/chat/messages/"+uuid.NewString()+"/reactions",
		dto.AddReactionRequest{Emoji: "heart"})
}

func TestAddReaction_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	messageID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().AddReaction(mock.Anything, messageID, userID, "heart").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/chat/messages/"+messageID.String()+"/reactions").
		WithCookie("valid-cookie").
		WithJSONBody(dto.AddReactionRequest{Emoji: "heart"}).Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAddReaction_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/messages/not-a-uuid/reactions").
		WithCookie("valid-cookie").
		WithJSONBody(dto.AddReactionRequest{Emoji: "heart"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid messageID")
}

func TestAddReaction_BadJSON(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/chat/messages/"+uuid.NewString()+"/reactions").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request")
}

func TestAddReaction_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"invalid emoji", chatsvc.ErrInvalidEmoji, http.StatusBadRequest, "invalid emoji"},
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"message not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to add reaction"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			messageID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().AddReaction(mock.Anything, messageID, userID, "heart").Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/chat/messages/"+messageID.String()+"/reactions").
				WithCookie("valid-cookie").
				WithJSONBody(dto.AddReactionRequest{Emoji: "heart"}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestRemoveReaction_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE",
		"/chat/messages/"+uuid.NewString()+"/reactions/heart", nil)
}

func TestRemoveReaction_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	messageID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().RemoveReaction(mock.Anything, messageID, userID, "heart").Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/chat/messages/"+messageID.String()+"/reactions/heart").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestRemoveReaction_URLEncodedEmoji(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	messageID := uuid.New()
	emoji := "heart eyes"
	encoded := url.PathEscape(emoji)
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().RemoveReaction(mock.Anything, messageID, userID, emoji).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/chat/messages/"+messageID.String()+"/reactions/"+encoded).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestRemoveReaction_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/messages/not-a-uuid/reactions/heart").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid messageID")
}

func TestRemoveReaction_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"invalid emoji", chatsvc.ErrInvalidEmoji, http.StatusBadRequest, "invalid emoji"},
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"message not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to remove reaction"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			messageID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().RemoveReaction(mock.Anything, messageID, userID, "heart").Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/chat/messages/"+messageID.String()+"/reactions/heart").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestSetMemberNicknameAsMod_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "PUT",
		"/chat/rooms/"+uuid.NewString()+"/members/"+uuid.NewString()+"/nickname",
		dto.UpdateMemberProfileRequest{Nickname: "x"})
}

func TestSetMemberNicknameAsMod_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", actorID)
	member := &dto.ChatRoomMemberResponse{
		User:           dto.UserResponse{ID: targetID, Username: "target"},
		Role:           "member",
		Nickname:       "Forced",
		NicknameLocked: true,
	}
	chatMock.EXPECT().SetMemberNicknameAsMod(mock.Anything, roomID, actorID, targetID, "Forced").Return(member, nil)

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()+"/nickname").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateMemberProfileRequest{Nickname: "Forced"}).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "Forced")
	assert.Contains(t, string(body), targetID.String())
}

func TestSetMemberNicknameAsMod_InvalidRoomID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/not-a-uuid/members/"+uuid.NewString()+"/nickname").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateMemberProfileRequest{Nickname: "x"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestSetMemberNicknameAsMod_InvalidUserID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/"+uuid.NewString()+"/members/not-a-uuid/nickname").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateMemberProfileRequest{Nickname: "x"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid userID")
}

func TestSetMemberNicknameAsMod_BadJSON(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/chat/rooms/"+uuid.NewString()+"/members/"+uuid.NewString()+"/nickname").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request")
}

func TestSetMemberNicknameAsMod_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"not mod", chatsvc.ErrModRoleRequired, http.StatusForbidden, "only site moderators"},
		{"target immune", chatsvc.ErrTargetImmune, http.StatusForbidden, "cannot be changed by moderators"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to set member nickname"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			actorID := uuid.New()
			roomID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", actorID)
			chatMock.EXPECT().SetMemberNicknameAsMod(mock.Anything, roomID, actorID, targetID, "x").Return(nil, tc.err)

			// when
			status, body := h.NewRequest("PUT", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()+"/nickname").
				WithCookie("valid-cookie").
				WithJSONBody(dto.UpdateMemberProfileRequest{Nickname: "x"}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnlockMemberNickname_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE",
		"/chat/rooms/"+uuid.NewString()+"/members/"+uuid.NewString()+"/nickname", nil)
}

func TestUnlockMemberNickname_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	actorID := uuid.New()
	roomID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", actorID)
	member := &dto.ChatRoomMemberResponse{
		User:           dto.UserResponse{ID: targetID, Username: "target"},
		Role:           "member",
		NicknameLocked: false,
	}
	chatMock.EXPECT().UnlockMemberNickname(mock.Anything, roomID, actorID, targetID).Return(member, nil)

	// when
	status, body := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()+"/nickname").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), targetID.String())
}

func TestUnlockMemberNickname_InvalidRoomID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/rooms/not-a-uuid/members/"+uuid.NewString()+"/nickname").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid roomID")
}

func TestUnlockMemberNickname_InvalidUserID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/rooms/"+uuid.NewString()+"/members/not-a-uuid/nickname").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid userID")
}

func TestUnlockMemberNickname_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
		{"not mod", chatsvc.ErrModRoleRequired, http.StatusForbidden, "only site moderators"},
		{"target immune", chatsvc.ErrTargetImmune, http.StatusForbidden, "not affected by nickname locks"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to unlock nickname"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			actorID := uuid.New()
			roomID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", actorID)
			chatMock.EXPECT().UnlockMemberNickname(mock.Anything, roomID, actorID, targetID).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/chat/rooms/"+roomID.String()+"/members/"+targetID.String()+"/nickname").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeleteMessage_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "DELETE",
		"/chat/messages/"+uuid.NewString(), nil)
}

func TestDeleteMessage_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	messageID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	chatMock.EXPECT().DeleteMessage(mock.Anything, messageID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/chat/messages/"+messageID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestDeleteMessage_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/chat/messages/not-a-uuid").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid messageID")
}

func TestDeleteMessage_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
		{"permission", chatsvc.ErrMessageDeletePermission, http.StatusForbidden, "do not have permission"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to delete message"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			messageID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().DeleteMessage(mock.Anything, messageID, userID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/chat/messages/"+messageID.String()).
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestEditMessage_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, chatFactory, "PATCH",
		"/chat/messages/"+uuid.NewString(), dto.EditMessageRequest{Body: "hi"})
}

func TestEditMessage_OK(t *testing.T) {
	// given
	h, chatMock := newChatHarness(t)
	userID := uuid.New()
	messageID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	editedAt := "2026-04-18T20:00:00Z"
	chatMock.EXPECT().EditMessage(mock.Anything, messageID, userID, "updated").
		Return(&dto.ChatMessageResponse{ID: messageID, Body: "updated", EditedAt: &editedAt}, nil)

	// when
	status, body := h.NewRequest("PATCH", "/chat/messages/"+messageID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.EditMessageRequest{Body: "updated"}).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), `"body":"updated"`)
	assert.Contains(t, string(body), `"edited_at"`)
}

func TestEditMessage_InvalidID(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PATCH", "/chat/messages/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.EditMessageRequest{Body: "hi"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid messageID")
}

func TestEditMessage_BadJSON(t *testing.T) {
	// given
	h, _ := newChatHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PATCH", "/chat/messages/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestEditMessage_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
		{"permission", chatsvc.ErrMessageEditPermission, http.StatusForbidden, "can only edit your own messages"},
		{"system", chatsvc.ErrCannotEditSystemMessage, http.StatusBadRequest, "system messages"},
		{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "body is required"},
		{"timed out", chatsvc.ErrTimedOut, http.StatusForbidden, "timed out"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to edit message"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := newChatHarness(t)
			userID := uuid.New()
			messageID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			chatMock.EXPECT().EditMessage(mock.Anything, messageID, userID, "new").Return(nil, tc.err)

			// when
			status, body := h.NewRequest("PATCH", "/chat/messages/"+messageID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(dto.EditMessageRequest{Body: "new"}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}
