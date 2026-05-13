type AnyRecord = Record<string, unknown>;

function asRecord(value: unknown): AnyRecord {
  return value && typeof value === 'object' ? (value as AnyRecord) : {};
}

export function pick<T = unknown>(value: unknown, keys: string[], fallback?: T): T {
  const record = asRecord(value);
  for (const key of keys) {
    const selected = record[key];
    if (selected !== undefined && selected !== null) return selected as T;
  }
  return fallback as T;
}

export function pickString(value: unknown, keys: string[], fallback = ''): string {
  const selected = pick<unknown>(value, keys, fallback);
  if (selected === undefined || selected === null) return fallback;
  return String(selected);
}

export function pickNullableString(value: unknown, keys: string[]): string | null {
  const selected = pick<unknown>(value, keys, null);
  if (selected === undefined || selected === null || selected === '') return null;
  return String(selected);
}

export function pickNumber(value: unknown, keys: string[], fallback = 0): number {
  const selected = pick<unknown>(value, keys, fallback);
  if (typeof selected === 'number') return selected;
  if (typeof selected === 'string' && selected.trim() !== '') {
    const parsed = Number(selected);
    return Number.isFinite(parsed) ? parsed : fallback;
  }
  return fallback;
}

export function pickNullableNumber(value: unknown, keys: string[]): number | null {
  const selected = pick<unknown>(value, keys, null);
  if (selected === undefined || selected === null || selected === '') return null;
  if (typeof selected === 'number') return selected;
  if (typeof selected === 'string') {
    const parsed = Number(selected);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

export function pickBoolean(value: unknown, keys: string[], fallback = false): boolean {
  const selected = pick<unknown>(value, keys, fallback);
  if (typeof selected === 'boolean') return selected;
  if (typeof selected === 'string') return selected === 'true' || selected === '1';
  if (typeof selected === 'number') return selected !== 0;
  return fallback;
}

export function asArray<T = unknown>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}
