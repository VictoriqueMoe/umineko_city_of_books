package middleware

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"umineko_city_of_books/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestLastSeenIP_Record(t *testing.T) {
	uid := uuid.New()

	cases := []struct {
		name   string
		userID uuid.UUID
		ip     string
		expect func(userSvc *user.MockService)
	}{
		{
			name:   "the ip is handed to the user service",
			userID: uid,
			ip:     "10.0.0.2",
			expect: func(userSvc *user.MockService) {
				userSvc.EXPECT().UpdateIP(mock.Anything, uid, "10.0.0.2").Return(nil).Once()
			},
		},
		{
			name:   "a failed update is only logged",
			userID: uid,
			ip:     "10.0.0.2",
			expect: func(userSvc *user.MockService) {
				userSvc.EXPECT().UpdateIP(mock.Anything, uid, "10.0.0.2").Return(errors.New("db down")).Once()
			},
		},
		{
			name:   "an anonymous request is ignored",
			userID: uuid.Nil,
			ip:     "10.0.0.1",
			expect: func(*user.MockService) {},
		},
		{
			name:   "a request without an ip is ignored",
			userID: uid,
			ip:     "",
			expect: func(*user.MockService) {},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				userSvc := user.NewMockService(t)
				tc.expect(userSvc)

				// when
				NewLastSeenIP(userSvc).Record(context.Background(), tc.userID, tc.ip)

				// then
				synctest.Wait()
				userSvc.AssertExpectations(t)
			})
		})
	}
}
