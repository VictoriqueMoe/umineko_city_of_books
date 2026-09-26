package dao

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrAlreadyVoted = errors.New("already voted")
)
