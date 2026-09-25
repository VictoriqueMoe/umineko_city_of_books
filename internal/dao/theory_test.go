package dao_test

import (
	"context"
	"strings"
	"testing"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/theory/params"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTheory(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title string) uuid.UUID {
	t.Helper()

	created, err := repos.Theory.Create(context.Background(), spec.NewTheory{UserID: userID, Title: title, Body: "body of " + title, Episode: 1, Series: "umineko"})
	require.NoError(t, err)

	return created.ID
}

func theoryCreateResponse(t *testing.T, repos *repository.Repositories, s spec.NewTheoryResponse) uuid.UUID {
	t.Helper()

	created, err := repos.Theory.CreateResponse(context.Background(), s)
	require.NoError(t, err)

	return created.ID
}

func theoryBackdate(t *testing.T, repos *repository.Repositories, table string, id uuid.UUID, createdAt string) {
	t.Helper()

	_, err := repos.DB().ExecContext(context.Background(), "UPDATE "+table+" SET created_at = $1 WHERE id = $2", createdAt, id)
	require.NoError(t, err)
}

func TestTheoryDAO_Create(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	req := spec.NewTheory{UserID: user.ID, Title: "My Theory", Body: "b", Episode: 1, Series: "umineko", Evidence: []dto.EvidenceInput{
		{AudioID: "a1", Note: "first"},
		{AudioID: "a2", Note: "second", Lang: "ja"},
	}}

	// when
	created, err := repos.Theory.Create(ctx, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)

	ev, err := repos.Theory.GetEvidence(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, ev, 2)
	assert.Equal(t, "a1", ev[0].AudioID)
	assert.Equal(t, 0, ev[0].SortOrder)
	assert.Equal(t, "en", ev[0].Lang, "evidence without a language defaults to en")
	assert.Equal(t, "a2", ev[1].AudioID)
	assert.Equal(t, 1, ev[1].SortOrder)
	assert.Equal(t, "ja", ev[1].Lang)
	assert.Equal(t, ev, created.Evidence)
}

