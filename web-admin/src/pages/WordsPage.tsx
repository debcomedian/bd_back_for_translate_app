import { useEffect, useMemo, useState } from "react";
import { fetchDirections, updateDirection } from "../shared/api/directions";
import { fetchCategories } from "../shared/api/categories";
import { extractApiError } from "../shared/api/client";
import { PageHeader } from "../shared/ui/PageHeader";
import {
  FixedToast,
  type AppToast,
  type ToastKind,
} from "../shared/ui/FixedToast";
import type {
  Category,
  DirectionListPagination,
  DirectionSortBy,
  DirectionSortDir,
  TrainingDirection,
} from "../types";

const LANGUAGE_OPTIONS = [
  { code: "", label: "Не выбрано" },
  { code: "ru", label: "Русский" },
  { code: "en", label: "Английский" },
  { code: "de", label: "Немецкий" },
];

const DIRECTION_CODES = ["en_ru", "ru_en", "en_de", "de_en", "ru_de", "de_ru"];
const PAGE_SIZE_OPTIONS = [25, 50, 100, 250];
const CEFR_LEVELS = ["A1", "A2", "B1", "B2", "C1", "C2"];

function formatOptional(value?: string | number | boolean | null) {
  if (value === undefined || value === null || value === "") return "—";
  if (typeof value === "boolean") return value ? "да" : "нет";
  return String(value);
}

