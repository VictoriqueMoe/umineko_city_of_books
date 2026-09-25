package daotest

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type (
	ChatRoomOpt func(*chatRoomOpts)

	chatRoomOpts struct {
		room    spec.NewChatRoom
		members []uuid.UUID
	}
)

func WithRoomName(name string) ChatRoomOpt {
	return func(o *chatRoomOpts) {
		o.room.Name = name
	}
}

func WithPublicRoom() ChatRoomOpt {
	return func(o *chatRoomOpts) {
		o.room.IsPublic = true
	}
}

func WithRPRoom() ChatRoomOpt {
	return func(o *chatRoomOpts) {
		o.room.IsRP = true
	}
}

func WithRoomMembers(userIDs ...uuid.UUID) ChatRoomOpt {
	return func(o *chatRoomOpts) {
		o.members = append(o.members, userIDs...)
	}
}

func CreateChatRoom(t *testing.T, repos *repository.Repositories, ownerID uuid.UUID, opts ...ChatRoomOpt) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	o := chatRoomOpts{
		room: spec.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: ownerID},
	}
	for _, opt := range opts {
		opt(&o)
	}

	room, err := repos.Chat.CreateRoom(ctx, o.room)
	require.NoError(t, err)

	for _, userID := range o.members {
		require.NoError(t, repos.Chat.AddMember(ctx, spec.ChatMemberRef{RoomID: room.ID, UserID: userID}))
	}

	return room.ID
}

func CreateDMRoom(t *testing.T, repos *repository.Repositories, userA, userB uuid.UUID) uuid.UUID {
	t.Helper()
	room, err := repos.Chat.CreateDMRoomAtomic(context.Background(), spec.ChatDMPair{UserA: userA, UserB: userB})
	require.NoError(t, err)

	return room.ID
}

func SendChatMessage(t *testing.T, repos *repository.Repositories, roomID, senderID uuid.UUID, body string) uuid.UUID {
	t.Helper()
	msg, err := repos.Chat.InsertMessageAndMarkRead(context.Background(), spec.NewChatMessage{RoomID: roomID, SenderID: senderID, Body: body})
	require.NoError(t, err)

	return msg.ID
}

func BackdateChatMessage(t *testing.T, repos *repository.Repositories, messageID uuid.UUID, createdAt string) {
	t.Helper()
	_, err := repos.DB().ExecContext(context.Background(), `UPDATE chat_messages SET created_at = $1 WHERE id = $2`, createdAt, messageID)
	require.NoError(t, err)
}

func BackdateChatRoomJoins(t *testing.T, repos *repository.Repositories, roomID uuid.UUID, joinedAt string) {
	t.Helper()
	_, err := repos.DB().ExecContext(context.Background(), `UPDATE chat_room_members SET joined_at = $1 WHERE room_id = $2`, joinedAt, roomID)
	require.NoError(t, err)
}
