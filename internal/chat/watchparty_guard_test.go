package chat

import (
	"context"
	"database/sql"
	"testing"

	"umineko_city_of_books/internal/livekit"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	participantActive  = "active"
	participantLeft    = "left"
	participantMissing = "missing"
)

func participantRow(sessionID, userID uuid.UUID, state string) *repository.ChatWatchPartyParticipantRow {
	switch state {
	case participantMissing:
		return nil

	case participantLeft:
		return &repository.ChatWatchPartyParticipantRow{
			SessionID: sessionID,
			UserID:    userID,
			LeftAt:    sql.NullString{Valid: true, String: "2026-07-29T10:00:00Z"},
		}

	default:
		return &repository.ChatWatchPartyParticipantRow{SessionID: sessionID, UserID: userID}
	}
}

func TestWatchPartyMutations_RequireActiveRoomMembership(t *testing.T) {
	cases := []struct {
		name  string
		setup func(m *testMocks)
		call  func(svc *service, roomID, sessionID, callerID, targetID uuid.UUID) error
	}{
		{
			name: "force mute session voice",
			setup: func(m *testMocks) {
				expectVoiceConfigured(m, true)
			},
			call: func(svc *service, roomID, sessionID, callerID, targetID uuid.UUID) error {
				return svc.ForceMuteSessionVoice(context.Background(), roomID, sessionID, callerID, targetID, true)
			},
		},
		{
			name: "grant watch party control",
			setup: func(m *testMocks) {
				m.hyperbeamSvc.EXPECT().Enabled().Return(true)
			},
			call: func(svc *service, roomID, sessionID, callerID, targetID uuid.UUID) error {
				return svc.GrantWatchPartyControl(context.Background(), roomID, sessionID, callerID, targetID)
			},
		},
		{
			name:  "end watch party",
			setup: func(m *testMocks) {},
			call: func(svc *service, roomID, sessionID, callerID, targetID uuid.UUID) error {
				return svc.EndWatchParty(context.Background(), roomID, sessionID, callerID, "manual")
			},
		},
		{
			name:  "kick watch party participant",
			setup: func(m *testMocks) {},
			call: func(svc *service, roomID, sessionID, callerID, targetID uuid.UUID) error {
				return svc.KickWatchPartyParticipant(context.Background(), roomID, sessionID, callerID, targetID)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a caller who has been removed from the chat room but still holds a session id
			svc, m := newTestService(t)
			roomID := uuid.New()
			sessionID := uuid.New()
			callerID := uuid.New()
			targetID := uuid.New()

			tc.setup(m)
			m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, callerID).Return(false, nil)

			// when they invoke the mutating watch party method
			err := tc.call(svc, roomID, sessionID, callerID, targetID)

			// then the room membership guard rejects them before the session is touched
			require.ErrorIs(t, err, ErrNotMember)
		})
	}
}

func TestLeaveWatchParty_NonMemberStillLeaves(t *testing.T) {
	// given a participant who is no longer a room member but still has an open participant row
	svc, m := newTestService(t)
	roomID := uuid.New()
	sessionID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(&repository.ChatWatchPartySessionRow{
		ID: sessionID, RoomID: roomID, StartedBy: ownerID, ControllerID: ownerID, Status: "active",
	}, nil)
	m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, memberID).Return(participantRow(sessionID, memberID, participantActive), nil)
	m.watchPartyRepo.EXPECT().MarkParticipantLeft(mock.Anything, sessionID, memberID).Return(nil)
	m.chatRepo.EXPECT().RemoveMember(mock.Anything, sessionID, memberID).Return(nil)

	// when they leave the party
	err := svc.LeaveWatchParty(context.Background(), roomID, sessionID, memberID)

	// then leaving is never gated on membership, so the row is closed instead of stranded
	require.NoError(t, err)
}

