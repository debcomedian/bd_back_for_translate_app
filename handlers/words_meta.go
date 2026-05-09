package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"bd_back_for_translate_app/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type recalcMetaResponse struct {
	Processed int `json:"processed"`
	Snapshot  any `json:"snapshot,omitempty"`
}

func RecalculateWordMeta(c *gin.Context) {
	var words []models.Word
	if err := DB.Where("is_active = TRUE").Find(&words).Error; err != nil {
		handleDBErr(c, err)
		return
	}

	tx := DB.Begin()
	if tx.Error != nil {
		handleDBErr(c, tx.Error)
		return
	}

	processed := 0
	for _, w := range words {
		source := sourceWordByLang(w)
		length := utf8.RuneCountInString(source)
		cefr := guessCEFR(length)
		importance := guessImportance(length)
		freqBucket := guessFreqBucket(length)
		difficulty := calcDifficulty(length, cefr)

		meta := models.WordMetaBase{
			WordID:                  w.ID,
			MetaCefrLevel:           &cefr,
			MetaImportanceScore:     importance,
			MetaFreqBucket:          freqBucket,
			MetaLengthChars:         length,
			MetaBaseDifficulty:      difficulty,
			MetaBlocklistFlag:       false,
			MetaForcedIntroduceFlag: false,
			MetaVersion:             1,
			CalculatedAt:            time.Now(),
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "word_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"meta_cefr_level",
				"meta_importance_score",
				"meta_freq_bucket",
				"meta_length_chars",
				"meta_base_difficulty",
				"meta_required_stage",
				"meta_blocklist_flag",
				"meta_forced_introduce_flag",
				"meta_version",
				"calculated_at",
			}),
		}).Create(&meta).Error; err != nil {
			tx.Rollback()
			handleDBErr(c, err)
			return
		}
		processed++
	}

	snap, err := publishNextSnapshotVersion(tx, "words_base")
	if err != nil {
		tx.Rollback()
		handleDBErr(c, err)
		return
	}
	if err := tx.Commit().Error; err != nil {
		handleDBErr(c, err)
		return
	}

	respondOK(c, recalcMetaResponse{Processed: processed, Snapshot: snap})
}

func sourceWordByLang(w models.Word) string {
	switch strings.TrimSpace(w.LangCode) {
	case "ru":
		return deref(w.WordRu)
	case "en":
		return deref(w.WordEn)
	case "de":
		return deref(w.WordDe)
	default:
		if w.WordRu != nil && *w.WordRu != "" {
			return deref(w.WordRu)
		}
		if w.WordEn != nil && *w.WordEn != "" {
			return deref(w.WordEn)
		}
		return deref(w.WordDe)
	}
}

func guessCEFR(length int) string {
	switch {
	case length <= 4:
		return "A1"
	case length <= 6:
		return "A2"
	case length <= 8:
		return "B1"
	case length <= 10:
		return "B2"
	case length <= 12:
		return "C1"
	default:
		return "C2"
	}
}

func guessImportance(length int) int {
	switch {
	case length <= 4:
		return 90
	case length <= 6:
		return 75
	case length <= 8:
		return 60
	case length <= 10:
		return 45
	default:
		return 30
	}
}

func guessFreqBucket(length int) int {
	switch {
	case length <= 4:
		return 1
	case length <= 6:
		return 2
	case length <= 8:
		return 3
	case length <= 10:
		return 4
	default:
		return 5
	}
}

func calcDifficulty(length int, cefr string) float64 {
	cefrWeight := map[string]float64{
		"A1": 1,
		"A2": 2,
		"B1": 3,
		"B2": 4,
		"C1": 5,
		"C2": 6,
	}
	return float64(length)*0.35 + cefrWeight[cefr]*1.25
}

func publishNextSnapshotVersion(tx *gorm.DB, snapshotType string) (*models.ContentSnapshotVersion, error) {
	var current models.ContentSnapshotVersion
	var nextVersion int64 = 1
	if err := tx.Where("snapshot_type = ?", snapshotType).Order("version_code DESC").First(&current).Error; err == nil {
		nextVersion = current.VersionCode + 1
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	checksum, err := calculateSnapshotChecksum(tx, snapshotType, nextVersion)
	if err != nil {
		return nil, err
	}

	if err := tx.Model(&models.ContentSnapshotVersion{}).
		Where("snapshot_type = ? AND is_active = TRUE", snapshotType).
		Update("is_active", false).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	snap := &models.ContentSnapshotVersion{
		SnapshotType: snapshotType,
		VersionCode:  nextVersion,
		Checksum:     checksum,
		IsActive:     true,
		PublishedAt:  &now,
	}
	if err := tx.Create(snap).Error; err != nil {
		return nil, err
	}
	return snap, nil
}

func calculateSnapshotChecksum(tx *gorm.DB, snapshotType string, version int64) (string, error) {
	var wordsCount int64
	var categoriesCount int64
	var synonymsCount int64
	var metasCount int64

	if err := tx.Model(&models.Word{}).Where("is_active = TRUE").Count(&wordsCount).Error; err != nil {
		return "", err
	}
	if err := tx.Model(&models.Category{}).Count(&categoriesCount).Error; err != nil {
		return "", err
	}
	if err := tx.Model(&models.WordSynonym{}).Count(&synonymsCount).Error; err != nil {
		return "", err
	}
	if err := tx.Model(&models.WordMetaBase{}).Count(&metasCount).Error; err != nil {
		return "", err
	}

	payload := fmt.Sprintf("%s:%d:%d:%d:%d:%d", snapshotType, version, wordsCount, categoriesCount, synonymsCount, metasCount)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:]), nil
}
