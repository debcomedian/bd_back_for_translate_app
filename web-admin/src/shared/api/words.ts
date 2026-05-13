import { api } from './client';
import { normalizeWord } from './normalizers';
import type { Word } from '../../types';

export type WordPayload = {
  lang_code: string;
  word_ru?: string | null;
  word_en?: string | null;
  word_de?: string | null;
  transcription_ru?: string | null;
  transcription_en?: string | null;
  transcription_de?: string | null;
  source_ref?: string | null;
  category_id?: number | null;
  is_active: boolean;
};

export async function fetchWords(): Promise<Word[]> {
  const { data } = await api.get<unknown[]>('/v1/admin/words');
  return Array.isArray(data) ? data.map(normalizeWord) : [];
}

export async function createWord(payload: WordPayload): Promise<Word> {
  const { data } = await api.post<unknown>('/v1/admin/words', payload);
  return normalizeWord(data);
}

export async function updateWord(id: number, payload: WordPayload): Promise<Word> {
  const { data } = await api.put<unknown>(`/v1/admin/words/${id}`, payload);
  return normalizeWord(data);
}
