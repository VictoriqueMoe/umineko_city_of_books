package search

import (
	"fmt"

	"umineko_city_of_books/internal/repository"
)

func init() {
	for _, t := range AllEntityTypes() {
		if _, ok := urlBuilders[t]; !ok {
			panic(fmt.Sprintf("search.urlBuilders: missing URL builder for entity type %q (registered in repository.searchSources but not in internal/search/urls.go)", t))
		}
	}
}

var urlBuilders = map[repository.SearchEntityType]func(repository.SearchResult) string{
	repository.SearchEntityTheory:              selfURL("/theory/"),
	repository.SearchEntityResponse:            parentURL("/theory/", "#response-"),
	repository.SearchEntityPost:                selfURL("/game-board/"),
	repository.SearchEntityPostComment:         parentURL("/game-board/", "#comment-"),
	repository.SearchEntityArt:                 selfURL("/gallery/art/"),
	repository.SearchEntityArtComment:          parentURL("/gallery/art/", "#comment-"),
	repository.SearchEntityMystery:             selfURL("/mystery/"),
	repository.SearchEntityMysteryAttempt:      parentURL("/mystery/", "#attempt-"),
	repository.SearchEntityMysteryComment:      parentURL("/mystery/", "#comment-"),
	repository.SearchEntityShip:                selfURL("/ships/"),
	repository.SearchEntityShipComment:         parentURL("/ships/", "#comment-"),
	repository.SearchEntityAnnouncement:        selfURL("/announcements/"),
	repository.SearchEntityAnnouncementComment: parentURL("/announcements/", "#comment-"),
	repository.SearchEntityFanfic:              selfURL("/fanfiction/"),
	repository.SearchEntityFanficComment:       parentURL("/fanfiction/", "#comment-"),
	repository.SearchEntityJournal:             selfURL("/journals/"),
	repository.SearchEntityJournalComment:      parentURL("/journals/", "#comment-"),
	repository.SearchEntityUser: func(r repository.SearchResult) string {
		return "/user/" + r.AuthorUsername
	},
}

func selfURL(prefix string) func(repository.SearchResult) string {
	return func(r repository.SearchResult) string {
		return prefix + r.ID
	}
}

func parentURL(prefix, suffix string) func(repository.SearchResult) string {
	return func(r repository.SearchResult) string {
		if r.ParentID == nil {
			return ""
		}
		return prefix + *r.ParentID + suffix + r.ID
	}
}

func BuildURL(r repository.SearchResult) string {
	if fn, ok := urlBuilders[r.EntityType]; ok {
		return fn(r)
	}
	return ""
}
