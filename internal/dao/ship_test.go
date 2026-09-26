package dao_test

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	shipTally struct {
		VoteScore    int
		UserVote     int
		CommentCount int
	}
)

func makeChars() []dto.ShipCharacter {
	return []dto.ShipCharacter{
		{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"},
		{Series: "umineko", CharacterID: "beatrice", CharacterName: "Beatrice"},
	}
}

func createShip(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title string, chars []dto.ShipCharacter) uuid.UUID {
	t.Helper()
	created, err := repos.Ship.CreateWithCharacters(context.Background(), spec.NewShipWithCharacters{
		UserID:      userID,
		Title:       title,
		Description: "desc",
		Characters:  chars,
	})
	require.NoError(t, err)

	return created.ID
}

func createShipComment(t *testing.T, repos *repository.Repositories, shipID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Comments.ByID[string(mention.KindShipComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{
		TargetID: shipID,
		ParentID: parentID,
		UserID:   userID,
		Body:     body,
	})
	require.NoError(t, err)

	return created.ID
}

func shipAddCommentMedia(t *testing.T, repos *repository.Repositories, media spec.NewMedia) int64 {
	t.Helper()
	id, err := repos.Ship.AddCommentMedia(context.Background(), media)
	require.NoError(t, err)

	return id
}

func shipCastVotes(t *testing.T, repos *repository.Repositories, shipID uuid.UUID, values ...int) {
	t.Helper()
	for _, value := range values {
		voter := daotest.CreateUser(t, repos)
		require.NoError(t, repos.Ship.Vote(context.Background(), spec.Vote{UserID: voter.ID, TargetID: shipID, Value: value}))
	}
}

func shipSetCreatedAt(t *testing.T, repos *repository.Repositories, table string, id uuid.UUID, createdAt string) {
	t.Helper()
	_, err := repos.DB().ExecContext(context.Background(), "UPDATE "+table+" SET created_at = $1 WHERE id = $2", createdAt, id)
	require.NoError(t, err)
}

func shipComments(t *testing.T, repos *repository.Repositories, shipID, viewerID uuid.UUID) ([]model.CommentRow, int) {
	t.Helper()
	rows, total, err := repos.Ship.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: shipID, ViewerID: viewerID, Limit: 10, Offset: 0})
	require.NoError(t, err)

	return rows, total
}

func shipAuditEntries(t *testing.T, repos *repository.Repositories, action audit.Action) []audit.Entry {
	t.Helper()
	entries, _, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Action: action, Page: bounds.NewPage(10, 0)})
	require.NoError(t, err)

	return entries
}

func shipRowIDs(rows []model.ShipRow) []uuid.UUID {
	var ids []uuid.UUID
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

func shipCommentIDs(rows []model.CommentRow) []uuid.UUID {
	var ids []uuid.UUID
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

func shipCharacterNames(rows []model.ShipCharacterRow) []string {
	var names []string
	for _, row := range rows {
		names = append(names, row.CharacterName)
	}

	return names
}

func TestShipDAO_CreateWithCharacters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos, daotest.WithUsername("captain_ship"), daotest.WithDisplayName("Captain"))
	viewer := daotest.CreateUser(t, repos)
	chars := []dto.ShipCharacter{
		{Series: "umineko", CharacterID: "battler", CharacterName: "  Padded  "},
		{Series: "umineko", CharacterID: "beatrice", CharacterName: "Beatrice"},
		{Series: "umineko", CharacterID: "ange", CharacterName: "Ange"},
	}

	// when
	created, err := repos.Ship.CreateWithCharacters(ctx, spec.NewShipWithCharacters{
		UserID:      author.ID,
		Title:       "Ship A",
		Description: "About them",
		Characters:  chars,
	})

	// then
	require.NoError(t, err)

	row, err := repos.Ship.GetByID(ctx, spec.ShipLookup{ID: created.ID, ViewerID: viewer.ID})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Ship A", row.Title)
	assert.Equal(t, "About them", row.Description)
	assert.Equal(t, author.ID, row.UserID)
	assert.Equal(t, "captain_ship", row.AuthorUsername)
	assert.Equal(t, "Captain", row.AuthorDisplayName)

	authorID, err := repos.Ship.GetAuthorID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, author.ID, authorID)

	got, err := repos.Ship.GetCharacters(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"Padded", "Beatrice", "Ange"}, shipCharacterNames(got))

	for i, character := range got {
		assert.Equal(t, i, character.SortOrder)
	}
}

