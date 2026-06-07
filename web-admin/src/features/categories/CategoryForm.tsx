import { FormEvent, useEffect, useMemo, useState } from "react";
import type { Category } from "../../types";

export type CategoryFormValue = {
  slug: string;
  name_ru: string;
  name_en: string;
  name_de: string;
  entity: string;
};

function normalize(initial?: Partial<Category>): CategoryFormValue {
  return {
    slug: initial?.slug ?? "",
    name_ru: initial?.name_ru ?? "",
    name_en: initial?.name_en ?? "",
    name_de: initial?.name_de ?? "",
    entity: initial?.entity ?? "concept",
  };
}

function hasDigit(value: string) {
  return /\d/.test(value);
}

function validate(form: CategoryFormValue): string | null {
  const slug = form.slug.trim();
  if (!slug) return "Код категории не должен быть пустым.";
  if (hasDigit(slug)) return "Код категории не должен содержать цифры.";
  if (!/^[a-z_-]+$/.test(slug)) {
    return "Код категории может содержать только латинские буквы, дефис и подчёркивание.";
  }

  const labels: Array<[keyof CategoryFormValue, string]> = [
    ["name_ru", "Название RU"],
    ["name_en", "Название EN"],
    ["name_de", "Название DE"],
  ];

  for (const [key, label] of labels) {
    const value = String(form[key]).trim();
    if (!value) return `${label} не должно быть пустым.`;
    if (hasDigit(value)) return `${label} не должно содержать цифры.`;
  }

  return null;
}

export function CategoryForm({
  initial,
  submitLabel,
  onSubmit,
}: {
  initial?: Partial<Category>;
  submitLabel: string;
  onSubmit: (value: CategoryFormValue) => Promise<void>;
}) {
  const [form, setForm] = useState<CategoryFormValue>(() => normalize(initial));
  const [loading, setLoading] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);

  useEffect(() => {
    setForm(normalize(initial));
    setLocalError(null);
  }, [initial?.id]);

  const validationError = useMemo(() => validate(form), [form]);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const error = validate(form);
    if (error) {
      setLocalError(error);
      return;
    }
    setLoading(true);
    setLocalError(null);
    try {
      await onSubmit({
        ...form,
        slug: form.slug.trim().toLowerCase(),
        name_ru: form.name_ru.trim(),
        name_en: form.name_en.trim(),
        name_de: form.name_de.trim(),
        entity: "concept",
      });
      if (!initial?.id) {
        setForm(normalize());
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className="stack" onSubmit={handleSubmit}>
      {localError ? <div className="error">{localError}</div> : null}
      <div>
        <label className="label">Код категории</label>
        <input
          className="input"
          value={form.slug}
          placeholder="например: household"
          onChange={(e) => setForm((p) => ({ ...p, slug: e.target.value }))}
          required
        />
      </div>
      <div className="grid-3">
        <div>
          <label className="label">Название RU</label>
          <input
            className="input"
            value={form.name_ru}
            onChange={(e) => setForm((p) => ({ ...p, name_ru: e.target.value }))}
            required
          />
        </div>
        <div>
          <label className="label">Название EN</label>
          <input
            className="input"
            value={form.name_en}
            onChange={(e) => setForm((p) => ({ ...p, name_en: e.target.value }))}
            required
          />
        </div>
        <div>
          <label className="label">Название DE</label>
          <input
            className="input"
            value={form.name_de}
            onChange={(e) => setForm((p) => ({ ...p, name_de: e.target.value }))}
            required
          />
        </div>
      </div>
      <button className="btn btn-primary" type="submit" disabled={loading || Boolean(validationError)}>
        {loading ? "Сохранение..." : submitLabel}
      </button>
      {validationError ? <div className="muted">{validationError}</div> : null}
    </form>
  );
}
