import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { ReactNode } from 'react';
import { ApiSourceProvider, useApiSource } from './api-source-context';

function wrapper({ children }: { children: ReactNode }) {
  return <ApiSourceProvider>{children}</ApiSourceProvider>;
}

describe('ApiSourceContext', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('defaults to api-server when no localStorage value', () => {
    const { result } = renderHook(() => useApiSource(), { wrapper });
    expect(result.current.source).toBe('api-server');
    expect(result.current.isApiServer).toBe(true);
    expect(result.current.isK8s).toBe(false);
  });

  it('respects k8s value from localStorage', () => {
    localStorage.setItem('ambient-api-source', 'k8s');
    const { result } = renderHook(() => useApiSource(), { wrapper });
    expect(result.current.source).toBe('k8s');
    expect(result.current.isK8s).toBe(true);
  });

  it('toggles between sources', () => {
    const { result } = renderHook(() => useApiSource(), { wrapper });
    expect(result.current.source).toBe('api-server');

    act(() => result.current.toggle());
    expect(result.current.source).toBe('k8s');

    act(() => result.current.toggle());
    expect(result.current.source).toBe('api-server');
  });

  it('persists source to localStorage', () => {
    const { result } = renderHook(() => useApiSource(), { wrapper });

    act(() => result.current.setSource('k8s'));
    expect(localStorage.getItem('ambient-api-source')).toBe('k8s');

    act(() => result.current.setSource('api-server'));
    expect(localStorage.getItem('ambient-api-source')).toBe('api-server');
  });

  it('respects defaultSource prop', () => {
    function k8sWrapper({ children }: { children: ReactNode }) {
      return <ApiSourceProvider defaultSource="k8s">{children}</ApiSourceProvider>;
    }
    const { result } = renderHook(() => useApiSource(), { wrapper: k8sWrapper });
    expect(result.current.source).toBe('k8s');
  });

  it('throws when used outside provider', () => {
    expect(() => {
      renderHook(() => useApiSource());
    }).toThrow('useApiSource must be used within an ApiSourceProvider');
  });
});
