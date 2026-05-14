package lexicon

import (
	"crypto/sha1"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const directionalSnapshotType = "lexicon_directional"

func clearLexiconContent(tx *gorm.DB) error {
	return tx.Exec(`
TRUNCATE TABLE
	lexicon.audit_log,
	lexicon.sync_events,
	lexicon.user_direction_progress,
	lexicon.attempts,
	lexicon.content_snapshot_versions,
	lexicon.training_direction_meta,
	lexicon.training_directions,
	lexicon.lexical_form_synonyms,
	lexicon.lexical_form_meta,
	lexicon.lexical_concept_meta,
	lexicon.lexical_forms,
	lexicon.lexical_concepts,
	lexicon.categories
RESTART IDENTITY CASCADE;
`).Error
}

func publishSnapshotVersion(tx *gorm.DB) (*ContentSnapshotVersion, error) {
	var last ContentSnapshotVersion
	err := tx.
		Where("snapshot_type = ?", directionalSnapshotType).
		Order("version_code DESC").
		First(&last).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	nextVersion := int64(1)
	if last.ID != 0 {
		nextVersion = last.VersionCode + 1
	}

	checksum, err := calculateSnapshotChecksum(tx, nextVersion)
	if err != nil {
		return nil, err
	}

	if err := tx.
		Model(&ContentSnapshotVersion{}).
		Where("snapshot_type = ?", directionalSnapshotType).
		Update("is_active", false).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	snapshot := ContentSnapshotVersion{
		SnapshotType: directionalSnapshotType,
		VersionCode:  nextVersion,
		Checksum:     checksum,
		IsActive:     true,
		PublishedAt:  &now,
	}

	if err := tx.Create(&snapshot).Error; err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func calculateSnapshotChecksum(tx *gorm.DB, versionCode int64) (string, error) {
	counts := map[string]int64{}

	items := []struct {
		name  string
		model any
	}{
		{"languages", &Language{}},
		{"categories", &Category{}},
		{"concepts", &LexicalConcept{}},
		{"forms", &LexicalForm{}},
		{"concept_meta", &LexicalConceptMeta{}},
		{"form_meta", &LexicalFormMeta{}},
		{"form_synonyms", &LexicalFormSynonym{}},
		{"directions", &TrainingDirection{}},
		{"direction_meta", &TrainingDirectionMeta{}},
	}

	for _, item := range items {
		var count int64
		if err := tx.Model(item.model).Count(&count).Error; err != nil {
			return "", err
		}
		counts[item.name] = count
	}

	source := fmt.Sprintf(
		"%s|%d|languages=%d|categories=%d|concepts=%d|forms=%d|concept_meta=%d|form_meta=%d|form_synonyms=%d|directions=%d|direction_meta=%d",
		directionalSnapshotType,
		versionCode,
		counts["languages"],
		counts["categories"],
		counts["concepts"],
		counts["forms"],
		counts["concept_meta"],
		counts["form_meta"],
		counts["form_synonyms"],
		counts["directions"],
		counts["direction_meta"],
	)

	sum := sha1.Sum([]byte(source))
	return fmt.Sprintf("%x", sum), nil
}
