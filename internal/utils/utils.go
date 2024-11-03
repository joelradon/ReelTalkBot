// internal/utils/utils.go

package utils

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

// DetermineCategories determines categories based on keywords.
func DetermineCategories(keywords []string) string {
	categoryMap := map[string][]string{
		"Timing":                          {"timing", "season", "best time", "peak season"},
		"Gear Selection":                  {"gear", "equipment", "rod", "reel", "line"},
		"Bait/Lures/Fly Selection":        {"bait", "lures", "fly selection", "fly patterns"},
		"Reading Water":                   {"reading water", "water conditions", "pools", "seams"},
		"Presenting Bait/Lure/Fly":        {"presentation", "drift", "swing", "dead drift"},
		"Handling the Strike or Fight":    {"handling strike", "fighting fish", "hook set"},
		"Casting/Presentation":            {"casting", "presentation", "mending"},
		"Fish Handling/Catch and Release": {"handling fish", "catch and release", "revive"},
	}

	determinedCategories := make(map[string]struct{})

	for _, kw := range keywords {
		for category, kws := range categoryMap {
			for _, ckw := range kws {
				if kw == ckw {
					determinedCategories[category] = struct{}{}
				}
			}
		}
	}

	if len(determinedCategories) == 0 {
		return "Uncategorized"
	}

	var categories []string
	for category := range determinedCategories {
		categories = append(categories, category)
	}

	return strings.Join(categories, ", ")
}

// IdentifyTaxonomyCategories parses the user query to extract taxonomy categories.
// This function can be further enhanced based on specific taxonomy requirements.
func IdentifyTaxonomyCategories(query string) (bodyOfWater, fishSpecies, waterType, category string) {
	lowerQuery := strings.ToLower(query)

	bodyOfWaterKeywords := []string{"salmon river", "lake ontario", "hoh river", "chesapeake bay", "great lake tributaries"}
	fishSpeciesKeywords := []string{"steelhead", "blue crab", "striped bass", "king salmon", "coho salmon", "brown trout", "eastern menhaden", "spot", "croaker", "black drum", "atlantic sturgeon"}
	waterTypeKeywords := []string{"adronomous", "lentic", "lotic"}
	categoryKeywords := map[string][]string{
		"Timing":                          {"timing", "season", "best time", "peak season"},
		"Gear Selection":                  {"gear", "equipment", "rod", "reel", "line"},
		"Bait/Lures/Fly Selection":        {"bait", "lures", "fly selection", "fly patterns"},
		"Reading Water":                   {"reading water", "water conditions", "pools", "seams"},
		"Presenting Bait/Lure/Fly":        {"presentation", "drift", "swing", "dead drift"},
		"Handling the Strike or Fight":    {"handling strike", "fighting fish", "hook set"},
		"Casting/Presentation":            {"casting", "presentation", "mending"},
		"Fish Handling/Catch and Release": {"handling fish", "catch and release", "revive"},
	}

	// Identify BodyOfWater
	for _, kw := range bodyOfWaterKeywords {
		if strings.Contains(lowerQuery, kw) {
			bodyOfWater = kw
			break
		}
	}

	// Identify FishSpecies
	for _, kw := range fishSpeciesKeywords {
		if strings.Contains(lowerQuery, kw) {
			fishSpecies = kw
			break
		}
	}

	// Identify WaterType
	for _, kw := range waterTypeKeywords {
		if strings.Contains(lowerQuery, kw) {
			waterType = kw
			break
		}
	}

	// Identify Category
	for cat, kws := range categoryKeywords {
		for _, kw := range kws {
			if strings.Contains(lowerQuery, kw) {
				category = cat
				break
			}
		}
		if category != "" {
			break
		}
	}

	return
}
