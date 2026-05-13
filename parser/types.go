package main

import "encoding/json"

type Entry struct {
	Word         string        `json:"word"`
	LangCode     string        `json:"lang_code"`
	Pos          string        `json:"pos"`
	Senses       []Sense       `json:"senses,omitempty"`
	Translations []Translation `json:"translations,omitempty"`
}

type Sense struct {
	Glosses []string `json:"glosses,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	ID      string   `json:"id,omitempty"`
}

type Translation struct {
	LangCode string   `json:"lang_code"`
	Word     string   `json:"word"`
	Sense    string   `json:"sense,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type TargetSpec struct {
	LangCode   string
	InputPath  string
	BridgeLang string
	PeerLangs  []string
}

type TargetLemmaInfo struct {
	Lemma            string              `json:"lemma"`
	Key              string              `json:"key"`
	LangCode         string              `json:"lang_code"`
	HasGloss         bool                `json:"has_gloss"`
	POS              []string            `json:"pos,omitempty"`
	BridgeTranslations []string          `json:"bridge_translations,omitempty"`
	PeerTranslations map[string][]string `json:"peer_translations,omitempty"`

	bridgeKeys       map[string]struct{}
}

type CandidateTargetState struct {
	FromEnglish     []string `json:"from_english,omitempty"`
	StrictValidated []string `json:"strict_validated,omitempty"`
	SoftValidated   []string `json:"soft_validated,omitempty"`
	Completed       []string `json:"completed,omitempty"`
}

type Stage31Candidate struct {
	EnLemma     string                          `json:"en_lemma"`
	Pos         string                          `json:"pos"`
	Glosses     []string                        `json:"glosses,omitempty"`
	Targets     map[string]CandidateTargetState `json:"targets"`
	Status      string                          `json:"status"`
	Layer       string                          `json:"layer"`
	Confidence  string                          `json:"confidence"`
	Flags       []string                        `json:"flags,omitempty"`
	Source      string                          `json:"source"`
}

type Stage31Stats struct {
	EnglishInputPath  string         `json:"english_input_path"`
	TargetInputPaths  map[string]string `json:"target_input_paths"`

	EnglishScanned    int            `json:"english_scanned"`
	WantedCounts      map[string]int `json:"wanted_counts"`
	TargetScanned     map[string]int `json:"target_scanned"`
	TargetMatched     map[string]int `json:"target_matched"`

	WrittenCore       int            `json:"written_core"`
	WrittenExtendedAll int           `json:"written_extended_all"`
	StatusCounts      map[string]int `json:"status_counts"`

	SkippedEmptyLine                 int `json:"skipped_empty_line"`
	SkippedInvalidJSON               int `json:"skipped_invalid_json"`
	SkippedByLang                    int `json:"skipped_by_lang"`
	SkippedByPOS                     int `json:"skipped_by_pos"`
	SkippedByLemma                   int `json:"skipped_by_lemma"`
	SkippedByGloss                   int `json:"skipped_by_gloss"`
	SkippedByNoCandidateTranslations int `json:"skipped_by_no_candidate_translations"`
}

type DebugSkippedEntry struct {
	Reason              string                 `json:"reason"`
	Word                string                 `json:"word"`
	NormalizedWord      string                 `json:"normalized_word,omitempty"`
	RawLangCode         string                 `json:"raw_lang_code,omitempty"`
	RawPOS              string                 `json:"raw_pos,omitempty"`
	CanonicalPOS        string                 `json:"canonical_pos,omitempty"`
	Glosses             []string               `json:"glosses,omitempty"`
	Translations        []Translation          `json:"translations,omitempty"`
	RawTargets          map[string][]string    `json:"raw_targets,omitempty"`
	States              map[string]CandidateTargetState `json:"states,omitempty"`
	Note                string                 `json:"note,omitempty"`
}

func marshalLine(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
