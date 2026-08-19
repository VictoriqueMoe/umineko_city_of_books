package model

import (
	"testing"
	"time"

	"umineko_city_of_books/internal/role"

	"github.com/stretchr/testify/assert"
)

func TestIsRestrictedNewAccount(t *testing.T) {
	justNow := time.Now().Add(-time.Hour).Format(time.RFC3339)
	approved := time.Now().Format(time.RFC3339)

	tests := []struct {
		name  string
		user  *User
		hours int
		want  bool
	}{
		{
			name:  "a fresh account is restricted",
			user:  &User{CreatedAt: justNow},
			hours: 24,
			want:  true,
		},
		{
			name:  "an approved account is trusted straight away",
			user:  &User{CreatedAt: justNow, ApprovedAt: &approved},
			hours: 24,
			want:  false,
		},
		{
			name:  "staff are never restricted",
			user:  &User{CreatedAt: justNow, Role: string(role.RoleModerator)},
			hours: 24,
			want:  false,
		},
		{
			name:  "an established account is not restricted",
			user:  &User{CreatedAt: time.Now().Add(-48 * time.Hour).Format(time.RFC3339)},
			hours: 24,
			want:  false,
		},
		{
			name:  "the whole restriction switches off at zero hours",
			user:  &User{CreatedAt: justNow},
			hours: 0,
			want:  false,
		},
		{
			name:  "a nil user is never restricted",
			user:  nil,
			hours: 24,
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a user and the configured window

			// when
			got := tc.user.IsRestrictedNewAccount(tc.hours)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsApproved(t *testing.T) {
	// given
	stamp := time.Now().Format(time.RFC3339)

	// then
	assert.False(t, (*User)(nil).IsApproved())
	assert.False(t, new(User).IsApproved())
	assert.True(t, (&User{ApprovedAt: &stamp}).IsApproved())
}

func TestPostMediaRowToResponse_CarriesEveryColumn(t *testing.T) {
	// given a row read back from any media table, audio included
	row := PostMediaRow{
		ID:           7,
		MediaURL:     "/uploads/posts/abc.flac",
		MediaType:    "audio",
		ThumbnailURL: "/uploads/posts/abc.png",
		Filename:     "神様の言う通り.flac",
		SortOrder:    2,
	}

	// when
	resp := row.ToResponse()

	// then every column survives, because a dropped field here renders as a blank player
	assert.Equal(t, 7, resp.ID)
	assert.Equal(t, "/uploads/posts/abc.flac", resp.MediaURL)
	assert.Equal(t, "audio", resp.MediaType)
	assert.Equal(t, "/uploads/posts/abc.png", resp.ThumbnailURL)
	assert.Equal(t, "神様の言う通り.flac", resp.Filename)
	assert.Equal(t, 2, resp.SortOrder)
}

func TestMediaRowsToResponse_PreservesOrderAndFilenames(t *testing.T) {
	// given
	rows := []PostMediaRow{
		{ID: 1, MediaURL: "/a.png", MediaType: "image", SortOrder: 0},
		{ID: 2, MediaURL: "/b.mp3", MediaType: "audio", Filename: "track.mp3", SortOrder: 1},
	}

	// when
	list := MediaRowsToResponse(rows)

	// then
	assert.Len(t, list, 2)
	assert.Equal(t, "", list[0].Filename)
	assert.Equal(t, "track.mp3", list[1].Filename)
	assert.Equal(t, "audio", list[1].MediaType)
}
