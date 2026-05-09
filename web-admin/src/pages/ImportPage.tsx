import { useState } from 'react';
import { importWords, recalculateWordsMeta } from '../shared/api/import';
import { extractApiError } from '../shared/api/client';
import { PageHeader } from '../shared/ui/PageHeader';
import type { ImportResponse, RecalculateMetaResponse } from '../types';

export function ImportPage() {
  const [file, setFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<ImportResponse | null>(null);
  const [metaResult, setMetaResult] = useState<RecalculateMetaResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loadingImport, setLoadingImport] = useState(false);
  const [loadingMeta, setLoadingMeta] = useState(false);

  async function handleImport() {
    if (!file) {
      setError('Выбери CSV-файл.');
      return;
    }
    setError(null);
    setImportResult(null);
    setLoadingImport(true);
    try {
      const result = await importWords(file);
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
      const result = await recalculateWordsMeta();
      setMetaResult(result);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoadingMeta(false);
    }
  }

  return (
    <div className="stack">
      <PageHeader title="Импорт CSV и пересчёт меты" subtitle="Загрузка слов из CSV и запуск `words_meta_base` пересчёта." />
      {error ? <div className="error">{error}</div> : null}
      <div className="card stack">
        <div>
          <label className="label">CSV-файл</label>
          <input className="input" type="file" accept=".csv,text/csv" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
        </div>
        <div className="notice">
          Ожидается предметный CSV под словарный контент. Поля и формат должны соответствовать backend import contract.
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => void handleImport()} disabled={loadingImport}>{loadingImport ? 'Импорт...' : 'Загрузить CSV'}</button>
          <button className="btn btn-secondary" onClick={() => void handleRecalculate()} disabled={loadingMeta}>{loadingMeta ? 'Пересчёт...' : 'Пересчитать мету'}</button>
        </div>
      </div>
      {importResult ? (
        <div className="card stack">
          <h2>Результат импорта</h2>
          <dl className="kv">
            <dt>Processed</dt><dd>{importResult.processed ?? '—'}</dd>
            <dt>Created</dt><dd>{importResult.created ?? '—'}</dd>
            <dt>Updated</dt><dd>{importResult.updated ?? '—'}</dd>
            <dt>Rejected</dt><dd>{importResult.rejected ?? '—'}</dd>
          </dl>
          {importResult.errors?.length ? (
            <div className="error">
              {importResult.errors.map((item, index) => (
                <div key={index}>Строка {item.row ?? '—'}: {item.message}</div>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
      {metaResult ? (
        <div className="card stack">
          <h2>Результат пересчёта</h2>
          <dl className="kv">
            <dt>Processed</dt><dd>{metaResult.processed}</dd>
            <dt>Snapshot version</dt><dd>{metaResult.snapshot?.version_code ?? '—'}</dd>
            <dt>Checksum</dt><dd>{metaResult.snapshot?.checksum ?? '—'}</dd>
          </dl>
        </div>
      ) : null}
    </div>
  );
}
