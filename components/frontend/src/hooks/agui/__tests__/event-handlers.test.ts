import { describe, it, expect, vi, beforeEach } from 'vitest';
import { processAGUIEvent, type EventHandlerCallbacks } from '../event-handlers';
import { EventType } from '@/types/agui';
import type { AGUIClientState, PlatformMessage, PlatformToolCall } from '@/types/agui';

function makeState(overrides: Partial<AGUIClientState> = {}): AGUIClientState {
  return {
    threadId: null,
    runId: null,
    status: 'idle',
    messages: [],
    state: {},
    activities: [],
    currentMessage: null,
    currentToolCall: null,
    pendingToolCalls: new Map(),
    pendingChildren: new Map(),
    error: null,
    messageFeedback: new Map(),
    currentReasoning: null,
    currentThinking: null,
    backgroundTasks: new Map(),
    ...overrides,
  };
}

function makeCallbacks(overrides: Partial<EventHandlerCallbacks> = {}): EventHandlerCallbacks {
  return {
    setIsRunActive: vi.fn(),
    currentRunIdRef: { current: null },
    hiddenMessageIdsRef: { current: new Set() },
    onMessage: vi.fn(),
    onError: vi.fn(),
    onTraceId: vi.fn(),
    ...overrides,
  };
}

function msg(overrides: Partial<PlatformMessage> & { role: string }): PlatformMessage {
  return { id: crypto.randomUUID(), content: '', ...overrides } as PlatformMessage;
}

function toolCall(id: string, name: string, args = ''): PlatformToolCall {
  return {
    id,
    type: 'function',
    function: { name, arguments: args },
    status: 'completed',
  } as PlatformToolCall;
}

