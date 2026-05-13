import { useEffect, useMemo, useState } from 'react';
import { fetchDirections } from '../shared/api/directions';
import { extractApiError } from '../shared/api/client';
import { PageHeader } from '../shared/ui/PageHeader';
import type { TrainingDirection } from '../types';

type SortKey =
  | 'direction_id'
  | 'direction_code'
  | 'source_value'
  | 'target_value'
  | 'category_name_ru'
  | 'cefr_level'
  | 'final_difficulty'
  | 'is_active';

type SortDirection = 'asc' | 'desc';

function valueForSort(item: TrainingDirection, key: SortKey): string | number | boolean | null | undefined {
  return item[key];
}

function compareValues(a: string | number | boolean | null | undefined, b: string | number | boolean | null | undefined) {
  if (a === b) return 0;
  if (a === undefined || a === null) return -1;
  if (b === undefined || b === null) return 1;
  if (typeof a === 'number' && typeof b === 'number') return a - b;
  if (typeof a === 'boolean' && typeof b === 'boolean') return Number(a) - Number(b);
  return String(a).localeCompare(String(b), 'ru', { numeric: true, sensitivity: 'base' });
}

function formatNumber(value?: number | null) {
  if (value === undefined || value === null) return '—';
  return Number(value).toFixed(3);
}

export function WordsPage() {
  const [directions, setDirections] = useState<TrainingDirection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [directionCode, setDirectionCode] = useState('all');
  const [sortKey, setSortKey] = useState<SortKey>('direction_id');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchDirections();
      setDirections(data);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const directionCodes = useMemo(
    () => Array.from(new Set(directions.map((item) => item.direction_code).filter(Boolean))).sort(),
    [directions]
  );

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    const result = directions.filter((item) => {
      if (directionCode !== 'all' && item.direction_code !== directionCode) return false;
      if (!q) return true;
      return [
        item.direction_id,
        item.direction_code,
        item.source_value,
        item.target_value,
        item.source_lang_code,
        item.target_lang_code,
        item.category_name_ru,
        item.category_name_en,
        item.category_slug,
        item.cefr_level,
      ]
        .filter((value) => value !== undefined && value !== null)
        .some((value) => String(value).toLowerCase().includes(q));
    });

    result.sort((a, b) => {
      const order = compareValues(valueForSort(a, sortKey), valueForSort(b, sortKey));
      return sortDirection === 'asc' ? order : -order;
    });

    return result;
  }, [directions, directionCode, search, sortDirection, sortKey]);

  function changeSort(nextKey: SortKey) {
    if (sortKey === nextKey) {
      setSortDirection((current) => (current === 'asc' ? 'desc' : 'asc'));
      return;
    }
    setSortKey(nextKey);
    setSortDirection('asc');
  }

  function sortLabel(key: SortKey) {
    if (sortKey !== key) return '';
    return sortDirection === 'asc' ? ' ↑' : ' ↓';
  }

  return (
    <div className="stack">
      <PageHeader
        title="Направления"
        subtitle="Главный экран новой словарной модели: задания source → target с отдельной сложностью и прогрессом."
        actions={<button className="btn btn-secondary" onClick={() => void load()}>Обновить</button>}
      />

      {error ? <div className="error">{error}</div> : null}
      {loading ? <div className="notice">Загрузка направлений...</div> : null}

      <div className="card stack">
        <div className="toolbar">
          <div className="toolbar-field grow">
            <label className="label">Поиск</label>
            <input
              className="input"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="source, target, direction, category, CEFR"
            />
          </div>
          <div className="toolbar-field">
            <label className="label">Direction</label>
            <select className="select" value={directionCode} onChange={(e) => setDirectionCode(e.target.value)}>
              <option value="all">Все</option>
              {directionCodes.map((code) => (
                <option key={code} value={code}>{code}</option>
              ))}
            </select>
          </div>
          <div className="toolbar-stat">
            <span>Показано</span>
            <strong>{filtered.length}</strong>
            <span>из {directions.length}</span>
          </div>
        </div>
      </div>

      <div className="card table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th><button className="th-btn" onClick={() => changeSort('direction_id')}>ID{sortLabel('direction_id')}</button></th>
              <th><button className="th-btn" onClick={() => changeSort('direction_code')}>Direction{sortLabel('direction_code')}</button></th>
              <th><button className="th-btn" onClick={() => changeSort('source_value')}>Source{sortLabel('source_value')}</button></th>
              <th><button className="th-btn" onClick={() => changeSort('target_value')}>Target{sortLabel('target_value')}</button></th>
              <th>Lang</th>
              <th><button className="th-btn" onClick={() => changeSort('category_name_ru')}>Категория{sortLabel('category_name_ru')}</button></th>
              <th><button className="th-btn" onClick={() => changeSort('cefr_level')}>CEFR{sortLabel('cefr_level')}</button></th>
              <th><button className="th-btn" onClick={() => changeSort('final_difficulty')}>Difficulty{sortLabel('final_difficulty')}</button></th>
              <th><button className="th-btn" onClick={() => changeSort('is_active')}>Active{sortLabel('is_active')}</button></th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((item) => (
              <tr key={item.direction_id}>
                <td>{item.direction_id}</td>
                <td><span className="badge badge-muted">{item.direction_code}</span></td>
                <td>{item.source_value || '—'}</td>
                <td>{item.target_value || '—'}</td>
                <td>{item.source_lang_code} → {item.target_lang_code}</td>
                <td>{item.category_name_ru ?? item.category_slug ?? '—'}</td>
                <td>{item.cefr_level ?? '—'}</td>
                <td>{formatNumber(item.final_difficulty)}</td>
                <td>
                  <span className={`badge ${item.is_active ? 'badge-ok' : 'badge-muted'}`}>
                    {item.is_active ? 'active' : 'inactive'}
                  </span>
                </td>
              </tr>
            ))}
            {!loading && filtered.length === 0 ? (
              <tr>
                <td colSpan={9}>Направления не найдены.</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
