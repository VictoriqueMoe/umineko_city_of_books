package reserved

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPathSegment(t *testing.T) {
	tests := []struct {
		name    string
		segment string
		want    bool
	}{
		{name: "the live directory itself", segment: "live", want: true},
		{name: "a route whose second segment is also live", segment: "games", want: true},
		{name: "the profile prefix", segment: "user", want: true},
		{name: "a go wildcard mount", segment: "uploads", want: true},
		{name: "a hyphenated route", segment: "game-board", want: true},
		{name: "casing is folded", segment: "Games", want: true},
		{name: "shouty casing is folded", segment: "LIVE", want: true},
		{name: "an ordinary username is free", segment: "Featherine", want: false},
		{name: "a name merely containing a route word is free", segment: "gamesmaster", want: false},
		{name: "another substring near-miss is free", segment: "theorycrafter", want: false},
		{name: "empty is not reserved", segment: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a candidate first path segment

			// when it is checked against the reserved set
			got := IsPathSegment(tc.segment)

			// then only exact, case-folded route names are refused
			assert.Equal(t, tc.want, got)
		})
	}
}
