package ws

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/coalesce"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHub_WatchPosts_ReplacesTheWatchedSet(t *testing.T) {
	// given
	ids := make([]uuid.UUID, maxWatchedPosts+5)
	for i := range ids {
		ids[i] = uuid.New()
	}

	tests := []struct {
		name     string
		previous []uuid.UUID
		next     []uuid.UUID
		want     []uuid.UUID
	}{
		{name: "a first call watches every id", next: ids[:3], want: ids[:3]},
		{name: "a later call replaces the set rather than extending it", previous: ids[:3], next: ids[3:5], want: ids[3:5]},
		{name: "an empty list clears the set", previous: ids[:3], next: []uuid.UUID{}, want: nil},
		{name: "a repeated id is watched once", next: []uuid.UUID{ids[0], ids[0], ids[1]}, want: ids[:2]},
		{name: "only the first 200 ids are kept", next: ids, want: ids[:maxWatchedPosts]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			client := NewClient(uuid.New(), nil)
			hub.WatchPosts(client, tt.previous)

			// when
			hub.WatchPosts(client, tt.next)

			// then
			assert.ElementsMatch(t, tt.want, slices.Collect(maps.Keys(client.watchedPosts)))
			assert.ElementsMatch(t, tt.want, slices.Collect(maps.Keys(hub.postWatchers)), "the hub index must hold exactly the watched posts")
			for _, postID := range tt.want {
				assert.Contains(t, hub.postWatchers[postID], client)
			}
		})
	}
}

func TestHub_SendToPostWatchers_ReachesOnlyWatchingConnections(t *testing.T) {
	// given
	firstPost := uuid.New()
	secondPost := uuid.New()

	tests := []struct {
		name      string
		target    uuid.UUID
		wantTabA  int
		wantTabB  int
		wantOther int
		wantIdle  int
	}{
		{name: "the first post reaches the tab and the other member watching it", target: firstPost, wantTabA: 1, wantOther: 1},
		{name: "the second post reaches only the other tab of the same user", target: secondPost, wantTabB: 1},
		{name: "an unwatched post reaches nobody", target: uuid.New()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			userID := uuid.New()
			tabA := NewClient(userID, nil)
			tabB := NewClient(userID, nil)
			other := NewClient(uuid.New(), nil)
			idle := NewClient(uuid.New(), nil)
			hub.clients[userID] = []*Client{tabA, tabB}
			hub.clients[other.UserID] = []*Client{other}
			hub.clients[idle.UserID] = []*Client{idle}
			hub.WatchPosts(tabA, []uuid.UUID{firstPost})
			hub.WatchPosts(tabB, []uuid.UUID{secondPost})
			hub.WatchPosts(other, []uuid.UUID{firstPost})

			// when
			hub.SendToPostWatchers(tt.target, Message{Type: "post_like"})

			// then
			assert.Equal(t, tt.wantTabA, len(tabA.send), "first tab")
			assert.Equal(t, tt.wantTabB, len(tabB.send), "second tab of the same user")
			assert.Equal(t, tt.wantOther, len(other.send), "another member")
			assert.Equal(t, tt.wantIdle, len(idle.send), "a member watching nothing")
		})
	}
}

func TestHub_Unregister_DropsOnlyThatConnectionsWatches(t *testing.T) {
	// given the same user watching one post from two tabs
	hub := NewHub()
	userID := uuid.New()
	postID := uuid.New()
	first := NewClient(userID, nil)
	second := NewClient(userID, nil)
	hub.clients[userID] = []*Client{first, second}
	hub.WatchPosts(first, []uuid.UUID{postID})
	hub.WatchPosts(second, []uuid.UUID{postID})

	// when one tab closes
	hub.Unregister(first)
	hub.SendToPostWatchers(postID, Message{Type: "post_comment"})

	// then
	assert.Equal(t, 0, len(first.send), "the closed tab must not be sent to")
	assert.Equal(t, 1, len(second.send), "the open tab keeps its watch")
	assert.NotContains(t, hub.postWatchers[postID], first)

	// when the last tab closes
	hub.Unregister(second)

	// then
	assert.Empty(t, hub.postWatchers, "no watcher entries may outlive their connections")
}

func TestHub_SendToPostWatchers_ReapsAClientWithAFullBuffer(t *testing.T) {
	// given
	hub := NewHub()
	postID := uuid.New()
	stuck := NewClient(uuid.New(), nil)
	hub.clients[stuck.UserID] = []*Client{stuck}
	hub.WatchPosts(stuck, []uuid.UUID{postID})
	for range sendBufferSize {
		stuck.send <- []byte("{}")
	}

	// when
	hub.SendToPostWatchers(postID, Message{Type: "post_like"})

	// then
	assert.True(t, isKilled(stuck), "a client that cannot keep up is killed")
	assert.NotContains(t, hub.clients, stuck.UserID, "a killed client is unregistered")
	assert.Empty(t, hub.postWatchers, "a killed client's watches are dropped")

	// when its reader still asks to watch before it notices
	hub.WatchPosts(stuck, []uuid.UUID{postID})

	// then
	assert.Empty(t, hub.postWatchers, "a killed client cannot watch again")
}

