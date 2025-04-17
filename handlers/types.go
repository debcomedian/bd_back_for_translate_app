package handlers

type Word struct {
	ID              int    `json:"id"`
	WordRu          string `json:"word_ru"`
	WordEn          string `json:"word_en"`
	WordDe          string `json:"word_de"`
	TranscriptionRu string `json:"transcription_ru"`
	TranscriptionEn string `json:"transcription_en"`
	TranscriptionDe string `json:"transcription_de"`
	AudioRu         []byte `json:"audio_ru"`
	AudioEn         []byte `json:"audio_en"`
	AudioDe         []byte `json:"audio_de"`
	CategoryID      int    `json:"category_id"`
	TypeRu          string `json:"type_ru"`
	TypeEn          string `json:"type_en"`
	TypeDe          string `json:"type_de"`
	Status          string `json:"status"`
}

type ReadingText struct {
	ID              int    `json:"id"`
	TitleRu         string `json:"title_ru"`
	TitleEn         string `json:"title_en"`
	TitleDe         string `json:"title_de"`
	ContentRu       string `json:"content_ru"`
	ContentEn       string `json:"content_en"`
	ContentDe       string `json:"content_de"`
	TranscriptionRu string `json:"transcription_ru"`
	TranscriptionEn string `json:"transcription_en"`
	TranscriptionDe string `json:"transcription_de"`
	AudioRu         []byte `json:"audio_ru"`
	AudioEn         []byte `json:"audio_en"`
	AudioDe         []byte `json:"audio_de"`
	CategoryID      int    `json:"category_id"`
}

type Category struct {
	ID     int    `json:"id"`
	NameEn string `json:"name_en"`
	NameRu string `json:"name_ru"`
	NameDe string `json:"name_de"`
	Type   string `json:"type"`
	Entity string `json:"entity"`
}

type Grammar struct {
	ID            int    `json:"id"`
	TitleRu       string `json:"title_ru"`
	TitleEn       string `json:"title_en"`
	TitleDe       string `json:"title_de"`
	DescriptionRu string `json:"description_ru"`
	DescriptionEn string `json:"description_en"`
	DescriptionDe string `json:"description_de"`
	Language      string `json:"language"`
}

type GrammarRules struct {
	ID                int    `json:"id"`
	GrammarID         int    `json:"grammar_id"`
	RuleNameRu        string `json:"rule_name_ru"`
	RuleNameEn        string `json:"rule_name_en"`
	RuleNameDe        string `json:"rule_name_de"`
	RuleDescriptionRu string `json:"rule_description_ru"`
	RuleDescriptionEn string `json:"rule_description_en"`
	RuleDescriptionDe string `json:"rule_description_de"`
}

type GrammarExamples struct {
	ID        int    `json:"id"`
	RuleID    int    `json:"rule_id"`
	ExampleRu string `json:"example_ru"`
	ExampleEn string `json:"example_en"`
	ExampleDe string `json:"example_de"`
}

type GrammarExceptions struct {
	ID            int    `json:"id"`
	RuleID        int    `json:"rule_id"`
	DescriptionRu string `json:"description_ru"`
	DescriptionEn string `json:"description_en"`
	DescriptionDe string `json:"description_de"`
	ExplanationRu string `json:"explanation_ru"`
	ExplanationEn string `json:"explanation_en"`
	ExplanationDe string `json:"explanation_de"`
}
