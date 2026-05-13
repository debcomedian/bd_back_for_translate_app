import { useEffect, useMemo, useState } from 'react';
import { fetchBankQualityReport } from '../shared/api/bankQuality';
import { extractApiError } from '../shared/api/client';
import { PageHeader } from '../shared/ui/PageHeader';
import type { BankQualityReport } from '../types';

function percent(value: unknown) {
  if (typeof value !== 'number') return '—';
  return `${value.toFixed(2)}%`;
}

function count(value: unknown) {
  return typeof value === 'number' ? value : '—';
}

function mapEntries(value: unknown): Array<[string, number]> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return [];
  return Object.entries(value as Record<string, unknown>)
    .map(([key, raw]) => [key, typeof raw === 'number' ? raw : Number(raw)] as [string, number])
    .filter(([, raw]) => Number.isFinite(raw))
    .sort((a, b) => b[1] - a[1]);
}

export function BankQualityPage() {
  const [report, setReport] = useState<BankQualityReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      setReport(await fetchBankQualityReport());
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const statusEntries = useMemo(() => mapEntries(report?.status_counts), [report]);
  const bucketEntries = useMemo(() => mapEntries(report?.bucket_counts), [report]);
  const gateErrors = Array.isArray(report?.gate_errors) ? report?.gate_errors : [];
  const gateWarnings = Array.isArray(report?.gate_warnings) ? report?.gate_warnings : [];

  return (
    <div className="stack">
      <PageHeader
        title="Bank quality"
        subtitle="Аудит active_bank: статусы, частотные buckets, языковое покрытие и quality gate."
        actions={<button className="btn btn-secondary" onClick={() => void load()}>Обновить</button>}
      />
      {error ? <div className="error">{error}</div> : null}
      {loading ? <div className="notice">Загрузка отчёта...</div> : null}
      {report ? (
        <>
          <div className="grid-4">
            <div className="card stack stat-card"><h2>Gate</h2><div><span className={`badge ${report.gate_passed ? 'badge-ok' : 'badge-danger'}`}>{report.gate_passed ? 'passed' : 'failed'}</span></div></div>
            <div className="card stack stat-card"><h2>Total rows</h2><div className="stat-value">{count(report.total_rows)}</div></div>
            <div className="card stack stat-card"><h2>Full trilingual</h2><div className="stat-value">{percent(report.full_trilingual_percent)}</div><div className="muted">{count(report.full_trilingual_count)} строк</div></div>
            <div className="card stack stat-card"><h2>Half validated</h2><div className="stat-value">{percent(report.half_validated_percent)}</div></div>
            <div className="card stack stat-card"><h2>Tail</h2><div className="stat-value">{percent(report.tail_percent)}</div></div>
            <div className="card stack stat-card"><h2>Unsafe</h2><div className="stat-value">{count(report.unsafe_candidates_count)}</div></div>
            <div className="card stack stat-card"><h2>Wide synonyms</h2><div className="stat-value">{count(report.wide_synonym_candidates_count)}</div></div>
            <div className="card stack stat-card"><h2>Input</h2><div className="muted break-word">{report.input_path ?? '—'}</div></div>
          </div>

          {gateErrors.length ? (
            <div className="error stack">
              <strong>Gate errors</strong>
              {gateErrors.map((item, index) => <div key={index}>{item}</div>)}
            </div>
          ) : null}

          {gateWarnings.length ? (
            <div className="notice stack">
              <strong>Gate warnings</strong>
              {gateWarnings.map((item, index) => <div key={index}>{item}</div>)}
            </div>
          ) : null}

          <div className="grid-2">
            <div className="card stack">
              <h2>Status counts</h2>
              <table className="table compact">
                <tbody>
                  {statusEntries.map(([key, value]) => <tr key={key}><td>{key}</td><td>{value}</td></tr>)}
                </tbody>
              </table>
            </div>
            <div className="card stack">
              <h2>Bucket counts</h2>
              <table className="table compact">
                <tbody>
                  {bucketEntries.map(([key, value]) => <tr key={key}><td>{key}</td><td>{value}</td></tr>)}
                </tbody>
              </table>
            </div>
          </div>

          <div className="card stack">
            <h2>Raw report</h2>
            <pre className="pre">{JSON.stringify(report, null, 2)}</pre>
          </div>
        </>
      ) : null}
    </div>
  );
}