func TestTheoryDAO_GetByID(t *testing.T) {
	t.Run("returns the theory with its author, default credibility, vote score and side counts", func(t *testing.T) {
		// given
		repos := daotest.NewRepos(t)
		ctx := context.Background()
		author := daotest.CreateUser(t, repos, daotest.WithDisplayName("Author"))
		voter := daotest.CreateUser(t, repos)
		responder := daotest.CreateUser(t, repos)
		id := createTheory(t, repos, author.ID, "Title")
		require.NoError(t, repos.Theory.VoteTheory(ctx, spec.Vote{UserID: voter.ID, TargetID: id, Value: 1}))
		theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: id, UserID: responder.ID, Side: "with_love", Body: "yes"})
		theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: id, UserID: responder.ID, Side: "without_love", Body: "no"})

		// when
		got, err := repos.Theory.GetByID(ctx, id)

		// then
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, id, got.ID)
		assert.Equal(t, "Title", got.Title)
		assert.Equal(t, "umineko", got.Series)
		assert.Equal(t, author.ID, got.Author.ID)
		assert.Equal(t, "Author", got.Author.DisplayName)
		assert.InDelta(t, 50.0, got.CredibilityScore, 0.001)
		assert.Equal(t, 1, got.VoteScore)
		assert.Equal(t, 1, got.WithLoveCount)
		assert.Equal(t, 1, got.WithoutLoveCount)
	})

	t.Run("an unknown id maps to nil without an error", func(t *testing.T) {
		// given
		repos := daotest.NewRepos(t)

		// when
		got, err := repos.Theory.GetByID(context.Background(), uuid.New())

		// then
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestTheoryDAO_List(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	voters := []uuid.UUID{daotest.CreateUser(t, repos).ID, daotest.CreateUser(t, repos).ID, daotest.CreateUser(t, repos).ID}

	seeds := []struct {
		theory    spec.NewTheory
		createdAt string
		score     float64
		votes     []int
	}{
		{theory: spec.NewTheory{UserID: author.ID, Title: "Beatrice the Golden", Body: "the witch", Episode: 1, Series: "umineko"}, createdAt: "2024-01-01 00:00:00", score: 10, votes: []int{1, 1}},
		{theory: spec.NewTheory{UserID: author.ID, Title: "Battler", Body: "the detective", Episode: 2, Series: "umineko"}, createdAt: "2024-01-02 00:00:00", score: 90},
		{theory: spec.NewTheory{UserID: author.ID, Title: "Ange", Body: "the detective's sister", Episode: 1, Series: "umineko"}, createdAt: "2024-01-03 00:00:00", score: 30, votes: []int{1, 1, -1}},
		{theory: spec.NewTheory{UserID: other.ID, Title: "Erika", Body: "the witch's piece", Episode: 2, Series: "umineko"}, createdAt: "2024-01-04 00:00:00", score: 70, votes: []int{-1}},
		{theory: spec.NewTheory{UserID: author.ID, Title: "Rika", Body: "the witch of miracles", Episode: 1, Series: "higurashi"}, createdAt: "2024-01-05 00:00:00", score: 50},
	}

	seeded := map[uuid.UUID]spec.NewTheory{}
	for _, seed := range seeds {
		created, err := repos.Theory.Create(ctx, seed.theory)
		require.NoError(t, err)

		require.NoError(t, repos.Theory.UpdateCredibilityScore(ctx, spec.TheoryCredibilityUpdate{TheoryID: created.ID, Score: seed.score}))
		theoryBackdate(t, repos, "theories", created.ID, seed.createdAt)

		for i, value := range seed.votes {
			require.NoError(t, repos.Theory.VoteTheory(ctx, spec.Vote{UserID: voters[i], TargetID: created.ID, Value: value}))
		}

		seeded[created.ID] = seed.theory
	}

	cases := []struct {
		name      string
		sort      string
		episode   int
		authorID  uuid.UUID
		search    string
		series    string
		limit     int
		offset    int
		exclude   []uuid.UUID
		want      []string
		wantTotal int
	}{
		{name: "new puts the newest first", sort: "new", want: []string{"Erika", "Ange", "Battler", "Beatrice the Golden"}, wantTotal: 4},
		{name: "old puts the oldest first", sort: "old", want: []string{"Beatrice the Golden", "Battler", "Ange", "Erika"}, wantTotal: 4},
		{name: "credibility puts the highest score first", sort: "credibility", want: []string{"Battler", "Erika", "Ange", "Beatrice the Golden"}, wantTotal: 4},
		{name: "credibility_asc puts the lowest score first", sort: "credibility_asc", want: []string{"Beatrice the Golden", "Ange", "Erika", "Battler"}, wantTotal: 4},
		{name: "popular puts the highest vote sum first", sort: "popular", want: []string{"Beatrice the Golden", "Ange", "Battler", "Erika"}, wantTotal: 4},
		{name: "popular_asc puts the lowest vote sum first", sort: "popular_asc", want: []string{"Erika", "Battler", "Ange", "Beatrice the Golden"}, wantTotal: 4},
		{name: "controversial puts the most votes cast first", sort: "controversial", want: []string{"Ange", "Beatrice the Golden", "Erika", "Battler"}, wantTotal: 4},
		{name: "controversial_asc puts the fewest votes cast first", sort: "controversial_asc", want: []string{"Battler", "Erika", "Beatrice the Golden", "Ange"}, wantTotal: 4},
		{name: "the series filter keeps only that series", series: "higurashi", want: []string{"Rika"}, wantTotal: 1},
		{name: "the episode filter keeps only that episode", episode: 2, want: []string{"Erika", "Battler"}, wantTotal: 2},
		{name: "the author filter keeps only that author's theories", authorID: author.ID, want: []string{"Ange", "Battler", "Beatrice the Golden"}, wantTotal: 3},
		{name: "the search filter matches titles", search: "Golden", want: []string{"Beatrice the Golden"}, wantTotal: 1},
		{name: "the search filter matches bodies", search: "detective", want: []string{"Ange", "Battler"}, wantTotal: 2},
		{name: "excluded users' theories are left out", exclude: []uuid.UUID{other.ID}, want: []string{"Ange", "Battler", "Beatrice the Golden"}, wantTotal: 3},
		{name: "the first page stops at the limit while the total counts every match", limit: 3, offset: 0, want: []string{"Erika", "Ange", "Battler"}, wantTotal: 4},
		{name: "the last page holds only the remainder", limit: 3, offset: 3, want: []string{"Beatrice the Golden"}, wantTotal: 4},
		{name: "a filter matching nothing returns an empty page and a zero total", episode: 8, want: nil, wantTotal: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			rows, total, err := repos.Theory.List(ctx, spec.TheoryListFilter{
				Params:         params.NewListParams(tc.sort, tc.episode, tc.authorID, tc.search, tc.series, tc.limit, tc.offset),
				ViewerID:       uuid.Nil,
				ExcludeUserIDs: tc.exclude,
			})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)

			var titles []string
			for _, row := range rows {
				seed, ok := seeded[row.ID]
				require.True(t, ok, "row %s was never seeded", row.ID)
				assert.Equal(t, seed.Title, row.Title)
				assert.Equal(t, seed.Body, row.Body)
				assert.Equal(t, seed.Series, row.Series)
				assert.Equal(t, seed.Episode, row.Episode)
				assert.Equal(t, seed.UserID, row.Author.ID)

				titles = append(titles, row.Title)
			}
			assert.Equal(t, tc.want, titles)
		})
	}
}

