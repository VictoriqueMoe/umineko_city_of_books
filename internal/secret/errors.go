package secret

import (
	"fmt"

	"umineko_city_of_books/internal/daoerr"
)

var (
	ErrNotFound = fmt.Errorf("secret not found: %w", daoerr.ErrNotFound)
)
