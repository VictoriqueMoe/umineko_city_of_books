package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"
	"umineko_city_of_books/internal/theory/params"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTheoryRequest(title string, evidence ...dto.EvidenceInput) dto.CreateTheoryRequest {
	return dto.CreateTheoryRequest{
		Title:    title,
		Body:     "body of " + title,
		Episode:  1,
		Series:   "umineko",
		Evidence: evidence,
	}
}

func createTheory(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title string) uuid.UUID {
	t.Helper()
	id, err := repos.Theory.Create(context.Background(), userID, newTheoryRequest(title))
	require.NoError(t, err)
	return id
}

func defaultListParams() params.ListParams {
	return params.NewListParams("new", 0, uuid.Nil, "", "umineko", 20, 0)
}

func TestTheoryRepository_Create(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	req := newTheoryRequest("My Theory",
		dto.EvidenceInput{AudioID: "a1", Note: "first"},
		dto.EvidenceInput{AudioID: "a2", Note: "second", Lang: "ja"},
	)

	// when
	id, err := repos.Theory.Create(context.Background(), user.ID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestTheoryRepository_Create_DefaultsSeriesToUmineko(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	req := dto.CreateTheoryRequest{Title: "T", Body: "B", Episode: 1}

	// when
	id, err := repos.Theory.Create(context.Background(), user.ID, req)

	// then
	require.NoError(t, err)
	series, err := repos.Theory.GetTheorySeries(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "umineko", series)
}

func TestTheoryRepository_GetByID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Author"))
	id, err := repos.Theory.Create(context.Background(), user.ID, newTheoryRequest("Title"))
	require.NoError(t, err)

	// when
	got, err := repos.Theory.GetByID(context.Background(), id)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "Title", got.Title)
	assert.Equal(t, "umineko", got.Series)
	assert.Equal(t, user.ID, got.Author.ID)
	assert.Equal(t, "Author", got.Author.DisplayName)
	assert.InDelta(t, 50.0, got.CredibilityScore, 0.001)
}

func TestTheoryRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.Theory.GetByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestTheoryRepository_GetByID_VoteAndSideCounts(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id := createTheory(t, repos, author.ID, "T")
	require.NoError(t, repos.Theory.VoteTheory(ctx, voter.ID, id, 1))
	_, err := repos.Theory.CreateResponse(ctx, id, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "yes"})
	require.NoError(t, err)
	_, err = repos.Theory.CreateResponse(ctx, id, responder.ID, dto.CreateResponseRequest{Side: "without_love", Body: "no"})
	require.NoError(t, err)

	// when
	got, err := repos.Theory.GetByID(ctx, id)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 1, got.VoteScore)
	assert.Equal(t, 1, got.WithLoveCount)
	assert.Equal(t, 1, got.WithoutLoveCount)
}

func TestTheoryRepository_List_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	rows, total, err := repos.Theory.List(context.Background(), defaultListParams(), uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}

func TestTheoryRepository_List_FiltersBySeries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	_, err := repos.Theory.Create(ctx, user.ID, dto.CreateTheoryRequest{Title: "u", Body: "b", Series: "umineko"})
	require.NoError(t, err)
	_, err = repos.Theory.Create(ctx, user.ID, dto.CreateTheoryRequest{Title: "h", Body: "b", Series: "higurashi"})
	require.NoError(t, err)
	p := params.NewListParams("new", 0, uuid.Nil, "", "higurashi", 20, 0)

	// when
	rows, total, err := repos.Theory.List(ctx, p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "higurashi", rows[0].Series)
}

func TestTheoryRepository_List_FiltersByEpisode(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	_, err := repos.Theory.Create(ctx, user.ID, dto.CreateTheoryRequest{Title: "e1", Body: "b", Episode: 1, Series: "umineko"})
	require.NoError(t, err)
	_, err = repos.Theory.Create(ctx, user.ID, dto.CreateTheoryRequest{Title: "e2", Body: "b", Episode: 2, Series: "umineko"})
	require.NoError(t, err)
	p := params.NewListParams("new", 2, uuid.Nil, "", "umineko", 20, 0)

	// when
	rows, total, err := repos.Theory.List(ctx, p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, 2, rows[0].Episode)
}

func TestTheoryRepository_List_FiltersByAuthor(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	ctx := context.Background()
	createTheory(t, repos, a.ID, "from a")
	createTheory(t, repos, b.ID, "from b")
	p := params.NewListParams("new", 0, a.ID, "", "umineko", 20, 0)

	// when
	rows, total, err := repos.Theory.List(ctx, p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, a.ID, rows[0].Author.ID)
}

