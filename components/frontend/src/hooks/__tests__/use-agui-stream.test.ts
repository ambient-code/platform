import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useAGUIStream, initialState } from '../use-agui-stream';
import type { SessionEventsPort } from '@/services/ports/session-events';

// Mock the event-handlers module
vi.mock('../agui/event-handlers', () => ({
  processAGUIEvent: vi.fn((prev) => prev),
}));

// EventSource mock used by the fake port
class MockEventSource {
  static instances: MockEventSource[] = [];

  url: string;
  onopen: ((e: Event) => void) | null = null;
  onmessage: ((e: MessageEvent) => void) | null = null;
  onerror: ((e: Event) => void) | null = null;
  readyState = 0;
  close = vi.fn(() => { this.readyState = 2; });

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  simulateOpen() {
    this.readyState = 1;
    this.onopen?.(new Event('open'));
  }

  simulateMessage(data: string) {
    this.onmessage?.(new MessageEvent('message', { data }));
  }

  simulateError() {
    this.onerror?.(new Event('error'));
  }
}

function createFakePort(): SessionEventsPort {
  return {
    createEventSource: vi.fn((projectName, sessionName, runId) => {
      let url = `/api/projects/${projectName}/agentic-sessions/${sessionName}/agui/events`;
      if (runId) url += `?runId=${runId}`;
      return new MockEventSource(url) as unknown as EventSource;
    }),
    sendMessage: vi.fn().mockResolvedValue({ runId: 'new-run' }),
    interrupt: vi.fn().mockResolvedValue(undefined),
  };
}

