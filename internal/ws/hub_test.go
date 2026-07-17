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
