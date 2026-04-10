import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { RunnerModelSelector, getDefaultModel } from '../runner-model-selector';
import type { RunnerType } from '@/services/api/runner-types';
import type { ListModelsResponse } from '@/types/api';

const mockRunnerTypes: RunnerType[] = [
  {
    id: 'claude-code',
    displayName: 'Claude Code',
    description: 'Claude Code runner',
    framework: 'claude',
    provider: 'anthropic',
    auth: { requiredSecretKeys: [], secretKeyLogic: 'any', vertexSupported: false },
  },
  {
    id: 'gemini-cli',
    displayName: 'Gemini CLI',
    description: 'Gemini CLI runner',
    framework: 'gemini',
    provider: 'google',
    auth: { requiredSecretKeys: [], secretKeyLogic: 'any', vertexSupported: false },
  },
];

const mockAnthropicModels: ListModelsResponse = {
  models: [
    { id: 'claude-haiku-4-5', label: 'Claude Haiku 4.5', provider: 'anthropic', isDefault: false },
    { id: 'claude-sonnet-4-5', label: 'Claude Sonnet 4.5', provider: 'anthropic', isDefault: false },
    { id: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6', provider: 'anthropic', isDefault: true },
    { id: 'claude-opus-4-5', label: 'Claude Opus 4.5', provider: 'anthropic', isDefault: false },
    { id: 'claude-opus-4-6', label: 'Claude Opus 4.6', provider: 'anthropic', isDefault: false },
  ],
  defaultModel: 'claude-sonnet-4-6',
};

const mockUseRunnerTypes = vi.fn(() => ({ data: mockRunnerTypes }));
const mockUseModels = vi.fn(() => ({ data: mockAnthropicModels }));

vi.mock('@/services/queries/use-runner-types', () => ({
  useRunnerTypes: () => mockUseRunnerTypes(),
}));

vi.mock('@/services/queries/use-models', () => ({
  useModels: () => mockUseModels(),
}));

describe('RunnerModelSelector', () => {
  const defaultProps = {
    projectName: 'test-project',
    selectedRunner: 'claude-code',
    selectedModel: 'claude-sonnet-4-6',
    onSelect: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseRunnerTypes.mockReturnValue({ data: mockRunnerTypes });
    mockUseModels.mockReturnValue({ data: mockAnthropicModels });
  });

  it('renders trigger button with runner and model name', () => {
    render(<RunnerModelSelector {...defaultProps} />);
    const button = screen.getByRole('button');
    expect(button.textContent).toContain('Claude Code');
    expect(button.textContent).toContain('Claude Sonnet 4.6');
  });

  it('renders trigger button with unknown runner fallback', () => {
    render(
      <RunnerModelSelector
        {...defaultProps}
        selectedRunner="unknown-runner"
        selectedModel="default"
      />
    );
    const button = screen.getByRole('button');
    expect(button.textContent).toContain('unknown-runner');
  });

  it('renders trigger button when no runners available', () => {
    mockUseRunnerTypes.mockReturnValue({ data: [] });
    render(<RunnerModelSelector {...defaultProps} />);
    expect(screen.getByRole('button')).toBeDefined();
  });

  it('resolves 4.6 model name from API data in trigger button', () => {
    render(
      <RunnerModelSelector
        {...defaultProps}
        selectedModel="claude-sonnet-4-6"
      />
    );
    const button = screen.getByRole('button');
    expect(button.textContent).toContain('Claude Sonnet 4.6');
  });

  it('resolves 4.6 opus model name from API data in trigger button', () => {
    render(
      <RunnerModelSelector
        {...defaultProps}
        selectedModel="claude-opus-4-6"
      />
    );
    const button = screen.getByRole('button');
    expect(button.textContent).toContain('Claude Opus 4.6');
  });
});

describe('getDefaultModel', () => {
  it('returns the API default model when provided', () => {
    const models = [
      { id: 'model-1', name: 'Model 1' },
      { id: 'model-2', name: 'Model 2' },
    ];
    expect(getDefaultModel(models, 'model-1')).toBe('model-1');
  });

  it('returns second model when no default specified', () => {
    const models = [
      { id: 'model-1', name: 'Model 1' },
      { id: 'model-2', name: 'Model 2' },
      { id: 'model-3', name: 'Model 3' },
    ];
    expect(getDefaultModel(models)).toBe('model-2');
  });

  it('falls back to first model when only one exists', () => {
    const models = [{ id: 'model-1', name: 'Model 1' }];
    expect(getDefaultModel(models)).toBe('model-1');
  });

  it('returns "default" for empty model list', () => {
    expect(getDefaultModel([])).toBe('default');
  });
});
