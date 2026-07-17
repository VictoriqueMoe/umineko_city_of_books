package profile

import (
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveUsernames(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		wantQuery []string
		repoUsers []model.User
		want      []string
	}{
		{
			name:      "returns only usernames that exist",
			input:     []string{"alice", "foooobaaaaa"},
			wantQuery: []string{"alice", "foooobaaaaa"},
			repoUsers: []model.User{{ID: uuid.New(), Username: "alice"}},
			want:      []string{"alice"},
		},
		{
			name:      "returns canonical casing from the database",
			input:     []string{"ALICE"},
			wantQuery: []string{"ALICE"},
			repoUsers: []model.User{{ID: uuid.New(), Username: "alice"}},
			want:      []string{"alice"},
		},
		{
			name:      "deduplicates case-insensitively before querying",
			input:     []string{"alice", "Alice", "ALICE"},
			wantQuery: []string{"alice"},
			repoUsers: []model.User{{ID: uuid.New(), Username: "alice"}},
			want:      []string{"alice"},
		},
		{
			name:      "trims whitespace and drops empty entries",
			input:     []string{" alice ", "", "   "},
			wantQuery: []string{"alice"},
			repoUsers: []model.User{{ID: uuid.New(), Username: "alice"}},
			want:      []string{"alice"},
		},
		{
			name:      "no known usernames returns empty",
			input:     []string{"foooobaaaaa"},
			wantQuery: []string{"foooobaaaaa"},
			repoUsers: nil,
			want:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, userRepo, _, _, _, _ := newTestService(t)
			userRepo.EXPECT().GetByUsernames(mock.Anything, tt.wantQuery).Return(tt.repoUsers, nil)

			// when
			got, err := svc.ResolveUsernames(context.Background(), tt.input)

			// then
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveUsernames_EmptyInputSkipsRepo(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)

	// when
	got, err := svc.ResolveUsernames(context.Background(), []string{"", "  "})

	// then
	require.NoError(t, err)
	assert.Empty(t, got)
	userRepo.AssertNotCalled(t, "GetByUsernames", mock.Anything, mock.Anything)
}

func TestResolveUsernames_CapsBatchSize(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	input := make([]string, 0, maxResolveUsernames+50)
	for i := range maxResolveUsernames + 50 {
		input = append(input, "user"+strings.Repeat("x", i%5)+string(rune('a'+i%26))+string(rune('0'+i%10))+string(rune('A'+i%26)))
	}
	var captured []string
	userRepo.EXPECT().GetByUsernames(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, names []string) ([]model.User, error) {
			captured = names
			return nil, nil
		},
	)

	// when
	_, err := svc.ResolveUsernames(context.Background(), input)

	// then
	require.NoError(t, err)
	assert.Len(t, captured, maxResolveUsernames)
}

func TestResolveUsernames_RepoError(t *testing.T) {
	// given
	svc, userRepo, _, _, _, _ := newTestService(t)
	userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return(nil, errors.New("boom"))

	// when
	_, err := svc.ResolveUsernames(context.Background(), []string{"alice"})

	// then
	require.Error(t, err)
}
