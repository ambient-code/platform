import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { googleAdapter } from '../adapters/google';
import type { GooglePort } from '../ports/google';
import { BACKEND_VERSION } from './query-keys';

export const googleKeys = {
  all: [BACKEND_VERSION, 'google'] as const,
  status: () => [...googleKeys.all, 'status'] as const,
};

export function useGoogleStatus(port: GooglePort = googleAdapter) {
  return useQuery({
    queryKey: googleKeys.status(),
    queryFn: port.getGoogleStatus,
    staleTime: 60 * 1000,
  });
}

export function useDisconnectGoogle(port: GooglePort = googleAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: port.disconnectGoogle,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: googleKeys.status() });
    },
  });
}
