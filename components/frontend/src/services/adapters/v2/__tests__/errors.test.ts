import { describe, it, expect } from 'vitest';
import { wrapSdkError } from '../errors';
import { AmbientAPIError } from '@/lib/sdk';
import { ApiClientError } from '@/types/api/common';

describe('wrapSdkError', () => {
  it('converts AmbientAPIError to ApiClientError with statusCode and operationId', () => {
    const sdkError = new AmbientAPIError({
      id: 'err-1',
      kind: 'Error',
      href: '',
      code: 'NOT_FOUND',
      reason: 'Session not found',
      operation_id: 'op-123',
      status_code: 404,
    });

    expect(() => wrapSdkError(sdkError)).toThrow(ApiClientError);
    try {
      wrapSdkError(sdkError);
    } catch (e) {
      const err = e as ApiClientError;
      expect(err.message).toBe('Session not found');
      expect(err.code).toBe('NOT_FOUND');
      expect(err.details).toEqual({ statusCode: 404, operationId: 'op-123' });
    }
  });

  it('converts generic Error to ApiClientError', () => {
    const error = new Error('network failure');

    expect(() => wrapSdkError(error)).toThrow(ApiClientError);
    try {
      wrapSdkError(error);
    } catch (e) {
      const err = e as ApiClientError;
      expect(err.message).toBe('network failure');
      expect(err.code).toBeUndefined();
    }
  });

  it('converts non-Error values to ApiClientError with string message', () => {
    expect(() => wrapSdkError('string error')).toThrow(ApiClientError);
    try {
      wrapSdkError('string error');
    } catch (e) {
      expect((e as ApiClientError).message).toBe('string error');
    }
  });

  it('converts number to ApiClientError', () => {
    expect(() => wrapSdkError(404)).toThrow(ApiClientError);
    try {
      wrapSdkError(404);
    } catch (e) {
      expect((e as ApiClientError).message).toBe('404');
    }
  });
});
