/**
 * Learned files API service
 * Fetches learned files from the workspace repo via repo tree/blob endpoints
 * and manages draft PRs with the continuous-learning label.
 */

import { apiClient } from './client';

/**
 * A parsed learned file entry from docs/learned/
 */
export type LearnedEntry = {
  title: string;
  type: 'correction' | 'pattern';
  date: string;
  author: string;
  contentPreview: string;
  filePath: string;
  source: string;
  session: string;
};

/**
 * A draft PR with the continuous-learning label
 */
export type LearnedDraftPR = {
  number: number;
  title: string;
  url: string;
  createdAt: string;
  author: string;
  body: string;
};

/**
 * Response from the learned files endpoint
 */
export type LearnedFilesResponse = {
  entries: LearnedEntry[];
  totalCount: number;
};

/**
 * Response from the draft PRs endpoint
 */
export type LearnedDraftPRsResponse = {
  prs: LearnedDraftPR[];
};

/**
 * Request body for creating a new memory (Add Memory)
 */
export type CreateMemoryRequest = {
  title: string;
  content: string;
  type: 'correction' | 'pattern';
  repo?: string;
};

/**
 * Response from creating a memory
 */
export type CreateMemoryResponse = {
  prUrl: string;
  prNumber: number;
};

/**
 * Fetch learned files for a project
 */
export async function getLearnedFiles(
  projectName: string,
  params?: { type?: string; page?: number; pageSize?: number }
): Promise<LearnedFilesResponse> {
  const searchParams: Record<string, string | number | boolean> = {};
  if (params?.type) searchParams.type = params.type;
  if (params?.page !== undefined) searchParams.page = params.page;
  if (params?.pageSize !== undefined) searchParams.pageSize = params.pageSize;

  return apiClient.get<LearnedFilesResponse>(
    `/projects/${projectName}/learned`,
    { params: searchParams }
  );
}

/**
 * Fetch draft PRs with continuous-learning label
 */
export async function getLearnedDraftPRs(
  projectName: string,
): Promise<LearnedDraftPRsResponse> {
  return apiClient.get<LearnedDraftPRsResponse>(
    `/projects/${projectName}/learned/prs`
  );
}

/**
 * Create a new memory (Add Memory form submission)
 */
export async function createMemory(
  projectName: string,
  data: CreateMemoryRequest,
): Promise<CreateMemoryResponse> {
  return apiClient.post<CreateMemoryResponse, CreateMemoryRequest>(
    `/projects/${projectName}/learned/create`,
    data
  );
}
