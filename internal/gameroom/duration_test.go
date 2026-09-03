package gameroom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDurationSeconds(t *testing.T) {
	tests := []struct {
		name       string
		createdAt  string
		finishedAt string
		want       int
	}{
		{name: "a blank start is zero", createdAt: "", finishedAt: "2026-04-22T10:30:00Z"},
		{name: "an unparseable start is zero", createdAt: "not a time", finishedAt: "2026-04-22T10:30:00Z"},
		{name: "an unparseable finish is zero", createdAt: "2026-04-22T10:00:00Z", finishedAt: "half past"},
		{name: "rfc3339 across half an hour", createdAt: "2026-04-22T10:00:00Z", finishedAt: "2026-04-22T10:30:00Z", want: 1800},
		{name: "the database space separated layout", createdAt: "2026-04-22 10:00:00", finishedAt: "2026-04-22 10:00:45", want: 45},
		{name: "nanosecond precision is accepted", createdAt: "2026-04-22T10:00:00.500Z", finishedAt: "2026-04-22T10:00:10.500Z", want: 10},
		{name: "a finish before the start is floored at zero", createdAt: "2026-04-22T10:30:00Z", finishedAt: "2026-04-22T10:00:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got := DurationSeconds(tt.createdAt, tt.finishedAt)

			// then
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDurationSeconds_RunningGameMeasuresToNow(t *testing.T) {
	// given a game that started in the past and has not finished
	createdAt := "2020-01-01T00:00:00Z"

	// when
	got := DurationSeconds(createdAt, "")

	// then
	assert.Positive(t, got)
}

func TestBaseHandlerDefaults(t *testing.T) {
	// given
	var h BaseHandler

	// when
	projected, err := h.ProjectState(`{"fen":"x"}`, true)

	// then
	assert.Equal(t, ModeTurnBased, h.Mode())
	assert.True(t, h.SupportsDraw())
	assert.NoError(t, err)
	assert.Equal(t, `{"fen":"x"}`, projected)
}
