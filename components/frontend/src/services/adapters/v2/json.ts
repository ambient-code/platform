export function parseJsonField<T>(value: unknown, fallback: T): T {
  if (value === null || value === undefined || value === '') {
    return fallback;
  }
  if (typeof value === 'object') {
    return value as T;
  }
  if (typeof value === 'string') {
    try {
      return JSON.parse(value) as T;
    } catch {
      return fallback;
    }
  }
  return fallback;
}
