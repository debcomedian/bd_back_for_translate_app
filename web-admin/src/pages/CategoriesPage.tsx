import { useEffect, useMemo, useState } from 'react';
import { createCategory, fetchCategories, updateCategory } from '../shared/api/categories';
import { extractApiError } from '../shared/api/client';
import { PageHeader } from '../shared/ui/PageHeader';
import type { Category } from '../types';
import { CategoryForm, type CategoryFormValue } from '../features/categories/CategoryForm';

type CategorySortField = 'id' | 'slug' | 'name_ru' | 'name_en' | 'name_de' | 'entity';
type SortDirection = 'asc' | 'desc';

type CategorySortState = {
  field: CategorySortField;
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
  return Number(a ?? 0) - Number(b ?? 0);
}

function sortMark(sort: CategorySortState, field: CategorySortField): string {
  if (sort.field !== field) return '';
  return sort.direction === 'asc' ? ' ↑' : ' ↓';
}

export function CategoriesPage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [editing, setEditing] = useState<Category | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [entityFilter, setEntityFilter] = useState('all');
  const [sort, setSort] = useState<CategorySortState>({ field: 'id', direction: 'asc' });

  async function load() {
    try {
      const data = await fetchCategories();
      setCategories(data);
    } catch (e) {
      setError(extractApiError(e));
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const entityOptions = useMemo(() => {
    const values = new Set<string>();
    categories.forEach((item) => {
      if (item.entity) values.add(item.entity);
    });
    return Array.from(values).sort((a, b) => a.localeCompare(b));
  }, [categories]);

  const visibleCategories = useMemo(() => {
    const query = search.trim().toLowerCase();

    const filtered = categories.filter((item) => {
      if (entityFilter !== 'all' && item.entity !== entityFilter) return false;
      if (!query) return true;

      const haystack = [item.id, item.slug, item.name_ru, item.name_en, item.name_de, item.entity]
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
        case 'slug':
          result = compareText(a.slug, b.slug);
          break;
        case 'name_ru':
          result = compareText(a.name_ru, b.name_ru);
          break;
        case 'name_en':
          result = compareText(a.name_en, b.name_en);
          break;
        case 'name_de':
          result = compareText(a.name_de, b.name_de);
          break;
        case 'entity':
          result = compareText(a.entity, b.entity);
          break;
        default:
          result = 0;
      }

      return sort.direction === 'asc' ? result : -result;
    });
  }, [categories, entityFilter, search, sort]);

  function changeSort(field: CategorySortField) {
    setSort((current) => ({
      field,
      direction: current.field === field && current.direction === 'asc' ? 'desc' : 'asc',
    }));
  }

  function resetFilters() {
    setSearch('');
    setEntityFilter('all');
    setSort({ field: 'id', direction: 'asc' });
  }

  async function submitCreate(payload: CategoryFormValue) {
    setError(null);
    setSuccess(null);
    try {
      const created = await createCategory(payload);
      setSuccess(`Категория #${created.id} создана.`);
      await load();
    } catch (e) {
      setError(extractApiError(e));
    }
  }

  async function submitUpdate(payload: CategoryFormValue) {
    if (!editing) return;
    setError(null);
    setSuccess(null);
    try {
      const updated = await updateCategory(editing.id, payload);
      setSuccess(`Категория #${updated.id} обновлена.`);
      setEditing(updated);
      await load();
    } catch (e) {
      setError(extractApiError(e));
    }
  }

  return (
    <div className="stack">
      <PageHeader title="Категории" subtitle="Управление категориями слов для словарного контента." actions={<button className="btn btn-secondary" onClick={() => void load()}>Обновить</button>} />
      {error ? <div className="error">{error}</div> : null}
      {success ? <div className="success">{success}</div> : null}
      <div className="grid-2">
        <div className="card stack">
          <h2>Создать категорию</h2>
          <CategoryForm submitLabel="Создать категорию" onSubmit={submitCreate} />
        </div>
        <div className="card stack">
          <h2>Редактировать категорию</h2>
          {editing ? (
            <CategoryForm key={editing.id} initial={editing} submitLabel="Сохранить категорию" onSubmit={submitUpdate} />
          ) : (
            <div className="notice">Выбери категорию из списка ниже.</div>
          )}
        </div>
      </div>

      <div className="card stack">
        <div className="filter-grid filter-grid-compact">
          <label className="field">
            <span className="label">Поиск</span>
            <input
              className="input"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="ID, slug, RU, EN, DE, entity"
            />
          </label>

          <label className="field">
            <span className="label">Entity</span>
            <select className="select" value={entityFilter} onChange={(event) => setEntityFilter(event.target.value)}>
              <option value="all">Все</option>
              {entityOptions.map((entity) => (
                <option key={entity} value={entity}>{entity}</option>
              ))}
            </select>
          </label>

          <div className="filter-actions">
            <button className="btn btn-secondary" onClick={resetFilters}>Сбросить</button>
          </div>
        </div>

        <div className="muted">
          Показано {visibleCategories.length} из {categories.length}. Сортировка: {sort.field}, {sort.direction === 'asc' ? 'по возрастанию' : 'по убыванию'}.
        </div>
      </div>

      <div className="card table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th><button className="th-sort" onClick={() => changeSort('id')}>ID{sortMark(sort, 'id')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('slug')}>Slug{sortMark(sort, 'slug')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('name_ru')}>RU{sortMark(sort, 'name_ru')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('name_en')}>EN{sortMark(sort, 'name_en')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('name_de')}>DE{sortMark(sort, 'name_de')}</button></th>
              <th><button className="th-sort" onClick={() => changeSort('entity')}>Entity{sortMark(sort, 'entity')}</button></th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {visibleCategories.map((item) => (
              <tr key={item.id}>
                <td>{item.id}</td>
                <td>{item.slug}</td>
                <td>{item.name_ru}</td>
                <td>{item.name_en}</td>
                <td>{item.name_de}</td>
                <td>{item.entity}</td>
                <td><button className="btn btn-secondary" onClick={() => setEditing(item)}>Редактировать</button></td>
              </tr>
            ))}
            {visibleCategories.length === 0 ? (
              <tr><td colSpan={7}>По текущим фильтрам категории не найдены.</td></tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
