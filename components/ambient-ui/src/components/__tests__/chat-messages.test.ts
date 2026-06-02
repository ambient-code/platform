import { describe, it, expect } from 'vitest'
import type { DomainSessionMessage } from '@/domain/types'
import {
  tryParseToolPayload,
  tryParseToolResult,
  extractLastAssistantMessage,
  enrichMessages,
  groupChatItems,
  buildChatItems,
} from '../chat-messages'

// ---- Factory ----

let _seq = 0
function makeMsg(
  overrides: Partial<DomainSessionMessage> & Pick<DomainSessionMessage, 'eventType'>,
): DomainSessionMessage {
  _seq += 1
  return {
    id: `msg-${_seq}`,
    sessionId: 'sess-1',
    payload: '',
    seq: _seq,
    createdAt: new Date().toISOString(),
    ...overrides,
  }
}

// ---- tryParseToolPayload ----

describe('tryParseToolPayload', () => {
  it('extracts name from the "tool" field', () => {
    const result = tryParseToolPayload(JSON.stringify({ tool: 'Read', tool_call_id: 'tc-1' }))
    expect(result).not.toBeNull()
    expect(result!.name).toBe('Read')
  })

  it('extracts name and arguments from "name" + "arguments" fields', () => {
    const result = tryParseToolPayload(
      JSON.stringify({ name: 'Bash', arguments: { command: 'ls' } }),
    )
    expect(result).not.toBeNull()
    expect(result!.name).toBe('Bash')
    expect(result!.arguments).toEqual({ command: 'ls' })
  })

  it('extracts arguments from the "input" field when "arguments" is absent', () => {
    const result = tryParseToolPayload(
      JSON.stringify({ tool: 'Write', input: { path: '/a' } }),
    )
    expect(result).not.toBeNull()
    expect(result!.name).toBe('Write')
    expect(result!.arguments).toEqual({ path: '/a' })
  })

  it('returns null for invalid JSON', () => {
    expect(tryParseToolPayload('not json {')).toBeNull()
  })

  it('returns null for non-object JSON (array)', () => {
    expect(tryParseToolPayload('[1,2,3]')).toBeNull()
  })

  it('returns null for non-object JSON (string)', () => {
    expect(tryParseToolPayload('"hello"')).toBeNull()
  })

  it('returns null when neither "tool" nor "name" is present', () => {
    expect(tryParseToolPayload(JSON.stringify({ arguments: {} }))).toBeNull()
  })

  it('returns empty arguments when neither "arguments" nor "input" is present', () => {
    const result = tryParseToolPayload(JSON.stringify({ tool: 'Read' }))
    expect(result).not.toBeNull()
    expect(result!.arguments).toEqual({})
  })
})

// ---- tryParseToolResult ----

describe('tryParseToolResult', () => {
  it('extracts result and toolCallId', () => {
    const result = tryParseToolResult(
      JSON.stringify({ tool_call_id: 'tc-1', result: 'output' }),
    )
    expect(result).not.toBeNull()
    expect(result!.result).toBe('output')
    expect(result!.toolCallId).toBe('tc-1')
  })

  it('strips wrapping double-quotes from result', () => {
    const result = tryParseToolResult(
      JSON.stringify({ tool_call_id: 'tc-2', result: '"wrapped value"' }),
    )
    expect(result).not.toBeNull()
    expect(result!.result).toBe('wrapped value')
  })

  it('returns empty string toolCallId when tool_call_id is missing', () => {
    const result = tryParseToolResult(JSON.stringify({ result: 'data' }))
    expect(result).not.toBeNull()
    expect(result!.toolCallId).toBe('')
  })

  it('returns null for invalid JSON', () => {
    expect(tryParseToolResult('bad json')).toBeNull()
  })

  it('returns null for non-object JSON', () => {
    expect(tryParseToolResult('42')).toBeNull()
  })
})

