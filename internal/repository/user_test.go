package repository_test

import (
	"context"
	"strings"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleProfileRequest() dto.UpdateProfileRequest {
	return dto.UpdateProfileRequest{
		DisplayName:            "New Name",
		Bio:                    "A bio",
		AvatarURL:              "/avatar.png",
		BannerURL:              "/banner.png",
		BannerPosition:         0.5,
		FavouriteCharacter:     "beatrice",
		Gender:                 "female",
		PronounSubject:         "she",
		PronounPossessive:      "her",
		SocialTwitter:          "tw",
		SocialDiscord:          "dc",
		SocialWaifulist:        "wl",
		SocialTumblr:           "tb",
		SocialGithub:           "gh",
		Website:                "https://example.com",
		DmsEnabled:             true,
		EpisodeProgress:        4,
		HigurashiArcProgress:   7,
		CiconiaChapterProgress: 12,
		DOB:                    "2000-04-15",
		DOBPublic:              true,
		Email:                  "user@example.com",
		EmailPublic:            true,
		EmailNotifications:     true,
		HomePage:               "/home",
		GameBoardSort:          "newest",
	}
}

func insertSolvedMystery(t *testing.T, repos *repository.Repositories, gmID, winnerID uuid.UUID, difficulty string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := repos.DB().ExecContext(context.Background(),
		`INSERT INTO mysteries (id, user_id, title, body, difficulty, solved, winner_id) VALUES ($1, $2, $3, $4, $5, TRUE, $6)`,
		id, gmID, "title", "body", difficulty, winnerID,
	)
	require.NoError(t, err)
	return id
}

func insertMysteryAttempt(t *testing.T, repos *repository.Repositories, mysteryID, userID uuid.UUID) {
	t.Helper()
	_, err := repos.DB().ExecContext(context.Background(),
		`INSERT INTO mystery_attempts (id, mystery_id, user_id, body) VALUES ($1, $2, $3, $4)`,
		uuid.New(), mysteryID, userID, "guess",
	)
	require.NoError(t, err)
}

func TestUserRepository_Create(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	u, err := repos.User.Create(context.Background(), "alice", "secret123", "Alice")

	// then
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, "Alice", u.DisplayName)
	assert.NotEqual(t, uuid.Nil, u.ID)
}

func TestUserRepository_Create_DuplicateUsername(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	_, err := repos.User.Create(context.Background(), "dup", "pw1", "First")
	require.NoError(t, err)

	// when
	_, err = repos.User.Create(context.Background(), "dup", "pw2", "Second")

	// then
	require.Error(t, err)
}

func TestUserRepository_GetByID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	created := repotest.CreateUser(t, repos, repotest.WithUsername("byid"))

	// when
	got, err := repos.User.GetByID(context.Background(), created.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "byid", got.Username)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.User.GetByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserRepository_GetByUsername(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	created := repotest.CreateUser(t, repos, repotest.WithUsername("byname"))

	// when
	got, err := repos.User.GetByUsername(context.Background(), "byname")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
}

func TestUserRepository_GetByUsername_CaseInsensitive(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	created := repotest.CreateUser(t, repos, repotest.WithUsername("MixedCase"))

	// when
	got, err := repos.User.GetByUsername(context.Background(), "mixedcase")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
}

func TestUserRepository_GetByUsername_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.User.GetByUsername(context.Background(), "ghost")

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserRepository_ExistsByUsername(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos, repotest.WithUsername("exists"))

	// when
	exists, err := repos.User.ExistsByUsername(context.Background(), "exists")
	missing, err2 := repos.User.ExistsByUsername(context.Background(), "ghost")

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.True(t, exists)
	assert.False(t, missing)
}

func TestUserRepository_ExistsByUsername_CaseInsensitive(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos, repotest.WithUsername("CaseUser"))

	// when
	exists, err := repos.User.ExistsByUsername(context.Background(), "caseuser")

	// then
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUserRepository_Count(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos)
	repotest.CreateUser(t, repos)
	repotest.CreateUser(t, repos)

	// when
	count, err := repos.User.Count(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestUserRepository_Count_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	count, err := repos.User.Count(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestUserRepository_ValidatePassword_Success(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos, repotest.WithUsername("vpwd"), repotest.WithPassword("hunter2"))

	// when
	got, err := repos.User.ValidatePassword(context.Background(), "vpwd", "hunter2")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "vpwd", got.Username)
}

