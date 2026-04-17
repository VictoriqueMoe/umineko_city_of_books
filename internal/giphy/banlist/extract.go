package banlist

import (
	"regexp"
	"strings"
)

var (
	gifsPathRe    = regexp.MustCompile(`(?i)giphy\.com/gifs/(?:[a-zA-Z0-9_-]+-)?([a-zA-Z0-9]+)`)
	mediaPathRe   = regexp.MustCompile(`(?i)media\d*\.giphy\.com/media(?:/v\d+\.[A-Za-z0-9_\-]+)?/([a-zA-Z0-9]+)`)
	iPathRe       = regexp.MustCompile(`(?i)i\.giphy\.com/([a-zA-Z0-9]+)`)
	channelPathRe = regexp.MustCompile(`(?i)giphy\.com/channel/([a-zA-Z0-9_-]+)`)
	profilePathRe = regexp.MustCompile(`(?i)(?:^|[^a-zA-Z0-9.-])giphy\.com/([a-zA-Z][a-zA-Z0-9_-]*)`)
	rawIDRe       = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

var reservedProfileSegments = map[string]bool{
	"gifs": true, "stickers": true, "search": true, "explore": true,
	"channel": true, "trending": true, "tags": true, "clips": true,
	"login": true, "signup": true, "create": true, "upload": true,
	"embed": true, "about": true, "privacy": true, "terms": true,
	"api": true, "dashboard": true, "apps": true, "artists": true,
	"static": true, "media": true,
}

func FindGifIDs(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, re := range []*regexp.Regexp{gifsPathRe, mediaPathRe, iPathRe} {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			id := m[1]
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	return out
}

func FindUsers(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range channelPathRe.FindAllStringSubmatch(text, -1) {
		k := strings.ToLower(m[1])
		if !seen[k] {
			seen[k] = true
			out = append(out, m[1])
		}
	}
	for _, m := range profilePathRe.FindAllStringSubmatch(text, -1) {
		name := m[1]
		if reservedProfileSegments[strings.ToLower(name)] {
			continue
		}
		k := strings.ToLower(name)
		if !seen[k] {
			seen[k] = true
			out = append(out, name)
		}
	}
	return out
}

func ParseInput(raw string) (Kind, string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}
	if users := FindUsers(raw); len(users) > 0 {
		return KindUser, users[0], true
	}
	if gifs := FindGifIDs(raw); len(gifs) > 0 {
		return KindGif, gifs[0], true
	}
	if rawIDRe.MatchString(raw) {
		return KindGif, raw, true
	}
	return "", "", false
}
