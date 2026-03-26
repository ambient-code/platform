/**
 * Transforms raw Claude SDK JSONL transcript entries into the frontend's
 * Message[] types so they can be rendered by StreamMessage.
 */

import type { TaskOutputEntry } from "@/types/background-task"
import type {
  Message,
  UserMessage,
  AgentMessage,
  ToolUseMessages,
  ToolUseBlock,
  ToolResultBlock,
} from "@/types/agentic-session"

type SdkContentBlock = {
  type: string
  text?: string
  thinking?: string
  signature?: string
  id?: string
  name?: string
  input?: Record<string, unknown>
  tool_use_id?: string
  content?: string | Array<Record<string, unknown>>
  is_error?: boolean
}

export function transformTaskTranscript(entries: TaskOutputEntry[]): Message[] {
  const messages: Message[] = []
  const pendingToolUses = new Map<string, ToolUseBlock>()

  // Handle raw bash output (non-JSONL entries like {raw: "..."})
  // These come from local_bash background tasks where the output is plain text
  const rawEntries = entries.filter((e) => "raw" in e && Object.keys(e).length === 1)
  if (rawEntries.length > 0 && rawEntries.length === entries.length) {
    // All entries are raw text — this is a bash task output
    const rawText = rawEntries.map((e) => e.raw as string).join("\n")
    if (rawText.trim()) {
      messages.push({
        type: "agent_message",
        content: { type: "text_block", text: "```\n" + rawText + "\n```" },
        model: "bash",
        timestamp: new Date().toISOString(),
      } satisfies AgentMessage)
    }
    return messages
  }

  for (const entry of entries) {
    const entryType = entry.type as string | undefined
    if (!entryType || entryType === "progress") continue

    const msg = entry.message as Record<string, unknown> | undefined
    if (!msg && entryType !== "result") continue

    const timestamp = (entry.timestamp as string) ?? new Date().toISOString()

    if (entryType === "user") {
      const content = msg?.content
      if (typeof content === "string") {
        messages.push({
          type: "user_message",
          content,
          timestamp,
        } satisfies UserMessage)
      } else if (Array.isArray(content)) {
        // Tool result blocks — pair with pending tool uses
        for (const block of content as SdkContentBlock[]) {
          if (block.type === "tool_result" && block.tool_use_id) {
            const toolUse = pendingToolUses.get(block.tool_use_id)
            if (toolUse) {
              const resultContent = typeof block.content === "string"
                ? block.content
                : block.content ?? null
              messages.push({
                type: "tool_use_messages",
                toolUseBlock: toolUse,
                resultBlock: {
                  type: "tool_result_block",
                  tool_use_id: block.tool_use_id,
                  content: resultContent,
                  is_error: block.is_error ?? false,
                } satisfies ToolResultBlock,
                timestamp,
              } satisfies ToolUseMessages)
              pendingToolUses.delete(block.tool_use_id)
            }
          }
        }
      }
    }

    if (entryType === "assistant") {
      const contentBlocks = msg?.content as SdkContentBlock[] | undefined
      if (!Array.isArray(contentBlocks)) continue
      const model = (msg?.model as string) || "unknown"

      for (const block of contentBlocks) {
        if (block.type === "text" && block.text) {
          messages.push({
            type: "agent_message",
            content: { type: "text_block", text: block.text },
            model,
            timestamp,
          } satisfies AgentMessage)
        } else if (block.type === "thinking" && block.thinking) {
          messages.push({
            type: "agent_message",
            content: {
              type: "reasoning_block",
              thinking: block.thinking,
              signature: block.signature ?? "",
            },
            model,
            timestamp,
          } satisfies AgentMessage)
        } else if (block.type === "tool_use" && block.id && block.name) {
          pendingToolUses.set(block.id, {
            type: "tool_use_block",
            id: block.id,
            name: block.name,
            input: (block.input as Record<string, unknown>) ?? {},
          })
        }
      }
    }

    if (entryType === "result") {
      messages.push({
        type: "result_message",
        subtype: "task_result",
        duration_ms: (entry.duration_ms as number) ?? 0,
        duration_api_ms: (entry.duration_api_ms as number) ?? 0,
        is_error: (entry.is_error as boolean) ?? false,
        num_turns: (entry.num_turns as number) ?? 0,
        session_id: (entry.session_id as string) ?? "",
        timestamp,
      })
    }
  }

  return messages
}
