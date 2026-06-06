package lexicon

import (
	"math"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

const difficultyMetaVersion = 2

// DifficultyInput describes the language-independent part of lexical difficulty.
// The result is placed on a CEFR-linked scale where each CEFR band occupies one unit:
// A1=[0;1), A2=[1;2), B1=[2;3), B2=[3;4), C1=[4;5), C2=[5;6).
type DifficultyInput struct {
	ZipfFrequency    float64
	FreqBucket       string
	LemmaChars       int
	TokenCount       int
	Status           string
	POS              string
	Value            string
	ExplicitCEFR     string
	SynonymCount     int
	IsTargetExercise bool
}

type DifficultyBreakdown struct {
	CefrLevel         string  `json:"cefr_level"`
	CefrFloor         float64 `json:"cefr_floor"`
	FrequencyScore    float64 `json:"frequency_score"`
	LengthScore       float64 `json:"length_score"`
	MultiwordScore    float64 `json:"multiword_score"`
	OrthographyScore  float64 `json:"orthography_score"`
	POSScore          float64 `json:"pos_score"`
	ConfidencePenalty float64 `json:"confidence_penalty"`
	SynonymRelief     float64 `json:"synonym_relief"`
	LocalComplexity   float64 `json:"local_complexity"`
	Difficulty        float64 `json:"difficulty"`
	MetaVersion       int     `json:"meta_version"`
}

func CalculateWordDifficulty(input DifficultyInput) DifficultyBreakdown {
	cefr := NormalizeCEFR(input.ExplicitCEFR)
	if cefr == "" {
		cefr = InferCEFRFromFrequency(input.ZipfFrequency, input.FreqBucket)
	}
	if cefr == "" {
		cefr = "B1"
	}

	lemmaChars := input.LemmaChars
	if lemmaChars <= 0 {
		lemmaChars = utf8.RuneCountInString(strings.TrimSpace(input.Value))
	}
	tokenCount := input.TokenCount
	if tokenCount <= 0 {
		tokenCount = countTokens(input.Value)
	}
	if tokenCount <= 0 {
		tokenCount = 1
	}

	frequencyScore := normalizeFrequencyRarity(input.ZipfFrequency, input.FreqBucket)
	lengthScore := normalizeLengthDifficulty(lemmaChars)
	multiwordScore := normalizeMultiwordDifficulty(tokenCount)
	orthographyScore := EstimateOrthographyScore(input.Value)
	posScore := normalizePOSDifficulty(input.POS)
	confidencePenalty := normalizeConfidenceDifficulty(input.Status)
	synonymRelief := NormalizeSynonymRelief(input.SynonymCount)

	local := 0.52*frequencyScore +
		0.18*lengthScore +
		0.10*multiwordScore +
		0.08*orthographyScore +
		0.06*posScore +
		0.06*confidencePenalty

	if input.IsTargetExercise {
		local += 0.06
	}
	local -= synonymRelief
	local = clamp(local, 0, 0.999)

	floor := CEFRFloor(cefr)
	difficulty := clamp(floor+local, 0.050, 5.999)

	return DifficultyBreakdown{
		CefrLevel:         cefr,
		CefrFloor:         round3(floor),
		FrequencyScore:    round3(frequencyScore),
		LengthScore:       round3(lengthScore),
		MultiwordScore:    round3(multiwordScore),
		OrthographyScore:  round3(orthographyScore),
		POSScore:          round3(posScore),
		ConfidencePenalty: round3(confidencePenalty),
		SynonymRelief:     round3(synonymRelief),
		LocalComplexity:   round3(local),
		Difficulty:        round3(difficulty),
		MetaVersion:       difficultyMetaVersion,
	}
}

func NormalizeCEFR(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "A1", "A2", "B1", "B2", "C1", "C2":
		return strings.ToUpper(strings.TrimSpace(value))
	default:
		return ""
	}
}

func CEFRFloor(cefr string) float64 {
	switch NormalizeCEFR(cefr) {
	case "A1":
		return 0
	case "A2":
		return 1
	case "B1":
		return 2
	case "B2":
		return 3
	case "C1":
		return 4
	case "C2":
		return 5
	default:
		return 2
	}
}

func CEFRFromDifficulty(difficulty float64) string {
	switch {
	case difficulty < 1:
		return "A1"
	case difficulty < 2:
		return "A2"
	case difficulty < 3:
		return "B1"
	case difficulty < 4:
		return "B2"
	case difficulty < 5:
		return "C1"
	default:
		return "C2"
	}
}

