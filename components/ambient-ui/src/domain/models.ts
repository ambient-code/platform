export const MODEL_OPTIONS = [
  'claude-sonnet-4-6',
  'claude-opus-4-6',
  'claude-opus-4-5',
  'claude-opus-4-1',
  'claude-sonnet-4-5',
  'claude-haiku-4-5',
] as const

export type ModelId = (typeof MODEL_OPTIONS)[number]

export const DEFAULT_MODEL: ModelId = 'claude-sonnet-4-6'
