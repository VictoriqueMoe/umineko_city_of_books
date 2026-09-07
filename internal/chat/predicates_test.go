package chat

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPubliclyVisibleRoom(t *testing.T) {
	tests := []struct {
		name       string
		roomType   dto.RoomType
		isPublic   bool
		isSystem   bool
		wantHidden bool
	}{
		{name: "a public group room is visible", roomType: dto.RoomTypeGroup, isPublic: true},
		{name: "a private group room is hidden", roomType: dto.RoomTypeGroup, wantHidden: true},
		{name: "a public system room is hidden", roomType: dto.RoomTypeGroup, isPublic: true, isSystem: true, wantHidden: true},
		{name: "a dm is hidden even when flagged public", roomType: dto.RoomTypeDM, isPublic: true, wantHidden: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given the same room described by both repository shapes
			row := &model.ChatRoomRow{Type: tt.roomType, IsPublic: tt.isPublic, IsSystem: tt.isSystem}
			sendCtx := &model.ChatRoomSendContext{Type: tt.roomType, IsPublic: tt.isPublic, IsSystem: tt.isSystem}

			// when
			rowVisible := row.PubliclyVisible()
			sendCtxVisible := sendCtx.PubliclyVisible()

			// then both shapes agree, and the audit gate reads the same answer
			assert.Equal(t, !tt.wantHidden, rowVisible)
			assert.Equal(t, rowVisible, sendCtxVisible)
			assert.Equal(t, rowVisible, isAuditableRoom(row))
			assert.Equal(t, sendCtxVisible, isAuditableSendContext(sendCtx))
		})
	}
}

func TestPubliclyVisibleRoom_NilIsNeverVisible(t *testing.T) {
	// given
	var row *model.ChatRoomRow
	var sendCtx *model.ChatRoomSendContext

	// when / then
	assert.False(t, row.PubliclyVisible())
	assert.False(t, sendCtx.PubliclyVisible())
	assert.False(t, isAuditableRoom(row))
	assert.False(t, isAuditableSendContext(sendCtx))
}

func TestPlainMessageNotification(t *testing.T) {
	tests := []struct {
		name        string
		room        *model.ChatRoomSendContext
		wantType    dto.NotificationType
		wantMessage string
	}{
		{
			name:     "a dm carries no message so the dao can collapse it",
			room:     &model.ChatRoomSendContext{Type: dto.RoomTypeDM, Name: "ignored"},
			wantType: dto.NotifChatMessage,
		},
		{
			name:        "a group room carries the wording the dao strips to recover the room name",
			room:        &model.ChatRoomSendContext{Type: dto.RoomTypeGroup, Name: "General"},
			wantType:    dto.NotifChatRoomMessage,
			wantMessage: "sent a message in General",
		},
		{
			name:     "a missing room falls back to the direct message shape",
			room:     nil,
			wantType: dto.NotifChatMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given / when
			gotType, gotMessage := plainMessageNotification(tt.room)

			// then
			assert.Equal(t, tt.wantType, gotType)
			assert.Equal(t, tt.wantMessage, gotMessage)
		})
	}
}

