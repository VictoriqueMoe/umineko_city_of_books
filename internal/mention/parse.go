package mention

import "regexp"

const MaxMatches = 20

var Pattern = regexp.MustCompile(`\B@([a-zA-Z0-9_]+)`)

func Usernames(body string) []string {
	matches := Pattern.FindAllStringSubmatch(body, MaxMatches)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if _, dup := seen[m[1]]; dup {
			continue
		}

		seen[m[1]] = struct{}{}
		out = append(out, m[1])
	}

	return out
}
