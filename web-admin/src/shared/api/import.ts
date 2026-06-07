import { api } from "./client";
import type {
  ActiveBankImportCommitResponse,
  ActiveBankImportResolution,
  ActiveBankImportPreviewResponse,
  ImportResponse,
  RecalculateMetaResponse,
} from "../../types";

const LONG_IMPORT_TIMEOUT_MS = 180000;

export async function importActiveBank(file?: File): Promise<ImportResponse> {
  if (file) {
    const form = new FormData();
    form.append("file", file);

    const { data } = await api.post<ImportResponse>(
      "/admin/content/import-active-bank",
      form,
      {
        headers: {
          "Content-Type": "multipart/form-data",
        },
        timeout: LONG_IMPORT_TIMEOUT_MS,
      },
    );
    return data;
  }

  const { data } = await api.post<ImportResponse>(
    "/admin/content/import-active-bank",
    undefined,
    { timeout: LONG_IMPORT_TIMEOUT_MS },
  );
  return data;
}

export async function previewActiveBankImport(
  file?: File,
): Promise<ActiveBankImportPreviewResponse> {
  if (file) {
    const form = new FormData();
    form.append("file", file);

    const { data } = await api.post<ActiveBankImportPreviewResponse>(
      "/admin/content/import-active-bank/preview",
      form,
      {
        headers: {
          "Content-Type": "multipart/form-data",
        },
        timeout: LONG_IMPORT_TIMEOUT_MS,
      },
    );
    return data;
  }

  const { data } = await api.post<ActiveBankImportPreviewResponse>(
    "/admin/content/import-active-bank/preview",
    undefined,
    { timeout: LONG_IMPORT_TIMEOUT_MS },
  );
  return data;
}

export async function commitActiveBankImport(
  sessionId: string,
  resolutions: ActiveBankImportResolution[],
): Promise<ActiveBankImportCommitResponse> {
  const { data } = await api.post<ActiveBankImportCommitResponse>(
    "/admin/content/import-active-bank/commit",
    {
      session_id: sessionId,
      resolutions,
    },
    { timeout: LONG_IMPORT_TIMEOUT_MS },
  );
  return data;
}

export async function recalculateDirections(): Promise<RecalculateMetaResponse> {
  const { data } = await api.post<RecalculateMetaResponse>(
    "/admin/content/recalculate-directions",
    undefined,
    { timeout: LONG_IMPORT_TIMEOUT_MS },
  );
  return data;
}
