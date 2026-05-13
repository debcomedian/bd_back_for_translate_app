package lexicon

import (
	"os"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DirectionMetaReport struct {
	Processed int `json:"processed"`
}

func RecalculateDirectionMeta(db *gorm.DB) (*DirectionMetaReport, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	report, err := recalculateDirectionMetaTx(tx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if _, err := publishSnapshotVersion(tx); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return report, nil
}

func recalculateDirectionMetaTx(tx *gorm.DB) (*DirectionMetaReport, error) {
	var directions []TrainingDirection
	if err := tx.Where("is_active = TRUE").Order("id ASC").Find(&directions).Error; err != nil {
		return nil, err
	}

	report := &DirectionMetaReport{}
	for _, direction := range directions {
		meta, err := calculateDirectionMeta(tx, direction)
		if err != nil {
			return nil, err
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "direction_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"source_form_score",
				"target_form_score",
				"concept_base_difficulty",
				"direction_bias",
				"synonym_relief",
				"final_difficulty",
				"meta_version",
				"calculated_at",
			}),
		}).Create(&meta).Error; err != nil {
			return nil, err
		}
		report.Processed++
	}
	return report, nil
}

func calculateDirectionMeta(tx *gorm.DB, direction TrainingDirection) (TrainingDirectionMeta, error) {
	if err := ensureNonZeroID(direction.ID, "direction_id"); err != nil {
		return TrainingDirectionMeta{}, err
	}

	sourceScore, err := getFormScore(tx, direction.SourceFormID)
	if err != nil {
		return TrainingDirectionMeta{}, err
	}
	targetScore, err := getFormScore(tx, direction.TargetFormID)
	if err != nil {
		return TrainingDirectionMeta{}, err
	}
	conceptDifficulty, err := getConceptDifficulty(tx, direction.ConceptID)
	if err != nil {
		return TrainingDirectionMeta{}, err
	}
	synonymRelief, err := getSynonymRelief(tx, direction.TargetFormID)
	if err != nil {
		return TrainingDirectionMeta{}, err
	}

	bias := directionBias(direction.SourceLangCode, direction.TargetLangCode)
	final := conceptDifficulty*0.50 + targetScore*0.35 + sourceScore*0.15 + bias - synonymRelief
	if final < 0.1 {
		final = 0.1
	}

	return TrainingDirectionMeta{
		DirectionID:           direction.ID,
		SourceFormScore:       round3(sourceScore),
		TargetFormScore:       round3(targetScore),
		ConceptBaseDifficulty: round3(conceptDifficulty),
		DirectionBias:         round3(bias),
		SynonymRelief:         round3(synonymRelief),
		FinalDifficulty:       round3(final),
		MetaVersion:           1,
		CalculatedAt:          time.Now(),
	}, nil
}

func getFormScore(tx *gorm.DB, formID uint64) (float64, error) {
	var meta LexicalFormMeta
	if err := tx.Where("form_id = ?", formID).First(&meta).Error; err == nil {
		if meta.FormScore > 0 {
			return meta.FormScore, nil
		}
	} else if err != gorm.ErrRecordNotFound {
		return 0, err
	}

	var form LexicalForm
	if err := tx.Where("id = ?", formID).First(&form).Error; err != nil {
		return 0, err
	}
	return fallbackFormScore(form), nil
}

func getConceptDifficulty(tx *gorm.DB, conceptID uint64) (float64, error) {
	var meta LexicalConceptMeta
	if err := tx.Where("concept_id = ?", conceptID).First(&meta).Error; err == nil {
		if meta.BaseDifficulty > 0 {
			return meta.BaseDifficulty, nil
		}
	} else if err != gorm.ErrRecordNotFound {
		return 0, err
	}

	var forms []LexicalForm
	if err := tx.Where("concept_id = ?", conceptID).Find(&forms).Error; err != nil {
		return 0, err
	}
	return fallbackConceptDifficulty(forms), nil
}

func getSynonymRelief(tx *gorm.DB, targetFormID uint64) (float64, error) {
	var count int64
	if err := tx.Model(&LexicalFormSynonym{}).Where("form_id = ?", targetFormID).Count(&count).Error; err != nil {
		return 0, err
	}
	relief := float64(count) * 0.05
	if relief > 0.30 {
		relief = 0.30
	}
	return relief, nil
}

func directionBias(sourceLang, targetLang string) float64 {
	nativeLang := NormalizeValue(os.Getenv("NATIVE_LANG_CODE"))
	if nativeLang == "" {
		nativeLang = "ru"
	}

	switch {
	case targetLang == nativeLang:
		return -0.25
	case sourceLang == nativeLang && targetLang != nativeLang:
		return 0.60
	case sourceLang != nativeLang && targetLang != nativeLang:
		return 0.80
	default:
		return 0
	}
}
