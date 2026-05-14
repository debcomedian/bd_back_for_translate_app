import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { fetchCategories } from "../shared/api/categories";
import { fetchWords, updateWord } from "../shared/api/words";
import { extractApiError } from "../shared/api/client";
import { WordForm } from "../features/words/WordForm";
import { PageHeader } from "../shared/ui/PageHeader";
import type { Category, Word } from "../types";

export function WordEditPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [categories, setCategories] = useState<Category[]>([]);
  const [word, setWord] = useState<Word | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    async function load() {
      try {
        const [categoriesData, wordsData] = await Promise.all([
          fetchCategories(),
          fetchWords(),
        ]);
        setCategories(categoriesData);
        const found = wordsData.find((item) => String(item.id) === String(id));
        if (!found) {
          setError("Запись не найдена в текущем списке.");
          return;
        }
        setWord(found);
      } catch (e) {
        setError(extractApiError(e));
      }
    }
    void load();
  }, [id]);

  const meta = useMemo(() => word?.meta_base, [word]);

  return (
    <div className="stack">
      <PageHeader
        title={`Редактировать старую словарную запись #${id}`}
        subtitle="Изменение старой словарной записи и просмотр рассчитанных метаданных."
        actions={
          <button
            className="btn btn-secondary"
            onClick={() => navigate("/words")}
          >
            Назад к списку
          </button>
        }
      />
      {error ? <div className="error">{error}</div> : null}
      {success ? <div className="success">{success}</div> : null}
      {word ? (
        <>
          <div className="card">
            <WordForm
              key={`${word.id}-${word.updated_at ?? "new"}`}
              initial={word}
              categories={categories}
              submitLabel="Сохранить изменения"
              onSubmit={async (payload) => {
                setError(null);
                setSuccess(null);
                try {
                  const updated = await updateWord(word.id, payload);
                  setWord(updated);
                  setSuccess(`Запись #${updated.id} обновлена.`);
                } catch (e) {
                  setError(extractApiError(e));
                }
              }}
            />
          </div>
          <div className="card stack">
            <h2>Текущие метаданные</h2>
            {meta ? (
              <dl className="kv">
                <dt>CEFR</dt>
                <dd>{meta.meta_cefr_level ?? "—"}</dd>
                <dt>Важность</dt>
                <dd>{meta.meta_importance_score}</dd>
                <dt>Частотная группа</dt>
                <dd>{meta.meta_freq_bucket}</dd>
                <dt>Длина</dt>
                <dd>{meta.meta_length_chars}</dd>
                <dt>Базовая сложность</dt>
                <dd>{meta.meta_base_difficulty}</dd>
                <dt>Рассчитано</dt>
                <dd>{meta.calculated_at}</dd>
              </dl>
            ) : (
              <div className="notice">Метаданные ещё не рассчитаны.</div>
            )}
          </div>
        </>
      ) : null}
    </div>
  );
}