func TestTheoryDAO_List_ClipsLongBodiesAndAddsVotesAndSideCounts(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	created, err := repos.Theory.Create(ctx, spec.NewTheory{UserID: author.ID, Title: "long", Body: strings.Repeat("x", 250), Series: "umineko"})
	require.NoError(t, err)
	require.NoError(t, repos.Theory.VoteTheory(ctx, spec.Vote{UserID: voter.ID, TargetID: created.ID, Value: -1}))
	theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: created.ID, UserID: voter.ID, Side: "with_love", Body: "yes"})

	// when
	rows, _, err := repos.Theory.List(ctx, spec.TheoryListFilter{Params: params.NewListParams("new", 0, uuid.Nil, "", "umineko", 20, 0), ViewerID: voter.ID})

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, strings.Repeat("x", 200)+"...", rows[0].Body)
	assert.Equal(t, -1, rows[0].UserVote)
	assert.Equal(t, -1, rows[0].VoteScore)
	assert.Equal(t, 1, rows[0].WithLoveCount)
	assert.Equal(t, 0, rows[0].WithoutLoveCount)
}

func TestTheoryDAO_Update(t *testing.T) {
	cases := []struct {
		name        string
		editor      string
		unknown     bool
		wantErr     bool
		wantTitle   string
		wantBody    string
		wantEpisode int
		wantAudio   string
	}{
		{name: "the owner rewrites the theory and replaces its evidence", editor: "owner", wantTitle: "new", wantBody: "newbody", wantEpisode: 5, wantAudio: "x"},
		{name: "an admin rewrites someone else's theory", editor: "admin", wantTitle: "new", wantBody: "newbody", wantEpisode: 5, wantAudio: "x"},
		{name: "a non-owner cannot rewrite the theory", editor: "other", wantErr: true, wantTitle: "old", wantBody: "oldbody", wantEpisode: 1, wantAudio: "old"},
		{name: "an admin edit of an unknown theory fails", editor: "admin", unknown: true, wantErr: true, wantTitle: "old", wantBody: "oldbody", wantEpisode: 1, wantAudio: "old"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			created, err := repos.Theory.Create(ctx, spec.NewTheory{UserID: owner.ID, Title: "old", Body: "oldbody", Episode: 1, Evidence: []dto.EvidenceInput{{AudioID: "old"}}})
			require.NoError(t, err)

			editors := map[string]uuid.UUID{"owner": owner.ID, "other": other.ID, "admin": uuid.Nil}
			target := created.ID
			if tc.unknown {
				target = uuid.New()
			}

			// when
			err = repos.Theory.Update(ctx, spec.TheoryUpdate{
				ID:       target,
				UserID:   editors[tc.editor],
				Title:    "new",
				Body:     "newbody",
				Episode:  5,
				AsAdmin:  tc.editor == "admin",
				Evidence: []dto.EvidenceInput{{AudioID: "x", Note: "n"}},
			})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			got, err := repos.Theory.GetByID(ctx, created.ID)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.wantTitle, got.Title)
			assert.Equal(t, tc.wantBody, got.Body)
			assert.Equal(t, tc.wantEpisode, got.Episode)

			ev, err := repos.Theory.GetEvidence(ctx, created.ID)
			require.NoError(t, err)
			require.Len(t, ev, 1)
			assert.Equal(t, tc.wantAudio, ev[0].AudioID)
		})
	}
}

