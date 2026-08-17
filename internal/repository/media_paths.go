package repository

func dedupePaths(paths []string) []string {
	if len(paths) < 2 {
		return paths
	}

	seen := make(map[string]bool, len(paths))
	result := make([]string, 0, len(paths))
	for i := range paths {
		if seen[paths[i]] {
			continue
		}

		seen[paths[i]] = true
		result = append(result, paths[i])
	}

	return result
}