func TestForceMuteSessionVoice_RequiresActiveParticipants(t *testing.T) {
	cases := []struct {
		name   string
		caller string
		target string
	}{
		{name: "caller never joined the party", caller: participantMissing, target: participantActive},
		{name: "caller already left the party", caller: participantLeft, target: participantActive},
		{name: "target never joined the party", caller: participantActive, target: participantMissing},
		{name: "target already left the party", caller: participantActive, target: participantLeft},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a live screen share party in a room the caller belongs to
			svc, m := newTestService(t)
			roomID := uuid.New()
			sessionID := uuid.New()
			callerID := uuid.New()
			targetID := uuid.New()

			expectVoiceConfigured(m, true)
			m.chatRepo.EXPECT().IsMember(mock.Anything, roomID, callerID).Return(true, nil)
			m.watchPartyRepo.EXPECT().GetByID(mock.Anything, sessionID).Return(&repository.ChatWatchPartySessionRow{
				ID: sessionID, RoomID: roomID, StartedBy: callerID, ControllerID: callerID, Status: "active", Type: watchPartyTypeScreenShare,
			}, nil)
			m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, callerID).Return(participantRow(sessionID, callerID, tc.caller), nil)
			if tc.caller == participantActive {
				m.watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, targetID).Return(participantRow(sessionID, targetID, tc.target), nil)
			}

			// when the caller tries to force mute
			err := svc.ForceMuteSessionVoice(context.Background(), roomID, sessionID, callerID, targetID, true)

			// then the publish permissions are never touched
			require.ErrorIs(t, err, ErrWatchPartyNotParticipant)
		})
	}
}

func TestClearWatchPartyParticipation(t *testing.T) {
	cases := []struct {
		name           string
		participant    string
		wantMarkedLeft bool
	}{
		{name: "open participant row is closed", participant: participantActive, wantMarkedLeft: true},
		{name: "row that already left is untouched", participant: participantLeft, wantMarkedLeft: false},
		{name: "user was never in the party", participant: participantMissing, wantMarkedLeft: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given an active watch party in the room the user is being evicted from
			watchPartyRepo := repository.NewMockChatWatchPartyRepository(t)
			chatRepo := repository.NewMockChatRepository(t)
			lk := livekit.NewMockService(t)
			c := &core{
				chatRepo:       chatRepo,
				watchPartyRepo: watchPartyRepo,
				livekitSvc:     lk,
				hub:            ws.NewHub(),
			}
			roomID := uuid.New()
			sessionID := uuid.New()
			userID := uuid.New()
			markedLeft := false

			watchPartyRepo.EXPECT().ListActiveByRoom(mock.Anything, roomID).Return([]repository.ChatWatchPartySessionRow{
				{ID: sessionID, RoomID: roomID, Status: "active"},
			}, nil)
			watchPartyRepo.EXPECT().GetParticipant(mock.Anything, sessionID, userID).Return(participantRow(sessionID, userID, tc.participant), nil)
			if tc.wantMarkedLeft {
				watchPartyRepo.EXPECT().MarkParticipantLeft(mock.Anything, sessionID, userID).
					Run(func(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) { markedLeft = true }).
					Return(nil)
				lk.EXPECT().RemoveParticipant(mock.Anything, voiceSessionRoomPrefix+sessionID.String(), userID.String()).Return(nil)
				chatRepo.EXPECT().RemoveMember(mock.Anything, sessionID, userID).Return(nil)
			}

			// when the eviction clears their watch party participation
			c.clearWatchPartyParticipation(context.Background(), roomID, userID)

			// then only an open row is closed, and closing it drops both the live party call and party chat access
			require.Equal(t, tc.wantMarkedLeft, markedLeft)
		})
	}
}

func TestEvictUserFromRoom_DropsLiveKitSession(t *testing.T) {
	// given a member of a room with no active watch parties
	chatRepo := repository.NewMockChatRepository(t)
	watchPartyRepo := repository.NewMockChatWatchPartyRepository(t)
	lk := livekit.NewMockService(t)
	svc := &moderationService{core: &core{
		chatRepo:       chatRepo,
		watchPartyRepo: watchPartyRepo,
		livekitSvc:     lk,
		hub:            ws.NewHub(),
	}}
	roomID := uuid.New()
	targetID := uuid.New()

	chatRepo.EXPECT().GetRoomMembers(mock.Anything, roomID).Return(nil, nil)
	chatRepo.EXPECT().RemoveMember(mock.Anything, roomID, targetID).Return(nil)
	watchPartyRepo.EXPECT().ListActiveByRoom(mock.Anything, roomID).Return(nil, nil)
	lk.EXPECT().RemoveParticipant(mock.Anything, roomID.String(), targetID.String()).Return(nil)

	// when the user is evicted
	err := svc.evictUserFromRoom(context.Background(), roomID, targetID, "banned")

	// then their live voice session is revoked instead of surviving on their old token
	require.NoError(t, err)
}