// ---- extractLastAssistantMessage ----

describe('extractLastAssistantMessage', () => {
  it('extracts last_assistant_message from a system event payload', () => {
    const payload = JSON.stringify({
      value: { last_assistant_message: 'Hello from the assistant' },
    })
    expect(extractLastAssistantMessage(payload)).toBe('Hello from the assistant')
  })

  it('returns null when last_assistant_message is missing', () => {
    const payload = JSON.stringify({ value: { other_field: 'x' } })
    expect(extractLastAssistantMessage(payload)).toBeNull()
  })

  it('returns null when last_assistant_message is whitespace-only', () => {
    const payload = JSON.stringify({ value: { last_assistant_message: '   ' } })
    expect(extractLastAssistantMessage(payload)).toBeNull()
  })

  it('returns null when value is not an object', () => {
    const payload = JSON.stringify({ value: 'string-value' })
    expect(extractLastAssistantMessage(payload)).toBeNull()
  })

  it('returns null for invalid JSON', () => {
    expect(extractLastAssistantMessage('not json')).toBeNull()
  })

  it('returns null for a JSON array', () => {
    expect(extractLastAssistantMessage('[1]')).toBeNull()
  })
})

// ---- enrichMessages ----

describe('enrichMessages', () => {
  it('enriches empty assistant messages from a preceding system event', () => {
    const messages = [
      makeMsg({
        eventType: 'system',
        payload: JSON.stringify({
          value: { last_assistant_message: 'Extracted text' },
        }),
      }),
      makeMsg({ eventType: 'assistant', payload: '' }),
    ]

    const result = enrichMessages(messages)
    expect(result).toHaveLength(2)
    // The system event passes through
    expect(result[0].eventType).toBe('system')
    // The assistant message is enriched
    expect(result[1].eventType).toBe('assistant')
    expect(result[1].payload).toBe('Extracted text')
  })

  it('drops empty assistant messages with no nearby system event', () => {
    const messages = [
      makeMsg({ eventType: 'user', payload: 'hello' }),
      makeMsg({ eventType: 'assistant', payload: '' }),
    ]

    const result = enrichMessages(messages)
    expect(result).toHaveLength(1)
    expect(result[0].eventType).toBe('user')
  })

  it('drops empty assistant messages when system event lacks last_assistant_message', () => {
    const messages = [
      makeMsg({
        eventType: 'system',
        payload: JSON.stringify({ value: { some_other: 'data' } }),
      }),
      makeMsg({ eventType: 'assistant', payload: '' }),
    ]

    const result = enrichMessages(messages)
    // System passes through, empty assistant is dropped
    expect(result).toHaveLength(1)
    expect(result[0].eventType).toBe('system')
  })

  it('passes through non-empty assistant messages unchanged', () => {
    const messages = [
      makeMsg({ eventType: 'assistant', payload: 'I have content' }),
    ]

    const result = enrichMessages(messages)
    expect(result).toHaveLength(1)
    expect(result[0].payload).toBe('I have content')
  })

  it('passes through non-assistant messages unchanged', () => {
    const messages = [
      makeMsg({ eventType: 'user', payload: 'question' }),
      makeMsg({ eventType: 'tool_use', payload: '{}' }),
      makeMsg({ eventType: 'lifecycle', payload: 'started' }),
    ]

    const result = enrichMessages(messages)
    expect(result).toHaveLength(3)
    expect(result.map(m => m.eventType)).toEqual(['user', 'tool_use', 'lifecycle'])
  })

  it('looks back up to 3 positions for a system event', () => {
    const messages = [
      makeMsg({
        eventType: 'system',
        payload: JSON.stringify({
          value: { last_assistant_message: 'Found it' },
        }),
      }),
      makeMsg({ eventType: 'user', payload: 'filler1' }),
      makeMsg({ eventType: 'user', payload: 'filler2' }),
      makeMsg({ eventType: 'assistant', payload: '' }),
    ]

    const result = enrichMessages(messages)
    const assistant = result.find(m => m.eventType === 'assistant')
    expect(assistant).toBeDefined()
    expect(assistant!.payload).toBe('Found it')
  })

  it('does not look back more than 3 positions', () => {
    const messages = [
      makeMsg({
        eventType: 'system',
        payload: JSON.stringify({
          value: { last_assistant_message: 'Too far' },
        }),
      }),
      makeMsg({ eventType: 'user', payload: 'a' }),
      makeMsg({ eventType: 'user', payload: 'b' }),
      makeMsg({ eventType: 'user', payload: 'c' }),
      makeMsg({ eventType: 'assistant', payload: '' }),
    ]

    const result = enrichMessages(messages)
    // The empty assistant should be dropped since system event is 4 positions back
    const assistant = result.find(m => m.eventType === 'assistant')
    expect(assistant).toBeUndefined()
  })
})

