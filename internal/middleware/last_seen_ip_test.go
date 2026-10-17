package middleware

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestLastSeenIP_Record(t *testing.T) {
	uid := uuid.New()

	cases := []struct {
		name   string
		userID uuid.UUID
		ip     string
		expect func(repo *MockIPStore)
	}{
		{
			name:   "the same ip as stored writes nothing",
			userID: uid,
			ip:     "10.0.0.1",
			expect: func(repo *MockIPStore) {
				repo.EXPECT().GetIP(mock.Anything, uid).Return("10.0.0.1", nil).Once()
			},
		},
		{
			name:   "a changed ip is written",
			userID: uid,
			ip:     "10.0.0.2",
			expect: func(repo *MockIPStore) {
				repo.EXPECT().GetIP(mock.Anything, uid).Return("10.0.0.1", nil).Once()
				repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.2"}).Return(nil).Once()
			},
		},
		{
			name:   "a user with no ip yet gets one written",
			userID: uid,
			ip:     "10.0.0.1",
			expect: func(repo *MockIPStore) {
				repo.EXPECT().GetIP(mock.Anything, uid).Return("", nil).Once()
				repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.1"}).Return(nil).Once()
			},
		},
		{
			name:   "a failed read writes nothing",
			userID: uid,
			ip:     "10.0.0.1",
			expect: func(repo *MockIPStore) {
				repo.EXPECT().GetIP(mock.Anything, uid).Return("", errors.New("db down")).Once()
			},
		},
		{
			name:   "a failed write is only logged",
			userID: uid,
			ip:     "10.0.0.2",
			expect: func(repo *MockIPStore) {
				repo.EXPECT().GetIP(mock.Anything, uid).Return("10.0.0.1", nil).Once()
				repo.EXPECT().UpdateIP(mock.Anything, spec.UserIPUpdate{UserID: uid, IP: "10.0.0.2"}).Return(errors.New("db down")).Once()
			},
		},
		{
			name:   "an anonymous request is ignored",
			userID: uuid.Nil,
			ip:     "10.0.0.1",
			expect: func(*MockIPStore) {},
		},
		{
			name:   "a request without an ip is ignored",
			userID: uid,
			ip:     "",
			expect: func(*MockIPStore) {},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				repo := NewMockIPStore(t)
				tc.expect(repo)

				// when
				NewLastSeenIP(repo).Record(context.Background(), tc.userID, tc.ip)

				// then
				synctest.Wait()
				repo.AssertExpectations(t)
			})
		})
	}
}
