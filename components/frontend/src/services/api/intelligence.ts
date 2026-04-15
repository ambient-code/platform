/**
 * Repo Intelligence API service
 * Fetches project intelligence and findings from the api-server via Next.js proxy routes
 */

import { apiClient } from './client';

export type RepoIntelligence = {
  id: string;
  kind: string;
  href: string;
  created_at: string;
  updated_at: string;
  project_id: string;
  repo_url: string;
  repo_branch: string;
  summary: string;
  language: string;
  framework?: string;
  build_system?: string;
  test_strategy?: string;
  architecture?: string;
  conventions?: string;
  caveats?: string;
  analyzed_by_session_id?: string;
  analyzed_by_agent_id?: string;
  analyzed_at?: string;
  confidence?: number;
  version: number;
};

/**
 * Fetch intelligence for a repo in a project. Returns null if none exists.
 */
export async function getRepoIntelligence(
  projectName: string,
  repoUrl: string
): Promise<RepoIntelligence | null> {
  try {
    return await apiClient.get<RepoIntelligence>(
      `/projects/${projectName}/intelligence`,
      { params: { repo_url: repoUrl } }
    );
  } catch {
    return null;
  }
}
