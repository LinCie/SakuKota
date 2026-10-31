// Package text holds helpers for normalizing free-text input.
package text

import (
	"strings"
	"unicode/utf8"
)

// TrimOptional trims surrounding whitespace from text. It returns nil when text
// is nil or blank, so a blank optional value is stored as absent.
func TrimOptional(text *string) *string {
	if text == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*text)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

// TrimOptionalMax is TrimOptional that also reports whether the trimmed text
// has at most limit characters. Blank text is within any limit.
func TrimOptionalMax(text *string, limit int) (*string, bool) {
	trimmed := TrimOptional(text)
	if trimmed == nil {
		return nil, true
	}

	return trimmed, utf8.RuneCountInString(*trimmed) <= limit
}
