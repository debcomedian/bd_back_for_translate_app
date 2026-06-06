package lexicon

import "sort"

type UserKnowledgeSummary struct {
	UserID                    uint64  `json:"user_id"`
	PracticedDirections       int     `json:"practiced_directions"`
	MasteredDirections        int     `json:"mastered_directions"`
	AverageMastery            float64 `json:"average_mastery"`
	WeightedKnowledgeScore    float64 `json:"weighted_knowledge_score"`
	EstimatedDifficulty       float64 `json:"estimated_difficulty"`
	EstimatedCEFR             string  `json:"estimated_cefr"`
	NextRecommendedDifficulty float64 `json:"next_recommended_difficulty"`
}

type progressWithDifficulty struct {
	Progress   UserDirectionProgress
	Difficulty float64
}

func CalculateUserKnowledgeSummary(userID uint64, items []UserDirectionProgress, difficultyByDirection map[uint64]float64) UserKnowledgeSummary {
	summary := UserKnowledgeSummary{UserID: userID, EstimatedCEFR: "A1"}
	if len(items) == 0 {
		summary.NextRecommendedDifficulty = 0.250
		return summary
	}

	rows := make([]progressWithDifficulty, 0, len(items))
	for _, item := range items {
		difficulty := difficultyByDirection[item.DirectionID]
		if difficulty <= 0 {
			difficulty = defaultDifficultyForHLR
		}
		rows = append(rows, progressWithDifficulty{Progress: item, Difficulty: difficulty})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Difficulty < rows[j].Difficulty })

	totalMastery := 0.0
	weighted := 0.0
	weightTotal := 0.0
	mastered := 0
	frontier := 0.250

	for _, row := range rows {
		mastery := clamp(row.Progress.MasteryScore, 0, 1)
		totalMastery += mastery
		weight := 1.0 + row.Difficulty/6.0
		weighted += row.Difficulty * mastery * weight
		weightTotal += weight
		if mastery >= 0.80 || row.Progress.HalfLifeDays >= masteredHalfLifeDays {
			mastered++
			frontier = row.Difficulty
		}
	}

	summary.PracticedDirections = len(rows)
	summary.MasteredDirections = mastered
	summary.AverageMastery = round3(totalMastery / float64(len(rows)))
	if weightTotal > 0 {
		summary.WeightedKnowledgeScore = round3(weighted / weightTotal)
	}

	summary.EstimatedDifficulty = round3(clamp(frontier, 0.050, 5.999))
	summary.EstimatedCEFR = CEFRFromDifficulty(summary.EstimatedDifficulty)
	summary.NextRecommendedDifficulty = round3(clamp(summary.EstimatedDifficulty+0.250, 0.050, 5.999))
	return summary
}
