package chat

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAssertBlocksAllowParticipation(t *testing.T) {
	senderID := uuid.New()
	hostID := uuid.New()
	viewerID := uuid.New()

	tests := []struct {
		name    string
		room    *repository.ChatRoomSendContext
		members []uuid.UUID
		setup   func(m *testMocks)
		wantErr error
	}{
		{
			name:    "dm rejects when either side has blocked",
			room:    &repository.ChatRoomSendContext{Type: "dm"},
			members: []uuid.UUID{senderID, viewerID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, viewerID).Return(true, nil)
			},
			wantErr: ErrUserBlocked,
		},
		{
			name:    "dm allows when neither side has blocked",
			room:    &repository.ChatRoomSendContext{Type: "dm"},
			members: []uuid.UUID{senderID, viewerID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, senderID, viewerID).Return(false, nil)
			},
		},
		{
			name: "live stream rejects when the streamer has blocked the sender",
			room: &repository.ChatRoomSendContext{
				Type: "group", IsSystem: true, SystemKind: SystemKindLiveStream, CreatedBy: hostID,
			},
			members: []uuid.UUID{senderID, hostID, viewerID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlocked(mock.Anything, hostID, senderID).Return(true, nil)
			},
			wantErr: ErrBlockedByRoomHost,
		},
		{
			name: "live stream allows when the sender has blocked another viewer",
			room: &repository.ChatRoomSendContext{
				Type: "group", IsSystem: true, SystemKind: SystemKindLiveStream, CreatedBy: hostID,
			},
			members: []uuid.UUID{senderID, hostID, viewerID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlocked(mock.Anything, hostID, senderID).Return(false, nil)
			},
		},
		{
			name:    "live stream allows the streamer to post in their own room",
			room:    &repository.ChatRoomSendContext{Type: "group", IsSystem: true, SystemKind: SystemKindLiveStream, CreatedBy: senderID},
			members: []uuid.UUID{senderID, viewerID},
		},
		{
			name:    "group room rejects when the host has blocked the sender",
			room:    &repository.ChatRoomSendContext{Type: "group", CreatedBy: hostID},
			members: []uuid.UUID{senderID, hostID, viewerID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlocked(mock.Anything, hostID, senderID).Return(true, nil)
			},
			wantErr: ErrBlockedByRoomHost,
		},
		{
			name:    "group room allows when the host has not blocked the sender",
			room:    &repository.ChatRoomSendContext{Type: "group", CreatedBy: hostID},
			members: []uuid.UUID{senderID, hostID, viewerID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlocked(mock.Anything, hostID, senderID).Return(false, nil)
			},
		},
		{
			name:    "mods room ignores the seed creator's block list",
			room:    &repository.ChatRoomSendContext{Type: "group", IsSystem: true, SystemKind: SystemKindMods, CreatedBy: hostID},
			members: []uuid.UUID{senderID, hostID},
		},
		{
			name:    "admins room ignores the seed creator's block list",
			room:    &repository.ChatRoomSendContext{Type: "group", IsSystem: true, SystemKind: SystemKindAdmins, CreatedBy: hostID},
			members: []uuid.UUID{senderID, hostID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			// when
			err := svc.assertBlocksAllowParticipation(context.Background(), tt.room, senderID, tt.members)

			// then
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestMintVoiceToken_AppliesTheSameBlockGateAsSending(t *testing.T) {
	joinerID := uuid.New()
	hostID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name    string
		room    *repository.ChatRoomSendContext
		members []uuid.UUID
		setup   func(m *testMocks)
		wantErr error
	}{
		{
			name:    "a dm rejects the joiner when either side has blocked",
			room:    &repository.ChatRoomSendContext{Type: dto.RoomTypeDM},
			members: []uuid.UUID{joinerID, otherID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, joinerID, otherID).Return(true, nil)
			},
			wantErr: ErrUserBlocked,
		},
		{
			name:    "a group room rejects a joiner the host has blocked",
			room:    &repository.ChatRoomSendContext{Type: dto.RoomTypeGroup, CreatedBy: hostID},
			members: []uuid.UUID{joinerID, hostID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlocked(mock.Anything, hostID, joinerID).Return(true, nil)
			},
			wantErr: ErrBlockedByRoomHost,
		},
		{
			name:    "a live stream rejects a viewer the streamer has blocked",
			room:    &repository.ChatRoomSendContext{Type: dto.RoomTypeGroup, IsSystem: true, SystemKind: SystemKindLiveStream, CreatedBy: hostID},
			members: []uuid.UUID{joinerID, hostID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlocked(mock.Anything, hostID, joinerID).Return(true, nil)
			},
			wantErr: ErrBlockedByRoomHost,
		},
		{
			name:    "a group room admits a joiner the host has not blocked",
			room:    &repository.ChatRoomSendContext{Type: dto.RoomTypeGroup, CreatedBy: hostID},
			members: []uuid.UUID{joinerID, hostID},
			setup: func(m *testMocks) {
				m.blockSvc.EXPECT().IsBlocked(mock.Anything, hostID, joinerID).Return(false, nil)
			},
		},
		{
			name:    "the mods room ignores the seed creator's block list",
			room:    &repository.ChatRoomSendContext{Type: dto.RoomTypeGroup, IsSystem: true, SystemKind: SystemKindMods, CreatedBy: hostID},
			members: []uuid.UUID{joinerID, hostID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			expectVoiceConfigured(m, true)
			roomID := uuid.New()
			tt.room.ID = roomID

			m.chatRepo.EXPECT().GetRoomSendContext(mock.Anything, roomID).Return(tt.room, nil)
			m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, joinerID).Return(true, nil)
			m.chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(tt.members, nil)
			if tt.setup != nil {
				tt.setup(m)
			}
			if tt.wantErr == nil {
				m.chatRepo.EXPECT().IsVoiceForceMuted(mock.Anything, roomID, joinerID).Return(false, nil)
				m.userRepo.EXPECT().GetByID(mock.Anything, joinerID).Return(sampleUser(joinerID), nil)
			}

			// when
			token, _, err := svc.MintVoiceToken(context.Background(), roomID, joinerID)

			// then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			verifier, err := auth.ParseAPIToken(token)
			require.NoError(t, err)
			require.Equal(t, joinerID.String(), verifier.Identity())
		})
	}
}
