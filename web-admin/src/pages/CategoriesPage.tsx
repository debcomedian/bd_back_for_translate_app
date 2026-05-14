import { useEffect, useState } from "react";
import {
  createCategory,
  fetchCategories,
  updateCategory,
} from "../shared/api/categories";
import { extractApiError } from "../shared/api/client";
import { PageHeader } from "../shared/ui/PageHeader";
import type { Category } from "../types";
import {
  CategoryForm,
  type CategoryFormValue,
} from "../features/categories/CategoryForm";

export function CategoriesPage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [editing, setEditing] = useState<Category | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

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
      <PageHeader
        title="Категории"
        subtitle="Управление категориями концептов для направленной словарной модели."
        actions={
          <button className="btn btn-secondary" onClick={() => void load()}>
            Обновить
          </button>
        }
      />
      {error ? <div className="error">{error}</div> : null}
      {success ? <div className="success">{success}</div> : null}
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
              <th>Slug</th>
              <th>RU</th>
              <th>EN</th>
              <th>DE</th>
              <th>Тип сущности</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {categories.map((item) => (
              <tr key={item.id}>
                <td>{item.id}</td>
                <td>{item.slug}</td>
                <td>{item.name_ru}</td>
                <td>{item.name_en}</td>
                <td>{item.name_de}</td>
                <td>{item.entity}</td>
                <td>
                  <button
                    className="btn btn-secondary"
                    onClick={() => setEditing(item)}
                  >
                    Редактировать
                  </button>
                </td>
              </tr>
            ))}
            {categories.length === 0 ? (
              <tr>
                <td colSpan={7}>Категории пока не созданы.</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
