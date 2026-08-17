package dao_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createBotInvocation(t *testing.T, repos *repository.Repositories, botUserID, userID uuid.UUID, channel string, usage repository.InvocationUsage, status repository.InvocationStatus) {
	t.Helper()

	created, err := repos.Chatbot.CreateInvocation(context.Background(), repository.NewInvocation{
		BotUserID: botUserID,
		UserID:    userID,
		MessageID: uuid.New(),
		Channel:   channel,
		Model:     "gpt-5.6-luna",
	})
	require.NoError(t, err)
	id := created.ID
	require.NoError(t, repos.Chatbot.CompleteInvocation(context.Background(), id, usage, status))
}

func TestChatbotDAO_StatsSince_SplitsUsageByChannel(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	bot := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)

	createBotInvocation(t, repos, bot.ID, member.ID, "group",
		repository.InvocationUsage{PromptTokens: 100, CachedPromptTokens: 80, CacheWriteTokens: 10, CompletionTokens: 20, ReasoningTokens: 5},
		repository.InvocationReplied)
	createBotInvocation(t, repos, bot.ID, member.ID, "group",
		repository.InvocationUsage{PromptTokens: 200, CachedPromptTokens: 160, CacheWriteTokens: 20, CompletionTokens: 40, ReasoningTokens: 10},
		repository.InvocationReplied)
	createBotInvocation(t, repos, bot.ID, member.ID, "dm",
		repository.InvocationUsage{PromptTokens: 500, CachedPromptTokens: 400, CacheWriteTokens: 50, CompletionTokens: 60, ReasoningTokens: 15},
		repository.InvocationReplied)
	createBotInvocation(t, repos, bot.ID, member.ID, "post_comment",
		repository.InvocationUsage{PromptTokens: 7},
		repository.InvocationFailed)

	// when
	stats, err := repos.Chatbot.StatsSince(context.Background(), time.Now().Add(-time.Hour))

	// then
	require.NoError(t, err)
	assert.Equal(t, 4, stats.Invocations)
	assert.Equal(t, 807, stats.PromptTokens)
	assert.Equal(t, 640, stats.CachedPromptTokens)
	assert.Equal(t, 80, stats.CacheWriteTokens)
	assert.Equal(t, 120, stats.CompletionTokens)
	assert.Equal(t, 30, stats.ReasoningTokens)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, 0, stats.Quota)
	assert.Equal(t, []repository.ChatbotChannelStats{
		{Channel: "group", Invocations: 2, PromptTokens: 300, CachedPromptTokens: 240, CacheWriteTokens: 30, CompletionTokens: 60, ReasoningTokens: 15},
		{Channel: "dm", Invocations: 1, PromptTokens: 500, CachedPromptTokens: 400, CacheWriteTokens: 50, CompletionTokens: 60, ReasoningTokens: 15},
		{Channel: "post_comment", Invocations: 1, PromptTokens: 7},
	}, stats.Channels)
}

func TestChatbotDAO_StatsSince_IgnoresOlderInvocations(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	bot := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)

	createBotInvocation(t, repos, bot.ID, member.ID, "dm",
		repository.InvocationUsage{PromptTokens: 42},
		repository.InvocationReplied)

	// when
	stats, err := repos.Chatbot.StatsSince(context.Background(), time.Now().Add(time.Hour))

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, stats.Invocations)
	assert.Empty(t, stats.Channels)
	assert.NotNil(t, stats.Channels)
}

func TestChatbotDAO_CreateBotWithAccount_RollsBackEveryTableWhenTheBotInsertFails(t *testing.T) {
	// given a bot creation whose final write is guaranteed to fail, because
	// chatbots.base_prompt_id carries a foreign key onto chatbot_base_prompts
	repos := daotest.NewRepos(t)
	missingBasePrompt := uuid.New()

	account := repository.NewUser{
		Username:      "orphanwitch",
		PasswordHash:  "!",
		DisplayName:   "Orphan Witch",
		HomePage:      "landing",
		IsBot:         true,
		DMsEnabled:    true,
		EmailVerified: true,
	}

	// when the bot row is rejected after the user and the vanity role have already been written
	created, err := repos.Chatbot.CreateBotWithAccount(context.Background(), account, repository.Chatbot{
		SystemPrompt: "you are the golden witch",
		BasePromptID: &missingBasePrompt,
		Model:        "gpt-5.6-luna",
		Enabled:      true,
	}, "bot")

	// then the whole unit of work is undone, leaving no orphan in any of the three tables
	require.Error(t, err)
	assert.Nil(t, created)

	user, err := repos.User.GetByUsername(context.Background(), account.Username)
	require.NoError(t, err)
	assert.Nil(t, user, "the users row must not survive a failed bot creation")

	assignments, err := repos.VanityRole.GetAllAssignments(context.Background())
	require.NoError(t, err)
	assert.Empty(t, assignments, "the user_vanity_roles row must not survive a failed bot creation")

	bots, err := repos.Chatbot.ListBots(context.Background())
	require.NoError(t, err)
	assert.Empty(t, bots)
}
