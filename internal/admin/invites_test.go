package admin

import (
	"context"
	"errors"
	"testing"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateInvite_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), actor).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry repository.NewAuditEntry) bool {
		return entry.ActorID == actor && entry.Action == repository.AuditActionCreateInvite && entry.TargetType == repository.AuditTargetInvite && entry.Details == ""
	})).Return(nil)

	// when
	got, err := svc.CreateInvite(context.Background(), actor)

	// then
	require.NoError(t, err)
	assert.Len(t, got.Code, 8)
	assert.Equal(t, actor, got.CreatedBy)
}

func TestCreateInvite_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), actor).Return(errors.New("boom"))

	// when
	_, err := svc.CreateInvite(context.Background(), actor)

	// then
	require.Error(t, err)
}

func TestListInvites_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	m.inviteRepo.EXPECT().List(mock.Anything, 10, 0).Return([]repository.Invite{
		{Code: "abc", CreatedBy: creator, CreatedAt: "t"},
	}, 1, nil)

	// when
	got, err := svc.ListInvites(context.Background(), bounds.NewPage(10, 0))

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	require.Len(t, got.Invites, 1)
	assert.Equal(t, "abc", got.Invites[0].Code)
}

func TestListInvites_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.inviteRepo.EXPECT().List(mock.Anything, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListInvites(context.Background(), bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
}

func TestDeleteInvite_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Delete(mock.Anything, "abc").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionDeleteInvite, TargetType: repository.AuditTargetInvite, TargetID: "abc", Details: ""}).Return(nil)

	// when
	err := svc.DeleteInvite(context.Background(), actor, "abc")

	// then
	require.NoError(t, err)
}

func TestDeleteInvite_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Delete(mock.Anything, "abc").Return(errors.New("boom"))

	// when
	err := svc.DeleteInvite(context.Background(), actor, "abc")

	// then
	require.Error(t, err)
}
