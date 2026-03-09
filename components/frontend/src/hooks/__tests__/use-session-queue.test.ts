import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useSessionQueue } from '../use-session-queue';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => { store[key] = value; }),
    removeItem: vi.fn((key: string) => { delete store[key]; }),
    clear: vi.fn(() => { store = {}; }),
    get length() { return Object.keys(store).length; },
    key: vi.fn((i: number) => Object.keys(store)[i] ?? null),
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });

describe('useSessionQueue', () => {
  beforeEach(() => {
    localStorageMock.clear();
    vi.clearAllMocks();
  });

  describe('message queue operations', () => {
    it('starts with empty messages', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));
      expect(result.current.messages).toEqual([]);
      expect(result.current.pendingCount).toBe(0);
    });

    it('adds a message to the queue', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.addMessage('Hello world');
      });

      expect(result.current.messages).toHaveLength(1);
      expect(result.current.messages[0].content).toBe('Hello world');
      expect(result.current.messages[0].sentAt).toBeUndefined();
      expect(result.current.pendingCount).toBe(1);
    });

    it('adds multiple messages', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.addMessage('Message 1');
        result.current.addMessage('Message 2');
        result.current.addMessage('Message 3');
      });

      expect(result.current.messages).toHaveLength(3);
      expect(result.current.pendingCount).toBe(3);
    });

    it('marks a message as sent', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.addMessage('Hello');
      });

      const messageId = result.current.messages[0].id;

      act(() => {
        result.current.markMessageSent(messageId);
      });

      expect(result.current.messages[0].sentAt).toBeDefined();
      expect(result.current.pendingCount).toBe(0);
    });

    it('cancels a message', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.addMessage('To cancel');
        result.current.addMessage('To keep');
      });

      const cancelId = result.current.messages[0].id;

      act(() => {
        result.current.cancelMessage(cancelId);
      });

      expect(result.current.messages).toHaveLength(1);
      expect(result.current.messages[0].content).toBe('To keep');
    });

    it('updates a message', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.addMessage('Original');
      });

      const messageId = result.current.messages[0].id;

      act(() => {
        result.current.updateMessage(messageId, 'Updated');
      });

      expect(result.current.messages[0].content).toBe('Updated');
      expect(result.current.messages[0].id).toBe(messageId);
    });

    it('clears all messages', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.addMessage('Msg 1');
        result.current.addMessage('Msg 2');
      });

      act(() => {
        result.current.clearMessages();
      });

      expect(result.current.messages).toEqual([]);
      expect(result.current.pendingCount).toBe(0);
    });

    it('counts only unsent messages as pending', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.addMessage('Msg 1');
        result.current.addMessage('Msg 2');
        result.current.addMessage('Msg 3');
      });

      const firstId = result.current.messages[0].id;

      act(() => {
        result.current.markMessageSent(firstId);
      });

      expect(result.current.pendingCount).toBe(2);
    });
  });

  describe('workflow queue operations', () => {
    it('starts with null workflow', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));
      expect(result.current.workflow).toBeNull();
    });

    it('sets a workflow', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.setWorkflow({
          id: 'wf-1',
          gitUrl: 'https://github.com/org/repo',
          branch: 'main',
          path: '/workflows/test.yml',
        });
      });

      expect(result.current.workflow).not.toBeNull();
      expect(result.current.workflow!.id).toBe('wf-1');
      expect(result.current.workflow!.gitUrl).toBe('https://github.com/org/repo');
      expect(result.current.workflow!.timestamp).toBeDefined();
    });

    it('marks workflow as activated', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.setWorkflow({
          id: 'wf-1',
          gitUrl: 'https://github.com/org/repo',
          branch: 'main',
          path: '/workflows/test.yml',
        });
      });

      act(() => {
        result.current.markWorkflowActivated('wf-1');
      });

      expect(result.current.workflow!.activatedAt).toBeDefined();
    });

    it('does not activate workflow with wrong id', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.setWorkflow({
          id: 'wf-1',
          gitUrl: 'https://github.com/org/repo',
          branch: 'main',
          path: '/workflows/test.yml',
        });
      });

      act(() => {
        result.current.markWorkflowActivated('wrong-id');
      });

      expect(result.current.workflow!.activatedAt).toBeUndefined();
    });

    it('clears workflow', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.setWorkflow({
          id: 'wf-1',
          gitUrl: 'https://github.com/org/repo',
          branch: 'main',
          path: '/workflows/test.yml',
        });
      });

      act(() => {
        result.current.clearWorkflow();
      });

      expect(result.current.workflow).toBeNull();
    });
  });

  describe('metadata operations', () => {
    it('starts with empty metadata', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));
      expect(result.current.metadata).toEqual({});
    });

    it('updates metadata', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.updateMetadata({ sessionPhase: 'Running', processing: true });
      });

      expect(result.current.metadata.sessionPhase).toBe('Running');
      expect(result.current.metadata.processing).toBe(true);
    });

    it('merges metadata updates', () => {
      const { result } = renderHook(() => useSessionQueue('proj', 'sess'));

      act(() => {
        result.current.updateMetadata({ sessionPhase: 'Pending' });
      });

      act(() => {
        result.current.updateMetadata({ processing: true });
      });

      expect(result.current.metadata.sessionPhase).toBe('Pending');
      expect(result.current.metadata.processing).toBe(true);
    });
  });
});