func TestTheoryRepository_List_FiltersBySearch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	createTheory(t, repos, user.ID, "Beatrice the Golden")
	createTheory(t, repos, user.ID, "Battler theory")
	p := params.NewListParams("new", 0, uuid.Nil, "Golden", "umineko", 20, 0)

	// when
	rows, total, err := repos.Theory.List(ctx, p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Contains(t, rows[0].Title, "Golden")
}

func TestTheoryRepository_List_ExcludesUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	ctx := context.Background()
	createTheory(t, repos, a.ID, "a1")
	createTheory(t, repos, b.ID, "b1")

	// when
	rows, total, err := repos.Theory.List(ctx, defaultListParams(), uuid.Nil, []uuid.UUID{b.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, a.ID, rows[0].Author.ID)
}

func TestTheoryRepository_List_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		createTheory(t, repos, user.ID, "t")
	}
	p1 := params.NewListParams("new", 0, uuid.Nil, "", "umineko", 2, 0)
	p2 := params.NewListParams("new", 0, uuid.Nil, "", "umineko", 2, 2)
	p3 := params.NewListParams("new", 0, uuid.Nil, "", "umineko", 2, 4)

	// when
	page1, total, err := repos.Theory.List(ctx, p1, uuid.Nil, nil)
	require.NoError(t, err)
	page2, _, err := repos.Theory.List(ctx, p2, uuid.Nil, nil)
	require.NoError(t, err)
	page3, _, err := repos.Theory.List(ctx, p3, uuid.Nil, nil)
	require.NoError(t, err)

	// then
	assert.Equal(t, 5, total)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
	assert.Len(t, page3, 1)
}

func TestTheoryRepository_List_OrderByCredibility(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	low := createTheory(t, repos, user.ID, "low")
	high := createTheory(t, repos, user.ID, "high")
	require.NoError(t, repos.Theory.UpdateCredibilityScore(ctx, low, 10.0))
	require.NoError(t, repos.Theory.UpdateCredibilityScore(ctx, high, 90.0))
	p := params.NewListParams("credibility", 0, uuid.Nil, "", "umineko", 20, 0)

	// when
	rows, _, err := repos.Theory.List(ctx, p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, high, rows[0].ID)
	assert.Equal(t, low, rows[1].ID)
}

func TestTheoryRepository_List_OrderByCredibilityAsc(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	low := createTheory(t, repos, user.ID, "low")
	high := createTheory(t, repos, user.ID, "high")
	require.NoError(t, repos.Theory.UpdateCredibilityScore(ctx, low, 10.0))
	require.NoError(t, repos.Theory.UpdateCredibilityScore(ctx, high, 90.0))
	p := params.NewListParams("credibility_asc", 0, uuid.Nil, "", "umineko", 20, 0)

	// when
	rows, _, err := repos.Theory.List(ctx, p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, low, rows[0].ID)
	assert.Equal(t, high, rows[1].ID)
}

func TestTheoryRepository_List_OrderByPopular(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	quiet := createTheory(t, repos, author.ID, "quiet")
	loud := createTheory(t, repos, author.ID, "loud")
	require.NoError(t, repos.Theory.VoteTheory(ctx, voter.ID, loud, 1))
	p := params.NewListParams("popular", 0, uuid.Nil, "", "umineko", 20, 0)

	// when
	rows, _, err := repos.Theory.List(ctx, p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, loud, rows[0].ID)
	assert.Equal(t, quiet, rows[1].ID)
}

func TestTheoryRepository_List_OrderByControversial(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	v1 := repotest.CreateUser(t, repos)
	v2 := repotest.CreateUser(t, repos)
	ctx := context.Background()
	calm := createTheory(t, repos, author.ID, "calm")
	hot := createTheory(t, repos, author.ID, "hot")
	require.NoError(t, repos.Theory.VoteTheory(ctx, v1.ID, hot, 1))
	require.NoError(t, repos.Theory.VoteTheory(ctx, v2.ID, hot, -1))
	p := params.NewListParams("controversial", 0, uuid.Nil, "", "umineko", 20, 0)

	// when
	rows, _, err := repos.Theory.List(ctx, p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, hot, rows[0].ID)
	assert.Equal(t, calm, rows[1].ID)
}

