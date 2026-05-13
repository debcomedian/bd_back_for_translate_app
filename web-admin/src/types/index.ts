export type Category = {
  id: number;
  source_id?: number | null;
  slug: string;
  name_ru: string;
  name_en: string;
  name_de: string;
  entity: string;
  created_at?: string;
  updated_at?: string;
};

export type Language = {
  code: string;
  name_ru: string;
  name_en: string;
  is_active: boolean;
  created_at?: string;
};

export type LexicalConcept = {
  id: number;
  source_word_id?: number | null;
  category_id?: number | null;
  source_ref?: string | null;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
};

export type LexicalForm = {
  id: number;
  concept_id: number;
  lang_code: string;
  value: string;
  normalized_value?: string;
  transcription?: string | null;
  example?: string | null;
  is_primary: boolean;
  created_at?: string;
  updated_at?: string;
};

export type ConceptMeta = {
  concept_id: number;
  cefr_level?: string | null;
  importance_score: number;
  freq_bucket: number;
  length_chars: number;
  base_difficulty: number;
  required_stage?: number | null;
  blocklist_flag: boolean;
  forced_introduce_flag: boolean;
  meta_version: number;
  calculated_at?: string;
};

export type FormMeta = {
  form_id: number;
  lemma?: string | null;
  lemma_chars: number;
  token_count: number;
  zipf_frequency: number;
  freq_bucket: number;
  orthography_score: number;
  multiword_score: number;
  pos_score: number;
  confidence_penalty: number;
  form_score: number;
  calculated_at?: string;
};

export type FormSynonym = {
  id: number;
  form_id: number;
  synonym_value: string;
  normalized_value?: string;
  is_primary: boolean;
  created_at?: string;
};

export type TrainingDirection = {
  direction_id: number;
  id: number;
  concept_id: number;
  direction_code: string;
  source_lang_code: string;
  target_lang_code: string;
  source_form_id: number;
  source_value: string;
  source_normalized_value?: string;
  source_transcription?: string | null;
  target_form_id: number;
  target_value: string;
  target_normalized_value?: string;
  target_transcription?: string | null;
  category_id?: number | null;
  category_slug?: string | null;
  category_name_ru?: string | null;
  category_name_en?: string | null;
  category_name_de?: string | null;
  cefr_level?: string | null;
  importance_score?: number | null;
  concept_base_difficulty?: number | null;
  source_form_score?: number | null;
  target_form_score?: number | null;
  direction_bias?: number | null;
  synonym_relief?: number | null;
  final_difficulty?: number | null;
  is_active: boolean;
  created_at?: string;
};

export type SnapshotVersion = {
  id: number;
  snapshot_type: string;
  version_code: number;
  checksum: string;
  is_active: boolean;
  published_at?: string | null;
  created_at?: string;
};

export type SnapshotResponse = {
  snapshot_version?: SnapshotVersion | null;
  languages: Language[];
  categories: Category[];
  concepts: LexicalConcept[];
  forms: LexicalForm[];
  concept_meta: ConceptMeta[];
  form_meta: FormMeta[];
  form_synonyms: FormSynonym[];
  directions: TrainingDirection[];
};

export type ImportResponse = {
  processed?: number;
  concepts?: number;
  forms?: number;
  directions?: number;
  snapshot_version_code?: number;
  created?: number;
  updated?: number;
  rejected?: number;
  errors?: Array<{ row?: number; message: string }>;
  snapshot?: SnapshotVersion;
};

export type RecalculateMetaResponse = {
  processed: number;
  directions?: number;
  snapshot?: SnapshotVersion;
};

export type BankQualityReport = {
  generated_at?: string;
  input_path?: string;
  total_rows?: number;
  status_counts?: Record<string, number>;
  bucket_counts?: Record<string, number>;
  full_trilingual_count?: number;
  full_trilingual_percent?: number;
  half_validated_percent?: number;
  tail_percent?: number;
  unsafe_candidates_count?: number;
  wide_synonym_candidates_count?: number;
  gate_passed?: boolean;
  gate_errors?: string[];
  gate_warnings?: string[];
  unsafe_candidates?: unknown[];
  wide_synonym_candidates?: unknown[];
  [key: string]: unknown;
};

export type AuthResponse = {
  token: string;
  role?: string;
  login?: string;
  user?: {
    id: number;
    username: string;
    email: string;
    level?: string;
  };
};

// Legacy types kept only so old, currently unused editor files keep compiling.
export type Word = {
  id: number;
  lang_code: 'ru' | 'en' | 'de' | string;
  word_ru?: string | null;
  word_en?: string | null;
  word_de?: string | null;
  transcription_ru?: string | null;
  transcription_en?: string | null;
  transcription_de?: string | null;
  source_ref?: string | null;
  category_id?: number | null;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
  meta_base?: WordMetaBase | null;
};

export type WordMetaBase = {
  word_id: number;
  meta_cefr_level?: string | null;
  meta_importance_score: number;
  meta_freq_bucket: number;
  meta_length_chars: number;
  meta_base_difficulty: number;
  meta_required_stage?: number | null;
  meta_blocklist_flag: boolean;
  meta_forced_introduce_flag: boolean;
  meta_version: number;
  calculated_at: string;
};
