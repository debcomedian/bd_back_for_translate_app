package services

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"bd_back_for_translate_app/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ImportActiveBank(db *gorm.DB, inputPath string) (*ActiveBankImportReport, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть active_bank: %w", err)
	}
	defer f.Close()

	report := &ActiveBankImportReport{InputPath: inputPath}

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

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var rec ActiveBankRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			report.SkippedInvalid++
			continue
		}
		report.Processed++

		en := normalizeLexeme(rec.EnLemma)
		if en == "" {
			report.SkippedWithoutEnglish++
			continue
		}

		ruPrimary := choosePrimaryTarget(rec.Targets["ru"])
		dePrimary := choosePrimaryTarget(rec.Targets["de"])

		if ruPrimary != "" {
			report.ImportedWithRu++
		}
		if dePrimary != "" {
			report.ImportedWithDe++
		}
		if ruPrimary != "" && dePrimary != "" {
			report.ImportedWithBoth++
		}

		existing, found, err := findExistingImportedWord(tx, en)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		word, created, err := upsertImportedWord(tx, existing, found, rec, en, ruPrimary, dePrimary)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if created {
			report.CreatedWords++
		} else {
			report.UpdatedWords++
		}

		upserts, err := upsertMetaLangRows(tx, word.ID, rec, en, ruPrimary, dePrimary)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.MetaLangUpserts += upserts

		if err := upsertBaseMetaFromImportedRecord(tx, word.ID, rec, en, ruPrimary, dePrimary); err != nil {
			tx.Rollback()
			return nil, err
		}
		report.MetaBaseUpserts++

		insertedSynonyms, err := upsertTargetSynonyms(tx, word.ID, ruPrimary, dePrimary, rec)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.SynonymsInserted += insertedSynonyms
	}

	if err := scanner.Err(); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("ошибка чтения active_bank: %w", err)
	}

	snap, err := publishNextSnapshotVersionForImport(tx, "words_base")
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	report.SnapshotPublished = true
	report.SnapshotVersionCode = snap.VersionCode
	report.SnapshotChecksum = snap.Checksum
	return report, nil
}

func findExistingImportedWord(tx *gorm.DB, enLemma string) (models.Word, bool, error) {
	var word models.Word
	err := tx.Where("lang_code = ? AND word_en = ?", "en", enLemma).First(&word).Error
	if err == nil {
		return word, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return models.Word{}, false, nil
	}
	return models.Word{}, false, err
}

func upsertImportedWord(tx *gorm.DB, existing models.Word, found bool, rec ActiveBankRecord, en, ru, de string) (models.Word, bool, error) {
	sourceRef := fmt.Sprintf("active_bank:%s", strings.TrimSpace(rec.Status))
	exampleEn := firstGloss(rec.Glosses)

	if !found {
		obj := models.Word{
			LangCode:  "en",
			WordEn:    strPtr(en),
			WordRu:    nilIfEmpty(ru),
			WordDe:    nilIfEmpty(de),
			ExampleEn: nilIfEmpty(exampleEn),
			SourceRef: strPtr(sourceRef),
			IsActive:  true,
		}
		if err := tx.Create(&obj).Error; err != nil {
			return models.Word{}, false, err
		}
		return obj, true, nil
	}

	updates := map[string]any{
		"word_ru":    nilIfEmpty(ru),
		"word_de":    nilIfEmpty(de),
		"example_en": nilIfEmpty(exampleEn),
		"source_ref": strPtr(sourceRef),
		"is_active":  true,
	}
	if err := tx.Model(&existing).Updates(updates).Error; err != nil {
		return models.Word{}, false, err
	}

	existing.WordRu = nilIfEmpty(ru)
	existing.WordDe = nilIfEmpty(de)
	existing.ExampleEn = nilIfEmpty(exampleEn)
	existing.SourceRef = strPtr(sourceRef)
	existing.IsActive = true
	return existing, false, nil
}

