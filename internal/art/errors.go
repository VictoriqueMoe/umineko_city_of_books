package art

import (
	"errors"
	"fmt"

	"umineko_city_of_books/internal/daoerr"
)

var (
	ErrNotFound    = fmt.Errorf("art not found: %w", daoerr.ErrNotFound)
	ErrEmptyTitle  = errors.New("art title cannot be empty")
	ErrRateLimited = errors.New("you have reached your daily art upload limit")
)