describe('processAGUIEvent', () => {
  let callbacks: EventHandlerCallbacks;

  beforeEach(() => {
    callbacks = makeCallbacks();
  });

  // ── RUN_STARTED ──
  describe('handleRunStarted', () => {
    it('sets threadId, runId, status and marks run active', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: EventType.RUN_STARTED,
        threadId: 'thread-1',
        runId: 'run-1',
      } as never, callbacks);

      expect(result.threadId).toBe('thread-1');
      expect(result.runId).toBe('run-1');
      expect(result.status).toBe('connected');
      expect(result.error).toBeNull();
      expect(callbacks.currentRunIdRef.current).toBe('run-1');
      expect(callbacks.setIsRunActive).toHaveBeenCalledWith(true);
    });
  });

  // ── RUN_ERROR ──
  describe('handleRunError', () => {
    it('sets error status and surfaces error as chat message', () => {
      const state = makeState({ status: 'connected' });
      const result = processAGUIEvent(state, {
        type: EventType.RUN_ERROR,
        message: 'Something went wrong',
      } as never, callbacks);

      expect(result.status).toBe('error');
      expect(result.error).toBe('Something went wrong');
      expect(callbacks.onError).toHaveBeenCalledWith('Something went wrong');
      expect(callbacks.setIsRunActive).toHaveBeenCalledWith(false);
      // Error message surfaced inline
      const errorMsg = result.messages.find(m =>
        typeof m.content === 'string' && m.content.includes('Something went wrong')
      );
      expect(errorMsg).toBeDefined();
    });

    it('marks incomplete tool calls as errored', () => {
      const state = makeState({
        messages: [
          msg({
            role: 'assistant',
            toolCalls: [
              { ...toolCall('tc1', 'Read'), status: 'completed' },
              { ...toolCall('tc2', 'Bash'), status: 'running' as never },
            ],
          }),
        ],
      });

      const result = processAGUIEvent(state, {
        type: EventType.RUN_ERROR,
        message: 'timeout',
      } as never, callbacks);

      const assistantMsg = result.messages.find(m => m.role === 'assistant');
      expect(assistantMsg?.toolCalls?.[0].status).toBe('completed');
      expect(assistantMsg?.toolCalls?.[1].status).toBe('error');
      expect(assistantMsg?.toolCalls?.[1].error).toBe('timeout');
    });

    it('flushes pending message content on error', () => {
      const state = makeState({
        currentMessage: { id: 'msg-1', role: 'assistant', content: 'partial output' },
      });

      const result = processAGUIEvent(state, {
        type: EventType.RUN_ERROR,
        message: 'crash',
      } as never, callbacks);

      expect(result.currentMessage).toBeNull();
      const flushed = result.messages.find(m =>
        typeof m.content === 'string' && m.content === 'partial output'
      );
      expect(flushed).toBeDefined();
    });

    it('drains pending tool calls on error', () => {
      const pending = new Map();
      pending.set('tc-1', { id: 'tc-1', name: 'Bash', args: '{}' });
      const state = makeState({ pendingToolCalls: pending });

      const result = processAGUIEvent(state, {
        type: EventType.RUN_ERROR,
        message: 'crash',
      } as never, callbacks);

      expect(result.pendingToolCalls.size).toBe(0);
      expect(result.currentToolCall).toBeNull();
    });
  });

  // ── RUN_FINISHED ──
  describe('handleRunFinished', () => {
    it('sets completed status and deactivates run', () => {
      callbacks.currentRunIdRef.current = 'run-1';
      const state = makeState({ status: 'connected', runId: 'run-1' });

      const result = processAGUIEvent(state, {
        type: EventType.RUN_FINISHED,
        runId: 'run-1',
      } as never, callbacks);

      expect(result.status).toBe('completed');
      expect(callbacks.setIsRunActive).toHaveBeenCalledWith(false);
      expect(callbacks.currentRunIdRef.current).toBeNull();
    });

    it('flushes pending message', () => {
      callbacks.currentRunIdRef.current = 'run-1';
      const state = makeState({
        currentMessage: { id: 'msg-1', role: 'assistant', content: 'final output' },
      });

      const result = processAGUIEvent(state, {
        type: EventType.RUN_FINISHED,
        runId: 'run-1',
      } as never, callbacks);

      expect(result.currentMessage).toBeNull();
      const flushed = result.messages.find(m =>
        typeof m.content === 'string' && m.content === 'final output'
      );
      expect(flushed).toBeDefined();
      expect(callbacks.onMessage).toHaveBeenCalled();
    });

    it('flushes pending reasoning', () => {
      callbacks.currentRunIdRef.current = 'run-1';
      const state = makeState({
        currentReasoning: { id: 'r-1', content: 'thinking hard' },
      });

      const result = processAGUIEvent(state, {
        type: EventType.RUN_FINISHED,
        runId: 'run-1',
      } as never, callbacks);

      expect(result.currentReasoning).toBeNull();
      const reasoningMsg = result.messages.find(m =>
        typeof m.content === 'object' && m.content !== null && 'thinking' in m.content
      );
      expect(reasoningMsg).toBeDefined();
    });

    it('does not deactivate if runId does not match', () => {
      callbacks.currentRunIdRef.current = 'run-other';
      const state = makeState();

      processAGUIEvent(state, {
        type: EventType.RUN_FINISHED,
        runId: 'run-1',
      } as never, callbacks);

      expect(callbacks.setIsRunActive).not.toHaveBeenCalled();
      expect(callbacks.currentRunIdRef.current).toBe('run-other');
    });
  });

  // ── TEXT_MESSAGE lifecycle ──
  describe('text message lifecycle', () => {
    it('accumulates content across START/CONTENT/END', () => {
      let state = makeState();

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_START,
        messageId: 'msg-1',
        role: 'assistant',
      } as never, callbacks);

      expect(state.currentMessage).not.toBeNull();

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_CONTENT,
        delta: 'Hello ',
      } as never, callbacks);

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_CONTENT,
        delta: 'world',
      } as never, callbacks);

      expect(state.currentMessage?.content).toBe('Hello world');

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_END,
      } as never, callbacks);

      expect(state.currentMessage).toBeNull();
      expect(state.messages).toHaveLength(1);
      expect(state.messages[0].content).toBe('Hello world');
      expect(state.messages[0].role).toBe('assistant');
    });

    it('skips hidden messages on TEXT_MESSAGE_END', () => {
      callbacks.hiddenMessageIdsRef.current.add('hidden-msg');
      let state = makeState();

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_START,
        messageId: 'hidden-msg',
        role: 'user',
      } as never, callbacks);

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_CONTENT,
        delta: 'prompt',
      } as never, callbacks);

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_END,
      } as never, callbacks);

      expect(state.messages).toHaveLength(0);
      expect(state.currentMessage).toBeNull();
    });

    it('updates existing message on duplicate messageId', () => {
      const existing = msg({ role: 'assistant', content: 'old content' });
      let state = makeState({ messages: [existing] });

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_START,
        messageId: existing.id,
        role: 'assistant',
      } as never, callbacks);

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_CONTENT,
        delta: 'new content',
      } as never, callbacks);

      state = processAGUIEvent(state, {
        type: EventType.TEXT_MESSAGE_END,
      } as never, callbacks);

      expect(state.messages).toHaveLength(1);
      expect(state.messages[0].content).toBe('new content');
    });
  });

  // ── TOOL_CALL lifecycle ──
  describe('tool call lifecycle', () => {
    it('tracks tool call through START/ARGS/END', () => {
      let state = makeState({
        messages: [msg({ role: 'assistant', content: 'let me check' })],
      });

      state = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_START,
        toolCallId: 'tc-1',
        toolCallName: 'Read',
      } as never, callbacks);

      expect(state.pendingToolCalls.has('tc-1')).toBe(true);
      expect(state.currentToolCall?.id).toBe('tc-1');

      state = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_ARGS,
        toolCallId: 'tc-1',
        delta: '{"path":',
      } as never, callbacks);

      state = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_ARGS,
        toolCallId: 'tc-1',
        delta: '"/foo"}',
      } as never, callbacks);

      expect(state.pendingToolCalls.get('tc-1')?.args).toBe('{"path":"/foo"}');

      state = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_END,
        toolCallId: 'tc-1',
      } as never, callbacks);

      expect(state.pendingToolCalls.has('tc-1')).toBe(false);
      expect(state.currentToolCall).toBeNull();
      // Tool call should be attached to the assistant message
      const assistantMsg = state.messages.find(m => m.role === 'assistant');
      expect(assistantMsg?.toolCalls).toHaveLength(1);
      expect(assistantMsg?.toolCalls?.[0].function.name).toBe('Read');
    });
  });

  // ── TOOL_CALL_RESULT ──
  describe('handleToolCallResult', () => {
    it('attaches result to committed tool call', () => {
      const state = makeState({
        messages: [
          msg({
            role: 'assistant',
            toolCalls: [toolCall('tc-1', 'Read')],
          }),
        ],
      });

      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_RESULT,
        toolCallId: 'tc-1',
        content: 'file contents',
      } as never, callbacks);

      expect(result.messages[0].toolCalls?.[0].result).toBe('file contents');
    });

    it('returns state unchanged if toolCallId is missing', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_RESULT,
        content: 'data',
      } as never, callbacks);

      expect(result).toEqual(state);
    });
  });

  // ── STATE_DELTA ──
  describe('handleStateDelta', () => {
    it('applies add/replace/remove JSON Patch operations', () => {
      const state = makeState({ state: { foo: 'bar', toRemove: true } });

      const result = processAGUIEvent(state, {
        type: EventType.STATE_DELTA,
        delta: [
          { op: 'add', path: '/newKey', value: 42 },
          { op: 'replace', path: '/foo', value: 'baz' },
          { op: 'remove', path: '/toRemove' },
        ],
      } as never, callbacks);

      expect(result.state).toEqual({ foo: 'baz', newKey: 42 });
    });
  });

  // ── MESSAGES_SNAPSHOT ──
  describe('handleMessagesSnapshot', () => {
    it('merges snapshot messages with existing', () => {
      const existingMsg = msg({ role: 'user', content: 'hello' });
      const state = makeState({ messages: [existingMsg] });

      const snapshotMsg = msg({ role: 'assistant', content: 'hi there' });

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [
          { ...existingMsg },
          snapshotMsg,
        ],
      } as never, callbacks);

      expect(result.messages).toHaveLength(2);
    });

    it('filters hidden messages from snapshot', () => {
      const hiddenMsg = msg({ role: 'user', content: 'auto prompt' });
      callbacks.hiddenMessageIdsRef.current.add(hiddenMsg.id);

      const state = makeState();
      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [hiddenMsg],
      } as never, callbacks);

      expect(result.messages).toHaveLength(0);
    });

    it('sorts messages by timestamp', () => {
      const state = makeState();
      const msg1 = msg({ role: 'user', content: 'first', timestamp: '2025-01-01T00:00:01Z' });
      const msg2 = msg({ role: 'assistant', content: 'second', timestamp: '2025-01-01T00:00:02Z' });

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [msg2, msg1], // out of order
      } as never, callbacks);

      expect(result.messages[0].content).toBe('first');
      expect(result.messages[1].content).toBe('second');
    });

    it('merges tool call data from streaming into snapshot', () => {
      const assistantMsg = msg({
        role: 'assistant',
        content: '',
        toolCalls: [
          { ...toolCall('tc-1', 'Read'), function: { name: 'Read', arguments: '{"p":"a"}' } },
        ],
      });
      const state = makeState({ messages: [assistantMsg] });

      // Snapshot arrives with generic 'tool' name
      const snapshotAssistant = {
        ...assistantMsg,
        toolCalls: [
          { ...toolCall('tc-1', 'tool'), function: { name: 'tool', arguments: '' } },
        ],
      };

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [snapshotAssistant],
      } as never, callbacks);

      // Should preserve the specific tool name from streaming
      const tc = result.messages.find(m => m.role === 'assistant')?.toolCalls?.[0];
      expect(tc?.function.name).toBe('Read');
    });
  });

  // ── ACTIVITY_SNAPSHOT ──
  describe('handleActivitySnapshot', () => {
    it('replaces activities array', () => {
      const state = makeState();
      const activities = [
        { id: 'a1', type: 'task', title: 'Doing stuff', status: 'running' as const },
      ];

      const result = processAGUIEvent(state, {
        type: EventType.ACTIVITY_SNAPSHOT,
        activities,
      } as never, callbacks);

      expect(result.activities).toEqual(activities);
    });
  });

  // ── ACTIVITY_DELTA ──
  describe('handleActivityDelta', () => {
    it('adds new activity', () => {
      const state = makeState({ activities: [] });
      const result = processAGUIEvent(state, {
        type: EventType.ACTIVITY_DELTA,
        delta: [{ op: 'add', activity: { id: 'a1', type: 'task', title: 'New' } }],
      } as never, callbacks);

      expect(result.activities).toHaveLength(1);
      expect(result.activities[0].id).toBe('a1');
    });

    it('updates existing activity', () => {
      const state = makeState({
        activities: [{ id: 'a1', type: 'task', title: 'Old', status: 'running' }],
      });

      const result = processAGUIEvent(state, {
        type: EventType.ACTIVITY_DELTA,
        delta: [{ op: 'update', activity: { id: 'a1', type: 'task', title: 'Updated', status: 'completed' } }],
      } as never, callbacks);

      expect(result.activities[0].title).toBe('Updated');
      expect(result.activities[0].status).toBe('completed');
    });

    it('removes activity', () => {
      const state = makeState({
        activities: [{ id: 'a1', type: 'task' }, { id: 'a2', type: 'task' }],
      });

      const result = processAGUIEvent(state, {
        type: EventType.ACTIVITY_DELTA,
        delta: [{ op: 'remove', activity: { id: 'a1', type: 'task' } }],
      } as never, callbacks);

      expect(result.activities).toHaveLength(1);
      expect(result.activities[0].id).toBe('a2');
    });

    it('returns state unchanged when delta is missing', () => {
      const state = makeState({ activities: [{ id: 'a1', type: 'task' }] });
      const result = processAGUIEvent(state, {
        type: EventType.ACTIVITY_DELTA,
      } as never, callbacks);

      expect(result.activities).toHaveLength(1);
    });
  });

  // ── REASONING_MESSAGE lifecycle ──
  describe('reasoning message lifecycle', () => {
    it('accumulates reasoning across START/CONTENT/END', () => {
      let state = makeState();

      state = processAGUIEvent(state, {
        type: EventType.REASONING_MESSAGE_START,
        messageId: 'r-1',
      } as never, callbacks);

      expect(state.currentReasoning).not.toBeNull();

      state = processAGUIEvent(state, {
        type: EventType.REASONING_MESSAGE_CONTENT,
        delta: 'Let me think...',
      } as never, callbacks);

      expect(state.currentReasoning?.content).toBe('Let me think...');

      state = processAGUIEvent(state, {
        type: EventType.REASONING_MESSAGE_END,
      } as never, callbacks);

      expect(state.currentReasoning).toBeNull();
      expect(state.messages).toHaveLength(1);
      const reasoningMsg = state.messages[0];
      expect(typeof reasoningMsg.content).toBe('object');
      expect((reasoningMsg.content as { thinking: string }).thinking).toBe('Let me think...');
    });

    it('does not emit message if reasoning is empty', () => {
      let state = makeState();

      state = processAGUIEvent(state, {
        type: EventType.REASONING_MESSAGE_START,
        messageId: 'r-1',
      } as never, callbacks);

      state = processAGUIEvent(state, {
        type: EventType.REASONING_MESSAGE_END,
      } as never, callbacks);

      expect(state.messages).toHaveLength(0);
    });
  });

  // ── RAW event ──
  describe('handleRawEvent', () => {
    it('hides message when message_metadata with hidden=true', () => {
      const existingMsg = msg({ role: 'user', content: 'auto prompt' });
      const state = makeState({ messages: [existingMsg] });

      const result = processAGUIEvent(state, {
        type: EventType.RAW,
        event: { type: 'message_metadata', hidden: true, messageId: existingMsg.id },
      } as never, callbacks);

      expect(result.messages).toHaveLength(0);
      expect(callbacks.hiddenMessageIdsRef.current.has(existingMsg.id)).toBe(true);
    });

    it('emits traceId callback on langfuse_trace', () => {
      const state = makeState();
      processAGUIEvent(state, {
        type: EventType.RAW,
        event: { type: 'langfuse_trace', traceId: 'trace-abc' },
      } as never, callbacks);

      expect(callbacks.onTraceId).toHaveBeenCalledWith('trace-abc');
    });

    it('adds reasoning block from RAW event', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: EventType.RAW,
        event: { type: 'reasoning_block', thinking: 'deep thought', signature: 'sig' },
      } as never, callbacks);

      expect(result.messages).toHaveLength(1);
      expect((result.messages[0].content as { thinking: string }).thinking).toBe('deep thought');
    });

    it('adds user message echo from RAW event', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: EventType.RAW,
        event: { role: 'user', content: 'hello from backend', id: 'echo-1' },
      } as never, callbacks);

      expect(result.messages).toHaveLength(1);
      expect(result.messages[0].role).toBe('user');
      expect(result.messages[0].content).toBe('hello from backend');
    });

    it('skips duplicate user message echo', () => {
      const existingMsg = msg({ role: 'user', content: 'hello' });
      existingMsg.id = 'echo-1';
      const state = makeState({ messages: [existingMsg] });

      const result = processAGUIEvent(state, {
        type: EventType.RAW,
        event: { role: 'user', content: 'hello', id: 'echo-1' },
      } as never, callbacks);

      expect(result.messages).toHaveLength(1);
    });

    it('reads from data field when event field is absent', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: EventType.RAW,
        data: { role: 'assistant', content: 'via data field' },
      } as never, callbacks);

      expect(result.messages).toHaveLength(1);
      expect(result.messages[0].content).toBe('via data field');
    });
  });

  // ── META event ──
  describe('handleMetaEvent', () => {
    it('stores thumbs_up feedback', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: 'META',
        metaType: 'thumbs_up',
        payload: { messageId: 'msg-1' },
        threadId: 't-1',
      } as never, callbacks);

      expect(result.messageFeedback.get('msg-1')).toBe('thumbs_up');
    });

    it('stores thumbs_down feedback', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: 'META',
        metaType: 'thumbs_down',
        payload: { messageId: 'msg-2' },
        threadId: 't-1',
      } as never, callbacks);

      expect(result.messageFeedback.get('msg-2')).toBe('thumbs_down');
    });

    it('ignores meta events without messageId', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: 'META',
        metaType: 'thumbs_up',
        payload: {},
        threadId: 't-1',
      } as never, callbacks);

      expect(result.messageFeedback.size).toBe(0);
    });

    it('ignores unknown metaType', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: 'META',
        metaType: 'something_else',
        payload: { messageId: 'msg-1' },
        threadId: 't-1',
      } as never, callbacks);

      expect(result.messageFeedback.size).toBe(0);
    });
  });

  // ── STEP lifecycle ──
  describe('step lifecycle', () => {
    it('sets currentStep on STEP_STARTED', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: EventType.STEP_STARTED,
        stepName: 'analyze',
      } as never, callbacks);

      expect(result.state.currentStep).toEqual({
        id: 'analyze',
        name: 'analyze',
        status: 'running',
      });
    });

    it('clears currentStep on STEP_FINISHED', () => {
      const state = makeState({ state: { currentStep: { id: 'x', name: 'x', status: 'running' } } });
      const result = processAGUIEvent(state, {
        type: EventType.STEP_FINISHED,
      } as never, callbacks);

      expect(result.state.currentStep).toBeUndefined();
    });
  });

  // ── STATE_SNAPSHOT ──
  describe('handleStateSnapshot', () => {
    it('replaces state with snapshot', () => {
      const state = makeState({ state: { old: true } });
      const result = processAGUIEvent(state, {
        type: EventType.STATE_SNAPSHOT,
        snapshot: { new: 'data', count: 42 },
      } as never, callbacks);

      expect(result.state).toEqual({ new: 'data', count: 42 });
    });
  });

  // ── REASONING_START / REASONING_END (no-ops) ──
  describe('reasoning lifecycle bookends', () => {
    it('REASONING_START is a no-op', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: EventType.REASONING_START,
      } as never, callbacks);
      expect(result).toEqual(state);
    });

    it('REASONING_END is a no-op', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: EventType.REASONING_END,
      } as never, callbacks);
      expect(result).toEqual(state);
    });
  });

  // ── handleToolCallEnd: parent tool hierarchy ──
  describe('handleToolCallEnd — parent tool hierarchy', () => {
    it('skips tool that already exists in messages (duplicate guard)', () => {
      const state = makeState({
        messages: [
          msg({
            role: 'assistant',
            toolCalls: [toolCall('tc-1', 'Read')],
          }),
        ],
        pendingToolCalls: new Map([['tc-1', { id: 'tc-1', name: 'Read', args: '' }]]),
      });

      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_END,
        toolCallId: 'tc-1',
      } as never, callbacks);

      // Should have cleaned up pending but NOT duplicated tool call
      expect(result.pendingToolCalls.has('tc-1')).toBe(false);
      expect(result.messages[0].toolCalls).toHaveLength(1);
    });

    it('stores child tool in pendingChildren when parent is still pending', () => {
      const pending = new Map();
      pending.set('parent-tc', { id: 'parent-tc', name: 'Agent', args: '' });
      pending.set('child-tc', { id: 'child-tc', name: 'Read', args: '{}', parentToolUseId: 'parent-tc' });

      const state = makeState({
        messages: [msg({ role: 'assistant', content: 'working' })],
        pendingToolCalls: pending,
        pendingChildren: new Map(),
      });

      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_END,
        toolCallId: 'child-tc',
      } as never, callbacks);

      // Child should be stored in pendingChildren under parent-tc
      expect(result.pendingChildren.has('parent-tc')).toBe(true);
      expect(result.pendingChildren.get('parent-tc')).toHaveLength(1);
      expect(result.pendingChildren.get('parent-tc')![0].toolCalls![0].id).toBe('child-tc');
    });

    it('attaches child tool to parent tool in committed messages', () => {
      const state = makeState({
        messages: [
          msg({
            role: 'assistant',
            toolCalls: [toolCall('parent-tc', 'Agent')],
          }),
        ],
        pendingToolCalls: new Map([
          ['child-tc', { id: 'child-tc', name: 'Read', args: '{}', parentToolUseId: 'parent-tc' }],
        ]),
        pendingChildren: new Map(),
      });

      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_END,
        toolCallId: 'child-tc',
      } as never, callbacks);

      // Child should be nested under parent in the assistant message
      const assistantMsg = result.messages.find(m => m.role === 'assistant');
      expect(assistantMsg?.toolCalls).toHaveLength(2);
      expect(assistantMsg?.toolCalls?.[1].id).toBe('child-tc');
    });

    it('creates standalone tool message when no assistant message found', () => {
      const state = makeState({
        messages: [msg({ role: 'user', content: 'hi' })],
        pendingToolCalls: new Map([
          ['tc-orphan', { id: 'tc-orphan', name: 'Bash', args: '{}', timestamp: '2025-01-01T00:00:00.000Z' }],
        ]),
        pendingChildren: new Map(),
      });

      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_END,
        toolCallId: 'tc-orphan',
      } as never, callbacks);

      const toolMsg = result.messages.find(m => m.role === 'tool');
      expect(toolMsg).toBeDefined();
      expect(toolMsg?.toolCalls?.[0].id).toBe('tc-orphan');
      expect(callbacks.onMessage).toHaveBeenCalled();
    });

    it('does not duplicate child when parent already has it', () => {
      const state = makeState({
        messages: [
          msg({
            role: 'assistant',
            toolCalls: [
              toolCall('parent-tc', 'Agent'),
              toolCall('child-tc', 'Read'),
            ],
          }),
        ],
        pendingToolCalls: new Map([
          ['child-tc', { id: 'child-tc', name: 'Read', args: '{}', parentToolUseId: 'parent-tc' }],
        ]),
        pendingChildren: new Map(),
      });

      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_END,
        toolCallId: 'child-tc',
      } as never, callbacks);

      const assistantMsg = result.messages.find(m => m.role === 'assistant');
      // Should still be exactly 2, not 3
      expect(assistantMsg?.toolCalls).toHaveLength(2);
    });

    it('drains pendingChildren when attaching tool to assistant message', () => {
      const pendingChildMsg = {
        id: 'pc-1',
        role: 'tool',
        toolCallId: 'grandchild-tc',
        content: '',
        toolCalls: [toolCall('grandchild-tc', 'Grep')],
      } as PlatformMessage;

      const state = makeState({
        messages: [msg({ role: 'assistant', content: 'thinking' })],
        pendingToolCalls: new Map([
          ['tc-main', { id: 'tc-main', name: 'Agent', args: '{}' }],
        ]),
        pendingChildren: new Map([['tc-main', [pendingChildMsg]]]),
      });

      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_END,
        toolCallId: 'tc-main',
      } as never, callbacks);

      // pendingChildren for tc-main should have been drained
      expect(result.pendingChildren.has('tc-main')).toBe(false);
      // The assistant message should have both tc-main and grandchild-tc
      const assistantMsg = result.messages.find(m => m.role === 'assistant');
      expect(assistantMsg?.toolCalls?.some(tc => tc.id === 'tc-main')).toBe(true);
      expect(assistantMsg?.toolCalls?.some(tc => tc.id === 'grandchild-tc')).toBe(true);
    });
  });

  // ── handleToolCallResult: pendingChildren ──
  describe('handleToolCallResult — pendingChildren', () => {
    it('attaches result to tool call in pendingChildren', () => {
      const childMsg = {
        id: 'pc-1',
        role: 'tool',
        toolCallId: 'child-tc',
        content: '',
        toolCalls: [toolCall('child-tc', 'Read')],
      } as PlatformMessage;

      const state = makeState({
        pendingChildren: new Map([['parent-tc', [childMsg]]]),
      });

      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_RESULT,
        toolCallId: 'child-tc',
        content: 'file contents from child',
      } as never, callbacks);

      const updatedChild = result.pendingChildren.get('parent-tc')![0];
      expect(updatedChild.toolCalls![0].result).toBe('file contents from child');
      expect(updatedChild.toolCalls![0].status).toBe('completed');
    });

    it('handles result for tool not found anywhere gracefully', () => {
      const state = makeState({
        messages: [msg({ role: 'assistant', content: 'test' })],
      });

      const result = processAGUIEvent(state, {
        type: EventType.TOOL_CALL_RESULT,
        toolCallId: 'nonexistent-tc',
        content: 'orphaned result',
      } as never, callbacks);

      // Should not crash, messages unchanged
      expect(result.messages).toHaveLength(1);
    });
  });

  // ── handleMessagesSnapshot: tool call merge & tool name recovery ──
  describe('handleMessagesSnapshot — advanced merge', () => {
    it('preserves parentToolUseId from streaming during merge', () => {
      const assistantMsg = msg({
        role: 'assistant',
        content: '',
        toolCalls: [
          { ...toolCall('tc-1', 'Read'), parentToolUseId: 'parent-tc' } as PlatformToolCall,
        ],
      });
      const state = makeState({ messages: [assistantMsg] });

      const snapshotAssistant = {
        ...assistantMsg,
        toolCalls: [
          { ...toolCall('tc-1', 'Read'), parentToolUseId: undefined } as PlatformToolCall,
        ],
      };

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [snapshotAssistant],
      } as never, callbacks);

      const tc = result.messages.find(m => m.role === 'assistant')?.toolCalls?.[0];
      expect(tc?.parentToolUseId).toBe('parent-tc');
    });

    it('adds streaming-only tool calls not present in snapshot', () => {
      const assistantMsg = msg({
        role: 'assistant',
        content: '',
        toolCalls: [
          toolCall('tc-1', 'Read'),
          toolCall('tc-2', 'Bash'),
        ],
      });
      const state = makeState({ messages: [assistantMsg] });

      const snapshotAssistant = {
        ...assistantMsg,
        toolCalls: [toolCall('tc-1', 'tool')],
      };

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [snapshotAssistant],
      } as never, callbacks);

      const tcs = result.messages.find(m => m.role === 'assistant')?.toolCalls;
      expect(tcs?.length).toBeGreaterThanOrEqual(2);
      expect(tcs?.some(tc => tc.id === 'tc-2')).toBe(true);
    });

    it('recovers tool names from pendingToolCalls', () => {
      const pending = new Map();
      pending.set('tc-pending', { id: 'tc-pending', name: 'Grep', args: '{}' });

      const assistantMsg = msg({
        role: 'assistant',
        content: '',
        toolCalls: [
          { ...toolCall('tc-pending', 'tool') },
        ],
      });
      const state = makeState({
        messages: [assistantMsg],
        pendingToolCalls: pending,
      });

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [assistantMsg],
      } as never, callbacks);

      const tc = result.messages.find(m => m.role === 'assistant')?.toolCalls?.[0];
      expect(tc?.function.name).toBe('Grep');
    });

    it('recovers tool names from standalone tool messages', () => {
      const toolMsg = msg({
        role: 'tool',
        toolCalls: [toolCall('tc-1', 'Write')],
      });
      const assistantMsg = msg({
        role: 'assistant',
        content: '',
        toolCalls: [toolCall('tc-1', 'unknown_tool')],
      });
      const state = makeState({
        messages: [toolMsg, assistantMsg],
      });

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [toolMsg, assistantMsg],
      } as never, callbacks);

      const tc = result.messages.find(m => m.role === 'assistant')?.toolCalls?.[0];
      expect(tc?.function.name).toBe('Write');
    });

    it('removes redundant standalone tool messages when nested', () => {
      const assistantMsg = msg({
        role: 'assistant',
        content: '',
        toolCalls: [toolCall('tc-1', 'Read')],
      });
      const toolMsg = {
        ...msg({ role: 'tool', content: '' }),
        toolCallId: 'tc-1',
        toolCalls: [toolCall('tc-1', 'Read')],
      } as PlatformMessage;
      const state = makeState({ messages: [assistantMsg, toolMsg] });

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [assistantMsg, toolMsg],
      } as never, callbacks);

      // The standalone tool message should be filtered out
      expect(result.messages.filter(m => m.role === 'tool')).toHaveLength(0);
    });

    it('inserts new snapshot messages before existing ones by position', () => {
      const msg1 = msg({ role: 'user', content: 'hello', timestamp: '2025-01-01T00:00:01Z' });
      const msg3 = msg({ role: 'assistant', content: 'world', timestamp: '2025-01-01T00:00:03Z' });
      const state = makeState({ messages: [msg1, msg3] });

      const msg2 = msg({ role: 'assistant', content: 'inserted', timestamp: '2025-01-01T00:00:02Z' });

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: [msg1, msg2, msg3],
      } as never, callbacks);

      expect(result.messages).toHaveLength(3);
      // After timestamp sort, msg2 should be between msg1 and msg3
      expect(result.messages[1].content).toBe('inserted');
    });

    it('clears pendingChildren after snapshot merge', () => {
      const state = makeState({
        messages: [msg({ role: 'assistant', content: '' })],
        pendingChildren: new Map([['tc-1', [msg({ role: 'tool', content: '' })]]]),
      });

      const result = processAGUIEvent(state, {
        type: EventType.MESSAGES_SNAPSHOT,
        messages: state.messages,
      } as never, callbacks);

      expect(result.pendingChildren.size).toBe(0);
    });
  });

  // ── handleRunFinished: flush pending thinking ──
  describe('handleRunFinished — flush pending thinking', () => {
    it('flushes pending thinking (legacy path)', () => {
      callbacks.currentRunIdRef.current = 'run-1';
      const state = makeState({
        currentThinking: { id: 'think-1', content: 'deep thought legacy' },
      });

      const result = processAGUIEvent(state, {
        type: EventType.RUN_FINISHED,
        runId: 'run-1',
      } as never, callbacks);

      expect(result.currentThinking).toBeNull();
      const thinkingMsg = result.messages.find(m =>
        typeof m.content === 'object' && m.content !== null && 'thinking' in m.content
      );
      expect(thinkingMsg).toBeDefined();
      expect((thinkingMsg!.content as { thinking: string }).thinking).toBe('deep thought legacy');
    });

    it('flushes all three pending streams together', () => {
      callbacks.currentRunIdRef.current = 'run-1';
      const state = makeState({
        currentMessage: { id: 'msg-1', role: 'assistant', content: 'text' },
        currentReasoning: { id: 'r-1', content: 'reason' },
        currentThinking: { id: 't-1', content: 'think' },
      });

      const result = processAGUIEvent(state, {
        type: EventType.RUN_FINISHED,
        runId: 'run-1',
      } as never, callbacks);

      expect(result.currentMessage).toBeNull();
      expect(result.currentReasoning).toBeNull();
      expect(result.currentThinking).toBeNull();
      // Should have 3 flushed messages
      expect(result.messages).toHaveLength(3);
    });
  });

  // ── Unknown event type ──
  describe('unknown events', () => {
    it('returns state unchanged for unknown event type', () => {
      const state = makeState();
      const result = processAGUIEvent(state, {
        type: 'UNKNOWN_EVENT',
      } as never, callbacks);
      expect(result.status).toBe('idle');
    });
  });
});
