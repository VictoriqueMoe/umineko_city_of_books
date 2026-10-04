package engines

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"umineko_city_of_books/internal/storage/engine"

	"github.com/stretchr/testify/require"
)

func writeLocalSource(t *testing.T, content string) string {
	t.Helper()
	src := filepath.Join(t.TempDir(), "src.bin")
	require.NoError(t, os.WriteFile(src, []byte(content), 0644))
	return src
}

func readAll(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	defer body.Close()
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	return string(data)
}

func mustPut(t *testing.T, candidate engine.Engine, key string, content string) string {
	t.Helper()
	location, err := candidate.PutFile(context.Background(), key, writeLocalSource(t, content))
	require.NoError(t, err)
	return location
}
