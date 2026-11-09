package upload

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"umineko_city_of_books/internal/authctx"
	"umineko_city_of_books/internal/media"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testQueueLimit = 4
)

func newTestQueues() *userQueues {
	return &userQueues{entries: map[uuid.UUID]*userQueue{}}
}

func holdSlots(t *testing.T, queues *userQueues, userID uuid.UUID, limit int) []func() {
	t.Helper()

	held := make([]func(), 0, limit)
	for range limit {
		leave, err := queues.enter(context.Background(), userID, limit)
		require.NoError(t, err)

		held = append(held, leave)
	}

	return held
}

func TestUserQueues_AJobOverTheLimitWaitsForASlot(t *testing.T) {
	synctest.Test(t, testUserQueuesAJobOverTheLimitWaitsForASlot)
}

func testUserQueuesAJobOverTheLimitWaitsForASlot(t *testing.T) {
	// given a user holding every slot
	queues := newTestQueues()
	userID := uuid.New()
	held := holdSlots(t, queues, userID, testQueueLimit)

	// when one more job arrives
	var (
		entered  atomic.Bool
		enterErr error
	)
	go func() {
		leave, err := queues.enter(context.Background(), userID, testQueueLimit)
		enterErr = err

		if err == nil {
			entered.Store(true)
			leave()
		}
	}()
	synctest.Wait()

	// then it waits
	assert.False(t, entered.Load())

	// and it goes ahead as soon as a slot frees up
	held[0]()
	synctest.Wait()
	require.NoError(t, enterErr)
	assert.True(t, entered.Load())

	for _, leave := range held[1:] {
		leave()
	}
	assert.Empty(t, queues.entries, "a user with nothing queued must not stay in memory")
}

func TestUserQueues_OneUserDoesNotUseUpAnothersSlots(t *testing.T) {
	// given a user holding every slot
	queues := newTestQueues()
	holdSlots(t, queues, uuid.New(), testQueueLimit)

	// when someone else queues every one of theirs
	held := holdSlots(t, queues, uuid.New(), testQueueLimit)

	// then none of them had to wait
	assert.Len(t, held, testQueueLimit)
}

func TestUserQueues_GivingUpWhileWaitingTakesNoSlot(t *testing.T) {
	// given a user holding every slot and a request that has already given up
	queues := newTestQueues()
	userID := uuid.New()
	held := holdSlots(t, queues, userID, testQueueLimit)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// when
	_, err := queues.enter(ctx, userID, testQueueLimit)

	// then
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, testQueueLimit, queues.entries[userID].holders)

	for _, leave := range held {
		leave()
	}
	assert.Empty(t, queues.entries)
}

func TestUserQueues_ANewLimitAppliesOnceTheUserHasDrained(t *testing.T) {
	// given a user who queued under a limit of one and finished
	queues := newTestQueues()
	userID := uuid.New()
	for _, leave := range holdSlots(t, queues, userID, 1) {
		leave()
	}

	// when an admin raises it and they queue again
	held := holdSlots(t, queues, userID, 2)

	// then the second job did not have to wait
	assert.Len(t, held, 2)
}

func TestUserQueues_ANonsenseLimitStillLetsOneJobThrough(t *testing.T) {
	// given a limit no validated setting can hold
	queues := newTestQueues()

	// when
	leave, err := queues.enter(context.Background(), uuid.New(), 0)

	// then the upload is not hung forever
	require.NoError(t, err)
	leave()
}

func TestRunJob_TakesAUserSlotBeforeEnteringThePool(t *testing.T) {
	// given a service with no pool, a user holding every slot and a request that has already given up
	svc, _, _ := newTestService(t)
	userID := uuid.New()
	holdSlots(t, &svc.userQueues, userID, testQueueLimit)
	ctx, cancel := context.WithCancel(authctx.WithUserID(context.Background(), userID))
	cancel()

	// when
	_, err := svc.runJob(ctx, media.Job{Type: media.JobImage})

	// then the job stopped at the user's queue and never reached the pool
	require.ErrorIs(t, err, context.Canceled)
}
