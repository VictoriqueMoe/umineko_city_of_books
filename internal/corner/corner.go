package corner

import (
	"fmt"
	"slices"
)

type (
	GameBoard string

	Gallery string
)

const (
	GameBoardGeneral     GameBoard = "general"
	GameBoardUmineko     GameBoard = "umineko"
	GameBoardHigurashi   GameBoard = "higurashi"
	GameBoardCiconia     GameBoard = "ciconia"
	GameBoardHiganbana   GameBoard = "higanbana"
	GameBoardRoseguns    GameBoard = "roseguns"
	GameBoardSuggestions GameBoard = "suggestions"

	GalleryGeneral   Gallery = "general"
	GalleryUmineko   Gallery = "umineko"
	GalleryHigurashi Gallery = "higurashi"
	GalleryCiconia   Gallery = "ciconia"

	gameBoardPath   = "/game-board"
	suggestionsPath = "/suggestions"
	galleryPath     = "/gallery"
)

var (
	GameBoards = []GameBoard{
		GameBoardGeneral,
		GameBoardUmineko,
		GameBoardHigurashi,
		GameBoardCiconia,
		GameBoardHiganbana,
		GameBoardRoseguns,
		GameBoardSuggestions,
	}

	Galleries = []Gallery{
		GalleryGeneral,
		GalleryUmineko,
		GalleryHigurashi,
		GalleryCiconia,
	}
)

func ParseGameBoard(raw string) (GameBoard, error) {
	if raw == "" {
		return GameBoardGeneral, nil
	}

	board := GameBoard(raw)
	if !slices.Contains(GameBoards, board) {
		return "", fmt.Errorf("%w %q", ErrInvalidCorner, raw)
	}

	return board, nil
}

func ParseGallery(raw string) (Gallery, error) {
	if raw == "" {
		return GalleryGeneral, nil
	}

	gallery := Gallery(raw)
	if !slices.Contains(Galleries, gallery) {
		return "", fmt.Errorf("%w %q", ErrInvalidCorner, raw)
	}

	return gallery, nil
}

func GameBoardAtPath(path string) (GameBoard, bool) {
	for _, board := range GameBoards {
		if board.Path() == path {
			return board, true
		}
	}

	return "", false
}

func GalleryAtPath(path string) (Gallery, bool) {
	for _, gallery := range Galleries {
		if gallery.Path() == path {
			return gallery, true
		}
	}

	return "", false
}

func (b GameBoard) Path() string {
	switch b {
	case GameBoardGeneral:
		return gameBoardPath
	case GameBoardSuggestions:
		return suggestionsPath
	}

	return gameBoardPath + "/" + string(b)
}

func (b GameBoard) Name() string {
	switch b {
	case GameBoardGeneral:
		return "General"
	case GameBoardUmineko:
		return "Umineko"
	case GameBoardHigurashi:
		return "Higurashi"
	case GameBoardCiconia:
		return "Ciconia"
	case GameBoardHiganbana:
		return "Higanbana"
	case GameBoardRoseguns:
		return "Rose Guns Days"
	case GameBoardSuggestions:
		return "Suggestions"
	}

	return string(b)
}

func (g Gallery) Path() string {
	if g == GalleryGeneral {
		return galleryPath
	}

	return galleryPath + "/" + string(g)
}

func (g Gallery) Name() string {
	switch g {
	case GalleryGeneral:
		return "General"
	case GalleryUmineko:
		return "Umineko"
	case GalleryHigurashi:
		return "Higurashi"
	case GalleryCiconia:
		return "Ciconia"
	}

	return string(g)
}
