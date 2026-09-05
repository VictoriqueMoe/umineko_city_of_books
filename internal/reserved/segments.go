package reserved

import "strings"

var pathSegments = map[string]bool{
	"admin":           true,
	"announcements":   true,
	"api":             true,
	"chat":            true,
	"debug":           true,
	"fanfiction":      true,
	"forgot-password": true,
	"game-board":      true,
	"gallery":         true,
	"games":           true,
	"health":          true,
	"hls":             true,
	"journals":        true,
	"live":            true,
	"livez":           true,
	"login":           true,
	"metrics":         true,
	"mysteries":       true,
	"mystery":         true,
	"notifications":   true,
	"oc":              true,
	"og-image":        true,
	"quotes":          true,
	"reset-password":  true,
	"robots":          true,
	"rooms":           true,
	"rules":           true,
	"search":          true,
	"secrets":         true,
	"set-email":       true,
	"settings":        true,
	"ships":           true,
	"sitemap":         true,
	"suggestions":     true,
	"theories":        true,
	"theory":          true,
	"uploads":         true,
	"user":            true,
	"users":           true,
	"verify-email":    true,
	"watch":           true,
	"welcome":         true,
}

func IsPathSegment(segment string) bool {
	return pathSegments[strings.ToLower(segment)]
}
