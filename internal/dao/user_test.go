package dao_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func userInsertSolvedMystery(t *testing.T, repos *repository.Repositories, gmID, winnerID uuid.UUID, difficulty string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := repos.DB().ExecContext(context.Background(),
		`INSERT INTO mysteries (id, user_id, title, body, difficulty, solved, winner_id) VALUES ($1, $2, $3, $4, $5, TRUE, $6)`,
		id, gmID, "title", "body", difficulty, winnerID,
	)
	require.NoError(t, err)

	return id
}

func userIDs(users []model.User) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}

	return ids
}

func userNamesByID(users []model.User) map[uuid.UUID]string {
	names := make(map[uuid.UUID]string, len(users))
	for _, u := range users {
		names[u.ID] = u.Username
	}

	return names
}

func userRegistration(username, verificationHash, sessionToken string) spec.NewRegistration {
	return spec.NewRegistration{
		Account: spec.NewAccount{
			User: spec.NewUser{
				Username:     username,
				Email:        username + "@example.com",
				PasswordHash: "hashed-secret",
				DisplayName:  username,
				HomePage:     "landing",
				DMsEnabled:   true,
			},
			Role: role.RoleSuperAdmin,
		},
		VerificationHash:      verificationHash,
		VerificationExpiresAt: time.Now().Add(24 * time.Hour),
		SessionToken:          sessionToken,
		SessionExpiresAt:      time.Now().Add(time.Hour),
	}
}

func TestUserDAO_Create(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	u, err := repos.User.Create(ctx, spec.NewUser{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "opaque-hash",
		DisplayName:  "Alice",
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, "Alice", u.DisplayName)
	assert.NotEqual(t, uuid.Nil, u.ID)

	byID, err := repos.User.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, byID)
	assert.Equal(t, u.ID, byID.ID)
	assert.Equal(t, "alice", byID.Username)

	hash, err := repos.User.GetPasswordHash(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "opaque-hash", hash, "the hash is stored verbatim, never re-hashed")

	// when the password hash is replaced
	err = repos.User.SetPasswordHash(ctx, spec.UserPasswordHashUpdate{UserID: u.ID, PasswordHash: "brand-new-hash"})

	// then
	require.NoError(t, err)
	hash, err = repos.User.GetPasswordHash(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "brand-new-hash", hash)

	// when an unknown user is looked up
	unknown, err := repos.User.GetByID(ctx, uuid.New())
	require.NoError(t, err)

	unknownHash, err := repos.User.GetPasswordHash(ctx, uuid.New())
	require.NoError(t, err)

	// then
	assert.Nil(t, unknown)
	assert.Empty(t, unknownHash)

	// when the username is taken a second time
	_, err = repos.User.Create(ctx, spec.NewUser{Username: "alice", PasswordHash: "pw2", DisplayName: "Second"})

	// then
	require.Error(t, err)
}

func TestUserDAO_GetByUsername(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos, daotest.WithUsername("MixedCase"))
	daotest.CreateUser(t, repos, daotest.WithUsername("bystander"))

	cases := []struct {
		name     string
		username string
		found    bool
	}{
		{name: "the exact username", username: "MixedCase", found: true},
		{name: "the username in another case", username: "mixedcase", found: true},
		{name: "an unknown username", username: "ghost", found: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := repos.User.GetByUsername(ctx, tc.username)
			require.NoError(t, err)

			exists, err := repos.User.ExistsByUsername(ctx, tc.username)
			require.NoError(t, err)

			// then
			assert.Equal(t, tc.found, exists)
			if !tc.found {
				assert.Nil(t, got)

				return
			}

			require.NotNil(t, got)
			assert.Equal(t, user.ID, got.ID)
		})
	}
}

func TestUserDAO_GetByIDsAndUsernames(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice"))
	bob := daotest.CreateUser(t, repos, daotest.WithUsername("bob"))
	mixed := daotest.CreateUser(t, repos, daotest.WithUsername("MixedCase"))

	// when
	byIDs, err := repos.User.GetByIDs(ctx, []uuid.UUID{alice.ID, bob.ID, uuid.New()})
	require.NoError(t, err)

	byNames, err := repos.User.GetByUsernames(ctx, []string{"alice", "bob", "ghost"})
	require.NoError(t, err)

	byOtherCase, err := repos.User.GetByUsernames(ctx, []string{"mixedcase", "MIXEDCASE"})
	require.NoError(t, err)

	noIDs, err := repos.User.GetByIDs(ctx, nil)
	require.NoError(t, err)

	noNames, err := repos.User.GetByUsernames(ctx, nil)
	require.NoError(t, err)

	// then
	want := map[uuid.UUID]string{alice.ID: "alice", bob.ID: "bob"}

	require.Len(t, byIDs, 2)
	assert.Equal(t, want, userNamesByID(byIDs))

	require.Len(t, byNames, 2)
	assert.Equal(t, want, userNamesByID(byNames))

	require.Len(t, byOtherCase, 1)
	assert.Equal(t, mixed.ID, byOtherCase[0].ID)
	assert.Equal(t, "MixedCase", byOtherCase[0].Username)

	assert.Nil(t, noIDs)
	assert.Nil(t, noNames)
}