func TestShipDAO_UpdateWithCharacters(t *testing.T) {
	cases := []struct {
		name            string
		byOwner         bool
		asAdmin         bool
		wantErr         bool
		wantTitle       string
		wantDescription string
		wantNames       []string
	}{
		{name: "the owner replaces the details and the whole character list", byOwner: true, asAdmin: false, wantErr: false, wantTitle: "New", wantDescription: "ND", wantNames: []string{"Solo"}},
		{name: "an admin may update a ship they do not own", byOwner: false, asAdmin: true, wantErr: false, wantTitle: "New", wantDescription: "ND", wantNames: []string{"Solo"}},
		{name: "a stranger is refused and the ship and its characters are left untouched", byOwner: false, asAdmin: false, wantErr: true, wantTitle: "Old", wantDescription: "desc", wantNames: []string{"Battler", "Beatrice"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			actor := daotest.CreateUser(t, repos)
			if tc.byOwner {
				actor = owner
			}

			id := createShip(t, repos, owner.ID, "Old", makeChars())

			// when
			err := repos.Ship.UpdateWithCharacters(ctx, spec.ShipUpdate{
				ID:          id,
				UserID:      actor.ID,
				Title:       "New",
				Description: "ND",
				AsAdmin:     tc.asAdmin,
				Characters:  []dto.ShipCharacter{{Series: "u", CharacterID: "c", CharacterName: "Solo"}},
			})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			row, err := repos.Ship.GetByID(ctx, spec.ShipLookup{ID: id, ViewerID: owner.ID})
			require.NoError(t, err)
			require.NotNil(t, row)
			assert.Equal(t, tc.wantTitle, row.Title)
			assert.Equal(t, tc.wantDescription, row.Description)

			chars, err := repos.Ship.GetCharacters(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, tc.wantNames, shipCharacterNames(chars))
		})
	}
}

func TestShipDAO_UpdateImage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "T", makeChars())

	// when
	err := repos.Ship.UpdateImage(ctx, spec.ShipImageUpdate{ID: id, ImageURL: "/img.png", ThumbnailURL: "/thumb.png"})

	// then
	require.NoError(t, err)

	row, err := repos.Ship.GetByID(ctx, spec.ShipLookup{ID: id, ViewerID: user.ID})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "/img.png", row.ImageURL)
	assert.Equal(t, "/thumb.png", row.ThumbnailURL)
}

