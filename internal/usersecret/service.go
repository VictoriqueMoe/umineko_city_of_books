package usersecret

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/secrets"

	"github.com/google/uuid"
)

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrHuntAlreadySolved = errors.New("hunt already solved")
)

type (
	UnlockResult struct {
		Spec      *secrets.Spec
		Parent    *secrets.Spec
		HasParent bool
		IsParent  bool
	}

	Service interface {
		ListForUser(ctx context.Context, userID uuid.UUID) ([]string, error)
		IsSolvedByAnyone(ctx context.Context, secretID string) (bool, error)
		GetUserIDsWithSecret(ctx context.Context, secretID string) ([]uuid.UUID, error)
		Unlock(ctx context.Context, userID uuid.UUID, secretRef, phrase string) (*UnlockResult, error)
	}

	service struct {
		repo repository.UserSecretRepository
	}
)

func NewService(repo repository.UserSecretRepository) Service {
	return &service{repo: repo}
}

func (s *service) ListForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return s.repo.ListForUser(ctx, userID)
}

func (s *service) IsSolvedByAnyone(ctx context.Context, secretID string) (bool, error) {
	return s.repo.IsSolvedByAnyone(ctx, secretID)
}

func (s *service) GetUserIDsWithSecret(ctx context.Context, secretID string) ([]uuid.UUID, error) {
	return s.repo.GetUserIDsWithSecret(ctx, secretID)
}

func (s *service) Unlock(ctx context.Context, userID uuid.UUID, secretRef, phrase string) (*UnlockResult, error) {
	spec, ok := secrets.Lookup(secretRef)
	if !ok {
		return nil, ErrInvalidRequest
	}
	sum := sha256.Sum256([]byte(phrase))
	if hex.EncodeToString(sum[:]) != spec.ExpectedHash {
		return nil, ErrInvalidRequest
	}

	parent, hasParent := secrets.ParentOf(spec.ID)
	if hasParent && parent.Title != "" {
		alreadySolved, err := s.repo.IsSolvedByAnyone(ctx, string(parent.ID))
		if err != nil {
			return nil, err
		}
		if alreadySolved {
			return nil, ErrHuntAlreadySolved
		}
	}

	if len(spec.Pieces) > 0 {
		owned, err := s.repo.ListForUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		ownedSet := make(map[string]struct{}, len(owned))
		for _, id := range owned {
			ownedSet[id] = struct{}{}
		}
		for _, piece := range spec.Pieces {
			if _, ok := ownedSet[string(piece.ID)]; !ok {
				return nil, ErrInvalidRequest
			}
		}
	}

	if err := s.repo.Unlock(ctx, userID, string(spec.ID)); err != nil {
		return nil, err
	}

	result := &UnlockResult{
		Spec:      &spec,
		HasParent: hasParent && parent.Title != "",
	}
	if result.HasParent {
		parentCopy := parent
		result.Parent = &parentCopy
		result.IsParent = spec.ID == parent.ID
	}
	return result, nil
}
