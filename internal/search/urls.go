package search

import (
	"fmt"
	"net/url"

	"umineko_city_of_books/internal/model"
)

func init() {
	for _, t := range AllEntityTypes() {
		if _, ok := urlBuilders[t]; !ok {
			panic(fmt.Sprintf("search.urlBuilders: missing URL builder for entity type %q (registered in model.searchSources but not in internal/search/urls.go)", t))
		}
	}
}

var urlBuilders = map[model.SearchEntityType]func(model.SearchResult) string{
	model.SearchEntityTheory:              selfURL("/theory/"),
	model.SearchEntityResponse:            parentURL("/theory/", "#response-"),
	model.SearchEntityPost:                selfURL("/game-board/"),
	model.SearchEntityPostComment:         parentURL("/game-board/", "#comment-"),
	model.SearchEntityArt:                 selfURL("/gallery/art/"),
	model.SearchEntityArtComment:          parentURL("/gallery/art/", "#comment-"),
	model.SearchEntityMystery:             selfURL("/mystery/"),
	model.SearchEntityMysteryAttempt:      parentURL("/mystery/", "#attempt-"),
	model.SearchEntityMysteryComment:      parentURL("/mystery/", "#comment-"),
	model.SearchEntityShip:                selfURL("/ships/"),
	model.SearchEntityShipComment:         parentURL("/ships/", "#comment-"),
	model.SearchEntityOC:                  selfURL("/oc/"),
	model.SearchEntityOCComment:           parentURL("/oc/", "#comment-"),
	model.SearchEntityAnnouncement:        selfURL("/announcements/"),
	model.SearchEntityAnnouncementComment: parentURL("/announcements/", "#comment-"),
	model.SearchEntityFanfic:              selfURL("/fanfiction/"),
	model.SearchEntityFanficComment:       parentURL("/fanfiction/", "#comment-"),
	model.SearchEntityJournal:             selfURL("/journals/"),
	model.SearchEntityJournalEntry:        parentURL("/journals/", "/entry/"),
	model.SearchEntityJournalComment:      parentURL("/journals/", "#comment-"),
	model.SearchEntityUser: func(r model.SearchResult) string {
		return "/user/" + r.AuthorUsername
	},
	model.SearchEntityChatMessage: func(r model.SearchResult) string {
		if r.ParentID == nil {
			return ""
		}
		u := "/rooms/" + *r.ParentID
		if r.CreatedAt != "" {
			u += "?at=" + url.QueryEscape(r.CreatedAt)
		}
		return u + "#msg-" + r.ID
	},
	model.SearchEntityLiveStream: func(r model.SearchResult) string {
		if r.AuthorUsername == "" {
			return ""
		}
		return "/" + r.AuthorUsername + "/live"
	},
}

func selfURL(prefix string) func(model.SearchResult) string {
	return func(r model.SearchResult) string {
		return prefix + r.ID
	}
}

func parentURL(prefix, suffix string) func(model.SearchResult) string {
	return func(r model.SearchResult) string {
		if r.ParentID == nil {
			return ""
		}
		return prefix + *r.ParentID + suffix + r.ID
	}
}

func BuildURL(r model.SearchResult) string {
	if fn, ok := urlBuilders[r.EntityType]; ok {
		return fn(r)
	}
	return ""
}
