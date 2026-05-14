import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { fetchCategories } from "../shared/api/categories";
import { createWord } from "../shared/api/words";
import { extractApiError } from "../shared/api/client";
import { WordForm } from "../features/words/WordForm";
import { PageHeader } from "../shared/ui/PageHeader";
import type { Category } from "../types";

export function WordCreatePage() {
  const navigate = useNavigate();
  const [categories, setCategories] = useState<Category[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    fetchCategories()
      .then(setCategories)
      .catch((e) => setError(extractApiError(e)));
  }, []);

  return (
    <div className="stack">
      <PageHeader
        title="Создать старую словарную запись"
        subtitle="Ручное создание одной старой словарной записи."
      />
      {error ? <div className="error">{error}</div> : null}
      {success ? <div className="success">{success}</div> : null}
      <div className="card">
        <WordForm
          categories={categories}
          submitLabel="Создать запись"
          onSubmit={async (payload) => {
            setError(null);
            setSuccess(null);
            try {
              const created = await createWord(payload);
              setSuccess(`Запись #${created.id} создана.`);
              navigate(`/words/${created.id}/edit`, { replace: true });
            } catch (e) {
              setError(extractApiError(e));
            }
          }}
        />
      </div>
    </div>
  );
}
