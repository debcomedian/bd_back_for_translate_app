import axios from "axios";
import { clearAdminToken, getAdminToken } from "../lib/storage";

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "http://localhost:8080",
  timeout: 60000,
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
  },
);

const statusMessages: Record<number, string> = {
  400: "Некорректный запрос",
  401: "Неверный логин или пароль",
  403: "Доступ запрещён",
  404: "Запрошенный раздел или метод не найден",
  409: "Конфликт данных",
  422: "Данные не прошли проверку",
  500: "Внутренняя ошибка сервера",
  502: "Сервер временно недоступен",
  503: "Сервис временно недоступен",
  504: "Сервер не ответил вовремя",
};

const backendErrorMessages: Record<string, string> = {
  bad_request: "Некорректный запрос",
  unauthorized: "Неверный логин или пароль",
  forbidden: "Доступ запрещён",
  internal_error: "Внутренняя ошибка сервера",
  validation_error: "Данные не прошли проверку",
  password_hash_error: "Не удалось обработать пароль",
  token_error: "Не удалось сформировать токен авторизации",
};

function isRussianText(value: string): boolean {
  return /[А-Яа-яЁё]/.test(value);
}

export function extractApiError(error: unknown): string {
  if (axios.isAxiosError(error)) {
    if (error.code === "ECONNABORTED") {
      return "Операция выполнялась дольше отведённого времени. Для импорта и публикации снимка установлен увеличенный лимит ожидания.";
    }

    if (!error.response) {
      return "Не удалось подключиться к серверу. Проверьте, что сервер запущен и адрес API указан верно.";
    }

    const data = error.response.data as
      | { error?: string; message?: string }
      | string
      | undefined;

    if (typeof data === "string" && data.trim()) {
      return isRussianText(data) ? data : statusMessages[error.response.status] || "Ошибка при выполнении запроса";
    }

    const objectData = typeof data === "object" && data !== null ? data : undefined;

    if (typeof objectData?.message === "string" && objectData.message.trim()) {
      return objectData.message;
    }

    if (typeof objectData?.error === "string" && objectData.error.trim()) {
      return backendErrorMessages[objectData.error] || statusMessages[error.response.status] || "Ошибка при выполнении запроса";
    }

    return statusMessages[error.response.status] || "Ошибка при выполнении запроса";
  }

  if (error instanceof Error) {
    return isRussianText(error.message) ? error.message : "Ошибка при выполнении операции";
  }

  return "Неизвестная ошибка";
}
