package oc

import (
	"errors"
	"fmt"

	"umineko_city_of_books/internal/daoerr"
)

var (
	ErrNotFound          = fmt.Errorf("oc not found: %w", daoerr.ErrNotFound)
	ErrEmptyName         = errors.New("oc name cannot be empty")
	ErrInvalidSeries     = errors.New("oc series must be one of umineko, higurashi, ciconia, custom")
	ErrEmptyCustomSeries = errors.New("custom series name is required when series is custom")
	ErrDuplicateName     = errors.New("you already have an oc with that name")
	ErrNotOwner          = errors.New("not the oc owner")
)