func TestUserDAO_CountAndListAll(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice_one"), daotest.WithDisplayName("Alice"))
	bob := daotest.CreateUser(t, repos, daotest.WithUsername("bob_one"), daotest.WithDisplayName("Bob"))
	charlie := daotest.CreateUser(t, repos, daotest.WithUsername("charlie"), daotest.WithDisplayName("Alicia"))
	dora := daotest.CreateUser(t, repos)
	eve := daotest.CreateUser(t, repos)

	// when
	count, err := repos.User.Count(ctx)

	// then
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	cases := []struct {
		name      string
		filter    spec.UserListFilter
		wantTotal int
		wantLen   int
		wantIDs   []uuid.UUID
	}{
		{name: "no search lists everyone", filter: spec.UserListFilter{Limit: 10}, wantTotal: 5, wantLen: 5, wantIDs: []uuid.UUID{alice.ID, bob.ID, charlie.ID, dora.ID, eve.ID}},
		{name: "first page", filter: spec.UserListFilter{Limit: 2, Offset: 0}, wantTotal: 5, wantLen: 2},
		{name: "second page", filter: spec.UserListFilter{Limit: 2, Offset: 2}, wantTotal: 5, wantLen: 2},
		{name: "last page is partial", filter: spec.UserListFilter{Limit: 2, Offset: 4}, wantTotal: 5, wantLen: 1},
		{name: "a page past the end is empty but keeps the total", filter: spec.UserListFilter{Limit: 2, Offset: 10}, wantTotal: 5, wantLen: 0},
		{name: "search matches the username or the display name", filter: spec.UserListFilter{Search: "alic", Limit: 10}, wantTotal: 2, wantLen: 2, wantIDs: []uuid.UUID{alice.ID, charlie.ID}},
		{name: "a search nobody matches has a zero total and an empty page", filter: spec.UserListFilter{Search: "zzz", Limit: 10}, wantTotal: 0, wantLen: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			users, total, err := repos.User.ListAll(ctx, tc.filter)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)
			assert.Len(t, users, tc.wantLen)
			if tc.wantIDs != nil {
				assert.ElementsMatch(t, tc.wantIDs, userIDs(users))
			}
		})
	}
}

func TestUserDAO_UpdateProfile(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.UpdateAvatarURL(ctx, spec.UserAvatarUpdate{UserID: user.ID, AvatarURL: "/uploads/avatars/keep.webp"}))
	require.NoError(t, repos.User.UpdateBannerURL(ctx, spec.UserBannerUpdate{UserID: user.ID, BannerURL: "/uploads/banners/keep.webp"}))

	req := dto.UpdateProfileRequest{
		DisplayName:            "New Name",
		Bio:                    "A bio",
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

	// when
	err := repos.User.UpdateProfile(ctx, spec.UserProfileUpdate{UserID: user.ID, Profile: req})

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(ctx, user.ID)
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
	assert.Equal(t, "/uploads/avatars/keep.webp", got.AvatarURL, "a profile save carries no avatar and must not clobber the uploaded one")
	assert.Equal(t, "/uploads/banners/keep.webp", got.BannerURL, "a profile save carries no banner and must not clobber the uploaded one")

	// when the user does not exist
	err = repos.User.UpdateProfile(ctx, spec.UserProfileUpdate{UserID: uuid.New(), Profile: req})

	// then
	require.NoError(t, err)
}