func upsertMetaLangRows(tx *gorm.DB, wordID uint64, rec ActiveBankRecord, en, ru, de string) (int, error) {
	rows := []models.WordMetaLang{
		buildImportedLangMeta(wordID, "en", en, rec.Frequency.EN.Zipf, bucketLabelToInt(rec.Frequency.EN.Bucket), rec.Status, rec.Pos),
	}

	if ru != "" {
		rows = append(rows, buildImportedLangMeta(wordID, "ru", ru, 0, 0, rec.Status, rec.Pos))
	}
	if de != "" {
		rows = append(rows, buildImportedLangMeta(wordID, "de", de, 0, 0, rec.Status, rec.Pos))
	}

	for _, row := range rows {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "word_id"}, {Name: "lang_code"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"lemma",
				"lemma_chars",
				"token_count",
				"zipf_frequency",
				"freq_bucket",
				"orthography_score",
				"multiword_score",
				"pos_score",
				"confidence_penalty",
				"lang_score",
				"calculated_at",
			}),
		}).Create(&row).Error; err != nil {
			return 0, err
		}
	}
	return len(rows), nil
}

func buildImportedLangMeta(wordID uint64, langCode, lemma string, zipf float64, freqBucket int, status, pos string) models.WordMetaLang {
	tokenCount := countTokens(lemma)
	multiwordScore := seedMultiwordScore(tokenCount)

	return models.WordMetaLang{
		WordID:            wordID,
		LangCode:          langCode,
		Lemma:             lemma,
		LemmaChars:        utf8.RuneCountInString(lemma),
		TokenCount:        tokenCount,
		ZipfFrequency:     zipf,
		FreqBucket:        freqBucket,
		OrthographyScore:  0,
		MultiwordScore:    multiwordScore,
		POSScore:          seedPOSScore(pos),
		ConfidencePenalty: seedConfidencePenalty(status),
		LangScore:         0,
		CalculatedAt:      time.Now(),
	}
}

func upsertBaseMetaFromImportedRecord(tx *gorm.DB, wordID uint64, rec ActiveBankRecord, en, ru, de string) error {
	maxChars := utf8.RuneCountInString(en)
	for _, v := range []string{ru, de} {
		if c := utf8.RuneCountInString(v); c > maxChars {
			maxChars = c
		}
	}

	cefr := seedCEFRFromBucket(rec.Frequency.EN.Bucket)
	meta := models.WordMetaBase{
		WordID:                  wordID,
		MetaCefrLevel:           strPtr(cefr),
		MetaImportanceScore:     seedImportanceFromZipf(rec.Frequency.EN.Zipf),
		MetaFreqBucket:          bucketLabelToInt(rec.Frequency.EN.Bucket),
		MetaLengthChars:         maxChars,
		MetaBaseDifficulty:      seedBaseDifficulty(rec.Frequency.EN.Zipf, maxChars, rec.Status),
		MetaBlocklistFlag:       false,
		MetaForcedIntroduceFlag: false,
		MetaVersion:             1,
		CalculatedAt:            time.Now(),
	}

	return tx.Clauses(clause.OnConflict{
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
	}).Create(&meta).Error
}

func upsertTargetSynonyms(tx *gorm.DB, wordID uint64, ruPrimary, dePrimary string, rec ActiveBankRecord) (int, error) {
	inserted := 0

	for _, item := range buildSynonymRows(wordID, "ru", ruPrimary, rec.Targets["ru"]) {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&item).Error; err != nil {
			return inserted, err
		}
		inserted++
	}
	for _, item := range buildSynonymRows(wordID, "de", dePrimary, rec.Targets["de"]) {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&item).Error; err != nil {
			return inserted, err
		}
		inserted++
	}
	return inserted, nil
}

func buildSynonymRows(wordID uint64, langCode, primary string, state ActiveBankTargetState) []models.WordSynonym {
	seen := map[string]struct{}{}
	out := []models.WordSynonym{}

	add := func(v string, isPrimary bool) {
		v = normalizeLexeme(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, models.WordSynonym{
			WordID:       wordID,
			LangCode:     langCode,
			SynonymValue: v,
			IsPrimary:    isPrimary,
			CreatedAt:    time.Now(),
		})
	}

	add(primary, true)
	for _, group := range [][]string{state.StrictValidated, state.SoftValidated, state.Completed, state.FromEnglish} {
		for _, v := range group {
			add(v, normalizeLexeme(v) == normalizeLexeme(primary))
		}
	}
	return out
}

