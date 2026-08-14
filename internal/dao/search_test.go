package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/repository"

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

func TestSearchDAO_Theory_TitleMatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	created, err := repos.Theory.Create(context.Background(), repository.NewTheory{UserID: user.ID,
		Title:  "The Witch of Endless Magic",
		Body:   "Beatrice presides over the rokkenjima incident.",
		Series: "umineko",
	})
	require.NoError(t, err)
	id := created.ID

	// when
	results := searchRepoOnce(t, repos, "witch", nil)

	// then
	require.NotEmpty(t, results)
	assert.Equal(t, id.String(), results[0].ID)
	assert.Equal(t, repository.SearchEntityTheory, results[0].EntityType)
}

func TestSearchDAO_Body_HighlightsMatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	_, err := repos.Theory.Create(context.Background(), repository.NewTheory{UserID: user.ID,
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

func TestSearchDAO_TitleTrigram_HandlesTypo(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	created, err := repos.Theory.Create(context.Background(), repository.NewTheory{UserID: user.ID,
		Title: "Beatrice", Body: "The endless witch.", Series: "umineko",
	})
	require.NoError(t, err)
	id := created.ID

	// when
	results := searchRepoOnce(t, repos, "beatice", nil)

	// then
	require.NotEmpty(t, results)
	assert.Equal(t, id.String(), results[0].ID)
}

func TestSearchDAO_BannedUser_ContentHidden(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	bannedUser := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	_, err := repos.Theory.Create(context.Background(), repository.NewTheory{UserID: bannedUser.ID,
		Title: "Hidden treasure of rokkenjima", Body: "...", Series: "umineko",
	})
	require.NoError(t, err)
	require.NoError(t, repos.User.BanUser(context.Background(), bannedUser.ID, admin.ID, "spam"))

	// when
	results := searchRepoOnce(t, repos, "rokkenjima", nil)

	// then
	assert.Empty(t, results)
}

func TestSearchDAO_FanficDraft_Hidden(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	draft, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "Hidden Draft About Beatrice",
		Summary:  "Secret summary about beatrice",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "draft",
	})
	require.NoError(t, err)
	draftID := draft.ID
	published, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "Public Beatrice Story",
		Summary:  "Public summary",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
	})
	require.NoError(t, err)
	publishedID := published.ID

	// when
	results := searchRepoOnce(t, repos, "beatrice", nil)

	// then
	ids := resultIDs(results)
	assert.NotContains(t, ids, draftID.String())
	assert.Contains(t, ids, publishedID.String())
}

func TestSearchDAO_TypeFilter_ReturnsOnlyRequestedType(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	_, err := repos.Theory.Create(context.Background(), repository.NewTheory{UserID: user.ID,
		Title: "Maria's lullaby explanation", Body: "x", Series: "umineko",
	})
	require.NoError(t, err)
	_, err = repos.Post.Create(context.Background(), repository.NewPost{UserID: user.ID, Corner: "umineko", Body: "Maria's lullaby was the key"})
	require.NoError(t, err)

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

func TestSearchDAO_PostComment_HasParentID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	post, err := repos.Post.Create(context.Background(), repository.NewPost{UserID: user.ID, Corner: "umineko", Body: "the parent post body"})
	require.NoError(t, err)
	postID := post.ID
	comment, err := repos.Post.CreateComment(context.Background(), postID, nil, user.ID, "I think this kinzo theory is right")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	results := searchRepoOnce(t, repos, "kinzo", []repository.SearchEntityType{repository.SearchEntityPostComment})

	// then
	require.NotEmpty(t, results)
	assert.Equal(t, commentID.String(), results[0].ID)
	require.NotNil(t, results[0].ParentID)
	assert.Equal(t, postID.String(), *results[0].ParentID)
}

func TestSearchDAO_User_TrigramOnUsername(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	daotest.CreateUser(t, repos, daotest.WithUsername("battler1986"), daotest.WithDisplayName("Random Display"))

	// when
	results := searchRepoOnce(t, repos, "battler", []repository.SearchEntityType{repository.SearchEntityUser})

	// then
	require.NotEmpty(t, results)
	assert.Equal(t, "battler1986", results[0].AuthorUsername)
}

func TestSearchDAO_QuickSearch_CapsPerType(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 5 {
		_, err := repos.Theory.Create(context.Background(), repository.NewTheory{UserID: user.ID,
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

func TestSearchDAO_Pagination_RespectsLimitAndOffset(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	created := make([]uuid.UUID, 0, 5)
	for range 5 {
		createdRow, err := repos.Theory.Create(context.Background(), repository.NewTheory{UserID: user.ID,
			Title: "paginated theory", Body: "paginated body", Series: "umineko",
		})
		require.NoError(t, err)
		id := createdRow.ID
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

func TestSearchDAO_AllRegisteredEntitiesRoundTrip(t *testing.T) {
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

func TestSearchDAO_SearchSources_RegistryIntegrity(t *testing.T) {
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
