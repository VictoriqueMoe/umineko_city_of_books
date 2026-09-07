package mystery

import (
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func stubAuthor(m *testMocks, id, authorID uuid.UUID) {
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
}

func stubNotAuthor(m *testMocks, id, actorID uuid.UUID) {
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, actorID, authz.PermEditAnyTheory).Return(false)
}

func stubActor(m *testMocks, userID uuid.UUID, displayName string) {
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: displayName}, nil)
}

func stubMentionOf(m *testMocks, actorID, mentionedID uuid.UUID, username string) {
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{username}).Return([]model.User{{ID: mentionedID}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, actorID, mentionedID).Return(false, nil)
}