func TestShipDAO_DeleteShip(t *testing.T) {
	commentMedia := []spec.NewMedia{
		{MediaURL: "/uploads/ships/comment.png", MediaType: "image", ThumbnailURL: "/uploads/ships/comment_thumb.png"},
		{MediaURL: "/uploads/ships/comment_two.gif", MediaType: "image", SortOrder: 1},
	}

	cases := []struct {
		name         string
		byOwner      bool
		asAdmin      bool
		cover        string
		coverThumb   string
		commentMedia []spec.NewMedia
		wantErr      bool
		wantPaths    []string
	}{
		{
			name:         "the owner gets back the cover, its thumbnail and every comment media path",
			byOwner:      true,
			asAdmin:      false,
			cover:        "/uploads/ships/cover.png",
			coverThumb:   "/uploads/ships/cover_thumb.png",
			commentMedia: commentMedia,
			wantErr:      false,
			wantPaths: []string{
				"/uploads/ships/cover.png",
				"/uploads/ships/cover_thumb.png",
				"/uploads/ships/comment.png",
				"/uploads/ships/comment_thumb.png",
				"/uploads/ships/comment_two.gif",
			},
		},
		{
			name:         "an admin deleting someone else's ship gets back its comment media paths, the blank cover skipped",
			byOwner:      false,
			asAdmin:      true,
			cover:        "",
			coverThumb:   "",
			commentMedia: commentMedia,
			wantErr:      false,
			wantPaths: []string{
				"/uploads/ships/comment.png",
				"/uploads/ships/comment_thumb.png",
				"/uploads/ships/comment_two.gif",
			},
		},
		{
			name:         "the owner deleting a ship with no cover and no comment media gets back no paths",
			byOwner:      true,
			asAdmin:      false,
			cover:        "",
			coverThumb:   "",
			commentMedia: nil,
			wantErr:      false,
			wantPaths:    nil,
		},
		{
			name:         "a stranger is refused and the ship survives",
			byOwner:      false,
			asAdmin:      false,
			cover:        "/uploads/ships/cover.png",
			coverThumb:   "/uploads/ships/cover_thumb.png",
			commentMedia: commentMedia,
			wantErr:      true,
			wantPaths:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			actor := daotest.CreateUser(t, repos)
			if tc.byOwner {
				actor = owner
			}

			id := createShip(t, repos, owner.ID, "T", makeChars())
			require.NoError(t, repos.Ship.UpdateImage(ctx, spec.ShipImageUpdate{ID: id, ImageURL: tc.cover, ThumbnailURL: tc.coverThumb}))
			commentID := createShipComment(t, repos, id, nil, owner.ID, "c")

			for _, media := range tc.commentMedia {
				media.TargetID = commentID
				shipAddCommentMedia(t, repos, media)
			}

			// when
			paths, err := repos.Ship.DeleteShip(ctx, spec.ShipDeletion{ID: id, UserID: actor.ID, AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.ElementsMatch(t, tc.wantPaths, paths)

			row, err := repos.Ship.GetByID(ctx, spec.ShipLookup{ID: id, ViewerID: owner.ID})
			require.NoError(t, err)
			assert.Equal(t, tc.wantErr, row != nil, "only a refused delete leaves the ship in place")
		})
	}
}

func TestShipDAO_UnknownIDs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	viewer := daotest.CreateUser(t, repos)
	unknown := uuid.New()

	// when
	row, rowErr := repos.Ship.GetByID(ctx, spec.ShipLookup{ID: unknown, ViewerID: viewer.ID})
	_, authorErr := repos.Ship.GetAuthorID(ctx, unknown)
	_, entityErr := repos.Ship.GetCommentEntityID(ctx, unknown)

	// then
	require.NoError(t, rowErr)
	assert.Nil(t, row)
	require.ErrorIs(t, authorErr, dao.ErrNotFound)
	require.Error(t, entityErr)
}

func TestShipDAO_List_SortsAndPages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	split := createShip(t, repos, owner.ID, "Split", makeChars())
	loved := createShip(t, repos, owner.ID, "Loved", makeChars())
	disliked := createShip(t, repos, owner.ID, "Disliked", makeChars())
	newest := createShip(t, repos, owner.ID, "Newest", makeChars())

	shipSetCreatedAt(t, repos, "ships", split, "2020-01-01 00:00:00")
	shipSetCreatedAt(t, repos, "ships", loved, "2021-01-01 00:00:00")
	shipSetCreatedAt(t, repos, "ships", disliked, "2022-01-01 00:00:00")
	shipSetCreatedAt(t, repos, "ships", newest, "2023-01-01 00:00:00")

	require.NoError(t, repos.Ship.Vote(ctx, spec.Vote{UserID: viewer.ID, TargetID: split, Value: 1}))
	require.NoError(t, repos.Ship.Vote(ctx, spec.Vote{UserID: viewer.ID, TargetID: loved, Value: 1}))

	shipCastVotes(t, repos, split, -1)
	shipCastVotes(t, repos, loved, 1)
	shipCastVotes(t, repos, disliked, -1)

	createShipComment(t, repos, disliked, nil, owner.ID, "one")
	createShipComment(t, repos, disliked, nil, owner.ID, "two")
	createShipComment(t, repos, newest, nil, owner.ID, "three")

	tallies := map[uuid.UUID]shipTally{
		split:    {VoteScore: 0, UserVote: 1, CommentCount: 0},
		loved:    {VoteScore: 2, UserVote: 1, CommentCount: 0},
		disliked: {VoteScore: -1, UserVote: 0, CommentCount: 2},
		newest:   {VoteScore: 0, UserVote: 0, CommentCount: 1},
	}

	cases := []struct {
		name   string
		sort   string
		limit  int
		offset int
		want   []uuid.UUID
	}{
		{name: "the default sort lists the newest first", sort: "", limit: 10, offset: 0, want: []uuid.UUID{newest, disliked, loved, split}},
		{name: "a first page holds the limit while the total counts every ship", sort: "", limit: 2, offset: 0, want: []uuid.UUID{newest, disliked}},
		{name: "a later page continues from the offset", sort: "", limit: 2, offset: 2, want: []uuid.UUID{loved, split}},
		{name: "top orders by score, ties newest first", sort: "top", limit: 10, offset: 0, want: []uuid.UUID{loved, newest, split, disliked}},
		{name: "crackship orders by lowest score, ties newest first", sort: "crackship", limit: 10, offset: 0, want: []uuid.UUID{disliked, newest, split, loved}},
		{name: "controversial puts ships with both up and down votes first, ties newest first", sort: "controversial", limit: 10, offset: 0, want: []uuid.UUID{split, newest, disliked, loved}},
		{name: "comments orders by comment count, ties newest first", sort: "comments", limit: 10, offset: 0, want: []uuid.UUID{disliked, newest, loved, split}},
		{name: "old lists the oldest first", sort: "old", limit: 10, offset: 0, want: []uuid.UUID{split, loved, disliked, newest}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			rows, total, err := repos.Ship.List(ctx, spec.ShipListing{ViewerID: viewer.ID, Sort: tc.sort, Limit: tc.limit, Offset: tc.offset})

			// then
			require.NoError(t, err)
			assert.Equal(t, 4, total)
			assert.Equal(t, tc.want, shipRowIDs(rows))

			for _, row := range rows {
				assert.Equal(t, tallies[row.ID], shipTally{VoteScore: row.VoteScore, UserVote: row.UserVote, CommentCount: row.CommentCount})
			}
		})
	}
}

