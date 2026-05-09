export type Category = {
  id: number;
  slug: string;
  name_ru: string;
  name_en: string;
  name_de: string;
  entity: string;
  created_at?: string;
  updated_at?: string;
};

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
  categories: Category[];
  words: Word[];
  words_meta_base: WordMetaBase[];
  word_synonyms: Array<{
    id: number;
    word_id: number;
    lang_code: string;
    synonym_value: string;
    is_primary: boolean;
  }>;
};

export type ImportResponse = {
  processed?: number;
  created?: number;
  updated?: number;
  rejected?: number;
  errors?: Array<{ row?: number; message: string }>;
};

export type RecalculateMetaResponse = {
  processed: number;
  snapshot?: SnapshotVersion;
};

export type AuthResponse = {
  token: string;
  user?: {
    id: number;
    username: string;
    email: string;
    level?: string;
  };
};
