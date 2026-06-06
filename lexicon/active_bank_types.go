package lexicon

type ActiveBankRecord struct {
	EnLemma       string                           `json:"en_lemma"`
	Pos           string                           `json:"pos"`
	Status        string                           `json:"status"`
	Layer         string                           `json:"layer,omitempty"`
	Confidence    string                           `json:"confidence,omitempty"`
	Flags         []string                         `json:"flags,omitempty"`
	Glosses       []string                         `json:"glosses"`
	Targets       map[string]ActiveBankTargetState `json:"targets"`
	Frequency     ActiveBankFrequency              `json:"frequency"`
	BankSelection *ActiveBankSelection             `json:"bank_selection,omitempty"`
	QualityReview *ActiveBankQualityReview         `json:"quality_review,omitempty"`
}

type ActiveBankFrequency struct {
	EN            ActiveBankFrequencyValue `json:"en"`
	PriorityScore float64                  `json:"priority_score,omitempty"`
}

type ActiveBankFrequencyValue struct {
	Zipf          float64 `json:"zipf"`
	Bucket        string  `json:"bucket"`
	PriorityScore float64 `json:"priority_score,omitempty"`
}

type ActiveBankTargetState struct {
	StrictValidated []string `json:"strict_validated"`
	SoftValidated   []string `json:"soft_validated"`
	Completed       []string `json:"completed"`
	FromEnglish     []string `json:"from_english"`
	Candidates      []string `json:"candidates,omitempty"`
	Synonyms        []string `json:"synonyms,omitempty"`
}

type ActiveBankSelection struct {
	SelectedAs     string  `json:"selected_as,omitempty"`
	IsFunctionWord bool    `json:"is_function_word,omitempty"`
	DuplicateOf    *string `json:"duplicate_of,omitempty"`
}

type ActiveBankQualityReview struct {
	ReviewStatus        string   `json:"review_status,omitempty"`
	Flags               []string `json:"flags,omitempty"`
	BlockingFlags       []string `json:"blocking_flags,omitempty"`
	ContentFlags        []string `json:"content_flags,omitempty"`
	RequiredTargetLangs []string `json:"required_target_langs,omitempty"`
}