func TestTheoryDAO_Update_KeepsEachEvidenceLanguage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	created, err := repos.Theory.Create(ctx, spec.NewTheory{UserID: owner.ID, Title: "t", Body: "b", Episode: 1, Evidence: []dto.EvidenceInput{{AudioID: "old", Lang: "ja"}}})
	require.NoError(t, err)

	// when
	err = repos.Theory.Update(ctx, spec.TheoryUpdate{
		ID:       created.ID,
		UserID:   owner.ID,
		Title:    "t",
		Body:     "b",
		Episode:  1,
		Evidence: []dto.EvidenceInput{{AudioID: "japanese", Lang: "ja"}, {AudioID: "unmarked"}},
	})

	// then
	require.NoError(t, err)

	ev, err := repos.Theory.GetEvidence(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, ev, 2)
	assert.Equal(t, "japanese", ev[0].AudioID)
	assert.Equal(t, "ja", ev[0].Lang, "an edit must keep the language the evidence was quoted in")
	assert.Equal(t, "unmarked", ev[1].AudioID)
	assert.Equal(t, "en", ev[1].Lang, "evidence without a language still defaults to en")
}

func TestTheoryDAO_Delete(t *testing.T) {
	cases := []struct {
		name    string
		deleter string
		unknown bool
		wantErr bool
	}{
		{name: "the owner deletes their theory", deleter: "owner"},
		{name: "a non-owner cannot delete the theory", deleter: "other", wantErr: true},
		{name: "an admin deletes any theory", deleter: "admin"},
		{name: "an admin delete of an unknown theory fails", deleter: "admin", unknown: true, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			id := createTheory(t, repos, owner.ID, "x")

			deleters := map[string]uuid.UUID{"owner": owner.ID, "other": other.ID}
			target := id
			if tc.unknown {
				target = uuid.New()
			}

			// when
			var err error
			if tc.deleter == "admin" {
				err = repos.Theory.DeleteAsAdmin(ctx, target)
			} else {
				err = repos.Theory.Delete(ctx, spec.OwnedDeletion{ID: target, UserID: deleters[tc.deleter]})
			}

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			got, err := repos.Theory.GetByID(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, tc.wantErr, got != nil, "only a rejected delete leaves the theory in place")
		})
	}
}

func TestTheoryDAO_CreateResponse(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	responder := daotest.CreateUser(t, repos)
	tid := createTheory(t, repos, author.ID, "t")
	req := spec.NewTheoryResponse{TheoryID: tid, UserID: responder.ID, Side: "with_love", Body: "yes", Evidence: []dto.EvidenceInput{
		{AudioID: "first", Note: "1"},
		{AudioID: "second", Note: "2", Lang: "ja"},
	}}

	// when
	created, err := repos.Theory.CreateResponse(ctx, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)

	ev, err := repos.Theory.GetResponseEvidence(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, ev, 2)
	assert.Equal(t, "first", ev[0].AudioID)
	assert.Equal(t, "en", ev[0].Lang, "evidence without a language defaults to en")
	assert.Equal(t, "second", ev[1].AudioID)
	assert.Equal(t, "ja", ev[1].Lang)
	assert.Equal(t, ev, created.Evidence)
}

