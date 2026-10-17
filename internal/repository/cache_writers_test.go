package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cachedTableWrites = []*regexp.Regexp{
		regexp.MustCompile(`(?is)\b(INSERT\s+INTO|DELETE\s+FROM)\s+(posts|art)\b`),
		regexp.MustCompile(`(?is)\bUPDATE\s+(posts|art)\s+SET\b[^;]*\bcorner\s*=`),
		regexp.MustCompile(`(?is)\b(INSERT\s+INTO|DELETE\s+FROM|UPDATE)\s+(blocks|chat_banned_words)\b`),
		regexp.MustCompile(`(?is)\bUPDATE\s+users\s+SET\b[^;]*\bip\s*=`),
	}

	invalidatingWriters = []string{
		"CreatePost",
		"DeleteOwnedPost",
		"DeleteOwnedPostAsAdmin",
		"CreateArt",
		"DeleteArtInGallery",
		"DeleteOwnedArt",
		"DeleteOwnedArtAsAdmin",
		"CreateBlock",
		"DeleteBlock",
		"CreateChatBannedWord",
		"UpdateChatBannedWord",
		"DeleteChatBannedWord",
		"UpdateUserIP",
	}
)

func TestCachedTables_EveryWriteQueryIsAKnownInvalidatingPath(t *testing.T) {
	// given
	files, err := filepath.Glob(filepath.Join("..", "dao", "queries", "*.sql"))
	require.NoError(t, err)
	generated, err := filepath.Glob(filepath.Join("..", "dao", "queries", "gen", "*.sql"))
	require.NoError(t, err)
	files = append(files, generated...)
	require.NotEmpty(t, files)

	// when
	var writers []string
	for _, file := range files {
		body, err := os.ReadFile(file)
		require.NoError(t, err)

		for _, chunk := range strings.Split(string(body), "-- name: ")[1:] {
			name := strings.Fields(chunk)[0]
			matches := slices.ContainsFunc(cachedTableWrites, func(pattern *regexp.Regexp) bool {
				return pattern.MatchString(chunk)
			})
			if matches {
				writers = append(writers, name)
			}
		}
	}

	// then
	assert.ElementsMatch(t, invalidatingWriters, writers, "a query writes a table behind the corner-count, blocked-id, banned-word or user-ip caches; route it through a repository method that clears the cache after commit, then list it here")
}
