package mention

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsernames(t *testing.T) {
	var distinct strings.Builder
	for i := range 25 {
		distinct.WriteString("@user" + strconv.Itoa(i) + " ")
	}

	firstTwenty := make([]string, 0, MaxMatches)
	for i := range MaxMatches {
		firstTwenty = append(firstTwenty, "user"+strconv.Itoa(i))
	}

	var repeatedThenNew strings.Builder
	for range MaxMatches {
		repeatedThenNew.WriteString("@alice ")
	}
	repeatedThenNew.WriteString("@bob")

	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "empty body yields nothing",
			body: "",
			want: nil,
		},
		{
			name: "plain prose yields nothing",
			body: "Beatrice never showed her face on the game board.",
			want: nil,
		},
		{
			name: "a mention at the start of the body is matched",
			body: "@alice look at the epitaph",
			want: []string{"alice"},
		},
		{
			name: "usernames stop at punctuation",
			body: "well @alice, what do you think?",
			want: []string{"alice"},
		},
		{
			name: "underscores and digits are part of a username",
			body: "ping @ange_1986 please",
			want: []string{"ange_1986"},
		},
		{
			name: "email addresses are not mentions",
			body: "write to bob@example.com instead",
			want: nil,
		},
		{
			name: "a username glued to a word is not a mention",
			body: "beato@alice",
			want: nil,
		},
		{
			name: "a mention wrapped in brackets is matched",
			body: "(@alice)",
			want: []string{"alice"},
		},
		{
			name: "a dash ends a username",
			body: "@alice-bob",
			want: []string{"alice"},
		},
		{
			name: "repeated usernames are deduped in first-seen order",
			body: "@alice hi @bob and again @alice",
			want: []string{"alice", "bob"},
		},
		{
			name: "matching stops after twenty matches",
			body: distinct.String(),
			want: firstTwenty,
		},
		{
			name: "the cap counts matches before the dedupe",
			body: repeatedThenNew.String(),
			want: []string{"alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given the body under test

			// when
			got := Usernames(tt.body)

			// then
			assert.Equal(t, tt.want, got)
		})
	}
}
