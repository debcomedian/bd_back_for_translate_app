import { useEffect, useMemo, useState } from "react";
import { fetchBankQualityReport } from "../shared/api/bankQuality";
import { extractApiError } from "../shared/api/client";
import { FixedToast, type AppToast, type ToastKind } from "../shared/ui/FixedToast";
import { PageHeader } from "../shared/ui/PageHeader";
import type { BankQualityReport } from "../types";

type RiskRow = {
  line?: number;
  lemma?: string;
  pos?: string;
  status?: string;
  bucket?: string;
  risk_score?: number;
  risk_labels?: string[];
  ru?: string;
  de?: string;
};

type IssueSample = {
  line?: number;
  lemma?: string;
  pos?: string;
  status?: string;
  bucket?: string;
  lang_code?: string;
  value?: string;
  reason?: string;
};

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

function nested(record: unknown, ...keys: string[]): unknown {
  let cur: unknown = record;
  for (const key of keys) {
    const obj = asRecord(cur);
    if (!obj) return undefined;
    cur = obj[key];
  }
  return cur;
}

function numberValue(...values: unknown[]): number | undefined {
  for (const value of values) {
    if (typeof value === "number" && Number.isFinite(value)) return value;
    if (typeof value === "string" && value.trim() !== "") {
      const parsed = Number(value.replace(",", "."));
      if (Number.isFinite(parsed)) return parsed;
    }
  }
  return undefined;
}

function boolValue(...values: unknown[]): boolean | undefined {
  for (const value of values) {
    if (typeof value === "boolean") return value;
    if (value === "true" || value === "1") return true;
    if (value === "false" || value === "0") return false;
  }
  return undefined;
}

function stringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.map((item) => String(item)).filter(Boolean);
}

function percent(value: unknown) {
  const n = numberValue(value);
  if (n === undefined) return "—";
  return `${n.toFixed(2)}%`;
}

function count(value: unknown) {
  const n = numberValue(value);
  return n === undefined ? "—" : String(Math.round(n));
}

function mapEntries(value: unknown): Array<[string, number]> {
  const obj = asRecord(value);
  if (!obj) return [];
  return Object.entries(obj)
    .map(
      ([key, raw]) =>
        [key, numberValue(raw) ?? Number.NaN] as [string, number],
    )
    .filter(([, raw]) => Number.isFinite(raw))
    .sort((a, b) => b[1] - a[1]);
}

function getCountByKeyIncludes(value: unknown, ...parts: string[]): number | undefined {
  const obj = asRecord(value);
  if (!obj) return undefined;

  const lowered = parts.map((part) => part.toLowerCase());
  let sum = 0;
  let found = false;

  for (const [key, raw] of Object.entries(obj)) {
    const normalizedKey = key.toLowerCase();
    if (lowered.every((part) => normalizedKey.includes(part))) {
      const numeric = numberValue(raw);
      if (numeric !== undefined) {
        sum += numeric;
        found = true;
      }
    }
  }

  return found ? sum : undefined;
}

function humanIssueName(value: string) {
  const lower = value.toLowerCase();

  if (value === "unsafe_candidate" || lower.includes("небезопас") || lower.includes("обсцен")) {
    return "Небезопасная или обсценная лексика";
  }
  if (value === "weak_frequency_bucket" || lower.includes("хвостовой") || lower.includes("неизвестной частот")) {
    return "Хвостовая или неизвестная частотная группа";
  }
  if (value === "too_many_candidates_ru") {
    return "Слишком много русских вариантов";
  }
  if (value === "too_many_candidates_de") {
    return "Слишком много немецких вариантов";
  }
  if (lower.includes("слишком много кандидатов")) {
    return "Слишком много вариантов для одного языка";
  }
  if (value === "accent_variant_in_candidates" || lower.includes("ударени")) {
    return "Дубли с ударением, требуется нормализация";
  }
  if (value === "suspicious_short_lemma" || lower.includes("лемма отсутствует") || lower.includes("одного символа")) {
    return "Некорректная короткая английская лемма";
  }

  return value;
}

