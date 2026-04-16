/**
 * API client for Learning dashboard endpoints.
 * Proxies through Next.js API routes to backend.
 */

import { apiClient } from "./client";
import type { LearningSummary, TimelineResponse } from "@/types/learning";

/**
 * Fetch learning summary metrics for a project.
 */
export function getLearningSummary(
  projectName: string
): Promise<LearningSummary> {
  return apiClient.get<LearningSummary>(
    `/projects/${encodeURIComponent(projectName)}/learning/summary`
  );
}

/**
 * Fetch learning timeline (paginated, reverse-chronological).
 */
export function getLearningTimeline(
  projectName: string,
  page = 1,
  pageSize = 20
): Promise<TimelineResponse> {
  return apiClient.get<TimelineResponse>(
    `/projects/${encodeURIComponent(projectName)}/learning/timeline`,
    { params: { page, pageSize } }
  );
}
