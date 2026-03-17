import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useExplorerState } from '../use-explorer-state';

describe('useExplorerState', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('defaults to not visible with files tab', () => {
    const { result } = renderHook(() => useExplorerState());
    expect(result.current.visible).toBe(false);
    expect(result.current.activeTab).toBe('files');
  });

  it('open() makes the explorer visible', () => {
    const { result } = renderHook(() => useExplorerState());
    act(() => {
      result.current.open();
    });
    expect(result.current.visible).toBe(true);
  });

  it('open("context") makes visible and sets context tab', () => {
    const { result } = renderHook(() => useExplorerState());
    act(() => {
      result.current.open('context');
    });
    expect(result.current.visible).toBe(true);
    expect(result.current.activeTab).toBe('context');
  });

  it('open() without tab argument keeps the current tab', () => {
    const { result } = renderHook(() => useExplorerState());
    act(() => {
      result.current.open('context');
    });
    act(() => {
      result.current.close();
    });
    act(() => {
      result.current.open();
    });
    expect(result.current.visible).toBe(true);
    expect(result.current.activeTab).toBe('context');
  });

  it('close() hides the explorer', () => {
    const { result } = renderHook(() => useExplorerState());
    act(() => {
      result.current.open();
    });
    expect(result.current.visible).toBe(true);

    act(() => {
      result.current.close();
    });
    expect(result.current.visible).toBe(false);
  });

  it('toggle() toggles visibility', () => {
    const { result } = renderHook(() => useExplorerState());
    expect(result.current.visible).toBe(false);

    act(() => {
      result.current.toggle();
    });
    expect(result.current.visible).toBe(true);

    act(() => {
      result.current.toggle();
    });
    expect(result.current.visible).toBe(false);
  });

  it('persists visible state to localStorage', () => {
    const { result } = renderHook(() => useExplorerState());
    act(() => {
      result.current.open();
    });
    expect(localStorage.getItem('session-explorer-visible')).toBe('true');

    act(() => {
      result.current.close();
    });
    expect(localStorage.getItem('session-explorer-visible')).toBe('false');
  });

  it('persists active tab to localStorage', () => {
    const { result } = renderHook(() => useExplorerState());
    act(() => {
      result.current.setActiveTab('context');
    });
    expect(localStorage.getItem('session-explorer-tab')).toBe('"context"');
  });

  it('restores visible state from localStorage', () => {
    localStorage.setItem('session-explorer-visible', 'true');
    const { result } = renderHook(() => useExplorerState());
    expect(result.current.visible).toBe(true);
  });

  it('restores active tab from localStorage', () => {
    localStorage.setItem('session-explorer-tab', '"context"');
    const { result } = renderHook(() => useExplorerState());
    expect(result.current.activeTab).toBe('context');
  });

  it('defaults to files tab for invalid localStorage value', () => {
    localStorage.setItem('session-explorer-tab', 'invalid-json');
    const { result } = renderHook(() => useExplorerState());
    expect(result.current.activeTab).toBe('files');
  });
});
