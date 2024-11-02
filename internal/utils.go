// internal/utils.go

package internal

import (
	"strings"
)

// SummarizeToLength trims the text to the specified maximum length.
func SummarizeToLength(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength]
}

// ExtractKeywords extracts keywords from the input text.
// This is a simple implementation and can be enhanced.
func ExtractKeywords(text string) []string {
	words := strings.Fields(text)
	keywordSet := make(map[string]struct{})
	for _, word := range words {
		cleanedWord := strings.ToLower(strings.Trim(word, ".,!?\"'"))
		if len(cleanedWord) > 3 { // Simple filter: words longer than 3 characters
			keywordSet[cleanedWord] = struct{}{}
		}
	}

	var keywords []string
	for word := range keywordSet {
		keywords = append(keywords, word)
	}
	return keywords
}