func choosePrimaryTarget(state ActiveBankTargetState) string {
	for _, group := range [][]string{
		state.StrictValidated,
		state.SoftValidated,
		state.Completed,
		state.FromEnglish,
	} {
		for _, v := range group {
			n := normalizeLexeme(v)
			if n != "" {
				return n
			}
		}
	}
	return ""
}

func normalizeLexeme(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "ё", "е")
	s = strings.ReplaceAll(s, "Ё", "Е")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func nilIfEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func strPtr(s string) *string {
	return &s
}

func firstGloss(glosses []string) string {
	if len(glosses) == 0 {
		return ""
	}
	return strings.TrimSpace(glosses[0])
}

func countTokens(s string) int {
	if strings.TrimSpace(s) == "" {
		return 0
	}
	return len(strings.Fields(s))
}

func bucketLabelToInt(label string) int {
	switch strings.TrimSpace(label) {
	case "core_high":
		return 1
	case "core_mid":
		return 2
	case "core_low":
		return 3
	case "tail":
		return 4
	default:
		return 5
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

func seedMultiwordScore(tokenCount int) float64 {
	switch {
	case tokenCount <= 1:
		return 0.00
	case tokenCount == 2:
		return 0.35
	default:
		return 0.65
	}
}

func seedPOSScore(pos string) float64 {
	switch strings.TrimSpace(pos) {
	case "noun":
		return 0.15
	case "adj":
		return 0.20
	case "adv":
		return 0.25
	case "verb":
		return 0.30
	default:
		return 0.20
	}
}

func seedCEFRFromBucket(bucket string) string {
	switch strings.TrimSpace(bucket) {
	case "core_high":
		return "A1"
	case "core_mid":
		return "A2"
	case "core_low":
		return "B1"
	case "tail":
		return "B2"
	default:
		return "C1"
	}
}

func seedImportanceFromZipf(zipf float64) int {
	switch {
	case zipf >= 5.5:
		return 95
	case zipf >= 4.5:
		return 80
	case zipf >= 3.5:
		return 65
	case zipf > 0:
		return 45
	default:
		return 25
	}
}

func seedBaseDifficulty(zipf float64, length int, status string) float64 {
	freqInverse := 1.0
	if zipf > 0 {
		freqInverse = (6.5 - zipf) / 6.5
		if freqInverse < 0 {
			freqInverse = 0
		}
		if freqInverse > 1 {
			freqInverse = 1
		}
	}

	lengthScore := float64(length) / 12.0
	if lengthScore > 1 {
		lengthScore = 1
	}

	value := 0.65*freqInverse + 0.20*lengthScore + 0.15*seedConfidencePenalty(status)
	return value * 10.0
}

func publishNextSnapshotVersionForImport(tx *gorm.DB, snapshotType string) (*models.ContentSnapshotVersion, error) {
	var current models.ContentSnapshotVersion
	var nextVersion int64 = 1
	if err := tx.Where("snapshot_type = ?", snapshotType).Order("version_code DESC").First(&current).Error; err == nil {
		nextVersion = current.VersionCode + 1
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	checksum, err := calculateSnapshotChecksumForImport(tx, snapshotType, nextVersion)
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

func calculateSnapshotChecksumForImport(tx *gorm.DB, snapshotType string, version int64) (string, error) {
	var wordsCount int64
	var categoriesCount int64
	var synonymsCount int64
	var metasCount int64
	var metasLangCount int64

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
	if err := tx.Model(&models.WordMetaLang{}).Count(&metasLangCount).Error; err != nil {
		return "", err
	}

	payload := fmt.Sprintf("%s:%d:%d:%d:%d:%d:%d", snapshotType, version, wordsCount, categoriesCount, synonymsCount, metasCount, metasLangCount)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:]), nil
}
