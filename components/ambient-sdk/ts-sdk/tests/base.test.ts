import { buildQueryString, AmbientAPIError } from '../src';
import type { ListOptions, APIError } from '../src';

describe('buildQueryString', () => {
  it('returns empty string for undefined opts', () => {
    expect(buildQueryString(undefined)).toBe('');
  });

  it('returns empty string for empty opts', () => {
    expect(buildQueryString({})).toBe('');
  });

  it('builds page and size', () => {
    const qs = buildQueryString({ page: 2, size: 50 });
    expect(qs).toContain('page=2');
    expect(qs).toContain('size=50');
    expect(qs.startsWith('?')).toBe(true);
  });

  it('caps size at 65500', () => {
    const qs = buildQueryString({ size: 100000 });
    expect(qs).toContain('size=65500');
  });

  it('includes search param', () => {
    const qs = buildQueryString({ search: 'test query' });
    expect(qs).toContain('search=');
  });

  it('includes orderBy param', () => {
    const qs = buildQueryString({ orderBy: 'created_at desc' });
    expect(qs).toContain('orderBy=');
  });

  it('includes fields param', () => {
    const qs = buildQueryString({ fields: 'id,name,phase' });
    expect(qs).toContain('fields=');
  });

  it('combines multiple params', () => {
    const qs = buildQueryString({ page: 1, size: 25, search: 'test' });
    expect(qs).toContain('page=1');
    expect(qs).toContain('size=25');
    expect(qs).toContain('search=test');
  });
});

describe('AmbientAPIError', () => {
  it('is instanceof Error', () => {
    const err = new AmbientAPIError({
      id: '', kind: 'Error', href: '',
      code: 'forbidden', reason: 'Access denied',
      operation_id: '', status_code: 403,
    });
    expect(err).toBeInstanceOf(Error);
    expect(err).toBeInstanceOf(AmbientAPIError);
  });

  it('exposes structured fields', () => {
    const err = new AmbientAPIError({
      id: 'err-1', kind: 'Error', href: '/errors/err-1',
      code: 'validation_error', reason: 'Invalid field',
      operation_id: 'op-123', status_code: 422,
    });
    expect(err.statusCode).toBe(422);
    expect(err.code).toBe('validation_error');
    expect(err.reason).toBe('Invalid field');
    expect(err.operationId).toBe('op-123');
  });
});
