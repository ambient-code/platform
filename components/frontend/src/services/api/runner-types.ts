import { apiClient } from "./client";

export type RunnerTypeAuth = {
  requiredSecretKeys: string[];
  secretKeyLogic: "any" | "all";
  vertexSupported: boolean;
};

export type RunnerType = {
  id: string;
  displayName: string;
  description: string;
  framework: string;
  provider: string;
  auth: RunnerTypeAuth;
};

export const DEFAULT_RUNNER_TYPE_ID = "claude-agent-sdk" as const;

export async function getRunnerTypes(projectName: string): Promise<RunnerType[]> {
  return apiClient.get<RunnerType[]>(`/projects/${projectName}/runner-types`);
}

export async function getRunnerTypesGlobal(): Promise<RunnerType[]> {
  return apiClient.get<RunnerType[]>("/runner-types");
}
