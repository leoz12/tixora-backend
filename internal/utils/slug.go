package utils

import (
	"regexp"
	"strings"
)

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a string into a lowercase, hyphen-separated URL slug.
func Slugify(s string) string {
	slug := slugInvalidChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-")
	return strings.Trim(slug, "-")
}
