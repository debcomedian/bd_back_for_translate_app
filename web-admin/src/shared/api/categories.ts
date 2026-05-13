import { api } from './client';
import type { Category } from '../../types';
import { pickBoolean, pickNullableNumber, pickNumber, pickString } from './normalizers';

export type CategoryPayload = Omit<Category, 'id' | 'created_at' | 'updated_at'>;

function normalizeCategory(item: unknown): Category {
  return {
    id: pickNumber(item, ['id', 'ID']),
    source_id: pickNullableNumber(item, ['source_id', 'SourceID', 'SourceId']),
    slug: pickString(item, ['slug', 'Slug']),
    name_ru: pickString(item, ['name_ru', 'NameRu']),
    name_en: pickString(item, ['name_en', 'NameEn']),
    name_de: pickString(item, ['name_de', 'NameDe']),
    entity: pickString(item, ['entity', 'Entity'], 'concept'),
    created_at: pickString(item, ['created_at', 'CreatedAt'], ''),
    updated_at: pickString(item, ['updated_at', 'UpdatedAt'], ''),
  };
}

function normalizeCategoryPayload(payload: CategoryPayload): CategoryPayload {
  return {
    ...payload,
    entity: payload.entity || 'concept',
  };
}

export async function fetchCategories(): Promise<Category[]> {
  const { data } = await api.get<unknown[]>('/admin/categories');
  return Array.isArray(data) ? data.map(normalizeCategory) : [];
}

export async function createCategory(payload: CategoryPayload): Promise<Category> {
  const { data } = await api.post<unknown>('/admin/categories', normalizeCategoryPayload(payload));
  return normalizeCategory(data);
}

export async function updateCategory(id: number, payload: CategoryPayload): Promise<Category> {
  const { data } = await api.put<unknown>(`/admin/categories/${id}`, normalizeCategoryPayload(payload));
  return normalizeCategory(data);
}

export function isCategoryActive(item: unknown): boolean {
  return pickBoolean(item, ['is_active', 'IsActive'], true);
}
