import { AmbientClient } from '@ambient-platform/sdk';
import type { AmbientClientConfig } from '@ambient-platform/sdk';

const AMBIENT_API_URL =
  process.env.NEXT_PUBLIC_AMBIENT_API_URL || 'http://localhost:8000/api/ambient-api-server/v1';

export function createAmbientClient(project: string, token?: string): AmbientClient {
  const resolvedToken =
    token ||
    (typeof window !== 'undefined' ? process.env.NEXT_PUBLIC_E2E_TOKEN : undefined) ||
    process.env.OC_TOKEN ||
    'no-auth';

  const config: AmbientClientConfig = {
    baseUrl: AMBIENT_API_URL,
    token: resolvedToken,
    project: project || 'default',
  };

  return new AmbientClient(config);
}
