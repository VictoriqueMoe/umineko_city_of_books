package ws

import (
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
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

func TestReplyPong(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "the probe nonce is echoed so a client can match the reply to its own probe",
			data: `{"nonce":42}`,
			want: `{"type":"pong","data":{"nonce":42}}`,
		},
		{
			name: "a keepalive ping with no nonce is answered without one",
			data: `{}`,
			want: `{"type":"pong","data":{}}`,
		},
		{
			name: "an absent payload is answered without a nonce",
			data: "",
			want: `{"type":"pong","data":{}}`,
		},
		{
			name: "an unreadable payload is answered without a nonce rather than dropped",
			data: `"not an object"`,
			want: `{"type":"pong","data":{}}`,
		},
		{
			name: "the reported round trip is not echoed back to the reporter",
			data: `{"nonce":7,"rtt":240}`,
			want: `{"type":"pong","data":{"nonce":7}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			client := NewClient(uuid.New(), nil)

			// when
			replyPong(client, parsePing(json.RawMessage(tt.data)))

			// then
			select {
			case got := <-client.send:
				assert.JSONEq(t, tt.want, string(got))
			default:
				t.Fatal("no pong was queued")
			}
		})
	}
}

func TestParsePing(t *testing.T) {
	tests := []struct {
		name string
		data string
		want pingData
	}{
		{
			name: "a reported round trip is kept",
			data: `{"nonce":3,"rtt":240}`,
			want: pingData{Nonce: 3, RTT: 240},
		},
		{
			name: "a round trip beyond anything playable is discarded rather than displayed",
			data: `{"nonce":3,"rtt":600000}`,
			want: pingData{Nonce: 3},
		},
		{
			name: "a negative round trip is discarded",
			data: `{"nonce":3,"rtt":-5}`,
			want: pingData{Nonce: 3},
		},
		{
			name: "an unreadable payload yields nothing rather than an error",
			data: `[1,2,3]`,
			want: pingData{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given the raw payload under test

			// when
			got := parsePing(json.RawMessage(tt.data))

			// then
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBoundReportedRTT(t *testing.T) {
	tests := []struct {
		name     string
		reported int
		measured int64
		want     int
	}{
		{
			name:     "an unmeasured client is taken at its word",
			reported: 400,
			measured: 0,
			want:     400,
		},
		{
			name:     "an honest report well inside the measured ceiling is kept",
			reported: 160,
			measured: 150,
			want:     160,
		},
		{
			name:     "jitter above a single measured sample is still allowed through",
			reported: 300,
			measured: 150,
			want:     300,
		},
		{
			name:     "a claim far beyond what the socket observed is clipped",
			reported: 4000,
			measured: 40,
			want:     180,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given the reported and measured round trips under test

			// when
			got := boundReportedRTT(tt.reported, tt.measured)

			// then
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClientRecordPong(t *testing.T) {
	t.Run("a round trip is not reported until a probe completes", func(t *testing.T) {
		// given
		client := NewClient(uuid.New(), nil)

		// when
		client.RecordPong()

		// then
		assert.Zero(t, client.MeasuredRTT())
	})

	t.Run("a spike is believed at once but decays over later probes", func(t *testing.T) {
		// given
		client := NewClient(uuid.New(), nil)

		// when
		client.pingAt.Store(time.Now().Add(-200 * time.Millisecond).UnixNano())
		client.RecordPong()
		spike := client.MeasuredRTT()

		client.pingAt.Store(time.Now().UnixNano())
		client.RecordPong()
		afterQuiet := client.MeasuredRTT()

		// then
		assert.GreaterOrEqual(t, spike, int64(200))
		assert.Less(t, afterQuiet, spike)
		assert.Positive(t, afterQuiet)
	})
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

			authzSvc := authz.NewMockService(t)
			authzSvc.EXPECT().IsBanned(mock.Anything, userID).Return(tt.banned).Maybe()

			// when
			err := stillAuthorised(mgr, authzSvc, "tok", userID)

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

func TestWatchPosts_ReplacesTheConnectionsWatchedPosts(t *testing.T) {
	// given
	kept := uuid.New()
	previous := uuid.New()

	tests := []struct {
		name string
		data string
		want []uuid.UUID
	}{
		{name: "valid ids are watched", data: `{"post_ids":["` + kept.String() + `"]}`, want: []uuid.UUID{kept}},
		{name: "unparsable ids are ignored", data: `{"post_ids":["nope","` + kept.String() + `",""]}`, want: []uuid.UUID{kept}},
		{name: "an empty array clears the set", data: `{"post_ids":[]}`, want: nil},
		{name: "a malformed payload leaves the set alone", data: `"nope"`, want: []uuid.UUID{previous}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			client := NewClient(uuid.New(), nil)
			hub.WatchPosts(client, []uuid.UUID{previous})

			// when
			handleWSMessage(client, incomingMessage{Type: "watch_posts", Data: json.RawMessage(tt.data)}, hub, nil, nil, nil)

			// then
			assert.ElementsMatch(t, tt.want, slices.Collect(maps.Keys(client.watchedPosts)))
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
