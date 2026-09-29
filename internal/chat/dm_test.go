package chat

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func dmExpectRecipientReachable(m *testMocks, senderID, recipientID uuid.UUID) {
	m.userRepo.EXPECT().GetByID(mock.Anything, recipientID).Return(sampleUser(recipientID), nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, recipientID).Return(false, nil)
}

func TestResolveDMRoom_Rejections(t *testing.T) {
	errDB := errors.New("db")
	cases := []struct {
		name    string
		toSelf  bool
		given   func(m *testMocks, senderID, recipientID uuid.UUID)
		wantErr error
	}{
		{name: "cannot DM self", toSelf: true, wantErr: ErrCannotDMSelf},
		{
			name: "recipient lookup error",
			given: func(m *testMocks, _, recipientID uuid.UUID) {
				m.userRepo.EXPECT().GetByID(mock.Anything, recipientID).Return(nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "recipient not found",
			given: func(m *testMocks, _, recipientID uuid.UUID) {
				m.userRepo.EXPECT().GetByID(mock.Anything, recipientID).Return(nil, nil)
			},
			wantErr: ErrUserNotFound,
		},
		{
			name: "recipient has DMs disabled",
			given: func(m *testMocks, _, recipientID uuid.UUID) {
				recipient := sampleUser(recipientID)
				recipient.DmsEnabled = false
				m.userRepo.EXPECT().GetByID(mock.Anything, recipientID).Return(recipient, nil)
			},
			wantErr: ErrDmsDisabled,
		},
		{
			name: "pair is blocked",
			given: func(m *testMocks, senderID, recipientID uuid.UUID) {
				m.userRepo.EXPECT().GetByID(mock.Anything, recipientID).Return(sampleUser(recipientID), nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, recipientID).Return(true, nil)
			},
			wantErr: ErrUserBlocked,
		},
		{
			name: "find DM room error",
			given: func(m *testMocks, senderID, recipientID uuid.UUID) {
				dmExpectRecipientReachable(m, senderID, recipientID)
				m.chatRepo.EXPECT().FindDMRoom(mock.Anything, spec.ChatDMPair{UserA: senderID, UserB: recipientID}).Return(uuid.Nil, errDB)
			},
			wantErr: errDB,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			senderID := uuid.New()
			recipientID := uuid.New()
			if tc.toSelf {
				recipientID = senderID
			}

			if tc.given != nil {
				tc.given(m, senderID, recipientID)
			}

			// when
			got, err := svc.ResolveDMRoom(context.Background(), senderID, recipientID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
		})
	}
}

func TestResolveDMRoom_NoExistingRoom(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	recipientID := uuid.New()
	dmExpectRecipientReachable(m, senderID, recipientID)
	m.chatRepo.EXPECT().FindDMRoom(mock.Anything, spec.ChatDMPair{UserA: senderID, UserB: recipientID}).Return(uuid.Nil, nil)

	// when
	got, err := svc.ResolveDMRoom(context.Background(), senderID, recipientID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Nil(t, got.Room)
	assert.Equal(t, recipientID, got.Recipient.ID)
}

func TestResolveDMRoom_ExistingRoomRepairsHubMembership(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	recipientID := uuid.New()
	roomID := uuid.New()
	dmExpectRecipientReachable(m, senderID, recipientID)
	m.chatRepo.EXPECT().FindDMRoom(mock.Anything, spec.ChatDMPair{UserA: senderID, UserB: recipientID}).Return(roomID, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: senderID}).Return(&model.ChatRoomRow{ID: roomID, Type: "dm"}, nil)
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, recipientID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)

	// when
	got, err := svc.ResolveDMRoom(context.Background(), senderID, recipientID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got.Room)
	assert.Equal(t, roomID, got.Room.ID)
	assert.True(t, m.hub.IsUserInRoom(roomID, senderID))
	assert.True(t, m.hub.IsUserInRoom(roomID, recipientID))
}

func TestSendDMMessage_Rejections(t *testing.T) {
	errBoom := errors.New("boom")
	cases := []struct {
		name    string
		body    string
		toSelf  bool
		given   func(m *testMocks, senderID, recipientID uuid.UUID)
		wantErr error
	}{
		{name: "empty body with no files", wantErr: ErrMissingFields},
		{name: "DM precondition fails", body: "hi", toSelf: true, wantErr: ErrCannotDMSelf},
		{
			name: "create room error",
			body: "hi",
			given: func(m *testMocks, senderID, recipientID uuid.UUID) {
				dmExpectRecipientReachable(m, senderID, recipientID)
				m.chatRepo.EXPECT().CreateDMRoomAtomic(mock.Anything, spec.ChatDMPair{UserA: senderID, UserB: recipientID}).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			senderID := uuid.New()
			recipientID := uuid.New()
			if tc.toSelf {
				recipientID = senderID
			}

			if tc.given != nil {
				tc.given(m, senderID, recipientID)
			}

			// when
			got, err := svc.SendDMMessage(context.Background(), senderID, recipientID, tc.body, nil)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got)
		})
	}
}

