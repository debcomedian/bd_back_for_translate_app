package lexicon

import "time"

type Language struct {
	Code      string    `json:"code" gorm:"primaryKey;type:text"`
	NameRu    string    `json:"name_ru" gorm:"type:text;not null"`
	NameEn    string    `json:"name_en" gorm:"type:text;not null"`
	IsActive  bool      `json:"is_active" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (Language) TableName() string { return "lexicon.languages" }

type Category struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	SourceID  *uint64   `json:"source_id" gorm:"uniqueIndex:idx_categories_source_id"`
	Slug      string    `json:"slug" gorm:"type:text;not null;unique"`
	NameRu    string    `json:"name_ru" gorm:"type:text;not null"`
	NameEn    string    `json:"name_en" gorm:"type:text;not null"`
	NameDe    string    `json:"name_de" gorm:"type:text;not null"`
	Entity    string    `json:"entity" gorm:"type:text;not null;default:concept"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (Category) TableName() string { return "lexicon.categories" }

type LexicalConcept struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`
	SourceWordID *uint64   `json:"source_word_id" gorm:"uniqueIndex:idx_lexical_concepts_source_word_id"`
	CategoryID   *uint64   `json:"category_id" gorm:"index:idx_lexical_concepts_category_id"`
	SourceRef    *string   `json:"source_ref" gorm:"type:text;index:idx_lexical_concepts_source_ref"`
	IsActive     bool      `json:"is_active" gorm:"not null;default:true;index:idx_lexical_concepts_active"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"not null;default:now()"`

	Category *Category               `json:"category,omitempty" gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL;"`
	Forms    []LexicalForm           `json:"forms,omitempty" gorm:"foreignKey:ConceptID;constraint:OnDelete:CASCADE;"`
	Meta     *LexicalConceptMeta     `json:"meta,omitempty" gorm:"foreignKey:ConceptID;constraint:OnDelete:CASCADE;"`
	Dirs     []TrainingDirection     `json:"directions,omitempty" gorm:"foreignKey:ConceptID;constraint:OnDelete:CASCADE;"`
	Progress []UserDirectionProgress `json:"progress,omitempty" gorm:"foreignKey:ConceptID;constraint:OnDelete:CASCADE;"`
}

func (LexicalConcept) TableName() string { return "lexicon.lexical_concepts" }