function formatDateTime(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  const utc3 = new Date(date.getTime() + 3 * 60 * 60 * 1000);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${utc3.getUTCFullYear()}:${pad(utc3.getUTCMonth() + 1)}:${pad(utc3.getUTCDate())} ${pad(utc3.getUTCHours())}:${pad(utc3.getUTCMinutes())}:${pad(utc3.getUTCSeconds())}`;
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

type DirectionEditForm = {
  category_id: string;
  cefr_level: string;
  source_lang_code: string;
  target_lang_code: string;
  source_value: string;
  target_value: string;
  importance_score: string;
  concept_base_difficulty: string;
  source_form_score: string;
  target_form_score: string;
  direction_bias: string;
  synonym_relief: string;
  final_difficulty: string;
  is_active: boolean;
};

function numberString(value?: number | null) {
  if (value === undefined || value === null || Number.isNaN(value)) return "0";
  return String(value);
}

function editFormFromDirection(item: TrainingDirection): DirectionEditForm {
  return {
    category_id: item.category_id ? String(item.category_id) : "",
    cefr_level: item.cefr_level ?? "",
    source_lang_code: item.source_lang_code || "",
    target_lang_code: item.target_lang_code || "",
    source_value: item.source_value ?? "",
    target_value: item.target_value ?? "",
    importance_score: numberString(item.importance_score),
    concept_base_difficulty: numberString(item.concept_base_difficulty),
    source_form_score: numberString(item.source_form_score),
    target_form_score: numberString(item.target_form_score),
    direction_bias: numberString(item.direction_bias),
    synonym_relief: numberString(item.synonym_relief),
    final_difficulty: numberString(item.final_difficulty),
    is_active: item.is_active,
  };
}

function ReadOnlyInput({
  label,
  value,
}: {
  label: string;
  value: string | number | boolean | null | undefined;
}) {
  return (
    <label className="form-field">
      <span>{label}</span>
      <input
        className="input input-readonly"
        value={formatOptional(value)}
        readOnly
      />
    </label>
  );
}

function TextInput({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}) {
  return (
    <label className="form-field">
      <span>{label}</span>
      <input
        className="input"
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
  );
}

function NumberInput({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label className="form-field">
      <span>{label}</span>
      <input
        className="input"
        type="number"
        step="0.001"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
  );
}

function parseRequiredNumber(value: string, label: string) {
  const prepared = value.trim().replace(",", ".");
  if (prepared === "") {
    throw new Error(`Поле "${label}" не должно быть пустым.`);
  }
  const parsed = Number(prepared);
  if (!Number.isFinite(parsed)) {
    throw new Error(`Поле "${label}" должно быть числом.`);
  }
  return parsed;
}

export function WordsPage() {
  const [directions, setDirections] = useState<TrainingDirection[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [pagination, setPagination] = useState<DirectionListPagination>({
    page: 1,
    page_size: 100,
    total: 0,
    total_pages: 0,
    has_prev: false,
    has_next: false,
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
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
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [directionCode, setDirectionCode] = useState("all");
  const [sortKey, setSortKey] = useState<DirectionSortBy>("direction_id");
  const [sortDirection, setSortDirection] = useState<DirectionSortDir>("asc");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(100);
  const [editing, setEditing] = useState<TrainingDirection | null>(null);
  const [editForm, setEditForm] = useState<DirectionEditForm | null>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setPage(1);
      setDebouncedSearch(search.trim());
    }, 350);

    return () => window.clearTimeout(timer);
  }, [search]);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => {
      setToast((current) => (current?.id === toast.id ? null : current));
    }, 3000);
    return () => window.clearTimeout(timer);
  }, [toast]);

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
      showError(extractApiError(e));
    } finally {
      setLoading(false);
    }
  }

  async function loadCategories() {
    try {
      const data = await fetchCategories();
      setCategories(data);
    } catch (e) {
      showError(extractApiError(e));
    }
  }

  useEffect(() => {
    void loadCategories();
  }, []);

  useEffect(() => {
    void load();
  }, [debouncedSearch, directionCode, page, pageSize, sortDirection, sortKey]);

  const directionCodes = useMemo(() => {
    const fromPage = directions
      .map((item) => item.direction_code)
      .filter(Boolean);
    return Array.from(new Set([...DIRECTION_CODES, ...fromPage])).sort();
  }, [directions]);

  const editedDirectionCode =
    editForm &&
    editForm.source_lang_code &&
    editForm.target_lang_code &&
    editForm.source_lang_code !== editForm.target_lang_code
      ? `${editForm.source_lang_code}_${editForm.target_lang_code}`
      : "Не задан";

  const canSaveDirection = Boolean(
    editForm &&
    editForm.source_lang_code &&
    editForm.target_lang_code &&
    editForm.source_lang_code !== editForm.target_lang_code &&
    editForm.source_value.trim() &&
    editForm.target_value.trim(),
  );

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

  function startEdit(item: TrainingDirection) {
    setEditing(item);
    setEditForm(editFormFromDirection(item));
    setSuccess(null);
    setError(null);
  }

  function updateEditForm(patch: Partial<DirectionEditForm>) {
    setEditForm((current) => (current ? { ...current, ...patch } : current));
  }

  function changeSourceLanguage(nextSource: string) {
    setEditForm((current) => {
      if (!current) return current;

      return {
        ...current,
        source_lang_code: nextSource,
        target_lang_code:
          nextSource && nextSource === current.target_lang_code
            ? ""
            : current.target_lang_code,
      };
    });
  }

  function changeTargetLanguage(nextTarget: string) {
    setEditForm((current) => {
      if (!current) return current;

      return {
        ...current,
        source_lang_code:
          nextTarget && nextTarget === current.source_lang_code
            ? ""
            : current.source_lang_code,
        target_lang_code: nextTarget,
      };
    });
  }

  async function saveDirection() {
    if (!editing || !editForm) return;
    setSaving(true);
    setError(null);
    setSuccess(null);

    try {
      if (!canSaveDirection) {
        throw new Error(
          "Выберите разные исходный и целевой языки и заполните обе формы.",
        );
      }
      await updateDirection(editing.direction_id, {
        category_id: editForm.category_id ? Number(editForm.category_id) : 0,
        cefr_level: editForm.cefr_level || "",
        source_lang_code: editForm.source_lang_code,
        target_lang_code: editForm.target_lang_code,
        source_value: editForm.source_value,
        target_value: editForm.target_value,
        importance_score: Math.trunc(
          parseRequiredNumber(editForm.importance_score, "Важность"),
        ),
        concept_base_difficulty: parseRequiredNumber(
          editForm.concept_base_difficulty,
          "Базовая сложность карточки",
        ),
        source_form_score: parseRequiredNumber(
          editForm.source_form_score,
          "Оценка исходной формы",
        ),
        target_form_score: parseRequiredNumber(
          editForm.target_form_score,
          "Оценка целевой формы",
        ),
        direction_bias: parseRequiredNumber(
          editForm.direction_bias,
          "Поправка направления",
        ),
        synonym_relief: parseRequiredNumber(
          editForm.synonym_relief,
          "Снижение за синонимы",
        ),
        final_difficulty: parseRequiredNumber(
          editForm.final_difficulty,
          "Итоговая сложность",
        ),
        is_active: editForm.is_active,
      });
      setSuccess("Направление обновлено.");
      notify("success", "Направление обновлено.");
      setEditing(null);
      setEditForm(null);
      await load();
    } catch (e) {
      showError(extractApiError(e));
    } finally {
      setSaving(false);
    }
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
        title="Направления"
        subtitle="Учебные задания вида источник — цель. Поиск, фильтрация и сортировка выполняются на сервере."
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

      {loading ? <div className="notice">Загрузка направлений...</div> : null}

      <div className="card stack">
        <div className="toolbar">
          <div className="toolbar-field grow">
            <label className="label">Поиск</label>
            <input
              className="input"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="источник, цель, направление, категория, уровень"
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

      {editing && editForm ? (
        <div className="card stack">
          <div className="section-title-row">
            <div>
              <h2>Редактирование направления</h2>
              <p className="page-subtitle">
                {editForm.source_value || "—"} — {editForm.target_value || "—"}
              </p>
            </div>
            <div className="actions">
              <button
                className="btn btn-primary"
                onClick={() => void saveDirection()}
                disabled={saving || !canSaveDirection}
              >
                {saving ? "Сохранение..." : "Сохранить"}
              </button>
              <button
                className="btn btn-secondary"
                onClick={() => setEditing(null)}
              >
                Закрыть
              </button>
            </div>
          </div>

          <div className="edit-fields-title">Основные поля</div>
          <div className="direction-edit-grid">
            <ReadOnlyInput
              label="ID направления"
              value={editing.direction_id}
            />
            <ReadOnlyInput label="ID карточки" value={editing.concept_id} />
            <ReadOnlyInput
              label="ID исходной формы"
              value={editing.source_form_id}
            />
            <ReadOnlyInput
              label="ID целевой формы"
              value={editing.target_form_id}
            />
            <ReadOnlyInput
              label="Код направления"
              value={editedDirectionCode}
            />

            <label className="form-field">
              <span>Исходный язык</span>
              <select
                className="select"
                value={editForm.source_lang_code}
                onChange={(e) => changeSourceLanguage(e.target.value)}
              >
                {LANGUAGE_OPTIONS.map((language) => (
                  <option key={language.code || "none"} value={language.code}>
                    {language.code
                      ? `${language.code} — ${language.label}`
                      : language.label}
                  </option>
                ))}
              </select>
            </label>

            <label className="form-field">
              <span>Целевой язык</span>
              <select
                className="select"
                value={editForm.target_lang_code}
                onChange={(e) => changeTargetLanguage(e.target.value)}
              >
                {LANGUAGE_OPTIONS.map((language) => (
                  <option key={language.code || "none"} value={language.code}>
                    {language.code
                      ? `${language.code} — ${language.label}`
                      : language.label}
                  </option>
                ))}
              </select>
            </label>

            <TextInput
              label="Исходная форма"
              value={editForm.source_value}
              onChange={(value) => updateEditForm({ source_value: value })}
            />
            <TextInput
              label="Целевая форма"
              value={editForm.target_value}
              onChange={(value) => updateEditForm({ target_value: value })}
            />
            <label className="form-field">
              <span>Категория</span>
              <select
                className="select"
                value={editForm.category_id}
                onChange={(e) =>
                  updateEditForm({ category_id: e.target.value })
                }
              >
                <option value="">Без категории</option>
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name_ru || category.slug}
                  </option>
                ))}
              </select>
            </label>

            <label className="form-field">
              <span>Уровень</span>
              <select
                className="select"
                value={editForm.cefr_level}
                onChange={(e) => updateEditForm({ cefr_level: e.target.value })}
              >
                <option value="">Не задан</option>
                {CEFR_LEVELS.map((level) => (
                  <option key={level} value={level}>
                    {level}
                  </option>
                ))}
              </select>
            </label>

            <label className="form-field">
              <span>Активность</span>
              <select
                className="select"
                value={editForm.is_active ? "true" : "false"}
                onChange={(e) =>
                  updateEditForm({ is_active: e.target.value === "true" })
                }
              >
                <option value="true">Активно</option>
                <option value="false">Отключено</option>
              </select>
            </label>

            <ReadOnlyInput
              label="Создано UTC+3"
              value={formatDateTime(editing.created_at)}
            />
          </div>

          <div className="edit-fields-title">Расчётные поля</div>
          <div className="direction-edit-grid">
            <NumberInput
              label="Важность"
              value={editForm.importance_score}
              onChange={(value) => updateEditForm({ importance_score: value })}
            />
            <NumberInput
              label="Базовая сложность карточки"
              value={editForm.concept_base_difficulty}
              onChange={(value) =>
                updateEditForm({ concept_base_difficulty: value })
              }
            />
            <NumberInput
              label="Оценка исходной формы"
              value={editForm.source_form_score}
              onChange={(value) => updateEditForm({ source_form_score: value })}
            />
            <NumberInput
              label="Оценка целевой формы"
              value={editForm.target_form_score}
              onChange={(value) => updateEditForm({ target_form_score: value })}
            />
            <NumberInput
              label="Поправка направления"
              value={editForm.direction_bias}
              onChange={(value) => updateEditForm({ direction_bias: value })}
            />
            <NumberInput
              label="Снижение за синонимы"
              value={editForm.synonym_relief}
              onChange={(value) => updateEditForm({ synonym_relief: value })}
            />
            <NumberInput
              label="Итоговая сложность"
              value={editForm.final_difficulty}
              onChange={(value) => updateEditForm({ final_difficulty: value })}
            />
          </div>

          {!canSaveDirection ? (
            <div className="notice">
              Для сохранения выберите разные языки и заполните исходную и
              целевую форму.
            </div>
          ) : null}
        </div>
      ) : null}

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
                  Уровень{sortLabel(sortKey, sortDirection, "cefr_level")}
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
              <th>Действие</th>
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
                <td>
                  <span
                    className={`badge ${item.is_active ? "badge-ok" : "badge-muted"}`}
                  >
                    {item.is_active ? "активно" : "отключено"}
                  </span>
                </td>
                <td>
                  <button
                    className="btn btn-secondary"
                    onClick={() => startEdit(item)}
                  >
                    Редактировать
                  </button>
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
