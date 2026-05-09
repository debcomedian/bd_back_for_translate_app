import axios from 'axios';
import { clearAdminToken, getAdminToken } from '../lib/storage';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  timeout: 15000,
});

api.interceptors.request.use((config) => {
  const token = getAdminToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response?.status === 401 || error?.response?.status === 403) {
      clearAdminToken();
    }
    return Promise.reject(error);
  }
);

export function extractApiError(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as { error?: string; message?: string } | undefined;
    return data?.message || data?.error || error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'Неизвестная ошибка';
}
