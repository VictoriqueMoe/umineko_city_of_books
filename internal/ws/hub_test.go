package ws

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHub_IsUserInRoom_GatesNonMembers(t *testing.T) {
	// given
	hub := NewHub()
	roomID := uuid.New()
	member := uuid.New()
	stranger := uuid.New()
	hub.JoinRoom(roomID, member)

	// then
	assert.True(t, hub.IsUserInRoom(roomID, member), "member should be in room")
	assert.False(t, hub.IsUserInRoom(roomID, stranger), "stranger should not be in room")
	assert.False(t, hub.IsUserInRoom(uuid.New(), member), "member should not be in an unknown room")
}

func TestHub_IsUserInRoom_LeaveRemoves(t *testing.T) {
	// given
	hub := NewHub()
	roomID := uuid.New()
	userID := uuid.New()
	hub.JoinRoom(roomID, userID)

	// when
	hub.LeaveRoom(roomID, userID)

	// then
	assert.False(t, hub.IsUserInRoom(roomID, userID), "user should be out after LeaveRoom")
}

func TestHub_BroadcastFrameToRoom_LeavesHeadroomForOrdinaryMessages(t *testing.T) {
	cases := []struct {
		name       string
		backlog    int
		wantSent   int
		wantBuffer int
	}{
		{name: "an idle buffer takes the frame", backlog: 0, wantSent: 1, wantBuffer: 1},
		{name: "just under the watermark takes the frame", backlog: sendBufferSize/2 - 1, wantSent: 1, wantBuffer: sendBufferSize / 2},
		{name: "at the watermark the frame is dropped", backlog: sendBufferSize / 2, wantSent: 0, wantBuffer: sendBufferSize / 2},
		{name: "over the watermark the frame is dropped", backlog: sendBufferSize/2 + 5, wantSent: 0, wantBuffer: sendBufferSize/2 + 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			hub := NewHub()
			roomID := uuid.New()
			viewer := NewClient(uuid.New(), nil)
			hub.clients[viewer.UserID] = []*Client{viewer}
			hub.JoinRoom(roomID, viewer.UserID)
			for range tc.backlog {
				viewer.send <- []byte("backlog")
			}

			// when
			sent := hub.BroadcastFrameToRoom(roomID, Message{Type: "game_pong_frame"})

			// then
			assert.Equal(t, tc.wantSent, sent)
			assert.Equal(t, tc.wantBuffer, len(viewer.send))
		})
	}
}

func TestHub_BroadcastFrameToRoom_NeverKillsAClientTheNextBroadcastNeeds(t *testing.T) {
	// given
	hub := NewHub()
	roomID := uuid.New()
	viewer := NewClient(uuid.New(), nil)
	hub.clients[viewer.UserID] = []*Client{viewer}
	hub.JoinRoom(roomID, viewer.UserID)

	// when
	for range sendBufferSize * 4 {
		hub.BroadcastFrameToRoom(roomID, Message{Type: "game_pong_frame"})
	}
	hub.BroadcastToRoom(roomID, Message{Type: "chat_message"}, uuid.Nil)

	// then
	assert.LessOrEqual(t, len(viewer.send), sendBufferSize/2+1)
	assert.Len(t, hub.clients[viewer.UserID], 1, "a frame backlog must never cost a client its connection")
}

func TestHub_BroadcastPublic_ReachesAuthedAndAnon(t *testing.T) {
	// given
	hub := NewHub()
	authed := NewClient(uuid.New(), nil)
	anon := NewClient(uuid.Nil, nil)
	hub.clients[authed.UserID] = []*Client{authed}
	hub.anon[anon] = struct{}{}

	// when
	hub.BroadcastPublic(Message{Type: "stream_live"})

	// then
	assert.Equal(t, 1, len(authed.send), "authed client should receive a public broadcast")
	assert.Equal(t, 1, len(anon.send), "anon client should receive a public broadcast")
}

