export type LLMSettings = {
  model: string;
  temperature: number;
  maxTokens: number;
};

export type S3StorageConfig = {
  enabled: boolean;
  endpoint: string;
  bucket: string;
  region?: string;
};

export type ProjectDefaultSettings = {
  llmSettings: LLMSettings;
  defaultTimeout: number;
  allowedWebsiteDomains?: string[];
  maxConcurrentSessions: number;
  s3Storage?: S3StorageConfig;
};

export type ProjectResourceLimits = {
  maxCpuPerSession: string;
  maxMemoryPerSession: string;
  maxStoragePerSession: string;
  diskQuotaGB: number;
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

// Types for the project-settings REST API (backed by ProjectSettings CR)

export type DefaultConfigRepo = {
  gitUrl: string;
  branch?: string;
};

export type ProjectSettingsCR = {
  defaultConfigRepo?: DefaultConfigRepo;
};

export type UpdateProjectSettingsRequest = {
  defaultConfigRepo?: DefaultConfigRepo;
};