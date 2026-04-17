package bannedgiphy

import (
	"context"
	"strings"
	"testing"

	"umineko_city_of_books/internal/contentfilter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBanlist struct {
	gifs  map[string]bool
	users map[string]bool
}

func (b *fakeBanlist) ContainsGif(id string) bool        { return b.gifs[id] }
func (b *fakeBanlist) ContainsUser(username string) bool { return b.users[strings.ToLower(username)] }

type fakeLookup map[string]string

func (f fakeLookup) UserForGif(_ context.Context, gifID string) (string, bool) {
	u, ok := f[gifID]
	return u, ok
}

func TestCheck_AllowsCleanText(t *testing.T) {
	// given
	r := New(&fakeBanlist{}, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"just some text", "https://example.com"})

	// then
	require.NoError(t, err)
	assert.Nil(t, rej)
}

func TestCheck_DetectsBannedGifAcrossURLShapes(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"gifs-path", "here https://giphy.com/gifs/abc123 ok"},
		{"gifs-with-slug", "check https://giphy.com/gifs/funny-cat-abc123 ok"},
		{"media-simple", "https://media.giphy.com/media/abc123/giphy.gif"},
		{"media-numeric-subdomain", "https://media3.giphy.com/media/abc123/giphy.gif"},
		{"media-v1-versioned", "https://media0.giphy.com/media/v1.Y2lkPXh4eA/abc123/giphy.gif"},
		{"i-giphy", "https://i.giphy.com/abc123.gif"},
		{"trailing-query", "https://giphy.com/gifs/abc123?q=1"},
		{"trailing-slash", "https://giphy.com/gifs/abc123/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := New(&fakeBanlist{gifs: map[string]bool{"abc123": true}}, nil)

			// when
			rej, err := r.Check(context.Background(), []string{tc.text})

			// then
			require.NoError(t, err)
			require.NotNil(t, rej)
			assert.Equal(t, contentfilter.RuleBannedGiphy, rej.Rule)
			assert.Equal(t, "abc123", rej.Detail)
		})
	}
}

func TestCheck_AllowsDifferentGifID(t *testing.T) {
	// given
	r := New(&fakeBanlist{gifs: map[string]bool{"banned": true}}, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/gifs/allowed-xyz789"})

	// then
	require.NoError(t, err)
	assert.Nil(t, rej)
}

func TestCheck_DetectsBannedChannel(t *testing.T) {
	// given
	r := New(&fakeBanlist{users: map[string]bool{"larperine": true}}, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"linked: https://giphy.com/channel/Larperine"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
	assert.Equal(t, "Larperine", rej.Detail)
}

func TestCheck_DetectsBannedProfileURL(t *testing.T) {
	// given
	r := New(&fakeBanlist{users: map[string]bool{"larperine": true}}, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/Larperine/"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
	assert.Equal(t, "Larperine", rej.Detail)
}

func TestCheck_IgnoresReservedProfileSegments(t *testing.T) {
	// given — "gifs" is reserved even if "gifs" ends up as a username in the banlist
	r := New(&fakeBanlist{users: map[string]bool{"gifs": true}}, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/gifs/something"})

	// then
	require.NoError(t, err)
	assert.Nil(t, rej)
}

func TestCheck_UsesGifCheckBeforeUserCheck(t *testing.T) {
	// given — if GIF and user are both banned, the GIF detection takes precedence
	r := New(&fakeBanlist{
		gifs:  map[string]bool{"abc": true},
		users: map[string]bool{"larperine": true},
	}, nil)

	// when
	rej, err := r.Check(
		context.Background(),
		[]string{"https://giphy.com/gifs/abc https://giphy.com/channel/Larperine"},
	)

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
	assert.Equal(t, "abc", rej.Detail)
}

func TestCheck_ScansMultipleTextsInOrder(t *testing.T) {
	// given
	r := New(&fakeBanlist{gifs: map[string]bool{"bad": true}}, nil)

	// when — banned GIF is in the second text
	rej, err := r.Check(context.Background(), []string{"clean", "https://media.giphy.com/media/bad/giphy.gif"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
	assert.Equal(t, "bad", rej.Detail)
}

func TestCheck_ResolvesGifUploaderAgainstUserBanlist(t *testing.T) {
	// given
	banlist := &fakeBanlist{users: map[string]bool{"larperine": true}}
	lookup := fakeLookup{"abc123": "Larperine"}
	r := New(banlist, lookup)

	// when — GIF ID isn't banned but its uploader is
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/gifs/battler-abc123"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
	assert.Equal(t, "Larperine", rej.Detail)
}

func TestCheck_NoLookup_StillEnforcesDirectUserURL(t *testing.T) {
	// given
	r := New(&fakeBanlist{users: map[string]bool{"larperine": true}}, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/channel/Larperine"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
}
