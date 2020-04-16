package util

import (
	"fmt"
	"strings"
)

func MapFiltersToFilter(filters []string) string {
	if len(filters) > 0 {
		return strings.Join(filters[:], " ")
	} else {
		return ""
	}
}

func FilterByName(name string) string {
	return fmt.Sprintf("displayName eq '%s'", name)
}
