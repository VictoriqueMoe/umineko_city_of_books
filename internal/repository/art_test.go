package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

type (
	artWriterCase struct {
		name   string
		expect func(artDAO *dao.MockArtDAO, audit *MockAuditLogRepository, err error)
		call   func(repo ArtRepository) error
	}
)

func newCachedArtRepo(t *testing.T) (ArtRepository, *dao.MockArtDAO, *MockAuditLogRepository, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	artDAO := dao.NewMockArtDAO(t)
	audit := NewMockAuditLogRepository(t)

	return NewArtRepo(nil, artDAO, nil, audit, cache.NewManager(engines.NewValkeyWithClient(client))), artDAO, audit, client
}

func TestArtRepository_GetCornerCountsIsCached(t *testing.T) {
	cases := []struct {
		name         string
		warm         bool
		wantCommands int
	}{
		{name: "a cold cache loads from the dao and stores the counts", wantCommands: 2},
		{name: "a warm cache serves the counts without the dao", warm: true, wantCommands: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a strict dao mock that only expects a load on a cold cache
			repo, artDAO, _, client := newCachedArtRepo(t)
			reply := settingsMissThenStore
			if tc.warm {
				reply = cachedCornerCounts(`{"general":3}`)
			} else {
				artDAO.EXPECT().GetCornerCounts(mock.Anything).Return(map[string]int{"general": 3}, nil).Once()
			}
			commands := captureSettingsCommands(client, tc.wantCommands, reply)

			// when
			got, err := repo.GetCornerCounts(context.Background())

			// then
			require.NoError(t, err)
			assert.Equal(t, map[string]int{"general": 3}, got)
			assert.Equal(t, []string{"GET", cache.ArtCornerCounts.Key().String()}, (*commands)[0])
		})
	}
}

func artCornerCountWriters(artID, userID, galleryID uuid.UUID) []artWriterCase {
	return []artWriterCase{
		{
			name: "create",
			expect: func(artDAO *dao.MockArtDAO, _ *MockAuditLogRepository, err error) {
				var created *model.ArtRow
				if err == nil {
					created = &model.ArtRow{ID: artID}
				}

				artDAO.EXPECT().CreateArt(mock.Anything, spec.NewArt{UserID: userID, Corner: "general", Title: "t"}, mock.Anything).Return(created, err)
				if err == nil {
					artDAO.EXPECT().InsertTags(mock.Anything, spec.ArtTagInsert{ArtID: artID}, mock.Anything).Return(nil)
				}
			},
			call: func(repo ArtRepository) error {
				_, err := repo.CreateWithTags(context.Background(), spec.NewArtWithTags{UserID: userID, Corner: "general", Title: "t"}, new(sql.Tx))

				return err
			},
		},
		{
			name: "delete",
			expect: func(artDAO *dao.MockArtDAO, audit *MockAuditLogRepository, err error) {
				artDAO.EXPECT().GetArtImagePaths(mock.Anything, artID, mock.Anything).Return(nil, nil)
				artDAO.EXPECT().CollectCommentMediaPaths(mock.Anything, artID, mock.Anything).Return(nil, nil)
				artDAO.EXPECT().Delete(mock.Anything, spec.OwnedDeletion{ID: artID, UserID: userID}, mock.Anything).Return(err)
				if err == nil {
					audit.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				}
			},
			call: func(repo ArtRepository) error {
				_, err := repo.DeleteWithImage(context.Background(), spec.ArtDelete{ID: artID, UserID: userID}, new(sql.Tx))

				return err
			},
		},
		{
			name: "delete gallery",
			expect: func(artDAO *dao.MockArtDAO, _ *MockAuditLogRepository, err error) {
				ref := spec.GalleryRef{GalleryID: galleryID, UserID: userID}
				artDAO.EXPECT().ListGalleryArtImages(mock.Anything, ref, mock.Anything).Return(nil, nil)
				artDAO.EXPECT().DeleteArtInGallery(mock.Anything, ref, mock.Anything).Return(err)
				if err == nil {
					artDAO.EXPECT().DeleteGalleryRow(mock.Anything, ref, mock.Anything).Return(nil)
				}
			},
			call: func(repo ArtRepository) error {
				_, err := repo.DeleteGallery(context.Background(), spec.GalleryRef{GalleryID: galleryID, UserID: userID}, new(sql.Tx))

				return err
			},
		},
	}
}

func TestArtRepository_CornerCountWritersInvalidateTheCount(t *testing.T) {
	artID, userID, galleryID := uuid.New(), uuid.New(), uuid.New()

	for _, tc := range artCornerCountWriters(artID, userID, galleryID) {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, artDAO, audit, client := newCachedArtRepo(t)
			tc.expect(artDAO, audit, nil)

			var commands []string
			captureDel(client, &commands)

			// when
			err := tc.call(repo)

			// then
			require.NoError(t, err)
			assert.Equal(t, []string{"DEL", cache.ArtCornerCounts.Key().String()}, commands)
		})
	}
}

func TestArtRepository_CornerCountWritersSkipInvalidationOnDaoError(t *testing.T) {
	artID, userID, galleryID := uuid.New(), uuid.New(), uuid.New()

	for _, tc := range artCornerCountWriters(artID, userID, galleryID) {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, artDAO, audit, client := newCachedArtRepo(t)
			tc.expect(artDAO, audit, errors.New("db down"))
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

			// when
			err := tc.call(repo)

			// then
			require.Error(t, err)
		})
	}
}