func TestLockedAccountRuleReachesStaffOnly(t *testing.T) {
	senderID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name         string
		locked       bool
		room         *model.ChatRoomRow
		members      []uuid.UUID
		audienceRole role.Role
		wantErr      error
	}{
		{
			name:    "an unlocked sender is never questioned",
			room:    &model.ChatRoomRow{Type: dto.RoomTypeGroup},
			members: []uuid.UUID{senderID, otherID},
		},
		{
			name:         "a locked sender may speak in a dm with site staff",
			locked:       true,
			room:         &model.ChatRoomRow{Type: dto.RoomTypeDM},
			members:      []uuid.UUID{senderID, otherID},
			audienceRole: authz.RoleModerator,
		},
		{
			name:         "a locked sender may not speak in a dm with an ordinary member",
			locked:       true,
			room:         &model.ChatRoomRow{Type: dto.RoomTypeDM},
			members:      []uuid.UUID{senderID, otherID},
			audienceRole: role.Role("user"),
			wantErr:      ErrLockedNonStaffDM,
		},
		{
			name:    "a locked sender may not speak in a dm they are alone in",
			locked:  true,
			room:    &model.ChatRoomRow{Type: dto.RoomTypeDM},
			members: []uuid.UUID{senderID},
			wantErr: ErrLockedNonStaffDM,
		},
		{
			name:    "a locked sender may not speak in a group room",
			locked:  true,
			room:    &model.ChatRoomRow{Type: dto.RoomTypeGroup},
			members: []uuid.UUID{senderID, otherID},
			wantErr: ErrLockedNonStaffDM,
		},
		{
			name:    "a locked sender may not speak in the staff room either",
			locked:  true,
			room:    &model.ChatRoomRow{Type: dto.RoomTypeGroup, IsSystem: true, SystemKind: SystemKindMods},
			members: []uuid.UUID{senderID, otherID},
			wantErr: ErrLockedNonStaffDM,
		},
		{
			name:    "a missing room denies a locked sender",
			locked:  true,
			members: []uuid.UUID{senderID, otherID},
			wantErr: ErrLockedNonStaffDM,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given a room-id send and a first-ever dm to the same audience
			roomID := uuid.New()
			userRepo := repository.NewMockUserRepository(t)
			chatRepo := repository.NewMockChatRepository(t)
			authzSvc := authz.NewMockService(t)
			c := &core{userRepo: userRepo, chatRepo: chatRepo, authzSvc: authzSvc}

			userRepo.EXPECT().IsLocked(mock.Anything, senderID).Return(tt.locked, nil)
			chatRepo.EXPECT().GetRoomByID(mock.Anything, spec.ChatRoomViewer{RoomID: roomID, ViewerID: senderID}).Return(tt.room, nil).Maybe()
			chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(tt.members, nil).Maybe()
			if tt.audienceRole != "" {
				authzSvc.EXPECT().GetRoles(mock.Anything, []uuid.UUID{otherID}).
					Return(map[uuid.UUID]role.Role{otherID: tt.audienceRole}, nil).Maybe()
			}

			// when
			err := c.ensureLockAllowsRoom(context.Background(), senderID, roomID)

			// then
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}

			if tt.locked && tt.audienceRole == "" {
				return
			}

			// when the same audience is reached through the dm entry point instead
			dmErr := (&dmService{core: c}).ensureLockAllowsDMTo(context.Background(), senderID, otherID)

			// then both entry points answer alike
			if tt.wantErr == nil {
				require.NoError(t, dmErr)
				return
			}
			require.ErrorIs(t, dmErr, tt.wantErr)
		})
	}
}

func TestCapabilitiesFor(t *testing.T) {
	tests := []struct {
		name     string
		roomType dto.RoomType
		want     roomCapabilities
	}{
		{
			name:     "a pair has no room-level moderation, both participants pin, and it is the only kind the badge counts",
			roomType: dto.RoomTypeDM,
			want: roomCapabilities{
				participantsMayPin:     true,
				requiresRecipientOptIn: true,
				countsTowardsUnread:    true,
				pairwiseReadReceipts:   true,
			},
		},
		{
			name:     "a group room announces departures, is destroyable by a moderator and evicts on the word filter",
			roomType: dto.RoomTypeGroup,
			want: roomCapabilities{
				announcesDepartures:    true,
				destroyableByModerator: true,
				evictsOnBannedWord:     true,
			},
		},
		{
			name:     "an unnamed kind is treated as a room but is never destroyable by a moderator",
			roomType: dto.RoomType(""),
			want: roomCapabilities{
				announcesDepartures: true,
				evictsOnBannedWord:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given a room type

			// when its capabilities are resolved
			got := capabilitiesFor(tt.roomType)

			// then the descriptor matches the policy table in the plan
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSendContextCapabilities_MissingRoomGrantsNothing(t *testing.T) {
	// given no send context at all

	// when its capabilities are resolved
	got := sendContextCapabilities(nil)

	// then nothing is granted, so a lost room can never widen a policy
	assert.Equal(t, roomCapabilities{}, got)
}
