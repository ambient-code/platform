/**
 * Normalize MESSAGES_SNAPSHOT data for the internal PlatformMessage format.
 *
 * The runner sends snapshots where sub-agent child tool results appear as
 * separate flat role=tool messages instead of nested toolCalls entries with
 * parentToolUseId on the assistant message.
 *
 * This function nests child tool messages under their parent tool call so
 * the page rendering code (which builds hierarchy from parentToolUseId)
 * works correctly for both live-streamed and snapshot-restored sessions.
 *
 * Note: Since we now use the @ag-ui/core ToolCall format natively
 * ({type:"function", function:{name, arguments}}), no format conversion
 * is needed -- snapshots already arrive in the correct format.
 */

import type { PlatformMessage } from '@/types/agui'

export function normalizeSnapshotMessages(snapshotMessages: PlatformMessage[]): PlatformMessage[] {
  // Shallow-clone messages so we can mutate toolCalls arrays safely
  const messages = snapshotMessages.map(m => ({
    ...m,
    toolCalls: m.toolCalls ? [...m.toolCalls] : undefined,
  }))

  // Step 1: Identify parent tool call IDs from assistant messages' toolCalls
  const parentToolCallIds = new Set<string>()
  for (const msg of messages) {
    if (msg.role === 'assistant' && msg.toolCalls) {
      for (const tc of msg.toolCalls) {
        if (tc.id) parentToolCallIds.add(tc.id)
      }
    }
  }

  if (parentToolCallIds.size === 0) return messages

  // Step 2: Find parent tool result message indices
  const parentResultIndex = new Map<string, number>()
  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i]
    if (msg.role === 'tool' && 'toolCallId' in msg && msg.toolCallId && parentToolCallIds.has(msg.toolCallId)) {
      parentResultIndex.set(msg.toolCallId, i)
    }
  }

  // Step 3: Nest child tool messages under their parent tool call
  const indicesToRemove = new Set<number>()

  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i]
    if (msg.role !== 'tool' || !('toolCallId' in msg) || !msg.toolCallId) continue

    if (parentToolCallIds.has(msg.toolCallId)) {
      // This is a parent tool result -- move content to the parent's toolCall.result
      const parentId = msg.toolCallId
      for (const assistantMsg of messages) {
        if (assistantMsg.role !== 'assistant' || !assistantMsg.toolCalls) continue
        const parentTC = assistantMsg.toolCalls.find(tc => tc.id === parentId)
        if (parentTC) {
          parentTC.result = ('content' in msg ? msg.content : '') as string || ''
          if (!parentTC.status) parentTC.status = 'completed'
          indicesToRemove.add(i)
          break
        }
      }
      continue
    }

    // This is potentially a child tool result.
    // Find the nearest parent whose result message comes AFTER this child.
    let bestParentId: string | null = null
    let bestParentResultIdx = Infinity
    for (const [parentId, resultIdx] of parentResultIndex) {
      if (resultIdx > i && resultIdx < bestParentResultIdx) {
        bestParentId = parentId
        bestParentResultIdx = resultIdx
      }
    }
    if (!bestParentId) continue

    // Verify this child appears after the assistant message that owns the parent
    let isAfterAssistant = false
    for (let a = i - 1; a >= 0; a--) {
      if (messages[a].role === 'assistant' &&
          messages[a].toolCalls?.some(tc => tc.id === bestParentId)) {
        isAfterAssistant = true
        break
      }
    }
    if (!isAfterAssistant) continue

    // Add child as a toolCalls entry with parentToolUseId on the assistant message
    for (const assistantMsg of messages) {
      if (assistantMsg.role !== 'assistant' || !assistantMsg.toolCalls) continue
      if (!assistantMsg.toolCalls.some(tc => tc.id === bestParentId)) continue

      if (!assistantMsg.toolCalls.some(tc => tc.id === msg.toolCallId)) {
        assistantMsg.toolCalls.push({
          id: msg.toolCallId,
          type: 'function',
          function: {
            name: ('name' in msg ? msg.name : null) as string || 'tool',
            arguments: '',
          },
          result: ('content' in msg ? msg.content : '') as string || '',
          status: 'completed',
          parentToolUseId: bestParentId,
        })
      }
      indicesToRemove.add(i)
      break
    }
  }

  // Step 4: Remove nested messages from top level
  return messages.filter((_, idx) => !indicesToRemove.has(idx))
}
