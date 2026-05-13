import { api } from './client';
import { normalizeSnapshot } from './normalizers';
import type { SnapshotResponse } from '../../types';

export async function fetchSnapshot(): Promise<SnapshotResponse> {
  const { data } = await api.get<unknown>('/v1/content/snapshot');
  return normalizeSnapshot(data);
}
