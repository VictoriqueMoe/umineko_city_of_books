package chat

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateStreamRoom_CreatesRoomAndHost(t *testing.T) {
	// given
	svc, m := newTestService(t)
	streamID := uuid.New()
	streamerID := uuid.New()
	m.chatRepo.EXPECT().CreateSystemRoomWithHost(mock.Anything, repository.NewChatSystemRoom{
		ID:         streamID,
		Name:       "My stream",
		SystemKind: SystemKindLiveStream,
		CreatedBy:  streamerID,
	}).Return(&repository.ChatRoomRow{ID: uuid.New()}, nil)

	// when
	err := svc.CreateStreamRoom(context.Background(), streamID, streamerID, "My stream")

	// then
	require.NoError(t, err)
}

func TestDeleteStreamRoom_DeletesRoomThenMediaFiles(t *testing.T) {
	// given
	svc, m := newTestService(t)
	streamID := uuid.New()
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/a.webp", "/uploads/b.jpg"}).Return()
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, streamID).Return(nil, nil)
	m.chatRepo.EXPECT().DeleteRoomWithMessages(mock.Anything, streamID).Return([]string{"/uploads/a.webp", "/uploads/b.jpg"}, nil)

	// when
	err := svc.DeleteStreamRoom(context.Background(), streamID)

	// then
	require.NoError(t, err)
}

func TestJoinStreamChat_RejectsNonStreamRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	streamID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, streamID, userID).Return(&repository.ChatRoomRow{IsSystem: false}, nil)

	// when
	err := svc.JoinStreamChat(context.Background(), streamID, userID)

	// then
	require.ErrorIs(t, err, ErrRoomNotFound)
}

func TestJoinStreamChat_AddsNewMember(t *testing.T) {
	// given
	svc, m := newTestService(t)
	streamID := uuid.New()
	userID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, streamID, userID).Return(&repository.ChatRoomRow{IsSystem: true, SystemKind: SystemKindLiveStream}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, streamID, userID).Return(false, nil)
	m.chatRepo.EXPECT().AddMemberWithRole(mock.Anything, streamID, userID, "member", false).Return(nil)

	// when
	err := svc.JoinStreamChat(context.Background(), streamID, userID)

	// then
	require.NoError(t, err)
}

func TestJoinStreamChat_PreservesExistingHostRole(t *testing.T) {
	// given
	svc, m := newTestService(t)
	streamID := uuid.New()
	hostID := uuid.New()
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, streamID, hostID).Return(&repository.ChatRoomRow{IsSystem: true, SystemKind: SystemKindLiveStream}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, streamID, hostID).Return(true, nil)

	// when
	err := svc.JoinStreamChat(context.Background(), streamID, hostID)

	// then
	require.NoError(t, err)
}
