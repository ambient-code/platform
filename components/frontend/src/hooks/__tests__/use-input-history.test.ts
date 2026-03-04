import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useInputHistory } from '../use-input-history';

// Mock useLocalStorage
vi.mock('../use-local-storage', () => {
  let storage: string[] = [];
  return {
    useLocalStorage: vi.fn((_key: string, initial: string[]) => {
      // Use a simple in-memory implementation
      if (storage === undefined) storage = initial;
      const setState = (fn: string[] | ((prev: string[]) => string[])) => {
        storage = typeof fn === 'function' ? fn(storage) : fn;
      };
      return [storage, setState, () => { storage = initial; }];
    }),
  };
});

import { useLocalStorage } from '../use-local-storage';

describe('useInputHistory', () => {
  beforeEach(() => {
    // Reset mock storage
    vi.mocked(useLocalStorage).mockImplementation((_key, initial) => {
      let storage = initial as string[];
      const setState = vi.fn((fn: string[] | ((prev: string[]) => string[])) => {
        storage = typeof fn === 'function' ? fn(storage) : fn;
      });
      return [storage, setState, vi.fn()];
    });
  });

  it('returns empty history initially', () => {
    const { result } = renderHook(() => useInputHistory('test-key'));
    expect(result.current.history).toEqual([]);
  });

  it('provides addToHistory function', () => {
    const { result } = renderHook(() => useInputHistory('test-key'));
    expect(typeof result.current.addToHistory).toBe('function');
  });

  it('calls useLocalStorage with correct key prefix', () => {
    renderHook(() => useInputHistory('my-form'));
    expect(useLocalStorage).toHaveBeenCalledWith('form-input-history:my-form', []);
  });

  it('addToHistory calls setHistory with trimmed value at front', () => {
    let capturedSetter: ((fn: string[] | ((prev: string[]) => string[])) => void) | null = null;
    vi.mocked(useLocalStorage).mockImplementation((_key, initial) => {
      const setter = vi.fn();
      capturedSetter = setter;
      return [initial as string[], setter, vi.fn()];
    });

    const { result } = renderHook(() => useInputHistory('test-key'));
    act(() => {
      result.current.addToHistory('  hello world  ');
    });

    expect(capturedSetter).toHaveBeenCalled();
    // Get the updater function and apply it
    const updaterFn = vi.mocked(capturedSetter!).mock.calls[0][0] as (prev: string[]) => string[];
    const newHistory = updaterFn([]);
    expect(newHistory).toEqual(['hello world']);
  });

  it('addToHistory deduplicates existing entries', () => {
    let capturedSetter: ((fn: string[] | ((prev: string[]) => string[])) => void) | null = null;
    vi.mocked(useLocalStorage).mockImplementation((_key, initial) => {
      const setter = vi.fn();
      capturedSetter = setter;
      return [initial as string[], setter, vi.fn()];
    });

    const { result } = renderHook(() => useInputHistory('test-key'));
    act(() => {
      result.current.addToHistory('hello');
    });

    const updaterFn = vi.mocked(capturedSetter!).mock.calls[0][0] as (prev: string[]) => string[];
    const newHistory = updaterFn(['old', 'hello', 'other']);
    expect(newHistory).toEqual(['hello', 'old', 'other']);
  });

  it('addToHistory ignores empty/whitespace-only input', () => {
    let capturedSetter: ((fn: string[] | ((prev: string[]) => string[])) => void) | null = null;
    vi.mocked(useLocalStorage).mockImplementation((_key, initial) => {
      const setter = vi.fn();
      capturedSetter = setter;
      return [initial as string[], setter, vi.fn()];
    });

    const { result } = renderHook(() => useInputHistory('test-key'));
    act(() => {
      result.current.addToHistory('   ');
    });

    expect(capturedSetter).not.toHaveBeenCalled();
  });

  it('addToHistory respects maxItems', () => {
    let capturedSetter: ((fn: string[] | ((prev: string[]) => string[])) => void) | null = null;
    vi.mocked(useLocalStorage).mockImplementation((_key, initial) => {
      const setter = vi.fn();
      capturedSetter = setter;
      return [initial as string[], setter, vi.fn()];
    });

    const { result } = renderHook(() => useInputHistory('test-key', 3));
    act(() => {
      result.current.addToHistory('new');
    });

    const updaterFn = vi.mocked(capturedSetter!).mock.calls[0][0] as (prev: string[]) => string[];
    const newHistory = updaterFn(['a', 'b', 'c']);
    expect(newHistory).toEqual(['new', 'a', 'b']);
    expect(newHistory).toHaveLength(3);
  });
});
