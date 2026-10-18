package settings

type (
	RejectedError struct {
		Err error
	}
)

func (e *RejectedError) Error() string {
	return e.Err.Error()
}

func (e *RejectedError) Unwrap() error {
	return e.Err
}
