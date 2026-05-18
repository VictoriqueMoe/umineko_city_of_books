package slurs

import (
	"context"
	"regexp"

	"umineko_city_of_books/internal/contentfilter"
)

type Rule struct {
	patterns []*regexp.Regexp
}

func New() *Rule {
	compiled := make([]*regexp.Regexp, 0, len(rawPatterns))
	for _, p := range rawPatterns {
		compiled = append(compiled, regexp.MustCompile(`(?i)`+p))
	}
	return &Rule{patterns: compiled}
}

func (r *Rule) Name() contentfilter.RuleName {
	return contentfilter.RuleSlurs
}

func (r *Rule) Check(_ context.Context, texts []string) (*contentfilter.Rejection, error) {
	for i := 0; i < len(texts); i++ {
		normalised := contentfilter.Normalise(texts[i])
		for j := 0; j < len(r.patterns); j++ {
			if r.patterns[j].MatchString(normalised) {
				return &contentfilter.Rejection{
					Rule:   contentfilter.RuleSlurs,
					Reason: "Your message contains language that is not allowed on this site.",
				}, nil
			}
		}
	}
	return nil, nil
}