func TestHub_BumpSidebarActivity_ThrottlesEachKey(t *testing.T) {
	interval := 100 * time.Millisecond

	tests := []struct {
		name           string
		act            func(hub *Hub)
		wantNow        int
		wantEventually int
	}{
		{
			name: "a lone bump is sent at once and never repeated",
			act: func(hub *Hub) {
				hub.BumpSidebarActivity("ships")
			},
			wantNow:        1,
			wantEventually: 1,
		},
		{
			name: "a burst sends one leading and one trailing message",
			act: func(hub *Hub) {
				for range 5 {
					hub.BumpSidebarActivity("ships")
				}
			},
			wantNow:        1,
			wantEventually: 2,
		},
		{
			name: "different keys are throttled independently",
			act: func(hub *Hub) {
				hub.BumpSidebarActivity("ships")
				hub.BumpSidebarActivity("journals")
			},
			wantNow:        2,
			wantEventually: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			hub := NewHub()
			hub.sidebar = coalesce.NewKeyed(interval, coalesce.Latest[time.Time], hub.broadcastSidebarActivity)
			viewer := NewClient(uuid.New(), nil)
			hub.clients[viewer.UserID] = []*Client{viewer}

			// when
			tt.act(hub)

			// then
			assert.Equal(t, tt.wantNow, len(viewer.send), "leading sends go out immediately")
			require.Eventually(t, func() bool {
				return len(viewer.send) == tt.wantEventually
			}, 10*interval, interval/10)
			assert.Never(t, func() bool {
				return len(viewer.send) > tt.wantEventually
			}, 3*interval, interval/10)

			for range tt.wantEventually {
				var got Message
				require.NoError(t, json.Unmarshal(<-viewer.send, &got))
				assert.Equal(t, "sidebar_activity", got.Type)
			}
		})
	}
}

func TestHub_BroadcastSidebarActivity_CarriesTheKeyAndTimestamp(t *testing.T) {
	// given
	hub := NewHub()
	viewer := NewClient(uuid.New(), nil)
	hub.clients[viewer.UserID] = []*Client{viewer}
	at := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	// when
	hub.broadcastSidebarActivity(context.Background(), "ships", at)

	// then
	var got struct {
		Type string `json:"type"`
		Data struct {
			Key string `json:"key"`
			At  string `json:"at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(<-viewer.send, &got))
	assert.Equal(t, "sidebar_activity", got.Type)
	assert.Equal(t, "ships", got.Data.Key)
	assert.Equal(t, at.Format(time.RFC3339), got.Data.At)
}

func TestClient_FramePayload(t *testing.T) {
	tests := []struct {
		name          string
		batch         bool
		queued        int
		size          int
		wantArray     bool
		wantItems     int
		wantRemaining int
	}{
		{name: "a client without batching gets one message per frame", batch: false, queued: 3, size: 10, wantArray: false, wantItems: 1, wantRemaining: 2},
		{name: "a batching client with nothing else queued gets the bare message", batch: true, queued: 1, size: 10, wantArray: false, wantItems: 1, wantRemaining: 0},
		{name: "a batching client gets everything queued folded into one array", batch: true, queued: 5, size: 10, wantArray: true, wantItems: 5, wantRemaining: 0},
		{name: "a batching client gets a full buffer in one frame", batch: true, queued: sendBufferSize, size: 10, wantArray: true, wantItems: sendBufferSize, wantRemaining: 0},
		{name: "a batch stops once it reaches the byte cap", batch: true, queued: 10, size: 10 * 1024, wantArray: true, wantItems: 7, wantRemaining: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			client := NewClient(uuid.New(), nil)
			client.batch = tt.batch
			queued := make([][]byte, 0, tt.queued)
			for n := range tt.queued {
				data, err := json.Marshal(Message{Type: "post_like", Data: map[string]any{"n": n, "pad": strings.Repeat("a", tt.size)}})
				require.NoError(t, err)
				queued = append(queued, data)
				client.send <- data
			}

			// when
			frame := client.framePayload(<-client.send)

			// then
			assert.Equal(t, tt.wantRemaining, len(client.send))
			if !tt.wantArray {
				assert.Equal(t, queued[0], frame, "the frame is the bare message, never an array")
				return
			}

			var items []json.RawMessage
			require.NoError(t, json.Unmarshal(frame, &items), "a batched frame must be a JSON array")
			require.Len(t, items, tt.wantItems)
			for i, item := range items {
				assert.JSONEq(t, string(queued[i]), string(item), "items keep their queue order")
			}
		})
	}
}
