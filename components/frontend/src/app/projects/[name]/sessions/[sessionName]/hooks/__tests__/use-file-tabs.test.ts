import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useFileTabs } from '../use-file-tabs';

describe('useFileTabs', () => {
  it('starts with empty tabs and chat active', () => {
    const { result } = renderHook(() => useFileTabs());
    expect(result.current.openTabs).toEqual([]);
    expect(result.current.activeTab).toEqual({ type: 'chat' });
  });

  it('openFile adds a tab and activates it', () => {
    const { result } = renderHook(() => useFileTabs());
    act(() => {
      result.current.openFile({ path: '/src/index.ts', name: 'index.ts' });
    });
    expect(result.current.openTabs).toEqual([{ path: '/src/index.ts', name: 'index.ts' }]);
    expect(result.current.activeTab).toEqual({ type: 'file', path: '/src/index.ts' });
  });

  it('openFile does not duplicate an existing tab', () => {
    const { result } = renderHook(() => useFileTabs());
    act(() => {
      result.current.openFile({ path: '/src/index.ts', name: 'index.ts' });
    });
    act(() => {
      result.current.openFile({ path: '/src/index.ts', name: 'index.ts' });
    });
    expect(result.current.openTabs).toHaveLength(1);
  });

  it('openFile can add multiple different tabs', () => {
    const { result } = renderHook(() => useFileTabs());
    act(() => {
      result.current.openFile({ path: '/src/a.ts', name: 'a.ts' });
    });
    act(() => {
      result.current.openFile({ path: '/src/b.ts', name: 'b.ts' });
    });
    expect(result.current.openTabs).toHaveLength(2);
    expect(result.current.activeTab).toEqual({ type: 'file', path: '/src/b.ts' });
  });

  it('closeFile removes the tab and switches to chat if it was active', () => {
    const { result } = renderHook(() => useFileTabs());
    act(() => {
      result.current.openFile({ path: '/src/index.ts', name: 'index.ts' });
    });
    expect(result.current.activeTab).toEqual({ type: 'file', path: '/src/index.ts' });

    act(() => {
      result.current.closeFile('/src/index.ts');
    });
    expect(result.current.openTabs).toEqual([]);
    expect(result.current.activeTab).toEqual({ type: 'chat' });
  });

  it('closeFile keeps current active tab when closing a different one', () => {
    const { result } = renderHook(() => useFileTabs());
    act(() => {
      result.current.openFile({ path: '/src/a.ts', name: 'a.ts' });
    });
    act(() => {
      result.current.openFile({ path: '/src/b.ts', name: 'b.ts' });
    });
    // b.ts is active
    expect(result.current.activeTab).toEqual({ type: 'file', path: '/src/b.ts' });

    act(() => {
      result.current.closeFile('/src/a.ts');
    });
    expect(result.current.openTabs).toHaveLength(1);
    expect(result.current.activeTab).toEqual({ type: 'file', path: '/src/b.ts' });
  });

  it('switchToChat sets activeTab to chat', () => {
    const { result } = renderHook(() => useFileTabs());
    act(() => {
      result.current.openFile({ path: '/src/index.ts', name: 'index.ts' });
    });
    expect(result.current.activeTab).toEqual({ type: 'file', path: '/src/index.ts' });

    act(() => {
      result.current.switchToChat();
    });
    expect(result.current.activeTab).toEqual({ type: 'chat' });
  });

  it('switchToFile sets activeTab to the given file path', () => {
    const { result } = renderHook(() => useFileTabs());
    act(() => {
      result.current.openFile({ path: '/src/a.ts', name: 'a.ts' });
    });
    act(() => {
      result.current.openFile({ path: '/src/b.ts', name: 'b.ts' });
    });
    act(() => {
      result.current.switchToChat();
    });
    expect(result.current.activeTab).toEqual({ type: 'chat' });

    act(() => {
      result.current.switchToFile('/src/a.ts');
    });
    expect(result.current.activeTab).toEqual({ type: 'file', path: '/src/a.ts' });
  });
});
