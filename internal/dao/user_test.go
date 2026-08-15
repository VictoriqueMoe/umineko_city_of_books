package dao_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"

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
		SocialBluesky:          "bsky",
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
		DefaultProfileTab:      "ocs",
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

func TestUserDAO_Create(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	u, err := repos.User.Create(context.Background(), repository.NewUser{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hashed-secret123",
		DisplayName:  "Alice",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, "Alice", u.DisplayName)
	assert.NotEqual(t, uuid.Nil, u.ID)
}

func TestUserDAO_Create_DuplicateUsername(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	_, err := repos.User.Create(context.Background(), repository.NewUser{Username: "dup", PasswordHash: "pw1", DisplayName: "First"})
	require.NoError(t, err)

	// when
	_, err = repos.User.Create(context.Background(), repository.NewUser{Username: "dup", PasswordHash: "pw2", DisplayName: "Second"})

	// then
	require.Error(t, err)
}

func TestUserDAO_GetByID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	created := daotest.CreateUser(t, repos, daotest.WithUsername("byid"))

	// when
	got, err := repos.User.GetByID(context.Background(), created.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "byid", got.Username)
}

func TestUserDAO_GetByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.User.GetByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserDAO_GetByUsername(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	created := daotest.CreateUser(t, repos, daotest.WithUsername("byname"))

	// when
	got, err := repos.User.GetByUsername(context.Background(), "byname")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
}

func TestUserDAO_GetByUsername_CaseInsensitive(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	created := daotest.CreateUser(t, repos, daotest.WithUsername("MixedCase"))

	// when
	got, err := repos.User.GetByUsername(context.Background(), "mixedcase")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
}

func TestUserDAO_GetByUsername_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.User.GetByUsername(context.Background(), "ghost")

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserDAO_GetByIDs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	a := daotest.CreateUser(t, repos, daotest.WithUsername("alice"))
	b := daotest.CreateUser(t, repos, daotest.WithUsername("bob"))
	daotest.CreateUser(t, repos, daotest.WithUsername("carol"))
	ghost := uuid.New()

	// when
	got, err := repos.User.GetByIDs(context.Background(), []uuid.UUID{a.ID, b.ID, ghost})

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	byID := map[uuid.UUID]string{}
	for i := range got {
		byID[got[i].ID] = got[i].Username
	}
	assert.Equal(t, "alice", byID[a.ID])
	assert.Equal(t, "bob", byID[b.ID])
}

func TestUserDAO_GetByIDs_EmptyInput(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.User.GetByIDs(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserDAO_GetByUsernames(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	a := daotest.CreateUser(t, repos, daotest.WithUsername("alice"))
	b := daotest.CreateUser(t, repos, daotest.WithUsername("bob"))
	daotest.CreateUser(t, repos, daotest.WithUsername("carol"))

	// when
	got, err := repos.User.GetByUsernames(context.Background(), []string{"alice", "bob", "ghost"})

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	byID := map[uuid.UUID]string{}
	for i := range got {
		byID[got[i].ID] = got[i].Username
	}
	assert.Equal(t, "alice", byID[a.ID])
	assert.Equal(t, "bob", byID[b.ID])
}

func TestUserDAO_GetByUsernames_CaseInsensitive(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	daotest.CreateUser(t, repos, daotest.WithUsername("MixedCase"))

	// when
	got, err := repos.User.GetByUsernames(context.Background(), []string{"mixedcase", "MIXEDCASE"})

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "MixedCase", got[0].Username)
}

func TestUserDAO_GetByUsernames_EmptyInput(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.User.GetByUsernames(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserDAO_ExistsByUsername(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	daotest.CreateUser(t, repos, daotest.WithUsername("exists"))

	// when
	exists, err := repos.User.ExistsByUsername(context.Background(), "exists")
	missing, err2 := repos.User.ExistsByUsername(context.Background(), "ghost")

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.True(t, exists)
	assert.False(t, missing)
}

func TestUserDAO_ExistsByUsername_CaseInsensitive(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	daotest.CreateUser(t, repos, daotest.WithUsername("CaseUser"))

	// when
	exists, err := repos.User.ExistsByUsername(context.Background(), "caseuser")

	// then
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUserDAO_Count(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	daotest.CreateUser(t, repos)
	daotest.CreateUser(t, repos)
	daotest.CreateUser(t, repos)

	// when
	count, err := repos.User.Count(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestUserDAO_Count_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	count, err := repos.User.Count(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestUserDAO_GetPasswordHash_ReturnsStoredHashVerbatim(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	created, err := repos.User.Create(context.Background(), repository.NewUser{Username: "vpwd", PasswordHash: "opaque-hash", DisplayName: "V"})
	require.NoError(t, err)

	// when
	hash, err := repos.User.GetPasswordHash(context.Background(), created.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "opaque-hash", hash)
}

func TestUserDAO_GetPasswordHash_UnknownUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	hash, err := repos.User.GetPasswordHash(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Empty(t, hash)
}

func TestUserDAO_UpdateProfile(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	req := sampleProfileRequest()

	// when
	err := repos.User.UpdateProfile(context.Background(), user.ID, req)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, req.DisplayName, got.DisplayName)
	assert.Equal(t, req.Bio, got.Bio)
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
	assert.Equal(t, req.SocialBluesky, got.SocialBluesky)
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
	assert.Equal(t, req.DefaultProfileTab, got.DefaultProfileTab)
}

func TestUserDAO_UpdateProfile_NonExistentUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	err := repos.User.UpdateProfile(context.Background(), uuid.New(), sampleProfileRequest())

	// then
	require.NoError(t, err)
}

func TestUserDAO_UpdateProfile_DoesNotClobberAvatarOrBanner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.UpdateAvatarURL(context.Background(), user.ID, "/uploads/avatars/keep.webp"))
	require.NoError(t, repos.User.UpdateBannerURL(context.Background(), user.ID, "/uploads/banners/keep.webp"))

	req := sampleProfileRequest()
	req.AvatarURL = ""
	req.BannerURL = ""

	// when
	err := repos.User.UpdateProfile(context.Background(), user.ID, req)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "/uploads/avatars/keep.webp", got.AvatarURL)
	assert.Equal(t, "/uploads/banners/keep.webp", got.BannerURL)
}

func TestUserDAO_UpdateAvatarURL(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateAvatarURL(context.Background(), user.ID, "/new-avatar.png")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "/new-avatar.png", got.AvatarURL)
}

func TestUserDAO_UpdateBannerURL(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateBannerURL(context.Background(), user.ID, "/new-banner.png")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "/new-banner.png", got.BannerURL)
}