func TestUserDAO_SingleFieldUpdates(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.MarkEmailVerified(ctx, user.ID))

	// when
	require.NoError(t, repos.User.UpdateIP(ctx, spec.UserIPUpdate{UserID: user.ID, IP: "10.0.0.1"}))
	require.NoError(t, repos.User.MarkEmailUnverified(ctx, user.ID))
	require.NoError(t, repos.User.SetDisplayName(ctx, spec.UserDisplayNameUpdate{UserID: user.ID, DisplayName: "Renamed By Staff"}))
	require.NoError(t, repos.User.SetDisplayNameLocked(ctx, spec.UserDisplayNameLockUpdate{UserID: user.ID, Locked: true}))
	require.NoError(t, repos.User.UpdateGameBoardSort(ctx, spec.UserGameBoardSortUpdate{UserID: user.ID, Sort: "popular"}))
	require.NoError(t, repos.User.UpdateAppearance(ctx, spec.UserAppearanceUpdate{UserID: user.ID, Theme: "dark", Font: "serif", WideLayout: true}))
	require.NoError(t, repos.User.UpdateMysteryScoreAdjustment(ctx, spec.UserMysteryScoreUpdate{UserID: user.ID, Adjustment: 50}))
	require.NoError(t, repos.User.UpdateGMScoreAdjustment(ctx, spec.UserGMScoreUpdate{UserID: user.ID, Adjustment: -25}))

	// then
	got, err := repos.User.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, got.IP)
	assert.Equal(t, "10.0.0.1", *got.IP)
	assert.False(t, got.EmailVerified)
	assert.Equal(t, "Renamed By Staff", got.DisplayName)
	assert.True(t, got.DisplayNameLocked)
	assert.Equal(t, "popular", got.GameBoardSort)
	assert.Equal(t, "dark", got.Theme)
	assert.Equal(t, "serif", got.Font)
	assert.True(t, got.WideLayout)
	assert.Equal(t, 50, got.MysteryScoreAdjustment)
	assert.Equal(t, -25, got.GMScoreAdjustment)

	// when the display name is unlocked again
	err = repos.User.SetDisplayNameLocked(ctx, spec.UserDisplayNameLockUpdate{UserID: user.ID, Locked: false})

	// then
	require.NoError(t, err)
	got, err = repos.User.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, got.DisplayNameLocked)
}

func TestUserDAO_ListByIP(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	ip := "2a00:23c8:ec30:1001:65c3:a122:a356:90c4"
	target := daotest.CreateUser(t, repos, daotest.WithUsername("target"))
	alt := daotest.CreateUser(t, repos, daotest.WithUsername("alt"))
	elsewhere := daotest.CreateUser(t, repos, daotest.WithUsername("elsewhere"))
	require.NoError(t, repos.User.UpdateIP(ctx, spec.UserIPUpdate{UserID: target.ID, IP: ip}))
	require.NoError(t, repos.User.UpdateIP(ctx, spec.UserIPUpdate{UserID: alt.ID, IP: ip}))
	require.NoError(t, repos.User.UpdateIP(ctx, spec.UserIPUpdate{UserID: elsewhere.ID, IP: "10.0.0.1"}))

	cases := []struct {
		name   string
		filter spec.UserIPFilter
		want   []uuid.UUID
	}{
		{name: "lists the other accounts on the same ip", filter: spec.UserIPFilter{IP: ip, ExcludeUserID: target.ID}, want: []uuid.UUID{alt.ID}},
		{name: "an account alone on its ip has no matches", filter: spec.UserIPFilter{IP: "10.0.0.1", ExcludeUserID: elsewhere.ID}, want: []uuid.UUID{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := repos.User.ListByIP(ctx, tc.filter)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, userIDs(got))
		})
	}
}

func TestUserDAO_RawScores(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	gm := daotest.CreateUser(t, repos)
	soloGM := daotest.CreateUser(t, repos)
	winner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	for _, difficulty := range []string{"easy", "hard", "nightmare"} {
		userInsertSolvedMystery(t, repos, gm.ID, winner.ID, difficulty)
	}

	attempted := userInsertSolvedMystery(t, repos, soloGM.ID, winner.ID, "medium")
	for _, attempter := range []uuid.UUID{winner.ID, other.ID} {
		_, err := repos.DB().ExecContext(ctx, `INSERT INTO mystery_attempts (id, mystery_id, user_id, body) VALUES ($1, $2, $3, $4)`, uuid.New(), attempted, attempter, "guess")
		require.NoError(t, err)
	}

	cases := []struct {
		name   string
		score  func(repository.UserRepository, context.Context, uuid.UUID, ...*sql.Tx) (int, error)
		userID uuid.UUID
		want   int
	}{
		{name: "detective score weights every solved difficulty", score: repository.UserRepository.GetDetectiveRawScore, userID: winner.ID, want: 2 + 4 + 6 + 8},
		{name: "detective score with no win is zero", score: repository.UserRepository.GetDetectiveRawScore, userID: other.ID, want: 0},
		{name: "gm score weights each solved difficulty and ignores attempts on other mysteries", score: repository.UserRepository.GetGMRawScore, userID: gm.ID, want: 2 + 6 + 8},
		{name: "gm score adds one per distinct attempter", score: repository.UserRepository.GetGMRawScore, userID: soloGM.ID, want: 4 + 2},
		{name: "gm score with no solved mystery is zero", score: repository.UserRepository.GetGMRawScore, userID: other.ID, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			score, err := tc.score(repos.User, ctx, tc.userID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, score)
		})
	}
}