func TestTheoryRepository_List_OrderByOld(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	first := createTheory(t, repos, user.ID, "first")
	createTheory(t, repos, user.ID, "second")
	p := params.NewListParams("old", 0, uuid.Nil, "", "umineko", 20, 0)

	// when
	rows, _, err := repos.Theory.List(ctx, p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, first, rows[0].ID)
}

func TestTheoryRepository_List_TruncatesLongBody(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	body := ""
	for i := 0; i < 250; i++ {
		body += "x"
	}
	_, err := repos.Theory.Create(ctx, user.ID, dto.CreateTheoryRequest{Title: "long", Body: body, Series: "umineko"})
	require.NoError(t, err)

	// when
	rows, _, err := repos.Theory.List(ctx, defaultListParams(), uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 203, len(rows[0].Body))
	assert.Contains(t, rows[0].Body, "...")
}

func TestTheoryRepository_List_IncludesUserVote(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id := createTheory(t, repos, author.ID, "t")
	require.NoError(t, repos.Theory.VoteTheory(ctx, voter.ID, id, -1))

	// when
	rows, _, err := repos.Theory.List(ctx, defaultListParams(), voter.ID, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, -1, rows[0].UserVote)
}

func TestTheoryRepository_Update(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id := createTheory(t, repos, user.ID, "old")
	req := dto.CreateTheoryRequest{Title: "new", Body: "newbody", Episode: 5, Series: "umineko",
		Evidence: []dto.EvidenceInput{{AudioID: "x", Note: "n"}}}

	// when
	err := repos.Theory.Update(ctx, id, user.ID, req)

	// then
	require.NoError(t, err)
	got, err := repos.Theory.GetByID(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "new", got.Title)
	assert.Equal(t, "newbody", got.Body)
	assert.Equal(t, 5, got.Episode)
	ev, err := repos.Theory.GetEvidence(ctx, id)
	require.NoError(t, err)
	require.Len(t, ev, 1)
	assert.Equal(t, "x", ev[0].AudioID)
}

func TestTheoryRepository_Update_OnlyOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id := createTheory(t, repos, owner.ID, "x")

	// when
	err := repos.Theory.Update(ctx, id, other.ID, newTheoryRequest("hijack"))

	// then
	require.Error(t, err)
}

func TestTheoryRepository_UpdateAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id := createTheory(t, repos, owner.ID, "x")

	// when
	err := repos.Theory.UpdateAsAdmin(ctx, id, dto.CreateTheoryRequest{Title: "modded", Body: "modbody", Episode: 3, Series: "umineko"})

	// then
	require.NoError(t, err)
	got, err := repos.Theory.GetByID(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "modded", got.Title)
}

func TestTheoryRepository_UpdateAsAdmin_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Theory.UpdateAsAdmin(context.Background(), uuid.New(), newTheoryRequest("x"))

	// then
	require.Error(t, err)
}

func TestTheoryRepository_Delete(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id := createTheory(t, repos, user.ID, "x")

	// when
	err := repos.Theory.Delete(ctx, id, user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.Theory.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestTheoryRepository_Delete_OnlyOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id := createTheory(t, repos, owner.ID, "x")

	// when
	err := repos.Theory.Delete(ctx, id, other.ID)

	// then
	require.Error(t, err)
}

func TestTheoryRepository_DeleteAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id := createTheory(t, repos, user.ID, "x")

	// when
	err := repos.Theory.DeleteAsAdmin(ctx, id)

	// then
	require.NoError(t, err)
}

func TestTheoryRepository_DeleteAsAdmin_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Theory.DeleteAsAdmin(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestTheoryRepository_GetEvidence(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	req := newTheoryRequest("t",
		dto.EvidenceInput{AudioID: "a1", Note: "first", Lang: "en"},
		dto.EvidenceInput{AudioID: "a2", Note: "second", Lang: "ja"},
	)
	id, err := repos.Theory.Create(ctx, user.ID, req)
	require.NoError(t, err)

	// when
	ev, err := repos.Theory.GetEvidence(ctx, id)

	// then
	require.NoError(t, err)
	require.Len(t, ev, 2)
	assert.Equal(t, "a1", ev[0].AudioID)
	assert.Equal(t, 0, ev[0].SortOrder)
	assert.Equal(t, "en", ev[0].Lang)
	assert.Equal(t, "a2", ev[1].AudioID)
	assert.Equal(t, "ja", ev[1].Lang)
}

func TestTheoryRepository_GetEvidence_DefaultsLang(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	req := newTheoryRequest("t", dto.EvidenceInput{AudioID: "a1", Note: "x"})
	id, err := repos.Theory.Create(ctx, user.ID, req)
	require.NoError(t, err)

	// when
	ev, err := repos.Theory.GetEvidence(ctx, id)

	// then
	require.NoError(t, err)
	require.Len(t, ev, 1)
	assert.Equal(t, "en", ev[0].Lang)
}

func TestTheoryRepository_CreateResponse(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	req := dto.CreateResponseRequest{Side: "with_love", Body: "yes",
		Evidence: []dto.EvidenceInput{{AudioID: "x", Note: "n"}}}

	// when
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rid)
	ev, err := repos.Theory.GetResponseEvidence(ctx, rid)
	require.NoError(t, err)
	require.Len(t, ev, 1)
	assert.Equal(t, "x", ev[0].AudioID)
}

