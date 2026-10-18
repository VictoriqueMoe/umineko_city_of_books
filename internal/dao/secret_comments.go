package dao

import (
	"context"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"

	"github.com/google/uuid"
)

type (
	secretCommentTable struct {
		secretCommentQuerier
	}
)

func (secretCommentTable) ByID(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*model.CommentRow, error) {
	row, err := q.GetSecretCommentWithLikes(ctx, id)
	if err != nil {
		return nil, err
	}

	return new(toSecretCommentRow(row)), nil
}
