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
  dependencies?: string;
  caveats?: string;
  analyzed_by_session_id?: string;
  analyzed_by_agent_id?: string;
  analyzed_at?: string;
  confidence?: number;
  version: number;
};

export type RepoFinding = {
  id: string;
  kind: string;
  href: string;
  created_at: string;
  updated_at: string;
  intelligence_id: string;
  file_path: string;
  category: string;
  status: string;
  title: string;
  body: string;
  severity?: string;
  confidence?: number;
  source_type: string;
  source_ref?: string;
  session_id?: string;
  agent_id?: string;
  resolved_by?: string;
  resolved_reason?: string;
};

export type RepoFindingList = {
  kind: string;
  page: number;
  size: number;
  total: number;
  items: RepoFinding[];
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

/**
 * Fetch active findings for an intelligence record.
 */
export async function getRepoFindings(
  projectName: string,
  intelligenceId: string
): Promise<RepoFinding[]> {
  try {
    const data = await apiClient.get<RepoFindingList>(
      `/projects/${projectName}/intelligence/findings`,
      { params: { intelligence_id: intelligenceId } }
    );
    return data.items ?? [];
  } catch {
    return [];
  }
}
