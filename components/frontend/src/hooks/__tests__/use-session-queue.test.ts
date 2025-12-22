/**
 * Tests for useSessionQueue hook
 * 
 * These tests verify that the localStorage-backed queue functionality works correctly
 * for both messages and workflows.
 */

import { renderHook, act } from '@testing-library/react';
import { useSessionQueue } from '../use-session-queue';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};

  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(global, 'localStorage', {
  value: localStorageMock,
});

describe('useSessionQueue', () => {
  const projectName = 'test-project';
  const sessionName = 'test-session';

  beforeEach(() => {
    localStorageMock.clear();
  });

  describe('Message Queue Operations', () => {
    it('should add a message to the queue', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      act(() => {
        result.current.addMessage('Hello, world!');
      });

      expect(result.current.messages).toHaveLength(1);
      expect(result.current.messages[0].content).toBe('Hello, world!');
      expect(result.current.messages[0].sentAt).toBeUndefined();
    });

    it('should mark a message as sent', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      let messageId: string;

      act(() => {
        result.current.addMessage('Test message');
        messageId = result.current.messages[0].id;
      });

      act(() => {
        result.current.markMessageSent(messageId!);
      });

      expect(result.current.messages[0].sentAt).toBeDefined();
      expect(result.current.messages[0].sentAt).toBeGreaterThan(0);
    });

    it('should clear all messages', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      act(() => {
        result.current.addMessage('Message 1');
        result.current.addMessage('Message 2');
      });

      expect(result.current.messages).toHaveLength(2);

      act(() => {
        result.current.clearMessages();
      });

      expect(result.current.messages).toHaveLength(0);
      
      // Verify localStorage key is removed (not just empty array)
      const key = `vteam:queue:${projectName}:${sessionName}:messages`;
      const stored = localStorage.getItem(key);
      expect(stored).toBeNull();
    });

    it('should persist messages to localStorage', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      act(() => {
        result.current.addMessage('Persisted message');
      });

      // Check localStorage directly
      const key = `vteam:queue:${projectName}:${sessionName}:messages`;
      const stored = localStorage.getItem(key);
      expect(stored).toBeTruthy();
      
      const parsed = JSON.parse(stored!);
      expect(parsed).toHaveLength(1);
      expect(parsed[0].content).toBe('Persisted message');
    });

    it('should load messages from localStorage on mount', () => {
      // Pre-populate localStorage
      const key = `vteam:queue:${projectName}:${sessionName}:messages`;
      const messages = [
        { id: 'msg-1', content: 'Existing message', timestamp: Date.now() }
      ];
      localStorage.setItem(key, JSON.stringify(messages));

      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      expect(result.current.messages).toHaveLength(1);
      expect(result.current.messages[0].content).toBe('Existing message');
    });
  });

  describe('Workflow Queue Operations', () => {
    it('should set a workflow', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      const workflow = {
        id: 'workflow-1',
        gitUrl: 'https://github.com/test/repo.git',
        branch: 'main',
        path: 'workflows/test',
      };

      act(() => {
        result.current.setWorkflow(workflow);
      });

      expect(result.current.workflow).toBeDefined();
      expect(result.current.workflow?.id).toBe('workflow-1');
      expect(result.current.workflow?.gitUrl).toBe('https://github.com/test/repo.git');
    });

    it('should mark workflow as activated', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      const workflow = {
        id: 'workflow-1',
        gitUrl: 'https://github.com/test/repo.git',
        branch: 'main',
        path: 'workflows/test',
      };

      act(() => {
        result.current.setWorkflow(workflow);
      });

      act(() => {
        result.current.markWorkflowActivated('workflow-1');
      });

      expect(result.current.workflow?.activatedAt).toBeDefined();
      expect(result.current.workflow?.activatedAt).toBeGreaterThan(0);
    });

    it('should clear workflow', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      const workflow = {
        id: 'workflow-1',
        gitUrl: 'https://github.com/test/repo.git',
        branch: 'main',
        path: 'workflows/test',
      };

      act(() => {
        result.current.setWorkflow(workflow);
      });

      expect(result.current.workflow).toBeDefined();

      act(() => {
        result.current.clearWorkflow();
      });

      expect(result.current.workflow).toBeNull();
    });

    it('should persist workflow to localStorage', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      const workflow = {
        id: 'workflow-1',
        gitUrl: 'https://github.com/test/repo.git',
        branch: 'main',
        path: 'workflows/test',
      };

      act(() => {
        result.current.setWorkflow(workflow);
      });

      // Check localStorage directly
      const key = `vteam:queue:${projectName}:${sessionName}:workflow`;
      const stored = localStorage.getItem(key);
      expect(stored).toBeTruthy();
      
      const parsed = JSON.parse(stored!);
      expect(parsed.id).toBe('workflow-1');
      expect(parsed.gitUrl).toBe('https://github.com/test/repo.git');
    });
  });

  describe('Metadata Operations', () => {
    it('should update metadata', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      act(() => {
        result.current.updateMetadata({
          sessionPhase: 'Running',
          processing: true,
        });
      });

      expect(result.current.metadata.sessionPhase).toBe('Running');
      expect(result.current.metadata.processing).toBe(true);
    });
  });

  describe('Cleanup and Error Handling', () => {
    it('should filter out old messages (>24h)', () => {
      const key = `vteam:queue:${projectName}:${sessionName}:messages`;
      const oldTimestamp = Date.now() - (25 * 60 * 60 * 1000); // 25 hours ago
      const recentTimestamp = Date.now();
      
      const messages = [
        { id: 'msg-old', content: 'Old message', timestamp: oldTimestamp },
        { id: 'msg-new', content: 'Recent message', timestamp: recentTimestamp },
      ];
      localStorage.setItem(key, JSON.stringify(messages));

      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      // Should only have the recent message
      expect(result.current.messages).toHaveLength(1);
      expect(result.current.messages[0].id).toBe('msg-new');
    });

    it('should handle corrupted localStorage data gracefully', () => {
      const key = `vteam:queue:${projectName}:${sessionName}:messages`;
      localStorage.setItem(key, 'invalid json{{{');

      // Should not throw
      expect(() => {
        renderHook(() => useSessionQueue(projectName, sessionName));
      }).not.toThrow();
    });

    it('should limit messages to max count', () => {
      const { result } = renderHook(() => useSessionQueue(projectName, sessionName));

      // Add more than MAX_MESSAGES (100)
      act(() => {
        for (let i = 0; i < 105; i++) {
          result.current.addMessage(`Message ${i}`);
        }
      });

      // Should only keep last 100
      expect(result.current.messages.length).toBeLessThanOrEqual(100);
    });
  });
});