function isCountLikeIssue(reason: string) {
  const humanName = humanIssueName(reason).toLowerCase();
  return humanName.includes("слишком много") || humanName.includes("количество");
}

function renderCountTable(entries: Array<[string, number]>, emptyText: string) {
  if (!entries.length) return <div className="muted">{emptyText}</div>;
  return (
    <table className="table compact">
      <tbody>
        {entries.map(([key, value]) => (
          <tr key={key}>
            <td>{humanIssueName(key)}</td>
            <td>{value}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function normalizeRiskLabel(label: string) {
  return humanIssueName(label);
}

function getIssueSamples(report: BankQualityReport | null): Array<[string, IssueSample[]]> {
  const raw = asRecord(report?.issue_samples);
  if (!raw) return [];

  return Object.entries(raw)
    .map(([reason, value]) => {
      const samples = Array.isArray(value) ? (value as IssueSample[]) : [];
      return [reason, samples] as [string, IssueSample[]];
    })
    .filter(([, samples]) => samples.length > 0);
}

function issueGroupCount(report: BankQualityReport | null, reason: string) {
  const direct = numberValue(nested(report, "issue_counts", reason));
  if (direct !== undefined) return direct;

  const codeDirect = numberValue(nested(report, "issue_code_counts", reason));
  if (codeDirect !== undefined) return codeDirect;

  return undefined;
}

export function BankQualityPage() {
  const [report, setReport] = useState<BankQualityReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toast, setToast] = useState<AppToast | null>(null);
  const [riskRowsLimit, setRiskRowsLimit] = useState(12);
  const [issueGroupLimit, setIssueGroupLimit] = useState(4);

  function showToast(kind: ToastKind, message: string) {
    const id = Date.now();
    setToast({ id, kind, message });
    window.setTimeout(() => {
      setToast((current) => (current?.id === id ? null : current));
    }, 3000);
  }

  async function load(showSuccess = false) {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchBankQualityReport();
      setReport(data);
      setRiskRowsLimit(12);
      setIssueGroupLimit(4);
      const passed = boolValue(
        data.gate_passed,
        nested(data, "summary", "quality_gate_passed"),
      );
      if (showSuccess) {
        if (passed) {
          showToast("success", "Отчёт качества обновлён: проверка пройдена.");
        } else {
          showToast("warning", "Отчёт качества обновлён: есть замечания по банку.");
        }
      }
    } catch (e) {
      const message = extractApiError(e);
      setError(message);
      showToast("error", message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load(false);
  }, []);

  const statusEntries = useMemo(
    () => mapEntries(report?.status_counts),
    [report],
  );
  const bucketEntries = useMemo(
    () => mapEntries(report?.bucket_counts),
    [report],
  );
  const issueEntries = useMemo(
    () =>
      mapEntries(report?.issue_code_counts).length
        ? mapEntries(report?.issue_code_counts)
        : mapEntries(report?.issue_counts),
    [report],
  );
  const directionEntries = useMemo(
    () => mapEntries(report?.direction_counts),
    [report],
  );
  const issueSamples = useMemo(() => getIssueSamples(report), [report]);
  const visibleIssueSamples = useMemo(
    () =>
      issueSamples
        .slice()
        .sort(
          ([reasonA, samplesA], [reasonB, samplesB]) =>
            (issueGroupCount(report, reasonB) ?? samplesB.length) -
            (issueGroupCount(report, reasonA) ?? samplesA.length),
        )
        .slice(0, issueGroupLimit),
    [issueSamples, issueGroupLimit, report],
  );

  const gatePassed = boolValue(
    report?.gate_passed,
    nested(report, "summary", "quality_gate_passed"),
  );
  const qualityGateReason =
    String(
      report?.quality_gate_reason ??
        nested(report, "summary", "quality_gate_reason") ??
        "",
    ).trim() || "—";

  const totalRows = numberValue(report?.total_rows);
  const fullTrilingualCount = numberValue(
    report?.full_trilingual_count,
    nested(report, "target_coverage", "full_trilingual"),
  );
  const fullTrilingualPercent = numberValue(
    report?.full_trilingual_percent,
    nested(report, "summary", "full_trilingual_percent"),
  );
  const halfValidatedPercent = numberValue(
    report?.half_validated_percent,
    nested(report, "summary", "half_validated_percent"),
  );
  const tailPercent = numberValue(
    report?.tail_percent,
    nested(report, "summary", "tail_percent"),
  );
  const rowsWithIssuesPercent = numberValue(
    report?.rows_with_issues_percent,
    nested(report, "summary", "rows_with_issues_percent"),
  );
  const unsafeCandidates = numberValue(
    report?.unsafe_candidates_count,
    nested(report, "issue_code_counts", "unsafe_candidate"),
  ) ?? getCountByKeyIncludes(report?.issue_counts, "небезопас");
  const calculatedWideSynonyms =
    (numberValue(nested(report, "issue_code_counts", "too_many_candidates_ru")) ?? 0) +
    (numberValue(nested(report, "issue_code_counts", "too_many_candidates_de")) ?? 0);
  const wideSynonymCandidates =
    numberValue(report?.wide_synonym_candidates_count) ??
    (calculatedWideSynonyms > 0 ? calculatedWideSynonyms : undefined) ??
    getCountByKeyIncludes(report?.issue_counts, "слишком много кандидатов");
  const weakFrequencyBucket = numberValue(
    report?.weak_frequency_bucket_count,
    nested(report, "issue_code_counts", "weak_frequency_bucket"),
  ) ?? getCountByKeyIncludes(report?.issue_counts, "хвостовой");
  const gateErrors = stringArray(report?.gate_errors);
  const gateWarnings = stringArray(report?.gate_warnings).length
    ? stringArray(report?.gate_warnings)
    : stringArray(report?.recommendations);
  const topRiskRows = Array.isArray(report?.top_risk_rows)
    ? (report?.top_risk_rows as RiskRow[])
    : [];
  const visibleRiskRows = topRiskRows.slice(0, riskRowsLimit);
  const firstRows = Array.isArray(report?.first_rows)
    ? (report?.first_rows as RiskRow[])
    : [];

  return (
    <div className="stack">
      <FixedToast toast={toast} />
      <PageHeader
        title="Качество банка"
        subtitle="Аудит active_bank: статусы, частотные группы, языковое покрытие и проверка качества."
        actions={
          <button className="btn btn-secondary" onClick={() => void load(true)}>
            Обновить
          </button>
        }
      />
      {error ? <div className="error">{error}</div> : null}
      {loading ? <div className="notice">Загрузка отчёта...</div> : null}
      {report ? (
        <>
          <div className="grid-4">
            <div className="card stack stat-card">
              <h2>Проверка качества</h2>
              <div>
                <span
                  className={`badge ${gatePassed ? "badge-ok" : "badge-danger"}`}
                >
                  {gatePassed ? "пройдена" : "не пройдена"}
                </span>
              </div>
              <div className="muted break-word">{qualityGateReason}</div>
            </div>
            <div className="card stack stat-card">
              <h2>Всего строк</h2>
              <div className="stat-value">{count(totalRows)}</div>
            </div>
            <div className="card stack stat-card">
              <h2>Полное покрытие трёх языков</h2>
              <div className="stat-value">{percent(fullTrilingualPercent)}</div>
              <div className="muted">{count(fullTrilingualCount)} строк</div>
            </div>
            <div className="card stack stat-card">
              <h2>Строк с проблемами</h2>
              <div className="stat-value">{percent(rowsWithIssuesPercent)}</div>
            </div>
            <div className="card stack stat-card">
              <h2>Частичная валидация</h2>
              <div className="stat-value">{percent(halfValidatedPercent)}</div>
            </div>
            <div className="card stack stat-card">
              <h2>Хвостовая частотность</h2>
              <div className="stat-value">{percent(tailPercent)}</div>
            </div>
            <div className="card stack stat-card">
              <h2>Небезопасные кандидаты</h2>
              <div className="stat-value">{count(unsafeCandidates)}</div>
            </div>
            <div className="card stack stat-card">
              <h2>Широкие синонимы</h2>
              <div className="stat-value">{count(wideSynonymCandidates)}</div>
            </div>
            <div className="card stack stat-card">
              <h2>Слабая частотность</h2>
              <div className="stat-value">{count(weakFrequencyBucket)}</div>
            </div>
            <div className="card stack stat-card">
              <h2>Входной файл</h2>
              <div className="muted break-word">{report.input_path ?? "—"}</div>
            </div>
          </div>

          {gateErrors.length ? (
            <div className="error stack">
              <strong>Ошибки проверки качества</strong>
              {gateErrors.map((item, index) => (
                <div key={index}>{item}</div>
              ))}
            </div>
          ) : null}

          {gateWarnings.length ? (
            <div className="notice stack">
              <strong>Предупреждения и рекомендации</strong>
              {gateWarnings.map((item, index) => (
                <div key={index}>{item}</div>
              ))}
            </div>
          ) : null}

          <div className="grid-2">
            <div className="card stack">
              <h2>Статусы записей</h2>
              {renderCountTable(statusEntries, "Статусы не найдены")}
            </div>
            <div className="card stack">
              <h2>Частотные группы</h2>
              {renderCountTable(bucketEntries, "Частотные группы не найдены")}
            </div>
          </div>

          <div className="grid-2">
            <div className="card stack">
              <h2>Направления, которые можно сформировать</h2>
              {renderCountTable(directionEntries, "Направления не рассчитаны")}
            </div>
            <div className="card stack">
              <h2>Найденные проблемы</h2>
              {renderCountTable(issueEntries, "Проблем не найдено")}
            </div>
          </div>

          {issueSamples.length ? (
            <div className="card stack">
              <h2>Примеры найденных проблем</h2>
              <div className="issue-sample-grid">
                {visibleIssueSamples.map(([reason, samples]) => (
                  <details className="issue-sample-card" key={reason}>
                    <summary>
                      <span>{humanIssueName(reason)}</span>
                      <span className="badge badge-warn">
                        {count(issueGroupCount(report, reason) ?? samples.length)}
                      </span>
                    </summary>
                    <div className="table-scroll">
                      <table className="table compact">
                        <thead>
                          <tr>
                            <th>Строка</th>
                            <th>EN</th>
                            <th>Часть речи</th>
                            <th>Статус</th>
                            <th>Группа</th>
                            <th>Язык</th>
                            <th>{isCountLikeIssue(reason) ? "Количество" : "Деталь"}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {samples.slice(0, 8).map((sample, index) => (
                            <tr key={`${reason}-${sample.line ?? index}-${sample.value ?? ""}`}>
                              <td>{sample.line ?? "—"}</td>
                              <td>{sample.lemma ?? "—"}</td>
                              <td>{sample.pos ?? "—"}</td>
                              <td>{sample.status ?? "—"}</td>
                              <td>{sample.bucket ?? "—"}</td>
                              <td>{sample.lang_code ?? "—"}</td>
                              <td>{sample.value ?? "—"}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    {samples.length > 8 ? (
                      <div className="muted">Показаны первые 8 примеров из {samples.length}.</div>
                    ) : null}
                  </details>
                ))}
              </div>
              {issueSamples.length > issueGroupLimit ? (
                <div className="table-actions">
                  <button
                    className="btn btn-secondary btn-small"
                    onClick={() => setIssueGroupLimit((current) => current + 4)}
                  >
                    Показать ещё проблемы
                  </button>
                  <span className="muted">
                    Показано {Math.min(issueGroupLimit, issueSamples.length)} из {issueSamples.length}
                  </span>
                </div>
              ) : issueSamples.length > 4 ? (
                <div className="table-actions">
                  <button
                    className="btn btn-secondary btn-small"
                    onClick={() => setIssueGroupLimit(4)}
                  >
                    Свернуть список проблем
                  </button>
                  <span className="muted">Показаны все группы проблем</span>
                </div>
              ) : null}
            </div>
          ) : null}

          {topRiskRows.length ? (
            <div className="card stack">
              <h2>Строки с наибольшим риском</h2>
              <div className="table-scroll">
                <table className="table compact">
                  <thead>
                    <tr>
                      <th>Строка</th>
                      <th>EN</th>
                      <th>Часть речи</th>
                      <th>Статус</th>
                      <th>Группа</th>
                      <th>Риск</th>
                      <th>Метки</th>
                      <th>RU</th>
                      <th>DE</th>
                    </tr>
                  </thead>
                  <tbody>
                    {visibleRiskRows.map((row, index) => (
                      <tr key={`${row.line ?? index}-${row.lemma ?? ""}`}>
                        <td>{row.line ?? "—"}</td>
                        <td>{row.lemma ?? "—"}</td>
                        <td>{row.pos ?? "—"}</td>
                        <td>{row.status ?? "—"}</td>
                        <td>{row.bucket ?? "—"}</td>
                        <td>{row.risk_score ?? "—"}</td>
                        <td>
                          <div className="risk-labels">
                            {(row.risk_labels ?? []).map((label) => (
                              <span className="risk-chip" key={label}>
                                {normalizeRiskLabel(label)}
                              </span>
                            ))}
                          </div>
                        </td>
                        <td>{row.ru ?? "—"}</td>
                        <td>{row.de ?? "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {topRiskRows.length > riskRowsLimit ? (
                <div className="table-actions">
                  <button
                    className="btn btn-secondary btn-small"
                    onClick={() => setRiskRowsLimit((current) => current + 12)}
                  >
                    Показать ещё строки риска
                  </button>
                  <span className="muted">
                    Показано {Math.min(riskRowsLimit, topRiskRows.length)} из {topRiskRows.length}
                  </span>
                </div>
              ) : topRiskRows.length > 12 ? (
                <div className="table-actions">
                  <button
                    className="btn btn-secondary btn-small"
                    onClick={() => setRiskRowsLimit(12)}
                  >
                    Свернуть до 12 строк
                  </button>
                  <span className="muted">Показаны все строки риска</span>
                </div>
              ) : null}
            </div>
          ) : null}

          {firstRows.length ? (
            <details className="card stack details-panel">
              <summary>Первые строки файла</summary>
              <div className="table-scroll">
                <table className="table compact">
                  <thead>
                    <tr>
                      <th>Строка</th>
                      <th>EN</th>
                      <th>Часть речи</th>
                      <th>Статус</th>
                      <th>Группа</th>
                      <th>RU</th>
                      <th>DE</th>
                    </tr>
                  </thead>
                  <tbody>
                    {firstRows.map((row, index) => (
                      <tr key={`${row.line ?? index}-${row.lemma ?? ""}`}>
                        <td>{row.line ?? "—"}</td>
                        <td>{row.lemma ?? "—"}</td>
                        <td>{row.pos ?? "—"}</td>
                        <td>{row.status ?? "—"}</td>
                        <td>{row.bucket ?? "—"}</td>
                        <td>{row.ru ?? "—"}</td>
                        <td>{row.de ?? "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </details>
          ) : null}
        </>
      ) : null}
    </div>
  );
}
