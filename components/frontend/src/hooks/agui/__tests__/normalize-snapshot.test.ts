import { describe, it, expect } from 'vitest';
import { normalizeSnapshotMessages } from '../normalize-snapshot';
import type { PlatformMessage } from '@/types/agui';

function msg(overrides: Partial<PlatformMessage> & { role: string }): PlatformMessage {
  return { id: crypto.randomUUID(), ...overrides } as PlatformMessage;
}

describe('normalizeSnapshotMessages', () => {
  it('returns empty array for empty input', () => {
    expect(normalizeSnapshotMessages([])).toEqual([]);
  });

  it('returns messages unchanged when no tool calls exist', () => {
    const messages = [
      msg({ role: 'user', content: 'hello' }),
      msg({ role: 'assistant', content: 'hi' }),
    ];
    const result = normalizeSnapshotMessages(messages);
    expect(result).toHaveLength(2);
    expect(result[0].content).toBe('hello');
  });

  it('returns messages unchanged when assistant has no toolCalls', () => {
    const messages = [
      msg({ role: 'user', content: 'do something' }),
      msg({ role: 'assistant', content: 'okay', toolCalls: undefined }),
    ];
    const result = normalizeSnapshotMessages(messages);
    expect(result).toHaveLength(2);
  });

  it('nests parent tool result into assistant toolCall', () => {
    const parentId = 'tc-parent-1';
    const messages = [
      msg({
        role: 'assistant',
        content: '',
        toolCalls: [{ id: parentId, type: 'function', function: { name: 'bash', arguments: '{"cmd":"ls"}' } }],
      }),
      msg({ role: 'tool', toolCallId: parentId, content: 'file1.txt\nfile2.txt' }),
    ];

    const result = normalizeSnapshotMessages(messages);
    // Parent tool result message should be removed from top level
    expect(result).toHaveLength(1);
    expect(result[0].role).toBe('assistant');
    expect(result[0].toolCalls![0].result).toBe('file1.txt\nfile2.txt');
    expect(result[0].toolCalls![0].status).toBe('completed');
  });

  it('nests child tool messages under parent tool call', () => {
    const parentId = 'tc-parent-1';
    const childId = 'tc-child-1';
    const messages = [
      msg({
        role: 'assistant',
        content: '',
        toolCalls: [{ id: parentId, type: 'function', function: { name: 'subagent', arguments: '{}' } }],
      }),
      msg({ role: 'tool', toolCallId: childId, name: 'readFile', content: 'child result' }),
      msg({ role: 'tool', toolCallId: parentId, content: 'parent result' }),
    ];

    const result = normalizeSnapshotMessages(messages);
    // Both tool messages should be removed; child nested under assistant's toolCalls
    expect(result).toHaveLength(1);
    const toolCalls = result[0].toolCalls!;
    // Parent tool call + child tool call
    expect(toolCalls.length).toBe(2);
    const child = toolCalls.find(tc => tc.id === childId);
    expect(child).toBeDefined();
    expect(child!.parentToolUseId).toBe(parentId);
    expect(child!.result).toBe('child result');
  });

  it('does not mutate original messages', () => {
    const parentId = 'tc-parent-1';
    const original = [
      msg({
        role: 'assistant',
        content: '',
        toolCalls: [{ id: parentId, type: 'function', function: { name: 'bash', arguments: '' } }],
      }),
      msg({ role: 'tool', toolCallId: parentId, content: 'result' }),
    ];
    const originalToolCallsLength = original[0].toolCalls!.length;

    normalizeSnapshotMessages(original);

    // Original should not be mutated
    expect(original[0].toolCalls).toHaveLength(originalToolCallsLength);
    expect(original).toHaveLength(2);
  });

  it('handles multiple parent tool calls', () => {
    const parent1 = 'tc-p1';
    const parent2 = 'tc-p2';
    const messages = [
      msg({
        role: 'assistant',
        content: '',
        toolCalls: [
          { id: parent1, type: 'function', function: { name: 'tool1', arguments: '' } },
          { id: parent2, type: 'function', function: { name: 'tool2', arguments: '' } },
        ],
      }),
      msg({ role: 'tool', toolCallId: parent1, content: 'result1' }),
      msg({ role: 'tool', toolCallId: parent2, content: 'result2' }),
    ];

    const result = normalizeSnapshotMessages(messages);
    expect(result).toHaveLength(1);
    const tc1 = result[0].toolCalls!.find(tc => tc.id === parent1);
    const tc2 = result[0].toolCalls!.find(tc => tc.id === parent2);
    expect(tc1!.result).toBe('result1');
    expect(tc2!.result).toBe('result2');
  });
});