func TestTheoryDAO_DeleteResponse(t *testing.T) {
	cases := []struct {
		name     string
		deleter  string
		unknown  bool
		wantErr  bool
		wantLeft int
	}{
		{name: "the responder deletes their response", deleter: "responder", wantLeft: 0},
		{name: "another user cannot delete the response", deleter: "other", wantErr: true, wantLeft: 1},
		{name: "an admin deletes any response", deleter: "admin", wantLeft: 0},
		{name: "an admin delete of an unknown response fails", deleter: "admin", unknown: true, wantErr: true, wantLeft: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			author := daotest.CreateUser(t, repos)
			responder := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			tid := createTheory(t, repos, author.ID, "t")
			rid := theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: tid, UserID: responder.ID, Side: "with_love", Body: "x"})

			deleters := map[string]uuid.UUID{"responder": responder.ID, "other": other.ID}
			target := rid
			if tc.unknown {
				target = uuid.New()
			}

			// when
			var err error
			if tc.deleter == "admin" {
				err = repos.Theory.DeleteResponseAsAdmin(ctx, target)
			} else {
				err = repos.Theory.DeleteResponse(ctx, spec.OwnedDeletion{ID: target, UserID: deleters[tc.deleter]})
			}

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			resps, err := repos.Theory.GetResponses(ctx, spec.TheoryResponseQuery{TheoryID: tid, ViewerID: uuid.Nil})
			require.NoError(t, err)
			assert.Len(t, resps, tc.wantLeft)
		})
	}
}

func TestTheoryDAO_GetResponses(t *testing.T) {
	cases := []struct {
		name         string
		asVoter      bool
		wantUserVote int
	}{
		{name: "an anonymous viewer gets the reply tree with evidence and vote scores", asVoter: false, wantUserVote: 0},
		{name: "a viewer who voted also sees their own vote", asVoter: true, wantUserVote: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			author := daotest.CreateUser(t, repos)
			responder := daotest.CreateUser(t, repos)
			voter := daotest.CreateUser(t, repos)
			tid := createTheory(t, repos, author.ID, "t")
			parent := theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: tid, UserID: responder.ID, Side: "with_love", Body: "parent", Evidence: []dto.EvidenceInput{{AudioID: "ev", Note: "n"}}})
			theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: tid, UserID: responder.ID, ParentID: &parent, Side: "with_love", Body: "child"})
			require.NoError(t, repos.Theory.VoteResponse(ctx, spec.Vote{UserID: voter.ID, TargetID: parent, Value: 1}))

			viewer := uuid.Nil
			if tc.asVoter {
				viewer = voter.ID
			}

			// when
			rows, err := repos.Theory.GetResponses(ctx, spec.TheoryResponseQuery{TheoryID: tid, ViewerID: viewer})

			// then
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, parent, rows[0].ID)
			assert.Equal(t, 1, rows[0].VoteScore)
			assert.Equal(t, tc.wantUserVote, rows[0].UserVote)
			require.Len(t, rows[0].Evidence, 1)
			assert.Equal(t, "ev", rows[0].Evidence[0].AudioID)

			require.Len(t, rows[0].Replies, 1)
			reply := rows[0].Replies[0]
			assert.Equal(t, "child", reply.Body)
			assert.Equal(t, responder.ID, reply.Author.ID)
			require.NotNil(t, reply.ParentID)
			assert.Equal(t, parent, *reply.ParentID)
		})
	}
}

func TestTheoryDAO_ListAndGetResponses_SurfaceAFailedBatchQuery(t *testing.T) {
	cases := []struct {
		name     string
		sabotage string
		read     func(ctx context.Context, repos *repository.Repositories, theoryID, viewerID uuid.UUID) error
	}{
		{
			name:     "the list fails when the vote scores cannot be read",
			sabotage: `DROP TABLE theory_votes`,
			read:     theoryListRead,
		},
		{
			name:     "the list fails when the reply side counts cannot be read",
			sabotage: `ALTER TABLE responses RENAME COLUMN side TO side_gone`,
			read:     theoryListRead,
		},
		{
			name:     "the list fails when the viewer's own votes cannot be read",
			sabotage: `ALTER TABLE theory_votes DROP COLUMN user_id`,
			read:     theoryListRead,
		},
		{
			name:     "the replies fail when their vote scores cannot be read",
			sabotage: `DROP TABLE response_votes`,
			read:     theoryResponsesRead,
		},
		{
			name:     "the replies fail when the viewer's own reply votes cannot be read",
			sabotage: `ALTER TABLE response_votes DROP COLUMN user_id`,
			read:     theoryResponsesRead,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			author := daotest.CreateUser(t, repos)
			viewer := daotest.CreateUser(t, repos)
			theoryID := createTheory(t, repos, author.ID, "t")
			theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: theoryID, UserID: author.ID, Side: "with_love", Body: "reply"})

			_, err := repos.DB().ExecContext(ctx, tc.sabotage)
			require.NoError(t, err)

			// when
			err = tc.read(ctx, repos, theoryID, viewer.ID)

			// then
			require.Error(t, err, "a failed batch query must not be rendered as zero scores")
		})
	}
}