func TestUserDAO_UpdateIP(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateIP(context.Background(), user.ID, "10.0.0.1")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, got.IP)
	assert.Equal(t, "10.0.0.1", *got.IP)
}

func TestUserDAO_MarkEmailUnverified(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.MarkEmailVerified(context.Background(), user.ID))

	// when
	err := repos.User.MarkEmailUnverified(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.False(t, got.EmailVerified)
}

func TestUserDAO_SetDisplayName(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.SetDisplayName(context.Background(), user.ID, "Renamed By Staff")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed By Staff", got.DisplayName)
}

func TestUserDAO_SetDisplayNameLocked(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.SetDisplayNameLocked(context.Background(), user.ID, true)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.True(t, got.DisplayNameLocked)

	// when
	err = repos.User.SetDisplayNameLocked(context.Background(), user.ID, false)

	// then
	require.NoError(t, err)
	got, err = repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.False(t, got.DisplayNameLocked)
}

func TestUserDAO_ListByIP(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ip := "2a00:23c8:ec30:1001:65c3:a122:a356:90c4"
	target := daotest.CreateUser(t, repos, daotest.WithUsername("target"))
	alt := daotest.CreateUser(t, repos, daotest.WithUsername("alt"))
	elsewhere := daotest.CreateUser(t, repos, daotest.WithUsername("elsewhere"))
	require.NoError(t, repos.User.UpdateIP(context.Background(), target.ID, ip))
	require.NoError(t, repos.User.UpdateIP(context.Background(), alt.ID, ip))
	require.NoError(t, repos.User.UpdateIP(context.Background(), elsewhere.ID, "10.0.0.1"))

	// when
	got, err := repos.User.ListByIP(context.Background(), ip, target.ID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, alt.ID, got[0].ID)
}

func TestUserDAO_ListByIP_NoMatches(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	target := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.UpdateIP(context.Background(), target.ID, "10.0.0.1"))

	// when
	got, err := repos.User.ListByIP(context.Background(), "10.0.0.1", target.ID)

	// then
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestUserDAO_UpdateGameBoardSort(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateGameBoardSort(context.Background(), user.ID, "popular")

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "popular", got.GameBoardSort)
}

func TestUserDAO_UpdateAppearance(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

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

func TestUserDAO_UpdateMysteryScoreAdjustment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateMysteryScoreAdjustment(context.Background(), user.ID, 50)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, got.MysteryScoreAdjustment)
}

func TestUserDAO_UpdateGMScoreAdjustment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.UpdateGMScoreAdjustment(context.Background(), user.ID, -25)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, -25, got.GMScoreAdjustment)
}

