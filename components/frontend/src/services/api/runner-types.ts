import { apiClient } from "./client";

export interface RunnerModel {
  value: string;
  label: string;
}

export interface RunnerType {
  id: string;
  displayName: string;
  description: string;
  defaultModel: string;
  models: RunnerModel[];
}

export const DEFAULT_RUNNER_TYPE_ID = "claude-agent-sdk" as const;

export async function getRunnerTypes(): Promise<RunnerType[]> {
  return apiClient.get<RunnerType[]>("/runner-types");
}
