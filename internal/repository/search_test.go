package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func searchRepoOnce(t *testing.T, repos *repository.Repositories, query string, types []repository.SearchEntityType) []repository.SearchResult {
	t.Helper()
	results, _, err := repos.Search.Search(context.Background(), query, types, 20, 0)
	require.NoError(t, err)
	return results
}

func resultIDs(results []repository.SearchResult) []string {
	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.ID
	}
	return out
}

func TestSearchRepository_Theory_TitleMatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id, err := repos.Theory.Create(context.Background(), user.ID, dto.CreateTheoryRequest{
		Title:  "The Witch of Endless Magic",
		Body:   "Beatrice presides over the rokkenjima incident.",
		Series: "umineko",
	})
	require.NoError(t, err)

	// when
	results := searchRepoOnce(t, repos, "witch", nil)

	// then
	require.NotEmpty(t, results)
	assert.Equal(t, id.String(), results[0].ID)
	assert.Equal(t, repository.SearchEntityTheory, results[0].EntityType)
}

func TestSearchRepository_Body_HighlightsMatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	_, err := repos.Theory.Create(context.Background(), user.ID, dto.CreateTheoryRequest{
		Title:  "Episode notes",
		Body:   "The golden truth uncovers Beatrice once and for all.",
		Series: "umineko",
	})
	require.NoError(t, err)

	// when
	results := searchRepoOnce(t, repos, "golden truth", nil)

	// then
	require.NotEmpty(t, results)
	assert.Contains(t, results[0].Snippet, "<mark>")
	assert.Contains(t, results[0].Snippet, "</mark>")
}

func TestSearchRepository_TitleTrigram_HandlesTypo(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id, err := repos.Theory.Create(context.Background(), user.ID, dto.CreateTheoryRequest{
		Title: "Beatrice", Body: "The endless witch.", Series: "umineko",
	})
	require.NoError(t, err)

	// when
	results := searchRepoOnce(t, repos, "beatice", nil)

	// then
	require.NotEmpty(t, results)
	assert.Equal(t, id.String(), results[0].ID)
}

func TestSearchRepository_BannedUser_ContentHidden(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	bannedUser := repotest.CreateUser(t, repos)
	admin := repotest.CreateUser(t, repos)
	_, err := repos.Theory.Create(context.Background(), bannedUser.ID, dto.CreateTheoryRequest{
		Title: "Hidden treasure of rokkenjima", Body: "...", Series: "umineko",
	})
	require.NoError(t, err)
	require.NoError(t, repos.User.BanUser(context.Background(), bannedUser.ID, admin.ID, "spam"))

	// when
	results := searchRepoOnce(t, repos, "rokkenjima", nil)

	// then
	assert.Empty(t, results)
}

func TestSearchRepository_FanficDraft_Hidden(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	draftID := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), draftID, user.ID,
		"Hidden Draft About Beatrice", "Secret summary about beatrice",
		"Umineko", "K", "English", "draft", false, false, nil, nil, nil, false,
	))
	publishedID := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), publishedID, user.ID,
		"Public Beatrice Story", "Public summary",
		"Umineko", "K", "English", "in_progress", false, false, nil, nil, nil, false,
	))

	// when
	results := searchRepoOnce(t, repos, "beatrice", nil)

	// then
	ids := resultIDs(results)
	assert.NotContains(t, ids, draftID.String())
	assert.Contains(t, ids, publishedID.String())
}

func TestSearchRepository_TypeFilter_ReturnsOnlyRequestedType(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	_, err := repos.Theory.Create(context.Background(), user.ID, dto.CreateTheoryRequest{
		Title: "Maria's lullaby explanation", Body: "x", Series: "umineko",
	})
	require.NoError(t, err)
	postID := uuid.New()
	require.NoError(t, repos.Post.Create(context.Background(), postID, user.ID, "umineko", "Maria's lullaby was the key", nil, nil))

	// when
	theoryOnly := searchRepoOnce(t, repos, "lullaby", []repository.SearchEntityType{repository.SearchEntityTheory})
	postOnly := searchRepoOnce(t, repos, "lullaby", []repository.SearchEntityType{repository.SearchEntityPost})

	// then
	for _, r := range theoryOnly {
		assert.Equal(t, repository.SearchEntityTheory, r.EntityType)
	}
	for _, r := range postOnly {
		assert.Equal(t, repository.SearchEntityPost, r.EntityType)
	}
	assert.NotEmpty(t, theoryOnly)
	assert.NotEmpty(t, postOnly)
}

