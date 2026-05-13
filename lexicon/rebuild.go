package lexicon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	base "bd_back_for_translate_app/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const directionalSnapshotType = "words_directional"

type RebuildReport struct {
	Categories          int    `json:"categories"`
	Concepts            int    `json:"concepts"`
	Forms               int    `json:"forms"`
	FormMeta            int    `json:"form_meta"`
	ConceptMeta         int    `json:"concept_meta"`
	Synonyms            int    `json:"synonyms"`
	Directions          int    `json:"directions"`
	DirectionMeta       int    `json:"direction_meta"`
	SnapshotVersionCode int64  `json:"snapshot_version_code"`
	SnapshotChecksum    string `json:"snapshot_checksum"`
}

func RebuildFromCurrent(db *gorm.DB) (*RebuildReport, error) {
	report := &RebuildReport{}

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

	if err := clearLexiconContent(tx); err != nil {
		tx.Rollback()
		return nil, err
	}

	categoryMap, copiedCategories, err := copyCategories(tx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	report.Categories = copiedCategories

	var words []base.Word
	if err := tx.Where("is_active = TRUE").Order("id ASC").Find(&words).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, word := range words {
		concept, err := createConceptFromWord(tx, word, categoryMap)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.Concepts++

		forms, err := createFormsFromWord(tx, concept.ID, word)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.Forms += len(forms)

		createdFormMeta, err := copyFormMeta(tx, word.ID, forms)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.FormMeta += createdFormMeta

		if err := copyConceptMeta(tx, concept.ID, word.ID, forms); err != nil {
			tx.Rollback()
			return nil, err
		}
		report.ConceptMeta++

		createdSynonyms, err := copySynonyms(tx, word.ID, forms)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.Synonyms += createdSynonyms

		directions, err := generateDirectionsForConcept(tx, concept.ID, forms)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.Directions += directions
	}

	metaReport, err := recalculateDirectionMetaTx(tx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	report.DirectionMeta = metaReport.Processed

	snapshot, err := publishSnapshotVersion(tx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	report.SnapshotVersionCode = snapshot.VersionCode
	report.SnapshotChecksum = snapshot.Checksum

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return report, nil
}

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

func copyCategories(tx *gorm.DB) (map[uint64]uint64, int, error) {
	var source []base.Category
	if err := tx.Order("id ASC").Find(&source).Error; err != nil {
		return nil, 0, err
	}

	result := make(map[uint64]uint64, len(source))
	created := 0
	for _, src := range source {
		obj := Category{
			SourceID:  &src.ID,
			Slug:      src.Slug,
			NameRu:    src.NameRu,
			NameEn:    src.NameEn,
			NameDe:    src.NameDe,
			Entity:    "concept",
			CreatedAt: src.CreatedAt,
			UpdatedAt: src.UpdatedAt,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"source_id",
				"name_ru",
				"name_en",
				"name_de",
				"entity",
				"updated_at",
			}),
		}).Create(&obj).Error; err != nil {
			return nil, 0, err
		}
		result[src.ID] = obj.ID
		created++
	}
	return result, created, nil
}

func createConceptFromWord(tx *gorm.DB, word base.Word, categoryMap map[uint64]uint64) (LexicalConcept, error) {
	var categoryID *uint64
	if word.CategoryID != nil {
		if mapped, ok := categoryMap[*word.CategoryID]; ok {
			categoryID = &mapped
		}
	}

	obj := LexicalConcept{
		SourceWordID: &word.ID,
		CategoryID:   categoryID,
		SourceRef:    word.SourceRef,
		IsActive:     word.IsActive,
		CreatedAt:    word.CreatedAt,
		UpdatedAt:    word.UpdatedAt,
	}
	if err := tx.Create(&obj).Error; err != nil {
		return LexicalConcept{}, err
	}
	return obj, nil
}

func createFormsFromWord(tx *gorm.DB, conceptID uint64, word base.Word) ([]LexicalForm, error) {
	candidates := []struct {
		lang          string
		value         *string
		transcription *string
		example       *string
	}{
		{lang: "ru", value: word.WordRu, transcription: word.TranscriptionRu, example: word.ExampleRu},
		{lang: "en", value: word.WordEn, transcription: word.TranscriptionEn, example: word.ExampleEn},
		{lang: "de", value: word.WordDe, transcription: word.TranscriptionDe, example: word.ExampleDe},
	}

	forms := make([]LexicalForm, 0, len(candidates))
	for _, c := range candidates {
		value := cleanValue(c.value)
		if value == "" {
			continue
		}
		obj := LexicalForm{
			ConceptID:       conceptID,
			LangCode:        c.lang,
			Value:           value,
			NormalizedValue: NormalizeValue(value),
			Transcription:   cleanPtr(c.transcription),
			Example:         cleanPtr(c.example),
			IsPrimary:       true,
		}
		if err := tx.Create(&obj).Error; err != nil {
			return nil, err
		}
		forms = append(forms, obj)
	}
	return forms, nil
}

