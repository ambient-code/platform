import { AmbientClient } from '@/lib/sdk';

export function getClient(projectName?: string): AmbientClient {
  return new AmbientClient({ baseUrl: '/', project: projectName });
}