func TestUserDAO_GetDetectiveRawScore_NoMysteries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	score, err := repos.User.GetDetectiveRawScore(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, score)
}

func TestUserDAO_GetDetectiveRawScore_VariousDifficulties(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	winner := daotest.CreateUser(t, repos)
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

func TestUserDAO_GetGMRawScore_NoMysteries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	score, err := repos.User.GetGMRawScore(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, score)
}

func TestUserDAO_GetGMRawScore_WithAttempts(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	winner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	mysteryID := insertSolvedMystery(t, repos, gm.ID, winner.ID, "medium")
	insertMysteryAttempt(t, repos, mysteryID, winner.ID)
	insertMysteryAttempt(t, repos, mysteryID, other.ID)

	// when
	score, err := repos.User.GetGMRawScore(context.Background(), gm.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 4+2, score)
}

func TestUserDAO_SetPasswordHash(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("cp"))

	// when
	err := repos.User.SetPasswordHash(context.Background(), user.ID, "brand-new-hash")

	// then
	require.NoError(t, err)
	hash, err := repos.User.GetPasswordHash(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "brand-new-hash", hash)
}

func TestUserDAO_DeleteAccount_Success(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.DeleteAccount(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserDAO_DeleteAccount_UnknownUserIsANoOp(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.DeleteAccount(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestUserDAO_GetProfileByUsername(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("profuser"))

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

func TestUserDAO_GetProfileByUsername_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	u, stats, err := repos.User.GetProfileByUsername(context.Background(), "ghost")

	// then
	require.NoError(t, err)
	assert.Nil(t, u)
	assert.Nil(t, stats)
}

func TestUserDAO_GetProfileByID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	gm := daotest.CreateUser(t, repos)
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

func TestUserDAO_GetProfileByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	u, stats, err := repos.User.GetProfileByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, u)
	assert.Nil(t, stats)
}

func TestUserDAO_GetProfileByID_CountsMysteries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	insertSolvedMystery(t, repos, user.ID, user.ID, "easy")
	insertSolvedMystery(t, repos, user.ID, user.ID, "hard")

	// when
	_, stats, err := repos.User.GetProfileByID(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 2, stats.MysteryCount)
}

func TestUserDAO_ListAll_NoSearch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	daotest.CreateUser(t, repos, daotest.WithUsername("user1"))
	daotest.CreateUser(t, repos, daotest.WithUsername("user2"))
	daotest.CreateUser(t, repos, daotest.WithUsername("user3"))

	// when
	users, total, err := repos.User.ListAll(context.Background(), "", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, users, 3)
}

func TestUserDAO_ListAll_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	for range 5 {
		daotest.CreateUser(t, repos)
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

func TestUserDAO_ListAll_Search(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	daotest.CreateUser(t, repos, daotest.WithUsername("alice_one"), daotest.WithDisplayName("Alice"))
	daotest.CreateUser(t, repos, daotest.WithUsername("bob_one"), daotest.WithDisplayName("Bob"))
	daotest.CreateUser(t, repos, daotest.WithUsername("charlie"), daotest.WithDisplayName("Alicia"))

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

func TestUserDAO_ListAll_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	users, total, err := repos.User.ListAll(context.Background(), "", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, users)
}

func TestUserDAO_ListPublic_ExcludesBanned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	good := daotest.CreateUser(t, repos, daotest.WithDisplayName("Good"))
	bad := daotest.CreateUser(t, repos, daotest.WithDisplayName("Bad"))
	mod := daotest.CreateUser(t, repos)
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

func TestUserDAO_ListPublic_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	users, err := repos.User.ListPublic(context.Background())

	// then
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestUserDAO_SearchByName_MatchesUsernameAndDisplay(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	daotest.CreateUser(t, repos, daotest.WithUsername("battler"), daotest.WithDisplayName("Battler"))
	daotest.CreateUser(t, repos, daotest.WithUsername("ushiromiya_b"), daotest.WithDisplayName("Battler U"))
	daotest.CreateUser(t, repos, daotest.WithUsername("beato"), daotest.WithDisplayName("Beatrice"))

	// when
	users, err := repos.User.SearchByName(context.Background(), "battler", 10)

	// then
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestUserDAO_SearchByName_ExcludesBanned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	visible := daotest.CreateUser(t, repos, daotest.WithUsername("visible_one"))
	hidden := daotest.CreateUser(t, repos, daotest.WithUsername("visible_two"))
	mod := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.BanUser(context.Background(), hidden.ID, mod.ID, "x"))

	// when
	users, err := repos.User.SearchByName(context.Background(), "visible", 10)

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, visible.ID, users[0].ID)
}