func TestShipDAO_List_Filters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	umineko := createShip(t, repos, owner.ID, "Umi", []dto.ShipCharacter{{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"}})
	higurashi := createShip(t, repos, owner.ID, "Hig", []dto.ShipCharacter{{Series: "higurashi", CharacterID: "rena", CharacterName: "Rena"}})
	crack := createShip(t, repos, owner.ID, "Crack", []dto.ShipCharacter{{Series: "umineko", CharacterID: "beatrice", CharacterName: "Beatrice"}})
	hidden := createShip(t, repos, blocked.ID, "Hidden", []dto.ShipCharacter{{Series: "higurashi", CharacterID: "keiichi", CharacterName: "Keiichi"}})

	shipCastVotes(t, repos, crack, -1, -1, -1, -1)

	cases := []struct {
		name    string
		listing spec.ShipListing
		want    []uuid.UUID
	}{
		{name: "no filter lists every ship", listing: spec.ShipListing{}, want: []uuid.UUID{umineko, higurashi, crack, hidden}},
		{name: "series keeps ships with a character from that series", listing: spec.ShipListing{Series: "umineko"}, want: []uuid.UUID{umineko, crack}},
		{name: "character keeps ships featuring that character", listing: spec.ShipListing{CharacterID: "battler"}, want: []uuid.UUID{umineko}},
		{name: "crackships only keeps ships scored at or below the crackship threshold", listing: spec.ShipListing{CrackshipsOnly: true}, want: []uuid.UUID{crack}},
		{name: "an excluded user's ships leave both the page and the total", listing: spec.ShipListing{ExcludeUserIDs: []uuid.UUID{blocked.ID}}, want: []uuid.UUID{umineko, higurashi, crack}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			listing := tc.listing
			listing.ViewerID = owner.ID
			listing.Limit = 10
			rows, total, err := repos.Ship.List(ctx, listing)

			// then
			require.NoError(t, err)
			assert.Equal(t, len(tc.want), total)
			assert.ElementsMatch(t, tc.want, shipRowIDs(rows))
		})
	}
}

func TestShipDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	oldest := createShip(t, repos, user.ID, "Mine1", makeChars())
	middle := createShip(t, repos, user.ID, "Mine2", makeChars())
	newest := createShip(t, repos, user.ID, "Mine3", makeChars())
	theirs := createShip(t, repos, other.ID, "Theirs", makeChars())

	shipSetCreatedAt(t, repos, "ships", oldest, "2020-01-01 00:00:00")
	shipSetCreatedAt(t, repos, "ships", middle, "2021-01-01 00:00:00")
	shipSetCreatedAt(t, repos, "ships", newest, "2022-01-01 00:00:00")
	shipSetCreatedAt(t, repos, "ships", theirs, "2023-01-01 00:00:00")

	cases := []struct {
		name  string
		limit int
		want  []uuid.UUID
	}{
		{name: "only the user's ships are listed, newest first", limit: 10, want: []uuid.UUID{newest, middle, oldest}},
		{name: "a page holds the limit while the total counts every ship the user owns", limit: 2, want: []uuid.UUID{newest, middle}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			rows, total, err := repos.Ship.ListByUser(ctx, spec.ShipUserListing{UserID: user.ID, ViewerID: user.ID, Limit: tc.limit, Offset: 0})

			// then
			require.NoError(t, err)
			assert.Equal(t, 3, total)
			assert.Equal(t, tc.want, shipRowIDs(rows))

			for _, row := range rows {
				assert.Equal(t, user.ID, row.UserID)
			}
		})
	}
}

