package admin

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAddBannedGif_RecordsTheBan(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.bannedRepo.EXPECT().Add(mock.Anything, "gif", "abc123", "spam", mock.Anything).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    actor,
		Action:     repository.AuditActionBannedGifCreate,
		TargetType: repository.AuditTargetBannedGif,
		TargetID:   "abc123",
		Details:    "kind=gif id=abc123",
	}).Return(nil)

	// when
	res, err := svc.AddBannedGif(context.Background(), actor, dto.AddBannedGiphyRequest{Input: "abc123", Reason: "spam"})

	// then
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestRemoveBannedGif_RecordsTheLift(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.bannedRepo.EXPECT().Remove(mock.Anything, "user", "Larperine").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    actor,
		Action:     repository.AuditActionBannedGifDelete,
		TargetType: repository.AuditTargetBannedGif,
		TargetID:   "Larperine",
		Details:    "kind=user id=Larperine",
	}).Return(nil)

	// when
	err := svc.RemoveBannedGif(context.Background(), actor, "user", "Larperine")

	// then
	require.NoError(t, err)
}

func TestRemoveBannedGif_InvalidKindWritesNothing(t *testing.T) {
	// given
	svc, m := newTestService(t)

	// when
	err := svc.RemoveBannedGif(context.Background(), uuid.New(), "sticker", "abc123")

	// then
	require.Error(t, err)
	m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}
