package httpclient

import "errors"

var (
	ErrBodyConflict   = errors.New("httpclient: a request carries either a JSON body or a form body, not both")
	ErrBlockedAddress = errors.New("httpclient: refusing to dial non-public address")
)
