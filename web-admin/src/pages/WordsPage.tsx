import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { fetchCategories } from '../shared/api/categories';
import { fetchWords } from '../shared/api/words';
import { extractApiError } from '../shared/api/client';
import { PageHeader } from '../shared/ui/PageHeader';
import type { Category, Word } from '../types';

export function WordsPage() {
  const [words, setWords] = useState<Word[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const [wordsData, categoriesData] = await Promise.all([fetchWords(), fetchCategories()]);
      setWords(wordsData);
      setCategories(categoriesData);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const categoryMap = useMemo(() => new Map(categories.map((item) => [item.id, item.name_ru])), [categories]);

  return (
    <div className="stack">
      <PageHeader
        title="Слова"
        subtitle="Главный рабочий экран администратора. Просмотр слов, переход к созданию и редактированию."
        actions={
          <>
            <button className="btn btn-secondary" onClick={() => void load()}>Обновить</button>
            <Link className="btn btn-primary" to="/words/new">Создать слово</Link>
          </>
        }
      />

      {error ? <div className="error">{error}</div> : null}
      {loading ? <div className="notice">Загрузка...</div> : null}

      <div className="card table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Lang</th>
              <th>RU</th>
              <th>EN</th>
              <th>DE</th>
              <th>Категория</th>
              <th>CEFR</th>
              <th>Difficulty</th>
              <th>Freq</th>
              <th>Active</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {words.map((word) => (
              <tr key={word.id}>
                <td>{word.id}</td>
                <td>{word.lang_code}</td>
                <td>{word.word_ru ?? '—'}</td>
                <td>{word.word_en ?? '—'}</td>
                <td>{word.word_de ?? '—'}</td>
                <td>{word.category_id ? categoryMap.get(word.category_id) ?? word.category_id : '—'}</td>
                <td>{word.meta_base?.meta_cefr_level ?? '—'}</td>
                <td>{word.meta_base?.meta_base_difficulty ?? '—'}</td>
                <td>{word.meta_base?.meta_freq_bucket ?? '—'}</td>
                <td>
                  <span className={`badge ${word.is_active ? 'badge-ok' : 'badge-muted'}`}>
                    {word.is_active ? 'active' : 'inactive'}
                  </span>
                </td>
                <td><Link className="btn btn-secondary" to={`/words/${word.id}/edit`}>Редактировать</Link></td>
              </tr>
            ))}
            {!loading && words.length === 0 ? (
              <tr>
                <td colSpan={11}>Слова ещё не созданы.</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