func TestShipDAO_GetCharactersBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	a := createShip(t, repos, user.ID, "A", []dto.ShipCharacter{{Series: "u", CharacterID: "x", CharacterName: "X"}})
	b := createShip(t, repos, user.ID, "B", []dto.ShipCharacter{{Series: "u", CharacterID: "y", CharacterName: "Y"}, {Series: "u", CharacterID: "z", CharacterName: "Z"}})

	// when
	got, err := repos.Ship.GetCharactersBatch(context.Background(), []uuid.UUID{a, b})

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"X"}, shipCharacterNames(got[a]))
	assert.Equal(t, []string{"Y", "Z"}, shipCharacterNames(got[b]))
}

func TestShipDAO_BatchLookups_NoIDs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	characters, charactersErr := repos.Ship.GetCharactersBatch(ctx, nil)
	media, mediaErr := repos.Ship.GetCommentMediaBatch(ctx, nil)

	// then
	require.NoError(t, charactersErr)
	assert.Nil(t, characters)
	require.NoError(t, mediaErr)
	assert.Nil(t, media)
}

func TestShipDAO_Vote(t *testing.T) {
	cases := []struct {
		name         string
		otherVotes   []int
		earlierVotes []int
		vote         int
		wantScore    int
		wantUserVote int
	}{
		{name: "a first vote is recorded as the voter's own and counted in the score", otherVotes: nil, earlierVotes: nil, vote: 1, wantScore: 1, wantUserVote: 1},
		{name: "voting again replaces the earlier vote instead of adding to it", otherVotes: nil, earlierVotes: []int{1}, vote: -1, wantScore: -1, wantUserVote: -1},
		{name: "a zero vote clears the earlier vote", otherVotes: nil, earlierVotes: []int{1}, vote: 0, wantScore: 0, wantUserVote: 0},
		{name: "the score sums every user's vote", otherVotes: []int{1, 1, -1}, earlierVotes: nil, vote: 1, wantScore: 2, wantUserVote: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			voter := daotest.CreateUser(t, repos)
			id := createShip(t, repos, owner.ID, "T", makeChars())
			shipCastVotes(t, repos, id, tc.otherVotes...)

			for _, value := range tc.earlierVotes {
				require.NoError(t, repos.Ship.Vote(ctx, spec.Vote{UserID: voter.ID, TargetID: id, Value: value}))
			}

			// when
			err := repos.Ship.Vote(ctx, spec.Vote{UserID: voter.ID, TargetID: id, Value: tc.vote})

			// then
			require.NoError(t, err)

			row, err := repos.Ship.GetByID(ctx, spec.ShipLookup{ID: id, ViewerID: voter.ID})
			require.NoError(t, err)
			require.NotNil(t, row)
			assert.Equal(t, tc.wantScore, row.VoteScore)
			assert.Equal(t, tc.wantUserVote, row.UserVote)
		})
	}
}

func TestShipDAO_CreateComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	parentID := createShipComment(t, repos, shipID, nil, owner.ID, "parent")

	// when
	childID := createShipComment(t, repos, shipID, &parentID, commenter.ID, "child")

	// then
	comments, total := shipComments(t, repos, shipID, owner.ID)
	assert.Equal(t, 2, total)
	require.Len(t, comments, 2)

	byID := make(map[uuid.UUID]model.CommentRow)
	for _, comment := range comments {
		byID[comment.ID] = comment
	}

	require.Contains(t, byID, parentID)
	require.Contains(t, byID, childID)
	assert.Nil(t, byID[parentID].ParentID)
	require.NotNil(t, byID[childID].ParentID)
	assert.Equal(t, parentID, *byID[childID].ParentID)

	for _, id := range []uuid.UUID{parentID, childID} {
		entityID, err := repos.Ship.GetCommentEntityID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, shipID, entityID)
	}

	authorID, err := repos.Ship.GetCommentAuthorID(ctx, childID)
	require.NoError(t, err)
	assert.Equal(t, commenter.ID, authorID)
}

