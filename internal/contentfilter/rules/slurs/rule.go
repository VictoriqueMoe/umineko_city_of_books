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
	for _, text := range texts {
		for _, re := range r.patterns {
			if re.MatchString(text) {
				return &contentfilter.Rejection{
					Rule:   contentfilter.RuleSlurs,
					Reason: "Your message contains language that is not allowed on this site.",
				}, nil
			}
		}
	}
	return nil, nil
}