func TestSearchRepository_PostComment_HasParentID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := uuid.New()
	require.NoError(t, repos.Post.Create(context.Background(), postID, user.ID, "umineko", "the parent post body", nil, nil))
	commentID := uuid.New()
	require.NoError(t, repos.Post.CreateComment(context.Background(), commentID, postID, nil, user.ID, "I think this kinzo theory is right"))

	// when
	results := searchRepoOnce(t, repos, "kinzo", []repository.SearchEntityType{repository.SearchEntityPostComment})

	// then
	require.NotEmpty(t, results)
	assert.Equal(t, commentID.String(), results[0].ID)
	require.NotNil(t, results[0].ParentID)
	assert.Equal(t, postID.String(), *results[0].ParentID)
}

func TestSearchRepository_User_TrigramOnUsername(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos, repotest.WithUsername("battler1986"), repotest.WithDisplayName("Random Display"))

	// when
	results := searchRepoOnce(t, repos, "battler", []repository.SearchEntityType{repository.SearchEntityUser})

	// then
	require.NotEmpty(t, results)
	assert.Equal(t, "battler1986", results[0].AuthorUsername)
}

func TestSearchRepository_QuickSearch_CapsPerType(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 5; i++ {
		_, err := repos.Theory.Create(context.Background(), user.ID, dto.CreateTheoryRequest{
			Title: "kinzo theory", Body: "kinzo body", Series: "umineko",
		})
		require.NoError(t, err)
	}

	// when
	results, err := repos.Search.QuickSearch(context.Background(), "kinzo", 2)
	require.NoError(t, err)

	// then
	theoryCount := 0
	for _, r := range results {
		if r.EntityType == repository.SearchEntityTheory {
			theoryCount++
		}
	}
	assert.Equal(t, 2, theoryCount, "QuickSearch should cap each type to perTypeLimit")
}

func TestSearchRepository_Pagination_RespectsLimitAndOffset(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	created := make([]uuid.UUID, 0, 5)
	for i := 0; i < 5; i++ {
		id, err := repos.Theory.Create(context.Background(), user.ID, dto.CreateTheoryRequest{
			Title: "paginated theory", Body: "paginated body", Series: "umineko",
		})
		require.NoError(t, err)
		created = append(created, id)
	}

	// when
	page1, total1, err := repos.Search.Search(context.Background(), "paginated",
		[]repository.SearchEntityType{repository.SearchEntityTheory}, 2, 0)
	require.NoError(t, err)
	page2, total2, err := repos.Search.Search(context.Background(), "paginated",
		[]repository.SearchEntityType{repository.SearchEntityTheory}, 2, 2)
	require.NoError(t, err)

	// then
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
	assert.Equal(t, total1, total2)
	assert.GreaterOrEqual(t, total1, len(created))
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestSearchRepository_AllRegisteredEntitiesRoundTrip(t *testing.T) {
	// given
	registered := []repository.SearchEntityType{
		repository.SearchEntityTheory, repository.SearchEntityResponse,
		repository.SearchEntityPost, repository.SearchEntityPostComment,
		repository.SearchEntityArt, repository.SearchEntityArtComment,
		repository.SearchEntityMystery, repository.SearchEntityMysteryAttempt, repository.SearchEntityMysteryComment,
		repository.SearchEntityShip, repository.SearchEntityShipComment,
		repository.SearchEntityAnnouncement, repository.SearchEntityAnnouncementComment,
		repository.SearchEntityFanfic, repository.SearchEntityFanficComment,
		repository.SearchEntityJournal, repository.SearchEntityJournalComment,
		repository.SearchEntityUser,
	}

	// when / then - just confirms each entity has a valid registry entry
	for _, typ := range registered {
		_, ok := repository.SearchSourceFor(typ)
		require.Truef(t, ok, "missing registry entry for %s", typ)
	}
}

func TestSearchRepository_SearchSources_RegistryIntegrity(t *testing.T) {
	// given / when
	srcs := repository.SearchSources()

	// then
	assert.NotEmpty(t, srcs)
	for _, s := range srcs {
		assert.NotEmptyf(t, s.From, "%s missing From", s.Type)
		assert.NotEmptyf(t, s.IDExpr, "%s missing IDExpr", s.Type)
		assert.NotEmptyf(t, s.SearchVector, "%s missing SearchVector", s.Type)
	}
}
