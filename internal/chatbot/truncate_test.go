package chatbot

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncate_CutsOnARuneBoundaryAndKeepsTheEllipsis(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		limit int
		want  string
	}{
		{name: "an empty body is untouched", body: "", limit: 5, want: ""},
		{name: "a body under the budget is untouched", body: "beato", limit: 10, want: "beato"},
		{name: "a body exactly on the budget is untouched", body: "beato", limit: 5, want: "beato"},
		{name: "an ascii body over the budget keeps the ellipsis", body: "beatrice", limit: 5, want: "beatr..."},
		{name: "a two byte rune is never halved", body: "éééé", limit: 5, want: "éé..."},
		{name: "a three byte rune is never halved", body: "ですですです", limit: 8, want: "です..."},
		{name: "a four byte rune is never halved", body: "🎉🎉🎉", limit: 6, want: "🎉..."},
		{name: "a cut landing on a boundary keeps the whole prefix", body: "ですです", limit: 6, want: "です..."},
		{name: "a budget shorter than the first rune yields only the ellipsis", body: "です", limit: 2, want: "..."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := truncate(tc.body, tc.limit)

			// then
			assert.Equal(t, tc.want, got)
			assert.True(t, utf8.ValidString(got))
			assert.LessOrEqual(t, len(strings.TrimSuffix(got, "...")), tc.limit)
		})
	}
}

func TestReplyContext_AQuotedJapaneseLineNeverReachesTheModelAsEscapes(t *testing.T) {
	cases := []struct {
		name     string
		row      promptRow
		wantHead string
	}{
		{
			name:     "a named target still gets a clean quote",
			row:      promptRow{ReplyToName: "Battler", ReplyToBody: strings.Repeat("赤き真実", 60)},
			wantHead: " (replying to Battler: ",
		},
		{
			name:     "an unnamed target still gets a clean quote",
			row:      promptRow{ReplyToBody: strings.Repeat("赤き真実", 60)},
			wantHead: " (replying to ",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			require.Greater(t, len(tc.row.ReplyToBody), replyQuoteMax)

			// when
			got := replyContext(tc.row)

			// then
			assert.True(t, strings.HasPrefix(got, tc.wantHead))
			assert.True(t, utf8.ValidString(got))
			assert.NotContains(t, got, `\x`)
			assert.Contains(t, got, "赤き真実")
		})
	}
}
