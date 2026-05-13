import type {
  Category,
  ImportResponse,
  RecalculateMetaResponse,
  SnapshotResponse,
  SnapshotVersion,
  Word,
  WordMetaBase,
} from '../../types';

function pick<T = unknown>(raw: any, snakeKey: string, pascalKey: string, fallback?: T): T {
  const value = raw?.[snakeKey] ?? raw?.[pascalKey] ?? fallback;
  return value as T;
}

function normalizeNumber(value: unknown): number | undefined {
  if (value === null || value === undefined || value === '') return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function normalizeRequiredNumber(value: unknown, fallback = 0): number {
  return normalizeNumber(value) ?? fallback;
}

function normalizeBoolean(value: unknown, fallback = false): boolean {
  if (typeof value === 'boolean') return value;
  if (typeof value === 'string') return value === 'true';
  if (typeof value === 'number') return value !== 0;
  return fallback;
}

export function normalizeCategory(raw: any): Category {
  return {
    id: normalizeRequiredNumber(pick(raw, 'id', 'ID')),
    slug: pick<string>(raw, 'slug', 'Slug', ''),
    name_ru: pick<string>(raw, 'name_ru', 'NameRu', ''),
    name_en: pick<string>(raw, 'name_en', 'NameEn', ''),
    name_de: pick<string>(raw, 'name_de', 'NameDe', ''),
    entity: pick<string>(raw, 'entity', 'Entity', 'word'),
    created_at: pick<string | undefined>(raw, 'created_at', 'CreatedAt', undefined),
    updated_at: pick<string | undefined>(raw, 'updated_at', 'UpdatedAt', undefined),
  };
}

export function normalizeWordMetaBase(raw: any): WordMetaBase | null {
  if (!raw) return null;

  return {
    word_id: normalizeRequiredNumber(pick(raw, 'word_id', 'WordID')),
    meta_cefr_level: pick<string | null>(raw, 'meta_cefr_level', 'MetaCefrLevel', null),
    meta_importance_score: normalizeRequiredNumber(pick(raw, 'meta_importance_score', 'MetaImportanceScore')),
    meta_freq_bucket: normalizeRequiredNumber(pick(raw, 'meta_freq_bucket', 'MetaFreqBucket')),
    meta_length_chars: normalizeRequiredNumber(pick(raw, 'meta_length_chars', 'MetaLengthChars')),
    meta_base_difficulty: normalizeRequiredNumber(pick(raw, 'meta_base_difficulty', 'MetaBaseDifficulty')),
    meta_required_stage: normalizeNumber(pick(raw, 'meta_required_stage', 'MetaRequiredStage', null)) ?? null,
    meta_blocklist_flag: normalizeBoolean(pick(raw, 'meta_blocklist_flag', 'MetaBlocklistFlag'), false),
    meta_forced_introduce_flag: normalizeBoolean(pick(raw, 'meta_forced_introduce_flag', 'MetaForcedIntroduceFlag'), false),
    meta_version: normalizeRequiredNumber(pick(raw, 'meta_version', 'MetaVersion'), 1),
    calculated_at: pick<string>(raw, 'calculated_at', 'CalculatedAt', ''),
  };
}

export function normalizeWord(raw: any): Word {
  const metaBase = pick<any>(raw, 'meta_base', 'MetaBase', null);

  return {
    id: normalizeRequiredNumber(pick(raw, 'id', 'ID')),
    lang_code: pick<string>(raw, 'lang_code', 'LangCode', ''),
    word_ru: pick<string | null>(raw, 'word_ru', 'WordRu', null),
    word_en: pick<string | null>(raw, 'word_en', 'WordEn', null),
    word_de: pick<string | null>(raw, 'word_de', 'WordDe', null),
    transcription_ru: pick<string | null>(raw, 'transcription_ru', 'TranscriptionRu', null),
    transcription_en: pick<string | null>(raw, 'transcription_en', 'TranscriptionEn', null),
    transcription_de: pick<string | null>(raw, 'transcription_de', 'TranscriptionDe', null),
    source_ref: pick<string | null>(raw, 'source_ref', 'SourceRef', null),
    category_id: normalizeNumber(pick(raw, 'category_id', 'CategoryID', null)) ?? null,
    is_active: normalizeBoolean(pick(raw, 'is_active', 'IsActive'), false),
    created_at: pick<string | undefined>(raw, 'created_at', 'CreatedAt', undefined),
    updated_at: pick<string | undefined>(raw, 'updated_at', 'UpdatedAt', undefined),
    meta_base: normalizeWordMetaBase(metaBase),
  };
}

export function normalizeSnapshotVersion(raw: any): SnapshotVersion | null {
  if (!raw) return null;

  return {
    id: normalizeRequiredNumber(pick(raw, 'id', 'ID')),
    snapshot_type: pick<string>(raw, 'snapshot_type', 'SnapshotType', ''),
    version_code: normalizeRequiredNumber(pick(raw, 'version_code', 'VersionCode')),
    checksum: pick<string>(raw, 'checksum', 'Checksum', ''),
    is_active: normalizeBoolean(pick(raw, 'is_active', 'IsActive'), false),
    published_at: pick<string | null>(raw, 'published_at', 'PublishedAt', null),
    created_at: pick<string | undefined>(raw, 'created_at', 'CreatedAt', undefined),
  };
}

export function normalizeSnapshot(raw: any): SnapshotResponse {
  return {
    snapshot_version: normalizeSnapshotVersion(pick(raw, 'snapshot_version', 'SnapshotVersion', null)),
    categories: (pick<any[]>(raw, 'categories', 'Categories', []) ?? []).map(normalizeCategory),
    words: (pick<any[]>(raw, 'words', 'Words', []) ?? []).map(normalizeWord),
    words_meta_base: (pick<any[]>(raw, 'words_meta_base', 'WordsMetaBase', []) ?? []).map(normalizeWordMetaBase).filter(Boolean) as WordMetaBase[],
    word_synonyms: (pick<any[]>(raw, 'word_synonyms', 'WordSynonyms', []) ?? []).map((item) => ({
      id: normalizeRequiredNumber(pick(item, 'id', 'ID')),
      word_id: normalizeRequiredNumber(pick(item, 'word_id', 'WordID')),
      lang_code: pick<string>(item, 'lang_code', 'LangCode', ''),
      synonym_value: pick<string>(item, 'synonym_value', 'SynonymValue', ''),
      is_primary: normalizeBoolean(pick(item, 'is_primary', 'IsPrimary'), false),
    })),
  };
}

export function normalizeImportResponse(raw: any): ImportResponse {
  return {
    processed: normalizeNumber(pick(raw, 'processed', 'Processed')),
    created: normalizeNumber(pick(raw, 'created', 'Created')),
    updated: normalizeNumber(pick(raw, 'updated', 'Updated')),
    rejected: normalizeNumber(pick(raw, 'rejected', 'Rejected')),
    errors: pick<ImportResponse['errors']>(raw, 'errors', 'Errors', undefined),
  };
}

export function normalizeRecalculateMetaResponse(raw: any): RecalculateMetaResponse {
  return {
    processed: normalizeRequiredNumber(pick(raw, 'processed', 'Processed')),
    snapshot: normalizeSnapshotVersion(pick(raw, 'snapshot', 'Snapshot', null)) ?? undefined,
  };
}
