import { useEffect, useState } from "react";
import { fetchSnapshot } from "../shared/api/snapshot";
import { recalculateDirections } from "../shared/api/import";
import { extractApiError } from "../shared/api/client";
import { PageHeader } from "../shared/ui/PageHeader";
import type { SnapshotResponse } from "../types";

function CountCard({
  title,
  value,
  hint,
}: {
  title: string;
  value: number;
  hint?: string;
}) {
  return (
    <div className="card stack stat-card">
      <h2>{title}</h2>
      <div className="stat-value">{value}</div>
      {hint ? <div className="muted">{hint}</div> : null}
    </div>
  );
}

export function SnapshotPage() {
  const [data, setData] = useState<SnapshotResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const result = await fetchSnapshot();
      setData(result);
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  async function handleRecalculate() {
    setBusy(true);
    setError(null);
    try {
      await recalculateDirections();
      await load();
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const version = data?.snapshot_version;

  return (
    <div className="stack">
      <PageHeader
        title="Снимок контента"
        subtitle="Полная offline-выгрузка новой направленной модели для мобильного клиента."
        actions={
          <>
            <button className="btn btn-secondary" onClick={() => void load()}>
              Обновить
            </button>
            <button
              className="btn btn-primary"
              onClick={() => void handleRecalculate()}
              disabled={busy}
            >
              {busy ? "Пересчёт..." : "Пересчитать направления"}
            </button>
          </>
        }
      />
      {error ? <div className="error">{error}</div> : null}
      {loading ? <div className="notice">Загрузка snapshot...</div> : null}
      <div className="card stack">
        <h2>Активная версия snapshot</h2>
        {version ? (
          <dl className="kv">
            <dt>ID</dt>
            <dd>{version.id}</dd>
            <dt>Тип</dt>
            <dd>{version.snapshot_type}</dd>
            <dt>Код версии</dt>
            <dd>{version.version_code}</dd>
            <dt>Контрольная сумма</dt>
            <dd>{version.checksum}</dd>
            <dt>Опубликовано</dt>
            <dd>{version.published_at ?? "—"}</dd>
          </dl>
        ) : (
          <div className="notice">
            Активная версия snapshot ещё не опубликована.
          </div>
        )}
      </div>
      <div className="grid-4">
        <CountCard title="Языки" value={data?.languages.length ?? 0} />
        <CountCard title="Категории" value={data?.categories.length ?? 0} />
        <CountCard
          title="Концепты"
          value={data?.concepts.length ?? 0}
          hint="Смысловые карточки"
        />
        <CountCard
          title="Формы"
          value={data?.forms.length ?? 0}
          hint="Языковые формы"
        />
        <CountCard
          title="Метаданные карточек"
          value={data?.concept_meta.length ?? 0}
        />
        <CountCard
          title="Метаданные форм"
          value={data?.form_meta.length ?? 0}
        />
        <CountCard title="Синонимы" value={data?.form_synonyms.length ?? 0} />
        <CountCard
          title="Направления"
          value={data?.directions.length ?? 0}
          hint="источник → цель"
        />
      </div>
    </div>
  );
}
