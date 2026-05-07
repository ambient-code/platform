import { AmbientAPIError } from '@/lib/sdk';
import { ApiClientError } from '@/types/api/common';

export function wrapSdkError(err: unknown): never {
  if (err instanceof AmbientAPIError) {
    throw new ApiClientError(err.reason, err.code, {
      statusCode: err.statusCode,
      operationId: err.operationId,
    });
  }
  if (err instanceof Error) {
    throw new ApiClientError(err.message);
  }
  throw new ApiClientError(String(err));
}
