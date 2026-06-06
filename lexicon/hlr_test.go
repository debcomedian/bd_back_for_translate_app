package lexicon

import (
	"math"
	"testing"
	"time"
)

func TestHalfLifeCorrectAnswerIncreasesHalfLife(t *testing.T) {
	at := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	got := CalculateHalfLifeUpdate(HalfLifeInput{
		PreviousHalfLifeDays: 1.0,
		PreviousMasteryScore: 0.1,
		AttemptedAt:          at,
		Result:               "correct",
		ResponseTimeMS:       1800,
		Difficulty:           1.2,
		RepeatCount:          2,
	})

	if got.HalfLifeDays <= 1.0 {
		t.Fatalf("expected half-life growth, got %.3f", got.HalfLifeDays)
	}
	if got.MasteryScore <= 0.1 {
		t.Fatalf("expected mastery growth, got %.3f", got.MasteryScore)
	}
	if !got.NextDue.After(at) {
		t.Fatalf("expected next due after attempt time")
	}
}

func TestHalfLifeIncorrectAnswerReducesHalfLife(t *testing.T) {
	at := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	last := at.Add(-24 * time.Hour)
	got := CalculateHalfLifeUpdate(HalfLifeInput{
		PreviousHalfLifeDays: 2.0,
		PreviousMasteryScore: 0.4,
		LastSeenAt:           &last,
		AttemptedAt:          at,
		Result:               "incorrect",
		ResponseTimeMS:       6000,
		Difficulty:           2.5,
		RepeatCount:          4,
	})

	if got.HalfLifeDays >= 2.0 {
		t.Fatalf("expected half-life reduction, got %.3f", got.HalfLifeDays)
	}
	if got.NextDue.Sub(at) > 3*time.Hour {
		t.Fatalf("expected incorrect answer to return soon, got interval %s", got.NextDue.Sub(at))
	}
}

func TestHalfLifeRecallProbabilityDecaysByHalfAfterOneHalfLife(t *testing.T) {
	at := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	last := at.Add(-48 * time.Hour)
	got := CalculateHalfLifeUpdate(HalfLifeInput{
		PreviousHalfLifeDays: 2.0,
		PreviousMasteryScore: 0.3,
		LastSeenAt:           &last,
		AttemptedAt:          at,
		Result:               "partial",
		ResponseTimeMS:       5000,
		Difficulty:           1.5,
		RepeatCount:          3,
	})

	if math.Abs(got.RecallProbability-0.5) > 0.02 {
		t.Fatalf("expected recall probability near 0.5, got %.3f", got.RecallProbability)
	}
}
