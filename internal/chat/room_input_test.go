package chat

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormaliseRoomInput_ClampsByRunesNotBytes(t *testing.T) {
	cases := []struct {
		name            string
		rawName         string
		rawDescription  string
		wantName        string
		wantDescription string
	}{
		{
			name:            "ascii within the limits is untouched",
			rawName:         "Rokkenjima",
			rawDescription:  "the golden land",
			wantName:        "Rokkenjima",
			wantDescription: "the golden land",
		},
		{
			name:            "a japanese name the form accepted is no longer chopped",
			rawName:         strings.Repeat("黄金", 20),
			rawDescription:  strings.Repeat("蝶", 100),
			wantName:        strings.Repeat("黄金", 20),
			wantDescription: strings.Repeat("蝶", 100),
		},
		{
			name:            "an ascii name over the limit keeps eighty characters",
			rawName:         strings.Repeat("a", 100),
			rawDescription:  strings.Repeat("b", 600),
			wantName:        strings.Repeat("a", 80),
			wantDescription: strings.Repeat("b", 500),
		},
		{
			name:            "a japanese name over the limit keeps eighty characters",
			rawName:         strings.Repeat("あ", 100),
			rawDescription:  strings.Repeat("い", 600),
			wantName:        strings.Repeat("あ", 80),
			wantDescription: strings.Repeat("い", 500),
		},
		{
			name:            "an emoji name over the limit is never halved",
			rawName:         strings.Repeat("🎉", 100),
			rawDescription:  strings.Repeat("🎉", 600),
			wantName:        strings.Repeat("🎉", 80),
			wantDescription: strings.Repeat("🎉", 500),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _ := newTestService(t)

			// when
			name, description, _, err := svc.normaliseRoomInput(context.Background(), tc.rawName, tc.rawDescription, nil)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, name)
			assert.Equal(t, tc.wantDescription, description)
			assert.True(t, utf8.ValidString(name))
			assert.True(t, utf8.ValidString(description))
			assert.LessOrEqual(t, utf8.RuneCountInString(name), maxRoomNameLength)
			assert.LessOrEqual(t, utf8.RuneCountInString(description), maxRoomDescriptionLength)
		})
	}
}