type LexicalForm struct {
	ID              uint64    `json:"id" gorm:"primaryKey"`
	ConceptID       uint64    `json:"concept_id" gorm:"not null;uniqueIndex:idx_lexical_forms_unique_value_per_concept_lang;uniqueIndex:idx_lexical_forms_one_primary_per_concept_lang;index:idx_lexical_forms_concept_id"`
	LangCode        string    `json:"lang_code" gorm:"type:text;not null;uniqueIndex:idx_lexical_forms_unique_value_per_concept_lang;uniqueIndex:idx_lexical_forms_one_primary_per_concept_lang;index:idx_lexical_forms_lang_code"`
	Value           string    `json:"value" gorm:"type:text;not null;index:idx_lexical_forms_value"`
	NormalizedValue string    `json:"normalized_value" gorm:"type:text;not null;uniqueIndex:idx_lexical_forms_unique_value_per_concept_lang;index:idx_lexical_forms_normalized_value"`
	Transcription   *string   `json:"transcription" gorm:"type:text"`
	Example         *string   `json:"example" gorm:"type:text"`
	IsPrimary       bool      `json:"is_primary" gorm:"not null;default:true"`
	CreatedAt       time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"not null;default:now()"`

	Concept  LexicalConcept       `json:"concept,omitempty" gorm:"foreignKey:ConceptID;references:ID;constraint:OnDelete:CASCADE;"`
	Meta     *LexicalFormMeta     `json:"meta,omitempty" gorm:"foreignKey:FormID;constraint:OnDelete:CASCADE;"`
	Synonyms []LexicalFormSynonym `json:"synonyms,omitempty" gorm:"foreignKey:FormID;constraint:OnDelete:CASCADE;"`
}

func (LexicalForm) TableName() string { return "lexicon.lexical_forms" }

type LexicalFormMeta struct {
	FormID            uint64    `json:"form_id" gorm:"primaryKey"`
	Lemma             *string   `json:"lemma" gorm:"type:text"`
	LemmaChars        int       `json:"lemma_chars" gorm:"not null;default:0"`
	TokenCount        int       `json:"token_count" gorm:"not null;default:1"`
	ZipfFrequency     float64   `json:"zipf_frequency" gorm:"not null;default:0"`
	FreqBucket        int       `json:"freq_bucket" gorm:"not null;default:0;index:idx_lexical_form_meta_freq_bucket"`
	OrthographyScore  float64   `json:"orthography_score" gorm:"not null;default:0"`
	MultiwordScore    float64   `json:"multiword_score" gorm:"not null;default:0"`
	POSScore          float64   `json:"pos_score" gorm:"column:pos_score;not null;default:0"`
	ConfidencePenalty float64   `json:"confidence_penalty" gorm:"not null;default:0"`
	FormScore         float64   `json:"form_score" gorm:"not null;default:0;index:idx_lexical_form_meta_score"`
	CalculatedAt      time.Time `json:"calculated_at" gorm:"not null;default:now()"`

	Form LexicalForm `json:"form,omitempty" gorm:"foreignKey:FormID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (LexicalFormMeta) TableName() string { return "lexicon.lexical_form_meta" }

type LexicalConceptMeta struct {
	ConceptID           uint64    `json:"concept_id" gorm:"primaryKey"`
	CefrLevel           *string   `json:"cefr_level" gorm:"type:text;index:idx_lexical_concept_meta_cefr"`
	ImportanceScore     int       `json:"importance_score" gorm:"not null;default:0"`
	FreqBucket          int       `json:"freq_bucket" gorm:"not null;default:0;index:idx_lexical_concept_meta_freq_bucket"`
	LengthChars         int       `json:"length_chars" gorm:"not null;default:0"`
	BaseDifficulty      float64   `json:"base_difficulty" gorm:"not null;default:0;index:idx_lexical_concept_meta_difficulty"`
	RequiredStage       *int      `json:"required_stage"`
	BlocklistFlag       bool      `json:"blocklist_flag" gorm:"not null;default:false;index:idx_lexical_concept_meta_blocklist"`
	ForcedIntroduceFlag bool      `json:"forced_introduce_flag" gorm:"not null;default:false"`
	MetaVersion         int       `json:"meta_version" gorm:"not null;default:1"`
	CalculatedAt        time.Time `json:"calculated_at" gorm:"not null;default:now()"`

	Concept LexicalConcept `json:"concept,omitempty" gorm:"foreignKey:ConceptID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (LexicalConceptMeta) TableName() string { return "lexicon.lexical_concept_meta" }

type LexicalFormSynonym struct {
	ID              uint64    `json:"id" gorm:"primaryKey"`
	FormID          uint64    `json:"form_id" gorm:"not null;uniqueIndex:idx_lexical_form_synonyms_unique;index:idx_lexical_form_synonyms_form_id"`
	SynonymValue    string    `json:"synonym_value" gorm:"type:text;not null"`
	NormalizedValue string    `json:"normalized_value" gorm:"type:text;not null;uniqueIndex:idx_lexical_form_synonyms_unique;index:idx_lexical_form_synonyms_normalized_value"`
	IsPrimary       bool      `json:"is_primary" gorm:"not null;default:false"`
	CreatedAt       time.Time `json:"created_at" gorm:"not null;default:now()"`

	Form LexicalForm `json:"form,omitempty" gorm:"foreignKey:FormID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (LexicalFormSynonym) TableName() string { return "lexicon.lexical_form_synonyms" }

type TrainingDirection struct {
	ID             uint64    `json:"id" gorm:"primaryKey"`
	ConceptID      uint64    `json:"concept_id" gorm:"not null;uniqueIndex:idx_training_directions_unique_forms;index:idx_training_directions_concept_id"`
	SourceFormID   uint64    `json:"source_form_id" gorm:"not null;uniqueIndex:idx_training_directions_unique_forms"`
	TargetFormID   uint64    `json:"target_form_id" gorm:"not null;uniqueIndex:idx_training_directions_unique_forms"`
	SourceLangCode string    `json:"source_lang_code" gorm:"type:text;not null;index:idx_training_directions_source_target"`
	TargetLangCode string    `json:"target_lang_code" gorm:"type:text;not null;index:idx_training_directions_source_target"`
	DirectionCode  string    `json:"direction_code" gorm:"type:text;not null;index:idx_training_directions_code"`
	IsActive       bool      `json:"is_active" gorm:"not null;default:true;index:idx_training_directions_active"`
	CreatedAt      time.Time `json:"created_at" gorm:"not null;default:now()"`

	Concept    LexicalConcept         `json:"concept,omitempty" gorm:"foreignKey:ConceptID;references:ID;constraint:OnDelete:CASCADE;"`
	SourceForm LexicalForm            `json:"source_form,omitempty" gorm:"foreignKey:SourceFormID;references:ID;constraint:OnDelete:CASCADE;"`
	TargetForm LexicalForm            `json:"target_form,omitempty" gorm:"foreignKey:TargetFormID;references:ID;constraint:OnDelete:CASCADE;"`
	Meta       *TrainingDirectionMeta `json:"meta,omitempty" gorm:"foreignKey:DirectionID;constraint:OnDelete:CASCADE;"`
}

func (TrainingDirection) TableName() string { return "lexicon.training_directions" }

type TrainingDirectionMeta struct {
	DirectionID           uint64    `json:"direction_id" gorm:"primaryKey"`
	SourceFormScore       float64   `json:"source_form_score" gorm:"not null;default:0"`
	TargetFormScore       float64   `json:"target_form_score" gorm:"not null;default:0"`
	ConceptBaseDifficulty float64   `json:"concept_base_difficulty" gorm:"not null;default:0"`
	DirectionBias         float64   `json:"direction_bias" gorm:"not null;default:0"`
	SynonymRelief         float64   `json:"synonym_relief" gorm:"not null;default:0"`
	FinalDifficulty       float64   `json:"final_difficulty" gorm:"not null;default:0;index:idx_training_direction_meta_difficulty"`
	MetaVersion           int       `json:"meta_version" gorm:"not null;default:1;index:idx_training_direction_meta_version"`
	CalculatedAt          time.Time `json:"calculated_at" gorm:"not null;default:now()"`

	Direction TrainingDirection `json:"direction,omitempty" gorm:"foreignKey:DirectionID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (TrainingDirectionMeta) TableName() string { return "lexicon.training_direction_meta" }

type ContentSnapshotVersion struct {
	ID           uint64     `json:"id" gorm:"primaryKey"`
	SnapshotType string     `json:"snapshot_type" gorm:"type:text;not null;uniqueIndex:idx_content_snapshot_versions_type_version;index:idx_content_snapshot_versions_type_active"`
	VersionCode  int64      `json:"version_code" gorm:"not null;uniqueIndex:idx_content_snapshot_versions_type_version"`
	Checksum     string     `json:"checksum" gorm:"type:text;not null"`
	IsActive     bool       `json:"is_active" gorm:"not null;default:false;index:idx_content_snapshot_versions_type_active"`
	PublishedAt  *time.Time `json:"published_at"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null;default:now()"`
}

func (ContentSnapshotVersion) TableName() string { return "lexicon.content_snapshot_versions" }

type Attempt struct {
	ID             uint64    `json:"id" gorm:"primaryKey"`
	UserID         uint64    `json:"user_id" gorm:"not null;index:idx_attempts_user_id;index:idx_attempts_user_direction;index:idx_attempts_user_attempted_at"`
	DirectionID    uint64    `json:"direction_id" gorm:"not null;index:idx_attempts_direction_id;index:idx_attempts_user_direction"`
	ConceptID      uint64    `json:"concept_id" gorm:"not null;index:idx_attempts_concept_id"`
	SourceFormID   uint64    `json:"source_form_id" gorm:"not null"`
	TargetFormID   uint64    `json:"target_form_id" gorm:"not null"`
	DirectionCode  string    `json:"direction_code" gorm:"type:text;not null;index:idx_attempts_direction_code"`
	PromptValue    string    `json:"prompt_value" gorm:"type:text;not null"`
	ExpectedValue  string    `json:"expected_value" gorm:"type:text;not null"`
	ResponseValue  *string   `json:"response_value" gorm:"type:text"`
	Result         string    `json:"result" gorm:"type:text;not null"`
	ResponseTimeMS int       `json:"response_time_ms" gorm:"not null;default:0"`
	DeviceID       string    `json:"device_id" gorm:"type:text;not null"`
	AttemptedAt    time.Time `json:"attempted_at" gorm:"not null;index:idx_attempts_attempted_at;index:idx_attempts_user_attempted_at"`
	SyncedAt       time.Time `json:"synced_at" gorm:"not null;default:now()"`
}

func (Attempt) TableName() string { return "lexicon.attempts" }

type UserDirectionProgress struct {
	UserID         uint64     `json:"user_id" gorm:"primaryKey;index:idx_user_direction_progress_due;index:idx_user_direction_progress_box;index:idx_user_direction_progress_code;index:idx_user_direction_progress_concept"`
	DirectionID    uint64     `json:"direction_id" gorm:"primaryKey"`
	ConceptID      uint64     `json:"concept_id" gorm:"not null;index:idx_user_direction_progress_concept"`
	DirectionCode  string     `json:"direction_code" gorm:"type:text;not null;index:idx_user_direction_progress_code"`
	Box            int        `json:"box" gorm:"not null;default:0;index:idx_user_direction_progress_box"`
	RepeatCount    int        `json:"repeat_count" gorm:"not null;default:0"`
	CorrectCount   int        `json:"correct_count" gorm:"not null;default:0"`
	IncorrectCount int        `json:"incorrect_count" gorm:"not null;default:0"`
	MasteryScore   float64    `json:"mastery_score" gorm:"not null;default:0"`
	LastSeenAt     *time.Time `json:"last_seen_at"`
	NextDue        *time.Time `json:"next_due" gorm:"index:idx_user_direction_progress_due"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"not null;default:now()"`
}

func (UserDirectionProgress) TableName() string { return "lexicon.user_direction_progress" }

type SyncEvent struct {
	ID               uint64     `json:"id" gorm:"primaryKey"`
	EventID          string     `json:"event_id" gorm:"type:text;not null;unique"`
	UserID           uint64     `json:"user_id" gorm:"not null;index:idx_sync_events_user_id;index:idx_sync_events_user_received"`
	EntityType       string     `json:"entity_type" gorm:"type:text;not null;index:idx_sync_events_entity"`
	EntityID         *uint64    `json:"entity_id" gorm:"index:idx_sync_events_entity"`
	EventType        string     `json:"event_type" gorm:"type:text;not null"`
	PayloadJSON      []byte     `json:"payload_json" gorm:"type:jsonb;not null"`
	ClientCreatedAt  time.Time  `json:"client_created_at" gorm:"not null"`
	DeviceID         string     `json:"device_id" gorm:"type:text;not null;index:idx_sync_events_device_id"`
	ServerReceivedAt time.Time  `json:"server_received_at" gorm:"not null;default:now();index:idx_sync_events_user_received"`
	ProcessedAt      *time.Time `json:"processed_at"`
	Status           string     `json:"status" gorm:"type:text;not null;default:received;index:idx_sync_events_status"`
	CreatedAt        time.Time  `json:"created_at" gorm:"not null;default:now()"`
}

func (SyncEvent) TableName() string { return "lexicon.sync_events" }

type AuditLog struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	AdminUserID uint64    `json:"admin_user_id" gorm:"not null;index:idx_audit_log_admin_user_id"`
	ActionType  string    `json:"action_type" gorm:"type:text;not null"`
	EntityType  string    `json:"entity_type" gorm:"type:text;not null;index:idx_audit_log_entity"`
	EntityID    *uint64   `json:"entity_id" gorm:"index:idx_audit_log_entity"`
	PayloadJSON []byte    `json:"payload_json" gorm:"type:jsonb;not null;default:'{}'::jsonb"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now();index:idx_audit_log_created_at"`
}

func (AuditLog) TableName() string { return "lexicon.audit_log" }

type TrainingDirectionView struct {
	DirectionID           uint64    `json:"direction_id"`
	ConceptID             uint64    `json:"concept_id"`
	DirectionCode         string    `json:"direction_code"`
	SourceLangCode        string    `json:"source_lang_code"`
	TargetLangCode        string    `json:"target_lang_code"`
	SourceFormID          uint64    `json:"source_form_id"`
	SourceValue           string    `json:"source_value"`
	SourceNormalizedValue string    `json:"source_normalized_value"`
	SourceTranscription   *string   `json:"source_transcription"`
	TargetFormID          uint64    `json:"target_form_id"`
	TargetValue           string    `json:"target_value"`
	TargetNormalizedValue string    `json:"target_normalized_value"`
	TargetTranscription   *string   `json:"target_transcription"`
	CategoryID            *uint64   `json:"category_id"`
	CategorySlug          *string   `json:"category_slug"`
	CategoryNameRu        *string   `json:"category_name_ru"`
	CategoryNameEn        *string   `json:"category_name_en"`
	CategoryNameDe        *string   `json:"category_name_de"`
	CefrLevel             *string   `json:"cefr_level"`
	ImportanceScore       *int      `json:"importance_score"`
	ConceptBaseDifficulty *float64  `json:"concept_base_difficulty"`
	SourceFormScore       *float64  `json:"source_form_score"`
	TargetFormScore       *float64  `json:"target_form_score"`
	DirectionBias         *float64  `json:"direction_bias"`
	SynonymRelief         *float64  `json:"synonym_relief"`
	FinalDifficulty       *float64  `json:"final_difficulty"`
	IsActive              bool      `json:"is_active"`
	CreatedAt             time.Time `json:"created_at"`
}

func (TrainingDirectionView) TableName() string { return "lexicon.training_direction_view" }
