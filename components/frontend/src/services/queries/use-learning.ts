/**
 * React Query hooks for Learning dashboard data.
 * Gated behind learning-agent-loop feature flag at the component level.
 */

import { useQuery } from "@tanstack/react-query";
import * as learningApi from "../api/learning";

export const learningKeys = {
  all: ["learning"] as const,
  summary: (projectName: string) =>
    [...learningKeys.all, "summary", projectName] as const,
  timeline: (projectName: string, page: number) =>
    [...learningKeys.all, "timeline", projectName, page] as const,
};

/**
 * Fetch learning summary for a project.
 */
export function useLearningSummary(projectName: string) {
  return useQuery({
    queryKey: learningKeys.summary(projectName),
    queryFn: () => learningApi.getLearningSummary(projectName),
    enabled: !!projectName,
    staleTime: 30000,
  });
}

/**
 * Fetch paginated learning timeline for a project.
 */
export function useLearningTimeline(
  projectName: string,
  page = 1,
  pageSize = 20
) {
  return useQuery({
    queryKey: learningKeys.timeline(projectName, page),
    queryFn: () => learningApi.getLearningTimeline(projectName, page, pageSize),
    enabled: !!projectName,
    staleTime: 30000,
  });
}
