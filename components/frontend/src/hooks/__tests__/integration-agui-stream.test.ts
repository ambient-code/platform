import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useAGUIStream } from '../use-agui-stream';
import { createSessionEventsAdapter } from '@/services/adapters/session-events';
import type { SessionEventsPort } from '@/services/ports/session-events';

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

  simulateMessage(data: unknown) {
    this.onmessage?.(new MessageEvent('message', { data: JSON.stringify(data) }));
  }
}

function createFakeApi() {
  return {
    createEventSource: vi.fn(() => {
      return new MockEventSource('/fake') as unknown as EventSource;
    }),
    sendMessage: vi.fn().mockResolvedValue({ runId: 'new-run' }),
    interrupt: vi.fn().mockResolvedValue(undefined),
  };
}

const defaultOptions = { projectName: 'proj', sessionName: 'sess' };

describe('integration: useAGUIStream → real adapter → real processAGUIEvent', () => {
  let fakeApi: ReturnType<typeof createFakeApi>;
  let adapter: SessionEventsPort;

  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    fakeApi = createFakeApi();
    adapter = createSessionEventsAdapter(fakeApi);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('RUN_STARTED sets runId, threadId, status, and isRunActive', () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    act(() => { result.current.connect(); });

    expect(fakeApi.createEventSource).toHaveBeenCalledWith('proj', 'sess', undefined);

    act(() => { MockEventSource.instances[0].simulateOpen(); });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_STARTED', runId: 'r1', threadId: 't1',
      });
    });

    expect(result.current.state.runId).toBe('r1');
    expect(result.current.state.threadId).toBe('t1');
    expect(result.current.state.status).toBe('connected');
    expect(result.current.isRunActive).toBe(true);
  });

  it('connect(runId) forwards runId to createEventSource', () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    act(() => { result.current.connect('existing-run-42'); });

    expect(fakeApi.createEventSource).toHaveBeenCalledWith('proj', 'sess', 'existing-run-42');
  });

  it('autoConnect with runId forwards runId to createEventSource', () => {
    renderHook(() => useAGUIStream({ ...defaultOptions, autoConnect: true, runId: 'initial-run' }, adapter));

    expect(fakeApi.createEventSource).toHaveBeenCalledWith('proj', 'sess', 'initial-run');
  });

  it('text message lifecycle accumulates content into state.messages', () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    act(() => { result.current.connect(); });
    act(() => { MockEventSource.instances[0].simulateOpen(); });

    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_STARTED', runId: 'r1', threadId: 't1',
      });
    });

    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_START', messageId: 'msg-1', role: 'assistant',
      });
    });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_CONTENT', messageId: 'msg-1', delta: 'Hello ',
      });
    });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_CONTENT', messageId: 'msg-1', delta: 'world',
      });
    });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_END', messageId: 'msg-1',
      });
    });

    const assistantMsgs = result.current.state.messages.filter(m => m.role === 'assistant');
    expect(assistantMsgs).toHaveLength(1);
    expect(assistantMsgs[0].content).toBe('Hello world');
    expect(assistantMsgs[0].id).toBe('msg-1');
  });

  it('RUN_FINISHED sets status to completed and clears isRunActive', () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    act(() => { result.current.connect(); });
    act(() => { MockEventSource.instances[0].simulateOpen(); });

    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_STARTED', runId: 'r1', threadId: 't1',
      });
    });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_START', messageId: 'msg-1', role: 'assistant',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_CONTENT', messageId: 'msg-1', delta: 'done',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_END', messageId: 'msg-1',
      });
    });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_FINISHED', runId: 'r1', threadId: 't1',
      });
    });

    expect(result.current.state.status).toBe('completed');
    expect(result.current.isRunActive).toBe(false);
    expect(result.current.state.messages.filter(m => m.role === 'assistant')).toHaveLength(1);
  });

  it('RUN_ERROR sets error state and adds error message to messages', () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useAGUIStream({ ...defaultOptions, onError }, adapter));

    act(() => { result.current.connect(); });
    act(() => { MockEventSource.instances[0].simulateOpen(); });

    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_STARTED', runId: 'r1', threadId: 't1',
      });
    });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_ERROR', runId: 'r1', message: 'Something broke',
      });
    });

    expect(result.current.state.status).toBe('error');
    expect(result.current.state.error).toBe('Something broke');
    expect(result.current.isRunActive).toBe(false);
    expect(onError).toHaveBeenCalledWith('Something broke');

    const errorMsg = result.current.state.messages.find(m =>
      typeof m.content === 'string' && m.content.includes('Something broke'),
    );
    expect(errorMsg).toBeDefined();
  });

  it('tool call lifecycle attaches toolCalls to assistant message', () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    act(() => { result.current.connect(); });
    act(() => { MockEventSource.instances[0].simulateOpen(); });

    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_STARTED', runId: 'r1', threadId: 't1',
      });
    });

    // Assistant message first
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_START', messageId: 'msg-1', role: 'assistant',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_CONTENT', messageId: 'msg-1', delta: 'Let me check.',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_END', messageId: 'msg-1',
      });
    });

    // Tool call
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TOOL_CALL_START', toolCallId: 'tc-1', toolCallName: 'read_file', parentMessageId: 'msg-1',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TOOL_CALL_ARGS', toolCallId: 'tc-1', delta: '{"path":"/tmp/f"}',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TOOL_CALL_END', toolCallId: 'tc-1',
      });
    });

    const assistantMsg = result.current.state.messages.find(m => m.id === 'msg-1');
    expect(assistantMsg?.toolCalls).toBeDefined();
    expect(assistantMsg!.toolCalls).toHaveLength(1);
    expect(assistantMsg!.toolCalls![0].function.name).toBe('read_file');
    expect(assistantMsg!.toolCalls![0].function.arguments).toBe('{"path":"/tmp/f"}');
    expect(assistantMsg!.toolCalls![0].status).toBe('completed');
  });

  it('sendMessage adds user message to state, calls port.sendMessage, and sets isRunActive from response', async () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    await act(async () => {
      await result.current.sendMessage('Hello Claude');
    });

    expect(result.current.state.messages).toHaveLength(1);
    expect(result.current.state.messages[0].role).toBe('user');
    expect(result.current.state.messages[0].content).toBe('Hello Claude');

    expect(fakeApi.sendMessage).toHaveBeenCalledWith('proj', 'sess', expect.objectContaining({
      messages: [expect.objectContaining({ content: 'Hello Claude', role: 'user' })],
    }));

    expect(result.current.isRunActive).toBe(true);
  });

  it('sendMessage response runId is used by subsequent interrupt', async () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    await act(async () => {
      await result.current.sendMessage('Hello');
    });

    expect(result.current.isRunActive).toBe(true);

    await act(async () => {
      await result.current.interrupt();
    });

    expect(fakeApi.interrupt).toHaveBeenCalledWith('proj', 'sess', 'new-run');
    expect(result.current.isRunActive).toBe(false);
  });

  it('interrupt calls port.interrupt with the current runId', async () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    act(() => { result.current.connect(); });
    act(() => { MockEventSource.instances[0].simulateOpen(); });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_STARTED', runId: 'r1', threadId: 't1',
      });
    });

    await act(async () => {
      await result.current.interrupt();
    });

    expect(fakeApi.interrupt).toHaveBeenCalledWith('proj', 'sess', 'r1');
    expect(result.current.isRunActive).toBe(false);
  });

  it('multiple text messages accumulate in order', () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    act(() => { result.current.connect(); });
    act(() => { MockEventSource.instances[0].simulateOpen(); });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_STARTED', runId: 'r1', threadId: 't1',
      });
    });

    // First message
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_START', messageId: 'msg-1', role: 'assistant',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_CONTENT', messageId: 'msg-1', delta: 'First',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_END', messageId: 'msg-1',
      });
    });

    // Second message
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_START', messageId: 'msg-2', role: 'assistant',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_CONTENT', messageId: 'msg-2', delta: 'Second',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_END', messageId: 'msg-2',
      });
    });

    const assistantMsgs = result.current.state.messages.filter(m => m.role === 'assistant');
    expect(assistantMsgs).toHaveLength(2);
    expect(assistantMsgs[0].content).toBe('First');
    expect(assistantMsgs[1].content).toBe('Second');
  });

  it('onMessage callback receives the real processed message', () => {
    const onMessage = vi.fn();
    const { result } = renderHook(() => useAGUIStream({ ...defaultOptions, onMessage }, adapter));

    act(() => { result.current.connect(); });
    act(() => { MockEventSource.instances[0].simulateOpen(); });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_STARTED', runId: 'r1', threadId: 't1',
      });
    });

    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_START', messageId: 'msg-1', role: 'assistant',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_CONTENT', messageId: 'msg-1', delta: 'Hi there',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_END', messageId: 'msg-1',
      });
    });

    expect(onMessage).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'msg-1', role: 'assistant', content: 'Hi there' }),
    );
  });

  it('RUN_ERROR flushes partial text message content', () => {
    const { result } = renderHook(() => useAGUIStream(defaultOptions, adapter));

    vi.spyOn(console, 'error').mockImplementation(() => {});

    act(() => { result.current.connect(); });
    act(() => { MockEventSource.instances[0].simulateOpen(); });
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_STARTED', runId: 'r1', threadId: 't1',
      });
    });

    // Start text but don't finish
    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_START', messageId: 'msg-1', role: 'assistant',
      });
      MockEventSource.instances[0].simulateMessage({
        type: 'TEXT_MESSAGE_CONTENT', messageId: 'msg-1', delta: 'Partial content',
      });
    });

    act(() => {
      MockEventSource.instances[0].simulateMessage({
        type: 'RUN_ERROR', runId: 'r1', message: 'Crash',
      });
    });

    expect(result.current.state.status).toBe('error');

    const partialMsg = result.current.state.messages.find(m =>
      typeof m.content === 'string' && m.content === 'Partial content',
    );
    expect(partialMsg).toBeDefined();

    const errorMsg = result.current.state.messages.find(m =>
      typeof m.content === 'string' && m.content.includes('Crash'),
    );
    expect(errorMsg).toBeDefined();

    vi.mocked(console.error).mockRestore();
  });
});
