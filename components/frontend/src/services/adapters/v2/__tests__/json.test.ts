import { describe, it, expect } from 'vitest';
import { parseJsonField } from '../json';

describe('parseJsonField', () => {
  it('returns fallback for null', () => {
    expect(parseJsonField(null, {})).toEqual({});
  });

  it('returns fallback for undefined', () => {
    expect(parseJsonField(undefined, [])).toEqual([]);
  });

  it('returns fallback for empty string', () => {
    expect(parseJsonField('', { a: 1 })).toEqual({ a: 1 });
  });

  it('returns object as-is when value is already an object', () => {
    const obj = { key: 'value' };
    expect(parseJsonField(obj, {})).toBe(obj);
  });

  it('returns array as-is when value is already an array', () => {
    const arr = [1, 2, 3];
    expect(parseJsonField(arr, [])).toBe(arr);
  });

  it('parses valid JSON string', () => {
    expect(parseJsonField('{"a":"b"}', {})).toEqual({ a: 'b' });
  });

  it('parses JSON array string', () => {
    expect(parseJsonField('[1,2]', [])).toEqual([1, 2]);
  });

  it('returns fallback for malformed JSON string', () => {
    expect(parseJsonField('{not-json', 'default')).toBe('default');
  });

  it('returns fallback for non-string non-object values', () => {
    expect(parseJsonField(42, 'fallback')).toBe('fallback');
    expect(parseJsonField(true, 'fallback')).toBe('fallback');
  });
});
