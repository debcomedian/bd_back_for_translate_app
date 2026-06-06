package lexicon

import (
	"math"
	"strings"
	"time"
)

const (
	minHalfLifeDays          = 0.020
	maxHalfLifeDays          = 365.000
	masteredHalfLifeDays     = 90.000
	defaultResponseTimeMS    = 4500
	defaultDifficultyForHLR  = 2.500
	minimumNextIntervalMins  = 15
	maximumNextIntervalHours = 24 * 120
)

type HalfLifeInput struct {
	PreviousHalfLifeDays float64
	PreviousMasteryScore float64
	LastSeenAt           *time.Time
	AttemptedAt          time.Time
	Result               string
	ResponseTimeMS       int
	Difficulty           float64
	RepeatCount          int
}

type HalfLifeUpdate struct {
	HalfLifeDays      float64   `json:"half_life_days"`
	RecallProbability float64   `json:"recall_probability"`
	MasteryScore      float64   `json:"mastery_score"`
	NextDue           time.Time `json:"next_due"`
	IntervalMinutes   int       `json:"interval_minutes"`
	Difficulty        float64   `json:"difficulty"`
	Result            string    `json:"result"`
}

func CalculateHalfLifeUpdate(input HalfLifeInput) HalfLifeUpdate {
	attemptedAt := input.AttemptedAt
	if attemptedAt.IsZero() {
		attemptedAt = time.Now()
	}

	difficulty := input.Difficulty
	if difficulty <= 0 {
		difficulty = defaultDifficultyForHLR
	}
	difficulty = clamp(difficulty, 0.050, 5.999)

	previousHalfLife := input.PreviousHalfLifeDays
	if previousHalfLife <= 0 {
		previousHalfLife = InitialHalfLifeDays(difficulty)
	}
	previousHalfLife = clamp(previousHalfLife, minHalfLifeDays, maxHalfLifeDays)

	elapsedDays := 0.0
	if input.LastSeenAt != nil && !input.LastSeenAt.IsZero() && attemptedAt.After(*input.LastSeenAt) {
		elapsedDays = attemptedAt.Sub(*input.LastSeenAt).Hours() / 24.0
	}
	predictedRecall := math.Pow(2, -elapsedDays/previousHalfLife)
	predictedRecall = clamp(predictedRecall, 0, 1)

	result := normalizeAttemptResult(input.Result)
	responseTime := input.ResponseTimeMS
	if responseTime <= 0 {
		responseTime = defaultResponseTimeMS
	}
	speed := responseSpeedMultiplier(responseTime)
	difficultyBrake := 1.0 / (1.0 + difficulty*0.08)

	newHalfLife := previousHalfLife
	switch result {
	case "correct":
		newHalfLife = previousHalfLife * (1.0 + 0.45*difficultyBrake*speed)
	case "partial":
		newHalfLife = previousHalfLife * (1.0 + 0.16*difficultyBrake*speed)
	default:
		forgetFactor := 0.58 - difficulty*0.025
		forgetFactor = clamp(forgetFactor, 0.35, 0.62)
		newHalfLife = previousHalfLife * forgetFactor
	}
	newHalfLife = clamp(newHalfLife, minHalfLifeDays, maxHalfLifeDays)

	mastery := MasteryFromHalfLifeDays(newHalfLife)
	if result == "incorrect" && input.PreviousMasteryScore <= 0.001 && input.RepeatCount <= 1 {
		mastery = 0
	}
	if result != "incorrect" && mastery < input.PreviousMasteryScore {
		mastery = input.PreviousMasteryScore
	}
	mastery = clamp(mastery, 0, 1)

	targetRecall := targetRecallForResult(result)
	intervalDays := -newHalfLife * log2(targetRecall)
	intervalMinutes := int(math.Round(intervalDays * 24 * 60))
	if intervalMinutes < minimumNextIntervalMins {
		intervalMinutes = minimumNextIntervalMins
	}
	maxMinutes := maximumNextIntervalHours * 60
	if intervalMinutes > maxMinutes {
		intervalMinutes = maxMinutes
	}

	return HalfLifeUpdate{
		HalfLifeDays:      round3(newHalfLife),
		RecallProbability: round3(predictedRecall),
		MasteryScore:      round3(mastery),
		NextDue:           attemptedAt.Add(time.Duration(intervalMinutes) * time.Minute),
		IntervalMinutes:   intervalMinutes,
		Difficulty:        round3(difficulty),
		Result:            result,
	}
}

func InitialHalfLifeDays(difficulty float64) float64 {
	difficulty = clamp(difficulty, 0.050, 5.999)
	return round3(clamp(1.25-difficulty*0.13, 0.25, 1.25))
}

func MasteryFromHalfLifeDays(halfLifeDays float64) float64 {
	if halfLifeDays <= 0 {
		return 0
	}
	value := math.Log1p(halfLifeDays) / math.Log1p(masteredHalfLifeDays)
	return round3(clamp(value, 0, 1))
}

func normalizeAttemptResult(result string) string {
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "correct", "partial", "incorrect":
		return strings.ToLower(strings.TrimSpace(result))
	default:
		return "incorrect"
	}
}

func responseSpeedMultiplier(responseTimeMS int) float64 {
	switch {
	case responseTimeMS <= 0:
		return 1.0
	case responseTimeMS <= 2500:
		return 1.08
	case responseTimeMS >= 10000:
		return 0.92
	default:
		return 1.0
	}
}

func targetRecallForResult(result string) float64 {
	switch normalizeAttemptResult(result) {
	case "correct":
		return 0.85
	case "partial":
		return 0.90
	default:
		return 0.95
	}
}

func log2(value float64) float64 {
	return math.Log(value) / math.Ln2
}
