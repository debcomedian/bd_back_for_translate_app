import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { fetchCategories } from '../shared/api/categories';
import { fetchWords } from '../shared/api/words';
import { extractApiError } from '../shared/api/client';
import { PageHeader } from '../shared/ui/PageHeader';
import type { Category, Word } from '../types';

type WordSortField =
  | 'id'
  | 'lang_code'
  | 'word_ru'
  | 'word_en'
  | 'word_de'
  | 'category'
  | 'cefr'
  | 'difficulty'
  | 'freq'
  | 'active';

type SortDirection = 'asc' | 'desc';

type WordSortState = {
  field: WordSortField;
  direction: SortDirection;
};

function text(value: unknown): string {
  if (value === null || value === undefined) return '';
  return String(value).toLowerCase();
}

function compareText(a: unknown, b: unknown): number {
  return text(a).localeCompare(text(b), 'ru', { numeric: true, sensitivity: 'base' });
}

function compareNumber(a: unknown, b: unknown): number {
  const left = Number(a ?? Number.NEGATIVE_INFINITY);
  const right = Number(b ?? Number.NEGATIVE_INFINITY);

  if (Number.isNaN(left) && Number.isNaN(right)) return 0;
  if (Number.isNaN(left)) return -1;
  if (Number.isNaN(right)) return 1;

  return left - right;
}

function sortMark(sort: WordSortState, field: WordSortField): string {
  if (sort.field !== field) return '';
  return sort.direction === 'asc' ? ' ↑' : ' ↓';
}

function getCategoryName(word: Word, categoryMap: Map<number, string>): string {
  if (!word.category_id) return '';
  return categoryMap.get(word.category_id) ?? String(word.category_id);
}