func TestHub_Broadcast_DoesNotReachAnon(t *testing.T) {
	// given
	hub := NewHub()
	authed := NewClient(uuid.New(), nil)
	anon := NewClient(uuid.Nil, nil)
	hub.clients[authed.UserID] = []*Client{authed}
	hub.anon[anon] = struct{}{}

	// when
	hub.Broadcast(Message{Type: "ban_changed"})

	// then
	assert.Equal(t, 1, len(authed.send), "authed client should receive an authed broadcast")
	assert.Equal(t, 0, len(anon.send), "anon client must never receive an authed broadcast")
}

func TestHub_SendToUser_DoesNotReachAnon(t *testing.T) {
	// given
	hub := NewHub()
	anon := NewClient(uuid.Nil, nil)
	hub.anon[anon] = struct{}{}

	// when
	hub.SendToUser(uuid.New(), Message{Type: "notification"})

	// then
	assert.Equal(t, 0, len(anon.send), "anon client must never receive a user-targeted message")
}

func TestHub_SendToUser_ReachesEveryConnectionOfThatUser(t *testing.T) {
	// given the same user connected from two devices
	hub := NewHub()
	userID := uuid.New()
	phone := NewClient(userID, nil)
	desktop := NewClient(userID, nil)
	hub.clients[userID] = []*Client{phone, desktop}

	// when a message is sent to that user
	hub.SendToUser(userID, Message{Type: "chat_message"})

	// then it lands on every one of their connections, so a message sent from one device
	assert.Equal(t, 1, len(phone.send), "phone should receive the message")
	assert.Equal(t, 1, len(desktop.send), "desktop should receive the message")
}

func TestHub_BroadcastToRoom_ExcludeSkipsAllOfThatUsersConnections(t *testing.T) {
	// given a room with a sender on two devices and another member on one device
	hub := NewHub()
	roomID := uuid.New()
	senderID := uuid.New()
	otherID := uuid.New()
	senderPhone := NewClient(senderID, nil)
	senderDesktop := NewClient(senderID, nil)
	otherPhone := NewClient(otherID, nil)
	hub.clients[senderID] = []*Client{senderPhone, senderDesktop}
	hub.clients[otherID] = []*Client{otherPhone}
	hub.JoinRoom(roomID, senderID)
	hub.JoinRoom(roomID, otherID)

	// when a room broadcast excludes the sender by user id
	hub.BroadcastToRoom(roomID, Message{Type: "chat_message"}, senderID)

	// then the exclusion drops the sender on every device, which is why excluding the sender
	assert.Equal(t, 0, len(senderPhone.send), "excluded sender's phone should be skipped")
	assert.Equal(t, 0, len(senderDesktop.send), "excluded sender's desktop should be skipped")
	assert.Equal(t, 1, len(otherPhone.send), "other member should still receive the message")
}

func TestHub_DisconnectUser_KillsEveryConnectionOfThatUser(t *testing.T) {
	// given
	hub := NewHub()
	victim := uuid.New()
	bystander := uuid.New()
	first := NewClient(victim, nil)
	second := NewClient(victim, nil)
	other := NewClient(bystander, nil)
	hub.clients[victim] = []*Client{first, second}
	hub.clients[bystander] = []*Client{other}

	// when
	killed := hub.DisconnectUser(victim)

	// then
	assert.Equal(t, 2, killed)
	assert.True(t, isKilled(first), "first socket should be killed")
	assert.True(t, isKilled(second), "second socket should be killed")
	assert.False(t, isKilled(other), "another user's socket must be untouched")
}

func TestHub_DisconnectUser_UnknownUserIsNoOp(t *testing.T) {
	// given
	hub := NewHub()
	present := NewClient(uuid.New(), nil)
	hub.clients[present.UserID] = []*Client{present}

	// when
	killed := hub.DisconnectUser(uuid.New())

	// then
	assert.Equal(t, 0, killed)
	assert.False(t, isKilled(present), "existing socket must survive")
}

func isKilled(c *Client) bool {
	select {
	case <-c.closeCh:
		return true
	default:
		return false
	}
}

