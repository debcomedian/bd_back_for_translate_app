import { FormEvent, useState } from 'react';
import type { Category } from '../../types';

export type CategoryFormValue = {
  slug: string;
  name_ru: string;
  name_en: string;
  name_de: string;
  entity: string;
};

function normalize(initial?: Partial<Category>): CategoryFormValue {
  return {
    slug: initial?.slug ?? '',
    name_ru: initial?.name_ru ?? '',
    name_en: initial?.name_en ?? '',
    name_de: initial?.name_de ?? '',
    entity: initial?.entity ?? 'concept',
  };
}

export function CategoryForm({ initial, submitLabel, onSubmit }: { initial?: Partial<Category>; submitLabel: string; onSubmit: (value: CategoryFormValue) => Promise<void>; }) {
  const [form, setForm] = useState<CategoryFormValue>(() => normalize(initial));
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    try {
      await onSubmit(form);
      if (!initial?.id) {
        setForm(normalize());
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className="stack" onSubmit={handleSubmit}>
      <div className="grid-2">
        <div>
          <label className="label">Slug</label>
          <input className="input" value={form.slug} onChange={(e) => setForm((p) => ({ ...p, slug: e.target.value }))} required />
        </div>
        <div>
          <label className="label">Entity</label>
          <select className="select" value={form.entity} onChange={(e) => setForm((p) => ({ ...p, entity: e.target.value }))}>
            <option value="concept">concept</option>
            <option value="word">word</option>
          </select>
        </div>
      </div>
      <div className="grid-3">
        <div>
          <label className="label">Название RU</label>
          <input className="input" value={form.name_ru} onChange={(e) => setForm((p) => ({ ...p, name_ru: e.target.value }))} required />
        </div>
        <div>
          <label className="label">Title EN</label>
          <input className="input" value={form.name_en} onChange={(e) => setForm((p) => ({ ...p, name_en: e.target.value }))} required />
        </div>
        <div>
          <label className="label">Titel DE</label>
          <input className="input" value={form.name_de} onChange={(e) => setForm((p) => ({ ...p, name_de: e.target.value }))} required />
        </div>
      </div>
      <button className="btn btn-primary" type="submit" disabled={loading}>{loading ? 'Сохранение...' : submitLabel}</button>
    </form>
  );
}
