// internal/utils.go

package internal

// SummarizeToLength trims the text to the specified maximum length.
func SummarizeToLength(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength]
}
