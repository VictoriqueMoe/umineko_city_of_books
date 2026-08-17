package utils

import (
	"strconv"
)

func PlaceholderArgs[T any](values []T, startIndex int) ([]string, []any) {
	placeholders := make([]string, len(values))
	args := make([]any, len(values))

	for i := range values {
		placeholders[i] = "$" + strconv.Itoa(startIndex+i)
		args[i] = values[i]
	}

	return placeholders, args
}

func QuestionArgs[T any](values []T) ([]string, []any) {
	placeholders := make([]string, len(values))
	args := make([]any, len(values))

	for i := range values {
		placeholders[i] = "?"
		args[i] = values[i]
	}

	return placeholders, args
}