func TestTheoryRepository_CreateResponse_WithParent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	parentID, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "p"})
	require.NoError(t, err)

	// when
	childID, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{ParentID: &parentID, Side: "with_love", Body: "c"})

	// then
	require.NoError(t, err)
	authorID, theoryID, err := repos.Theory.GetResponseInfo(ctx, childID)
	require.NoError(t, err)
	assert.Equal(t, responder.ID, authorID)
	assert.Equal(t, tid, theoryID)
}

func TestTheoryRepository_DeleteResponse(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x"})
	require.NoError(t, err)

	// when
	err = repos.Theory.DeleteResponse(ctx, rid, responder.ID)

	// then
	require.NoError(t, err)
	resps, err := repos.Theory.GetResponses(ctx, tid, uuid.Nil)
	require.NoError(t, err)
	assert.Empty(t, resps)
}

func TestTheoryRepository_DeleteResponse_OnlyOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x"})
	require.NoError(t, err)

	// when
	err = repos.Theory.DeleteResponse(ctx, rid, other.ID)

	// then
	require.Error(t, err)
}

func TestTheoryRepository_DeleteResponseAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x"})
	require.NoError(t, err)

	// when
	err = repos.Theory.DeleteResponseAsAdmin(ctx, rid)

	// then
	require.NoError(t, err)
}

func TestTheoryRepository_DeleteResponseAsAdmin_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Theory.DeleteResponseAsAdmin(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestTheoryRepository_GetResponses_BuildsTree(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	parent, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "parent"})
	require.NoError(t, err)
	_, err = repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{ParentID: &parent, Side: "with_love", Body: "child"})
	require.NoError(t, err)

	// when
	rows, err := repos.Theory.GetResponses(ctx, tid, uuid.Nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, parent, rows[0].ID)
	require.Len(t, rows[0].Replies, 1)
	assert.Equal(t, "child", rows[0].Replies[0].Body)
}

func TestTheoryRepository_GetResponses_IncludesEvidenceAndUserVote(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x",
		Evidence: []dto.EvidenceInput{{AudioID: "ev", Note: "n"}}})
	require.NoError(t, err)
	require.NoError(t, repos.Theory.VoteResponse(ctx, voter.ID, rid, 1))

	// when
	rows, err := repos.Theory.GetResponses(ctx, tid, voter.ID)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].VoteScore)
	assert.Equal(t, 1, rows[0].UserVote)
	require.Len(t, rows[0].Evidence, 1)
	assert.Equal(t, "ev", rows[0].Evidence[0].AudioID)
}

func TestTheoryRepository_GetResponseEvidence(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x",
		Evidence: []dto.EvidenceInput{
			{AudioID: "first", Note: "1"},
			{AudioID: "second", Note: "2", Lang: "ja"},
		}})
	require.NoError(t, err)

	// when
	ev, err := repos.Theory.GetResponseEvidence(ctx, rid)

	// then
	require.NoError(t, err)
	require.Len(t, ev, 2)
	assert.Equal(t, "first", ev[0].AudioID)
	assert.Equal(t, "en", ev[0].Lang)
	assert.Equal(t, "second", ev[1].AudioID)
	assert.Equal(t, "ja", ev[1].Lang)
}

func TestTheoryRepository_VoteTheory_Insert(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")

	// when
	err := repos.Theory.VoteTheory(ctx, voter.ID, tid, 1)

	// then
	require.NoError(t, err)
	v, err := repos.Theory.GetUserTheoryVote(ctx, voter.ID, tid)
	require.NoError(t, err)
	assert.Equal(t, 1, v)
}

