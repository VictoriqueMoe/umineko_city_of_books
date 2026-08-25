package bannedgiphy

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/contentfilter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheck_AllowsCleanText(t *testing.T) {
	// given
	r := New(NewMockBanlist(t), nil)

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
			b := NewMockBanlist(t)
			b.EXPECT().ContainsGif("abc123").Return(true).Once()

			r := New(b, nil)

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
	b := NewMockBanlist(t)
	b.EXPECT().ContainsGif("xyz789").Return(false).Once()

	r := New(b, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/gifs/allowed-xyz789"})

	// then
	require.NoError(t, err)
	assert.Nil(t, rej)
}

func TestCheck_DetectsBannedChannel(t *testing.T) {
	// given
	b := NewMockBanlist(t)
	b.EXPECT().ContainsUser("Larperine").Return(true).Once()

	r := New(b, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"linked: https://giphy.com/channel/Larperine"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
	assert.Equal(t, "Larperine", rej.Detail)
}

func TestCheck_DetectsBannedProfileURL(t *testing.T) {
	// given
	b := NewMockBanlist(t)
	b.EXPECT().ContainsUser("Larperine").Return(true).Once()

	r := New(b, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/Larperine/"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
	assert.Equal(t, "Larperine", rej.Detail)
}

func TestCheck_IgnoresReservedProfileSegments(t *testing.T) {
	// given — "gifs" is reserved, so it is never offered to ContainsUser and no such call is expected
	b := NewMockBanlist(t)
	b.EXPECT().ContainsGif("something").Return(false).Once()

	r := New(b, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/gifs/something"})

	// then
	require.NoError(t, err)
	assert.Nil(t, rej)
}

func TestCheck_UsesGifCheckBeforeUserCheck(t *testing.T) {
	// given — if GIF and user are both banned, the GIF detection takes precedence
	b := NewMockBanlist(t)
	b.EXPECT().ContainsGif("abc").Return(true).Once()

	r := New(b, nil)

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
	b := NewMockBanlist(t)
	b.EXPECT().ContainsGif("bad").Return(true).Once()

	r := New(b, nil)

	// when — banned GIF is in the second text
	rej, err := r.Check(context.Background(), []string{"clean", "https://media.giphy.com/media/bad/giphy.gif"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
	assert.Equal(t, "bad", rej.Detail)
}

func TestCheck_ResolvesGifUploaderAgainstUserBanlist(t *testing.T) {
	// given
	b := NewMockBanlist(t)
	b.EXPECT().ContainsGif("abc123").Return(false).Once()
	b.EXPECT().ContainsUser("Larperine").Return(true).Once()

	lookup := NewMockLookup(t)
	lookup.EXPECT().UserForGif(mock.Anything, "abc123").Return("Larperine", true).Once()

	r := New(b, lookup)

	// when — GIF ID isn't banned but its uploader is
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/gifs/battler-abc123"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
	assert.Equal(t, "Larperine", rej.Detail)
}

func TestCheck_NoLookup_StillEnforcesDirectUserURL(t *testing.T) {
	// given
	b := NewMockBanlist(t)
	b.EXPECT().ContainsUser("Larperine").Return(true).Once()

	r := New(b, nil)

	// when
	rej, err := r.Check(context.Background(), []string{"https://giphy.com/channel/Larperine"})

	// then
	require.NoError(t, err)
	require.NotNil(t, rej)
}