func TestShipDAO_UpdateCommentBody(t *testing.T) {
	cases := []struct {
		name          string
		actorIsAuthor bool
		asAdmin       bool
		wantErr       bool
		wantBody      string
		wantAudited   int
	}{
		{name: "the author edits their own comment and nothing is audited", actorIsAuthor: true, asAdmin: false, wantErr: false, wantBody: "edited", wantAudited: 0},
		{name: "a moderator editing their own comment as admin writes no audit entry", actorIsAuthor: true, asAdmin: true, wantErr: false, wantBody: "edited", wantAudited: 0},
		{name: "a moderator editing someone else's comment is audited against its author", actorIsAuthor: false, asAdmin: true, wantErr: false, wantBody: "edited", wantAudited: 1},
		{name: "a stranger is refused, the comment is left alone and nothing is audited", actorIsAuthor: false, asAdmin: false, wantErr: true, wantBody: "old", wantAudited: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			author := daotest.CreateUser(t, repos)
			actor := daotest.CreateUser(t, repos)
			if tc.actorIsAuthor {
				actor = author
			}

			shipID := createShip(t, repos, author.ID, "T", makeChars())
			commentID := createShipComment(t, repos, shipID, nil, author.ID, "old")

			// when
			err := repos.Ship.UpdateCommentBody(ctx, spec.CommentUpdate{CommentID: commentID, UserID: actor.ID, Body: "edited", AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			comments, _ := shipComments(t, repos, shipID, author.ID)
			require.Len(t, comments, 1)
			assert.Equal(t, tc.wantBody, comments[0].Body)

			entries := shipAuditEntries(t, repos, audit.ActionShipCommentUpdateAdmin)
			require.Len(t, entries, tc.wantAudited)

			for _, entry := range entries {
				assert.Equal(t, actor.ID, entry.ActorID)
				assert.Equal(t, audit.TargetShipComment, entry.TargetType)
				assert.Equal(t, commentID.String(), entry.TargetID)
				require.NotNil(t, entry.SubjectID)
				assert.Equal(t, author.ID, *entry.SubjectID)
				assert.Empty(t, entry.Details)
			}
		})
	}
}

func TestShipDAO_DeleteCommentWithAudit(t *testing.T) {
	targetPaths := []string{
		"/uploads/ships/target.png",
		"/uploads/ships/target_thumb.png",
		"/uploads/ships/target_two.gif",
	}

	cases := []struct {
		name             string
		actorIsAuthor    bool
		asAdmin          bool
		wantErr          bool
		wantPaths        []string
		wantTotal        int
		wantOwnEntries   int
		wantAdminEntries int
	}{
		{name: "the author deletes their own comment, gets back only its media and is audited as the owner", actorIsAuthor: true, asAdmin: false, wantErr: false, wantPaths: targetPaths, wantTotal: 1, wantOwnEntries: 1, wantAdminEntries: 0},
		{name: "a moderator deleting their own comment is audited as the owner, not as an admin", actorIsAuthor: true, asAdmin: true, wantErr: false, wantPaths: targetPaths, wantTotal: 1, wantOwnEntries: 1, wantAdminEntries: 0},
		{name: "a moderator deleting someone else's comment is audited as an admin", actorIsAuthor: false, asAdmin: true, wantErr: false, wantPaths: targetPaths, wantTotal: 1, wantOwnEntries: 0, wantAdminEntries: 1},
		{name: "a stranger is refused, the comment survives and nothing is audited", actorIsAuthor: false, asAdmin: false, wantErr: true, wantPaths: nil, wantTotal: 2, wantOwnEntries: 0, wantAdminEntries: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			author := daotest.CreateUser(t, repos)
			actor := daotest.CreateUser(t, repos)
			if tc.actorIsAuthor {
				actor = author
			}

			shipID := createShip(t, repos, author.ID, "T", makeChars())
			target := createShipComment(t, repos, shipID, nil, author.ID, "target")
			other := createShipComment(t, repos, shipID, nil, author.ID, "other")

			shipAddCommentMedia(t, repos, spec.NewMedia{TargetID: target, MediaURL: "/uploads/ships/target.png", MediaType: "image", ThumbnailURL: "/uploads/ships/target_thumb.png"})
			shipAddCommentMedia(t, repos, spec.NewMedia{TargetID: target, MediaURL: "/uploads/ships/target_two.gif", MediaType: "image", SortOrder: 1})
			shipAddCommentMedia(t, repos, spec.NewMedia{TargetID: other, MediaURL: "/uploads/ships/other.png", MediaType: "image"})

			// when
			paths, err := repos.Ship.DeleteCommentWithAudit(ctx, spec.CommentDeletion{CommentID: target, UserID: actor.ID, AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.ElementsMatch(t, tc.wantPaths, paths)

			comments, total := shipComments(t, repos, shipID, author.ID)
			assert.Equal(t, tc.wantTotal, total)
			assert.Equal(t, tc.wantErr, slices.Contains(shipCommentIDs(comments), target), "only a refused delete leaves the comment in place")

			remaining, err := repos.Ship.GetCommentMedia(ctx, other)
			require.NoError(t, err)
			require.Len(t, remaining, 1)
			assert.Equal(t, "/uploads/ships/other.png", remaining[0].MediaURL)

			own := shipAuditEntries(t, repos, audit.ActionShipCommentDelete)
			admin := shipAuditEntries(t, repos, audit.ActionShipCommentDeleteAdmin)
			require.Len(t, own, tc.wantOwnEntries)
			require.Len(t, admin, tc.wantAdminEntries)

			for _, entry := range slices.Concat(own, admin) {
				assert.Equal(t, actor.ID, entry.ActorID)
				assert.Equal(t, audit.TargetShipComment, entry.TargetType)
				assert.Equal(t, target.String(), entry.TargetID)
				require.NotNil(t, entry.SubjectID)
				assert.Equal(t, author.ID, *entry.SubjectID)
			}
		})
	}
}

func TestShipDAO_GetComments(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	first := createShipComment(t, repos, shipID, nil, owner.ID, "first")
	second := createShipComment(t, repos, shipID, nil, owner.ID, "second")
	hidden := createShipComment(t, repos, shipID, nil, blocked.ID, "hidden")
	last := createShipComment(t, repos, shipID, nil, owner.ID, "last")

	shipSetCreatedAt(t, repos, "ship_comments", first, "2020-01-01 00:00:00")
	shipSetCreatedAt(t, repos, "ship_comments", second, "2021-01-01 00:00:00")
	shipSetCreatedAt(t, repos, "ship_comments", hidden, "2022-01-01 00:00:00")
	shipSetCreatedAt(t, repos, "ship_comments", last, "2023-01-01 00:00:00")

	cases := []struct {
		name      string
		limit     int
		exclude   []uuid.UUID
		wantTotal int
		want      []uuid.UUID
	}{
		{name: "a page holds the limit, oldest first, while the total counts every comment", limit: 2, exclude: nil, wantTotal: 4, want: []uuid.UUID{first, second}},
		{name: "an excluded user's comments leave both the page and the total", limit: 10, exclude: []uuid.UUID{blocked.ID}, wantTotal: 3, want: []uuid.UUID{first, second, last}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			rows, total, err := repos.Ship.GetComments(ctx, spec.CommentQuery[uuid.UUID]{TargetID: shipID, ViewerID: owner.ID, Limit: tc.limit, Offset: 0, ExcludeUserIDs: tc.exclude})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)
			assert.Equal(t, tc.want, shipCommentIDs(rows))
		})
	}
}

