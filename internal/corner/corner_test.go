package corner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGameBoard(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    GameBoard
		wantErr error
	}{
		{name: "an empty corner defaults to the general board", raw: "", want: GameBoardGeneral},
		{name: "the general board", raw: "general", want: GameBoardGeneral},
		{name: "umineko", raw: "umineko", want: GameBoardUmineko},
		{name: "higurashi", raw: "higurashi", want: GameBoardHigurashi},
		{name: "ciconia", raw: "ciconia", want: GameBoardCiconia},
		{name: "higanbana", raw: "higanbana", want: GameBoardHiganbana},
		{name: "roseguns", raw: "roseguns", want: GameBoardRoseguns},
		{name: "suggestions", raw: "suggestions", want: GameBoardSuggestions},
		{name: "a made-up corner is rejected", raw: "phantom", wantErr: ErrInvalidCorner},
		{name: "a case variant is rejected", raw: "Umineko", wantErr: ErrInvalidCorner},
		{name: "padding is rejected", raw: " umineko", wantErr: ErrInvalidCorner},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := ParseGameBoard(tc.raw)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseGallery(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    Gallery
		wantErr error
	}{
		{name: "an empty corner defaults to the general gallery", raw: "", want: GalleryGeneral},
		{name: "the general gallery", raw: "general", want: GalleryGeneral},
		{name: "umineko", raw: "umineko", want: GalleryUmineko},
		{name: "higurashi", raw: "higurashi", want: GalleryHigurashi},
		{name: "ciconia", raw: "ciconia", want: GalleryCiconia},
		{name: "a game board only corner is rejected", raw: "higanbana", wantErr: ErrInvalidCorner},
		{name: "suggestions is not a gallery", raw: "suggestions", wantErr: ErrInvalidCorner},
		{name: "a made-up corner is rejected", raw: "phantom", wantErr: ErrInvalidCorner},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := ParseGallery(tc.raw)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParse_RejectionNamesTheCorner(t *testing.T) {
	cases := []struct {
		name  string
		parse func(raw string) error
	}{
		{
			name: "game board",
			parse: func(raw string) error {
				_, err := ParseGameBoard(raw)

				return err
			},
		},
		{
			name: "gallery",
			parse: func(raw string) error {
				_, err := ParseGallery(raw)

				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			err := tc.parse("phantom")

			// then
			assert.EqualError(t, err, `invalid corner "phantom"`)
		})
	}
}

func TestGameBoardPathsAndNames(t *testing.T) {
	cases := []struct {
		board    GameBoard
		wantPath string
		wantName string
	}{
		{board: GameBoardGeneral, wantPath: "/game-board", wantName: "General"},
		{board: GameBoardUmineko, wantPath: "/game-board/umineko", wantName: "Umineko"},
		{board: GameBoardHigurashi, wantPath: "/game-board/higurashi", wantName: "Higurashi"},
		{board: GameBoardCiconia, wantPath: "/game-board/ciconia", wantName: "Ciconia"},
		{board: GameBoardHiganbana, wantPath: "/game-board/higanbana", wantName: "Higanbana"},
		{board: GameBoardRoseguns, wantPath: "/game-board/roseguns", wantName: "Rose Guns Days"},
		{board: GameBoardSuggestions, wantPath: "/suggestions", wantName: "Suggestions"},
	}

	for _, tc := range cases {
		t.Run(string(tc.board), func(t *testing.T) {
			// when
			path := tc.board.Path()
			name := tc.board.Name()
			found, ok := GameBoardAtPath(path)

			// then
			assert.Equal(t, tc.wantPath, path)
			assert.Equal(t, tc.wantName, name)
			require.True(t, ok)
			assert.Equal(t, tc.board, found)
		})
	}
}

func TestGalleryPathsAndNames(t *testing.T) {
	cases := []struct {
		gallery  Gallery
		wantPath string
		wantName string
	}{
		{gallery: GalleryGeneral, wantPath: "/gallery", wantName: "General"},
		{gallery: GalleryUmineko, wantPath: "/gallery/umineko", wantName: "Umineko"},
		{gallery: GalleryHigurashi, wantPath: "/gallery/higurashi", wantName: "Higurashi"},
		{gallery: GalleryCiconia, wantPath: "/gallery/ciconia", wantName: "Ciconia"},
	}

	for _, tc := range cases {
		t.Run(string(tc.gallery), func(t *testing.T) {
			// when
			path := tc.gallery.Path()
			name := tc.gallery.Name()
			found, ok := GalleryAtPath(path)

			// then
			assert.Equal(t, tc.wantPath, path)
			assert.Equal(t, tc.wantName, name)
			require.True(t, ok)
			assert.Equal(t, tc.gallery, found)
		})
	}
}

func TestAtPath_UnknownPaths(t *testing.T) {
	cases := []struct {
		name   string
		lookup func(path string) bool
		path   string
	}{
		{name: "a game board corner that does not exist", lookup: gameBoardFound, path: "/game-board/phantom"},
		{name: "the general board is not a sub-section", lookup: gameBoardFound, path: "/game-board/general"},
		{name: "suggestions does not live under the game board", lookup: gameBoardFound, path: "/game-board/suggestions"},
		{name: "a gallery corner that does not exist", lookup: galleryFound, path: "/gallery/phantom"},
		{name: "the general gallery is not a sub-section", lookup: galleryFound, path: "/gallery/general"},
		{name: "a game board only corner has no gallery", lookup: galleryFound, path: "/gallery/roseguns"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			ok := tc.lookup(tc.path)

			// then
			assert.False(t, ok)
		})
	}
}

func gameBoardFound(path string) bool {
	_, ok := GameBoardAtPath(path)

	return ok
}

func galleryFound(path string) bool {
	_, ok := GalleryAtPath(path)

	return ok
}
