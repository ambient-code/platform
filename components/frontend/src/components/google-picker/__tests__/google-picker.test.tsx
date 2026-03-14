import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { GooglePicker } from '../google-picker';

vi.mock('@/lib/google-picker-loader', () => ({
  loadPickerApi: vi.fn().mockResolvedValue(undefined),
}));

vi.mock('@/services/drive-api', () => ({
  usePickerToken: vi.fn().mockReturnValue({
    refetch: vi.fn().mockResolvedValue({
      data: { accessToken: 'test-token' },
    }),
  }),
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  }
  return Wrapper;
}

const defaultProps = {
  projectName: 'my-project',
  apiKey: 'test-api-key',
  appId: 'test-app-id',
  onFilesPicked: vi.fn(),
};

describe('GooglePicker', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the button with correct label', () => {
    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <GooglePicker {...defaultProps} />
      </Wrapper>,
    );
    expect(screen.getByText('Choose Files')).toBeDefined();
  });

  it('renders custom button label', () => {
    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <GooglePicker {...defaultProps} buttonLabel="Select Documents" />
      </Wrapper>,
    );
    expect(screen.getByText('Select Documents')).toBeDefined();
  });

  it('button is disabled when disabled prop is true', () => {
    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <GooglePicker {...defaultProps} disabled={true} />
      </Wrapper>,
    );
    const button = screen.getByRole('button');
    expect(button).toBeDefined();
    expect(button.hasAttribute('disabled')).toBe(true);
  });

  it('button is not disabled when disabled prop is false', () => {
    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <GooglePicker {...defaultProps} disabled={false} />
      </Wrapper>,
    );
    const button = screen.getByRole('button');
    expect(button.hasAttribute('disabled')).toBe(false);
  });

  it('shows loading state when picker is being opened', async () => {
    // Make loadPickerApi hang so we can observe loading state
    const { loadPickerApi } = await import('@/lib/google-picker-loader');
    vi.mocked(loadPickerApi).mockReturnValue(new Promise(() => {}));

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <GooglePicker {...defaultProps} />
      </Wrapper>,
    );

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText('Opening file picker...')).toBeDefined();
    });
  });
});
