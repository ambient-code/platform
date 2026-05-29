'use client';

/**
 * React Query Provider
 * Wraps the app with QueryClientProvider for data fetching.
 * Includes global 401 detection and session expired dialog.
 */

import { QueryClientProvider } from '@tanstack/react-query';
import { getQueryClient, onSessionExpired } from '@/lib/query-client';
import { SessionExpiredDialog } from '@/components/session-expired-dialog';
import { useState, useEffect, useCallback } from 'react';

type QueryProviderProps = {
  children: React.ReactNode;
};

export function QueryProvider({ children }: QueryProviderProps) {
  const [queryClient] = useState(() => getQueryClient());
  const [sessionExpired, setSessionExpired] = useState(false);

  const handleSessionExpired = useCallback(() => {
    setSessionExpired(true);
  }, []);

  useEffect(() => {
    onSessionExpired(handleSessionExpired);
  }, [handleSessionExpired]);

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <SessionExpiredDialog open={sessionExpired} />
    </QueryClientProvider>
  );
}