func TestUserRepository_ValidatePassword_WrongPassword(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos, repotest.WithUsername("vpwd"), repotest.WithPassword("hunter2"))

	// when
	got, err := repos.User.ValidatePassword(context.Background(), "vpwd", "wrong")

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserRepository_ValidatePassword_UnknownUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.User.ValidatePassword(context.Background(), "nobody", "pw")

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserRepository_UpdateProfile(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	req := sampleProfileRequest()

	// when
	err := repos.User.UpdateProfile(context.Background(), user.ID, req)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, req.DisplayName, got.DisplayName)
	assert.Equal(t, req.Bio, got.Bio)
	assert.Equal(t, req.AvatarURL, got.AvatarURL)
	assert.Equal(t, req.BannerURL, got.BannerURL)
	assert.Equal(t, req.BannerPosition, got.BannerPosition)
	assert.Equal(t, req.FavouriteCharacter, got.FavouriteCharacter)
	assert.Equal(t, req.Gender, got.Gender)
	assert.Equal(t, req.PronounSubject, got.PronounSubject)
	assert.Equal(t, req.PronounPossessive, got.PronounPossessive)
	assert.Equal(t, req.SocialTwitter, got.SocialTwitter)
	assert.Equal(t, req.SocialDiscord, got.SocialDiscord)
	assert.Equal(t, req.SocialWaifulist, got.SocialWaifulist)
	assert.Equal(t, req.SocialTumblr, got.SocialTumblr)
	assert.Equal(t, req.SocialGithub, got.SocialGithub)
	assert.Equal(t, req.Website, got.Website)
	assert.Equal(t, req.DmsEnabled, got.DmsEnabled)
	assert.Equal(t, req.EpisodeProgress, got.EpisodeProgress)
	assert.Equal(t, req.HigurashiArcProgress, got.HigurashiArcProgress)
	assert.Equal(t, req.CiconiaChapterProgress, got.CiconiaChapterProgress)
	assert.Equal(t, req.DOB, got.DOB)
	assert.Equal(t, req.DOBPublic, got.DOBPublic)
	assert.Equal(t, req.Email, got.Email)
	assert.Equal(t, req.EmailPublic, got.EmailPublic)
	assert.Equal(t, req.EmailNotifications, got.EmailNotifications)
	assert.Equal(t, req.HomePage, got.HomePage)
	assert.Equal(t, req.GameBoardSort, got.GameBoardSort)
}

func TestUserRepository_UpdateProfile_NonExistentUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.User.UpdateProfile(context.Background(), uuid.New(), sampleProfileRequest())

	// then
	require.NoError(t, err)
}

func TestUserRepository_UpdateAvatarURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateAvatarURL(context.Background(), user.ID, "/new-avatar.png")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "/new-avatar.png", got.AvatarURL)
}

func TestUserRepository_UpdateBannerURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateBannerURL(context.Background(), user.ID, "/new-banner.png")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "/new-banner.png", got.BannerURL)
}

func TestUserRepository_UpdateIP(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateIP(context.Background(), user.ID, "10.0.0.1")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, got.IP)
	assert.Equal(t, "10.0.0.1", *got.IP)
}

func TestUserRepository_UpdateGameBoardSort(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateGameBoardSort(context.Background(), user.ID, "popular")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "popular", got.GameBoardSort)
}

func TestUserRepository_UpdateAppearance(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateAppearance(context.Background(), user.ID, "dark", "serif", true)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "dark", got.Theme)
	assert.Equal(t, "serif", got.Font)
	assert.True(t, got.WideLayout)
}

func TestUserRepository_UpdateMysteryScoreAdjustment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateMysteryScoreAdjustment(context.Background(), user.ID, 50)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, got.MysteryScoreAdjustment)
}

func TestUserRepository_UpdateGMScoreAdjustment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateGMScoreAdjustment(context.Background(), user.ID, -25)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, -25, got.GMScoreAdjustment)
}

func TestUserRepository_GetDetectiveRawScore_NoMysteries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	score, err := repos.User.GetDetectiveRawScore(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, score)
}

func TestUserRepository_GetDetectiveRawScore_VariousDifficulties(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	winner := repotest.CreateUser(t, repos)
	insertSolvedMystery(t, repos, gm.ID, winner.ID, "easy")
	insertSolvedMystery(t, repos, gm.ID, winner.ID, "medium")
	insertSolvedMystery(t, repos, gm.ID, winner.ID, "hard")
	insertSolvedMystery(t, repos, gm.ID, winner.ID, "nightmare")

	// when
	score, err := repos.User.GetDetectiveRawScore(context.Background(), winner.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2+4+6+8, score)
}

func TestUserRepository_GetGMRawScore_NoMysteries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	score, err := repos.User.GetGMRawScore(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, score)
}

