import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as googleAuthApi from "@/services/api/google-auth";

export const googleKeys = {
  all: ["google"] as const,
  status: () => [...googleKeys.all, "status"] as const,
};

/**
 * Hook to fetch Google OAuth connection status
 */
export function useGoogleStatus() {
  return useQuery({
    queryKey: googleKeys.status(),
    queryFn: googleAuthApi.getGoogleStatus,
    staleTime: 60 * 1000, // 1 minute
  });
}

/**
 * Hook to disconnect Google OAuth
 */
export function useDisconnectGoogle() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: googleAuthApi.disconnectGoogle,
    onSuccess: () => {
      // Invalidate status query to refetch
      queryClient.invalidateQueries({ queryKey: googleKeys.status() });
    },
  });
}

