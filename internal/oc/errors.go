package oc

import (
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dao"
)

var (
	ErrNotFound          = fmt.Errorf("oc not found: %w", dao.ErrNotFound)
	ErrEmptyName         = errors.New("oc name cannot be empty")
	ErrEmptyBody         = errors.New("comment body cannot be empty")
	ErrInvalidSeries     = errors.New("oc series must be one of umineko, higurashi, ciconia, custom")
	ErrEmptyCustomSeries = errors.New("custom series name is required when series is custom")
	ErrDuplicateName     = errors.New("you already have an oc with that name")
	ErrNotOwner          = errors.New("not the oc owner")
)
