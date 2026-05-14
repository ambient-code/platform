import { useQuery } from '@tanstack/react-query';
import { authAdapter } from '../adapters/auth';
import type { AuthPort } from '../ports/auth';
import { BACKEND_VERSION } from './query-keys';

export const authKeys = {
  all: [BACKEND_VERSION, 'auth'] as const,
  currentUser: () => [...authKeys.all, 'currentUser'] as const,
};

export function useCurrentUser(port: AuthPort = authAdapter) {
  return useQuery({
    queryKey: authKeys.currentUser(),
    queryFn: port.getCurrentUser,
    staleTime: 5 * 60 * 1000,
  });
}
