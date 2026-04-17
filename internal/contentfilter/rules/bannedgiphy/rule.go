package bannedgiphy

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/giphy/banlist"
)

type Banlist interface {
	ContainsGif(id string) bool
	ContainsUser(username string) bool
}

type Lookup interface {
	UserForGif(ctx context.Context, gifID string) (string, bool)
}

type Rule struct {
	banlist Banlist
	lookup  Lookup
}

func New(b Banlist, lookup Lookup) *Rule {
	return &Rule{banlist: b, lookup: lookup}
}

func (r *Rule) Name() contentfilter.RuleName {
	return contentfilter.RuleBannedGiphy
}

func (r *Rule) Check(ctx context.Context, texts []string) (*contentfilter.Rejection, error) {
	for _, text := range texts {
		for _, id := range banlist.FindGifIDs(text) {
			if r.banlist.ContainsGif(id) {
				return &contentfilter.Rejection{
					Rule:   contentfilter.RuleBannedGiphy,
					Reason: fmt.Sprintf("This GIF has been banned by a moderator (%s).", id),
					Detail: id,
				}, nil
			}
			if r.lookup != nil {
				if user, ok := r.lookup.UserForGif(ctx, id); ok && r.banlist.ContainsUser(user) {
					return &contentfilter.Rejection{
						Rule:   contentfilter.RuleBannedGiphy,
						Reason: fmt.Sprintf("GIFs from the Giphy channel \"%s\" have been banned by a moderator.", user),
						Detail: user,
					}, nil
				}
			}
		}
		for _, user := range banlist.FindUsers(text) {
			if r.banlist.ContainsUser(user) {
				return &contentfilter.Rejection{
					Rule:   contentfilter.RuleBannedGiphy,
					Reason: fmt.Sprintf("GIFs from the Giphy channel \"%s\" have been banned by a moderator.", user),
					Detail: user,
				}, nil
			}
		}
	}
	return nil, nil
}

var _ Banlist = (banlist.Service)(nil)
