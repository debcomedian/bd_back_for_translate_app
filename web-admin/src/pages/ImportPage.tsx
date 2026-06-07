import { useEffect, useMemo, useState } from "react";
import {
  commitActiveBankImport,
  previewActiveBankImport,
  recalculateDirections,
} from "../shared/api/import";
import { extractApiError } from "../shared/api/client";
import { PageHeader } from "../shared/ui/PageHeader";
import {
  FixedToast,
  type AppToast,
  type ToastKind,
} from "../shared/ui/FixedToast";
import type {
  ActiveBankImportCommitResponse,
  ActiveBankImportItemStatus,
  ActiveBankImportPreviewItem,
  ActiveBankImportPreviewResponse,
  ActiveBankImportResolution,
  ActiveBankImportResolutionAction,
  ActiveBankRecord,
  ActiveBankTargetState,
  RecalculateMetaResponse,
} from "../types";

const MAX_IMPORT_FILE_SIZE = 10 * 1024 * 1024;
const IMPORT_TABS: Array<{ value: ActiveBankImportItemStatus; label: string }> =
  [
    { value: "new", label: "Новые" },
    { value: "duplicate", label: "Повторы" },
    { value: "collision", label: "Коллизии" },
    { value: "invalid", label: "Ошибки" },
  ];

const RESOLUTION_LABELS: Record<ActiveBankImportResolutionAction, string> = {
  keep_existing: "Оставить текущую запись",
  use_incoming: "Принять новую запись",
  merge_edit: "Редактировать поля вручную",
  skip: "Пропустить строку",
};

const POS_OPTIONS = ["noun", "verb", "adj", "adv"];
const STATUS_OPTIONS = [
  "strict_validated_all",
  "soft_validated_all",
  "soft_completed_all",
  "half_validated",
];
const BUCKET_OPTIONS = ["core_high", "core_mid", "core_low", "tail"];

