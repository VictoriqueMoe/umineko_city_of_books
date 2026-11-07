package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func claimOf(userID uuid.UUID, ids []uuid.UUID, mediaTypes []string, maxAge time.Duration) any {
	return mock.MatchedBy(func(s spec.UploadClaim) bool {
		cutoff := time.Now().Add(-maxAge)

		return s.UserID == userID &&
			assert.ObjectsAreEqual(ids, s.IDs) &&
			assert.ObjectsAreEqual(mediaTypes, s.MediaTypes) &&
			!s.CreatedAfter.After(cutoff) &&
			cutoff.Sub(s.CreatedAfter) < time.Minute
	})
}

func TestClaimMediaRows(t *testing.T) {
	userID := uuid.New()
	first, second := uuid.New(), uuid.New()
	mediaTypes := []string{"image", "video", "audio"}
	uploads := []spec.UploadRef{{ID: first}, {ID: second, IsSpoiler: true}}

	tests := []struct {
		name    string
		uploads []spec.UploadRef
		expect  func(pending *dao.MockPendingUploadDAO, tx *sql.Tx)
		want    []spec.NewMedia
		wantErr error
	}{
		{
			name: "nothing attached claims nothing",
		},
		{
			name:    "the same upload attached twice is refused before anything is claimed",
			uploads: []spec.UploadRef{{ID: first}, {ID: first}},
			wantErr: daoerr.ErrUploadUnavailable,
		},
		{
			name:    "an upload that is missing, still processing or expired fails the whole claim",
			uploads: uploads,
			expect: func(pending *dao.MockPendingUploadDAO, tx *sql.Tx) {
				pending.EXPECT().Claim(mock.Anything, claimOf(userID, []uuid.UUID{first, second}, mediaTypes, time.Hour), []*sql.Tx{tx}).
					Return([]model.PendingUploadRow{{ID: first, MediaURL: "/uploads/media/a.webp", MediaType: "image"}}, nil).Once()
			},
			wantErr: daoerr.ErrUploadUnavailable,
		},
		{
			name:    "claimed uploads become media rows in the order they were attached, with their spoiler flags",
			uploads: uploads,
			expect: func(pending *dao.MockPendingUploadDAO, tx *sql.Tx) {
				pending.EXPECT().Claim(mock.Anything, claimOf(userID, []uuid.UUID{first, second}, mediaTypes, time.Hour), []*sql.Tx{tx}).
					Return([]model.PendingUploadRow{
						{ID: second, MediaURL: "/uploads/media/b.mp4", ThumbnailURL: "/uploads/media/thumb_b.webp", MediaType: "video", Filename: "b.mp4"},
						{ID: first, MediaURL: "/uploads/media/a.webp", MediaType: "image", Filename: "a.png"},
					}, nil).Once()
			},
			want: []spec.NewMedia{
				{MediaURL: "/uploads/media/a.webp", MediaType: "image", Filename: "a.png", SortOrder: 0},
				{MediaURL: "/uploads/media/b.mp4", ThumbnailURL: "/uploads/media/thumb_b.webp", MediaType: "video", Filename: "b.mp4", SortOrder: 1, IsSpoiler: true},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a strict pending upload DAO
			pending := dao.NewMockPendingUploadDAO(t)
			tx := new(sql.Tx)
			if tc.expect != nil {
				tc.expect(pending, tx)
			}

			// when
			rows, err := claimMediaRows(context.Background(), pending, spec.ClaimedUploads{UserID: userID, Uploads: tc.uploads, MediaTypes: mediaTypes, MaxAge: time.Hour}, tx)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.want, rows)
		})
	}
}