func TestUserDAO_DeleteAccount(t *testing.T) {
	cases := []struct {
		name   string
		remove func(repository.UserRepository, context.Context, uuid.UUID, ...*sql.Tx) error
	}{
		{name: "self-service delete", remove: repository.UserRepository.DeleteAccount},
		{name: "admin delete", remove: repository.UserRepository.AdminDeleteAccount},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			user := daotest.CreateUser(t, repos)

			// when an unknown user is deleted
			err := tc.remove(repos.User, ctx, uuid.New())

			// then it is a no-op
			require.NoError(t, err)
			still, err := repos.User.GetByID(ctx, user.ID)
			require.NoError(t, err)
			assert.NotNil(t, still)

			// when
			err = tc.remove(repos.User, ctx, user.ID)

			// then
			require.NoError(t, err)
			gone, err := repos.User.GetByID(ctx, user.ID)
			require.NoError(t, err)
			assert.Nil(t, gone)
		})
	}
}

func TestUserDAO_GetProfile(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos, daotest.WithUsername("profuser"))
	solver := daotest.CreateUser(t, repos)
	userInsertSolvedMystery(t, repos, author.ID, author.ID, "easy")
	userInsertSolvedMystery(t, repos, author.ID, solver.ID, "hard")

	// when
	byName, byNameStats, err := repos.User.GetProfileByUsername(ctx, "profuser")
	require.NoError(t, err)

	authorByID, authorStats, err := repos.User.GetProfileByID(ctx, author.ID)
	require.NoError(t, err)

	solverByID, solverStats, err := repos.User.GetProfileByID(ctx, solver.ID)
	require.NoError(t, err)

	unknownName, unknownNameStats, err := repos.User.GetProfileByUsername(ctx, "ghost")
	require.NoError(t, err)

	unknownID, unknownIDStats, err := repos.User.GetProfileByID(ctx, uuid.New())
	require.NoError(t, err)

	// then
	require.NotNil(t, byName)
	assert.Equal(t, author.ID, byName.ID)
	assert.Equal(t, &model.UserStats{MysteryCount: 2}, byNameStats)

	require.NotNil(t, authorByID)
	assert.Equal(t, author.ID, authorByID.ID)
	assert.Equal(t, &model.UserStats{MysteryCount: 2}, authorStats)

	require.NotNil(t, solverByID)
	assert.Equal(t, solver.ID, solverByID.ID)
	assert.Equal(t, &model.UserStats{}, solverStats, "solving someone else's mystery does not count as one of the solver's own")

	assert.Nil(t, unknownName)
	assert.Nil(t, unknownNameStats)
	assert.Nil(t, unknownID)
	assert.Nil(t, unknownIDStats)
}

func TestUserDAO_ListPublic_ExcludesBanned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	good := daotest.CreateUser(t, repos, daotest.WithDisplayName("Good"))
	bad := daotest.CreateUser(t, repos, daotest.WithDisplayName("Bad"))
	mod := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.BanUser(ctx, spec.UserBan{UserID: bad.ID, BannedBy: mod.ID, Reason: "bad behaviour"}))

	// when
	users, err := repos.User.ListPublic(ctx)

	// then
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{good.ID, mod.ID}, userIDs(users), "the banned user is left out and the rest are ordered by display name")
}