func copyFormMeta(tx *gorm.DB, sourceWordID uint64, forms []LexicalForm) (int, error) {
	created := 0
	for _, form := range forms {
		var source base.WordMetaLang
		err := tx.Where("word_id = ? AND lang_code = ?", sourceWordID, form.LangCode).First(&source).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return 0, err
		}

		obj := LexicalFormMeta{
			FormID:       form.ID,
			Lemma:        strPtrIfNotEmpty(form.Value),
			LemmaChars:   utf8.RuneCountInString(form.Value),
			TokenCount:   countTokens(form.Value),
			CalculatedAt: time.Now(),
		}

		if err == nil {
			lemma := source.Lemma
			obj.Lemma = strPtrIfNotEmpty(lemma)
			obj.LemmaChars = source.LemmaChars
			obj.TokenCount = source.TokenCount
			obj.ZipfFrequency = source.ZipfFrequency
			obj.FreqBucket = source.FreqBucket
			obj.OrthographyScore = source.OrthographyScore
			obj.MultiwordScore = source.MultiwordScore
			obj.POSScore = source.POSScore
			obj.ConfidencePenalty = source.ConfidencePenalty
			obj.FormScore = source.LangScore
			obj.CalculatedAt = source.CalculatedAt
		} else {
			obj.FormScore = fallbackFormScore(form)
		}

		if obj.FormScore == 0 {
			obj.FormScore = fallbackFormScore(form)
		}
		if obj.TokenCount < 1 {
			obj.TokenCount = 1
		}

		if err := tx.Create(&obj).Error; err != nil {
			return 0, err
		}
		created++
	}
	return created, nil
}

func copyConceptMeta(tx *gorm.DB, conceptID uint64, sourceWordID uint64, forms []LexicalForm) error {
	var source base.WordMetaBase
	err := tx.Where("word_id = ?", sourceWordID).First(&source).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	obj := LexicalConceptMeta{
		ConceptID:       conceptID,
		ImportanceScore: 0,
		FreqBucket:      0,
		LengthChars:     maxFormLength(forms),
		BaseDifficulty:  fallbackConceptDifficulty(forms),
		MetaVersion:     1,
		CalculatedAt:    time.Now(),
	}

	if err == nil {
		obj.CefrLevel = source.MetaCefrLevel
		obj.ImportanceScore = source.MetaImportanceScore
		obj.FreqBucket = source.MetaFreqBucket
		obj.LengthChars = source.MetaLengthChars
		obj.BaseDifficulty = source.MetaBaseDifficulty
		obj.RequiredStage = source.MetaRequiredStage
		obj.BlocklistFlag = source.MetaBlocklistFlag
		obj.ForcedIntroduceFlag = source.MetaForcedIntroduceFlag
		obj.MetaVersion = source.MetaVersion
		obj.CalculatedAt = source.CalculatedAt
	}

	if obj.LengthChars == 0 {
		obj.LengthChars = maxFormLength(forms)
	}
	if obj.BaseDifficulty == 0 {
		obj.BaseDifficulty = fallbackConceptDifficulty(forms)
	}

	return tx.Create(&obj).Error
}

func copySynonyms(tx *gorm.DB, sourceWordID uint64, forms []LexicalForm) (int, error) {
	formsByLang := make(map[string]LexicalForm, len(forms))
	for _, form := range forms {
		formsByLang[form.LangCode] = form
	}

	created := 0
	for _, form := range forms {
		if err := createSynonymIfMissing(tx, form.ID, form.Value, true); err != nil {
			return 0, err
		}
		created++
	}

	var source []base.WordSynonym
	if err := tx.Where("word_id = ?", sourceWordID).Order("id ASC").Find(&source).Error; err != nil {
		return 0, err
	}

	for _, syn := range source {
		form, ok := formsByLang[syn.LangCode]
		if !ok {
			continue
		}
		if err := createSynonymIfMissing(tx, form.ID, syn.SynonymValue, syn.IsPrimary); err != nil {
			return 0, err
		}
		created++
	}
	return created, nil
}

func createSynonymIfMissing(tx *gorm.DB, formID uint64, value string, isPrimary bool) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	obj := LexicalFormSynonym{
		FormID:          formID,
		SynonymValue:    value,
		NormalizedValue: NormalizeValue(value),
		IsPrimary:       isPrimary,
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "form_id"}, {Name: "normalized_value"}},
		DoNothing: true,
	}).Create(&obj).Error
}

