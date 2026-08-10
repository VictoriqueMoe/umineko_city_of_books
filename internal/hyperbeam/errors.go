package hyperbeam

import (
	"errors"
	"fmt"
)

var (
	ErrDisabled          = errors.New("hyperbeam integration is not configured")
	ErrMissingAdminToken = errors.New("hyperbeam: this watch party has no stored admin token")
)

const codeNoAvailableVM = "err_no_available_vm"

type APIError struct {
	StatusCode int
	Code       string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("hyperbeam api %d: %s", e.StatusCode, e.Body)
}

func IsNoCapacity(err error) bool {
	apiErr, ok := errors.AsType[*APIError](err)
	if !ok {
		return false
	}
	return apiErr.Code == codeNoAvailableVM
}
