import { api } from './client';
import type { TrainingDirection } from '../../types';
import {
  pickBoolean,
  pickNullableNumber,
  pickNullableString,
  pickNumber,
  pickString,
} from './normalizers';

export function normalizeDirection(item: unknown): TrainingDirection {
  const directionId = pickNumber(item, ['direction_id', 'DirectionID', 'DirectionId', 'id', 'ID']);
  return {
    direction_id: directionId,
    id: directionId,
    concept_id: pickNumber(item, ['concept_id', 'ConceptID', 'ConceptId']),
    direction_code: pickString(item, ['direction_code', 'DirectionCode']),
    source_lang_code: pickString(item, ['source_lang_code', 'SourceLangCode']),
    target_lang_code: pickString(item, ['target_lang_code', 'TargetLangCode']),
    source_form_id: pickNumber(item, ['source_form_id', 'SourceFormID', 'SourceFormId']),
    source_value: pickString(item, ['source_value', 'SourceValue']),
    source_normalized_value: pickString(item, ['source_normalized_value', 'SourceNormalizedValue'], ''),
    source_transcription: pickNullableString(item, ['source_transcription', 'SourceTranscription']),
    target_form_id: pickNumber(item, ['target_form_id', 'TargetFormID', 'TargetFormId']),
    target_value: pickString(item, ['target_value', 'TargetValue']),
    target_normalized_value: pickString(item, ['target_normalized_value', 'TargetNormalizedValue'], ''),
    target_transcription: pickNullableString(item, ['target_transcription', 'TargetTranscription']),
    category_id: pickNullableNumber(item, ['category_id', 'CategoryID', 'CategoryId']),
    category_slug: pickNullableString(item, ['category_slug', 'CategorySlug']),
    category_name_ru: pickNullableString(item, ['category_name_ru', 'CategoryNameRu']),
    category_name_en: pickNullableString(item, ['category_name_en', 'CategoryNameEn']),
    category_name_de: pickNullableString(item, ['category_name_de', 'CategoryNameDe']),
    cefr_level: pickNullableString(item, ['cefr_level', 'CefrLevel', 'CEFRLevel']),
    importance_score: pickNullableNumber(item, ['importance_score', 'ImportanceScore']),
    concept_base_difficulty: pickNullableNumber(item, ['concept_base_difficulty', 'ConceptBaseDifficulty']),
    source_form_score: pickNullableNumber(item, ['source_form_score', 'SourceFormScore']),
    target_form_score: pickNullableNumber(item, ['target_form_score', 'TargetFormScore']),
    direction_bias: pickNullableNumber(item, ['direction_bias', 'DirectionBias']),
    synonym_relief: pickNullableNumber(item, ['synonym_relief', 'SynonymRelief']),
    final_difficulty: pickNullableNumber(item, ['final_difficulty', 'FinalDifficulty']),
    is_active: pickBoolean(item, ['is_active', 'IsActive'], true),
    created_at: pickString(item, ['created_at', 'CreatedAt'], ''),
  };
}

function directionItemsFromResponse(data: unknown): unknown[] {
  if (Array.isArray(data)) return data;

  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>;

    if (Array.isArray(obj.items)) return obj.items;
    if (Array.isArray(obj.directions)) return obj.directions;
    if (Array.isArray(obj.data)) return obj.data;
  }

  return [];
}

export async function fetchDirections(): Promise<TrainingDirection[]> {
  const { data } = await api.get<unknown>('/admin/directions');
  return directionItemsFromResponse(data).map(normalizeDirection);
}