func TestSendDMMessage_FirstEverDMJoinsBothPartiesToTheHub(t *testing.T) {
	// given
	svc, m := newTestService(t)
	senderID := uuid.New()
	recipientID := uuid.New()
	roomID := uuid.New()
	dmExpectRecipientReachable(m, senderID, recipientID)
	m.userRepo.EXPECT().GetByID(mock.Anything, senderID).Return(sampleUser(senderID), nil)
	m.chatRepo.EXPECT().CreateDMRoomAtomic(mock.Anything, spec.ChatDMPair{UserA: senderID, UserB: recipientID}).Return(&model.ChatRoomRow{ID: roomID, Type: "dm"}, nil)
	m.chatRepo.EXPECT().IsMember(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: senderID}).Return(true, nil)
	m.chatRepo.EXPECT().GetMemberTimeoutState(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: senderID}).Return(false, "", false, nil)
	m.bannedWordRepo.EXPECT().ListApplicable(mock.Anything, roomID).Return(nil, nil).Maybe()
	m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return([]uuid.UUID{senderID, recipientID}, nil)
	m.chatRepo.EXPECT().InsertMessageAndMarkRead(mock.Anything, spec.NewChatMessage{RoomID: roomID, SenderID: senderID, Body: "hi"}).Return(&model.ChatMessageRow{ID: uuid.New()}, nil)
	m.chatRepo.EXPECT().GetRoomMembersDetailed(mock.Anything, roomID).Return(nil, nil)
	m.vanityRoleRepo.EXPECT().GetRolesForUser(mock.Anything, senderID).Return(nil, nil)
	m.chatRepo.EXPECT().GetRoomSendContext(mock.Anything, roomID).Return(&model.ChatRoomSendContext{Type: "dm"}, nil)
	m.userRepo.EXPECT().GetByIDs(mock.Anything, []uuid.UUID{recipientID}).Return([]model.User{*sampleUser(recipientID)}, nil)
	m.chatRepo.EXPECT().IsMuted(mock.Anything, spec.ChatMemberRef{RoomID: roomID, UserID: recipientID}).Return(false, nil)
	m.chatRepo.EXPECT().CountUnreadRoomsForUser(mock.Anything, recipientID).Return(1, nil)
	m.chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: senderID}).Return(&model.ChatRoomRow{ID: roomID, Type: "dm"}, nil)

	require.False(t, m.hub.IsUserInRoom(roomID, senderID), "the room cannot be in the hub before it exists")
	require.False(t, m.hub.IsUserInRoom(roomID, recipientID), "the room cannot be in the hub before it exists")

	// when
	got, err := svc.SendDMMessage(context.Background(), senderID, recipientID, "hi", nil)
	svc.sideEffectsWG.Wait()

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, got.Room.ID)
	assert.True(t, m.hub.IsUserInRoom(roomID, senderID), "the sender must be in hub.rooms so join_room is accepted on their live socket")
	assert.True(t, m.hub.IsUserInRoom(roomID, recipientID), "the recipient must be in hub.rooms so join_room is accepted on their live socket")
}
