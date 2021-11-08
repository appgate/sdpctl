package internal

import (
	"os"
	"sort"
)

// Getenv returns environment variable value, if it does not exist, return fallback
func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func AppendIfMissing(slice []int, i int) []int {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func InSlice(needle string, haystack []string) bool {
	sort.Strings(haystack)
	i := sort.Search(len(haystack),
		func(i int) bool { return haystack[i] >= needle })
	if i < len(haystack) && haystack[i] == needle {
		return true
	}
	return false
}
