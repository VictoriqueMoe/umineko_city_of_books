package middleware

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type fakeUpdater struct {
	calls atomic.Int32
	done  chan struct{}
}

func (f *fakeUpdater) UpdateIP(_ context.Context, _ uuid.UUID, _ string) error {
	f.calls.Add(1)
	if f.done != nil {
		select {
		case f.done <- struct{}{}:
		default:
		}
	}
	return nil
}

func waitForWrites(t *testing.T, f *fakeUpdater, want int32) {
	t.Helper()
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if f.calls.Load() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("expected %d writes, got %d", want, f.calls.Load())
}

func TestLastSeenIP_FirstCallWrites(t *testing.T) {
	// given
	f := &fakeUpdater{}
	r := NewLastSeenIP(f, time.Hour)
	uid := uuid.New()

	// when
	r.Record(uid, "10.0.0.1")

	// then
	waitForWrites(t, f, 1)
}

func TestLastSeenIP_SameIPWithinWindowIsSkipped(t *testing.T) {
	// given
	f := &fakeUpdater{}
	r := NewLastSeenIP(f, time.Hour)
	uid := uuid.New()

	// when
	r.Record(uid, "10.0.0.1")
	waitForWrites(t, f, 1)
	for i := 0; i < 50; i++ {
		r.Record(uid, "10.0.0.1")
	}
	time.Sleep(50 * time.Millisecond)

	// then
	assert.Equal(t, int32(1), f.calls.Load())
}

func TestLastSeenIP_ChangedIPWritesImmediately(t *testing.T) {
	// given
	f := &fakeUpdater{}
	r := NewLastSeenIP(f, time.Hour)
	uid := uuid.New()

	// when
	r.Record(uid, "10.0.0.1")
	waitForWrites(t, f, 1)
	r.Record(uid, "10.0.0.2")

	// then
	waitForWrites(t, f, 2)
}

func TestLastSeenIP_WindowElapsedWritesAgain(t *testing.T) {
	// given
	f := &fakeUpdater{}
	r := NewLastSeenIP(f, 10*time.Millisecond)
	uid := uuid.New()

	// when
	r.Record(uid, "10.0.0.1")
	waitForWrites(t, f, 1)
	time.Sleep(20 * time.Millisecond)
	r.Record(uid, "10.0.0.1")

	// then
	waitForWrites(t, f, 2)
}

func TestLastSeenIP_NilUserOrEmptyIPNoOp(t *testing.T) {
	// given
	f := &fakeUpdater{}
	r := NewLastSeenIP(f, time.Hour)

	// when
	r.Record(uuid.Nil, "10.0.0.1")
	r.Record(uuid.New(), "")
	time.Sleep(20 * time.Millisecond)

	// then
	assert.Equal(t, int32(0), f.calls.Load())
}
