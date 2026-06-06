package lexicon

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

func ensureNonZeroID(id uint64, name string) error {
	if id == 0 {
		return fmt.Errorf("%s не указан", name)
	}
	return nil
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func NormalizeValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)
	return strings.Join(strings.Fields(value), " ")
}

func strPtrIfNotEmpty(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func countTokens(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	return len(strings.Fields(value))
}

func maxFormLength(forms []LexicalForm) int {
	maxLength := 0
	for _, form := range forms {
		length := utf8.RuneCountInString(form.Value)
		if length > maxLength {
			maxLength = length
		}
	}
	return maxLength
}

func fallbackFormScore(form LexicalForm) float64 {
	length := utf8.RuneCountInString(form.Value)
	tokens := countTokens(form.Value)
	return CalculateWordDifficulty(DifficultyInput{
		Value:      form.Value,
		LemmaChars: length,
		TokenCount: tokens,
	}).Difficulty
}

func fallbackConceptDifficulty(forms []LexicalForm) float64 {
	if len(forms) == 0 {
		return 1
	}

	maxScore := 0.0
	for _, form := range forms {
		score := fallbackFormScore(form)
		if score > maxScore {
			maxScore = score
		}
	}

	if maxScore < 0.1 {
		maxScore = 0.1
	}
	return round3(maxScore)
}
