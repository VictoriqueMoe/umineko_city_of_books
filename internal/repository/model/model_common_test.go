package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
