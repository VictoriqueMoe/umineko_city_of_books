package gameroom

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPostSpectatorChat_ClampsByRunesNotBytes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "an ascii body within the limit is untouched",
			body: "nice move",
			want: "nice move",
		},
		{
			name: "a japanese body the form accepted is no longer chopped",
			body: strings.Repeat("素晴らしい", 40),
			want: strings.Repeat("素晴らしい", 40),
		},
		{
			name: "an ascii body over the limit keeps five hundred characters",
			body: strings.Repeat("a", 600),
			want: strings.Repeat("a", 500),
		},
		{
			name: "a japanese body over the limit keeps five hundred characters",
			body: strings.Repeat("あ", 600),
			want: strings.Repeat("あ", 500),
		},
		{
			name: "an emoji body over the limit is never halved",
			body: strings.Repeat("🎉", 600),
			want: strings.Repeat("🎉", 500),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			m := newTestService(t)
			roomID := uuid.New()
			creator := uuid.New()
			spectator := uuid.New()

			m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(activeRoomRow(roomID, creator), nil)
			m.roomRepo.EXPECT().IsParticipant(mock.Anything, roomID, spectator).Return(false, nil)
			seedUser(t, m, spectator, "Spectator")

			// when
			msg, err := m.svc.PostSpectatorChat(context.Background(), roomID, spectator, tc.body)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, msg.Body)
			assert.True(t, utf8.ValidString(msg.Body))
			assert.LessOrEqual(t, utf8.RuneCountInString(msg.Body), maxChatBodyLen)
		})
	}
}
