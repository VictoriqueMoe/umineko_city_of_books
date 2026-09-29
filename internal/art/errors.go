package art

import (
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dao"
)

var (
	ErrNotFound    = fmt.Errorf("art not found: %w", dao.ErrNotFound)
	ErrEmptyTitle  = errors.New("art title cannot be empty")
	ErrEmptyBody   = errors.New("comment body cannot be empty")
	ErrRateLimited = errors.New("you have reached your daily art upload limit")
)
