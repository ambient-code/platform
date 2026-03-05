import { apiClient } from "./client";

export type RunnerModel = {
  value: string;
  label: string;
};

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
  defaultModel: string;
  models: RunnerModel[];
  /** @deprecated Use auth.requiredSecretKeys instead */
  requiredSecretKeys?: string[];
  auth: RunnerTypeAuth;
};

export const DEFAULT_RUNNER_TYPE_ID = "claude-agent-sdk" as const;

export async function getRunnerTypes(): Promise<RunnerType[]> {
  return apiClient.get<RunnerType[]>("/runner-types");
}
