import { useState } from 'react';
import { importWords, rebuildFromCurrent, recalculateWordsMeta } from '../shared/api/import';
import { extractApiError } from '../shared/api/client';
import { PageHeader } from '../shared/ui/PageHeader';
import type { ImportResponse, RecalculateMetaResponse } from '../types';

export function ImportPage() {
  const [importResult, setImportResult] = useState<ImportResponse | null>(null);
  const [rebuildResult, setRebuildResult] = useState<ImportResponse | null>(null);
  const [metaResult, setMetaResult] = useState<RecalculateMetaResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loadingImport, setLoadingImport] = useState(false);
  const [loadingRebuild, setLoadingRebuild] = useState(false);
  const [loadingMeta, setLoadingMeta] = useState(false);

  async function handleImport() {
    setError(null);
    setImportResult(null);
    setLoadingImport(true);
    try {
      const result = await importWords();
      setImportResult(result);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoadingImport(false);
    }
  }

  async function handleRebuild() {
    setError(null);
    setRebuildResult(null);
    setLoadingRebuild(true);
    try {
      const result = await rebuildFromCurrent();
      setRebuildResult(result);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoadingRebuild(false);
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

  function renderImportResult(title: string, result: ImportResponse) {
    return (
      <div className="card stack">
        <h2>{title}</h2>
        <dl className="kv">
          <dt>Processed</dt><dd>{result.processed ?? '—'}</dd>
          <dt>Concepts</dt><dd>{result.concepts ?? '—'}</dd>
          <dt>Forms</dt><dd>{result.forms ?? '—'}</dd>
          <dt>Directions</dt><dd>{result.directions ?? '—'}</dd>
          <dt>Snapshot version</dt><dd>{result.snapshot?.version_code ?? result.snapshot_version_code ?? '—'}</dd>
          <dt>Created</dt><dd>{result.created ?? '—'}</dd>
          <dt>Updated</dt><dd>{result.updated ?? '—'}</dd>
          <dt>Rejected</dt><dd>{result.rejected ?? '—'}</dd>
        </dl>
        {result.errors?.length ? (
          <div className="error">
            {result.errors.map((item, index) => (
              <div key={index}>Строка {item.row ?? '—'}: {item.message}</div>
            ))}
          </div>
        ) : null}
      </div>
    );
  }

  return (
    <div className="stack">
      <PageHeader title="Импорт и пересчёт" subtitle="Операции над направленной словарной моделью: active_bank, rebuild и пересчёт difficulty." />
      {error ? <div className="error">{error}</div> : null}
      <div className="card stack">
        <div className="notice">
          Импорт берёт файл из backend-пути <code>data/lexicon/active_bank.jsonl</code>. Загрузка CSV из браузера больше не используется в новой модели.
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => void handleImport()} disabled={loadingImport}>{loadingImport ? 'Импорт...' : 'Импортировать active_bank'}</button>
          <button className="btn btn-secondary" onClick={() => void handleRebuild()} disabled={loadingRebuild}>{loadingRebuild ? 'Rebuild...' : 'Rebuild from current'}</button>
          <button className="btn btn-secondary" onClick={() => void handleRecalculate()} disabled={loadingMeta}>{loadingMeta ? 'Пересчёт...' : 'Пересчитать направления'}</button>
        </div>
      </div>
      {importResult ? renderImportResult('Результат active_bank import', importResult) : null}
      {rebuildResult ? renderImportResult('Результат rebuild', rebuildResult) : null}
      {metaResult ? (
        <div className="card stack">
          <h2>Результат пересчёта направлений</h2>
          <dl className="kv">
            <dt>Processed</dt><dd>{metaResult.processed}</dd>
            <dt>Directions</dt><dd>{metaResult.directions ?? '—'}</dd>
            <dt>Snapshot version</dt><dd>{metaResult.snapshot?.version_code ?? '—'}</dd>
            <dt>Checksum</dt><dd>{metaResult.snapshot?.checksum ?? '—'}</dd>
          </dl>
        </div>
      ) : null}
    </div>
  );
}