func TestTheoryRepository_VoteTheory_Update(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	require.NoError(t, repos.Theory.VoteTheory(ctx, voter.ID, tid, 1))

	// when
	err := repos.Theory.VoteTheory(ctx, voter.ID, tid, -1)

	// then
	require.NoError(t, err)
	v, err := repos.Theory.GetUserTheoryVote(ctx, voter.ID, tid)
	require.NoError(t, err)
	assert.Equal(t, -1, v)
}

func TestTheoryRepository_VoteTheory_Zero_Deletes(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	require.NoError(t, repos.Theory.VoteTheory(ctx, voter.ID, tid, 1))

	// when
	err := repos.Theory.VoteTheory(ctx, voter.ID, tid, 0)

	// then
	require.NoError(t, err)
	v, err := repos.Theory.GetUserTheoryVote(ctx, voter.ID, tid)
	require.NoError(t, err)
	assert.Equal(t, 0, v)
}

func TestTheoryRepository_VoteResponse_Insert(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x"})
	require.NoError(t, err)

	// when
	err = repos.Theory.VoteResponse(ctx, voter.ID, rid, 1)

	// then
	require.NoError(t, err)
	rows, err := repos.Theory.GetResponses(ctx, tid, voter.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].UserVote)
}

func TestTheoryRepository_VoteResponse_Update(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x"})
	require.NoError(t, err)
	require.NoError(t, repos.Theory.VoteResponse(ctx, voter.ID, rid, 1))

	// when
	err = repos.Theory.VoteResponse(ctx, voter.ID, rid, -1)

	// then
	require.NoError(t, err)
	rows, err := repos.Theory.GetResponses(ctx, tid, voter.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, -1, rows[0].UserVote)
}

func TestTheoryRepository_VoteResponse_Zero_Deletes(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x"})
	require.NoError(t, err)
	require.NoError(t, repos.Theory.VoteResponse(ctx, voter.ID, rid, 1))

	// when
	err = repos.Theory.VoteResponse(ctx, voter.ID, rid, 0)

	// then
	require.NoError(t, err)
	rows, err := repos.Theory.GetResponses(ctx, tid, voter.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 0, rows[0].UserVote)
}

func TestTheoryRepository_GetUserTheoryVote_None(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")

	// when
	v, err := repos.Theory.GetUserTheoryVote(ctx, voter.ID, tid)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, v)
}

func TestTheoryRepository_GetTheoryAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, user.ID, "t")

	// when
	got, err := repos.Theory.GetTheoryAuthorID(ctx, tid)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestTheoryRepository_GetTheoryAuthorID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Theory.GetTheoryAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestTheoryRepository_GetResponseInfo(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x"})
	require.NoError(t, err)

	// when
	gotAuthor, gotTheory, err := repos.Theory.GetResponseInfo(ctx, rid)

	// then
	require.NoError(t, err)
	assert.Equal(t, responder.ID, gotAuthor)
	assert.Equal(t, tid, gotTheory)
}

func TestTheoryRepository_GetResponseInfo_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, _, err := repos.Theory.GetResponseInfo(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestTheoryRepository_GetTheoryTitle(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, user.ID, "MyTitle")

	// when
	title, err := repos.Theory.GetTheoryTitle(ctx, tid)

	// then
	require.NoError(t, err)
	assert.Equal(t, "MyTitle", title)
}

func TestTheoryRepository_GetTheoryTitle_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Theory.GetTheoryTitle(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestTheoryRepository_GetTheorySeries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id, err := repos.Theory.Create(ctx, user.ID, dto.CreateTheoryRequest{Title: "t", Body: "b", Series: "higurashi"})
	require.NoError(t, err)

	// when
	series, err := repos.Theory.GetTheorySeries(ctx, id)

	// then
	require.NoError(t, err)
	assert.Equal(t, "higurashi", series)
}

func TestTheoryRepository_GetTheorySeries_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Theory.GetTheorySeries(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestTheoryRepository_GetRecentActivityByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, user.ID, "MyTheory")
	_, err := repos.Theory.CreateResponse(ctx, tid, user.ID, dto.CreateResponseRequest{Side: "with_love", Body: "resp"})
	require.NoError(t, err)

	// when
	items, total, err := repos.Theory.GetRecentActivityByUser(ctx, user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	types := []string{items[0].Type, items[1].Type}
	assert.Contains(t, types, "theory")
	assert.Contains(t, types, "response")
}