func TestShipDAO_CommentLikes(t *testing.T) {
	cases := []struct {
		name       string
		priorLikes int
		act        func(repository.ShipRepository, context.Context, spec.CommentLike, ...*sql.Tx) error
		wantCount  int
		wantLiked  bool
	}{
		{name: "a like is counted and marks the comment as liked by the viewer", priorLikes: 0, act: repository.ShipRepository.LikeComment, wantCount: 1, wantLiked: true},
		{name: "liking an already liked comment is idempotent", priorLikes: 1, act: repository.ShipRepository.LikeComment, wantCount: 1, wantLiked: true},
		{name: "unliking removes the viewer's like", priorLikes: 1, act: repository.ShipRepository.UnlikeComment, wantCount: 0, wantLiked: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			liker := daotest.CreateUser(t, repos)
			shipID := createShip(t, repos, owner.ID, "T", makeChars())
			like := spec.CommentLike{UserID: liker.ID, CommentID: createShipComment(t, repos, shipID, nil, owner.ID, "x")}

			for range tc.priorLikes {
				require.NoError(t, repos.Ship.LikeComment(ctx, like))
			}

			// when
			err := tc.act(repos.Ship, ctx, like)

			// then
			require.NoError(t, err)

			rows, _ := shipComments(t, repos, shipID, liker.ID)
			require.Len(t, rows, 1)
			assert.Equal(t, tc.wantCount, rows[0].LikeCount)
			assert.Equal(t, tc.wantLiked, rows[0].UserLiked)
		})
	}
}