func theoryListRead(ctx context.Context, repos *repository.Repositories, _, viewerID uuid.UUID) error {
	_, _, err := repos.Theory.List(ctx, spec.TheoryListFilter{Params: params.NewListParams("new", 0, uuid.Nil, "", "umineko", 20, 0), ViewerID: viewerID})

	return err
}

func theoryResponsesRead(ctx context.Context, repos *repository.Repositories, theoryID, viewerID uuid.UUID) error {
	_, err := repos.Theory.GetResponses(ctx, spec.TheoryResponseQuery{TheoryID: theoryID, ViewerID: viewerID})

	return err
}

func TestTheoryDAO_Vote(t *testing.T) {
	cases := []struct {
		name  string
		votes []int
		want  int
	}{
		{name: "no vote reads as zero", votes: nil, want: 0},
		{name: "a first vote is stored", votes: []int{1}, want: 1},
		{name: "a second vote replaces the first", votes: []int{1, -1}, want: -1},
		{name: "a zero vote clears the stored vote", votes: []int{1, 0}, want: 0},
	}

	for _, tc := range cases {
		t.Run("on a theory: "+tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			author := daotest.CreateUser(t, repos)
			voter := daotest.CreateUser(t, repos)
			tid := createTheory(t, repos, author.ID, "t")

			// when
			for _, value := range tc.votes {
				require.NoError(t, repos.Theory.VoteTheory(ctx, spec.Vote{UserID: voter.ID, TargetID: tid, Value: value}))
			}

			// then
			got, err := repos.Theory.GetUserTheoryVote(ctx, spec.TheoryVoteLookup{UserID: voter.ID, TheoryID: tid})
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})

		t.Run("on a response: "+tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			author := daotest.CreateUser(t, repos)
			responder := daotest.CreateUser(t, repos)
			voter := daotest.CreateUser(t, repos)
			tid := createTheory(t, repos, author.ID, "t")
			rid := theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: tid, UserID: responder.ID, Side: "with_love", Body: "x"})

			// when
			for _, value := range tc.votes {
				require.NoError(t, repos.Theory.VoteResponse(ctx, spec.Vote{UserID: voter.ID, TargetID: rid, Value: value}))
			}

			// then
			rows, err := repos.Theory.GetResponses(ctx, spec.TheoryResponseQuery{TheoryID: tid, ViewerID: voter.ID})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, tc.want, rows[0].UserVote)
		})
	}
}

