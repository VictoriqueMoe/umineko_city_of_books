package newaccountlinks

import (
	"context"
	"regexp"
	"slices"

	"umineko_city_of_books/internal/authctx"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/contentfilter"
)

var (
	linkPattern = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+`)
)

type Rule struct {
	authzSvc authz.Service
}

func New(authzSvc authz.Service) *Rule {
	return &Rule{authzSvc: authzSvc}
}

func (r *Rule) Name() contentfilter.RuleName {
	return contentfilter.RuleNewAccountLinks
}

func (r *Rule) Check(ctx context.Context, texts []string) (*contentfilter.Rejection, error) {
	if !containsLink(texts) {
		return nil, nil
	}

	userID, ok := authctx.UserID(ctx)
	if !ok {
		return nil, nil
	}

	if !r.authzSvc.IsRestrictedNewAccount(ctx, userID) {
		return nil, nil
	}

	return &contentfilter.Rejection{
		Rule:   contentfilter.RuleNewAccountLinks,
		Reason: "New accounts cannot post links. Please try again once your account is a day old.",
	}, nil
}

func containsLink(texts []string) bool {
	return slices.ContainsFunc(texts, linkPattern.MatchString)
}