func TestShipDAO_AddCommentMedia(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := createShipComment(t, repos, shipID, nil, user.ID, "x")

	// when
	firstID, firstErr := repos.Ship.AddCommentMedia(ctx, spec.NewMedia{TargetID: commentID, MediaURL: "/a.png", MediaType: "image", ThumbnailURL: "/t.png"})
	secondID, secondErr := repos.Ship.AddCommentMedia(ctx, spec.NewMedia{TargetID: commentID, MediaURL: "/b.png", MediaType: "image"})

	// then
	require.NoError(t, firstErr)
	require.NoError(t, secondErr)
	assert.Greater(t, firstID, int64(0))
	assert.Greater(t, secondID, int64(0))

	media, err := repos.Ship.GetCommentMedia(ctx, commentID)
	require.NoError(t, err)
	require.Len(t, media, 2)
	assert.Equal(t, "/a.png", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
	assert.Equal(t, "/t.png", media[0].ThumbnailURL)
	assert.Equal(t, "/b.png", media[1].MediaURL)
}

func TestShipDAO_UpdateCommentMedia(t *testing.T) {
	cases := []struct {
		name      string
		update    func(repository.ShipRepository, context.Context, spec.MediaURLUpdate, ...*sql.Tx) error
		wantURL   string
		wantThumb string
	}{
		{name: "the media url is replaced and the thumbnail left alone", update: repository.ShipRepository.UpdateCommentMediaURL, wantURL: "/new.png", wantThumb: "/old_thumb.png"},
		{name: "the thumbnail is replaced and the media url left alone", update: repository.ShipRepository.UpdateCommentMediaThumbnail, wantURL: "/old.png", wantThumb: "/new.png"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos)
			shipID := createShip(t, repos, user.ID, "T", makeChars())
			commentID := createShipComment(t, repos, shipID, nil, user.ID, "x")
			id := shipAddCommentMedia(t, repos, spec.NewMedia{TargetID: commentID, MediaURL: "/old.png", MediaType: "image", ThumbnailURL: "/old_thumb.png"})

			// when
			err := tc.update(repos.Ship, ctx, spec.MediaURLUpdate{ID: id, URL: "/new.png"})

			// then
			require.NoError(t, err)

			media, err := repos.Ship.GetCommentMedia(ctx, commentID)
			require.NoError(t, err)
			require.Len(t, media, 1)
			assert.Equal(t, tc.wantURL, media[0].MediaURL)
			assert.Equal(t, tc.wantThumb, media[0].ThumbnailURL)
		})
	}
}

func TestShipDAO_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	c1 := createShipComment(t, repos, shipID, nil, user.ID, "a")
	c2 := createShipComment(t, repos, shipID, nil, user.ID, "b")

	shipAddCommentMedia(t, repos, spec.NewMedia{TargetID: c1, MediaURL: "/a.png", MediaType: "image"})
	shipAddCommentMedia(t, repos, spec.NewMedia{TargetID: c2, MediaURL: "/b1.png", MediaType: "image"})
	shipAddCommentMedia(t, repos, spec.NewMedia{TargetID: c2, MediaURL: "/b2.png", MediaType: "image", SortOrder: 1})

	// when
	got, err := repos.Ship.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	assert.Len(t, got[c1], 1)
	require.Len(t, got[c2], 2)
	assert.Equal(t, "/b1.png", got[c2][0].MediaURL)
}