func TestUserDAO_SearchByName(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	battler := daotest.CreateUser(t, repos, daotest.WithUsername("battler"), daotest.WithDisplayName("Battler"))
	ushiromiya := daotest.CreateUser(t, repos, daotest.WithUsername("ushiromiya_b"), daotest.WithDisplayName("Battler U"))
	daotest.CreateUser(t, repos, daotest.WithUsername("beato"), daotest.WithDisplayName("Beatrice"))
	visible := daotest.CreateUser(t, repos, daotest.WithUsername("visible_one"))
	hidden := daotest.CreateUser(t, repos, daotest.WithUsername("visible_two"))
	mod := daotest.CreateUser(t, repos)
	require.NoError(t, repos.User.BanUser(ctx, spec.UserBan{UserID: hidden.ID, BannedBy: mod.ID, Reason: "x"}))

	matchers := make([]uuid.UUID, 0, 5)
	for n := range 5 {
		matchers = append(matchers, daotest.CreateUser(t, repos, daotest.WithDisplayName(fmt.Sprintf("matcher %d", n+1))).ID)
	}

	cases := []struct {
		name   string
		filter spec.UserSearchFilter
		want   []uuid.UUID
	}{
		{name: "matches the username or the display name, username prefix first", filter: spec.UserSearchFilter{Query: "battler", Limit: 10}, want: []uuid.UUID{battler.ID, ushiromiya.ID}},
		{name: "leaves out banned users", filter: spec.UserSearchFilter{Query: "visible", Limit: 10}, want: []uuid.UUID{visible.ID}},
		{name: "respects the limit", filter: spec.UserSearchFilter{Query: "matcher", Limit: 3}, want: matchers[:3]},
		{name: "nothing matches", filter: spec.UserSearchFilter{Query: "zzz", Limit: 10}, want: []uuid.UUID{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			users, err := repos.User.SearchByName(ctx, tc.filter)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, userIDs(users))
		})
	}
}

func TestUserDAO_BanAndUnban(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	mod := daotest.CreateUser(t, repos)

	// when
	notBanned, err := repos.User.IsBanned(ctx, user.ID)
	require.NoError(t, err)

	unknown, err := repos.User.IsBanned(ctx, uuid.New())
	require.NoError(t, err)

	// then
	assert.False(t, notBanned)
	assert.False(t, unknown)

	// when the user is banned
	err = repos.User.BanUser(ctx, spec.UserBan{UserID: user.ID, BannedBy: mod.ID, Reason: "spamming"})

	// then
	require.NoError(t, err)
	got, err := repos.User.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, got.BannedAt)
	require.NotNil(t, got.BannedBy)
	assert.Equal(t, mod.ID, *got.BannedBy)
	assert.Equal(t, "spamming", got.BanReason)

	banned, err := repos.User.IsBanned(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, banned)

	// when the ban is lifted
	err = repos.User.UnbanUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	got, err = repos.User.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Nil(t, got.BannedAt)
	assert.Nil(t, got.BannedBy)
	assert.Empty(t, got.BanReason)

	banned, err = repos.User.IsBanned(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, banned)
}

func TestUserRepository_RegisterAccountWritesEveryRow(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	inviter := daotest.CreateUser(t, repos, daotest.WithUsername("inviter"))
	require.NoError(t, repos.Invite.Create(ctx, spec.NewInvite{Code: "invite-code-1", CreatedBy: inviter.ID}))

	registration := userRegistration("newcomer", "verify-hash-1", "session-token-1")
	registration.InviteCode = "invite-code-1"

	// when
	created, err := repos.User.RegisterAccount(ctx, registration)

	// then
	require.NoError(t, err)
	require.NotNil(t, created)

	assigned, err := repos.Role.GetRole(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, role.RoleSuperAdmin, assigned)

	verification, err := repos.EmailVerification.GetByTokenHash(ctx, "verify-hash-1")
	require.NoError(t, err)
	require.NotNil(t, verification)
	assert.Equal(t, created.ID, verification.UserID)

	invite, err := repos.Invite.GetByCode(ctx, "invite-code-1")
	require.NoError(t, err)
	require.NotNil(t, invite)
	require.NotNil(t, invite.UsedBy)
	assert.Equal(t, created.ID, *invite.UsedBy)

	sessionUserID, _, err := repos.Session.GetUserID(ctx, "session-token-1")
	require.NoError(t, err)
	assert.Equal(t, created.ID, sessionUserID)

	entries, total, err := repos.AuditLog.ListForUser(ctx, spec.AuditLogUserListing{UserID: created.ID, Page: bounds.NewPage(10, 0)})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionUserCreated, entries[0].Action)
	assert.Equal(t, "username=newcomer", entries[0].Details)
}

func TestUserRepository_RegisterAccountRollsBackWhenSessionCreationFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	holder := daotest.CreateUser(t, repos, daotest.WithUsername("tokenholder"))
	takenToken := daotest.CreateSession(t, repos, holder.ID)

	// when
	created, err := repos.User.RegisterAccount(ctx, userRegistration("rolledback", "verify-hash-2", takenToken))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create session")
	assert.Nil(t, created)

	orphan, err := repos.User.GetByUsername(ctx, "rolledback")
	require.NoError(t, err)
	assert.Nil(t, orphan)

	verification, err := repos.EmailVerification.GetByTokenHash(ctx, "verify-hash-2")
	require.NoError(t, err)
	assert.Nil(t, verification)
}
