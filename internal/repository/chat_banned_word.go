package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	ChatBannedWordRepository interface {
		Create(ctx context.Context, spec ChatBannedWordSpec) (uuid.UUID, error)
		Update(ctx context.Context, id uuid.UUID, spec ChatBannedWordUpdate) error
		Delete(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID) (*ChatBannedWordRow, error)
		ListGlobal(ctx context.Context) ([]ChatBannedWordRow, error)
		ListForRoom(ctx context.Context, roomID uuid.UUID) ([]ChatBannedWordRow, error)
		ListApplicable(ctx context.Context, roomID uuid.UUID) ([]ChatBannedWordRow, error)
	}

	ChatBannedWordSpec struct {
		Scope         string
		RoomID        *uuid.UUID
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
		CreatedBy     *uuid.UUID
	}

	ChatBannedWordUpdate struct {
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
	}

	ChatBannedWordRow struct {
		ID            uuid.UUID
		Scope         string
		RoomID        *uuid.UUID
		Pattern       string
		MatchMode     string
		CaseSensitive bool
		Action        string
		CreatedBy     *uuid.UUID
		CreatedByName string
		CreatedAt     string
	}
)

type chatBannedWordRepository struct {
	dao ChatBannedWordRepository
}

func NewChatBannedWordRepo(dao ChatBannedWordRepository) ChatBannedWordRepository {
	return &chatBannedWordRepository{dao: dao}
}

func (r *chatBannedWordRepository) Create(ctx context.Context, spec ChatBannedWordSpec) (uuid.UUID, error) {
	return r.dao.Create(ctx, spec)
}

func (r *chatBannedWordRepository) Update(ctx context.Context, id uuid.UUID, spec ChatBannedWordUpdate) error {
	return r.dao.Update(ctx, id, spec)
}

func (r *chatBannedWordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.dao.Delete(ctx, id)
}

func (r *chatBannedWordRepository) GetByID(ctx context.Context, id uuid.UUID) (*ChatBannedWordRow, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *chatBannedWordRepository) ListGlobal(ctx context.Context) ([]ChatBannedWordRow, error) {
	return r.dao.ListGlobal(ctx)
}

func (r *chatBannedWordRepository) ListForRoom(ctx context.Context, roomID uuid.UUID) ([]ChatBannedWordRow, error) {
	return r.dao.ListForRoom(ctx, roomID)
}

func (r *chatBannedWordRepository) ListApplicable(ctx context.Context, roomID uuid.UUID) ([]ChatBannedWordRow, error) {
	return r.dao.ListApplicable(ctx, roomID)
}
