import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import React from 'react';
import { WelcomeWizard } from '../welcome-wizard';

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useParams: () => ({}),
}));

vi.mock('@/services/queries', () => ({
  useCreateProject: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('@/services/queries/use-integrations', () => ({
  useIntegrationsStatus: () => ({
    data: {
      github: { installed: false, pat: { configured: false } },
      gitlab: { connected: false },
      google: { connected: false },
      jira: { connected: false },
      coderabbit: { connected: false },
      gerrit: { connected: false },
    },
    isLoading: false,
    refetch: vi.fn(),
  }),
}));

vi.mock('@/hooks/use-cluster-info', () => ({
  useClusterInfo: () => ({ isOpenShift: false, isLoading: false, vertexEnabled: false }),
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
  Wrapper.displayName = 'TestQueryWrapper';
  return Wrapper;
}

function renderWizard(onDismiss = vi.fn()) {
  const Wrapper = createWrapper();
  return {
    onDismiss,
    ...render(
      React.createElement(Wrapper, null,
        React.createElement(WelcomeWizard, { open: true, onDismiss })
      )
    ),
  };
}

describe('WelcomeWizard', () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  it('renders step 1 (Welcome) initially', () => {
    renderWizard();
    expect(screen.getByText('Welcome to Ambient Code Platform')).toBeTruthy();
    expect(screen.getByText('Get Started')).toBeTruthy();
  });

  it('advances to step 2 on Get Started click', () => {
    renderWizard();
    fireEvent.click(screen.getByText('Get Started'));
    expect(screen.getByText('Create your workspace')).toBeTruthy();
  });

  it('shows Skip setup link on step 2', () => {
    renderWizard();
    fireEvent.click(screen.getByText('Get Started'));
    expect(screen.getByText('Skip setup')).toBeTruthy();
  });

  it('calls onDismiss when Skip setup is clicked', () => {
    const onDismiss = vi.fn();
    renderWizard(onDismiss);
    fireEvent.click(screen.getByText('Get Started'));
    fireEvent.click(screen.getByText('Skip setup'));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it('does not show Skip setup on step 1', () => {
    renderWizard();
    expect(screen.queryByText('Skip setup')).toBeNull();
  });
});