func TestTheoryDAO_Lookups(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	responder := daotest.CreateUser(t, repos)

	stored, err := repos.Theory.Create(ctx, spec.NewTheory{UserID: author.ID, Title: "MyTitle", Body: "b", Series: "higurashi"})
	require.NoError(t, err)

	unset, err := repos.Theory.Create(ctx, spec.NewTheory{UserID: author.ID, Title: "T", Body: "B", Episode: 1})
	require.NoError(t, err)

	parentID := theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: stored.ID, UserID: responder.ID, Side: "with_love", Body: "p"})
	replyID := theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: stored.ID, UserID: responder.ID, ParentID: &parentID, Side: "with_love", Body: "c"})

	t.Run("existing ids resolve to their author, title, series and theory", func(t *testing.T) {
		// when
		authorID, err := repos.Theory.GetTheoryAuthorID(ctx, stored.ID)
		require.NoError(t, err)

		title, err := repos.Theory.GetTheoryTitle(ctx, stored.ID)
		require.NoError(t, err)

		series, err := repos.Theory.GetTheorySeries(ctx, stored.ID)
		require.NoError(t, err)

		defaultedSeries, err := repos.Theory.GetTheorySeries(ctx, unset.ID)
		require.NoError(t, err)

		parentAuthor, parentTheory, err := repos.Theory.GetResponseInfo(ctx, parentID)
		require.NoError(t, err)

		replyAuthor, replyTheory, err := repos.Theory.GetResponseInfo(ctx, replyID)
		require.NoError(t, err)

		// then
		assert.Equal(t, author.ID, authorID)
		assert.Equal(t, "MyTitle", title)
		assert.Equal(t, "higurashi", series)
		assert.Equal(t, "umineko", defaultedSeries, "a theory created without a series defaults to umineko")
		assert.Equal(t, responder.ID, parentAuthor)
		assert.Equal(t, stored.ID, parentTheory)
		assert.Equal(t, responder.ID, replyAuthor)
		assert.Equal(t, stored.ID, replyTheory)
	})

	t.Run("an unknown id reads as not found in every lookup", func(t *testing.T) {
		// given
		unknown := uuid.New()

		// when
		_, authorErr := repos.Theory.GetTheoryAuthorID(ctx, unknown)
		_, titleErr := repos.Theory.GetTheoryTitle(ctx, unknown)
		_, seriesErr := repos.Theory.GetTheorySeries(ctx, unknown)
		_, _, infoErr := repos.Theory.GetResponseInfo(ctx, unknown)
		_, metaErr := repos.Theory.GetResponseMeta(ctx, unknown)

		// then
		assert.ErrorIs(t, authorErr, dao.ErrNotFound)
		assert.ErrorIs(t, titleErr, dao.ErrNotFound)
		assert.ErrorIs(t, seriesErr, dao.ErrNotFound)
		assert.ErrorIs(t, infoErr, dao.ErrNotFound)
		assert.ErrorIs(t, metaErr, dao.ErrNotFound)
	})
}

func TestTheoryDAO_GetRecentActivityByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	idle := daotest.CreateUser(t, repos)

	first := createTheory(t, repos, user.ID, "First")
	theoryBackdate(t, repos, "theories", first, "2024-01-01 00:00:00")
	reply := theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: first, UserID: user.ID, Side: "with_love", Body: "resp"})
	theoryBackdate(t, repos, "responses", reply, "2024-01-02 00:00:00")
	second := createTheory(t, repos, user.ID, "Second")
	theoryBackdate(t, repos, "theories", second, "2024-01-03 00:00:00")
	third := createTheory(t, repos, user.ID, "Third")
	theoryBackdate(t, repos, "theories", third, "2024-01-04 00:00:00")

	createTheory(t, repos, other.ID, "Not mine")
	theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: first, UserID: other.ID, Side: "without_love", Body: "theirs"})

	cases := []struct {
		name      string
		userID    uuid.UUID
		limit     int
		offset    int
		want      []string
		wantTotal int
	}{
		{name: "the first page holds the user's newest theories and responses", userID: user.ID, limit: 2, offset: 0, want: []string{"theory:Third", "theory:Second"}, wantTotal: 4},
		{name: "the next page continues from the offset", userID: user.ID, limit: 2, offset: 2, want: []string{"response:First", "theory:First"}, wantTotal: 4},
		{name: "a user with no theories or responses has no activity", userID: idle.ID, limit: 10, offset: 0, want: nil, wantTotal: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			items, total, err := repos.Theory.GetRecentActivityByUser(ctx, spec.UserActivityQuery{UserID: tc.userID, Limit: tc.limit, Offset: tc.offset})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)

			var got []string
			for _, item := range items {
				got = append(got, item.Type+":"+item.TheoryTitle)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTheoryDAO_CountUserTheoriesAndResponsesToday(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)

	tid := createTheory(t, repos, user.ID, "a")
	createTheory(t, repos, user.ID, "b")
	createTheory(t, repos, other.ID, "c")
	staleTheory := createTheory(t, repos, user.ID, "last year")
	theoryBackdate(t, repos, "theories", staleTheory, "2024-01-01 00:00:00")

	theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: tid, UserID: user.ID, Side: "with_love", Body: "x"})
	theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: tid, UserID: user.ID, Side: "without_love", Body: "y"})
	theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: tid, UserID: other.ID, Side: "with_love", Body: "z"})
	staleResponse := theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: tid, UserID: user.ID, Side: "with_love", Body: "last year"})
	theoryBackdate(t, repos, "responses", staleResponse, "2024-01-01 00:00:00")

	// when
	theories, err := repos.Theory.CountUserTheoriesToday(ctx, user.ID)
	require.NoError(t, err)

	responses, err := repos.Theory.CountUserResponsesToday(ctx, user.ID)
	require.NoError(t, err)

	// then
	assert.Equal(t, 2, theories)
	assert.Equal(t, 2, responses)
}

