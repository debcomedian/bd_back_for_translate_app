import { useEffect, useMemo, useState } from "react";
import { fetchDirections } from "../shared/api/directions";
import { extractApiError } from "../shared/api/client";
import { PageHeader } from "../shared/ui/PageHeader";
import type {
  DirectionListPagination,
  DirectionSortBy,
  DirectionSortDir,
  TrainingDirection,
} from "../types";

const DIRECTION_CODES = ["en_ru", "ru_en", "en_de", "de_en", "ru_de", "de_ru"];
const PAGE_SIZE_OPTIONS = [25, 50, 100, 250];

function formatNumber(value?: number | null) {
  if (value === undefined || value === null) return "—";
  return Number(value).toFixed(3);
}

function sortLabel(
  currentKey: DirectionSortBy,
  currentDirection: DirectionSortDir,
  key: DirectionSortBy,
) {
  if (currentKey !== key) return "";
  return currentDirection === "asc" ? " ↑" : " ↓";
}

function pageRange(pagination: DirectionListPagination) {
  if (pagination.total === 0) {
    return { from: 0, to: 0 };
  }

  const from = (pagination.page - 1) * pagination.page_size + 1;
  const to = Math.min(pagination.page * pagination.page_size, pagination.total);
  return { from, to };
}

export function WordsPage() {
  const [directions, setDirections] = useState<TrainingDirection[]>([]);
  const [pagination, setPagination] = useState<DirectionListPagination>({
    page: 1,
    page_size: 100,
    total: 0,
    total_pages: 0,
    has_prev: false,
    has_next: false,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [directionCode, setDirectionCode] = useState("all");
  const [sortKey, setSortKey] = useState<DirectionSortBy>("direction_id");
  const [sortDirection, setSortDirection] = useState<DirectionSortDir>("asc");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(100);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setPage(1);
      setDebouncedSearch(search.trim());
    }, 350);

    return () => window.clearTimeout(timer);
  }, [search]);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchDirections({
        page,
        page_size: pageSize,
        q: debouncedSearch || undefined,
        direction_code: directionCode === "all" ? undefined : directionCode,
        active: "true",
        sort_by: sortKey,
        sort_dir: sortDirection,
      });
      setDirections(data.items);
      setPagination(data.pagination);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [debouncedSearch, directionCode, page, pageSize, sortDirection, sortKey]);

  const directionCodes = useMemo(() => {
    const fromPage = directions
      .map((item) => item.direction_code)
      .filter(Boolean);
    return Array.from(new Set([...DIRECTION_CODES, ...fromPage])).sort();
  }, [directions]);

  const range = pageRange(pagination);

  function changeSort(nextKey: DirectionSortBy) {
    setPage(1);
    if (sortKey === nextKey) {
      setSortDirection((current) => (current === "asc" ? "desc" : "asc"));
      return;
    }
    setSortKey(nextKey);
    setSortDirection("asc");
  }

  function changeDirectionCode(nextCode: string) {
    setPage(1);
    setDirectionCode(nextCode);
  }

  function changePageSize(nextPageSize: number) {
    setPage(1);
    setPageSize(nextPageSize);
  }

  function goToPage(nextPage: number) {
    if (nextPage < 1) return;
    if (pagination.total_pages > 0 && nextPage > pagination.total_pages) return;
    setPage(nextPage);
  }

  const paginationControls = (
    <div className="pagination-bar">
      <div className="pagination-info">
        Показано {range.from}–{range.to} из {pagination.total}. Страница{" "}
        {pagination.page} из {pagination.total_pages || 1}.
      </div>
      <div className="pagination-actions">
        <button
          className="btn btn-secondary"
          disabled={!pagination.has_prev || loading}
          onClick={() => goToPage(1)}
        >
          Первая
        </button>
        <button
          className="btn btn-secondary"
          disabled={!pagination.has_prev || loading}
          onClick={() => goToPage(page - 1)}
        >
          Назад
        </button>
        <button
          className="btn btn-secondary"
          disabled={!pagination.has_next || loading}
          onClick={() => goToPage(page + 1)}
        >
          Вперёд
        </button>
        <button
          className="btn btn-secondary"
          disabled={!pagination.has_next || loading}
          onClick={() => goToPage(pagination.total_pages)}
        >
          Последняя
        </button>
      </div>
    </div>
  );

  return (
    <div className="stack">
      <PageHeader
        title="Направления"
        subtitle="Серверная таблица учебных заданий источник → цель. Пагинация, поиск и сортировка выполняются на сервере."
        actions={
          <button
            className="btn btn-secondary"
            onClick={() => void load()}
            disabled={loading}
          >
            Обновить
          </button>
        }
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
              placeholder="источник, цель, направление, категория, CEFR"
            />
          </div>
          <div className="toolbar-field">
            <label className="label">Направление</label>
            <select
              className="select"
              value={directionCode}
              onChange={(e) => changeDirectionCode(e.target.value)}
            >
              <option value="all">Все</option>
              {directionCodes.map((code) => (
                <option key={code} value={code}>
                  {code}
                </option>
              ))}
            </select>
          </div>
          <div className="toolbar-field small">
            <label className="label">На странице</label>
            <select
              className="select"
              value={pageSize}
              onChange={(e) => changePageSize(Number(e.target.value))}
            >
              {PAGE_SIZE_OPTIONS.map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </div>
          <div className="toolbar-stat">
            <span>Всего</span>
            <strong>{pagination.total}</strong>
          </div>
        </div>
        {paginationControls}
      </div>

      <div className="card table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>
                <button
                  className="th-btn"
                  onClick={() => changeSort("direction_id")}
                >
                  ID{sortLabel(sortKey, sortDirection, "direction_id")}
                </button>
              </th>
              <th>
                <button
                  className="th-btn"
                  onClick={() => changeSort("direction_code")}
                >
                  Направление
                  {sortLabel(sortKey, sortDirection, "direction_code")}
                </button>
              </th>
              <th>
                <button
                  className="th-btn"
                  onClick={() => changeSort("source_value")}
                >
                  Источник{sortLabel(sortKey, sortDirection, "source_value")}
                </button>
              </th>
              <th>
                <button
                  className="th-btn"
                  onClick={() => changeSort("target_value")}
                >
                  Цель{sortLabel(sortKey, sortDirection, "target_value")}
                </button>
              </th>
              <th>
                <button
                  className="th-btn"
                  onClick={() => changeSort("source_lang_code")}
                >
                  Язык{sortLabel(sortKey, sortDirection, "source_lang_code")}
                </button>
              </th>
              <th>
                <button
                  className="th-btn"
                  onClick={() => changeSort("category_name_ru")}
                >
                  Категория
                  {sortLabel(sortKey, sortDirection, "category_name_ru")}
                </button>
              </th>
              <th>
                <button
                  className="th-btn"
                  onClick={() => changeSort("cefr_level")}
                >
                  CEFR{sortLabel(sortKey, sortDirection, "cefr_level")}
                </button>
              </th>
              <th>
                <button
                  className="th-btn"
                  onClick={() => changeSort("final_difficulty")}
                >
                  Сложность
                  {sortLabel(sortKey, sortDirection, "final_difficulty")}
                </button>
              </th>
              <th>
                <button
                  className="th-btn"
                  onClick={() => changeSort("is_active")}
                >
                  Активность{sortLabel(sortKey, sortDirection, "is_active")}
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            {directions.map((item) => (
              <tr key={item.direction_id}>
                <td>{item.direction_id}</td>
                <td>
                  <span className="badge badge-muted">
                    {item.direction_code}
                  </span>
                </td>
                <td>{item.source_value || "—"}</td>
                <td>{item.target_value || "—"}</td>
                <td>
                  {item.source_lang_code} → {item.target_lang_code}
                </td>
                <td>{item.category_name_ru ?? item.category_slug ?? "—"}</td>
                <td>{item.cefr_level ?? "—"}</td>
                <td>{formatNumber(item.final_difficulty)}</td>
                <td>
                  <span
                    className={`badge ${item.is_active ? "badge-ok" : "badge-muted"}`}
                  >
                    {item.is_active ? "активно" : "отключено"}
                  </span>
                </td>
              </tr>
            ))}
            {!loading && directions.length === 0 ? (
              <tr>
                <td colSpan={9}>Направления не найдены.</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>

      <div className="card">{paginationControls}</div>
    </div>
  );
}