func InferCEFRFromFrequency(zipf float64, bucket string) string {
	bucket = strings.TrimSpace(bucket)
	switch bucket {
	case "core_high":
		return "A1"
	case "core_mid":
		return "A2"
	case "core_low":
		return "B1"
	case "tail":
		return "B2"
	}

	switch {
	case zipf >= 5.2:
		return "A1"
	case zipf >= 4.4:
		return "A2"
	case zipf >= 3.7:
		return "B1"
	case zipf >= 3.0:
		return "B2"
	case zipf >= 2.3:
		return "C1"
	case zipf > 0:
		return "C2"
	default:
		return "B1"
	}
}

func normalizeFrequencyRarity(zipf float64, bucket string) float64 {
	if zipf > 0 {
		return clamp((6.5-zipf)/5.5, 0, 1)
	}
	switch strings.TrimSpace(bucket) {
	case "core_high":
		return 0.10
	case "core_mid":
		return 0.32
	case "core_low":
		return 0.55
	case "tail":
		return 0.78
	default:
		return 0.62
	}
}

func normalizeLengthDifficulty(chars int) float64 {
	if chars <= 0 {
		return 0.40
	}
	return clamp(float64(chars-3)/14.0, 0, 1)
}

func normalizeMultiwordDifficulty(tokenCount int) float64 {
	if tokenCount <= 1 {
		return 0
	}
	return clamp(float64(tokenCount-1)/3.0, 0, 1)
}

func normalizePOSDifficulty(pos string) float64 {
	switch strings.ToLower(strings.TrimSpace(pos)) {
	case "noun", "pron":
		return 0.20
	case "adj", "adjective":
		return 0.35
	case "adv", "adverb":
		return 0.45
	case "verb":
		return 0.55
	case "prep", "preposition", "conj", "conjunction", "det":
		return 0.25
	default:
		return 0.40
	}
}

func seedConfidencePenalty(status string) float64 {
	switch strings.TrimSpace(status) {
	case "strict_validated_all":
		return 0.00
	case "soft_validated_all":
		return 0.15
	case "soft_completed_all":
		return 0.25
	case "half_validated":
		return 0.45
	default:
		return 0.70
	}
}

func normalizeConfidenceDifficulty(status string) float64 {
	penalty := seedConfidencePenalty(status)
	return clamp(penalty/0.70, 0, 1)
}

func EstimateOrthographyScore(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0.40
	}

	var letters, nonLetters, latin, cyrillic, otherScripts, marks int
	for _, r := range value {
		switch {
		case unicode.IsLetter(r):
			letters++
			if unicode.In(r, unicode.Latin) {
				latin++
			} else if unicode.In(r, unicode.Cyrillic) {
				cyrillic++
			} else {
				otherScripts++
			}
		case unicode.IsMark(r):
			marks++
		case unicode.IsSpace(r) || r == '-' || r == '\'':
		default:
			nonLetters++
		}
	}
	if letters == 0 {
		return 0.50
	}

	score := 0.0
	if latin > 0 && cyrillic > 0 {
		score += 0.35
	}
	if otherScripts > 0 {
		score += 0.25
	}
	score += clamp(float64(marks)/float64(letters), 0, 0.20)
	score += clamp(float64(nonLetters)/float64(letters), 0, 0.20)
	return clamp(score, 0, 1)
}

func NormalizeSynonymRelief(count int) float64 {
	if count <= 1 {
		return 0
	}
	return clamp(float64(count-1)*0.025, 0, 0.10)
}

func DirectionDifficultyBias(sourceLang, targetLang string) float64 {
	nativeLang := NormalizeValue(strings.TrimSpace(getNativeLangCode()))
	if nativeLang == "" {
		nativeLang = "ru"
	}
	sourceLang = NormalizeValue(sourceLang)
	targetLang = NormalizeValue(targetLang)

	switch {
	case targetLang == nativeLang:
		return -0.120
	case sourceLang == nativeLang && targetLang != nativeLang:
		return 0.180
	case sourceLang != nativeLang && targetLang != nativeLang:
		return 0.240
	default:
		return 0
	}
}

func BlendDirectionDifficulty(concept, source, target, bias, synonymRelief float64) float64 {
	value := concept*0.55 + target*0.30 + source*0.15 + bias - synonymRelief
	return round3(clamp(value, 0.050, 5.999))
}

func clamp(value, minValue, maxValue float64) float64 {
	return math.Min(math.Max(value, minValue), maxValue)
}

func getNativeLangCode() string {
	return strings.TrimSpace(getenvOrDefault("NATIVE_LANG_CODE", "ru"))
}

func getenvOrDefault(name string, fallback string) string {
	// Kept as a tiny wrapper to make formula tests deterministic through explicit inputs
	// and keep environment access isolated.
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