func TestTheoryDAO_UpdateCredibilityScore(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	tid := createTheory(t, repos, user.ID, "t")

	// when
	err := repos.Theory.UpdateCredibilityScore(ctx, spec.TheoryCredibilityUpdate{TheoryID: tid, Score: 77.5})

	// then
	require.NoError(t, err)

	got, err := repos.Theory.GetByID(ctx, tid)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.InDelta(t, 77.5, got.CredibilityScore, 0.001)
}

func TestTheoryDAO_GetResponseEvidenceWeights(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	responder := daotest.CreateUser(t, repos)

	bothSides := createTheory(t, repos, author.ID, "both sides")
	theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: bothSides, UserID: responder.ID, Side: "with_love", Body: "wl", Evidence: []dto.EvidenceInput{{AudioID: "a"}, {AudioID: "b"}}})
	theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: bothSides, UserID: responder.ID, Side: "without_love", Body: "wol", Evidence: []dto.EvidenceInput{{AudioID: "c"}}})

	withReply := createTheory(t, repos, author.ID, "with reply")
	parent := theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: withReply, UserID: responder.ID, Side: "with_love", Body: "p", Evidence: []dto.EvidenceInput{{AudioID: "a"}}})
	theoryCreateResponse(t, repos, spec.NewTheoryResponse{TheoryID: withReply, UserID: responder.ID, ParentID: &parent, Side: "with_love", Body: "child", Evidence: []dto.EvidenceInput{{AudioID: "b"}, {AudioID: "c"}}})

	unanswered := createTheory(t, repos, author.ID, "unanswered")

	weighted := createTheory(t, repos, author.ID, "weighted")
	weightedResponse, err := repos.Theory.CreateResponse(ctx, spec.NewTheoryResponse{TheoryID: weighted, UserID: responder.ID, Side: "with_love", Body: "x", Evidence: []dto.EvidenceInput{{AudioID: "a"}}})
	require.NoError(t, err)
	require.Len(t, weightedResponse.Evidence, 1)
	require.NoError(t, repos.Theory.SetEvidenceTruthWeight(ctx, spec.EvidenceTruthWeightUpdate{EvidenceID: weightedResponse.Evidence[0].ID, Weight: 3.5}))

	cases := []struct {
		name            string
		theoryID        uuid.UUID
		wantWithLove    float64
		wantWithoutLove float64
	}{
		{name: "top-level evidence is summed per side at the default weight of one", theoryID: bothSides, wantWithLove: 2, wantWithoutLove: 1},
		{name: "evidence on replies is left out", theoryID: withReply, wantWithLove: 1, wantWithoutLove: 0},
		{name: "a theory with no responses weighs nothing on either side", theoryID: unanswered, wantWithLove: 0, wantWithoutLove: 0},
		{name: "a stored truth weight replaces the default", theoryID: weighted, wantWithLove: 3.5, wantWithoutLove: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			withLove, withoutLove, err := repos.Theory.GetResponseEvidenceWeights(ctx, tc.theoryID)

			// then
			require.NoError(t, err)
			assert.InDelta(t, tc.wantWithLove, withLove, 0.001)
			assert.InDelta(t, tc.wantWithoutLove, withoutLove, 0.001)
		})
	}
}
