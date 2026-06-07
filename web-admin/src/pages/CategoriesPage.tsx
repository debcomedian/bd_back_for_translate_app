import { useEffect, useMemo, useState } from "react";
import {
  createCategory,
  fetchCategories,
  updateCategory,
} from "../shared/api/categories";
import {
  fetchDirections,
  fetchCategoryDirectionSelection,
  updateCategoryDirections,
} from "../shared/api/directions";
import { extractApiError } from "../shared/api/client";
import { PageHeader } from "../shared/ui/PageHeader";
import {
  FixedToast,
  type AppToast,
  type ToastKind,
} from "../shared/ui/FixedToast";
import type {
  Category,
  DirectionSortBy,
  DirectionSortDir,
  TrainingDirection,
} from "../types";
import {
  CategoryForm,
  type CategoryFormValue,
} from "../features/categories/CategoryForm";

const PAGE_SIZE_OPTIONS = [25, 50, 100, 250] as const;
const DIRECTION_CODES = [
  "all",
  "en_ru",
  "ru_en",
  "en_de",
  "de_en",
  "ru_de",
  "de_ru",
] as const;
const LANGUAGE_OPTIONS = [
  { value: "all", label: "Все" },
  { value: "ru", label: "Русский" },
  { value: "en", label: "Английский" },
  { value: "de", label: "Немецкий" },
] as const;

function directionTitle(item: TrainingDirection) {
  return `${item.source_value || "—"} → ${item.target_value || "—"}`;
}

function categoryTitle(item: Category) {
  return `${item.name_ru} / ${item.name_en} / ${item.name_de}`;
}

function directionCodeLabel(value: string) {
  if (value === "all") return "Все";
  return value;
}

function oppositeSortDir(current: DirectionSortDir): DirectionSortDir {
  return current === "asc" ? "desc" : "asc";
}

function compareDirectionField(
  a: TrainingDirection,
  b: TrainingDirection,
  field: DirectionSortBy,
  dir: DirectionSortDir,
) {
  const av = a[field];
  const bv = b[field];

  let result = 0;
  if (typeof av === "number" && typeof bv === "number") {
    result = av - bv;
  } else if (typeof av === "boolean" && typeof bv === "boolean") {
    result = Number(av) - Number(bv);
  } else {
    result = String(av ?? "").localeCompare(String(bv ?? ""), "ru", {
      numeric: true,
      sensitivity: "base",
    });
  }

  if (result === 0) {
    result = a.direction_id - b.direction_id;
  }

  return dir === "asc" ? result : -result;
}

function sortDirectionItems(
  items: TrainingDirection[],
  field: DirectionSortBy,
  dir: DirectionSortDir,
) {
  return [...items].sort((a, b) => compareDirectionField(a, b, field, dir));
}

function SortHeader({
  label,
  field,
  sortBy,
  sortDir,
  onSort,
}: {
  label: string;
  field: DirectionSortBy;
  sortBy: DirectionSortBy;
  sortDir: DirectionSortDir;
  onSort: (field: DirectionSortBy) => void;
}) {
  const active = sortBy === field;
  return (
    <button className="th-btn" type="button" onClick={() => onSort(field)}>
      {label}
      {active ? (sortDir === "asc" ? " ↑" : " ↓") : ""}
    </button>
  );
}

