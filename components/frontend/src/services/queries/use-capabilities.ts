import { useQuery } from "@tanstack/react-query";
import * as sessionsApi from "@/services/api/sessions";

export const capabilitiesKeys = {
  all: ["capabilities"] as const,
  session: (projectName: string, sessionName: string) =>
    [...capabilitiesKeys.all, projectName, sessionName] as const,
};

/**
 * Fetch the runner's capabilities manifest for a session.
 *
 * Returns which AG-UI features the framework supports, which platform
 * features are available, and runtime config (model, tracing, etc.).
 * The frontend uses this to conditionally render UI panels.
 */
export function useCapabilities(
  projectName: string,
  sessionName: string,
  enabled: boolean = true
) {
  return useQuery({
    queryKey: capabilitiesKeys.session(projectName, sessionName),
    queryFn: () => sessionsApi.getCapabilities(projectName, sessionName),
    enabled: enabled && !!projectName && !!sessionName,
    staleTime: 60 * 1000, // 1 minute — capabilities rarely change mid-session
    retry: 2,
    // Poll until runner is ready (returns real data)
    refetchInterval: (query) => {
      if (query.state.data?.framework && query.state.data.framework !== "unknown") {
        return false;
      }
      // Stop after ~1 min (6 × 10s)
      const updatedCount =
        (query.state as { dataUpdatedCount?: number }).dataUpdatedCount ?? 0;
      if (updatedCount >= 6) return false;
      return 10 * 1000;
    },
  });
}