export function WordsPage() {
  const [words, setWords] = useState<Word[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [langFilter, setLangFilter] = useState('all');
  const [activeFilter, setActiveFilter] = useState('all');
  const [sort, setSort] = useState<WordSortState>({ field: 'id', direction: 'asc' });

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const [wordsData, categoriesData] = await Promise.all([fetchWords(), fetchCategories()]);
      setWords(wordsData);
      setCategories(categoriesData);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const categoryMap = useMemo(() => new Map(categories.map((item) => [item.id, item.name_ru])), [categories]);

  const langOptions = useMemo(() => {
    const values = new Set<string>();
    words.forEach((word) => {
      if (word.lang_code) values.add(word.lang_code);
    });
    return Array.from(values).sort((a, b) => a.localeCompare(b));
  }, [words]);

  const visibleWords = useMemo(() => {
    const query = search.trim().toLowerCase();

    const filtered = words.filter((word) => {
      if (langFilter !== 'all' && word.lang_code !== langFilter) return false;
      if (activeFilter === 'active' && !word.is_active) return false;
      if (activeFilter === 'inactive' && word.is_active) return false;

      if (!query) return true;

      const categoryName = getCategoryName(word, categoryMap);
      const haystack = [
        word.id,
        word.lang_code,
        word.word_ru,
        word.word_en,
        word.word_de,
        word.transcription_ru,
        word.transcription_en,
        word.transcription_de,
        word.source_ref,
        categoryName,
        word.category_id,
        word.meta_base?.meta_cefr_level,
        word.meta_base?.meta_base_difficulty,
        word.meta_base?.meta_freq_bucket,
      ]
        .map(text)
        .join(' ');

      return haystack.includes(query);
    });

    return [...filtered].sort((a, b) => {
      let result = 0;

      switch (sort.field) {
        case 'id':
          result = compareNumber(a.id, b.id);
          break;
        case 'lang_code':
          result = compareText(a.lang_code, b.lang_code);
          break;
        case 'word_ru':
          result = compareText(a.word_ru, b.word_ru);
          break;
        case 'word_en':
          result = compareText(a.word_en, b.word_en);
          break;
        case 'word_de':
          result = compareText(a.word_de, b.word_de);
          break;
        case 'category':
          result = compareText(getCategoryName(a, categoryMap), getCategoryName(b, categoryMap));
          break;
        case 'cefr':
          result = compareText(a.meta_base?.meta_cefr_level, b.meta_base?.meta_cefr_level);
          break;
        case 'difficulty':
          result = compareNumber(a.meta_base?.meta_base_difficulty, b.meta_base?.meta_base_difficulty);
          break;
        case 'freq':
          result = compareNumber(a.meta_base?.meta_freq_bucket, b.meta_base?.meta_freq_bucket);
          break;
        case 'active':
          result = Number(a.is_active) - Number(b.is_active);
          break;
        default:
          result = 0;
      }

      return sort.direction === 'asc' ? result : -result;
    });
  }, [activeFilter, categoryMap, langFilter, search, sort, words]);

  function changeSort(field: WordSortField) {
    setSort((current) => ({
      field,
      direction: current.field === field && current.direction === 'asc' ? 'desc' : 'asc',
    }));
  }

  function resetFilters() {
    setSearch('');
    setLangFilter('all');
    setActiveFilter('all');
    setSort({ field: 'id', direction: 'asc' });
  }

  return (
    <div className="stack">
      <PageHeader
        title="Слова"
        subtitle="Главный рабочий экран администратора. Просмотр слов, поиск, сортировка, переход к созданию и редактированию."
        actions={
          <>
            <button className="btn btn-secondary" onClick={() => void load()}>Обновить</button>
            <Link className="btn btn-primary" to="/words/new">Создать слово</Link>
          </>
        }
      />

      {error ? <div className="error">{error}</div> : null}
      {loading ? <div className="notice">Загрузка...</div> : null}

      <div className="card stack">
        <div className="filter-grid">
          <label className="field">
            <span className="label">Поиск</span>
            <input
              className="input"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="ID, RU, EN, DE, категория, CEFR, источник"
            />
          </label>

          <label className="field">
            <span className="label">Язык</span>
            <select className="select" value={langFilter} onChange={(event) => setLangFilter(event.target.value)}>
              <option value="all">Все языки</option>
              {langOptions.map((lang) => (
                <option key={lang} value={lang}>{lang}</option>
              ))}
            </select>
          </label>

          <label className="field">
            <span className="label">Активность</span>
            <select className="select" value={activeFilter} onChange={(event) => setActiveFilter(event.target.value)}>
              <option value="all">Все</option>
              <option value="active">Только active</option>
              <option value="inactive">Только inactive</option>
            </select>
          </label>

          <div className="filter-actions">
            <button className="btn btn-secondary" onClick={resetFilters}>Сбросить</button>
          </div>
        </div>

        <div className="muted">
          Показано {visibleWords.length} из {words.length}. Сортировка: {sort.field}, {sort.direction === 'asc' ? 'по возрастанию' : 'по убыванию'}.
        </div>
      </div>

      <div className="card table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th><button className="th-sort" onClick={() => changeSort('id')}>ID{sortMark(sort, 'id')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('lang_code')}>Lang{sortMark(sort, 'lang_code')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('word_ru')}>RU{sortMark(sort, 'word_ru')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('word_en')}>EN{sortMark(sort, 'word_en')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('word_de')}>DE{sortMark(sort, 'word_de')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('category')}>Категория{sortMark(sort, 'category')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('cefr')}>CEFR{sortMark(sort, 'cefr')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('difficulty')}>Difficulty{sortMark(sort, 'difficulty')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('freq')}>Freq{sortMark(sort, 'freq')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('active')}>Active{sortMark(sort, 'active')}</button></th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {visibleWords.map((word) => (
              <tr key={word.id}>
                <td>{word.id}</td>
                <td>{word.lang_code}</td>
                <td>{word.word_ru ?? '—'}</td>
                <td>{word.word_en ?? '—'}</td>
                <td>{word.word_de ?? '—'}</td>
                <td>{word.category_id ? categoryMap.get(word.category_id) ?? word.category_id : '—'}</td>
                <td>{word.meta_base?.meta_cefr_level ?? '—'}</td>
                <td>{word.meta_base?.meta_base_difficulty ?? '—'}</td>
                <td>{word.meta_base?.meta_freq_bucket ?? '—'}</td>
                <td>
                  <span className={`badge ${word.is_active ? 'badge-ok' : 'badge-muted'}`}>
                    {word.is_active ? 'active' : 'inactive'}
                  </span>
                </td>
                <td><Link className="btn btn-secondary" to={`/words/${word.id}/edit`}>Редактировать</Link></td>
              </tr>
            ))}
            {!loading && visibleWords.length === 0 ? (
              <tr>
                <td colSpan={11}>По текущим фильтрам слова не найдены.</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
