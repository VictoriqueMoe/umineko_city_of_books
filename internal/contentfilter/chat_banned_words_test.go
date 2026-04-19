package contentfilter

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCompileBannedWordPattern_Substring(t *testing.T) {
	re, err := CompileBannedWordPattern("dog", MatchModeSubstring, false)
	require.NoError(t, err)
	assert.True(t, re.MatchString("I love Dogs"))
	assert.True(t, re.MatchString("hotdog!"))
	assert.False(t, re.MatchString("cat"))
}

func TestCompileBannedWordPattern_WholeWord(t *testing.T) {
	re, err := CompileBannedWordPattern("class", MatchModeWholeWord, false)
	require.NoError(t, err)
	assert.True(t, re.MatchString("a class act"))
	assert.False(t, re.MatchString("classic dish"))
}

func TestCompileBannedWordPattern_Regex(t *testing.T) {
	re, err := CompileBannedWordPattern(`\bbombs?\b`, MatchModeRegex, false)
	require.NoError(t, err)
	assert.True(t, re.MatchString("Bomb in hand"))
	assert.True(t, re.MatchString("many bombs"))
	assert.False(t, re.MatchString("bombastic"))
}

func TestCompileBannedWordPattern_CaseSensitive(t *testing.T) {
	re, err := CompileBannedWordPattern("Dog", MatchModeSubstring, true)
	require.NoError(t, err)
	assert.True(t, re.MatchString("my Dog"))
	assert.False(t, re.MatchString("my dog"))
}

func TestCompileBannedWordPattern_InvalidRegexFails(t *testing.T) {
	_, err := CompileBannedWordPattern(`\b(foo`, MatchModeRegex, false)
	require.Error(t, err)
}

func TestCompileBannedWordPattern_UnknownModeFails(t *testing.T) {
	_, err := CompileBannedWordPattern("x", "nope", false)
	require.Error(t, err)
}

func TestCheckForRoom_NoMatch(t *testing.T) {
	repo := repository.NewMockChatBannedWordRepository(t)
	repo.EXPECT().ListApplicable(mock.Anything, mock.Anything).Return([]repository.ChatBannedWordRow{
		{ID: uuid.New(), Pattern: "dogs", MatchMode: MatchModeSubstring, Action: BannedWordActionDelete, Scope: "global"},
	}, nil)
	rule := NewChatBannedWordsRule(repo)
	match, err := rule.CheckForRoom(context.Background(), uuid.New(), "I love cats")
	require.NoError(t, err)
	assert.Nil(t, match)
}

func TestCheckForRoom_FirstMatchWins(t *testing.T) {
	repo := repository.NewMockChatBannedWordRepository(t)
	roomID := uuid.New()
	first := uuid.New()
	second := uuid.New()
	repo.EXPECT().ListApplicable(mock.Anything, roomID).Return([]repository.ChatBannedWordRow{
		{ID: first, Pattern: "dogs", MatchMode: MatchModeSubstring, Action: BannedWordActionDelete, Scope: "global"},
		{ID: second, Pattern: "dogs", MatchMode: MatchModeSubstring, Action: BannedWordActionKick, Scope: "room"},
	}, nil)
	rule := NewChatBannedWordsRule(repo)
	match, err := rule.CheckForRoom(context.Background(), roomID, "I love dogs")
	require.NoError(t, err)
	require.NotNil(t, match)
	assert.Equal(t, first, match.RuleID)
	assert.Equal(t, BannedWordActionDelete, match.Action)
	assert.Equal(t, "dogs", match.MatchedOn)
}

func TestCheckForRoom_KickAction(t *testing.T) {
	repo := repository.NewMockChatBannedWordRepository(t)
	roomID := uuid.New()
	repo.EXPECT().ListApplicable(mock.Anything, roomID).Return([]repository.ChatBannedWordRow{
		{ID: uuid.New(), Pattern: `\bbombs?\b`, MatchMode: MatchModeRegex, Action: BannedWordActionKick, Scope: "global"},
	}, nil)
	rule := NewChatBannedWordsRule(repo)
	match, err := rule.CheckForRoom(context.Background(), roomID, "bombs away")
	require.NoError(t, err)
	require.NotNil(t, match)
	assert.Equal(t, BannedWordActionKick, match.Action)
}

func TestCheckForRoom_InvalidRulesSkipped(t *testing.T) {
	repo := repository.NewMockChatBannedWordRepository(t)
	roomID := uuid.New()
	valid := uuid.New()
	repo.EXPECT().ListApplicable(mock.Anything, roomID).Return([]repository.ChatBannedWordRow{
		{ID: uuid.New(), Pattern: `\b(foo`, MatchMode: MatchModeRegex, Action: BannedWordActionDelete, Scope: "global"},
		{ID: valid, Pattern: "cat", MatchMode: MatchModeSubstring, Action: BannedWordActionDelete, Scope: "global"},
	}, nil)
	rule := NewChatBannedWordsRule(repo)
	match, err := rule.CheckForRoom(context.Background(), roomID, "the cat sat")
	require.NoError(t, err)
	require.NotNil(t, match)
	assert.Equal(t, valid, match.RuleID)
}
