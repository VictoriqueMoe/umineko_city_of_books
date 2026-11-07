package dao_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPendingUploadDAO_AnUploadMovesFromProcessingToReadyOnce(t *testing.T) {
	// given a processing upload
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	id := uuid.New()
	require.NoError(t, repos.PendingUpload.Insert(ctx, spec.NewPendingUpload{ID: id, UserID: user.ID, MediaType: "video", Filename: "clip.mp4", FileSize: 99}))

	// when
	first, firstErr := repos.PendingUpload.MarkReady(ctx, spec.ProcessedUpload{ID: id, MediaURL: "/uploads/media/clip.mp4", ThumbnailURL: "/uploads/media/thumb_clip.webp"})
	again, againErr := repos.PendingUpload.MarkReady(ctx, spec.ProcessedUpload{ID: id, MediaURL: "/uploads/media/other.mp4"})
	failed, failedErr := repos.PendingUpload.MarkFailed(ctx, spec.UserUpload{ID: id, UserID: user.ID})
	rows, listErr := repos.PendingUpload.List(ctx, spec.UserUploads{UserID: user.ID, IDs: []uuid.UUID{id}})

	// then only the first transition lands
	require.NoError(t, firstErr)
	require.NoError(t, againErr)
	require.NoError(t, failedErr)
	require.NoError(t, listErr)
	assert.True(t, first)
	assert.False(t, again)
	assert.False(t, failed)
	require.Len(t, rows, 1)
	assert.Equal(t, model.PendingUploadReady, rows[0].Status)
	assert.Equal(t, "/uploads/media/clip.mp4", rows[0].MediaURL)
	assert.Equal(t, "/uploads/media/thumb_clip.webp", rows[0].ThumbnailURL)
	assert.Equal(t, "clip.mp4", rows[0].Filename)
	assert.Equal(t, int64(99), rows[0].FileSize)
}

func TestPendingUploadDAO_ListShowsOnlyTheCallersUploads(t *testing.T) {
	// given two users with one upload each
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	mine := daotest.ReadyUpload(t, repos, owner.ID, "image", "/uploads/media/a.webp")
	theirs := daotest.ReadyUpload(t, repos, stranger.ID, "image", "/uploads/media/b.webp")

	// when
	rows, err := repos.PendingUpload.List(ctx, spec.UserUploads{UserID: owner.ID, IDs: []uuid.UUID{mine, theirs}})

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, mine, rows[0].ID)
}

func TestPendingUploadDAO_Claim(t *testing.T) {
	media := []string{"image", "video", "audio"}

	tests := []struct {
		name   string
		claim  func(owner uuid.UUID, ids map[string]uuid.UUID) spec.UploadClaim
		want   []string
		remain []string
	}{
		{
			name: "the caller's ready uploads of an accepted type are claimed and removed",
			claim: func(owner uuid.UUID, ids map[string]uuid.UUID) spec.UploadClaim {
				return spec.UploadClaim{UserID: owner, IDs: []uuid.UUID{ids["picture"], ids["video"]}, MediaTypes: media, CreatedAfter: time.Now().Add(-time.Hour)}
			},
			want:   []string{"picture", "video"},
			remain: []string{"processing", "document", "theirs"},
		},
		{
			name: "an upload still processing is not claimed",
			claim: func(owner uuid.UUID, ids map[string]uuid.UUID) spec.UploadClaim {
				return spec.UploadClaim{UserID: owner, IDs: []uuid.UUID{ids["processing"]}, MediaTypes: media, CreatedAfter: time.Now().Add(-time.Hour)}
			},
			remain: []string{"picture", "video", "processing", "document", "theirs"},
		},
		{
			name: "another user's upload is not claimed",
			claim: func(owner uuid.UUID, ids map[string]uuid.UUID) spec.UploadClaim {
				return spec.UploadClaim{UserID: owner, IDs: []uuid.UUID{ids["theirs"]}, MediaTypes: media, CreatedAfter: time.Now().Add(-time.Hour)}
			},
			remain: []string{"picture", "video", "processing", "document", "theirs"},
		},
		{
			name: "a document is not claimed as media",
			claim: func(owner uuid.UUID, ids map[string]uuid.UUID) spec.UploadClaim {
				return spec.UploadClaim{UserID: owner, IDs: []uuid.UUID{ids["document"]}, MediaTypes: media, CreatedAfter: time.Now().Add(-time.Hour)}
			},
			remain: []string{"picture", "video", "processing", "document", "theirs"},
		},
		{
			name: "an upload older than the cutoff is not claimed",
			claim: func(owner uuid.UUID, ids map[string]uuid.UUID) spec.UploadClaim {
				return spec.UploadClaim{UserID: owner, IDs: []uuid.UUID{ids["picture"]}, MediaTypes: media, CreatedAfter: time.Now().Add(time.Hour)}
			},
			remain: []string{"picture", "video", "processing", "document", "theirs"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			stranger := daotest.CreateUser(t, repos)
			ids := map[string]uuid.UUID{
				"picture":    daotest.ReadyUpload(t, repos, owner.ID, "image", "/uploads/media/a.webp"),
				"video":      daotest.ReadyUpload(t, repos, owner.ID, "video", "/uploads/media/b.mp4"),
				"document":   daotest.ReadyUpload(t, repos, owner.ID, "file", "/uploads/attachments/c.pdf"),
				"theirs":     daotest.ReadyUpload(t, repos, stranger.ID, "image", "/uploads/media/d.webp"),
				"processing": uuid.New(),
			}
			require.NoError(t, repos.PendingUpload.Insert(ctx, spec.NewPendingUpload{ID: ids["processing"], UserID: owner.ID, MediaType: "image", Filename: "e.png", FileSize: 1}))

			// when
			claimed, err := repos.PendingUpload.Claim(ctx, tc.claim(owner.ID, ids))

			// then
			require.NoError(t, err)
			var got []uuid.UUID
			for _, row := range claimed {
				got = append(got, row.ID)
			}
			var want []uuid.UUID
			for _, name := range tc.want {
				want = append(want, ids[name])
			}
			assert.ElementsMatch(t, want, got)

			for _, name := range tc.remain {
				userID := owner.ID
				if name == "theirs" {
					userID = stranger.ID
				}
				rows, err := repos.PendingUpload.List(ctx, spec.UserUploads{UserID: userID, IDs: []uuid.UUID{ids[name]}})
				require.NoError(t, err)
				assert.Len(t, rows, 1, "%s should still be pending", name)
			}
		})
	}
}

