import { api } from './client';
import type { ImportResponse, RecalculateMetaResponse } from '../../types';

export async function importWords(_file?: File | null): Promise<ImportResponse> {
  const { data } = await api.post<ImportResponse>('/admin/content/import-active-bank');
  return data;
}

export async function recalculateWordsMeta(): Promise<RecalculateMetaResponse> {
  const { data } = await api.post<RecalculateMetaResponse>('/admin/content/recalculate-directions');
  return data;
}

export async function rebuildFromCurrent(): Promise<ImportResponse> {
  const { data } = await api.post<ImportResponse>('/admin/content/rebuild-from-current');
  return data;
}