func TestIsOnline_AlwaysOnlineMembersNeedNoConnection(t *testing.T) {
	// given
	hub := NewHub("test")
	bot := uuid.New()
	human := uuid.New()

	// then
	assert.False(t, hub.IsOnline(bot), "nobody is online before registration")

	// when
	hub.SetAlwaysOnline([]uuid.UUID{bot})

	// then
	assert.True(t, hub.IsOnline(bot), "a bot reads as online without a websocket")
	assert.False(t, hub.IsOnline(human), "real members still need a connection")
	assert.True(t, hub.IsAlwaysOnline(bot), "the bot is in the always online set")
	assert.False(t, hub.IsAlwaysOnline(human), "a human is never in the always online set")

	// when
	hub.SetAlwaysOnline(nil)

	// then
	assert.False(t, hub.IsOnline(bot), "a deleted or disabled bot stops being online")
	assert.False(t, hub.IsAlwaysOnline(bot), "a deleted or disabled bot leaves the always online set")
}

func TestGetRoomPresence_ReportsOnlyRoomViewers(t *testing.T) {
	// given
	hub := NewHub("test")
	roomID := uuid.New()
	bot := uuid.New()

	// when
	hub.SetAlwaysOnline([]uuid.UUID{bot})

	// then
	assert.Empty(t, hub.GetRoomPresence(roomID), "an always online member is not viewing any room")
}

func TestHub_AddViewerAfterJoinRoomMakesTheUserViewing(t *testing.T) {
	// given
	tests := []struct {
		name        string
		arrange     func(hub *Hub, roomID, userID uuid.UUID)
		wantInRoom  bool
		wantViewing bool
	}{
		{
			name:        "a cold room is neither joined nor viewed",
			arrange:     func(hub *Hub, roomID, userID uuid.UUID) {},
			wantInRoom:  false,
			wantViewing: false,
		},
		{
			name: "membership on its own is not viewing",
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.JoinRoom(roomID, userID)
			},
			wantInRoom:  true,
			wantViewing: false,
		},
		{
			name: "AddViewer after JoinRoom makes the user viewing",
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.JoinRoom(roomID, userID)
				hub.AddViewer(roomID, userID)
			},
			wantInRoom:  true,
			wantViewing: true,
		},
		{
			name: "closing one of two tabs keeps the user viewing",
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.JoinRoom(roomID, userID)
				hub.AddViewer(roomID, userID)
				hub.AddViewer(roomID, userID)
				hub.RemoveViewer(roomID, userID)
			},
			wantInRoom:  true,
			wantViewing: true,
		},
		{
			name: "closing the last tab ends viewing but keeps membership",
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.JoinRoom(roomID, userID)
				hub.AddViewer(roomID, userID)
				hub.RemoveViewer(roomID, userID)
			},
			wantInRoom:  true,
			wantViewing: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			roomID := uuid.New()
			userID := uuid.New()

			// when
			tt.arrange(hub, roomID, userID)

			// then
			assert.Equal(t, tt.wantInRoom, hub.IsUserInRoom(roomID, userID))
			assert.Equal(t, tt.wantViewing, hub.IsUserViewing(roomID, userID))
		})
	}
}

func TestHub_IsUserViewing_IsScopedToOneUserAndOneRoom(t *testing.T) {
	// given
	hub := NewHub()
	roomID := uuid.New()
	otherRoomID := uuid.New()
	viewer := uuid.New()
	otherMember := uuid.New()
	hub.JoinRoom(roomID, viewer)
	hub.JoinRoom(roomID, otherMember)
	hub.JoinRoom(otherRoomID, viewer)

	// when
	hub.AddViewer(roomID, viewer)

	// then
	assert.True(t, hub.IsUserViewing(roomID, viewer), "the viewer is viewing the room they opened")
	assert.False(t, hub.IsUserViewing(roomID, otherMember), "another member of the same room is not viewing it")
	assert.False(t, hub.IsUserViewing(otherRoomID, viewer), "the viewer is not viewing their other room")
}
