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
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	chatCtlRequest struct {
		method      string
		path        string
		json        any
		raw         string
		contentType string
		anonymous   bool
	}

	chatCtlErrCase struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}
)

const (
	chatCtlCookie     = "valid-cookie"
	chatCtlAvatarData = "pngdata"
)

var (
	chatCtlUserID    = uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	chatCtlRoomID    = uuid.MustParse("c0000000-0000-0000-0000-000000000002")
	chatCtlTargetID  = uuid.MustParse("c0000000-0000-0000-0000-000000000003")
	chatCtlMessageID = uuid.MustParse("c0000000-0000-0000-0000-000000000004")

	chatCtlDMPath      = "/chat/dm/" + chatCtlTargetID.String()
	chatCtlRoomPath    = "/chat/rooms/" + chatCtlRoomID.String()
	chatCtlMemberPath  = chatCtlRoomPath + "/members/" + chatCtlTargetID.String()
	chatCtlMessagePath = "/chat/messages/" + chatCtlMessageID.String()
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

func chatCtlHarnessFor(t *testing.T, r chatCtlRequest) (*testutil.Harness, *chatsvc.MockService) {
	h, chatMock := newChatHarness(t)
	if !r.anonymous {
		h.ExpectValidSession(chatCtlCookie, chatCtlUserID)
	}

	return h, chatMock
}

func chatCtlMessageForm(t *testing.T, path string) chatCtlRequest {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("body", "hi"))
	require.NoError(t, w.Close())

	return chatCtlRequest{method: "POST", path: path, raw: buf.String(), contentType: w.FormDataContentType()}
}

