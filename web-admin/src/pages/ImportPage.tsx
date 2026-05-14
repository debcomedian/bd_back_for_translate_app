import { useState } from "react";
import { importActiveBank, recalculateDirections } from "../shared/api/import";
import { extractApiError } from "../shared/api/client";
import { PageHeader } from "../shared/ui/PageHeader";
import type { ImportResponse, RecalculateMetaResponse } from "../types";

export function ImportPage() {
  const [importResult, setImportResult] = useState<ImportResponse | null>(null);
  const [metaResult, setMetaResult] = useState<RecalculateMetaResponse | null>(
    null,
  );
  const [error, setError] = useState<string | null>(null);
  const [loadingImport, setLoadingImport] = useState(false);
  const [loadingMeta, setLoadingMeta] = useState(false);

  async function handleImport() {
    setError(null);
    setImportResult(null);
    setLoadingImport(true);

    try {
      const result = await importActiveBank();
      setImportResult(result);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoadingImport(false);
    }
  }

  async function handleRecalculate() {
    setError(null);
    setMetaResult(null);
    setLoadingMeta(true);

    try {
      const result = await recalculateDirections();
      setMetaResult(result);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoadingMeta(false);
    }
  }

  function renderImportResult(title: string, result: ImportResponse) {
    return (
      <div className="card stack">
        <h2>{title}</h2>
        <dl className="kv">
          <dt>Обработано</dt>
          <dd>{result.processed ?? "—"}</dd>

          <dt>Смысловые карточки</dt>
          <dd>{result.concepts ?? "—"}</dd>

          <dt>Языковые формы</dt>
          <dd>{result.forms ?? "—"}</dd>

          <dt>Направления</dt>
          <dd>{result.directions ?? "—"}</dd>

          <dt>Версия snapshot</dt>
          <dd>
            {result.snapshot?.version_code ??
              result.snapshot_version_code ??
              "—"}
          </dd>

          <dt>Создано</dt>
          <dd>{result.created ?? "—"}</dd>

          <dt>Обновлено</dt>
          <dd>{result.updated ?? "—"}</dd>

          <dt>Отклонено</dt>
          <dd>{result.rejected ?? "—"}</dd>
        </dl>

        {result.errors?.length ? (
          <div className="error">
            {result.errors.map((item, index) => (
              <div key={index}>
                Строка {item.row ?? "—"}: {item.message}
              </div>
            ))}
          </div>
        ) : null}
      </div>
    );
  }

  return (
    <div className="stack">
      <PageHeader
        title="Импорт и пересчёт"
        subtitle="Операции над направленной словарной моделью: импорт active_bank и пересчёт сложности."
      />

      {error ? <div className="error">{error}</div> : null}

      <div className="card stack">
        <div className="notice">
          Импорт берёт файл из серверного пути{" "}
          <code>data/lexicon/active_bank.jsonl</code>. Загрузка CSV из браузера
          больше не используется в новой модели.
        </div>

        <div className="actions">
          <button
            className="btn btn-primary"
            onClick={() => void handleImport()}
            disabled={loadingImport}
          >
            {loadingImport ? "Импорт..." : "Импортировать active_bank"}
          </button>

          <button
            className="btn btn-secondary"
            onClick={() => void handleRecalculate()}
            disabled={loadingMeta}
          >
            {loadingMeta ? "Пересчёт..." : "Пересчитать направления"}
          </button>
        </div>
      </div>

      {importResult
        ? renderImportResult("Результат импорта active_bank", importResult)
        : null}

      {metaResult ? (
        <div className="card stack">
          <h2>Результат пересчёта направлений</h2>
          <dl className="kv">
            <dt>Обработано</dt>
            <dd>{metaResult.processed}</dd>

            <dt>Направления</dt>
            <dd>{metaResult.directions ?? "—"}</dd>

            <dt>Версия snapshot</dt>
            <dd>{metaResult.snapshot?.version_code ?? "—"}</dd>

            <dt>Контрольная сумма</dt>
            <dd>{metaResult.snapshot?.checksum ?? "—"}</dd>
          </dl>
        </div>
      ) : null}
    </div>
  );
}
