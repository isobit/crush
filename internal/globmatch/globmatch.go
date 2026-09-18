// Package globmatch provides the shared identifier glob semantics used by
// configuration allowlists and denylists.
package globmatch

import (
	"fmt"
	"path"
)

// Match reports whether pattern matches value using path.Match semantics.
// Invalid patterns never match.
func Match(pattern, value string) bool {
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}

// Any reports whether any pattern matches value.
func Any(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if Match(pattern, value) {
			return true
		}
	}
	return false
}

// Validate checks that every pattern is syntactically valid.
func Validate(field string, patterns []string) error {
	for _, pattern := range patterns {
		if _, err := path.Match(pattern, ""); err != nil {
			return fmt.Errorf("%s contains invalid glob %q: %w", field, pattern, err)
		}
	}
	return nil
}