export function CategoriesPage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [editing, setEditing] = useState<Category | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [toast, setToast] = useState<AppToast | null>(null);

  function notify(kind: ToastKind, message: string) {
    setToast({ id: Date.now(), kind, message });
  }

  function showError(message: string) {
    setError(message);
    notify("error", message);
  }

  const [directions, setDirections] = useState<TrainingDirection[]>([]);
  const [directionsTotal, setDirectionsTotal] = useState(0);
  const [directionsPage, setDirectionsPage] = useState(1);
  const [directionsPageSize, setDirectionsPageSize] = useState<number>(50);
  const [directionsTotalPages, setDirectionsTotalPages] = useState(0);
  const [directionSearch, setDirectionSearch] = useState("");
  const [debouncedDirectionSearch, setDebouncedDirectionSearch] = useState("");
  const [directionCode, setDirectionCode] = useState("all");
  const [sourceLangCode, setSourceLangCode] = useState("all");
  const [targetLangCode, setTargetLangCode] = useState("all");
  const [sortBy, setSortBy] = useState<DirectionSortBy>("direction_id");
  const [sortDir, setSortDir] = useState<DirectionSortDir>("asc");
  const [selectedSortBy, setSelectedSortBy] =
    useState<DirectionSortBy>("direction_id");
  const [selectedSortDir, setSelectedSortDir] =
    useState<DirectionSortDir>("asc");

  const [selectedDirections, setSelectedDirections] = useState<
    TrainingDirection[]
  >([]);
  const [storedDirectionCount, setStoredDirectionCount] = useState(0);
  const [
    selectionLoadedFromRepresentatives,
    setSelectionLoadedFromRepresentatives,
  ] = useState(false);
  const [directionsLoading, setDirectionsLoading] = useState(false);
  const [savingComposition, setSavingComposition] = useState(false);

  async function load() {
    try {
      const data = await fetchCategories();
      setCategories(data);
    } catch (e) {
      showError(extractApiError(e));
    }
  }

  async function loadDirections() {
    setDirectionsLoading(true);
    try {
      const data = await fetchDirections({
        page: directionsPage,
        page_size: directionsPageSize,
        q: debouncedDirectionSearch || undefined,
        direction_code: directionCode,
        source_lang_code: sourceLangCode,
        target_lang_code: targetLangCode,
        active: "true",
        unassigned: "true",
        sort_by: sortBy,
        sort_dir: sortDir,
      });
      setDirections(data.items);
      setDirectionsTotal(data.total);
      setDirectionsTotalPages(data.pagination.total_pages);
    } catch (e) {
      showError(extractApiError(e));
    } finally {
      setDirectionsLoading(false);
    }
  }

  async function loadCategoryComposition(category: Category) {
    setDirectionsLoading(true);
    setSelectedDirections([]);
    setStoredDirectionCount(0);
    setSelectionLoadedFromRepresentatives(false);
    try {
      const data = await fetchCategoryDirectionSelection(category.id, {
        sort_by: selectedSortBy,
        sort_dir: selectedSortDir,
      });
      setSelectedDirections(data.items);
      setStoredDirectionCount(
        data.associated_direction_count ?? data.items.length,
      );
      setSelectionLoadedFromRepresentatives(
        Boolean(data.has_explicit_representatives),
      );
    } catch (e) {
      showError(extractApiError(e));
    } finally {
      setDirectionsLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => {
      setToast((current) => (current?.id === toast.id ? null : current));
    }, 3000);
    return () => window.clearTimeout(timer);
  }, [toast]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDirectionsPage(1);
      setDebouncedDirectionSearch(directionSearch.trim());
    }, 350);
    return () => window.clearTimeout(timer);
  }, [directionSearch]);

  useEffect(() => {
    void loadDirections();
  }, [
    directionsPage,
    directionsPageSize,
    debouncedDirectionSearch,
    directionCode,
    sourceLangCode,
    targetLangCode,
    sortBy,
    sortDir,
  ]);

  async function submitCreate(payload: CategoryFormValue) {
    setError(null);
    setSuccess(null);
    try {
      const created = await createCategory(payload);
      setSuccess(`Категория #${created.id} создана.`);
      notify("success", `Категория #${created.id} создана.`);
      await load();
    } catch (e) {
      showError(extractApiError(e));
    }
  }

  async function submitUpdate(payload: CategoryFormValue) {
    if (!editing) return;
    setError(null);
    setSuccess(null);
    try {
      const updated = await updateCategory(editing.id, payload);
      setSuccess(`Категория #${updated.id} обновлена.`);
      notify("success", `Категория #${updated.id} обновлена.`);
      setEditing(updated);
      await load();
    } catch (e) {
      showError(extractApiError(e));
    }
  }

  function selectCategory(item: Category) {
    setEditing(item);
    setError(null);
    setSuccess(null);
    void loadCategoryComposition(item);
  }

  function selectDirectionRepresentative(item: TrainingDirection) {
    setSelectedDirections((current) => {
      const existingIndex = current.findIndex(
        (selected) => selected.concept_id === item.concept_id,
      );
      const next = [...current];
      if (existingIndex >= 0) {
        next[existingIndex] = item;
      } else {
        next.push(item);
      }
      return sortDirectionItems(next, selectedSortBy, selectedSortDir);
    });
  }

  function removeSelectedConcept(conceptId: number) {
    setSelectedDirections((current) =>
      current.filter((item) => item.concept_id !== conceptId),
    );
  }

  function moveSelected(index: number, delta: number) {
    setSelectedDirections((current) => {
      const nextIndex = index + delta;
      if (nextIndex < 0 || nextIndex >= current.length) return current;
      const copy = [...current];
      const [item] = copy.splice(index, 1);
      copy.splice(nextIndex, 0, item);
      return copy;
    });
  }

  async function saveCategoryComposition() {
    if (!editing) return;
    setSavingComposition(true);
    setError(null);
    setSuccess(null);
    try {
      const response = await updateCategoryDirections(editing.id, {
        direction_ids: selectedDirections.map((item) => item.direction_id),
        mode: "replace",
      });
      const message = `Состав категории сохранён. Выбрано карточек: ${response.concept_count ?? 0}. Связано направлений: ${response.associated_direction_count ?? response.affected_directions ?? 0}.`;
      setSuccess(message);
      notify("success", message);
      setDirectionsPage(1);
      await loadDirections();
      await loadCategoryComposition(editing);
    } catch (e) {
      showError(extractApiError(e));
    } finally {
      setSavingComposition(false);
    }
  }

  function handleSort(field: DirectionSortBy) {
    if (field === sortBy) {
      setSortDir((current) => oppositeSortDir(current));
    } else {
      setSortBy(field);
      setSortDir("asc");
    }
    setDirectionsPage(1);
  }

  function handleSelectedSort(field: DirectionSortBy) {
    const nextDir =
      field === selectedSortBy ? oppositeSortDir(selectedSortDir) : "asc";
    setSelectedSortBy(field);
    setSelectedSortDir(nextDir);
    setSelectedDirections((current) =>
      sortDirectionItems(current, field, nextDir),
    );
  }

  const selectedConceptIds = useMemo(
    () => new Set(selectedDirections.map((item) => item.concept_id)),
    [selectedDirections],
  );

  const availableDirections = useMemo(
    () => directions.filter((item) => !selectedConceptIds.has(item.concept_id)),
    [directions, selectedConceptIds],
  );

  return (
    <div className="stack">
      <FixedToast toast={toast} />
      {error ? (
        <div className="visually-hidden" role="alert">
          {error}
        </div>
      ) : null}
      {success ? (
        <div className="visually-hidden" role="status">
          {success}
        </div>
      ) : null}
      <PageHeader
        title="Категории"
        subtitle="Управление категориями и составом словарных карточек. Администратор выбирает одно направление как представитель карточки, а сервер связывает с категорией все смежные направления этой карточки."
        actions={
          <button className="btn btn-secondary" onClick={() => void load()}>
            Обновить
          </button>
        }
      />
      <div className="grid-2">
        <div className="card stack">
          <h2>Создать категорию</h2>
          <CategoryForm
            submitLabel="Создать категорию"
            onSubmit={submitCreate}
          />
        </div>
        <div className="card stack">
          <h2>Редактировать категорию</h2>
          {editing ? (
            <CategoryForm
              initial={editing}
              submitLabel="Сохранить категорию"
              onSubmit={submitUpdate}
            />
          ) : (
            <div className="notice">Выбери категорию из списка ниже.</div>
          )}
        </div>
      </div>
      <div className="card table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Код категории</th>
              <th>Название RU</th>
              <th>Название EN</th>
              <th>Название DE</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {categories.map((item) => (
              <tr
                key={item.id}
                className={editing?.id === item.id ? "row-selected" : undefined}
              >
                <td>{item.id}</td>
                <td>{item.slug}</td>
                <td>{item.name_ru}</td>
                <td>{item.name_en}</td>
                <td>{item.name_de}</td>
                <td>
                  <button
                    className="btn btn-secondary"
                    onClick={() => selectCategory(item)}
                  >
                    Редактировать
                  </button>
                </td>
              </tr>
            ))}
            {categories.length === 0 ? (
              <tr>
                <td colSpan={6}>Категории пока не созданы.</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>

      {editing ? (
        <div className="card stack">
          <div className="section-title-row">
            <div>
              <h2>Состав категории</h2>
              <p className="page-subtitle">
                {categoryTitle(editing)}. Выбрано карточек:{" "}
                {selectedDirections.length}. Связанных направлений:{" "}
                {storedDirectionCount}.
              </p>
            </div>
            <button
              className="btn btn-primary"
              onClick={() => void saveCategoryComposition()}
              disabled={savingComposition || directionsLoading}
            >
              {savingComposition ? "Сохранение..." : "Сохранить состав"}
            </button>
          </div>

          {!selectionLoadedFromRepresentatives &&
          selectedDirections.length > 0 ? (
            <div className="notice">
              Для этой категории найдены старые связи без сохранённого порядка
              выбора. Интерфейс показал по одному представителю на карточку.
              После сохранения порядок будет зафиксирован.
            </div>
          ) : null}

          <div className="category-selection-layout">
            <div className="category-selection-panel stack">
              <div className="section-title-row compact-title-row">
                <div>
                  <h3>Отобранные карточки</h3>
                  <p className="muted">
                    Показан выбранный представитель. В категории сохраняется вся
                    карточка.
                  </p>
                </div>
                <span className="badge badge-ok">
                  {selectedDirections.length}
                </span>
              </div>
              <div className="table-wrap selection-table-wrap">
                <table className="table compact">
                  <thead>
                    <tr>
                      <th>№</th>
                      <th>
                        <SortHeader
                          label="ID"
                          field="direction_id"
                          sortBy={selectedSortBy}
                          sortDir={selectedSortDir}
                          onSort={handleSelectedSort}
                        />
                      </th>
                      <th>
                        <SortHeader
                          label="Направление"
                          field="direction_code"
                          sortBy={selectedSortBy}
                          sortDir={selectedSortDir}
                          onSort={handleSelectedSort}
                        />
                      </th>
                      <th>
                        <SortHeader
                          label="Источник"
                          field="source_value"
                          sortBy={selectedSortBy}
                          sortDir={selectedSortDir}
                          onSort={handleSelectedSort}
                        />
                      </th>
                      <th>
                        <SortHeader
                          label="Цель"
                          field="target_value"
                          sortBy={selectedSortBy}
                          sortDir={selectedSortDir}
                          onSort={handleSelectedSort}
                        />
                      </th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {selectedDirections.map((item, index) => (
                      <tr
                        key={`${item.concept_id}-${item.direction_id}`}
                        className="row-selected"
                      >
                        <td>{index + 1}</td>
                        <td>{item.direction_id}</td>
                        <td>
                          <span className="badge badge-muted">
                            {item.direction_code}
                          </span>
                        </td>
                        <td>{item.source_value || "—"}</td>
                        <td>{item.target_value || "—"}</td>
                        <td>
                          <div className="row-actions">
                            <button
                              className="btn btn-secondary btn-small"
                              type="button"
                              disabled={index === 0}
                              onClick={() => moveSelected(index, -1)}
                            >
                              ↑
                            </button>
                            <button
                              className="btn btn-secondary btn-small"
                              type="button"
                              disabled={index === selectedDirections.length - 1}
                              onClick={() => moveSelected(index, 1)}
                            >
                              ↓
                            </button>
                            <button
                              className="btn btn-danger btn-small"
                              type="button"
                              onClick={() =>
                                removeSelectedConcept(item.concept_id)
                              }
                            >
                              Убрать
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                    {selectedDirections.length === 0 ? (
                      <tr>
                        <td colSpan={6}>Пока ничего не выбрано.</td>
                      </tr>
                    ) : null}
                  </tbody>
                </table>
              </div>
            </div>

            <div className="category-selection-panel stack">
              <div className="section-title-row compact-title-row">
                <div>
                  <h3>Банк направлений</h3>
                  <p className="muted">
                    Показаны только свободные карточки, не закреплённые за
                    другими категориями.
                  </p>
                </div>
                <span className="badge badge-muted">{directionsTotal}</span>
              </div>

              <div className="toolbar">
                <div className="toolbar-field grow">
                  <label className="label">Поиск</label>
                  <input
                    className="input"
                    value={directionSearch}
                    placeholder="исходная форма, целевая форма, код направления"
                    onChange={(e) => setDirectionSearch(e.target.value)}
                  />
                </div>
                <div className="toolbar-field small">
                  <label className="label">Направление</label>
                  <select
                    className="select"
                    value={directionCode}
                    onChange={(e) => {
                      setDirectionCode(e.target.value);
                      setDirectionsPage(1);
                    }}
                  >
                    {DIRECTION_CODES.map((code) => (
                      <option key={code} value={code}>
                        {directionCodeLabel(code)}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="toolbar-field small">
                  <label className="label">Источник</label>
                  <select
                    className="select"
                    value={sourceLangCode}
                    onChange={(e) => {
                      setSourceLangCode(e.target.value);
                      setDirectionsPage(1);
                    }}
                  >
                    {LANGUAGE_OPTIONS.map((lang) => (
                      <option key={lang.value} value={lang.value}>
                        {lang.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="toolbar-field small">
                  <label className="label">Цель</label>
                  <select
                    className="select"
                    value={targetLangCode}
                    onChange={(e) => {
                      setTargetLangCode(e.target.value);
                      setDirectionsPage(1);
                    }}
                  >
                    {LANGUAGE_OPTIONS.map((lang) => (
                      <option key={lang.value} value={lang.value}>
                        {lang.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="toolbar-field small">
                  <label className="label">На странице</label>
                  <select
                    className="select"
                    value={directionsPageSize}
                    onChange={(e) => {
                      setDirectionsPageSize(Number(e.target.value));
                      setDirectionsPage(1);
                    }}
                  >
                    {PAGE_SIZE_OPTIONS.map((size) => (
                      <option key={size} value={size}>
                        {size}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="pagination-bar">
                <div className="pagination-info">
                  Страница {directionsPage} из{" "}
                  {Math.max(directionsTotalPages, 1)}. Свободно найдено:{" "}
                  {directionsTotal}. Доступно на странице:{" "}
                  {availableDirections.length}.
                </div>
                <div className="pagination-actions">
                  <button
                    className="btn btn-secondary"
                    disabled={directionsPage <= 1 || directionsLoading}
                    onClick={() => setDirectionsPage(1)}
                  >
                    Первая
                  </button>
                  <button
                    className="btn btn-secondary"
                    disabled={directionsPage <= 1 || directionsLoading}
                    onClick={() =>
                      setDirectionsPage((page) => Math.max(1, page - 1))
                    }
                  >
                    Назад
                  </button>
                  <button
                    className="btn btn-secondary"
                    disabled={
                      directionsTotalPages === 0 ||
                      directionsPage >= directionsTotalPages ||
                      directionsLoading
                    }
                    onClick={() => setDirectionsPage((page) => page + 1)}
                  >
                    Вперёд
                  </button>
                  <button
                    className="btn btn-secondary"
                    disabled={
                      directionsTotalPages === 0 ||
                      directionsPage >= directionsTotalPages ||
                      directionsLoading
                    }
                    onClick={() => setDirectionsPage(directionsTotalPages)}
                  >
                    Последняя
                  </button>
                </div>
              </div>

              <div className="table-wrap selection-table-wrap">
                <table className="table compact">
                  <thead>
                    <tr>
                      <th>Выбор</th>
                      <th>
                        <SortHeader
                          label="ID"
                          field="direction_id"
                          sortBy={sortBy}
                          sortDir={sortDir}
                          onSort={handleSort}
                        />
                      </th>
                      <th>
                        <SortHeader
                          label="Направление"
                          field="direction_code"
                          sortBy={sortBy}
                          sortDir={sortDir}
                          onSort={handleSort}
                        />
                      </th>
                      <th>
                        <SortHeader
                          label="Источник"
                          field="source_value"
                          sortBy={sortBy}
                          sortDir={sortDir}
                          onSort={handleSort}
                        />
                      </th>
                      <th>
                        <SortHeader
                          label="Цель"
                          field="target_value"
                          sortBy={sortBy}
                          sortDir={sortDir}
                          onSort={handleSort}
                        />
                      </th>
                      <th>
                        <SortHeader
                          label="Уровень"
                          field="cefr_level"
                          sortBy={sortBy}
                          sortDir={sortDir}
                          onSort={handleSort}
                        />
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {availableDirections.map((item) => (
                      <tr key={item.direction_id}>
                        <td>
                          <button
                            className="btn btn-secondary btn-small"
                            type="button"
                            onClick={() => selectDirectionRepresentative(item)}
                            aria-label={`Выбрать ${directionTitle(item)}`}
                          >
                            Выбрать
                          </button>
                        </td>
                        <td>{item.direction_id}</td>
                        <td>
                          <span className="badge badge-muted">
                            {item.direction_code}
                          </span>
                        </td>
                        <td>{item.source_value || "—"}</td>
                        <td>{item.target_value || "—"}</td>
                        <td>{item.cefr_level || "—"}</td>
                      </tr>
                    ))}
                    {!directionsLoading && availableDirections.length === 0 ? (
                      <tr>
                        <td colSpan={6}>Свободные направления не найдены.</td>
                      </tr>
                    ) : null}
                    {directionsLoading ? (
                      <tr>
                        <td colSpan={6}>Загрузка...</td>
                      </tr>
                    ) : null}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