func chatCtlAvatarUpload(t *testing.T) chatCtlRequest {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="avatar"; filename="avatar.png"`},
		"Content-Type":        {"image/png"},
	})
	require.NoError(t, err)

	_, err = io.WriteString(part, chatCtlAvatarData)
	require.NoError(t, err)

	require.NoError(t, w.Close())

	return chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/me/avatar", raw: buf.String(), contentType: w.FormDataContentType()}
}

func (r chatCtlRequest) do(h *testutil.Harness) (int, []byte) {
	req := h.NewRequest(r.method, r.path)
	if !r.anonymous {
		req = req.WithCookie(chatCtlCookie)
	}
	if r.json != nil {
		req = req.WithJSONBody(r.json)
	}
	if r.contentType != "" {
		req = req.WithRawBody(r.raw, r.contentType)
	}

	return req.Do()
}

func TestChatRoutes_AuthFailures(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"resolve dm", "GET", chatCtlDMPath + "/resolve", nil},
		{"send first dm", "POST", chatCtlDMPath + "/messages", dto.SendMessageRequest{Body: "hi"}},
		{"create group room", "POST", "/chat/rooms", dto.CreateGroupRoomRequest{Name: "room"}},
		{"update room", "PUT", chatCtlRoomPath, dto.UpdateGroupRoomRequest{Name: "room"}},
		{"list rooms", "GET", "/chat/rooms", nil},
		{"list my group rooms", "GET", "/chat/rooms/mine", nil},
		{"join room", "POST", chatCtlRoomPath + "/join", nil},
		{"leave room", "POST", chatCtlRoomPath + "/leave", nil},
		{"get room members", "GET", chatCtlRoomPath + "/members", nil},
		{"kick member", "DELETE", chatCtlMemberPath, nil},
		{"invite members", "POST", chatCtlRoomPath + "/members", dto.InviteMembersRequest{UserIDs: []uuid.UUID{chatCtlTargetID}}},
		{"set member timeout", "PUT", chatCtlMemberPath + "/timeout", dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"}},
		{"clear member timeout", "DELETE", chatCtlMemberPath + "/timeout", nil},
		{"set room mute", "PUT", chatCtlRoomPath + "/mute", map[string]bool{"muted": true}},
		{"get messages", "GET", chatCtlRoomPath + "/messages", nil},
		{"send message", "POST", chatCtlRoomPath + "/messages", dto.SendMessageRequest{Body: "hi"}},
		{"delete chat", "DELETE", chatCtlRoomPath, nil},
		{"unread count", "GET", "/chat/unread-count", nil},
		{"mark room read", "POST", chatCtlRoomPath + "/read", nil},
		{"set room nickname", "PUT", chatCtlRoomPath + "/me", dto.UpdateMemberProfileRequest{Nickname: "nick"}},
		{"set room avatar", "POST", chatCtlRoomPath + "/me/avatar", nil},
		{"clear room avatar", "DELETE", chatCtlRoomPath + "/me/avatar", nil},
		{"pin message", "POST", chatCtlMessagePath + "/pin", nil},
		{"unpin message", "DELETE", chatCtlMessagePath + "/pin", nil},
		{"list pinned messages", "GET", chatCtlRoomPath + "/pins", nil},
		{"add reaction", "POST", chatCtlMessagePath + "/reactions", dto.AddReactionRequest{Emoji: "heart"}},
		{"remove reaction", "DELETE", chatCtlMessagePath + "/reactions/heart", nil},
		{"set member nickname as mod", "PUT", chatCtlMemberPath + "/nickname", dto.UpdateMemberProfileRequest{Nickname: "x"}},
		{"unlock member nickname", "DELETE", chatCtlMemberPath + "/nickname", nil},
		{"delete message", "DELETE", chatCtlMessagePath, nil},
		{"edit message", "PATCH", chatCtlMessagePath, dto.EditMessageRequest{Body: "hi"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.RunAuthFailureSuite(t, newChatHarness, tc.method, tc.path, tc.body)
		})
	}
}

func TestChatRoutes_RejectMalformedInput(t *testing.T) {
	cases := []struct {
		name     string
		req      chatCtlRequest
		wantBody string
	}{
		{"resolve dm with a bad user id", chatCtlRequest{method: "GET", path: "/chat/dm/not-a-uuid/resolve"}, "invalid userID"},
		{"send first dm with a bad user id", chatCtlMessageForm(t, "/chat/dm/not-a-uuid/messages"), "invalid userID"},
		{"create group room with bad json", chatCtlRequest{method: "POST", path: "/chat/rooms", raw: "not json", contentType: "application/json"}, "invalid request body"},
		{"update room with a bad room id", chatCtlRequest{method: "PUT", path: "/chat/rooms/not-a-uuid", json: dto.UpdateGroupRoomRequest{Name: "room"}}, "invalid roomID"},
		{"update room with bad json", chatCtlRequest{method: "PUT", path: chatCtlRoomPath, raw: "not json", contentType: "application/json"}, "invalid request body"},
		{"join room with a bad room id", chatCtlRequest{method: "POST", path: "/chat/rooms/not-a-uuid/join"}, "invalid roomID"},
		{"leave room with a bad room id", chatCtlRequest{method: "POST", path: "/chat/rooms/not-a-uuid/leave"}, "invalid roomID"},
		{"get room members with a bad room id", chatCtlRequest{method: "GET", path: "/chat/rooms/not-a-uuid/members"}, "invalid roomID"},
		{"kick member with a bad room id", chatCtlRequest{method: "DELETE", path: "/chat/rooms/not-a-uuid/members/" + chatCtlTargetID.String()}, "invalid roomID"},
		{"kick member with a bad user id", chatCtlRequest{method: "DELETE", path: chatCtlRoomPath + "/members/not-a-uuid"}, "invalid userID"},
		{"invite members with a bad room id", chatCtlRequest{method: "POST", path: "/chat/rooms/not-a-uuid/members", json: dto.InviteMembersRequest{UserIDs: []uuid.UUID{chatCtlTargetID}}}, "invalid roomID"},
		{"invite members with no user ids", chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/members", json: dto.InviteMembersRequest{UserIDs: []uuid.UUID{}}}, "user_ids is required"},
		{"set member timeout with a bad room id", chatCtlRequest{method: "PUT", path: "/chat/rooms/not-a-uuid/members/" + chatCtlTargetID.String() + "/timeout", json: dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"}}, "invalid roomID"},
		{"clear member timeout with a bad user id", chatCtlRequest{method: "DELETE", path: chatCtlRoomPath + "/members/not-a-uuid/timeout"}, "invalid userID"},
		{"set room mute with a bad room id", chatCtlRequest{method: "PUT", path: "/chat/rooms/not-a-uuid/mute", json: map[string]bool{"muted": true}}, "invalid roomID"},
		{"set room mute with bad json", chatCtlRequest{method: "PUT", path: chatCtlRoomPath + "/mute", raw: "not json", contentType: "application/json"}, "invalid request"},
		{"get messages with a bad room id", chatCtlRequest{method: "GET", path: "/chat/rooms/not-a-uuid/messages"}, "invalid roomID"},
		{"send message with a bad room id", chatCtlMessageForm(t, "/chat/rooms/not-a-uuid/messages"), "invalid roomID"},
		{"send message with a body that is not a form", chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/messages", json: dto.SendMessageRequest{Body: "hi"}}, "invalid form data"},
		{"send first dm with a body that is not a form", chatCtlRequest{method: "POST", path: chatCtlDMPath + "/messages", json: dto.SendMessageRequest{Body: "hi"}}, "invalid form data"},
		{"delete chat with a bad room id", chatCtlRequest{method: "DELETE", path: "/chat/rooms/not-a-uuid"}, "invalid roomID"},
		{"mark room read with a bad room id", chatCtlRequest{method: "POST", path: "/chat/rooms/not-a-uuid/read"}, "invalid roomID"},
		{"set room nickname with a bad room id", chatCtlRequest{method: "PUT", path: "/chat/rooms/not-a-uuid/me", json: dto.UpdateMemberProfileRequest{Nickname: "nick"}}, "invalid roomID"},
		{"set room nickname with bad json", chatCtlRequest{method: "PUT", path: chatCtlRoomPath + "/me", raw: "not json", contentType: "application/json"}, "invalid request"},
		{"set room avatar with a bad room id", chatCtlRequest{method: "POST", path: "/chat/rooms/not-a-uuid/me/avatar"}, "invalid roomID"},
		{"set room avatar with no file", chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/me/avatar", contentType: "multipart/form-data; boundary=----xxx"}, "avatar file is required"},
		{"clear room avatar with a bad room id", chatCtlRequest{method: "DELETE", path: "/chat/rooms/not-a-uuid/me/avatar"}, "invalid roomID"},
		{"pin message with a bad message id", chatCtlRequest{method: "POST", path: "/chat/messages/not-a-uuid/pin"}, "invalid messageID"},
		{"unpin message with a bad message id", chatCtlRequest{method: "DELETE", path: "/chat/messages/not-a-uuid/pin"}, "invalid messageID"},
		{"list pinned messages with a bad room id", chatCtlRequest{method: "GET", path: "/chat/rooms/not-a-uuid/pins"}, "invalid roomID"},
		{"add reaction with a bad message id", chatCtlRequest{method: "POST", path: "/chat/messages/not-a-uuid/reactions", json: dto.AddReactionRequest{Emoji: "heart"}}, "invalid messageID"},
		{"add reaction with bad json", chatCtlRequest{method: "POST", path: chatCtlMessagePath + "/reactions", raw: "not json", contentType: "application/json"}, "invalid request"},
		{"remove reaction with a bad message id", chatCtlRequest{method: "DELETE", path: "/chat/messages/not-a-uuid/reactions/heart"}, "invalid messageID"},
		{"set member nickname as mod with a bad room id", chatCtlRequest{method: "PUT", path: "/chat/rooms/not-a-uuid/members/" + chatCtlTargetID.String() + "/nickname", json: dto.UpdateMemberProfileRequest{Nickname: "x"}}, "invalid roomID"},
		{"set member nickname as mod with a bad user id", chatCtlRequest{method: "PUT", path: chatCtlRoomPath + "/members/not-a-uuid/nickname", json: dto.UpdateMemberProfileRequest{Nickname: "x"}}, "invalid userID"},
		{"set member nickname as mod with bad json", chatCtlRequest{method: "PUT", path: chatCtlMemberPath + "/nickname", raw: "not json", contentType: "application/json"}, "invalid request"},
		{"unlock member nickname with a bad room id", chatCtlRequest{method: "DELETE", path: "/chat/rooms/not-a-uuid/members/" + chatCtlTargetID.String() + "/nickname"}, "invalid roomID"},
		{"unlock member nickname with a bad user id", chatCtlRequest{method: "DELETE", path: chatCtlRoomPath + "/members/not-a-uuid/nickname"}, "invalid userID"},
		{"delete message with a bad message id", chatCtlRequest{method: "DELETE", path: "/chat/messages/not-a-uuid"}, "invalid messageID"},
		{"edit message with a bad message id", chatCtlRequest{method: "PATCH", path: "/chat/messages/not-a-uuid", json: dto.EditMessageRequest{Body: "hi"}}, "invalid messageID"},
		{"edit message with bad json", chatCtlRequest{method: "PATCH", path: chatCtlMessagePath, raw: "not json", contentType: "application/json"}, "invalid request body"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := chatCtlHarnessFor(t, tc.req)

			// when
			status, body := tc.req.do(h)

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestChatRoutes_Success(t *testing.T) {
	renamed := dto.UpdateGroupRoomRequest{Name: "new name", Description: "desc", Tags: []string{"tag"}, IsPublic: true}

	cases := []struct {
		name     string
		req      chatCtlRequest
		expect   func(m *chatsvc.MockService)
		wantCode int
		wantBody []string
		check    func(t *testing.T, body []byte)
	}{
		{
			name: "resolve dm",
			req:  chatCtlRequest{method: "GET", path: chatCtlDMPath + "/resolve"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().ResolveDMRoom(mock.Anything, chatCtlUserID, chatCtlTargetID).Return(&dto.ResolveDMResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "send first dm",
			req:  chatCtlMessageForm(t, chatCtlDMPath+"/messages"),
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().SendDMMessage(mock.Anything, chatCtlUserID, chatCtlTargetID, "hi", []chatsvc.FileUpload(nil)).Return(&dto.SendDMResponse{}, nil)
			},
			wantCode: http.StatusCreated,
		},
		{
			name: "create group room",
			req:  chatCtlRequest{method: "POST", path: "/chat/rooms", json: dto.CreateGroupRoomRequest{Name: "room"}},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().CreateGroupRoom(mock.Anything, chatCtlUserID, dto.CreateGroupRoomRequest{Name: "room"}).
					Return(&dto.ChatRoomResponse{ID: uuid.New(), Name: "room"}, nil)
			},
			wantCode: http.StatusCreated,
		},
		{
			name: "update room",
			req:  chatCtlRequest{method: "PUT", path: chatCtlRoomPath, json: renamed},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().UpdateGroupRoom(mock.Anything, chatCtlRoomID, chatCtlUserID, renamed).
					Return(&dto.ChatRoomResponse{ID: chatCtlRoomID, Name: "new name"}, nil)
			},
			wantCode: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				got := testutil.UnmarshalJSON[dto.ChatRoomResponse](t, body)
				assert.Equal(t, chatCtlRoomID, got.ID)
				assert.Equal(t, "new name", got.Name)
			},
		},
		{
			name: "list rooms",
			req:  chatCtlRequest{method: "GET", path: "/chat/rooms"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().ListRooms(mock.Anything, chatCtlUserID).Return(&dto.ChatRoomListResponse{Rooms: []dto.ChatRoomResponse{}, Total: 0}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "list my group rooms passes every filter through",
			req:  chatCtlRequest{method: "GET", path: "/chat/rooms/mine?search=foo&rp=true&tag=tag&role=admin&limit=10&offset=5"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().ListUserGroupRooms(mock.Anything, chatCtlUserID, "foo", true, "tag", "admin", false, 10, 5).Return(&dto.ChatRoomListResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "list public rooms anonymously uses the defaults",
			req:  chatCtlRequest{method: "GET", path: "/chat/rooms/public", anonymous: true},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().ListPublicRooms(mock.Anything, "", false, "", uuid.Nil, false, 20, 0).Return(&dto.ChatRoomListResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "list public rooms signed in passes the viewer and filters through",
			req:  chatCtlRequest{method: "GET", path: "/chat/rooms/public?search=foo&rp=true&tag=tag&limit=5&offset=2"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().ListPublicRooms(mock.Anything, "foo", true, "tag", chatCtlUserID, false, 5, 2).Return(&dto.ChatRoomListResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "join room",
			req:  chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/join"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().JoinRoom(mock.Anything, chatCtlRoomID, chatCtlUserID, false).Return(&dto.ChatRoomResponse{ID: chatCtlRoomID}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "leave room",
			req:  chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/leave"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().LeaveRoom(mock.Anything, chatCtlRoomID, chatCtlUserID).Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "get room members",
			req:  chatCtlRequest{method: "GET", path: chatCtlRoomPath + "/members"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().GetMembers(mock.Anything, chatCtlUserID, chatCtlRoomID).Return([]dto.ChatRoomMemberResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "kick member",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMemberPath},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().KickMember(mock.Anything, chatCtlUserID, chatCtlRoomID, chatCtlTargetID).Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "invite members",
			req:  chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/members", json: dto.InviteMembersRequest{UserIDs: []uuid.UUID{chatCtlTargetID}}},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().InviteMembers(mock.Anything, chatCtlUserID, chatCtlRoomID, []uuid.UUID{chatCtlTargetID}).
					Return(&dto.InviteMembersResponse{InvitedCount: 1, SkippedCount: 0}, nil)
			},
			wantCode: http.StatusOK,
			wantBody: []string{`"invited_count":1`},
		},
		{
			name: "set member timeout",
			req:  chatCtlRequest{method: "PUT", path: chatCtlMemberPath + "/timeout", json: dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"}},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().SetMemberTimeout(mock.Anything, chatCtlRoomID, chatCtlUserID, chatCtlTargetID, dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"}).
					Return(&dto.ChatRoomMemberResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "clear member timeout",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMemberPath + "/timeout"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().ClearMemberTimeout(mock.Anything, chatCtlRoomID, chatCtlUserID, chatCtlTargetID).Return(&dto.ChatRoomMemberResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "set room mute",
			req:  chatCtlRequest{method: "PUT", path: chatCtlRoomPath + "/mute", json: map[string]bool{"muted": true}},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().SetRoomMuted(mock.Anything, chatCtlRoomID, chatCtlUserID, true).Return(nil)
			},
			wantCode: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				got := testutil.UnmarshalJSON[map[string]bool](t, body)
				assert.True(t, got["muted"])
			},
		},
		{
			name: "get messages uses the default page",
			req:  chatCtlRequest{method: "GET", path: chatCtlRoomPath + "/messages"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().GetMessages(mock.Anything, chatCtlUserID, chatCtlRoomID, 50, 0).Return(&dto.ChatMessageListResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "get messages passes limit and offset through",
			req:  chatCtlRequest{method: "GET", path: chatCtlRoomPath + "/messages?limit=10&offset=20"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().GetMessages(mock.Anything, chatCtlUserID, chatCtlRoomID, 10, 20).Return(&dto.ChatMessageListResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "get messages with a before cursor pages backwards",
			req:  chatCtlRequest{method: "GET", path: chatCtlRoomPath + "/messages?before=2024-01-01T00:00:00Z"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().GetMessagesBefore(mock.Anything, chatCtlUserID, chatCtlRoomID, "2024-01-01T00:00:00Z", 50).Return(&dto.ChatMessageListResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "send message",
			req:  chatCtlMessageForm(t, chatCtlRoomPath+"/messages"),
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().SendMessage(mock.Anything, chatCtlUserID, chatCtlRoomID, dto.SendMessageRequest{Body: "hi"}, []chatsvc.FileUpload(nil)).
					Return(&dto.ChatMessageResponse{ID: uuid.New()}, nil)
			},
			wantCode: http.StatusCreated,
		},
		{
			name: "delete chat",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlRoomPath},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().DeleteChat(mock.Anything, chatCtlRoomID, chatCtlUserID).Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "unread count",
			req:  chatCtlRequest{method: "GET", path: "/chat/unread-count"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().GetUnreadCount(mock.Anything, chatCtlUserID).Return(7, nil)
			},
			wantCode: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				got := testutil.UnmarshalJSON[map[string]int](t, body)
				assert.Equal(t, 7, got["count"])
			},
		},
		{
			name: "mark room read",
			req:  chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/read"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().MarkRead(mock.Anything, chatCtlRoomID, chatCtlUserID).Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "set room nickname",
			req:  chatCtlRequest{method: "PUT", path: chatCtlRoomPath + "/me", json: dto.UpdateMemberProfileRequest{Nickname: "nick"}},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().SetRoomNickname(mock.Anything, chatCtlRoomID, chatCtlUserID, "nick").
					Return(&dto.ChatRoomMemberResponse{User: dto.UserResponse{ID: chatCtlUserID}, Nickname: "nick"}, nil)
			},
			wantCode: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				got := testutil.UnmarshalJSON[dto.ChatRoomMemberResponse](t, body)
				assert.Equal(t, "nick", got.Nickname)
			},
		},
		{
			name: "set room avatar",
			req:  chatCtlAvatarUpload(t),
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().SetRoomAvatar(mock.Anything, chatCtlRoomID, chatCtlUserID, "image/png", int64(len(chatCtlAvatarData)), mock.Anything).
					Return(&dto.ChatRoomMemberResponse{User: dto.UserResponse{ID: chatCtlUserID}, MemberAvatarURL: "https://cdn/avatar.png"}, nil)
			},
			wantCode: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				got := testutil.UnmarshalJSON[dto.ChatRoomMemberResponse](t, body)
				assert.Equal(t, "https://cdn/avatar.png", got.MemberAvatarURL)
			},
		},
		{
			name: "clear room avatar",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlRoomPath + "/me/avatar"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().ClearRoomAvatar(mock.Anything, chatCtlRoomID, chatCtlUserID).
					Return(&dto.ChatRoomMemberResponse{User: dto.UserResponse{ID: chatCtlUserID}, MemberAvatarURL: ""}, nil)
			},
			wantCode: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				got := testutil.UnmarshalJSON[dto.ChatRoomMemberResponse](t, body)
				assert.Equal(t, "", got.MemberAvatarURL)
			},
		},
		{
			name: "pin message",
			req:  chatCtlRequest{method: "POST", path: chatCtlMessagePath + "/pin"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().PinMessage(mock.Anything, chatCtlMessageID, chatCtlUserID).Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "unpin message",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMessagePath + "/pin"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().UnpinMessage(mock.Anything, chatCtlMessageID, chatCtlUserID).Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "list pinned messages",
			req:  chatCtlRequest{method: "GET", path: chatCtlRoomPath + "/pins"},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().ListPinnedMessages(mock.Anything, chatCtlRoomID, chatCtlUserID).Return(&dto.ChatMessageListResponse{}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "add reaction",
			req:  chatCtlRequest{method: "POST", path: chatCtlMessagePath + "/reactions", json: dto.AddReactionRequest{Emoji: "heart"}},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().AddReaction(mock.Anything, chatCtlMessageID, chatCtlUserID, "heart").Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "remove reaction unescapes a url-encoded emoji",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMessagePath + "/reactions/" + url.PathEscape("heart eyes")},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().RemoveReaction(mock.Anything, chatCtlMessageID, chatCtlUserID, "heart eyes").Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "set member nickname as mod",
			req:  chatCtlRequest{method: "PUT", path: chatCtlMemberPath + "/nickname", json: dto.UpdateMemberProfileRequest{Nickname: "Forced"}},
			expect: func(m *chatsvc.MockService) {
				member := &dto.ChatRoomMemberResponse{
					User:           dto.UserResponse{ID: chatCtlTargetID, Username: "target"},
					Role:           "member",
					Nickname:       "Forced",
					NicknameLocked: true,
				}
				m.EXPECT().SetMemberNicknameAsMod(mock.Anything, chatCtlRoomID, chatCtlUserID, chatCtlTargetID, "Forced").Return(member, nil)
			},
			wantCode: http.StatusOK,
			wantBody: []string{"Forced", chatCtlTargetID.String()},
		},
		{
			name: "unlock member nickname",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMemberPath + "/nickname"},
			expect: func(m *chatsvc.MockService) {
				member := &dto.ChatRoomMemberResponse{
					User:           dto.UserResponse{ID: chatCtlTargetID, Username: "target"},
					Role:           "member",
					NicknameLocked: false,
				}
				m.EXPECT().UnlockMemberNickname(mock.Anything, chatCtlRoomID, chatCtlUserID, chatCtlTargetID).Return(member, nil)
			},
			wantCode: http.StatusOK,
			wantBody: []string{chatCtlTargetID.String()},
		},
		{
			name: "delete message",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMessagePath},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().DeleteMessage(mock.Anything, chatCtlMessageID, chatCtlUserID).Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "edit message",
			req:  chatCtlRequest{method: "PATCH", path: chatCtlMessagePath, json: dto.EditMessageRequest{Body: "updated"}},
			expect: func(m *chatsvc.MockService) {
				m.EXPECT().EditMessage(mock.Anything, chatCtlMessageID, chatCtlUserID, "updated").
					Return(&dto.ChatMessageResponse{ID: chatCtlMessageID, Body: "updated", EditedAt: new("2026-04-18T20:00:00Z")}, nil)
			},
			wantCode: http.StatusOK,
			wantBody: []string{`"body":"updated"`, `"edited_at"`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, chatMock := chatCtlHarnessFor(t, tc.req)
			tc.expect(chatMock)

			// when
			status, body := tc.req.do(h)

			// then
			require.Equal(t, tc.wantCode, status)
			for _, want := range tc.wantBody {
				assert.Contains(t, string(body), want)
			}
			if tc.check != nil {
				tc.check(t, body)
			}
		})
	}
}

func TestChatRoutes_ServiceErrors(t *testing.T) {
	boom := errors.New("boom")

	endpoints := []struct {
		name   string
		req    chatCtlRequest
		expect func(m *chatsvc.MockService, err error)
		cases  []chatCtlErrCase
	}{
		{
			name: "resolve dm",
			req:  chatCtlRequest{method: "GET", path: chatCtlDMPath + "/resolve"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().ResolveDMRoom(mock.Anything, chatCtlUserID, chatCtlTargetID).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"blocked", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot message"},
				{"dms disabled", chatsvc.ErrDmsDisabled, http.StatusForbidden, "DMs disabled"},
				{"user not found", chatsvc.ErrUserNotFound, http.StatusNotFound, "user not found"},
				{"cannot dm self", chatsvc.ErrCannotDMSelf, http.StatusBadRequest, "cannot DM yourself"},
				{"internal", boom, http.StatusInternalServerError, "chat operation failed"},
			},
		},
		{
			name: "send first dm",
			req:  chatCtlMessageForm(t, chatCtlDMPath+"/messages"),
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().SendDMMessage(mock.Anything, chatCtlUserID, chatCtlTargetID, "hi", []chatsvc.FileUpload(nil)).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "message body is required"},
				{"blocked", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot message"},
				{"internal", boom, http.StatusInternalServerError, "chat operation failed"},
			},
		},
		{
			name: "create group room",
			req:  chatCtlRequest{method: "POST", path: "/chat/rooms", json: dto.CreateGroupRoomRequest{Name: ""}},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().CreateGroupRoom(mock.Anything, chatCtlUserID, dto.CreateGroupRoomRequest{Name: ""}).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "room name is required"},
				{"bot outside an rp room", chatsvc.ErrBotsRPRoomsOnly, http.StatusBadRequest, "bots can only be added to roleplay rooms"},
				{"internal", boom, http.StatusInternalServerError, "failed to create group room"},
			},
		},
		{
			name: "update room",
			req:  chatCtlRequest{method: "PUT", path: chatCtlRoomPath, json: dto.UpdateGroupRoomRequest{Name: "room"}},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().UpdateGroupRoom(mock.Anything, chatCtlRoomID, chatCtlUserID, dto.UpdateGroupRoomRequest{Name: "room"}).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"content filter rejection maps first", &contentfilter.RejectedError{Rejection: contentfilter.Rejection{Rule: "slurs", Reason: "nope", Detail: "slur"}}, http.StatusBadRequest, "content_rejected"},
				{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "room name is required"},
				{"room not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "room not found"},
				{"system room", chatsvc.ErrSystemRoom, http.StatusForbidden, "this room is managed automatically"},
				{"not group room", chatsvc.ErrNotGroupRoom, http.StatusBadRequest, "only group rooms can be edited"},
				{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host or a moderator can do this"},
				{"internal", boom, http.StatusInternalServerError, "failed to update room"},
			},
		},
		{
			name: "list rooms",
			req:  chatCtlRequest{method: "GET", path: "/chat/rooms"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().ListRooms(mock.Anything, chatCtlUserID).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"internal", boom, http.StatusInternalServerError, "failed to list rooms"},
			},
		},
		{
			name: "list my group rooms",
			req:  chatCtlRequest{method: "GET", path: "/chat/rooms/mine"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().ListUserGroupRooms(mock.Anything, chatCtlUserID, "", false, "", "", false, 20, 0).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"internal", boom, http.StatusInternalServerError, "failed to list rooms"},
			},
		},
		{
			name: "list public rooms",
			req:  chatCtlRequest{method: "GET", path: "/chat/rooms/public", anonymous: true},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().ListPublicRooms(mock.Anything, "", false, "", uuid.Nil, false, 20, 0).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"internal", boom, http.StatusInternalServerError, "failed to list public rooms"},
			},
		},
		{
			name: "join room",
			req:  chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/join"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().JoinRoom(mock.Anything, chatCtlRoomID, chatCtlUserID, false).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"room not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "room not found"},
				{"not group room", chatsvc.ErrNotGroupRoom, http.StatusBadRequest, "not a group room"},
				{"not public", chatsvc.ErrNotPublic, http.StatusForbidden, "not public"},
				{"room full", chatsvc.ErrRoomFull, http.StatusConflict, "room is full"},
				{"blocked", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot join this room"},
				{"internal", boom, http.StatusInternalServerError, "failed to join room"},
			},
		},
		{
			name: "leave room",
			req:  chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/leave"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().LeaveRoom(mock.Anything, chatCtlRoomID, chatCtlUserID).Return(err)
			},
			cases: []chatCtlErrCase{
				{"cannot leave as host", chatsvc.ErrCannotLeaveAsHost, http.StatusForbidden, "host cannot leave"},
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to leave room"},
			},
		},
		{
			name: "get room members",
			req:  chatCtlRequest{method: "GET", path: chatCtlRoomPath + "/members"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().GetMembers(mock.Anything, chatCtlUserID, chatCtlRoomID).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to get members"},
			},
		},
		{
			name: "kick member",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMemberPath},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().KickMember(mock.Anything, chatCtlUserID, chatCtlRoomID, chatCtlTargetID).Return(err)
			},
			cases: []chatCtlErrCase{
				{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host can kick"},
				{"cannot kick host", chatsvc.ErrCannotKickHost, http.StatusBadRequest, "cannot kick the host"},
				{"not member", chatsvc.ErrNotMember, http.StatusNotFound, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to kick member"},
			},
		},
		{
			name: "invite members",
			req:  chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/members", json: dto.InviteMembersRequest{UserIDs: []uuid.UUID{chatCtlTargetID}}},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().InviteMembers(mock.Anything, chatCtlUserID, chatCtlRoomID, []uuid.UUID{chatCtlTargetID}).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"room not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "room not found"},
				{"not group room", chatsvc.ErrNotGroupRoom, http.StatusBadRequest, "only group rooms"},
				{"system room", chatsvc.ErrSystemRoom, http.StatusForbidden, "managed automatically"},
				{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host can invite"},
				{"bot outside an rp room", chatsvc.ErrBotsRPRoomsOnly, http.StatusBadRequest, "bots can only be added to roleplay rooms"},
				{"internal", boom, http.StatusInternalServerError, "failed to invite members"},
			},
		},
		{
			name: "set member timeout",
			req:  chatCtlRequest{method: "PUT", path: chatCtlMemberPath + "/timeout", json: dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"}},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().SetMemberTimeout(mock.Anything, chatCtlRoomID, chatCtlUserID, chatCtlTargetID, dto.SetMemberTimeoutRequest{Amount: 1, Unit: "hours"}).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only room hosts and site moderators"},
				{"not member", chatsvc.ErrNotMember, http.StatusNotFound, "user is not a member"},
				{"cannot timeout host", chatsvc.ErrCannotKickHost, http.StatusBadRequest, "cannot timeout the host"},
				{"target immune", chatsvc.ErrTargetImmune, http.StatusForbidden, "cannot be timed out"},
				{"invalid duration", chatsvc.ErrInvalidTimeoutDuration, http.StatusBadRequest, "invalid timeout duration"},
				{"locked by staff", chatsvc.ErrTimeoutLockedByStaff, http.StatusForbidden, "can only be changed by site moderators"},
				{"internal", boom, http.StatusInternalServerError, "failed to set timeout"},
			},
		},
		{
			name: "clear member timeout",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMemberPath + "/timeout"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().ClearMemberTimeout(mock.Anything, chatCtlRoomID, chatCtlUserID, chatCtlTargetID).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only room hosts and site moderators"},
				{"not member", chatsvc.ErrNotMember, http.StatusNotFound, "user is not a member"},
				{"locked by staff", chatsvc.ErrTimeoutLockedByStaff, http.StatusForbidden, "can only be removed by site moderators"},
				{"internal", boom, http.StatusInternalServerError, "failed to clear timeout"},
			},
		},
		{
			name: "set room mute",
			req:  chatCtlRequest{method: "PUT", path: chatCtlRoomPath + "/mute", json: map[string]bool{"muted": false}},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().SetRoomMuted(mock.Anything, chatCtlRoomID, chatCtlUserID, false).Return(err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to set mute"},
			},
		},
		{
			name: "get messages",
			req:  chatCtlRequest{method: "GET", path: chatCtlRoomPath + "/messages"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().GetMessages(mock.Anything, chatCtlUserID, chatCtlRoomID, 50, 0).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to get messages"},
			},
		},
		{
			name: "send message",
			req:  chatCtlMessageForm(t, chatCtlRoomPath+"/messages"),
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().SendMessage(mock.Anything, chatCtlUserID, chatCtlRoomID, dto.SendMessageRequest{Body: "hi"}, []chatsvc.FileUpload(nil)).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"blocked", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot message"},
				{"timed out", chatsvc.ErrTimedOut, http.StatusForbidden, chatsvc.ErrTimedOut.Error()},
				{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "message body is required"},
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"invalid file type", upload.ErrInvalidFileType, http.StatusBadRequest, upload.ErrInvalidFileType.Error()},
				{"internal", boom, http.StatusInternalServerError, "failed to send message"},
			},
		},
		{
			name: "delete chat",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlRoomPath},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().DeleteChat(mock.Anything, chatCtlRoomID, chatCtlUserID).Return(err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to delete chat"},
			},
		},
		{
			name: "unread count",
			req:  chatCtlRequest{method: "GET", path: "/chat/unread-count"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().GetUnreadCount(mock.Anything, chatCtlUserID).Return(0, err)
			},
			cases: []chatCtlErrCase{
				{"internal", boom, http.StatusInternalServerError, "failed to get unread count"},
			},
		},
		{
			name: "mark room read",
			req:  chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/read"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().MarkRead(mock.Anything, chatCtlRoomID, chatCtlUserID).Return(err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to mark room read"},
			},
		},
		{
			name: "set room nickname",
			req:  chatCtlRequest{method: "PUT", path: chatCtlRoomPath + "/me", json: dto.UpdateMemberProfileRequest{Nickname: "nick"}},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().SetRoomNickname(mock.Anything, chatCtlRoomID, chatCtlUserID, "nick").Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to update nickname"},
			},
		},
		{
			name: "set room avatar",
			req:  chatCtlAvatarUpload(t),
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().SetRoomAvatar(mock.Anything, chatCtlRoomID, chatCtlUserID, "image/png", int64(len(chatCtlAvatarData)), mock.Anything).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"file too large", upload.ErrFileTooLarge, http.StatusBadRequest, "file too large"},
				{"invalid file type", upload.ErrInvalidFileType, http.StatusBadRequest, "PNG"},
				{"internal", boom, http.StatusInternalServerError, "failed to upload avatar"},
			},
		},
		{
			name: "clear room avatar",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlRoomPath + "/me/avatar"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().ClearRoomAvatar(mock.Anything, chatCtlRoomID, chatCtlUserID).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to clear avatar"},
			},
		},
		{
			name: "pin message",
			req:  chatCtlRequest{method: "POST", path: chatCtlMessagePath + "/pin"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().PinMessage(mock.Anything, chatCtlMessageID, chatCtlUserID).Return(err)
			},
			cases: []chatCtlErrCase{
				{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host can pin"},
				{"message not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to pin message"},
			},
		},
		{
			name: "unpin message",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMessagePath + "/pin"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().UnpinMessage(mock.Anything, chatCtlMessageID, chatCtlUserID).Return(err)
			},
			cases: []chatCtlErrCase{
				{"not host", chatsvc.ErrNotHost, http.StatusForbidden, "only the host can unpin"},
				{"not pinned", chatsvc.ErrMessageNotPinned, http.StatusBadRequest, "not pinned"},
				{"message not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to unpin message"},
			},
		},
		{
			name: "list pinned messages",
			req:  chatCtlRequest{method: "GET", path: chatCtlRoomPath + "/pins"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().ListPinnedMessages(mock.Anything, chatCtlRoomID, chatCtlUserID).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"internal", boom, http.StatusInternalServerError, "failed to list pinned messages"},
			},
		},
		{
			name: "add reaction",
			req:  chatCtlRequest{method: "POST", path: chatCtlMessagePath + "/reactions", json: dto.AddReactionRequest{Emoji: "heart"}},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().AddReaction(mock.Anything, chatCtlMessageID, chatCtlUserID, "heart").Return(err)
			},
			cases: []chatCtlErrCase{
				{"invalid emoji", chatsvc.ErrInvalidEmoji, http.StatusBadRequest, "invalid emoji"},
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"message not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to add reaction"},
			},
		},
		{
			name: "remove reaction",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMessagePath + "/reactions/heart"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().RemoveReaction(mock.Anything, chatCtlMessageID, chatCtlUserID, "heart").Return(err)
			},
			cases: []chatCtlErrCase{
				{"invalid emoji", chatsvc.ErrInvalidEmoji, http.StatusBadRequest, "invalid emoji"},
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"message not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
				{"internal", boom, http.StatusInternalServerError, "failed to remove reaction"},
			},
		},
		{
			name: "set member nickname as mod",
			req:  chatCtlRequest{method: "PUT", path: chatCtlMemberPath + "/nickname", json: dto.UpdateMemberProfileRequest{Nickname: "x"}},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().SetMemberNicknameAsMod(mock.Anything, chatCtlRoomID, chatCtlUserID, chatCtlTargetID, "x").Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"not mod", chatsvc.ErrModRoleRequired, http.StatusForbidden, "only site moderators"},
				{"target immune", chatsvc.ErrTargetImmune, http.StatusForbidden, "cannot be changed by moderators"},
				{"internal", boom, http.StatusInternalServerError, "failed to set member nickname"},
			},
		},
		{
			name: "unlock member nickname",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMemberPath + "/nickname"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().UnlockMemberNickname(mock.Anything, chatCtlRoomID, chatCtlUserID, chatCtlTargetID).Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"not mod", chatsvc.ErrModRoleRequired, http.StatusForbidden, "only site moderators"},
				{"target immune", chatsvc.ErrTargetImmune, http.StatusForbidden, "not affected by nickname locks"},
				{"internal", boom, http.StatusInternalServerError, "failed to unlock nickname"},
			},
		},
		{
			name: "delete message",
			req:  chatCtlRequest{method: "DELETE", path: chatCtlMessagePath},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().DeleteMessage(mock.Anything, chatCtlMessageID, chatCtlUserID).Return(err)
			},
			cases: []chatCtlErrCase{
				{"not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
				{"permission", chatsvc.ErrMessageDeletePermission, http.StatusForbidden, "do not have permission"},
				{"internal", boom, http.StatusInternalServerError, "failed to delete message"},
			},
		},
		{
			name: "edit message",
			req:  chatCtlRequest{method: "PATCH", path: chatCtlMessagePath, json: dto.EditMessageRequest{Body: "new"}},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().EditMessage(mock.Anything, chatCtlMessageID, chatCtlUserID, "new").Return(nil, err)
			},
			cases: []chatCtlErrCase{
				{"not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "message not found"},
				{"permission", chatsvc.ErrMessageEditPermission, http.StatusForbidden, "can only edit your own messages"},
				{"kicked author is a refusal not a server error", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"banned word is a refusal not a server error", &chatsvc.ErrBannedWordMatch{Pattern: "badword", Action: "delete"}, http.StatusUnprocessableEntity, `"code":"banned_word"`},
				{"system", chatsvc.ErrCannotEditSystemMessage, http.StatusBadRequest, "system messages"},
				{"missing fields", chatsvc.ErrMissingFields, http.StatusBadRequest, "body is required"},
				{"timed out", chatsvc.ErrTimedOut, http.StatusForbidden, "timed out"},
				{"internal", boom, http.StatusInternalServerError, "failed to edit message"},
			},
		},
		{
			name: "voice token",
			req:  chatCtlRequest{method: "POST", path: chatCtlRoomPath + "/voice/token"},
			expect: func(m *chatsvc.MockService, err error) {
				m.EXPECT().MintVoiceToken(mock.Anything, chatCtlRoomID, chatCtlUserID).Return("", "", err)
			},
			cases: []chatCtlErrCase{
				{"voice disabled", chatsvc.ErrVoiceDisabled, http.StatusServiceUnavailable, "not configured"},
				{"not a member", chatsvc.ErrNotMember, http.StatusForbidden, "not a member"},
				{"room not found", chatsvc.ErrRoomNotFound, http.StatusNotFound, "room not found"},
				{"blocked in a dm", chatsvc.ErrUserBlocked, http.StatusForbidden, "cannot call this user"},
				{"blocked by the room host", chatsvc.ErrBlockedByRoomHost, http.StatusForbidden, "host of this room has blocked you"},
				{"internal does not leak the cause", boom, http.StatusInternalServerError, `{"error":"voice request failed"}`},
			},
		},
	}
	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			for _, tc := range ep.cases {
				t.Run(tc.name, func(t *testing.T) {
					// given
					h, chatMock := chatCtlHarnessFor(t, ep.req)
					ep.expect(chatMock, tc.err)

					// when
					status, body := ep.req.do(h)

					// then
					require.Equal(t, tc.wantCode, status)
					assert.Contains(t, string(body), tc.wantBody)
				})
			}
		})
	}
}

func TestUpdateRoom_RouteIsRegistered(t *testing.T) {
	// given
	h, _ := newChatHarness(t)

	// when
	found := false
	for _, r := range h.App.GetRoutes(true) {
		if r.Method == http.MethodPut && r.Path == "/chat/rooms/:roomID" {
			found = true
		}
	}

	// then
	assert.True(t, found, "PUT /chat/rooms/:roomID must be wired into getAllChatRoutes")
}

func TestUpdateRoom_BotsWillBeKickedIs409WithTheBotList(t *testing.T) {
	// given
	update := dto.UpdateGroupRoomRequest{Name: "room"}
	req := chatCtlRequest{method: "PUT", path: chatCtlRoomPath, json: update}
	h, chatMock := chatCtlHarnessFor(t, req)
	botID := uuid.New()
	chatMock.EXPECT().UpdateGroupRoom(mock.Anything, chatCtlRoomID, chatCtlUserID, update).
		Return(nil, &chatsvc.ErrBotsWillBeKicked{Bots: []dto.UserResponse{{ID: botID, Username: "beato", DisplayName: "Beatrice"}}})

	// when
	status, body := req.do(h)

	// then
	require.Equal(t, http.StatusConflict, status)
	got := testutil.UnmarshalJSON[struct {
		Error string             `json:"error"`
		Code  string             `json:"code"`
		Bots  []dto.UserResponse `json:"bots"`
	}](t, body)
	assert.Equal(t, "bots_will_be_kicked", got.Code)
	assert.Equal(t, "turning roleplay off will remove 1 bot from this room", got.Error)
	require.Len(t, got.Bots, 1)
	assert.Equal(t, botID, got.Bots[0].ID)
	assert.Equal(t, "Beatrice", got.Bots[0].DisplayName)
}

func TestParseSpoilerIndexes(t *testing.T) {
	tests := []struct {
		name   string
		form   *multipart.Form
		want   map[int]struct{}
		wantOK bool
	}{
		{"no form means no spoilers", nil, nil, true},
		{"an absent field means no spoilers", &multipart.Form{Value: map[string][]string{}}, nil, true},
		{"an empty field means no spoilers", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {""}}}, nil, true},
		{"a single index", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {"0"}}}, map[int]struct{}{0: {}}, true},
		{"several indexes", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {"0,2"}}}, map[int]struct{}{0: {}, 2: {}}, true},
		{"surrounding space is tolerated", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {" 1 , 3 "}}}, map[int]struct{}{1: {}, 3: {}}, true},
		{"a negative index is refused", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {"-1"}}}, nil, false},
		{"a non-numeric token is refused", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {"0,beatrice"}}}, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a multipart form carrying the sender's spoiler choices

			// when it is parsed
			got, ok := parseSpoilerIndexes(tc.form)

			// then only a well-formed list is accepted
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCollectChatFileUploads_MarksOnlyTheChosenFiles(t *testing.T) {
	// given two attachments where only the second was marked
	form := &multipart.Form{
		File: map[string][]*multipart.FileHeader{
			"media": {
				{Filename: "one.png"},
				{Filename: "two.png"},
			},
		},
	}

	// when they are collected
	uploads := collectChatFileUploads(form, map[int]struct{}{1: {}})

	// then the flag lands on the file the sender picked, and the filename is carried
	require.Len(t, uploads, 2)
	assert.False(t, uploads[0].IsSpoiler)
	assert.True(t, uploads[1].IsSpoiler)
	assert.Equal(t, "one.png", uploads[0].Filename)
	assert.Equal(t, "two.png", uploads[1].Filename)
}
