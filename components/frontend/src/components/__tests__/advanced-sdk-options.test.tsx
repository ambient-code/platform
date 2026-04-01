import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { AdvancedSdkOptions } from '../advanced-sdk-options';
import type { SdkOptions } from '@/types/api/sessions';

describe('AdvancedSdkOptions', () => {
  const defaultProps = {
    value: {} as SdkOptions,
    onChange: vi.fn(),
    models: [
      { id: 'claude-sonnet-4-5', name: 'Claude Sonnet 4.5' },
      { id: 'claude-opus-4-5', name: 'Claude Opus 4.5' },
    ],
  };

  it('renders the collapsible trigger button', () => {
    render(<AdvancedSdkOptions {...defaultProps} />);
    expect(screen.getByText('Advanced SDK Options')).toBeDefined();
  });

  it('is collapsed by default', () => {
    render(<AdvancedSdkOptions {...defaultProps} />);
    expect(screen.queryByText('Model & Generation')).toBeNull();
  });

  it('expands when trigger is clicked', () => {
    render(<AdvancedSdkOptions {...defaultProps} />);
    fireEvent.click(screen.getByText('Advanced SDK Options'));
    expect(screen.getByText('Model & Generation')).toBeDefined();
    expect(screen.getByText('Execution & Control')).toBeDefined();
    expect(screen.getByText('Allowed Tools')).toBeDefined();
    expect(screen.getByText('System Prompt')).toBeDefined();
    expect(screen.getByText('Beta Feature Flags')).toBeDefined();
  });

  it('calls onChange when temperature is set', () => {
    render(<AdvancedSdkOptions {...defaultProps} />);
    fireEvent.click(screen.getByText('Advanced SDK Options'));
    const tempInput = screen.getByLabelText('Temperature');
    fireEvent.change(tempInput, { target: { value: '0.5' } });
    expect(defaultProps.onChange).toHaveBeenCalledWith(
      expect.objectContaining({ temperature: 0.5 })
    );
  });

  it('shows JSON preview when toggled', () => {
    const props = {
      ...defaultProps,
      value: { temperature: 0.5, max_tokens: 8000 } as SdkOptions,
    };
    render(<AdvancedSdkOptions {...props} />);
    fireEvent.click(screen.getByText('Advanced SDK Options'));
    fireEvent.click(screen.getByText('Show JSON Preview'));
    expect(screen.getByText(/"temperature": 0.5/)).toBeDefined();
    expect(screen.getByText(/"max_tokens": 8000/)).toBeDefined();
  });

  it('renders tool toggles', () => {
    render(<AdvancedSdkOptions {...defaultProps} />);
    fireEvent.click(screen.getByText('Advanced SDK Options'));
    expect(screen.getByText('Read')).toBeDefined();
    expect(screen.getByText('Write')).toBeDefined();
    expect(screen.getByText('Bash')).toBeDefined();
  });
});
