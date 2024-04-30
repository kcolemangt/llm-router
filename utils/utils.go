package utils

import (
	"strings"
	"unicode"
)

// RedactAuthorization redacts sensitive information from authorization keys.
func RedactAuthorization(auth string) string {
	if strings.HasPrefix(auth, "Bearer ") && len(auth) > 29 {
		// Display the first 3 characters, ellipses, and the last 4 characters
		return auth[:10] + "..." + auth[len(auth)-4:]
	}
	// Replace each non-whitespace character with '*'
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return r
		}
		return '*'
	}, auth)
}
