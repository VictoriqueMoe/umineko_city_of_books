package secret

import (
	"fmt"

	"umineko_city_of_books/internal/dao"
)

var (
	ErrNotFound = fmt.Errorf("secret not found: %w", dao.ErrNotFound)
)
