package spec

import (
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	NewShip struct {
		UserID      uuid.UUID
		Title       string
		Description string
	}

	NewShipWithCharacters struct {
		UserID      uuid.UUID
		Title       string
		Description string
		Characters  []dto.ShipCharacter
	}

	NewShipCharacters struct {
		ShipID     uuid.UUID
		Characters []dto.ShipCharacter
	}

	ShipDetailsUpdate struct {
		ID          uuid.UUID
		UserID      uuid.UUID
		Title       string
		Description string
		AsAdmin     bool
	}

	ShipUpdate struct {
		ID          uuid.UUID
		UserID      uuid.UUID
		Title       string
		Description string
		AsAdmin     bool
		Characters  []dto.ShipCharacter
	}

	ShipImageUpdate struct {
		ID           uuid.UUID
		ImageURL     string
		ThumbnailURL string
	}

	ShipDeletion struct {
		ID      uuid.UUID
		UserID  uuid.UUID
		AsAdmin bool
	}

	ShipLookup struct {
		ID       uuid.UUID
		ViewerID uuid.UUID
	}

	ShipListing struct {
		ViewerID       uuid.UUID
		Sort           string
		CrackshipsOnly bool
		Series         string
		CharacterID    string
		Limit          int
		Offset         int
		ExcludeUserIDs []uuid.UUID
	}

	ShipUserListing struct {
		UserID   uuid.UUID
		ViewerID uuid.UUID
		Limit    int
		Offset   int
	}
)