func TestUserRepository_GetGMRawScore_WithAttempts(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	winner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	mysteryID := insertSolvedMystery(t, repos, gm.ID, winner.ID, "medium")
	insertMysteryAttempt(t, repos, mysteryID, winner.ID)
	insertMysteryAttempt(t, repos, mysteryID, other.ID)

	// when
	score, err := repos.User.GetGMRawScore(context.Background(), gm.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 4+2, score)
}

func TestUserRepository_ChangePassword_Success(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithUsername("cp"), repotest.WithPassword("oldpw1"))

	// when
	err := repos.User.ChangePassword(context.Background(), user.ID, "oldpw1", "newpw2")

	// then
	require.NoError(t, err)
	good, err := repos.User.ValidatePassword(context.Background(), "cp", "newpw2")
	require.NoError(t, err)
	assert.NotNil(t, good)
	bad, err := repos.User.ValidatePassword(context.Background(), "cp", "oldpw1")
	require.NoError(t, err)
	assert.Nil(t, bad)
}

func TestUserRepository_ChangePassword_WrongOld(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithUsername("cp2"), repotest.WithPassword("rightpw"))

	// when
	err := repos.User.ChangePassword(context.Background(), user.ID, "wrongpw", "newpw")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incorrect password")
}

func TestUserRepository_ChangePassword_UserNotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.User.ChangePassword(context.Background(), uuid.New(), "x", "y")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUserRepository_DeleteAccount_Success(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithPassword("delme123"))

	// when
	err := repos.User.DeleteAccount(context.Background(), user.ID, "delme123")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserRepository_DeleteAccount_WrongPassword(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithPassword("rightpw"))

	// when
	err := repos.User.DeleteAccount(context.Background(), user.ID, "wrongpw")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incorrect password")
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestUserRepository_DeleteAccount_UserNotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.User.DeleteAccount(context.Background(), uuid.New(), "pw")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUserRepository_GetProfileByUsername(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithUsername("profuser"))

	// when
	u, stats, err := repos.User.GetProfileByUsername(context.Background(), "profuser")

	// then
	require.NoError(t, err)
	require.NotNil(t, u)
	require.NotNil(t, stats)
	assert.Equal(t, user.ID, u.ID)
	assert.Equal(t, 0, stats.TheoryCount)
	assert.Equal(t, 0, stats.ResponseCount)
	assert.Equal(t, 0, stats.VotesReceived)
	assert.Equal(t, 0, stats.ShipCount)
	assert.Equal(t, 0, stats.MysteryCount)
	assert.Equal(t, 0, stats.FanficCount)
}

func TestUserRepository_GetProfileByUsername_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	u, stats, err := repos.User.GetProfileByUsername(context.Background(), "ghost")

	// then
	require.NoError(t, err)
	assert.Nil(t, u)
	assert.Nil(t, stats)
}

func TestUserRepository_GetProfileByID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	gm := repotest.CreateUser(t, repos)
	insertSolvedMystery(t, repos, gm.ID, user.ID, "medium")

	// when
	u, stats, err := repos.User.GetProfileByID(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, u)
	require.NotNil(t, stats)
	assert.Equal(t, user.ID, u.ID)
	assert.Equal(t, 0, stats.MysteryCount)
}

func TestUserRepository_GetProfileByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	u, stats, err := repos.User.GetProfileByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, u)
	assert.Nil(t, stats)
}

func TestUserRepository_GetProfileByID_CountsMysteries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	insertSolvedMystery(t, repos, user.ID, user.ID, "easy")
	insertSolvedMystery(t, repos, user.ID, user.ID, "hard")

	// when
	_, stats, err := repos.User.GetProfileByID(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 2, stats.MysteryCount)
}

func TestUserRepository_ListAll_NoSearch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos, repotest.WithUsername("user1"))
	repotest.CreateUser(t, repos, repotest.WithUsername("user2"))
	repotest.CreateUser(t, repos, repotest.WithUsername("user3"))

	// when
	users, total, err := repos.User.ListAll(context.Background(), "", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, users, 3)
}

func TestUserRepository_ListAll_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	for i := 0; i < 5; i++ {
		repotest.CreateUser(t, repos)
	}

	// when
	page1, total1, err1 := repos.User.ListAll(context.Background(), "", 2, 0)
	page2, total2, err2 := repos.User.ListAll(context.Background(), "", 2, 2)
	page3, total3, err3 := repos.User.ListAll(context.Background(), "", 2, 4)

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)
	assert.Equal(t, 5, total1)
	assert.Equal(t, 5, total2)
	assert.Equal(t, 5, total3)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
	assert.Len(t, page3, 1)
}

