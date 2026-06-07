import { useEffect, useState } from "react";
import { fetchSnapshot } from "../shared/api/snapshot";
import { recalculateDirections } from "../shared/api/import";
import { extractApiError } from "../shared/api/client";
import { PageHeader } from "../shared/ui/PageHeader";
import type { SnapshotResponse } from "../types";

function pad(value: number): string {
  return String(value).padStart(2, "0");
}

function formatUtcPlus3(value?: string | null): string {
  if (!value) return "—";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  const shifted = new Date(date.getTime() + 3 * 60 * 60 * 1000);
  const year = shifted.getUTCFullYear();
  const month = pad(shifted.getUTCMonth() + 1);
  const day = pad(shifted.getUTCDate());
  const hours = pad(shifted.getUTCHours());
  const minutes = pad(shifted.getUTCMinutes());
  const seconds = pad(shifted.getUTCSeconds());

  return `${year}:${month}:${day} ${hours}:${minutes}:${seconds}`;
}

function snapshotTypeLabel(value?: string | null): string {
  if (value === "lexicon_directional") return "Направленный словарный контур";
  if (value === "concepts") return "Смысловые карточки";
  if (value === "training") return "Учебные задания";
  return value || "—";
}

function CountCard({
  title,
  value,
}: {
  title: string;
  value: number;
}) {
  return (
    <div className="card stack stat-card">
      <h2>{title}</h2>
      <div className="stat-value">{value}</div>
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
        title="Снимок данных"
        subtitle="Опубликованный набор данных для загрузки мобильным клиентом."
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
      {loading ? <div className="notice">Загрузка снимка данных...</div> : null}
      <div className="card stack">
        <h2>Активная версия</h2>
        {version ? (
          <dl className="kv">
            <dt>ID</dt>
            <dd>{version.id}</dd>
            <dt>Состав данных</dt>
            <dd>{snapshotTypeLabel(version.snapshot_type)}</dd>
            <dt>Код версии</dt>
            <dd>{version.version_code}</dd>
            <dt>Код проверки целостности</dt>
            <dd className="break-word">{version.checksum}</dd>
            <dt>Опубликовано UTC+3</dt>
            <dd>{formatUtcPlus3(version.published_at)}</dd>
          </dl>
        ) : (
          <div className="notice">
            Активная версия ещё не опубликована.
          </div>
        )}
      </div>
      <div className="grid-4">
        <CountCard title="Языки" value={data?.languages.length ?? 0} />
        <CountCard title="Категории" value={data?.categories.length ?? 0} />
        <CountCard title="Карточки" value={data?.concepts.length ?? 0} />
        <CountCard title="Формы" value={data?.forms.length ?? 0} />
        <CountCard
          title="Метаданные карточек"
          value={data?.concept_meta.length ?? 0}
        />
        <CountCard
          title="Метаданные форм"
          value={data?.form_meta.length ?? 0}
        />
        <CountCard title="Синонимы" value={data?.form_synonyms.length ?? 0} />
        <CountCard title="Направления" value={data?.directions.length ?? 0} />
      </div>
    </div>
  );
}
