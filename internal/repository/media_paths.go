package repository

func dedupePaths(paths []string) []string {
	if len(paths) < 2 {
		return paths
	}

	seen := make(map[string]bool, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if seen[path] {
			continue
		}

		seen[path] = true
		result = append(result, path)
	}

	return result
}