func TestPendingUploadDAO_DeleteRemovesOnlyTheCallersUpload(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := daotest.ReadyUpload(t, repos, owner.ID, "image", "/uploads/media/a.webp")

	// when
	byStranger, strangerErr := repos.PendingUpload.Delete(ctx, spec.UserUpload{ID: id, UserID: stranger.ID})
	byOwner, ownerErr := repos.PendingUpload.Delete(ctx, spec.UserUpload{ID: id, UserID: owner.ID})

	// then
	require.NoError(t, strangerErr)
	require.NoError(t, ownerErr)
	assert.Empty(t, byStranger)
	require.Len(t, byOwner, 1)
	assert.Equal(t, "/uploads/media/a.webp", byOwner[0].MediaURL)
}

func TestPendingUploadDAO_DeleteExpiredRemovesEverythingOlderThanTheCutoff(t *testing.T) {
	// given an upload created before the cutoff and one created after it
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	old := daotest.ReadyUpload(t, repos, user.ID, "image", "/uploads/media/old.webp")
	_, err := repos.DB().ExecContext(ctx, `UPDATE pending_uploads SET created_at = now() - interval '2 hours' WHERE id = $1`, old)
	require.NoError(t, err)
	fresh := daotest.ReadyUpload(t, repos, user.ID, "image", "/uploads/media/fresh.webp")

	// when
	expired, err := repos.PendingUpload.DeleteExpired(ctx, time.Now().Add(-time.Hour))

	// then
	require.NoError(t, err)
	require.Len(t, expired, 1)
	assert.Equal(t, old, expired[0].ID)
	rows, err := repos.PendingUpload.List(ctx, spec.UserUploads{UserID: user.ID, IDs: []uuid.UUID{old, fresh}})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, fresh, rows[0].ID)
}

func TestPendingUploadDAO_FailProcessingFailsOnlyUploadsStillProcessing(t *testing.T) {
	// given one processing upload and one ready upload
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	processing := uuid.New()
	require.NoError(t, repos.PendingUpload.Insert(ctx, spec.NewPendingUpload{ID: processing, UserID: user.ID, MediaType: "video", Filename: "clip.mp4", FileSize: 1}))
	ready := daotest.ReadyUpload(t, repos, user.ID, "image", "/uploads/media/a.webp")

	// when
	n, err := repos.PendingUpload.FailProcessing(ctx)

	// then
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	rows, err := repos.PendingUpload.List(ctx, spec.UserUploads{UserID: user.ID, IDs: []uuid.UUID{processing, ready}})
	require.NoError(t, err)
	statuses := map[uuid.UUID]model.PendingUploadStatus{}
	for _, row := range rows {
		statuses[row.ID] = row.Status
	}
	assert.Equal(t, model.PendingUploadFailed, statuses[processing])
	assert.Equal(t, model.PendingUploadReady, statuses[ready])
}
