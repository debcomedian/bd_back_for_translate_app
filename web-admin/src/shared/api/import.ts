import { api } from './client';
import { normalizeImportResponse, normalizeRecalculateMetaResponse } from './normalizers';
import type { ImportResponse, RecalculateMetaResponse } from '../../types';

export async function importWords(file: File): Promise<ImportResponse> {
  const formData = new FormData();
  formData.append('file', file);
  const { data } = await api.post<unknown>('/v1/admin/words/import', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
  return normalizeImportResponse(data);
}

export async function recalculateWordsMeta(): Promise<RecalculateMetaResponse> {
  const { data } = await api.post<unknown>('/v1/admin/words/recalculate-meta', { scope: 'all' });
  return normalizeRecalculateMetaResponse(data);
}
