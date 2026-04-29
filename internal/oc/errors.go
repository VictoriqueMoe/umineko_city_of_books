package oc

import "errors"

var (
	ErrNotFound          = errors.New("oc not found")
	ErrEmptyName         = errors.New("oc name cannot be empty")
	ErrEmptyBody         = errors.New("comment body cannot be empty")
	ErrInvalidSeries     = errors.New("oc series must be one of umineko, higurashi, ciconia, custom")
	ErrEmptyCustomSeries = errors.New("custom series name is required when series is custom")
	ErrDuplicateName     = errors.New("you already have an oc with that name")
)
