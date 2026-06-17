package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotifEmail_RendersSubjectAndSiteName(t *testing.T) {
	// given
	siteName := "Custom Site"

	// when
	subject, body := notifEmail("Beatrice", "commented on your post", "", "https://example.com/post", siteName)

	// then
	assert.Equal(t, "Beatrice commented on your post", subject)
	assert.Contains(t, body, siteName)
	assert.NotContains(t, body, "Umineko City of Books")
}

func TestReportEmail_RendersSiteName(t *testing.T) {
	// given
	siteName := "Custom Site"

	// when
	_, body := reportEmail("Battler", "post", "spam", "https://example.com/reports", siteName)

	// then
	assert.Contains(t, body, siteName)
	assert.NotContains(t, body, "Umineko City of Books")
}
