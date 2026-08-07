package media

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessorShutdown(t *testing.T) {
	tests := []struct {
		name          string
		enqueueBefore bool
		enqueueAfter  bool
		wantReported  int
	}{
		{name: "job still queued when the signal lands is reported", enqueueBefore: true, wantReported: 1},
		{name: "job enqueued after shutdown is rejected", enqueueAfter: true, wantReported: 1},
		{name: "nothing queued drains cleanly", wantReported: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a single worker pinned inside a job, so nothing else can be picked up
			p := NewProcessor(1)
			busy := make(chan struct{})
			release := make(chan struct{})

			p.Enqueue(Job{Type: JobImage, InputPath: "blocker", ErrorCallback: func(error) {
				close(busy)
				<-release
			}})
			<-busy

			var mu sync.Mutex
			var errs []error
			record := func(err error) {
				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, err)
			}

			if tc.enqueueBefore {
				p.Enqueue(Job{Type: JobImage, InputPath: "pending", ErrorCallback: record})
			}

			// when the process is asked to stop while that worker is still busy
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			shutdown := make(chan error, 1)
			go func() {
				shutdown <- p.Shutdown(ctx)
			}()

			require.Eventually(t, func() bool {
				select {
				case <-p.quit:
					return true
				default:
					return false
				}
			}, time.Second, time.Millisecond)

			close(release)
			require.NoError(t, <-shutdown)

			if tc.enqueueAfter {
				p.Enqueue(Job{Type: JobImage, InputPath: "late", ErrorCallback: record})
			}

			// then every job that will never run has told its owner so
			mu.Lock()
			defer mu.Unlock()
			assert.Len(t, errs, tc.wantReported)
			for _, err := range errs {
				assert.ErrorIs(t, err, ErrShuttingDown)
			}
		})
	}
}

func TestProcessorShutdownIsIdempotent(t *testing.T) {
	// given an already drained processor
	p := NewProcessor(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, p.Shutdown(ctx))

	// when it is drained again
	err := p.Shutdown(ctx)

	// then the second call is a no-op rather than a panic on a closed channel
	require.NoError(t, err)
}
