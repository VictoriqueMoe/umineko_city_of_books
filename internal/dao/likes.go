package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type likeDAO struct {
	db    *sql.DB
	table string
	fk    string
}

func newLikeDAO(db *sql.DB, table string, fk string) *likeDAO {
	return &likeDAO{db: db, table: table, fk: fk}
}

func (l *likeDAO) Like(ctx context.Context, userID uuid.UUID, entityID uuid.UUID) error {
	_, err := l.db.ExecContext(ctx,
		`INSERT INTO `+l.table+` (user_id, `+l.fk+`) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, entityID,
	)
	if err != nil {
		return fmt.Errorf("like in %s: %w", l.table, err)
	}

	return nil
}

func (l *likeDAO) Unlike(ctx context.Context, userID uuid.UUID, entityID uuid.UUID) error {
	_, err := l.db.ExecContext(ctx,
		`DELETE FROM `+l.table+` WHERE user_id = $1 AND `+l.fk+` = $2`,
		userID, entityID,
	)
	if err != nil {
		return fmt.Errorf("unlike in %s: %w", l.table, err)
	}

	return nil
}

func (l *likeDAO) GetLikedBy(ctx context.Context, entityID uuid.UUID, excludeUserIDs []uuid.UUID) ([]model.PostLikeUser, error) {
	exclSQL, exclArgs := ExcludeClause("lk.user_id", excludeUserIDs, 2)

	queryArgs := []interface{}{entityID}
	queryArgs = append(queryArgs, exclArgs...)

	rows, err := l.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')
		FROM `+l.table+` lk
		JOIN users u ON lk.user_id = u.id
		LEFT JOIN user_roles r ON r.user_id = u.id
		WHERE lk.`+l.fk+` = $1`+exclSQL+`
		ORDER BY lk.created_at DESC`,
		queryArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("get liked by in %s: %w", l.table, err)
	}
	defer rows.Close()

	var users []model.PostLikeUser
	for rows.Next() {
		var u model.PostLikeUser
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Role); err != nil {
			return nil, fmt.Errorf("scan like user: %w", err)
		}
		users = append(users, u)
	}

	return users, rows.Err()
}
