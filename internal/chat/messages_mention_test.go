package chat

import (
	"testing"

	"umineko_city_of_books/internal/social"

	"github.com/stretchr/testify/assert"
)

func TestMentionRegex_IgnoresEmailAddresses(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{"single mention", "hello @alice", []string{"alice"}},
		{"multiple mentions", "@alice said hi to @bob", []string{"alice", "bob"}},
		{"underscores and digits", "ping @user_42", []string{"user_42"}},
		{"no mentions", "just some text", nil},
		{"email address is not a mention", "mail me at kujo@alice.example", nil},
		{"username glued to a word is not a mention", "beato@alice", nil},
		{"mention after punctuation still counts", "(@alice)", []string{"alice"}},
		{"mention at the very start still counts", "@alice hello", []string{"alice"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			matches := social.MentionRegex.FindAllStringSubmatch(tc.body, -1)

			// when
			var got []string
			for i := range matches {
				got = append(got, matches[i][1])
			}

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}
