package lexicon

type ActiveBankRecord struct {
	EnLemma   string                           `json:"en_lemma"`
	Pos       string                           `json:"pos"`
	Status    string                           `json:"status"`
	Glosses   []string                         `json:"glosses"`
	Targets   map[string]ActiveBankTargetState `json:"targets"`
	Frequency ActiveBankFrequency              `json:"frequency"`
}

type ActiveBankFrequency struct {
	EN ActiveBankFrequencyValue `json:"en"`
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
}