func TestUserRepository_ListAll_Search(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos, repotest.WithUsername("alice_one"), repotest.WithDisplayName("Alice"))
	repotest.CreateUser(t, repos, repotest.WithUsername("bob_one"), repotest.WithDisplayName("Bob"))
	repotest.CreateUser(t, repos, repotest.WithUsername("charlie"), repotest.WithDisplayName("Alicia"))

	// when
	users, total, err := repos.User.ListAll(context.Background(), "alic", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, users, 2)
	for _, u := range users {
		matchesUsername := strings.Contains(strings.ToLower(u.Username), "alic")
		matchesDisplay := strings.Contains(strings.ToLower(u.DisplayName), "alic")
		assert.True(t, matchesUsername || matchesDisplay)
	}
}

func TestUserRepository_ListAll_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	users, total, err := repos.User.ListAll(context.Background(), "", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, users)
}

func TestUserRepository_ListPublic_ExcludesBanned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	good := repotest.CreateUser(t, repos, repotest.WithDisplayName("Good"))
	bad := repotest.CreateUser(t, repos, repotest.WithDisplayName("Bad"))
	mod := repotest.CreateUser(t, repos)
	require.NoError(t, repos.User.BanUser(context.Background(), bad.ID, mod.ID, "bad behaviour"))

	// when
	users, err := repos.User.ListPublic(context.Background())

	// then
	require.NoError(t, err)
	ids := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	assert.Contains(t, ids, good.ID)
	assert.Contains(t, ids, mod.ID)
	assert.NotContains(t, ids, bad.ID)
}

func TestUserRepository_ListPublic_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	users, err := repos.User.ListPublic(context.Background())

	// then
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestUserRepository_SearchByName_MatchesUsernameAndDisplay(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos, repotest.WithUsername("battler"), repotest.WithDisplayName("Battler"))
	repotest.CreateUser(t, repos, repotest.WithUsername("ushiromiya_b"), repotest.WithDisplayName("Battler U"))
	repotest.CreateUser(t, repos, repotest.WithUsername("beato"), repotest.WithDisplayName("Beatrice"))

	// when
	users, err := repos.User.SearchByName(context.Background(), "battler", 10)

	// then
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestUserRepository_SearchByName_ExcludesBanned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	visible := repotest.CreateUser(t, repos, repotest.WithUsername("visible_one"))
	hidden := repotest.CreateUser(t, repos, repotest.WithUsername("visible_two"))
	mod := repotest.CreateUser(t, repos)
	require.NoError(t, repos.User.BanUser(context.Background(), hidden.ID, mod.ID, "x"))

	// when
	users, err := repos.User.SearchByName(context.Background(), "visible", 10)

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, visible.ID, users[0].ID)
}

func TestUserRepository_SearchByName_RespectsLimit(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	for i := 0; i < 5; i++ {
		repotest.CreateUser(t, repos, repotest.WithDisplayName("matcher"))
	}

	// when
	users, err := repos.User.SearchByName(context.Background(), "matcher", 3)

	// then
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestUserRepository_SearchByName_NoMatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repotest.CreateUser(t, repos, repotest.WithDisplayName("Alice"))

	// when
	users, err := repos.User.SearchByName(context.Background(), "zzz", 10)

	// then
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestUserRepository_BanUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	mod := repotest.CreateUser(t, repos)

	// when
	err := repos.User.BanUser(context.Background(), user.ID, mod.ID, "spamming")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, got.BannedAt)
	require.NotNil(t, got.BannedBy)
	assert.Equal(t, mod.ID, *got.BannedBy)
	assert.Equal(t, "spamming", got.BanReason)
}

func TestUserRepository_UnbanUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	mod := repotest.CreateUser(t, repos)
	require.NoError(t, repos.User.BanUser(context.Background(), user.ID, mod.ID, "x"))

	// when
	err := repos.User.UnbanUser(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Nil(t, got.BannedAt)
	assert.Nil(t, got.BannedBy)
	assert.Empty(t, got.BanReason)
}

func TestUserRepository_IsBanned_NotBanned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	banned, err := repos.User.IsBanned(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.False(t, banned)
}

func TestUserRepository_IsBanned_Banned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	mod := repotest.CreateUser(t, repos)
	require.NoError(t, repos.User.BanUser(context.Background(), user.ID, mod.ID, "x"))

	// when
	banned, err := repos.User.IsBanned(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.True(t, banned)
}

func TestUserRepository_IsBanned_UserNotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	banned, err := repos.User.IsBanned(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.False(t, banned)
}

func TestUserRepository_AdminDeleteAccount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.User.AdminDeleteAccount(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserRepository_AdminDeleteAccount_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.User.AdminDeleteAccount(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
}
