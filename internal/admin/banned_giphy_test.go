package admin

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAddBannedGif_RecordsTheBan(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	createdBy := actor.String()
	m.bannedRepo.EXPECT().Add(mock.Anything, spec.NewBannedGiphy{
		Kind:      "gif",
		Value:     "abc123",
		Reason:    "spam",
		CreatedBy: &createdBy,
	}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    actor,
		Action:     audit.ActionBannedGifCreate,
		TargetType: audit.TargetBannedGif,
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
	m.bannedRepo.EXPECT().Remove(mock.Anything, spec.BannedGiphyDeletion{Kind: "user", Value: "Larperine"}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    actor,
		Action:     audit.ActionBannedGifDelete,
		TargetType: audit.TargetBannedGif,
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