func TestUserDAO_SearchByName_RespectsLimit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	for range 5 {
		daotest.CreateUser(t, repos, daotest.WithDisplayName("matcher"))
	}

	// when
	users, err := repos.User.SearchByName(context.Background(), "matcher", 3)

	// then
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestUserDAO_SearchByName_NoMatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	daotest.CreateUser(t, repos, daotest.WithDisplayName("Alice"))

	// when
	users, err := repos.User.SearchByName(context.Background(), "zzz", 10)

	// then
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestUserDAO_BanUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	mod := daotest.CreateUser(t, repos)

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

func TestUserDAO_UnbanUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	mod := daotest.CreateUser(t, repos)
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

func TestUserDAO_IsBanned_NotBanned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	banned, err := repos.User.IsBanned(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.False(t, banned)
}

func TestUserDAO_IsBanned_Banned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	mod := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.BanUser(context.Background(), user.ID, mod.ID, "x"))

	// when
	banned, err := repos.User.IsBanned(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.True(t, banned)
}

func TestUserDAO_IsBanned_UserNotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	banned, err := repos.User.IsBanned(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.False(t, banned)
}

func TestUserDAO_AdminDeleteAccount(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.User.AdminDeleteAccount(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserDAO_AdminDeleteAccount_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	err := repos.User.AdminDeleteAccount(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
}

func TestUserRepository_RegisterAccountWritesEveryRow(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	inviter := daotest.CreateUser(t, repos, daotest.WithUsername("inviter"))
	require.NoError(t, repos.Invite.Create(context.Background(), "invite-code-1", inviter.ID))

	spec := repository.NewRegistration{
		Account: repository.NewAccount{
			User: repository.NewUser{
				Username:     "newcomer",
				Email:        "newcomer@example.com",
				PasswordHash: "hashed-secret",
				DisplayName:  "Newcomer",
				HomePage:     "landing",
				DMsEnabled:   true,
			},
			Role: role.RoleSuperAdmin,
		},
		InviteCode:            "invite-code-1",
		VerificationHash:      "verify-hash-1",
		VerificationExpiresAt: time.Now().Add(24 * time.Hour),
		SessionToken:          "session-token-1",
		SessionExpiresAt:      time.Now().Add(time.Hour),
	}

	// when
	created, err := repos.User.RegisterAccount(context.Background(), spec)

	// then
	require.NoError(t, err)
	require.NotNil(t, created)

	assigned, err := repos.Role.GetRole(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, role.RoleSuperAdmin, assigned)

	verification, err := repos.EmailVerification.GetByTokenHash(context.Background(), "verify-hash-1")
	require.NoError(t, err)
	require.NotNil(t, verification)
	assert.Equal(t, created.ID, verification.UserID)

	invite, err := repos.Invite.GetByCode(context.Background(), "invite-code-1")
	require.NoError(t, err)
	require.NotNil(t, invite)
	require.NotNil(t, invite.UsedBy)
	assert.Equal(t, created.ID, *invite.UsedBy)

	sessionUserID, _, err := repos.Session.GetUserID(context.Background(), "session-token-1")
	require.NoError(t, err)
	assert.Equal(t, created.ID, sessionUserID)

	entries, total, err := repos.AuditLog.ListForUser(context.Background(), created.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, entries, 1)
	assert.Equal(t, repository.AuditActionUserCreated, entries[0].Action)
	assert.Equal(t, "username=newcomer", entries[0].Details)
}

func TestUserRepository_RegisterAccountRollsBackWhenSessionCreationFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	holder := daotest.CreateUser(t, repos, daotest.WithUsername("tokenholder"))
	takenToken := daotest.CreateSession(t, repos, holder.ID)

	spec := repository.NewRegistration{
		Account: repository.NewAccount{
			User: repository.NewUser{
				Username:     "rolledback",
				Email:        "rolledback@example.com",
				PasswordHash: "hashed-secret",
				DisplayName:  "Rolled Back",
				HomePage:     "landing",
				DMsEnabled:   true,
			},
			Role: role.RoleSuperAdmin,
		},
		VerificationHash:      "verify-hash-2",
		VerificationExpiresAt: time.Now().Add(24 * time.Hour),
		SessionToken:          takenToken,
		SessionExpiresAt:      time.Now().Add(time.Hour),
	}

	// when
	created, err := repos.User.RegisterAccount(context.Background(), spec)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create session")
	assert.Nil(t, created)

	orphan, err := repos.User.GetByUsername(context.Background(), "rolledback")
	require.NoError(t, err)
	assert.Nil(t, orphan)

	verification, err := repos.EmailVerification.GetByTokenHash(context.Background(), "verify-hash-2")
	require.NoError(t, err)
	assert.Nil(t, verification)
}