// ---- groupChatItems ----

describe('groupChatItems', () => {
  it('groups a tool_use followed by a matching tool_result', () => {
    const toolCallId = 'tc-pair-1'
    const messages = [
      makeMsg({
        eventType: 'tool_use',
        payload: JSON.stringify({ tool: 'Read', tool_call_id: toolCallId }),
      }),
      makeMsg({
        eventType: 'tool_result',
        payload: JSON.stringify({ tool_call_id: toolCallId, result: 'file contents' }),
      }),
    ]

    const items = groupChatItems(messages)
    expect(items).toHaveLength(1)
    expect(items[0].kind).toBe('tool_call')
    if (items[0].kind === 'tool_call') {
      expect(items[0].group.toolUse.eventType).toBe('tool_use')
      expect(items[0].group.toolResult).not.toBeNull()
      expect(items[0].group.toolResult!.eventType).toBe('tool_result')
    }
  })

  it('treats tool_result with no matching tool_use as a standalone message', () => {
    const messages = [
      makeMsg({
        eventType: 'tool_result',
        payload: JSON.stringify({ tool_call_id: 'orphan-id', result: 'data' }),
      }),
    ]

    const items = groupChatItems(messages)
    expect(items).toHaveLength(1)
    expect(items[0].kind).toBe('message')
  })

  it('wraps user and assistant messages as message items', () => {
    const messages = [
      makeMsg({ eventType: 'user', payload: 'hi' }),
      makeMsg({ eventType: 'assistant', payload: 'hello' }),
    ]

    const items = groupChatItems(messages)
    expect(items).toHaveLength(2)
    expect(items[0].kind).toBe('message')
    expect(items[1].kind).toBe('message')
    if (items[0].kind === 'message') {
      expect(items[0].message.eventType).toBe('user')
    }
    if (items[1].kind === 'message') {
      expect(items[1].message.eventType).toBe('assistant')
    }
  })

  it('pairs multiple concurrent tool calls correctly', () => {
    const messages = [
      makeMsg({
        eventType: 'tool_use',
        payload: JSON.stringify({ tool: 'Read', tool_call_id: 'tc-a' }),
      }),
      makeMsg({
        eventType: 'tool_use',
        payload: JSON.stringify({ tool: 'Bash', tool_call_id: 'tc-b' }),
      }),
      makeMsg({
        eventType: 'tool_result',
        payload: JSON.stringify({ tool_call_id: 'tc-b', result: 'bash output' }),
      }),
      makeMsg({
        eventType: 'tool_result',
        payload: JSON.stringify({ tool_call_id: 'tc-a', result: 'file data' }),
      }),
    ]

    const items = groupChatItems(messages)
    expect(items).toHaveLength(2)
    // Both should be tool_call items
    expect(items[0].kind).toBe('tool_call')
    expect(items[1].kind).toBe('tool_call')
    if (items[0].kind === 'tool_call' && items[1].kind === 'tool_call') {
      // tc-a was first, tc-b second
      expect(items[0].group.id).toBe('tc-a')
      expect(items[0].group.toolResult).not.toBeNull()
      expect(items[1].group.id).toBe('tc-b')
      expect(items[1].group.toolResult).not.toBeNull()
    }
  })

  it('leaves tool_use without a result as a tool_call with null toolResult', () => {
    const messages = [
      makeMsg({
        eventType: 'tool_use',
        payload: JSON.stringify({ tool: 'Bash', tool_call_id: 'tc-pending' }),
      }),
    ]

    const items = groupChatItems(messages)
    expect(items).toHaveLength(1)
    expect(items[0].kind).toBe('tool_call')
    if (items[0].kind === 'tool_call') {
      expect(items[0].group.toolResult).toBeNull()
    }
  })
})

