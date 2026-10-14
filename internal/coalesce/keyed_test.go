package coalesce

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	step struct {
		key       string
		value     int
		after     time.Duration
		wantFlush bool
	}

	recorder struct {
		mu      sync.Mutex
		flushed map[string][]int
	}
)

func (r *recorder) record(_ context.Context, key string, value int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.flushed[key] = append(r.flushed[key], value)
}

func (r *recorder) values(key string) []int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.flushed[key])
}

func TestKeyed_Claim(t *testing.T) {
	tests := []struct {
		name        string
		steps       []step
		forget      bool
		wantPending int
		wantHeld    bool
	}{
		{
			name:  "a lone value in a quiet window goes out at once and nothing is held",
			steps: []step{{key: "a", value: 1, wantFlush: true}},
		},
		{
			name: "values inside the window are held and merged",
			steps: []step{
				{key: "a", value: 1, wantFlush: true},
				{key: "a", value: 2, after: time.Second},
				{key: "a", value: 3, after: time.Second},
			},
			wantPending: 5,
			wantHeld:    true,
		},
		{
			name: "values that merge to the zero value are still held",
			steps: []step{
				{key: "a", value: 1, wantFlush: true},
				{key: "a", value: 1, after: time.Second},
				{key: "a", value: -1, after: time.Second},
			},
			wantPending: 0,
			wantHeld:    true,
		},
		{
			name: "a value after the window has passed goes out at once",
			steps: []step{
				{key: "a", value: 1, wantFlush: true},
				{key: "a", value: -1, after: time.Hour, wantFlush: true},
			},
		},
		{
			name: "keys are independent",
			steps: []step{
				{key: "a", value: 1, wantFlush: true},
				{key: "b", value: 1, wantFlush: true},
			},
		},
		{
			name: "a forgotten key drops what it was holding",
			steps: []step{
				{key: "a", value: 1, wantFlush: true},
				{key: "a", value: 5, after: time.Second},
			},
			forget: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			c := NewKeyed(time.Minute, Sum[int], func(context.Context, string, int) {})
			now := time.Now()

			// when
			for _, s := range tc.steps {
				now = now.Add(s.after)
				flushed := c.claim(s.key, s.value, now)

				// then
				assert.Equal(t, s.wantFlush, flushed)
			}

			if tc.forget {
				c.Forget("a")
			}

			// then
			pending, held := c.takePending("a", now)
			assert.Equal(t, tc.wantHeld, held)
			assert.Equal(t, tc.wantPending, pending)
			c.Forget("a")
			c.Forget("b")
		})
	}
}

func TestKeyed_ForgetsIdleKeys(t *testing.T) {
	// given
	c := NewKeyed(time.Minute, Latest[int], func(context.Context, string, int) {})
	start := time.Now()
	require.True(t, c.claim("old", 1, start))

	// when
	flushed := c.claim("new", 1, start.Add(time.Hour))

	// then
	assert.True(t, flushed)
	assert.NotContains(t, c.windows, "old", "a key with nothing held that is past its interval is dropped")
	assert.Contains(t, c.windows, "new", "the key just flushed stays throttled")
}

func TestKeyed_Add(t *testing.T) {
	interval := 50 * time.Millisecond

	tests := []struct {
		name  string
		merge func(pending, next int) int
		adds  []int
		want  []int
	}{
		{name: "a lone add flushes once", merge: Sum[int], adds: []int{1}, want: []int{1}},
		{name: "a burst flushes the leading value then one merged trailing value", merge: Sum[int], adds: []int{1, 1, 1, 1}, want: []int{1, 3}},
		{name: "the latest merge keeps only the newest held value", merge: Latest[int], adds: []int{1, 2, 3, 4}, want: []int{1, 4}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			rec := &recorder{flushed: make(map[string][]int)}
			c := NewKeyed(interval, tc.merge, rec.record)

			// when
			for _, value := range tc.adds {
				c.Add(context.Background(), "a", value)
			}

			// then
			require.Eventually(t, func() bool {
				return len(rec.values("a")) == len(tc.want)
			}, 20*interval, interval/10)
			assert.Never(t, func() bool {
				return len(rec.values("a")) > len(tc.want)
			}, 3*interval, interval/10)
			assert.Equal(t, tc.want, rec.values("a"))
		})
	}
}
