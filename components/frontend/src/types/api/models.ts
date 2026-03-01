/**
 * Model types for the models API
 */

export type LLMModel = {
  id: string;
  label: string;
  provider: string;
  tier: string;
  isDefault: boolean;
};

export type ListModelsResponse = {
  models: LLMModel[];
  defaultModel: string;
};
