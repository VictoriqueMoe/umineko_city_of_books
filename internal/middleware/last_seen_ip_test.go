package middleware

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func newTestLastSeenIP(repo IPWriter) *LastSeenIP {
	return NewLastSeenIP(repo, cache.NewManager(engines.NewInMemory(0)), time.Hour)
}

func TestLastSeenIP_FirstCallWrites(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		repo := NewMockIPWriter(t)
		uid := uuid.New()
		repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.1"}).Return(nil).Once()

		// when
		newTestLastSeenIP(repo).Record(context.Background(), uid, "10.0.0.1")

		// then
		synctest.Wait()
		repo.AssertExpectations(t)
	})
}

func TestLastSeenIP_SameIPWithinWindowIsSkipped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		repo := NewMockIPWriter(t)
		uid := uuid.New()
		repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.1"}).Return(nil).Once()

		r := newTestLastSeenIP(repo)

		// when
		for range 50 {
			r.Record(context.Background(), uid, "10.0.0.1")
		}

		// then
		synctest.Wait()
		repo.AssertExpectations(t)
	})
}

func TestLastSeenIP_ChangedIPWritesImmediately(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		repo := NewMockIPWriter(t)
		uid := uuid.New()
		repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.1"}).Return(nil).Once()
		repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.2"}).Return(nil).Once()

		r := newTestLastSeenIP(repo)

		// when
		r.Record(context.Background(), uid, "10.0.0.1")
		synctest.Wait()

		r.Record(context.Background(), uid, "10.0.0.2")

		// then
		synctest.Wait()
		repo.AssertExpectations(t)
	})
}

func TestLastSeenIP_WindowElapsedWritesAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		repo := NewMockIPWriter(t)
		uid := uuid.New()
		repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.1"}).Return(nil).Twice()

		r := newTestLastSeenIP(repo)

		// when
		r.Record(context.Background(), uid, "10.0.0.1")
		synctest.Wait()

		synctest.Sleep(2 * time.Hour)
		r.Record(context.Background(), uid, "10.0.0.1")

		// then
		synctest.Wait()
		repo.AssertExpectations(t)
	})
}

func TestLastSeenIP_NilUserOrEmptyIPNoOp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		repo := NewMockIPWriter(t)
		r := newTestLastSeenIP(repo)

		// when
		r.Record(context.Background(), uuid.Nil, "10.0.0.1")
		r.Record(context.Background(), uuid.New(), "")

		// then
		synctest.Wait()
		repo.AssertExpectations(t)
	})
}

func TestLastSeenIP_DebounceLivesInTheSharedCache(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given two recorders, as two processes would be, behind one cache
		repo := NewMockIPWriter(t)
		uid := uuid.New()
		repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.1"}).Return(nil).Once()

		shared := cache.NewManager(engines.NewInMemory(0))
		first := NewLastSeenIP(repo, shared, time.Hour)
		second := NewLastSeenIP(repo, shared, time.Hour)

		// when
		first.Record(context.Background(), uid, "10.0.0.1")
		second.Record(context.Background(), uid, "10.0.0.1")

		// then
		synctest.Wait()
		repo.AssertExpectations(t)
	})
}

func TestLastSeenIP_StoreFailureStillWrites(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given a cache with no engine to hold the debounce
		repo := NewMockIPWriter(t)
		uid := uuid.New()
		repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.1"}).Return(nil).Twice()

		r := NewLastSeenIP(repo, cache.NewManager(), time.Hour)

		// when
		r.Record(context.Background(), uid, "10.0.0.1")
		synctest.Wait()

		r.Record(context.Background(), uid, "10.0.0.1")

		// then it fails open and keeps the ip current rather than dropping the write
		synctest.Wait()
		repo.AssertExpectations(t)
	})
}
