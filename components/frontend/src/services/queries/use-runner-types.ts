import { useQuery } from "@tanstack/react-query";
import { getRunnerTypes } from "../api/runner-types";

export const runnerTypeKeys = {
  all: ["runner-types"] as const,
  global: () => [...runnerTypeKeys.all, "global"] as const,
  forProject: (projectName: string) => [...runnerTypeKeys.all, projectName] as const,
};

/**
 * Fetch available runner types for a project (with workspace override support).
 */
export function useRunnerTypes(projectName: string) {
  return useQuery({
    queryKey: runnerTypeKeys.forProject(projectName),
    queryFn: () => getRunnerTypes(projectName),
    enabled: !!projectName,
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
  });
}
