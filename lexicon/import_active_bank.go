package lexicon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"bd_back_for_translate_app/services"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ActiveBankImportReport struct {
	InputPath           string `json:"input_path"`
	Processed           int    `json:"processed"`
	SkippedInvalid      int    `json:"skipped_invalid"`
	SkippedWithoutForms int    `json:"skipped_without_forms"`
	Concepts            int    `json:"concepts"`
	Forms               int    `json:"forms"`
	FormMeta            int    `json:"form_meta"`
	ConceptMeta         int    `json:"concept_meta"`
	Synonyms            int    `json:"synonyms"`
	Directions          int    `json:"directions"`
	DirectionMeta       int    `json:"direction_meta"`
	SnapshotPublished   bool   `json:"snapshot_published"`
	SnapshotVersionCode int64  `json:"snapshot_version_code,omitempty"`
	SnapshotChecksum    string `json:"snapshot_checksum,omitempty"`
}

type formInput struct {
	Lang          string
	Value         string
	Transcription *string
	Example       *string
	Primary       bool
}

func ImportActiveBank(db *gorm.DB, inputPath string) (*ActiveBankImportReport, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("open active bank: %w", err)
	}
	defer file.Close()

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

	if err := clearLexiconContent(tx); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := seedKnownLanguages(tx); err != nil {
		tx.Rollback()
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec services.ActiveBankRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			report.SkippedInvalid++
			continue
		}
		report.Processed++

		input := buildFormsFromActiveBank(rec)
		if len(input) < 2 {
			report.SkippedWithoutForms++
			continue
		}

		concept := LexicalConcept{SourceRef: strPtrIfNotEmpty("active_bank:" + strings.TrimSpace(rec.Status)), IsActive: true}
		if err := tx.Create(&concept).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		report.Concepts++

		forms, err := createFormsFromInput(tx, concept.ID, input)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.Forms += len(forms)

		createdMeta, err := createFormMetaFromActiveBank(tx, forms, rec)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.FormMeta += createdMeta

		if err := createConceptMetaFromActiveBank(tx, concept.ID, forms, rec); err != nil {
			tx.Rollback()
			return nil, err
		}
		report.ConceptMeta++

		createdSynonyms, err := createSynonymsFromActiveBank(tx, forms, rec)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.Synonyms += createdSynonyms

		createdDirections, err := generateDirectionsForConcept(tx, concept.ID, forms)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		report.Directions += createdDirections
	}
	if err := scanner.Err(); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("scan active bank: %w", err)
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
	report.SnapshotPublished = true
	report.SnapshotVersionCode = snapshot.VersionCode
	report.SnapshotChecksum = snapshot.Checksum

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return report, nil
}

