package chatbot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBotEvent_Channel(t *testing.T) {
	cases := []struct {
		name    string
		surface Surface
		isDM    bool
		want    Channel
	}{
		{"a direct message is its own channel", SurfaceChat, true, ChannelDM},
		{"a room message is a group chat", SurfaceChat, false, ChannelGroup},
		{"a game board mention is a post", SurfacePost, false, ChannelPost},
		{"a comment mention is a post comment", SurfacePostComment, false, ChannelPostComment},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ev := botEvent{Surface: tc.surface, IsDM: tc.isDM}

			// when
			got := ev.channel()

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}
