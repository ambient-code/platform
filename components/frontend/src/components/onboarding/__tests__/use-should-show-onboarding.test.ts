import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import React from 'react';
import { useShouldShowOnboarding, ONBOARDING_FLAG } from '../use-should-show-onboarding';

vi.mock('@/services/api/projects', () => ({
  listProjectsPaginated: vi.fn(),
}));

import * as projectsApi from '@/services/api/projects';
const mockListPaginated = vi.mocked(projectsApi.listProjectsPaginated);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
  Wrapper.displayName = 'TestQueryWrapper';
  return Wrapper;
}

describe('useShouldShowOnboarding', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const emptyResponse = { items: [], totalCount: 0, hasMore: false, limit: 1, offset: 0 };

  const projectResponse = {
    items: [{ name: 'proj', displayName: 'Proj', labels: {}, annotations: {}, creationTimestamp: '', status: 'active' as const, isOpenShift: false }],
    totalCount: 1,
    hasMore: false,
    limit: 1,
    offset: 0,
  };

  it('shows wizard when zero projects and no localStorage flag', async () => {
    mockListPaginated.mockResolvedValue(emptyResponse);
    const wrapper = createWrapper();
    const { result } = renderHook(() => useShouldShowOnboarding(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.shouldShow).toBe(true);
  });

  it('hides wizard when zero projects but localStorage flag is set', async () => {
    localStorage.setItem(ONBOARDING_FLAG, 'true');
    mockListPaginated.mockResolvedValue(emptyResponse);
    const wrapper = createWrapper();
    const { result } = renderHook(() => useShouldShowOnboarding(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.shouldShow).toBe(false);
  });

  it('hides wizard when user has projects', async () => {
    mockListPaginated.mockResolvedValue(projectResponse);
    const wrapper = createWrapper();
    const { result } = renderHook(() => useShouldShowOnboarding(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.shouldShow).toBe(false);
  });

  it('hides wizard when user has projects even without localStorage flag', async () => {
    mockListPaginated.mockResolvedValue(projectResponse);
    const wrapper = createWrapper();
    const { result } = renderHook(() => useShouldShowOnboarding(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.shouldShow).toBe(false);
  });

  it('dismiss sets the localStorage flag', async () => {
    mockListPaginated.mockResolvedValue(emptyResponse);
    const wrapper = createWrapper();
    const { result } = renderHook(() => useShouldShowOnboarding(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    result.current.dismiss();
    expect(localStorage.getItem(ONBOARDING_FLAG)).toBe('true');
  });
});
