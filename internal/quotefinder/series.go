package quotefinder

import (
	"fmt"
	"strings"
)

type Series string

const (
	SeriesUmineko   Series = "umineko"
	SeriesHigurashi Series = "higurashi"
)

func (s Series) Valid() bool {
	return s == SeriesUmineko || s == SeriesHigurashi
}

func ParseSeries(value string) (Series, error) {
	s := Series(strings.ToLower(strings.TrimSpace(value)))
	if !s.Valid() {
		return "", fmt.Errorf("unsupported series: %s", value)
	}
	return s, nil
}
