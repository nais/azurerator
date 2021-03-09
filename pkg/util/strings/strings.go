package strings

func RemoveDuplicates(list []string) []string {
	seen := make(map[string]struct{}, 0)
	filtered := make([]string, 0)

	for _, key := range list {
		if _, found := seen[key]; !found {
			seen[key] = struct{}{}
			filtered = append(filtered, key)
		}
	}
	return filtered
}
