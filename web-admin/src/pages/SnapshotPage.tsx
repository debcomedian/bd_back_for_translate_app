import { useEffect, useState } from 'react';
import { fetchSnapshot } from '../shared/api/snapshot';
import { recalculateWordsMeta } from '../shared/api/import';
import { extractApiError } from '../shared/api/client';
import { PageHeader } from '../shared/ui/PageHeader';
import type { SnapshotResponse } from '../types';

export function SnapshotPage() {
  const [data, setData] = useState<SnapshotResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const result = await fetchSnapshot();
      setData(result);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  async function handleRecalculate() {
    setBusy(true);
    setError(null);
    try {
      await recalculateWordsMeta();
      await load();
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const version = data?.snapshot_version;

  return (
    <div className="stack">
      <PageHeader title="Snapshot" subtitle="Технический экран состояния контентного snapshot для mobile client." actions={<><button className="btn btn-secondary" onClick={() => void load()}>Обновить</button><button className="btn btn-primary" onClick={() => void handleRecalculate()} disabled={busy}>{busy ? 'Пересчёт...' : 'Пересчитать мету и обновить'}</button></>} />
      {error ? <div className="error">{error}</div> : null}
      {loading ? <div className="notice">Загрузка snapshot...</div> : null}
      <div className="card stack">
        <h2>Активная версия snapshot</h2>
        {version ? (
          <dl className="kv">
            <dt>ID</dt><dd>{version.id}</dd>
            <dt>Тип</dt><dd>{version.snapshot_type}</dd>
            <dt>Version code</dt><dd>{version.version_code}</dd>
            <dt>Checksum</dt><dd>{version.checksum}</dd>
            <dt>Published at</dt><dd>{version.published_at ?? '—'}</dd>
          </dl>
        ) : (
          <div className="notice">Активная версия snapshot ещё не опубликована.</div>
        )}
      </div>
      <div className="grid-4">
        <div className="card stack"><h2>Категории</h2><div>{data?.categories.length ?? 0}</div></div>
        <div className="card stack"><h2>Слова</h2><div>{data?.words.length ?? 0}</div></div>
        <div className="card stack"><h2>Мета</h2><div>{data?.words_meta_base.length ?? 0}</div></div>
        <div className="card stack"><h2>Синонимы</h2><div>{data?.word_synonyms.length ?? 0}</div></div>
      </div>
    </div>
  );
}
