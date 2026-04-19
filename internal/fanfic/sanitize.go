package fanfic

import (
	"regexp"
	"sync"

	"github.com/microcosm-cc/bluemonday"
)

var (
	bodyPolicyOnce sync.Once
	bodyPolicy     *bluemonday.Policy
	colourRegex    = regexp.MustCompile(`(?i)^#[0-9a-f]{3}([0-9a-f]{3})?$`)
)

func htmlPolicy() *bluemonday.Policy {
	bodyPolicyOnce.Do(func() {
		p := bluemonday.UGCPolicy()
		p.AllowAttrs("class").OnElements("p", "span", "blockquote", "code", "pre", "div", "h2", "h3", "ul", "ol", "li", "a")
		p.AllowStyles("text-align").MatchingEnum("left", "center", "right", "justify").OnElements("p", "h2", "h3")
		p.AllowStyles("color").Matching(colourRegex).OnElements("span", "p", "h2", "h3")
		bodyPolicy = p
	})
	return bodyPolicy
}

func sanitizeBody(html string) string {
	return htmlPolicy().Sanitize(html)
}