function collectTargetValues(state: ActiveBankTargetState | undefined) {
  if (!state) return [];
  const values = [
    ...(state.strict_validated ?? []),
    ...(state.soft_validated ?? []),
    ...(state.completed ?? []),
    ...(state.from_english ?? []),
    ...(state.candidates ?? []),
    ...(state.synonyms ?? []),
  ];
  const seen = new Set<string>();
  return values.filter((value) => {
    const key = value.trim().toLowerCase();
    if (!key || seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function primaryTarget(
  record: ActiveBankRecord | undefined,
  lang: "ru" | "de",
) {
  if (!record) return "—";
  const state = record.targets?.[lang];
  const groups = [
    state?.strict_validated,
    state?.soft_validated,
    state?.completed,
    state?.from_english,
    state?.candidates,
  ];
  for (const group of groups) {
    const value = group?.find((item) => item.trim());
    if (value) return value;
  }
  return "—";
}

function formatList(values: string[] | undefined) {
  if (!values || values.length === 0) return "—";
  return values.join(", ");
}

function parseListInput(value: string) {
  return value
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function listToText(values: string[] | undefined) {
  return (values ?? []).join("\n");
}

function cloneRecord(record: ActiveBankRecord | undefined): ActiveBankRecord {
  if (!record) {
    return {
      en_lemma: "",
      pos: "noun",
      status: "strict_validated_all",
      layer: "active_core",
      confidence: "high",
      flags: [],
      glosses: [""],
      targets: {
        ru: { strict_validated: [""], candidates: [], synonyms: [] },
        de: { strict_validated: [""], candidates: [], synonyms: [] },
      },
      frequency: {
        en: { zipf: 0, bucket: "core_mid", priority_score: 0 },
        priority_score: 0,
      },
    };
  }

  return JSON.parse(JSON.stringify(record)) as ActiveBankRecord;
}

function ensureTarget(record: ActiveBankRecord, lang: "ru" | "de") {
  return {
    strict_validated: [],
    soft_validated: [],
    completed: [],
    from_english: [],
    candidates: [],
    synonyms: [],
    ...(record.targets?.[lang] ?? {}),
  };
}

function replacePrimaryTarget(
  record: ActiveBankRecord,
  lang: "ru" | "de",
  value: string,
) {
  const next = cloneRecord(record);
  next.targets = { ...(next.targets ?? {}) };
  const target = ensureTarget(next, lang);
  const current = [...(target.strict_validated ?? [])];
  if (value.trim()) {
    current[0] = value.trim();
  } else {
    current.splice(0, 1);
  }
  target.strict_validated = current.filter(Boolean);
  next.targets[lang] = target;
  return next;
}

function replaceTargetList(
  record: ActiveBankRecord,
  lang: "ru" | "de",
  field: keyof ActiveBankTargetState,
  values: string[],
) {
  const next = cloneRecord(record);
  next.targets = { ...(next.targets ?? {}) };
  const target = ensureTarget(next, lang);
  target[field] = values;
  next.targets[lang] = target;
  return next;
}

function validateActiveBankRecordForCommit(
  record: ActiveBankRecord | undefined,
) {
  const errors: string[] = [];
  if (!record) {
    return ["Итоговая запись отсутствует."];
  }

  if (!record.en_lemma?.trim()) {
    errors.push("Английская лемма обязательна.");
  }

  if (!POS_OPTIONS.includes(record.pos)) {
    errors.push("Часть речи должна быть noun, verb, adj или adv.");
  }

  if (!STATUS_OPTIONS.includes(record.status)) {
    errors.push("Статус проверки содержит недопустимое значение.");
  }

  if (primaryTarget(record, "ru") === "—") {
    errors.push("Русская форма обязательна.");
  }

  if (primaryTarget(record, "de") === "—") {
    errors.push("Немецкая форма обязательна.");
  }

  const zipf = Number(record.frequency?.en?.zipf);
  if (!Number.isFinite(zipf) || zipf < 0) {
    errors.push("Частотность Zipf должна быть числом не меньше 0.");
  }

  if (!BUCKET_OPTIONS.includes(record.frequency?.en?.bucket ?? "")) {
    errors.push(
      "Частотная группа должна быть core_high, core_mid, core_low или tail.",
    );
  }

  return errors;
}

async function validateFileClientSide(file: File | null): Promise<string[]> {
  const errors: string[] = [];
  if (!file) return errors;

  if (!file.name.endsWith(".jsonl")) {
    errors.push("Файл должен иметь расширение .jsonl");
  }
  if (file.size === 0) {
    errors.push("Файл пустой");
  }
  if (file.size > MAX_IMPORT_FILE_SIZE) {
    errors.push("Файл больше 10 МБ");
  }
  if (errors.length > 0) return errors;

  const text = await file.text();
  const lines = text.split(/\r?\n/).filter((line) => line.trim() !== "");
  lines.slice(0, 5000).forEach((line, index) => {
    try {
      const parsed = JSON.parse(line) as Record<string, unknown>;
      if (!parsed.en_lemma)
        errors.push(`Строка ${index + 1}: отсутствует en_lemma`);
      if (!parsed.pos) errors.push(`Строка ${index + 1}: отсутствует pos`);
      if (!parsed.targets || typeof parsed.targets !== "object") {
        errors.push(`Строка ${index + 1}: отсутствует targets`);
      }
      if (!parsed.frequency || typeof parsed.frequency !== "object") {
        errors.push(`Строка ${index + 1}: отсутствует frequency`);
      }
    } catch {
      errors.push(`Строка ${index + 1}: невалидный JSON`);
    }
  });

  if (lines.length > 5000) {
    errors.push(
      "Клиент проверил первые 5000 строк. Полная проверка будет выполнена на сервере.",
    );
  }

  return errors;
}

function filterItems(
  preview: ActiveBankImportPreviewResponse | null,
  status: ActiveBankImportItemStatus,
) {
  return preview?.items.filter((item) => item.status === status) ?? [];
}

function renderRecordDetails(
  record: ActiveBankRecord | undefined,
  options?: { row?: number; existingConceptId?: number },
) {
  if (!record) {
    return <div className="notice">Запись отсутствует.</div>;
  }

  const ruValues = collectTargetValues(record.targets?.ru);
  const deValues = collectTargetValues(record.targets?.de);

  return (
    <dl className="record-details-grid">
      {options?.row ? (
        <>
          <dt>Строка файла</dt>
          <dd>{options.row}</dd>
        </>
      ) : null}
      {options?.existingConceptId ? (
        <>
          <dt>ID смыслового объекта</dt>
          <dd>{options.existingConceptId}</dd>
        </>
      ) : null}
      <dt>Английская лемма</dt>
      <dd>{record.en_lemma || "—"}</dd>
      <dt>Часть речи</dt>
      <dd>{record.pos || "—"}</dd>
      <dt>Русская форма</dt>
      <dd>{primaryTarget(record, "ru")}</dd>
      <dt>Немецкая форма</dt>
      <dd>{primaryTarget(record, "de")}</dd>
      <dt>Статус проверки</dt>
      <dd>{record.status || "—"}</dd>
      <dt>Слой банка</dt>
      <dd>{record.layer || "—"}</dd>
      <dt>Уверенность</dt>
      <dd>{record.confidence || "—"}</dd>
      <dt>Частотность Zipf</dt>
      <dd>{record.frequency?.en?.zipf ?? "—"}</dd>
      <dt>Частотная группа</dt>
      <dd>{record.frequency?.en?.bucket || "—"}</dd>
      <dt>Приоритет</dt>
      <dd>
        {record.frequency?.priority_score ??
          record.frequency?.en?.priority_score ??
          "—"}
      </dd>
      <dt>Описание</dt>
      <dd>{record.glosses?.[0] || "—"}</dd>
      <dt>Все русские ответы</dt>
      <dd>{formatList(ruValues)}</dd>
      <dt>Все немецкие ответы</dt>
      <dd>{formatList(deValues)}</dd>
    </dl>
  );
}

export function ImportPage() {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [preview, setPreview] =
    useState<ActiveBankImportPreviewResponse | null>(null);
  const [commitResult, setCommitResult] =
    useState<ActiveBankImportCommitResponse | null>(null);
  const [metaResult, setMetaResult] = useState<RecalculateMetaResponse | null>(
    null,
  );
  const [activeTab, setActiveTab] = useState<ActiveBankImportItemStatus>("new");
  const [resolutions, setResolutions] = useState<
    Record<number, ActiveBankImportResolution>
  >({});
  const [draftActions, setDraftActions] = useState<
    Record<number, ActiveBankImportResolutionAction | "">
  >({});
  const [mergedRecords, setMergedRecords] = useState<
    Record<number, ActiveBankRecord>
  >({});
  const [collisionIndex, setCollisionIndex] = useState(0);
  const [clientErrors, setClientErrors] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [toast, setToast] = useState<AppToast | null>(null);
  const [loadingPreview, setLoadingPreview] = useState(false);
  const [loadingCommit, setLoadingCommit] = useState(false);
  const [loadingMeta, setLoadingMeta] = useState(false);

  function notify(kind: ToastKind, message: string) {
    setToast({
      id: Date.now(),
      kind,
      message,
    });
  }

  function showError(message: string) {
    setError(message);
    notify("error", message);
  }

  useEffect(() => {
    if (!toast) return undefined;

    const timer = window.setTimeout(() => {
      setToast((current) => (current?.id === toast.id ? null : current));
    }, 3000);

    return () => window.clearTimeout(timer);
  }, [toast]);

  const currentItems = useMemo(
    () => filterItems(preview, activeTab),
    [preview, activeTab],
  );

  const collisionItems = useMemo(
    () => filterItems(preview, "collision"),
    [preview],
  );

  const invalidItems = useMemo(
    () => filterItems(preview, "invalid"),
    [preview],
  );

  const resolvedCollisionCount = useMemo(
    () =>
      collisionItems.filter((item) => Boolean(resolutions[item.row])).length,
    [collisionItems, resolutions],
  );

  const resolvedInvalidCount = useMemo(
    () => invalidItems.filter((item) => Boolean(resolutions[item.row])).length,
    [invalidItems, resolutions],
  );

  const unresolvedCollisionCount = Math.max(
    0,
    collisionItems.length - resolvedCollisionCount,
  );
  const unresolvedInvalidCount = Math.max(
    0,
    invalidItems.length - resolvedInvalidCount,
  );

  async function handlePreview() {
    setError(null);
    setPreview(null);
    setCommitResult(null);
    setClientErrors([]);
    setResolutions({});
    setDraftActions({});
    setMergedRecords({});
    setCollisionIndex(0);
    setLoadingPreview(true);

    try {
      const localErrors = await validateFileClientSide(selectedFile);
      const blockingLocalErrors = localErrors.filter(
        (item) => !item.startsWith("Клиент проверил"),
      );
      setClientErrors(localErrors);
      if (blockingLocalErrors.length > 0) {
        notify(
          "error",
          `Файл не прошёл клиентскую проверку: ${blockingLocalErrors[0]}`,
        );
        return;
      }
      if (localErrors.length > 0) {
        notify("warning", localErrors[0]);
      }

      const result = await previewActiveBankImport(selectedFile ?? undefined);
      setPreview(result);
      setActiveTab(
        result.stats.invalid > 0
          ? "invalid"
          : result.stats.collisions > 0
            ? "collision"
            : "new",
      );

      if (result.stats.invalid > 0) {
        notify(
          "error",
          `Файл проверен. Ошибок формата: ${result.stats.invalid}. Исправьте или исключите строки.`,
        );
      } else if (result.stats.collisions > 0) {
        notify(
          "warning",
          `Файл проверен. Коллизий: ${result.stats.collisions}. Разберите их последовательно.`,
        );
      } else if (result.stats.duplicates > 0) {
        notify(
          "warning",
          `Файл проверен. Новых: ${result.stats.new}, повторов: ${result.stats.duplicates}.`,
        );
      } else {
        notify("success", `Файл проверен. Новых записей: ${result.stats.new}.`);
      }

      const initialMergedRecords: Record<number, ActiveBankRecord> = {};
      result.items.forEach((item) => {
        if (item.status === "collision" || item.status === "invalid") {
          initialMergedRecords[item.row] = cloneRecord(item.incoming);
        }
      });
      setMergedRecords(initialMergedRecords);
    } catch (e) {
      showError(extractApiError(e));
    } finally {
      setLoadingPreview(false);
    }
  }

  async function handleCommit() {
    if (!preview) return;

    setError(null);
    setCommitResult(null);
    setLoadingCommit(true);

    try {
      if (unresolvedInvalidCount > 0) {
        throw new Error(
          `Осталось неразобранных ошибок формата: ${unresolvedInvalidCount}`,
        );
      }
      if (unresolvedCollisionCount > 0) {
        throw new Error(
          `Осталось неразрешённых коллизий: ${unresolvedCollisionCount}`,
        );
      }

      const payload = (
        Object.values(resolutions) as ActiveBankImportResolution[]
      ).sort((a, b) => a.row - b.row);
      for (const resolution of payload) {
        if (resolution.action === "merge_edit") {
          const validationErrors = validateActiveBankRecordForCommit(
            resolution.merged,
          );
          if (validationErrors.length > 0) {
            throw new Error(
              `Строка ${resolution.row}: ${validationErrors.join(" ")}`,
            );
          }
        }
      }

      const result = await commitActiveBankImport(preview.session_id, payload);
      setCommitResult(result);
      notify(
        "success",
        `Импорт применён. Вставлено: ${result.stats.inserted}, обновлено: ${result.stats.updated}, пропущено: ${result.stats.skipped}.`,
      );
    } catch (e) {
      showError(extractApiError(e));
    } finally {
      setLoadingCommit(false);
    }
  }

  async function handleRecalculate() {
    setError(null);
    setMetaResult(null);
    setLoadingMeta(true);

    try {
      const result = await recalculateDirections();
      setMetaResult(result);
      notify("success", `Пересчёт завершён. Обработано: ${result.processed}.`);
    } catch (e) {
      showError(extractApiError(e));
    } finally {
      setLoadingMeta(false);
    }
  }

  function handleChooseCollisionAction(
    row: number,
    action: ActiveBankImportResolutionAction,
  ) {
    setDraftActions((current) => ({ ...current, [row]: action }));
  }

  function handleStartMergeEdit(item: ActiveBankImportPreviewItem) {
    const currentAction =
      draftActions[item.row] || resolutions[item.row]?.action || "";
    const baseRecord =
      currentAction === "keep_existing" && item.existing
        ? item.existing
        : item.incoming;

    setMergedRecords((current) => ({
      ...current,
      [item.row]: current[item.row] ?? cloneRecord(baseRecord),
    }));
    handleChooseCollisionAction(item.row, "merge_edit");
  }

  function updateMergedRecord(
    row: number,
    updater: (record: ActiveBankRecord) => ActiveBankRecord,
  ) {
    setMergedRecords((current) => ({
      ...current,
      [row]: updater(cloneRecord(current[row])),
    }));
  }

  function applyBulkCollisionDecision(
    action: "keep_existing" | "use_incoming",
  ) {
    const nextResolutions: Record<number, ActiveBankImportResolution> = {};
    const nextDraftActions: Record<number, ActiveBankImportResolutionAction> =
      {};

    collisionItems.forEach((item) => {
      nextResolutions[item.row] = { row: item.row, action };
      nextDraftActions[item.row] = action;
    });

    setResolutions((current) => ({ ...current, ...nextResolutions }));
    setDraftActions((current) => ({ ...current, ...nextDraftActions }));
    setCollisionIndex(0);
    notify(
      "warning",
      action === "keep_existing"
        ? `Для всех коллизий выбрано: оставить текущие записи.`
        : `Для всех коллизий выбрано: принять новые записи.`,
    );
  }

  function clearCollisionDecisions() {
    const collisionRows = new Set(collisionItems.map((item) => item.row));
    setResolutions(
      (current) =>
        Object.fromEntries(
          Object.entries(current).filter(
            ([row]) => !collisionRows.has(Number(row)),
          ),
        ) as Record<number, ActiveBankImportResolution>,
    );
    setDraftActions(
      (current) =>
        Object.fromEntries(
          Object.entries(current).filter(
            ([row]) => !collisionRows.has(Number(row)),
          ),
        ) as Record<number, ActiveBankImportResolutionAction>,
    );
    setCollisionIndex(0);
    notify("warning", "Решения по коллизиям сброшены.");
  }

  function applyBulkInvalidSkip() {
    const nextResolutions: Record<number, ActiveBankImportResolution> = {};
    const nextDraftActions: Record<number, ActiveBankImportResolutionAction> =
      {};

    invalidItems.forEach((item) => {
      nextResolutions[item.row] = { row: item.row, action: "skip" };
      nextDraftActions[item.row] = "skip";
    });

    setResolutions((current) => ({ ...current, ...nextResolutions }));
    setDraftActions((current) => ({ ...current, ...nextDraftActions }));
    notify("warning", "Все строки с ошибками исключены из импорта.");
  }

  function clearInvalidDecisions() {
    const invalidRows = new Set(invalidItems.map((item) => item.row));
    setResolutions(
      (current) =>
        Object.fromEntries(
          Object.entries(current).filter(
            ([row]) => !invalidRows.has(Number(row)),
          ),
        ) as Record<number, ActiveBankImportResolution>,
    );
    setDraftActions(
      (current) =>
        Object.fromEntries(
          Object.entries(current).filter(
            ([row]) => !invalidRows.has(Number(row)),
          ),
        ) as Record<number, ActiveBankImportResolutionAction>,
    );
    notify("warning", "Решения по ошибочным строкам сброшены.");
  }

  function handleApplyInvalidDecision(item: ActiveBankImportPreviewItem) {
    setError(null);

    const draftAction =
      draftActions[item.row] || resolutions[item.row]?.action || "";
    if (!draftAction) {
      showError("Исключите строку из импорта либо отредактируйте поля.");
      return;
    }

    if (draftAction === "skip") {
      setResolutions((current) => ({
        ...current,
        [item.row]: { row: item.row, action: "skip" },
      }));
      notify("warning", `Строка ${item.row} исключена из импорта.`);
      return;
    }

    if (draftAction !== "merge_edit") {
      showError(
        "Для ошибки формата доступно только исключение или ручное редактирование.",
      );
      return;
    }

    const merged = mergedRecords[item.row] ?? cloneRecord(item.incoming);
    const validationErrors = validateActiveBankRecordForCommit(merged);
    if (validationErrors.length > 0) {
      showError(`Строка ${item.row}: ${validationErrors.join(" ")}`);
      return;
    }

    setResolutions((current) => ({
      ...current,
      [item.row]: { row: item.row, action: "merge_edit", merged },
    }));
    notify("success", `Исправление строки ${item.row} применено.`);
  }

  function moveToNextCollision(currentRow: number) {
    const nextAfterCurrent = collisionItems.findIndex(
      (item, index) =>
        index > collisionIndex &&
        item.row !== currentRow &&
        !resolutions[item.row],
    );
    if (nextAfterCurrent >= 0) {
      setCollisionIndex(nextAfterCurrent);
      return;
    }

    const firstUnresolved = collisionItems.findIndex(
      (item) => item.row !== currentRow && !resolutions[item.row],
    );
    if (firstUnresolved >= 0) {
      setCollisionIndex(firstUnresolved);
      return;
    }

    const nextIndex = Math.min(
      collisionIndex + 1,
      Math.max(0, collisionItems.length - 1),
    );
    setCollisionIndex(nextIndex);
  }

  function handleApplyCollisionDecision(item: ActiveBankImportPreviewItem) {
    setError(null);

    const draftAction =
      draftActions[item.row] || resolutions[item.row]?.action || "";
    if (!draftAction) {
      showError(
        "Выберите текущую или новую запись либо откройте ручное редактирование.",
      );
      return;
    }

    let resolution: ActiveBankImportResolution;
    if (draftAction === "merge_edit") {
      const merged = mergedRecords[item.row] ?? cloneRecord(item.incoming);
      const validationErrors = validateActiveBankRecordForCommit(merged);
      if (validationErrors.length > 0) {
        showError(`Строка ${item.row}: ${validationErrors.join(" ")}`);
        return;
      }
      resolution = { row: item.row, action: draftAction, merged };
    } else {
      resolution = { row: item.row, action: draftAction };
    }

    setResolutions((current) => ({ ...current, [item.row]: resolution }));
    notify("success", `Решение по коллизии ${collisionIndex + 1} применено.`);
    moveToNextCollision(item.row);
  }

  function renderPreviewStats(result: ActiveBankImportPreviewResponse) {
    return (
      <div className="grid-4">
        <div className="card stat-card">
          <h2>Всего строк</h2>
          <div className="stat-value">{result.stats.total}</div>
        </div>
        <div className="card stat-card">
          <h2>Новые</h2>
          <div className="stat-value">{result.stats.new}</div>
        </div>
        <div className="card stat-card">
          <h2>Повторы</h2>
          <div className="stat-value">{result.stats.duplicates}</div>
        </div>
        <div className="card stat-card">
          <h2>Коллизии</h2>
          <div className="stat-value">{result.stats.collisions}</div>
        </div>
      </div>
    );
  }

  function renderItem(item: ActiveBankImportPreviewItem) {
    return (
      <div key={item.id} className="import-item-card stack">
        <div className="section-title-row compact-title-row">
          <div>
            <strong>Строка {item.row}</strong>
          </div>
          <span className={`badge import-status-${item.status}`}>
            {item.status}
          </span>
        </div>

        {item.status === "duplicate" ? (
          <>
            <div className="notice">
              Такая запись уже есть в базе. При commit она будет пропущена.
            </div>
            <div className="diff-grid">
              <div className="card stack">
                <h3>Текущая запись</h3>
                {renderRecordDetails(item.existing, {
                  existingConceptId: item.existing_concept_id,
                })}
              </div>
              <div className="card stack">
                <h3>Запись из файла</h3>
                {renderRecordDetails(item.incoming)}
              </div>
            </div>
          </>
        ) : null}

        {item.status === "new" ? renderRecordDetails(item.incoming) : null}
      </div>
    );
  }

  function renderEditableRecordForm(row: number) {
    const record = mergedRecords[row] ?? cloneRecord(undefined);
    const ruTarget = ensureTarget(record, "ru");
    const deTarget = ensureTarget(record, "de");

    return (
      <div className="editable-record-form stack">
        <div className="form-grid-2">
          <label className="field">
            <span className="label">Английская лемма</span>
            <input
              className="input"
              value={record.en_lemma ?? ""}
              onChange={(event) =>
                updateMergedRecord(row, (current) => ({
                  ...current,
                  en_lemma: event.target.value,
                }))
              }
            />
          </label>

          <label className="field">
            <span className="label">Часть речи</span>
            <select
              className="select"
              value={record.pos ?? "noun"}
              onChange={(event) =>
                updateMergedRecord(row, (current) => ({
                  ...current,
                  pos: event.target.value,
                }))
              }
            >
              {POS_OPTIONS.map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
          </label>

          <label className="field">
            <span className="label">Русская форма</span>
            <input
              className="input"
              value={
                primaryTarget(record, "ru") === "—"
                  ? ""
                  : primaryTarget(record, "ru")
              }
              onChange={(event) =>
                updateMergedRecord(row, (current) =>
                  replacePrimaryTarget(current, "ru", event.target.value),
                )
              }
            />
          </label>

          <label className="field">
            <span className="label">Немецкая форма</span>
            <input
              className="input"
              value={
                primaryTarget(record, "de") === "—"
                  ? ""
                  : primaryTarget(record, "de")
              }
              onChange={(event) =>
                updateMergedRecord(row, (current) =>
                  replacePrimaryTarget(current, "de", event.target.value),
                )
              }
            />
          </label>

          <label className="field">
            <span className="label">Статус проверки</span>
            <select
              className="select"
              value={record.status ?? "strict_validated_all"}
              onChange={(event) =>
                updateMergedRecord(row, (current) => ({
                  ...current,
                  status: event.target.value,
                }))
              }
            >
              {STATUS_OPTIONS.map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
          </label>

          <label className="field">
            <span className="label">Слой банка</span>
            <input
              className="input"
              value={record.layer ?? ""}
              onChange={(event) =>
                updateMergedRecord(row, (current) => ({
                  ...current,
                  layer: event.target.value,
                }))
              }
            />
          </label>

          <label className="field">
            <span className="label">Уверенность</span>
            <input
              className="input"
              value={record.confidence ?? ""}
              onChange={(event) =>
                updateMergedRecord(row, (current) => ({
                  ...current,
                  confidence: event.target.value,
                }))
              }
            />
          </label>

          <label className="field">
            <span className="label">Частотная группа</span>
            <select
              className="select"
              value={record.frequency?.en?.bucket ?? "core_mid"}
              onChange={(event) =>
                updateMergedRecord(row, (current) => ({
                  ...current,
                  frequency: {
                    ...(current.frequency ?? {
                      en: { zipf: 0, bucket: "core_mid" },
                    }),
                    en: {
                      ...(current.frequency?.en ?? {
                        zipf: 0,
                        bucket: "core_mid",
                      }),
                      bucket: event.target.value,
                    },
                  },
                }))
              }
            >
              {BUCKET_OPTIONS.map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
          </label>

          <label className="field">
            <span className="label">Частотность Zipf</span>
            <input
              className="input"
              type="number"
              min="0"
              step="0.01"
              value={record.frequency?.en?.zipf ?? 0}
              onChange={(event) =>
                updateMergedRecord(row, (current) => ({
                  ...current,
                  frequency: {
                    ...(current.frequency ?? {
                      en: { zipf: 0, bucket: "core_mid" },
                    }),
                    en: {
                      ...(current.frequency?.en ?? {
                        zipf: 0,
                        bucket: "core_mid",
                      }),
                      zipf: Number(event.target.value),
                    },
                  },
                }))
              }
            />
          </label>

          <label className="field">
            <span className="label">Приоритет</span>
            <input
              className="input"
              type="number"
              min="0"
              step="0.001"
              value={
                record.frequency?.priority_score ??
                record.frequency?.en?.priority_score ??
                0
              }
              onChange={(event) =>
                updateMergedRecord(row, (current) => ({
                  ...current,
                  frequency: {
                    ...(current.frequency ?? {
                      en: { zipf: 0, bucket: "core_mid" },
                    }),
                    priority_score: Number(event.target.value),
                    en: {
                      ...(current.frequency?.en ?? {
                        zipf: 0,
                        bucket: "core_mid",
                      }),
                      priority_score: Number(event.target.value),
                    },
                  },
                }))
              }
            />
          </label>
        </div>

        <label className="field">
          <span className="label">Описание</span>
          <textarea
            className="textarea small-textarea"
            value={record.glosses?.[0] ?? ""}
            onChange={(event) =>
              updateMergedRecord(row, (current) => ({
                ...current,
                glosses: [event.target.value],
              }))
            }
          />
        </label>

        <div className="form-grid-2">
          <label className="field">
            <span className="label">Дополнительные русские ответы</span>
            <textarea
              className="textarea list-textarea"
              value={listToText(ruTarget.synonyms)}
              onChange={(event) =>
                updateMergedRecord(row, (current) =>
                  replaceTargetList(
                    current,
                    "ru",
                    "synonyms",
                    parseListInput(event.target.value),
                  ),
                )
              }
            />
          </label>

          <label className="field">
            <span className="label">Дополнительные немецкие ответы</span>
            <textarea
              className="textarea list-textarea"
              value={listToText(deTarget.synonyms)}
              onChange={(event) =>
                updateMergedRecord(row, (current) =>
                  replaceTargetList(
                    current,
                    "de",
                    "synonyms",
                    parseListInput(event.target.value),
                  ),
                )
              }
            />
          </label>

          <label className="field">
            <span className="label">Русские кандидаты</span>
            <textarea
              className="textarea list-textarea"
              value={listToText(ruTarget.candidates)}
              onChange={(event) =>
                updateMergedRecord(row, (current) =>
                  replaceTargetList(
                    current,
                    "ru",
                    "candidates",
                    parseListInput(event.target.value),
                  ),
                )
              }
            />
          </label>

          <label className="field">
            <span className="label">Немецкие кандидаты</span>
            <textarea
              className="textarea list-textarea"
              value={listToText(deTarget.candidates)}
              onChange={(event) =>
                updateMergedRecord(row, (current) =>
                  replaceTargetList(
                    current,
                    "de",
                    "candidates",
                    parseListInput(event.target.value),
                  ),
                )
              }
            />
          </label>
        </div>

        <div className="notice">
          Перед применением система проверит обязательные поля и только после
          этого соберёт итоговую JSON-запись для отправки на сервер.
        </div>
      </div>
    );
  }

  function renderCollision(item: ActiveBankImportPreviewItem) {
    const draftAction =
      draftActions[item.row] || resolutions[item.row]?.action || "";
    const appliedResolution = resolutions[item.row];

    return (
      <div key={item.id} className="collision-step stack">
        <div className="section-title-row compact-title-row">
          <div>
            <h2>
              Коллизия {collisionIndex + 1} из {collisionItems.length}
            </h2>
          </div>
          <span
            className={`badge ${appliedResolution ? "badge-success" : "import-status-collision"}`}
          >
            {appliedResolution ? "решение применено" : "ожидает решения"}
          </span>
        </div>

        <div className="merge-choice-grid">
          <div
            className={`merge-choice-card ${draftAction === "keep_existing" ? "selected" : ""}`}
            role="button"
            tabIndex={0}
            onClick={() =>
              handleChooseCollisionAction(item.row, "keep_existing")
            }
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ")
                handleChooseCollisionAction(item.row, "keep_existing");
            }}
          >
            <div className="merge-choice-title">
              <h3>Текущая запись</h3>
              <span>Оставить</span>
            </div>
            {renderRecordDetails(item.existing)}
          </div>

          <div
            className={`merge-choice-card ${draftAction === "use_incoming" ? "selected" : ""}`}
            role="button"
            tabIndex={0}
            onClick={() =>
              handleChooseCollisionAction(item.row, "use_incoming")
            }
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ")
                handleChooseCollisionAction(item.row, "use_incoming");
            }}
          >
            <div className="merge-choice-title">
              <h3>Новая запись</h3>
              <span>Принять</span>
            </div>
            {renderRecordDetails(item.incoming)}
          </div>
        </div>

        <div className="decision-panel stack">
          <div className="section-title-row compact-title-row">
            <div>
              <h3>Решение</h3>
              <div className="muted">
                {draftAction
                  ? RESOLUTION_LABELS[draftAction]
                  : "Нажмите на текущую или новую запись либо откройте ручное редактирование."}
              </div>
            </div>
            <div className="actions">
              <button
                type="button"
                className={`btn btn-secondary ${draftAction === "merge_edit" ? "active" : ""}`}
                onClick={() => handleStartMergeEdit(item)}
              >
                ✎ Редактировать
              </button>
              <button
                type="button"
                className="btn btn-primary"
                disabled={!draftAction}
                onClick={() => handleApplyCollisionDecision(item)}
              >
                Применить решение
              </button>
            </div>
          </div>

          {draftAction === "merge_edit"
            ? renderEditableRecordForm(item.row)
            : null}
        </div>
      </div>
    );
  }

  function renderCollisionDeck() {
    if (collisionItems.length === 0) {
      return <div className="notice">Коллизий нет.</div>;
    }

    const safeIndex = Math.min(collisionIndex, collisionItems.length - 1);
    const item = collisionItems[safeIndex];

    return (
      <div className="stack">
        <div className="collision-progress card">
          <div>
            <strong>Разрешение коллизий</strong>
            <div className="muted">
              Решено: {resolvedCollisionCount} из {collisionItems.length}. Не
              решено: {unresolvedCollisionCount}.
            </div>
          </div>
          <div className="actions">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={() => applyBulkCollisionDecision("keep_existing")}
            >
              Оставить все текущие
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              onClick={() => applyBulkCollisionDecision("use_incoming")}
            >
              Принять все новые
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              disabled={resolvedCollisionCount === 0}
              onClick={clearCollisionDecisions}
            >
              Сбросить решения
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              disabled={safeIndex === 0}
              onClick={() =>
                setCollisionIndex((value) => Math.max(0, value - 1))
              }
            >
              Назад
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              disabled={safeIndex >= collisionItems.length - 1}
              onClick={() =>
                setCollisionIndex((value) =>
                  Math.min(collisionItems.length - 1, value + 1),
                )
              }
            >
              Далее
            </button>
          </div>
        </div>
        {renderCollision(item)}
      </div>
    );
  }

  function renderInvalidDeck() {
    if (invalidItems.length === 0) {
      return <div className="notice">Ошибок формата нет.</div>;
    }

    return (
      <div className="stack">
        <div className="invalid-progress card">
          <div>
            <strong>Ошибки формата</strong>
            <div className="muted">
              Разобрано: {resolvedInvalidCount} из {invalidItems.length}. Не
              разобрано: {unresolvedInvalidCount}.
            </div>
          </div>
          <div className="actions">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={applyBulkInvalidSkip}
            >
              Исключить все
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              disabled={resolvedInvalidCount === 0}
              onClick={clearInvalidDecisions}
            >
              Сбросить решения
            </button>
          </div>
        </div>

        {invalidItems.map((item) => {
          const draftAction =
            draftActions[item.row] || resolutions[item.row]?.action || "";
          const appliedResolution = resolutions[item.row];

          return (
            <div key={item.id} className="invalid-step stack">
              <div className="section-title-row compact-title-row">
                <div>
                  <h2>Строка {item.row}</h2>
                  <div className="field-error-list">
                    {(
                      item.errors ?? ["Строка не прошла проверку формата."]
                    ).map((message, index) => (
                      <div key={index}>{message}</div>
                    ))}
                  </div>
                </div>
                <span
                  className={`badge ${appliedResolution ? "badge-success" : "import-status-invalid"}`}
                >
                  {appliedResolution ? "решение применено" : "ожидает решения"}
                </span>
              </div>

              <div className="decision-panel stack">
                <div className="section-title-row compact-title-row">
                  <div>
                    <h3>Решение</h3>
                    <div className="muted">
                      {draftAction === "skip"
                        ? "Строка будет исключена из импорта."
                        : draftAction === "merge_edit"
                          ? "Строка будет импортирована после ручного исправления полей."
                          : "Исключите строку или отредактируйте её поля."}
                    </div>
                  </div>
                  <div className="actions">
                    <button
                      type="button"
                      className={`btn btn-secondary ${draftAction === "skip" ? "active" : ""}`}
                      onClick={() =>
                        handleChooseCollisionAction(item.row, "skip")
                      }
                    >
                      Исключить строку
                    </button>
                    <button
                      type="button"
                      className={`btn btn-secondary ${draftAction === "merge_edit" ? "active" : ""}`}
                      onClick={() => {
                        setMergedRecords((current) => ({
                          ...current,
                          [item.row]:
                            current[item.row] ?? cloneRecord(item.incoming),
                        }));
                        handleChooseCollisionAction(item.row, "merge_edit");
                      }}
                    >
                      ✎ Редактировать
                    </button>
                    <button
                      type="button"
                      className="btn btn-primary"
                      disabled={!draftAction}
                      onClick={() => handleApplyInvalidDecision(item)}
                    >
                      Применить решение
                    </button>
                  </div>
                </div>

                {draftAction === "merge_edit"
                  ? renderEditableRecordForm(item.row)
                  : null}
              </div>
            </div>
          );
        })}
      </div>
    );
  }

  return (
    <div className="stack">
      <FixedToast toast={toast} />
      {error ? (
        <div className="visually-hidden" role="alert">
          {error}
        </div>
      ) : null}
      <PageHeader
        title="Импорт и пересчёт"
        subtitle="Безопасная загрузка словарного банка: проверка файла, подсчёт новых строк, повторов, исправление ошибок формата и последовательное разрешение коллизий перед сохранением."
      />

      <div className="card stack">
        <div>
          <label className="label">Файл словарного банка JSONL</label>
          <input
            className="input"
            type="file"
            accept=".jsonl"
            onChange={(event) => {
              setSelectedFile(event.target.files?.[0] ?? null);
              setPreview(null);
              setCommitResult(null);
              setClientErrors([]);
              setResolutions({});
              setDraftActions({});
              setMergedRecords({});
              setCollisionIndex(0);
            }}
          />
        </div>

        <div className="notice">
          Если файл не выбран, сервер проверит файл по умолчанию из
          ACTIVE_BANK_PATH или data/lexicon/active_bank.jsonl.
        </div>

        {clientErrors.length > 0 ? (
          <div className="error">
            {clientErrors.map((message, index) => (
              <div key={index}>{message}</div>
            ))}
          </div>
        ) : null}

        <div className="actions">
          <button
            className="btn btn-primary"
            onClick={() => void handlePreview()}
            disabled={loadingPreview || loadingCommit}
          >
            {loadingPreview ? "Проверка..." : "Проверить файл"}
          </button>

          <button
            className="btn btn-secondary"
            onClick={() => void handleRecalculate()}
            disabled={loadingMeta || loadingCommit}
          >
            {loadingMeta ? "Пересчёт..." : "Пересчитать направления"}
          </button>
        </div>
      </div>

      {preview ? (
        <div className="stack">
          {renderPreviewStats(preview)}

          <div className="card stack">
            <div className="section-title-row compact-title-row">
              <div>
                <h2>Предпросмотр импорта</h2>
                <div className="muted">Сессия: {preview.session_id}</div>
                {preview.stats.collisions > 0 ? (
                  <div className="muted">
                    Commit доступен после последовательного разрешения всех
                    коллизий.
                  </div>
                ) : null}
              </div>
              <button
                className="btn btn-primary"
                onClick={() => void handleCommit()}
                disabled={
                  loadingCommit ||
                  unresolvedInvalidCount > 0 ||
                  unresolvedCollisionCount > 0
                }
              >
                {loadingCommit ? "Применение..." : "Применить импорт"}
              </button>
            </div>

            {unresolvedInvalidCount > 0 ? (
              <div className="error">
                Сначала исправьте или исключите строки с ошибками формата:
                осталось {unresolvedInvalidCount}.
              </div>
            ) : null}

            {unresolvedCollisionCount > 0 ? (
              <div className="notice">
                Сначала разрешите все коллизии: осталось{" "}
                {unresolvedCollisionCount}.
              </div>
            ) : null}

            <div className="tabs-row">
              {IMPORT_TABS.map((tab) => (
                <button
                  key={tab.value}
                  type="button"
                  className={`tab-btn ${activeTab === tab.value ? "active" : ""}`}
                  onClick={() => setActiveTab(tab.value)}
                >
                  {tab.label} ({filterItems(preview, tab.value).length})
                </button>
              ))}
            </div>

            {activeTab === "collision" ? (
              renderCollisionDeck()
            ) : activeTab === "invalid" ? (
              renderInvalidDeck()
            ) : (
              <div className="stack">
                {currentItems.map(renderItem)}
                {currentItems.length === 0 ? (
                  <div className="notice">В этом разделе строк нет.</div>
                ) : null}
              </div>
            )}
          </div>
        </div>
      ) : null}

      {commitResult ? (
        <div className="card stack">
          <h2>Результат применения</h2>
          <dl className="kv">
            <dt>Вставлено</dt>
            <dd>{commitResult.stats.inserted}</dd>
            <dt>Обновлено</dt>
            <dd>{commitResult.stats.updated}</dd>
            <dt>Пропущено</dt>
            <dd>{commitResult.stats.skipped}</dd>
            <dt>Ошибок</dt>
            <dd>{commitResult.stats.failed}</dd>
            <dt>Версия снимка</dt>
            <dd>{commitResult.snapshot_version_code ?? "—"}</dd>
            <dt>Checksum</dt>
            <dd className="break-word">
              {commitResult.snapshot_checksum ?? "—"}
            </dd>
          </dl>
        </div>
      ) : null}

      {metaResult ? (
        <div className="card stack">
          <h2>Результат пересчёта направлений</h2>
          <dl className="kv">
            <dt>Обработано</dt>
            <dd>{metaResult.processed}</dd>
            <dt>Направления</dt>
            <dd>{metaResult.directions ?? "—"}</dd>
            <dt>Версия снимка</dt>
            <dd>{metaResult.snapshot?.version_code ?? "—"}</dd>
            <dt>Код проверки</dt>
            <dd className="break-word">
              {metaResult.snapshot?.checksum ?? "—"}
            </dd>
          </dl>
        </div>
      ) : null}
    </div>
  );
}
