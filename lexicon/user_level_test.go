package lexicon

import "testing"

func TestKnowledgeSummaryUsesMasteredFrontier(t *testing.T) {
	progress := []UserDirectionProgress{
		{UserID: 7, DirectionID: 1, MasteryScore: 0.90, HalfLifeDays: 100},
		{UserID: 7, DirectionID: 2, MasteryScore: 0.82, HalfLifeDays: 95},
		{UserID: 7, DirectionID: 3, MasteryScore: 0.30, HalfLifeDays: 2},
	}
	difficulties := map[uint64]float64{
		1: 0.700,
		2: 1.250,
		3: 2.400,
	}

	got := CalculateUserKnowledgeSummary(7, progress, difficulties)

	if got.MasteredDirections != 2 {
		t.Fatalf("expected 2 mastered directions, got %d", got.MasteredDirections)
	}
	if got.EstimatedCEFR != "A2" {
		t.Fatalf("expected A2 frontier, got %s", got.EstimatedCEFR)
	}
	if got.NextRecommendedDifficulty <= got.EstimatedDifficulty {
		t.Fatalf("expected next recommendation to be above current frontier")
	}
}
