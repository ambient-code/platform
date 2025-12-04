export type LLMSettings = {
  model: string;
  temperature: number;
  maxTokens: number;
};

export type ProjectDefaultSettings = {
  llmSettings: LLMSettings;
  defaultTimeout: number;
  allowedWebsiteDomains?: string[];
  maxConcurrentSessions: number;
};

export type ProjectResourceLimits = {
  maxCpuPerSession: string;
  maxMemoryPerSession: string;
  maxStoragePerSession: string;
  diskQuotaGB: number;
};

// Workspace container customization settings.
// Workspace container mode is always enabled (ADR-0006); these settings allow optional customization.
export type WorkspaceContainerSettings = {
  image?: string;
  resources?: {
    cpuRequest?: string;
    cpuLimit?: string;
    memoryRequest?: string;
    memoryLimit?: string;
  };
};

export type ObjectMeta = {
  name: string;
  namespace: string;
  creationTimestamp: string;
  uid?: string;
};

export type ProjectSettings = {
  projectName: string;
  adminUsers: string[];
  defaultSettings: ProjectDefaultSettings;
  resourceLimits: ProjectResourceLimits;
  metadata: ObjectMeta;
};

export type ProjectSettingsUpdateRequest = {
  projectName: string;
  adminUsers: string[];
  defaultSettings: ProjectDefaultSettings;
  resourceLimits: ProjectResourceLimits;
};