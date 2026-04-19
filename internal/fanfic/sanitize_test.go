package fanfic

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeBody_StripsScriptTag(t *testing.T) {
	out := sanitizeBody(`<p>hi</p><script>alert(1)</script>`)
	assert.NotContains(t, out, "<script>")
	assert.NotContains(t, out, "alert(1)")
	assert.Contains(t, out, "<p>hi</p>")
}

func TestSanitizeBody_StripsEventHandlers(t *testing.T) {
	out := sanitizeBody(`<img src="x" onerror="fetch('/steal?c='+document.cookie)">`)
	assert.NotContains(t, out, "onerror")
	assert.NotContains(t, out, "fetch")
}

func TestSanitizeBody_StripsJavaScriptURL(t *testing.T) {
	out := sanitizeBody(`<a href="javascript:alert(1)">click</a>`)
	assert.NotContains(t, out, "javascript:")
}

func TestSanitizeBody_StripsIframe(t *testing.T) {
	out := sanitizeBody(`<iframe src="https://evil.example"></iframe>`)
	assert.NotContains(t, out, "<iframe")
}

func TestSanitizeBody_StripsSvgScript(t *testing.T) {
	out := sanitizeBody(`<svg><script>alert(1)</script></svg>`)
	assert.NotContains(t, out, "alert(1)")
}

func TestSanitizeBody_AllowsFormattingTags(t *testing.T) {
	raw := `<p class="verse">Now then, <strong>let us begin</strong> <em>the game</em>.</p>` +
		`<ul><li>one</li><li>two</li></ul>` +
		`<blockquote>mama said</blockquote>` +
		`<h2>Act One</h2>`
	out := sanitizeBody(raw)
	for _, s := range []string{"<p", "<strong>", "<em>", "<ul>", "<li>", "<blockquote>", "<h2>", "class=\"verse\""} {
		assert.Contains(t, out, s, "expected %q to survive sanitise", s)
	}
}

func TestSanitizeBody_AllowsSafeLinks(t *testing.T) {
	out := sanitizeBody(`<a href="https://07th-expansion.net/">link</a>`)
	assert.Contains(t, out, `href="https://07th-expansion.net/"`)
	assert.True(t, strings.Contains(out, `rel="`), "should attach nofollow rel")
}

func TestSanitizeBody_PreservesTiptapStrike(t *testing.T) {
	out := sanitizeBody(`<p><s>gone</s></p>`)
	assert.Contains(t, out, "<s>gone</s>")
}

func TestSanitizeBody_PreservesOrderedList(t *testing.T) {
	out := sanitizeBody(`<ol><li>one</li><li>two</li></ol>`)
	assert.Contains(t, out, "<ol>")
	assert.Contains(t, out, "<li>one</li>")
}

func TestSanitizeBody_PreservesHorizontalRule(t *testing.T) {
	out := sanitizeBody(`<p>before</p><hr><p>after</p>`)
	assert.Contains(t, out, "<hr")
}

func TestSanitizeBody_PreservesTextAlignOnParagraph(t *testing.T) {
	out := sanitizeBody(`<p style="text-align: center">centred</p>`)
	assert.Contains(t, out, "text-align")
	assert.Contains(t, out, "center")
	assert.Contains(t, out, "centred")
}

func TestSanitizeBody_PreservesTextAlignOnHeading(t *testing.T) {
	out := sanitizeBody(`<h2 style="text-align: right">right heading</h2>`)
	assert.Contains(t, out, "text-align")
	assert.Contains(t, out, "right")
	assert.Contains(t, out, "<h2")
}

func TestSanitizeBody_RejectsInvalidTextAlign(t *testing.T) {
	out := sanitizeBody(`<p style="text-align: expression(alert(1))">x</p>`)
	assert.NotContains(t, out, "expression")
	assert.NotContains(t, out, "alert")
}

func TestSanitizeBody_PreservesTiptapColourSpan(t *testing.T) {
	out := sanitizeBody(`<p>uu <span style="color: #e53935">red</span> truth</p>`)
	assert.Contains(t, out, `color: #e53935`)
	assert.Contains(t, out, "<span")
	assert.Contains(t, out, "red")
}

func TestSanitizeBody_RejectsNonHexColour(t *testing.T) {
	out := sanitizeBody(`<span style="color: url(javascript:alert(1))">x</span>`)
	assert.NotContains(t, out, "url(")
	assert.NotContains(t, out, "javascript")
}

func TestSanitizeBody_PreservesFullTiptapRender(t *testing.T) {
	raw := `<h2 style="text-align: center">Act One</h2>` +
		`<p style="text-align: left"><strong>Bold</strong> and <em>italic</em> and <s>strike</s>.</p>` +
		`<p>A <span style="color: #ab47bc">purple</span> truth.</p>` +
		`<blockquote><p>Without love, it cannot be seen.</p></blockquote>` +
		`<ul><li>first</li><li>second</li></ul>` +
		`<ol><li>uno</li><li>dos</li></ol>` +
		`<hr>` +
		`<p><a href="https://example.com" rel="noopener noreferrer nofollow" target="_blank">link</a></p>`
	out := sanitizeBody(raw)
	for _, s := range []string{
		"<h2", "text-align", "center", "Act One",
		"<strong>Bold</strong>", "<em>italic</em>", "<s>strike</s>",
		"color: #ab47bc", "purple",
		"<blockquote>", "<ul>", "<ol>", "<hr",
		`href="https://example.com"`,
	} {
		assert.Contains(t, out, s, "expected %q to survive sanitise", s)
	}
}
