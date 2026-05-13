import { api } from './client';
import { normalizeCategory } from './normalizers';
import type { Category } from '../../types';

export type CategoryPayload = Omit<Category, 'id' | 'created_at' | 'updated_at'>;

export async function fetchCategories(): Promise<Category[]> {
  const { data } = await api.get<unknown[]>('/v1/admin/categories');
  return Array.isArray(data) ? data.map(normalizeCategory) : [];
}

export async function createCategory(payload: CategoryPayload): Promise<Category> {
  const { data } = await api.post<unknown>('/v1/admin/categories', payload);
  return normalizeCategory(data);
}

export async function updateCategory(id: number, payload: CategoryPayload): Promise<Category> {
  const { data } = await api.put<unknown>(`/v1/admin/categories/${id}`, payload);
  return normalizeCategory(data);
}
