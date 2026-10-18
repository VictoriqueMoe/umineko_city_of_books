package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

func (m *mediaDAO) CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	paths, err := m.q.Paths(ctx, m.queries(tx), entityID)
	if err != nil {
		return nil, fmt.Errorf("collect media paths: %w", err)
	}

	return paths, nil
}
