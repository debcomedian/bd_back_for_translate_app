import { api } from './client';
import type { SnapshotResponse } from '../../types';

export async function fetchSnapshot(): Promise<SnapshotResponse> {
  const { data } = await api.get<SnapshotResponse>('/v1/content/snapshot');
  return data;
}