// ---- buildChatItems (integration) ----

describe('buildChatItems', () => {
  it('processes a full conversation flow end-to-end', () => {
    const messages = [
      makeMsg({ eventType: 'user', payload: 'Fix the bug' }),
      makeMsg({ eventType: 'lifecycle', payload: 'session_started' }),
      makeMsg({
        eventType: 'system',
        payload: JSON.stringify({
          value: { last_assistant_message: 'I will fix the bug.' },
        }),
      }),
      makeMsg({ eventType: 'assistant', payload: '' }),
      makeMsg({
        eventType: 'tool_use',
        payload: JSON.stringify({ tool: 'Read', tool_call_id: 'tc-build' }),
      }),
      makeMsg({
        eventType: 'tool_result',
        payload: JSON.stringify({ tool_call_id: 'tc-build', result: 'src code' }),
      }),
      makeMsg({ eventType: 'assistant', payload: 'Done fixing.' }),
    ]

    const items = buildChatItems(messages)

    // lifecycle and system events are filtered out
    const kinds = items.map(i => i.kind)
    expect(kinds).toEqual(['message', 'message', 'tool_call', 'message'])

    // First message is the user message
    if (items[0].kind === 'message') {
      expect(items[0].message.eventType).toBe('user')
      expect(items[0].message.payload).toBe('Fix the bug')
    }

    // Second message is the enriched assistant
    if (items[1].kind === 'message') {
      expect(items[1].message.eventType).toBe('assistant')
      expect(items[1].message.payload).toBe('I will fix the bug.')
    }

    // Tool call is grouped
    if (items[2].kind === 'tool_call') {
      expect(items[2].group.toolResult).not.toBeNull()
    }

    // Final assistant message
    if (items[3].kind === 'message') {
      expect(items[3].message.eventType).toBe('assistant')
      expect(items[3].message.payload).toBe('Done fixing.')
    }
  })

  it('filters out lifecycle and system events from chat items', () => {
    const messages = [
      makeMsg({ eventType: 'lifecycle', payload: 'created' }),
      makeMsg({ eventType: 'system', payload: '{}' }),
      makeMsg({ eventType: 'user', payload: 'hello' }),
    ]

    const items = buildChatItems(messages)
    expect(items).toHaveLength(1)
    if (items[0].kind === 'message') {
      expect(items[0].message.eventType).toBe('user')
    }
  })

  it('includes enriched empty assistant messages derived from system events', () => {
    const messages = [
      makeMsg({
        eventType: 'system',
        payload: JSON.stringify({
          value: { last_assistant_message: 'Enriched reply' },
        }),
      }),
      makeMsg({ eventType: 'assistant', payload: '' }),
    ]

    const items = buildChatItems(messages)
    expect(items).toHaveLength(1)
    if (items[0].kind === 'message') {
      expect(items[0].message.eventType).toBe('assistant')
      expect(items[0].message.payload).toBe('Enriched reply')
    }
  })

  it('returns empty array when given no messages', () => {
    expect(buildChatItems([])).toEqual([])
  })
})
