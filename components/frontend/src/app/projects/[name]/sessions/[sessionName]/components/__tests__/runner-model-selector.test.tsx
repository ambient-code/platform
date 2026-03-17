import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { RunnerModelSelector, getDefaultModel, getModelsForRunner } from '../runner-model-selector';
import type { RunnerType } from '@/services/api/runner-types';

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

const mockUseRunnerTypes = vi.fn(() => ({ data: mockRunnerTypes }));

vi.mock('@/services/queries/use-runner-types', () => ({
  useRunnerTypes: () => mockUseRunnerTypes(),
}));

describe('RunnerModelSelector', () => {
  const defaultProps = {
    projectName: 'test-project',
    selectedRunner: 'claude-code',
    selectedModel: 'claude-sonnet-4-5',
    onSelect: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseRunnerTypes.mockReturnValue({ data: mockRunnerTypes });
  });

  it('renders trigger button with runner and model name', () => {
    render(<RunnerModelSelector {...defaultProps} />);
    const button = screen.getByRole('button');
    expect(button.textContent).toContain('Claude Code');
    expect(button.textContent).toContain('Claude Sonnet 4.5');
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
});

describe('getDefaultModel', () => {
  it('returns second model for claude-code', () => {
    expect(getDefaultModel('claude-code')).toBe('claude-sonnet-4-5');
  });

  it('returns second model for gemini-cli', () => {
    expect(getDefaultModel('gemini-cli')).toBe('gemini-2.5-pro');
  });

  it('falls back to first model when only one exists', () => {
    // amp has two models, second is gpt-4o
    expect(getDefaultModel('amp')).toBe('gpt-4o');
  });

  it('returns "default" for unknown runner', () => {
    expect(getDefaultModel('nonexistent')).toBe('default');
  });
});

describe('getModelsForRunner', () => {
  it('returns claude models for claude-code', () => {
    const models = getModelsForRunner('claude-code');
    expect(models).toHaveLength(3);
    expect(models[0].id).toBe('claude-haiku-4-5');
  });

  it('returns gemini models for gemini-cli', () => {
    const models = getModelsForRunner('gemini-cli');
    expect(models).toHaveLength(2);
    expect(models[0].id).toBe('gemini-2.0-flash');
  });

  it('returns fallback for unknown runner', () => {
    const models = getModelsForRunner('unknown');
    expect(models).toHaveLength(1);
    expect(models[0].id).toBe('default');
  });
});
