package favourite

import "errors"

var (
	ErrGiphyIDRequired = errors.New("giphy_id required")
	ErrURLRequired     = errors.New("url required")
)
