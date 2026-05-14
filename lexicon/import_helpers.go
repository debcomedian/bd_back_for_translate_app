package lexicon

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func generateDirectionsForConcept(tx *gorm.DB, conceptID uint64, forms []LexicalForm) (int, error) {
	if conceptID == 0 {
		return 0, fmt.Errorf("concept_id не указан")
	}

	created := 0

	for _, source := range forms {
		if source.ID == 0 {
			continue
		}

		sourceLang := NormalizeValue(source.LangCode)
		if sourceLang == "" {
			continue
		}

		for _, target := range forms {
			if target.ID == 0 {
				continue
			}

			targetLang := NormalizeValue(target.LangCode)
			if targetLang == "" {
				continue
			}

			if source.ID == target.ID || sourceLang == targetLang {
				continue
			}

			direction := TrainingDirection{
				ConceptID:      conceptID,
				SourceFormID:   source.ID,
				TargetFormID:   target.ID,
				SourceLangCode: sourceLang,
				TargetLangCode: targetLang,
				DirectionCode:  sourceLang + "_" + targetLang,
				IsActive:       true,
				CreatedAt:      time.Now(),
			}

			result := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "concept_id"},
					{Name: "source_form_id"},
					{Name: "target_form_id"},
				},
				DoNothing: true,
			}).Create(&direction)

			if result.Error != nil {
				return created, result.Error
			}

			if result.RowsAffected > 0 {
				created++
			}
		}
	}

	return created, nil
}

func createSynonymIfMissing(tx *gorm.DB, formID uint64, value string, isPrimary bool) error {
	if formID == 0 {
		return fmt.Errorf("form_id не указан")
	}

	value = normalizeLexeme(value)
	normalized := NormalizeValue(value)

	if normalized == "" {
		return nil
	}

	synonym := LexicalFormSynonym{
		FormID:          formID,
		SynonymValue:    value,
		NormalizedValue: normalized,
		IsPrimary:       isPrimary,
		CreatedAt:       time.Now(),
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "form_id"},
			{Name: "normalized_value"},
		},
		DoNothing: true,
	}).Create(&synonym).Error
}