func TestTheoryRepository_GetRecentActivityByUser_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	items, total, err := repos.Theory.GetRecentActivityByUser(context.Background(), user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, items)
}

func TestTheoryRepository_GetRecentActivityByUser_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		createTheory(t, repos, user.ID, "t")
	}

	// when
	page1, total, err := repos.Theory.GetRecentActivityByUser(ctx, user.ID, 2, 0)
	require.NoError(t, err)
	page2, _, err := repos.Theory.GetRecentActivityByUser(ctx, user.ID, 2, 2)
	require.NoError(t, err)

	// then
	assert.Equal(t, 4, total)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
}

func TestTheoryRepository_CountUserTheoriesToday(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	ctx := context.Background()
	createTheory(t, repos, user.ID, "a")
	createTheory(t, repos, user.ID, "b")
	createTheory(t, repos, other.ID, "c")

	// when
	count, err := repos.Theory.CountUserTheoriesToday(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestTheoryRepository_CountUserResponsesToday(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	_, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x"})
	require.NoError(t, err)
	_, err = repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "without_love", Body: "y"})
	require.NoError(t, err)

	// when
	count, err := repos.Theory.CountUserResponsesToday(ctx, responder.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestTheoryRepository_UpdateCredibilityScore(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, user.ID, "t")

	// when
	err := repos.Theory.UpdateCredibilityScore(ctx, tid, 77.5)

	// then
	require.NoError(t, err)
	got, err := repos.Theory.GetByID(ctx, tid)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.InDelta(t, 77.5, got.CredibilityScore, 0.001)
}

func TestTheoryRepository_GetResponseEvidenceWeights(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	_, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "wl",
		Evidence: []dto.EvidenceInput{{AudioID: "a", Note: "n"}, {AudioID: "b", Note: "n"}}})
	require.NoError(t, err)
	_, err = repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "without_love", Body: "wol",
		Evidence: []dto.EvidenceInput{{AudioID: "c", Note: "n"}}})
	require.NoError(t, err)

	// when
	wl, wol, err := repos.Theory.GetResponseEvidenceWeights(ctx, tid)

	// then
	require.NoError(t, err)
	assert.InDelta(t, 2.0, wl, 0.001)
	assert.InDelta(t, 1.0, wol, 0.001)
}

func TestTheoryRepository_GetResponseEvidenceWeights_ExcludesReplies(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	parent, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "p",
		Evidence: []dto.EvidenceInput{{AudioID: "a", Note: "n"}}})
	require.NoError(t, err)
	_, err = repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{ParentID: &parent, Side: "with_love", Body: "child",
		Evidence: []dto.EvidenceInput{{AudioID: "b", Note: "n"}, {AudioID: "c", Note: "n"}}})
	require.NoError(t, err)

	// when
	wl, wol, err := repos.Theory.GetResponseEvidenceWeights(ctx, tid)

	// then
	require.NoError(t, err)
	assert.InDelta(t, 1.0, wl, 0.001)
	assert.InDelta(t, 0.0, wol, 0.001)
}

func TestTheoryRepository_GetResponseEvidenceWeights_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, user.ID, "t")

	// when
	wl, wol, err := repos.Theory.GetResponseEvidenceWeights(ctx, tid)

	// then
	require.NoError(t, err)
	assert.InDelta(t, 0.0, wl, 0.001)
	assert.InDelta(t, 0.0, wol, 0.001)
}

func TestTheoryRepository_SetEvidenceTruthWeight(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	responder := repotest.CreateUser(t, repos)
	ctx := context.Background()
	tid := createTheory(t, repos, author.ID, "t")
	rid, err := repos.Theory.CreateResponse(ctx, tid, responder.ID, dto.CreateResponseRequest{Side: "with_love", Body: "x",
		Evidence: []dto.EvidenceInput{{AudioID: "a", Note: "n"}}})
	require.NoError(t, err)
	ev, err := repos.Theory.GetResponseEvidence(ctx, rid)
	require.NoError(t, err)
	require.Len(t, ev, 1)

	// when
	err = repos.Theory.SetEvidenceTruthWeight(ctx, ev[0].ID, 3.5)

	// then
	require.NoError(t, err)
	wl, _, err := repos.Theory.GetResponseEvidenceWeights(ctx, tid)
	require.NoError(t, err)
	assert.InDelta(t, 3.5, wl, 0.001)
}