func generateDirectionsForConcept(tx *gorm.DB, conceptID uint64, forms []LexicalForm) (int, error) {
	created := 0
	for _, source := range forms {
		for _, target := range forms {
			if source.ID == target.ID || source.LangCode == target.LangCode {
				continue
			}
			obj := TrainingDirection{
				ConceptID:      conceptID,
				SourceFormID:   source.ID,
				TargetFormID:   target.ID,
				SourceLangCode: source.LangCode,
				TargetLangCode: target.LangCode,
				DirectionCode:  source.LangCode + "_" + target.LangCode,
				IsActive:       true,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "concept_id"}, {Name: "source_form_id"}, {Name: "target_form_id"}},
				DoNothing: true,
			}).Create(&obj).Error; err != nil {
				return 0, err
			}
			created++
		}
	}
	return created, nil
}

func publishSnapshotVersion(tx *gorm.DB) (*ContentSnapshotVersion, error) {
	var current ContentSnapshotVersion
	nextVersion := int64(1)
	if err := tx.Where("snapshot_type = ?", directionalSnapshotType).Order("version_code DESC").First(&current).Error; err == nil {
		nextVersion = current.VersionCode + 1
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	checksum, err := calculateSnapshotChecksum(tx, nextVersion)
	if err != nil {
		return nil, err
	}

	if err := tx.Model(&ContentSnapshotVersion{}).
		Where("snapshot_type = ? AND is_active = TRUE", directionalSnapshotType).
		Update("is_active", false).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	snapshot := &ContentSnapshotVersion{
		SnapshotType: directionalSnapshotType,
		VersionCode:  nextVersion,
		Checksum:     checksum,
		IsActive:     true,
		PublishedAt:  &now,
	}
	if err := tx.Create(snapshot).Error; err != nil {
		return nil, err
	}
	return snapshot, nil
}

func calculateSnapshotChecksum(tx *gorm.DB, version int64) (string, error) {
	counts := map[string]int64{}
	for name, model := range map[string]any{
		"languages":      &Language{},
		"categories":     &Category{},
		"concepts":       &LexicalConcept{},
		"forms":          &LexicalForm{},
		"directions":     &TrainingDirection{},
		"direction_meta": &TrainingDirectionMeta{},
		"form_synonyms":  &LexicalFormSynonym{},
		"concept_meta":   &LexicalConceptMeta{},
		"form_meta":      &LexicalFormMeta{},
	} {
		var count int64
		if err := tx.Model(model).Count(&count).Error; err != nil {
			return "", err
		}
		counts[name] = count
	}

	payload := struct {
		SnapshotType string           `json:"snapshot_type"`
		Version      int64            `json:"version"`
		Counts       map[string]int64 `json:"counts"`
		CreatedAt    int64            `json:"created_at"`
	}{
		SnapshotType: directionalSnapshotType,
		Version:      version,
		Counts:       counts,
		CreatedAt:    time.Now().UnixNano(),
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func NormalizeValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func cleanValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func cleanPtr(value *string) *string {
	if value == nil {
		return nil
	}
	clean := strings.TrimSpace(*value)
	if clean == "" {
		return nil
	}
	return &clean
}

func strPtrIfNotEmpty(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func countTokens(value string) int {
	count := len(strings.Fields(value))
	if count < 1 {
		return 1
	}
	return count
}

func maxFormLength(forms []LexicalForm) int {
	maxLen := 0
	for _, form := range forms {
		if l := utf8.RuneCountInString(form.Value); l > maxLen {
			maxLen = l
		}
	}
	return maxLen
}

func fallbackFormScore(form LexicalForm) float64 {
	length := utf8.RuneCountInString(form.Value)
	if length < 1 {
		length = 1
	}
	base := float64(length) * 0.22
	if countTokens(form.Value) > 1 {
		base += 0.45
	}
	switch form.LangCode {
	case "ru":
		base *= 0.90
	case "de":
		base *= 1.10
	}
	return round3(base)
}

func fallbackConceptDifficulty(forms []LexicalForm) float64 {
	if len(forms) == 0 {
		return 1
	}
	total := 0.0
	for _, form := range forms {
		total += fallbackFormScore(form)
	}
	return round3(total/float64(len(forms)) + 1)
}

func round3(value float64) float64 {
	return float64(int(value*1000+0.5)) / 1000
}

func ensureNonZeroID(id uint64, name string) error {
	if id == 0 {
		return fmt.Errorf("%s is empty", name)
	}
	return nil
}
