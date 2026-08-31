package ws

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestOriginAllowed(t *testing.T) {
	cases := []struct {
		name    string
		origin  string
		allowed string
		want    bool
	}{
		{"exact match", "https://meta.auaurora.moe", "https://meta.auaurora.moe", true},
		{"localhost dev", "http://localhost:4323", "http://localhost:4323", true},
		{"different host", "https://evil.example.com", "https://meta.auaurora.moe", false},
		{"different scheme", "http://meta.auaurora.moe", "https://meta.auaurora.moe", false},
		{"different port", "https://meta.auaurora.moe:8080", "https://meta.auaurora.moe", false},
		{"browser with trailing slash is still rejected", "https://meta.auaurora.moe/", "https://meta.auaurora.moe", false},
		{"allowed with trailing slash is tolerated", "https://meta.auaurora.moe", "https://meta.auaurora.moe/", true},
		{"allowed with trailing slash, dev", "http://localhost:5173", "http://localhost:5173/", true},
		{"empty origin rejected", "", "https://meta.auaurora.moe", false},
		{"empty allowed rejected", "https://meta.auaurora.moe", "", false},
		{"both empty rejected", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, OriginAllowed(tc.origin, tc.allowed))
		})
	}
}

func TestStillAuthorised(t *testing.T) {
	// given
	userID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name        string
		validateFor uuid.UUID
		validateErr error
		banned      bool
		wantErr     bool
	}{
		{name: "valid session for the same user passes", validateFor: userID},
		{name: "revoked session fails", validateErr: errors.New("no rows"), wantErr: true},
		{name: "expired session fails", validateErr: errors.New("session expired"), wantErr: true},
		{name: "token now belongs to another user fails", validateFor: otherID, wantErr: true},
		{name: "banned account fails even with a valid session", validateFor: userID, banned: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMockSessionRepository(t)
			settingsSvc := settings.NewMockService(t)
			mgr := session.NewManager(repo, settingsSvc)
			if tt.validateErr != nil {
				repo.EXPECT().GetUserID(mock.Anything, "tok").Return(uuid.Nil, time.Time{}, tt.validateErr)
			} else {
				repo.EXPECT().GetUserID(mock.Anything, "tok").Return(tt.validateFor, time.Now().Add(time.Hour), nil)
			}

			banChecker := NewMockBanChecker(t)
			banChecker.EXPECT().IsBanned(mock.Anything, userID).Return(tt.banned).Maybe()

			// when
			err := stillAuthorised(mgr, banChecker, "tok", userID)

			// then
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestSecretJoin_OnlyRegisteredSecretsCreateTopics(t *testing.T) {
	// given
	tests := []struct {
		name     string
		secretID string
		wantJoin bool
	}{
		{name: "registered secret joins", secretID: "witchHunter", wantJoin: true},
		{name: "unregistered id is ignored", secretID: "not-a-real-secret", wantJoin: false},
		{name: "random string is ignored", secretID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", wantJoin: false},
		{name: "empty id is ignored", secretID: "", wantJoin: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			userID := uuid.New()
			payload, err := json.Marshal(secretTopicData{SecretID: tt.secretID})
			require.NoError(t, err)

			// when
			handleWSMessage(NewClient(userID, nil), incomingMessage{Type: "secret_join", Data: payload}, hub, nil, nil, nil)

			// then
			joined := hub.IsUserInRoom(TopicUUID("secret:"+tt.secretID), userID)
			assert.Equal(t, tt.wantJoin, joined)
			assert.Equal(t, tt.wantJoin, len(hub.rooms) == 1, "unregistered ids must not allocate hub state")
		})
	}
}

func TestOnlyJoinRoomFallsBackToStoredMembership(t *testing.T) {
	// given
	tests := []struct {
		name        string
		messageType string
		payload     func(roomID uuid.UUID) any
		wantRepair  bool
	}{
		{
			name:        "join_room repairs a cold hub from the database",
			messageType: "join_room",
			payload:     func(roomID uuid.UUID) any { return roomActionData{RoomID: roomID.String()} },
			wantRepair:  true,
		},
		{
			name:        "typing never reaches the database",
			messageType: TypingMessageType,
			payload:     func(roomID uuid.UUID) any { return typingData{RoomID: roomID.String()} },
			wantRepair:  false,
		},
		{
			name:        "viewer_state never reaches the database",
			messageType: "viewer_state",
			payload: func(roomID uuid.UUID) any {
				return viewerStateData{RoomID: roomID.String(), State: ViewerStateIdle}
			},
			wantRepair: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			userID := uuid.New()
			roomID := uuid.New()
			checker := NewMockRoomMembershipChecker(t)
			if tt.wantRepair {
				checker.EXPECT().IsRoomMember(mock.Anything, roomID, userID).Return(true, nil)
			}
			payload, err := json.Marshal(tt.payload(roomID))
			require.NoError(t, err)

			// when
			handleWSMessage(NewClient(userID, nil), incomingMessage{Type: tt.messageType, Data: payload}, hub, checker, nil, nil)

			// then
			assert.Equal(t, tt.wantRepair, hub.IsUserInRoom(roomID, userID),
				"only join_room may repair hub membership from the database")
		})
	}
}

func TestJoinRoomMembershipFallbackOutcomes(t *testing.T) {
	// given
	tests := []struct {
		name       string
		checker    func(t *testing.T, roomID, userID uuid.UUID) RoomMembershipChecker
		wantJoined bool
		wantViewer bool
	}{
		{
			name: "member is joined and starts viewing",
			checker: func(t *testing.T, roomID, userID uuid.UUID) RoomMembershipChecker {
				checker := NewMockRoomMembershipChecker(t)
				checker.EXPECT().IsRoomMember(mock.Anything, roomID, userID).Return(true, nil)

				return checker
			},
			wantJoined: true,
			wantViewer: true,
		},
		{
			name: "non member is silently ignored",
			checker: func(t *testing.T, roomID, userID uuid.UUID) RoomMembershipChecker {
				checker := NewMockRoomMembershipChecker(t)
				checker.EXPECT().IsRoomMember(mock.Anything, roomID, userID).Return(false, nil)

				return checker
			},
			wantJoined: false,
			wantViewer: false,
		},
		{
			name: "lookup failure is silently ignored",
			checker: func(t *testing.T, roomID, userID uuid.UUID) RoomMembershipChecker {
				checker := NewMockRoomMembershipChecker(t)
				checker.EXPECT().IsRoomMember(mock.Anything, roomID, userID).Return(false, errors.New("boom"))

				return checker
			},
			wantJoined: false,
			wantViewer: false,
		},
		{
			name: "nil checker keeps the old behaviour",
			checker: func(t *testing.T, roomID, userID uuid.UUID) RoomMembershipChecker {
				return nil
			},
			wantJoined: false,
			wantViewer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			userID := uuid.New()
			roomID := uuid.New()
			payload, err := json.Marshal(roomActionData{RoomID: roomID.String()})
			require.NoError(t, err)

			// when
			handleWSMessage(NewClient(userID, nil), incomingMessage{Type: "join_room", Data: payload}, hub, tt.checker(t, roomID, userID), nil, nil)

			// then
			assert.Equal(t, tt.wantJoined, hub.IsUserInRoom(roomID, userID))
			assert.Equal(t, tt.wantViewer, hub.IsUserViewing(roomID, userID))
		})
	}
}

func TestJoinRoomSkipsLookupWhenHubAlreadyKnowsMembership(t *testing.T) {
	// given
	hub := NewHub()
	userID := uuid.New()
	roomID := uuid.New()
	hub.JoinRoom(roomID, userID)
	checker := NewMockRoomMembershipChecker(t)
	payload, err := json.Marshal(roomActionData{RoomID: roomID.String()})
	require.NoError(t, err)

	// when
	handleWSMessage(NewClient(userID, nil), incomingMessage{Type: "join_room", Data: payload}, hub, checker, nil, nil)

	// then
	assert.True(t, hub.IsUserViewing(roomID, userID))
	checker.AssertNotCalled(t, "IsRoomMember", mock.Anything, mock.Anything, mock.Anything)
}
