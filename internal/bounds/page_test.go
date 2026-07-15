package bounds

import "testing"

func TestNewPage(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		offset     int
		wantLimit  int
		wantOffset int
	}{
		{name: "typical values pass through", limit: 50, offset: 100, wantLimit: 50, wantOffset: 100},
		{name: "zero limit becomes default", limit: 0, offset: 0, wantLimit: DefaultLimit, wantOffset: 0},
		{name: "negative limit becomes default", limit: -1, offset: 0, wantLimit: DefaultLimit, wantOffset: 0},
		{name: "limit above max is clamped", limit: 999999999, offset: 0, wantLimit: MaxLimit, wantOffset: 0},
		{name: "negative offset is floored", limit: 20, offset: -1, wantLimit: 20, wantOffset: 0},
		{name: "offset above max is clamped", limit: 20, offset: 99999999, wantLimit: 20, wantOffset: MaxOffset},
		{name: "boundary values are kept", limit: MaxLimit, offset: MaxOffset, wantLimit: MaxLimit, wantOffset: MaxOffset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given a limit and offset from an untrusted request
			// when the page is constructed
			p := NewPage(tt.limit, tt.offset)

			// then both bounds are enforced
			if p.Limit() != tt.wantLimit {
				t.Errorf("Limit() = %d, want %d", p.Limit(), tt.wantLimit)
			}
			if p.Offset() != tt.wantOffset {
				t.Errorf("Offset() = %d, want %d", p.Offset(), tt.wantOffset)
			}
		})
	}
}

func TestZeroPageIsSafe(t *testing.T) {
	// given a zero-value page that never went through the constructor
	var p Page

	// when its bounds are read
	// then it yields the default limit rather than zero rows
	if p.Limit() != DefaultLimit {
		t.Errorf("Limit() = %d, want %d", p.Limit(), DefaultLimit)
	}
	if p.Offset() != 0 {
		t.Errorf("Offset() = %d, want 0", p.Offset())
	}
}

func TestWindowIsBounded(t *testing.T) {
	// given the offset that previously allocated 15GB in search
	p := NewPage(100, 99999999)

	// when the fetch window is derived
	got := p.Window()

	// then it cannot exceed the sum of both caps
	if got != MaxLimit+MaxOffset {
		t.Errorf("Window() = %d, want %d", got, MaxLimit+MaxOffset)
	}
}
