package sidebar

import "umineko_city_of_books/internal/corner"

const (
	KeyFanfiction Key = "fanfiction"
	KeyJournals   Key = "journals"
	KeyMysteries  Key = "mysteries"
	KeyOCs        Key = "ocs"
	KeyShips      Key = "ships"
	KeySecrets    Key = "secrets"
	KeyRooms      Key = "rooms"

	galleryKeyPrefix   = "gallery_"
	gameBoardKeyPrefix = "game_board_"
	theoriesKeyPrefix  = "theories_"
)

type (
	Key string

	ActivityBumper interface {
		BumpSidebarActivity(key Key)
	}
)

func GalleryKey(gallery corner.Gallery) Key {
	return Key(galleryKeyPrefix + string(gallery))
}

func GameBoardKey(board corner.GameBoard) Key {
	return Key(gameBoardKeyPrefix + string(board))
}

func TheoriesKey(series string) Key {
	return Key(theoriesKeyPrefix + series)
}
