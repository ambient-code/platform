import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { LiveModelSelector } from '../live-model-selector';
import type { ListModelsResponse } from '@/types/api';

const mockAnthropicModels: ListModelsResponse = {
  models: [
    { id: 'claude-haiku-4-5', label: 'Claude Haiku 4.5', provider: 'anthropic', isDefault: false },
    { id: 'claude-sonnet-4-5', label: 'Claude Sonnet 4.5', provider: 'anthropic', isDefault: true },
    { id: 'claude-opus-4-6', label: 'Claude Opus 4.6', provider: 'anthropic', isDefault: false },
  ],
  defaultModel: 'claude-sonnet-4-5',
};

const mockUseModels = vi.fn(() => ({ data: mockAnthropicModels }));

vi.mock('@/services/queries/use-models', () => ({
  useModels: () => mockUseModels(),
}));

describe('LiveModelSelector', () => {
  const defaultProps = {
    projectName: 'test-project',
    currentModel: 'claude-sonnet-4-5',
    onSelect: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseModels.mockReturnValue({ data: mockAnthropicModels });
  });

  it('renders with current model name displayed', () => {
    render(<LiveModelSelector {...defaultProps} />);
    const button = screen.getByRole('button');
    expect(button.textContent).toContain('Claude Sonnet 4.5');
  });

  it('renders with model id fallback when model not in list', () => {
    render(
      <LiveModelSelector
        {...defaultProps}
        currentModel="unknown-model-id"
      />
    );
    const button = screen.getByRole('button');
    expect(button.textContent).toContain('unknown-model-id');
  });

  it('shows spinner when switching', () => {
    render(<LiveModelSelector {...defaultProps} switching />);
    const spinner = document.querySelector('.animate-spin');
    expect(spinner).not.toBeNull();
  });

  it('button is disabled when disabled prop is true', () => {
    render(<LiveModelSelector {...defaultProps} disabled />);
    const button = screen.getByRole('button');
    expect((button as HTMLButtonElement).disabled).toBe(true);
  });

  it('button is disabled when switching prop is true', () => {
    render(<LiveModelSelector {...defaultProps} switching />);
    const button = screen.getByRole('button');
    expect((button as HTMLButtonElement).disabled).toBe(true);
  });
});
