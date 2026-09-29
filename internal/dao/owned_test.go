package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOwnedWrites_AMissingOrForeignRowReadsAsNotFound(t *testing.T) {
	missing := uuid.New()
	user := uuid.New()
	cases := []struct {
		name  string
		write func(ctx context.Context, repos *repository.Repositories) error
	}{
		{name: "owned delete", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Post.Delete(ctx, spec.OwnedDeletion{ID: missing, UserID: user})
		}},
		{name: "art update", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Art.UpdateArt(ctx, spec.ArtUpdate{ID: missing, UserID: user, Title: "t"})
		}},
		{name: "gallery update", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Art.UpdateGallery(ctx, spec.GalleryUpdate{ID: missing, UserID: user, Name: "n"})
		}},
		{name: "gallery delete", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Art.DeleteGalleryRow(ctx, spec.GalleryRef{GalleryID: missing, UserID: user})
		}},
		{name: "journal update", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Journal.Update(ctx, spec.JournalUpdate{ID: missing, UserID: user, Title: "t"})
		}},
		{name: "journal pause", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Journal.SetPaused(ctx, spec.JournalPause{ID: missing, UserID: user, Paused: true})
		}},
		{name: "mystery update", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Mystery.Update(ctx, spec.MysteryOwnerUpdate{ID: missing, UserID: user, Title: "t"})
		}},
		{name: "mystery attempt delete", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Mystery.DeleteAttempt(ctx, spec.MysteryAttemptDeletion{ID: missing, UserID: user})
		}},
		{name: "oc update", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.OC.Update(ctx, spec.OCUpdate{ID: missing, UserID: user, Name: "n"})
		}},
		{name: "fanfic update", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Fanfic.Update(ctx, spec.FanficUpdate{ID: missing, UserID: user, Title: "t"})
		}},
		{name: "post update", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Post.UpdatePost(ctx, spec.PostUpdate{ID: missing, UserID: user, Body: "b"})
		}},
		{name: "ship update", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Ship.UpdateDetails(ctx, spec.ShipDetailsUpdate{ID: missing, UserID: user, Title: "t"})
		}},
		{name: "theory update", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Theory.UpdateTheory(ctx, spec.TheoryUpdate{ID: missing, UserID: user, Title: "t"})
		}},
		{name: "theory delete", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Theory.Delete(ctx, spec.OwnedDeletion{ID: missing, UserID: user})
		}},
		{name: "theory response delete", write: func(ctx context.Context, repos *repository.Repositories) error {
			return repos.Theory.DeleteResponse(ctx, spec.OwnedDeletion{ID: missing, UserID: user})
		}},
	}

	repos := daotest.NewRepos(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			err := tc.write(context.Background(), repos)

			// then
			require.ErrorIs(t, err, dao.ErrNotFound)
		})
	}
}
