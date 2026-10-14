package ws

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMysteryJoin_OnlyValidIDsCreateTopics(t *testing.T) {
	mysteryID := uuid.New()

	tests := []struct {
		name      string
		mysteryID string
		wantJoin  bool
	}{
		{name: "a mystery id joins its topic", mysteryID: mysteryID.String(), wantJoin: true},
		{name: "an upper case id joins the same topic", mysteryID: strings.ToUpper(mysteryID.String()), wantJoin: true},
		{name: "a non uuid id is ignored", mysteryID: "not-a-mystery", wantJoin: false},
		{name: "an empty id is ignored", mysteryID: "", wantJoin: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			hub := NewHub()
			userID := uuid.New()
			payload, err := json.Marshal(mysteryTopicData{MysteryID: tt.mysteryID})
			require.NoError(t, err)

			// when
			handleWSMessage(NewClient(userID, nil), incomingMessage{Type: "mystery_join", Data: payload}, hub, nil, nil, nil)

			// then
			joined := hub.IsUserInRoom(TopicUUID("mystery:"+mysteryID.String()), userID)
			assert.Equal(t, tt.wantJoin, joined)
			assert.Equal(t, tt.wantJoin, len(hub.rooms) == 1, "invalid ids must not allocate hub state")
		})
	}
}

func TestMysteryLeave_RemovesTheViewerFromTheTopic(t *testing.T) {
	// given
	hub := NewHub()
	userID := uuid.New()
	mysteryID := uuid.New()
	payload, err := json.Marshal(mysteryTopicData{MysteryID: mysteryID.String()})
	require.NoError(t, err)
	handleWSMessage(NewClient(userID, nil), incomingMessage{Type: "mystery_join", Data: payload}, hub, nil, nil, nil)
	require.True(t, hub.IsUserInRoom(TopicUUID("mystery:"+mysteryID.String()), userID))

	// when
	handleWSMessage(NewClient(userID, nil), incomingMessage{Type: "mystery_leave", Data: payload}, hub, nil, nil, nil)

	// then
	assert.False(t, hub.IsUserInRoom(TopicUUID("mystery:"+mysteryID.String()), userID))
}