describe('useAGUIStream', () => {
  let fakePort: ReturnType<typeof createFakePort>;

  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    fakePort = createFakePort();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  const defaultOptions = {
    projectName: 'proj',
    sessionName: 'sess',
  };

  it('initializes with idle state', () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

    expect(result.current.state.status).toBe('idle');
    expect(result.current.isConnected).toBe(false);
    expect(result.current.isStreaming).toBe(false);
    expect(result.current.isRunActive).toBe(false);
  });

  it('exports initialState', () => {
    expect(initialState.status).toBe('idle');
    expect(initialState.messages).toEqual([]);
  });

  describe('connect', () => {
    it('creates EventSource with correct URL', () => {
      const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

      act(() => {
        result.current.connect();
      });

      expect(fakePort.createEventSource).toHaveBeenCalledWith('proj', 'sess', undefined);
      expect(MockEventSource.instances).toHaveLength(1);
      expect(MockEventSource.instances[0].url).toBe('/api/projects/proj/agentic-sessions/sess/agui/events');
    });

    it('includes runId in URL when provided', () => {
      const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

      act(() => {
        result.current.connect('run-123');
      });

      expect(fakePort.createEventSource).toHaveBeenCalledWith('proj', 'sess', 'run-123');
      expect(MockEventSource.instances[0].url).toContain('?runId=run-123');
    });

    it('sets status to connecting then connected on open', () => {
      const onConnected = vi.fn();
      const { result } = renderHook(() => useAGUIStream({ ...defaultOptions, onConnected }, fakePort));

      act(() => {
        result.current.connect();
      });

      expect(result.current.state.status).toBe('connecting');

      act(() => {
        MockEventSource.instances[0].simulateOpen();
      });

      expect(result.current.state.status).toBe('connected');
      expect(result.current.isConnected).toBe(true);
      expect(onConnected).toHaveBeenCalled();
    });

    it('processes incoming events', () => {
      const onEvent = vi.fn();
      const { result } = renderHook(() => useAGUIStream({ ...defaultOptions, onEvent }, fakePort));

      act(() => {
        result.current.connect();
        MockEventSource.instances[0].simulateOpen();
      });

      act(() => {
        MockEventSource.instances[0].simulateMessage(JSON.stringify({ type: 'RUN_STARTED', runId: 'r1' }));
      });

      expect(onEvent).toHaveBeenCalledWith({ type: 'RUN_STARTED', runId: 'r1' });
    });

    it('handles malformed JSON gracefully', () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

      act(() => {
        result.current.connect();
        MockEventSource.instances[0].simulateOpen();
      });

      act(() => {
        MockEventSource.instances[0].simulateMessage('invalid json');
      });

      expect(consoleSpy).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });

    it('closes previous connection when connecting again', () => {
      const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

      act(() => {
        result.current.connect();
      });

      const firstInstance = MockEventSource.instances[0];

      act(() => {
        result.current.connect();
      });

      expect(firstInstance.close).toHaveBeenCalled();
    });
  });

  describe('disconnect', () => {
    it('closes EventSource and resets state', () => {
      const onDisconnected = vi.fn();
      const { result } = renderHook(() => useAGUIStream({ ...defaultOptions, onDisconnected }, fakePort));

      act(() => {
        result.current.connect();
        MockEventSource.instances[0].simulateOpen();
      });

      act(() => {
        result.current.disconnect();
      });

      expect(MockEventSource.instances[0].close).toHaveBeenCalled();
      expect(result.current.state.status).toBe('idle');
      expect(result.current.isRunActive).toBe(false);
      expect(onDisconnected).toHaveBeenCalled();
    });
  });

  describe('error handling and reconnect', () => {
    it('sets error state on connection error', () => {
      const onError = vi.fn();
      const onDisconnected = vi.fn();
      const { result } = renderHook(() => useAGUIStream({ ...defaultOptions, onError, onDisconnected }, fakePort));

      act(() => {
        result.current.connect();
        MockEventSource.instances[0].simulateOpen();
      });

      act(() => {
        MockEventSource.instances[0].simulateError();
      });

      expect(result.current.state.status).toBe('error');
      expect(result.current.state.error).toBe('Connection error');
      expect(onError).toHaveBeenCalledWith('Connection error');
      expect(onDisconnected).toHaveBeenCalled();
    });

    it('schedules reconnect with exponential backoff', () => {
      const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

      act(() => {
        result.current.connect();
        MockEventSource.instances[0].simulateOpen();
      });

      // First error - should schedule reconnect at 1000ms
      act(() => {
        MockEventSource.instances[0].simulateError();
      });

      expect(MockEventSource.instances).toHaveLength(1);

      // Advance timer to trigger reconnect
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      expect(MockEventSource.instances).toHaveLength(2);
    });
  });

  describe('sendMessage', () => {
    it('sends message and adds to state', async () => {
      const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

      await act(async () => {
        await result.current.sendMessage('Hello Claude');
      });

      expect(fakePort.sendMessage).toHaveBeenCalledWith('proj', 'sess', expect.objectContaining({
        messages: [expect.objectContaining({ content: 'Hello Claude', role: 'user' })],
      }));
      expect(result.current.state.messages).toHaveLength(1);
      expect(result.current.state.messages[0].content).toBe('Hello Claude');
      expect(result.current.state.messages[0].role).toBe('user');
    });

    it('handles send error', async () => {
      (fakePort.sendMessage as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new Error('Server error'));

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

      let error: Error | undefined;
      await act(async () => {
        try {
          await result.current.sendMessage('Hello');
        } catch (e) {
          error = e as Error;
        }
      });

      expect(error?.message).toContain('Server error');
      expect(result.current.state.status).toBe('error');
      consoleSpy.mockRestore();
    });
  });

  describe('interrupt', () => {
    it('sends interrupt request', async () => {
      const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

      // First send a message to establish a run
      await act(async () => {
        await result.current.sendMessage('Hello');
      });

      await act(async () => {
        await result.current.interrupt();
      });

      expect(fakePort.interrupt).toHaveBeenCalledWith('proj', 'sess', 'new-run');
      expect(result.current.isRunActive).toBe(false);
    });

    it('warns when no active run to interrupt', async () => {
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
      const { result } = renderHook(() => useAGUIStream(defaultOptions, fakePort));

      await act(async () => {
        await result.current.interrupt();
      });

      expect(consoleSpy).toHaveBeenCalledWith('[useAGUIStream] No active run to interrupt');
      consoleSpy.mockRestore();
    });
  });

  describe('autoConnect', () => {
    it('auto-connects when autoConnect is true', () => {
      renderHook(() => useAGUIStream({ ...defaultOptions, autoConnect: true }, fakePort));
      expect(MockEventSource.instances).toHaveLength(1);
    });

    it('does not auto-connect when autoConnect is false', () => {
      renderHook(() => useAGUIStream({ ...defaultOptions, autoConnect: false }, fakePort));
      expect(MockEventSource.instances).toHaveLength(0);
    });
  });
});
