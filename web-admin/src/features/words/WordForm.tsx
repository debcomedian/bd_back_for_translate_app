import { FormEvent, useMemo, useState } from "react";
import type { Category, Word } from "../../types";
import type { WordPayload } from "../../shared/api/words";

function normalize(initial?: Partial<Word>): WordPayload {
  return {
    lang_code: initial?.lang_code ?? "en",
    word_ru: initial?.word_ru ?? "",
    word_en: initial?.word_en ?? "",
    word_de: initial?.word_de ?? "",
    transcription_ru: initial?.transcription_ru ?? "",
    transcription_en: initial?.transcription_en ?? "",
    transcription_de: initial?.transcription_de ?? "",
    source_ref: initial?.source_ref ?? "",
    category_id: initial?.category_id ?? null,
    is_active: initial?.is_active ?? true,
  };
}

export function WordForm({
  initial,
  categories,
  submitLabel,
  onSubmit,
}: {
  initial?: Partial<Word>;
  categories: Category[];
  submitLabel: string;
  onSubmit: (payload: WordPayload) => Promise<void>;
}) {
  const [form, setForm] = useState<WordPayload>(() => normalize(initial));
  const [loading, setLoading] = useState(false);
  const categoryOptions = useMemo(() => categories, [categories]);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    try {
      await onSubmit({
        ...form,
        word_ru: form.word_ru || null,
        word_en: form.word_en || null,
        word_de: form.word_de || null,
        transcription_ru: form.transcription_ru || null,
        transcription_en: form.transcription_en || null,
        transcription_de: form.transcription_de || null,
        source_ref: form.source_ref || null,
      });
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className="stack" onSubmit={handleSubmit}>
      <div className="grid-3">
        <div>
          <label className="label">Базовый язык</label>
          <select
            className="select"
            value={form.lang_code}
            onChange={(e) =>
              setForm((p) => ({ ...p, lang_code: e.target.value }))
            }
          >
            <option value="en">en</option>
            <option value="ru">ru</option>
            <option value="de">de</option>
          </select>
        </div>
        <div>
          <label className="label">Категория</label>
          <select
            className="select"
            value={form.category_id ?? ""}
            onChange={(e) =>
              setForm((p) => ({
                ...p,
                category_id: e.target.value ? Number(e.target.value) : null,
              }))
            }
          >
            <option value="">Без категории</option>
            {categoryOptions.map((item) => (
              <option key={item.id} value={item.id}>
                {item.name_ru}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="label">Активность</label>
          <select
            className="select"
            value={form.is_active ? "true" : "false"}
            onChange={(e) =>
              setForm((p) => ({ ...p, is_active: e.target.value === "true" }))
            }
          >
            <option value="true">Активно</option>
            <option value="false">Отключено</option>
          </select>
        </div>
      </div>
      <div className="grid-3">
        <div>
          <label className="label">Слово RU</label>
          <input
            className="input"
            value={form.word_ru ?? ""}
            onChange={(e) =>
              setForm((p) => ({ ...p, word_ru: e.target.value }))
            }
          />
        </div>
        <div>
          <label className="label">Слово EN</label>
          <input
            className="input"
            value={form.word_en ?? ""}
            onChange={(e) =>
              setForm((p) => ({ ...p, word_en: e.target.value }))
            }
          />
        </div>
        <div>
          <label className="label">Слово DE</label>
          <input
            className="input"
            value={form.word_de ?? ""}
            onChange={(e) =>
              setForm((p) => ({ ...p, word_de: e.target.value }))
            }
          />
        </div>
      </div>
      <div className="grid-3">
        <div>
          <label className="label">Транскрипция RU</label>
          <input
            className="input"
            value={form.transcription_ru ?? ""}
            onChange={(e) =>
              setForm((p) => ({ ...p, transcription_ru: e.target.value }))
            }
          />
        </div>
        <div>
          <label className="label">Транскрипция EN</label>
          <input
            className="input"
            value={form.transcription_en ?? ""}
            onChange={(e) =>
              setForm((p) => ({ ...p, transcription_en: e.target.value }))
            }
          />
        </div>
        <div>
          <label className="label">Транскрипция DE</label>
          <input
            className="input"
            value={form.transcription_de ?? ""}
            onChange={(e) =>
              setForm((p) => ({ ...p, transcription_de: e.target.value }))
            }
          />
        </div>
      </div>
      <div>
        <label className="label">Источник</label>
        <input
          className="input"
          value={form.source_ref ?? ""}
          onChange={(e) =>
            setForm((p) => ({ ...p, source_ref: e.target.value }))
          }
        />
      </div>
      <button className="btn btn-primary" type="submit" disabled={loading}>
        {loading ? "Сохранение..." : submitLabel}
      </button>
    </form>
  );
}
