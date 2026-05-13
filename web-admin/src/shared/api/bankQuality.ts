import { api } from './client';
import type { BankQualityReport } from '../../types';

export async function fetchBankQualityReport(): Promise<BankQualityReport> {
  const { data } = await api.get<BankQualityReport>('/admin/content/bank-quality');
  return data;
}
