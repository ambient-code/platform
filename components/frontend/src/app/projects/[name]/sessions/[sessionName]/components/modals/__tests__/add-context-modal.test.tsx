import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { AddContextModal } from '../add-context-modal';

const HISTORY_STORAGE_KEY = 'form-input-history:add-context:url';

describe('AddContextModal', () => {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
    onAddRepository: vi.fn().mockResolvedValue(undefined),
    isLoading: false,
    autoBranch: 'session/auto-branch',
  };

  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('shows local history suggestions from storage inside the dialog', async () => {
    localStorage.setItem(
      HISTORY_STORAGE_KEY,
      JSON.stringify([
        'https://github.com/acme/alpha',
        'https://github.com/acme/beta',
      ])
    );

    render(<AddContextModal {...defaultProps} />);

    const urlInput = screen.getByLabelText('Repository URL');
    fireEvent.focus(urlInput);

    await waitFor(() => {
      expect(screen.getByText('https://github.com/acme/alpha')).toBeDefined();
      expect(screen.getByText('https://github.com/acme/beta')).toBeDefined();
    });
  });

  it('selects the first stored history item with ArrowDown and submits it', async () => {
    localStorage.setItem(
      HISTORY_STORAGE_KEY,
      JSON.stringify([
        'https://github.com/acme/alpha/',
        'https://github.com/acme/beta',
      ])
    );

    render(<AddContextModal {...defaultProps} />);

    const urlInput = screen.getByLabelText('Repository URL');
    fireEvent.focus(urlInput);

    await waitFor(() => {
      expect(screen.getByText('https://github.com/acme/alpha/')).toBeDefined();
    });

    fireEvent.keyDown(urlInput, { key: 'ArrowDown' });
    fireEvent.keyDown(urlInput, { key: 'Enter' });
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));

    await waitFor(() => {
      expect(defaultProps.onAddRepository).toHaveBeenCalledWith(
        'https://github.com/acme/alpha',
        'session/auto-branch',
        false
      );
    });
  });

  it('selects the last stored history item with ArrowUp and submits it', async () => {
    localStorage.setItem(
      HISTORY_STORAGE_KEY,
      JSON.stringify([
        'https://github.com/acme/alpha',
        'https://github.com/acme/beta',
      ])
    );

    render(<AddContextModal {...defaultProps} />);

    const urlInput = screen.getByLabelText('Repository URL');
    fireEvent.focus(urlInput);

    await waitFor(() => {
      expect(screen.getByText('https://github.com/acme/alpha')).toBeDefined();
    });

    fireEvent.keyDown(urlInput, { key: 'Escape' });
    fireEvent.keyDown(urlInput, { key: 'ArrowUp' });
    fireEvent.keyDown(urlInput, { key: 'Enter' });
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));

    await waitFor(() => {
      expect(defaultProps.onAddRepository).toHaveBeenCalledWith(
        'https://github.com/acme/beta',
        'session/auto-branch',
        false
      );
    });
  });
});
