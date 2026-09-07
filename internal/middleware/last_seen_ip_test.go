package middleware

import (
	"testing"
	"testing/synctest"
	"time"

	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestLastSeenIP_FirstCallWrites(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		repo := NewMockIPWriter(t)
		uid := uuid.New()
		repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.1"}).Return(nil).Once()

		// when
		NewLastSeenIP(repo, time.Hour).Record(uid, "10.0.0.1")

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

		r := NewLastSeenIP(repo, time.Hour)

		// when
		for range 50 {
			r.Record(uid, "10.0.0.1")
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

		r := NewLastSeenIP(repo, time.Hour)

		// when
		r.Record(uid, "10.0.0.1")
		synctest.Wait()

		r.Record(uid, "10.0.0.2")

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

		r := NewLastSeenIP(repo, time.Hour)

		// when
		r.Record(uid, "10.0.0.1")
		synctest.Wait()

		synctest.Sleep(2 * time.Hour)
		r.Record(uid, "10.0.0.1")

		// then
		synctest.Wait()
		repo.AssertExpectations(t)
	})
}

func TestLastSeenIP_NilUserOrEmptyIPNoOp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		repo := NewMockIPWriter(t)
		r := NewLastSeenIP(repo, time.Hour)

		// when
		r.Record(uuid.Nil, "10.0.0.1")
		r.Record(uuid.New(), "")

		// then
		synctest.Wait()
		repo.AssertExpectations(t)
	})
}
