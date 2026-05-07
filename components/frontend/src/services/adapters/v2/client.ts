import { AmbientClient } from '@/lib/sdk';

export type SdkClient = Pick<AmbientClient, 'projects' | 'sessions' | 'scheduledSessions' | 'agents'>;

export function getClient(projectName?: string): SdkClient {
  return new AmbientClient({ baseUrl: '/', project: projectName });
}
