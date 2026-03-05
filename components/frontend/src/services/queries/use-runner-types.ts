import { useQuery } from "@tanstack/react-query";
import { getRunnerTypes } from "../api/runner-types";

export const runnerTypeKeys = {
  all: ["runner-types"] as const,
  list: () => [...runnerTypeKeys.all, "list"] as const,
};

/**
 * Fetch available runner types from the backend registry.
 * Runner types rarely change, so we cache aggressively.
 */
export function useRunnerTypes() {
  return useQuery({
    queryKey: runnerTypeKeys.list(),
    queryFn: getRunnerTypes,
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 30 * 60 * 1000, // 30 minutes
  });
}
