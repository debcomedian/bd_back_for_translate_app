package lexicon

import "testing"

func TestDifficultyPlacesFrequentCoreWordIntoA1Band(t *testing.T) {
	got := CalculateWordDifficulty(DifficultyInput{
		ZipfFrequency: 5.8,
		FreqBucket:    "core_high",
		LemmaChars:    2,
		TokenCount:    1,
		Status:        "strict_validated_all",
		POS:           "verb",
		Value:         "go",
	})

	if got.CefrLevel != "A1" {
		t.Fatalf("expected A1, got %s", got.CefrLevel)
	}
	if got.Difficulty <= 0 || got.Difficulty >= 1 {
		t.Fatalf("expected A1 difficulty in [0;1), got %.3f", got.Difficulty)
	}
}

func TestDifficultyPlacesRareTailWordIntoB2Band(t *testing.T) {
	got := CalculateWordDifficulty(DifficultyInput{
		ZipfFrequency: 2.9,
		FreqBucket:    "tail",
		LemmaChars:    20,
		TokenCount:    1,
		Status:        "soft_validated_all",
		POS:           "noun",
		Value:         "institutionalization",
	})

	if got.CefrLevel != "B2" {
		t.Fatalf("expected B2, got %s", got.CefrLevel)
	}
	if got.Difficulty < 3 || got.Difficulty >= 4 {
		t.Fatalf("expected B2 difficulty in [3;4), got %.3f", got.Difficulty)
	}
}

func TestDirectionBiasMakesNativeToForeignHarderThanForeignToNative(t *testing.T) {
	t.Setenv("NATIVE_LANG_CODE", "ru")

	ruToEn := DirectionDifficultyBias("ru", "en")
	enToRu := DirectionDifficultyBias("en", "ru")

	if ruToEn <= enToRu {
		t.Fatalf("expected ru->en bias to be higher than en->ru, got %.3f <= %.3f", ruToEn, enToRu)
	}
}

func TestBlendDirectionDifficultyUsesTargetAndBias(t *testing.T) {
	base := BlendDirectionDifficulty(1.2, 1.0, 1.0, 0, 0)
	harderTarget := BlendDirectionDifficulty(1.2, 1.0, 2.0, 0, 0)
	withRelief := BlendDirectionDifficulty(1.2, 1.0, 2.0, 0, 0.1)

	if harderTarget <= base {
		t.Fatalf("expected harder target to increase difficulty: base %.3f target %.3f", base, harderTarget)
	}
	if withRelief >= harderTarget {
		t.Fatalf("expected synonym relief to reduce difficulty: with relief %.3f without %.3f", withRelief, harderTarget)
	}
}
