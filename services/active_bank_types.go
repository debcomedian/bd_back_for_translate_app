package services

type ActiveBankTargetState struct {
	FromEnglish     []string `json:"from_english,omitempty"`
	StrictValidated []string `json:"strict_validated,omitempty"`
	SoftValidated   []string `json:"soft_validated,omitempty"`
	Completed       []string `json:"completed,omitempty"`
}

type ActiveBankFrequencyEN struct {
	Zipf   float64 `json:"zipf"`
	Bucket string  `json:"bucket"`
}

type ActiveBankFrequency struct {
	EN            ActiveBankFrequencyEN `json:"en"`
	PriorityScore float64               `json:"priority_score"`
}

type ActiveBankRecord struct {
	EnLemma    string                           `json:"en_lemma"`
	Pos        string                           `json:"pos"`
	Glosses    []string                         `json:"glosses,omitempty"`
	Targets    map[string]ActiveBankTargetState `json:"targets"`
	Status     string                           `json:"status"`
	Layer      string                           `json:"layer"`
	Confidence string                           `json:"confidence"`
	Flags      []string                         `json:"flags,omitempty"`
	Source     string                           `json:"source"`
	Frequency  ActiveBankFrequency              `json:"frequency"`
}

type ActiveBankImportReport struct {
	InputPath             string `json:"input_path"`
	Processed             int    `json:"processed"`
	CreatedWords          int    `json:"created_words"`
	UpdatedWords          int    `json:"updated_words"`
	SkippedInvalid        int    `json:"skipped_invalid"`
	SkippedWithoutEnglish int    `json:"skipped_without_english"`
	ImportedWithRu        int    `json:"imported_with_ru"`
	ImportedWithDe        int    `json:"imported_with_de"`
	ImportedWithBoth      int    `json:"imported_with_both"`
	MetaLangUpserts       int    `json:"meta_lang_upserts"`
	MetaBaseUpserts       int    `json:"meta_base_upserts"`
	SynonymsInserted      int    `json:"synonyms_inserted"`
	SnapshotPublished     bool   `json:"snapshot_published"`
	SnapshotVersionCode   int64  `json:"snapshot_version_code,omitempty"`
	SnapshotChecksum      string `json:"snapshot_checksum,omitempty"`
}
