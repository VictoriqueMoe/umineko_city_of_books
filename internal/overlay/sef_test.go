package overlay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSEF_IncludesSiteNameAndURL(t *testing.T) {
	// given
	connectURL := "wss://umineko.example/api/v1/overlay?token=tok_abc"
	siteName := "Umineko DB"

	// when
	out, err := renderSEF(connectURL, siteName)

	// then
	require.NoError(t, err)
	assert.Contains(t, out, "Umineko DB Overlay")
	assert.Contains(t, out, connectURL)
	assert.Contains(t, out, "overlay_event")
	assert.Contains(t, out, "Overlay: Connect")
}

func TestRenderSEF_LeavesNoUnrenderedPlaceholders(t *testing.T) {
	// given
	siteName := "My Site"

	// when
	out, err := renderSEF("wss://x/y", siteName)

	// then
	require.NoError(t, err)
	assert.NotContains(t, out, "{{")
	assert.Contains(t, out, "My Site Overlay")
}