func seedKnownLanguages(tx *gorm.DB) error {
	items := []Language{
		{Code: "ru", NameRu: "Русский", NameEn: "Russian", IsActive: true},
		{Code: "en", NameRu: "Английский", NameEn: "English", IsActive: true},
		{Code: "de", NameRu: "Немецкий", NameEn: "German", IsActive: true},
	}
	for _, item := range items {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func buildFormsFromActiveBank(rec services.ActiveBankRecord) []formInput {
	items := []formInput{}
	en := normalizeLexeme(rec.EnLemma)
	if en != "" {
		items = append(items, formInput{Lang: "en", Value: en, Example: nilIfEmpty(firstGloss(rec.Glosses)), Primary: true})
	}
	for lang, state := range rec.Targets {
		lang = NormalizeValue(lang)
		if lang == "" || lang == "en" {
			continue
		}
		primary := choosePrimaryTarget(state)
		if primary == "" {
			continue
		}
		items = append(items, formInput{Lang: lang, Value: primary, Primary: true})
	}
	return items
}

func createFormsFromInput(tx *gorm.DB, conceptID uint64, inputs []formInput) ([]LexicalForm, error) {
	forms := make([]LexicalForm, 0, len(inputs))
	for _, input := range inputs {
		if err := ensureLanguage(tx, input.Lang); err != nil {
			return nil, err
		}
		obj := LexicalForm{ConceptID: conceptID, LangCode: input.Lang, Value: input.Value, NormalizedValue: NormalizeValue(input.Value), Transcription: input.Transcription, Example: input.Example, IsPrimary: input.Primary}
		if err := tx.Create(&obj).Error; err != nil {
			return nil, err
		}
		forms = append(forms, obj)
	}
	return forms, nil
}

func ensureLanguage(tx *gorm.DB, code string) error {
	code = NormalizeValue(code)
	if code == "" {
		return fmt.Errorf("empty language code")
	}
	obj := Language{Code: code, NameRu: code, NameEn: code, IsActive: true}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&obj).Error
}

func createFormMetaFromActiveBank(tx *gorm.DB, forms []LexicalForm, rec services.ActiveBankRecord) (int, error) {
	created := 0
	for _, form := range forms {
		meta := LexicalFormMeta{FormID: form.ID, Lemma: strPtrIfNotEmpty(form.Value), LemmaChars: utf8.RuneCountInString(form.Value), TokenCount: countTokens(form.Value), CalculatedAt: time.Now()}
		if form.LangCode == "en" {
			meta.ZipfFrequency = rec.Frequency.EN.Zipf
			meta.FreqBucket = bucketLabelToInt(rec.Frequency.EN.Bucket)
			meta.MultiwordScore = seedMultiwordScore(meta.TokenCount)
			meta.POSScore = seedPOSScore(rec.Pos)
			meta.ConfidencePenalty = seedConfidencePenalty(rec.Status)
		}
		meta.FormScore = fallbackFormScore(form) + meta.MultiwordScore + meta.POSScore + meta.ConfidencePenalty
		if form.LangCode == "en" && rec.Frequency.EN.Zipf > 0 {
			meta.FormScore = seedBaseDifficulty(rec.Frequency.EN.Zipf, meta.LemmaChars, rec.Status)
		}
		if meta.TokenCount < 1 {
			meta.TokenCount = 1
		}
		if err := tx.Create(&meta).Error; err != nil {
			return 0, err
		}
		created++
	}
	return created, nil
}

func createConceptMetaFromActiveBank(tx *gorm.DB, conceptID uint64, forms []LexicalForm, rec services.ActiveBankRecord) error {
	maxChars := maxFormLength(forms)
	meta := LexicalConceptMeta{ConceptID: conceptID, CefrLevel: strPtrIfNotEmpty(seedCEFRFromBucket(rec.Frequency.EN.Bucket)), ImportanceScore: seedImportanceFromZipf(rec.Frequency.EN.Zipf), FreqBucket: bucketLabelToInt(rec.Frequency.EN.Bucket), LengthChars: maxChars, BaseDifficulty: seedBaseDifficulty(rec.Frequency.EN.Zipf, maxChars, rec.Status), MetaVersion: 1, CalculatedAt: time.Now()}
	if meta.BaseDifficulty == 0 {
		meta.BaseDifficulty = fallbackConceptDifficulty(forms)
	}
	return tx.Create(&meta).Error
}

func createSynonymsFromActiveBank(tx *gorm.DB, forms []LexicalForm, rec services.ActiveBankRecord) (int, error) {
	formsByLang := map[string]LexicalForm{}
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
	for lang, state := range rec.Targets {
		form, ok := formsByLang[NormalizeValue(lang)]
		if !ok {
			continue
		}
		primary := choosePrimaryTarget(state)
		for _, value := range collectSynonyms(primary, state) {
			if err := createSynonymIfMissing(tx, form.ID, value, NormalizeValue(value) == NormalizeValue(primary)); err != nil {
				return 0, err
			}
			created++
		}
	}
	return created, nil
}

func collectSynonyms(primary string, state services.ActiveBankTargetState) []string {
	seen := map[string]struct{}{}
	out := []string{}
	add := func(value string) {
		value = normalizeLexeme(value)
		if value == "" {
			return
		}
		key := NormalizeValue(value)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	add(primary)
	for _, group := range [][]string{state.StrictValidated, state.SoftValidated, state.Completed, state.FromEnglish} {
		for _, value := range group {
			add(value)
		}
	}
	return out
}

func choosePrimaryTarget(state services.ActiveBankTargetState) string {
	for _, group := range [][]string{state.StrictValidated, state.SoftValidated, state.Completed, state.FromEnglish} {
		for _, value := range group {
			if normalized := normalizeLexeme(value); normalized != "" {
				return normalized
			}
		}
	}
	return ""
}

func normalizeLexeme(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "ё", "е")
	value = strings.ReplaceAll(value, "Ё", "Е")
	return strings.Join(strings.Fields(value), " ")
}

func nilIfEmpty(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func firstGloss(glosses []string) string {
	if len(glosses) == 0 {
		return ""
	}
	return strings.TrimSpace(glosses[0])
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
	if tokenCount <= 1 {
		return 0
	}
	if tokenCount == 2 {
		return 0.35
	}
	return 0.65
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
	return (0.65*freqInverse + 0.20*lengthScore + 0.15*seedConfidencePenalty(status)) * 10.0
}
