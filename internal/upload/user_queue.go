package upload

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
)

type (
	userQueue struct {
		slots   *semaphore.Weighted
		holders int
	}

	userQueues struct {
		mu      sync.Mutex
		entries map[uuid.UUID]*userQueue
	}
)

func (q *userQueues) enter(ctx context.Context, userID uuid.UUID, limit int) (func(), error) {
	q.mu.Lock()
	entry, ok := q.entries[userID]
	if !ok {
		entry = &userQueue{slots: semaphore.NewWeighted(int64(max(limit, 1)))}
		q.entries[userID] = entry
	}
	entry.holders++
	q.mu.Unlock()

	if err := entry.slots.Acquire(ctx, 1); err != nil {
		q.forget(userID, entry)

		return nil, err
	}

	return func() {
		entry.slots.Release(1)
		q.forget(userID, entry)
	}, nil
}

func (q *userQueues) forget(userID uuid.UUID, entry *userQueue) {
	q.mu.Lock()
	defer q.mu.Unlock()

	entry.holders--
	if entry.holders == 0 {
		delete(q.entries, userID)
	}
}
