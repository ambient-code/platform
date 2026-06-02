'use client'

import { useState, useCallback, useRef } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { User, Bot, Wrench, Send, ChevronDown, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import type { DomainSessionMessage, SessionEventType } from '@/domain/types'
import { useSendMessage } from '@/queries/use-send-message'
import { formatRelativeTime } from '@/lib/format-timestamp'
import { cn } from '@/lib/utils'

// ---- Constants ----

export const CHAT_EVENT_TYPES: ReadonlySet<SessionEventType> = new Set([
  'user',
  'assistant',
  'tool_use',
  'tool_result',
])

// ---- Payload Parsing Helpers ----

type ToolPayload = {
  name: string
  arguments: Record<string, unknown>
}

export function tryParseToolPayload(payload: string): ToolPayload | null {
  try {
    const parsed: unknown = JSON.parse(payload)
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
      return null
    }
    const obj = parsed as Record<string, unknown>
    const name =
      typeof obj.tool === 'string' ? obj.tool :
      typeof obj.name === 'string' ? obj.name : null
    if (!name) return null

    const args =
      typeof obj.arguments === 'object' && obj.arguments !== null && !Array.isArray(obj.arguments)
        ? (obj.arguments as Record<string, unknown>)
        : typeof obj.input === 'object' && obj.input !== null && !Array.isArray(obj.input)
          ? (obj.input as Record<string, unknown>)
          : {}

    return { name, arguments: args }
  } catch {
    return null
  }
}

type ToolResultPayload = {
  result: string
  toolCallId: string
}

export function tryParseToolResult(payload: string): ToolResultPayload | null {
  try {
    const parsed: unknown = JSON.parse(payload)
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
      return null
    }
    const obj = parsed as Record<string, unknown>
    let result = typeof obj.result === 'string' ? obj.result : payload
    if (result.startsWith('"') && result.endsWith('"')) {
      result = result.slice(1, -1)
    }
    return {
      result,
      toolCallId: typeof obj.tool_call_id === 'string' ? obj.tool_call_id : '',
    }
  } catch {
    return null
  }
}

export function tryFormatJson(payload: string): string {
  try {
    const parsed: unknown = JSON.parse(payload)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return payload
  }
}

// ---- Message Enrichment ----

export function extractLastAssistantMessage(payload: string): string | null {
  try {
    const parsed: unknown = JSON.parse(payload)
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) return null
    const obj = parsed as Record<string, unknown>
    if (typeof obj.value !== 'object' || obj.value === null) return null
    const value = obj.value as Record<string, unknown>
    if (typeof value.last_assistant_message === 'string' && value.last_assistant_message.trim()) {
      return value.last_assistant_message
    }
    return null
  } catch {
    return null
  }
}

export function enrichMessages(messages: DomainSessionMessage[]): DomainSessionMessage[] {
  const enriched: DomainSessionMessage[] = []
  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i]
    if (msg.eventType === 'assistant' && !msg.payload.trim()) {
      for (let j = i - 1; j >= Math.max(0, i - 3); j--) {
        if (messages[j].eventType === 'system') {
          const extracted = extractLastAssistantMessage(messages[j].payload)
          if (extracted) {
            enriched.push({ ...msg, payload: extracted })
            break
          }
        }
      }
      // Empty assistant messages with no extractable text are dropped — the
      // runner sometimes pushes an empty "assistant" record alongside a system
      // event that contains the actual response in last_assistant_message.
      continue
    }
    enriched.push(msg)
  }
  return enriched
}

// ---- Tool Call Grouping ----

export type ToolCallGroup = {
  id: string
  toolUse: DomainSessionMessage
  toolResult: DomainSessionMessage | null
}

export type ChatItem =
  | { kind: 'message'; message: DomainSessionMessage }
  | { kind: 'tool_call'; group: ToolCallGroup }

export function groupChatItems(messages: DomainSessionMessage[]): ChatItem[] {
  const items: ChatItem[] = []
  const pendingToolUses = new Map<string, ToolCallGroup>()

  for (const msg of messages) {
    if (msg.eventType === 'tool_use') {
      const toolPayload = tryParseToolPayload(msg.payload)
      let toolCallId = msg.id
      if (toolPayload) {
        try {
          const raw = JSON.parse(msg.payload) as Record<string, unknown>
          if (typeof raw.tool_call_id === 'string') {
            toolCallId = raw.tool_call_id
          }
        } catch {
          // use msg.id as fallback
        }
      }
      const group: ToolCallGroup = { id: toolCallId, toolUse: msg, toolResult: null }
      pendingToolUses.set(toolCallId, group)
      items.push({ kind: 'tool_call', group })
    } else if (msg.eventType === 'tool_result') {
      const resultParsed = tryParseToolResult(msg.payload)
      const toolCallId = resultParsed?.toolCallId
      if (toolCallId && pendingToolUses.has(toolCallId)) {
        pendingToolUses.get(toolCallId)!.toolResult = msg
      } else {
        items.push({ kind: 'message', message: msg })
      }
    } else {
      items.push({ kind: 'message', message: msg })
    }
  }
  return items
}

// ---- Message Bubble Components ----

export function UserMessage({ message }: { message: DomainSessionMessage }) {
  return (
    <article
      aria-label={`User message, ${formatRelativeTime(message.createdAt)}`}
      className="flex gap-3 px-4 py-3"
    >
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary/10">
        <User className="h-4 w-4 text-primary" aria-hidden="true" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-sm font-medium">You</span>
          <span className="text-xs text-muted-foreground">
            {formatRelativeTime(message.createdAt)}
          </span>
        </div>
        <div className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-foreground prose prose-sm dark:prose-invert max-w-none prose-pre:bg-muted prose-pre:text-foreground prose-code:text-foreground">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {message.payload}
          </ReactMarkdown>
        </div>
      </div>
    </article>
  )
}

export function AssistantMessage({ message }: { message: DomainSessionMessage }) {
  return (
    <article
      aria-label={`Assistant message, ${formatRelativeTime(message.createdAt)}`}
      className="flex gap-3 px-4 py-3"
    >
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-secondary">
        <Bot className="h-4 w-4 text-secondary-foreground" aria-hidden="true" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-sm font-medium">Assistant</span>
          <span className="text-xs text-muted-foreground">
            {formatRelativeTime(message.createdAt)}
          </span>
        </div>
        <div className="rounded-lg bg-muted/50 px-3 py-2 text-sm text-foreground prose prose-sm dark:prose-invert max-w-none prose-pre:bg-muted prose-pre:text-foreground prose-code:text-foreground">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {message.payload}
          </ReactMarkdown>
        </div>
      </div>
    </article>
  )
}

export function ToolCallBlock({ group }: { group: ToolCallGroup }) {
  const [expanded, setExpanded] = useState(false)
  const toolPayload = tryParseToolPayload(group.toolUse.payload)
  const toolName = toolPayload?.name ?? 'Tool Call'
  const hasArgs = Object.keys(toolPayload?.arguments ?? {}).length > 0
  const argsText = toolPayload
    ? JSON.stringify(toolPayload.arguments, null, 2)
    : tryFormatJson(group.toolUse.payload)

  const resultParsed = group.toolResult ? tryParseToolResult(group.toolResult.payload) : null
  const resultText = resultParsed ? resultParsed.result : group.toolResult ? tryFormatJson(group.toolResult.payload) : null

  return (
    <article
      aria-label={`Tool call: ${toolName}, ${formatRelativeTime(group.toolUse.createdAt)}`}
      className="flex gap-3 px-4 py-2"
    >
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted">
        <Wrench className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
      </div>
      <div className="min-w-0 flex-1">
        <button
          type="button"
          className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
          onClick={() => setExpanded(prev => !prev)}
          aria-expanded={expanded}
        >
          {expanded ? (
            <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
          )}
          <span className="font-mono text-xs font-medium text-foreground">{toolName}</span>
          {resultText && !expanded && (
            <span className="text-xs text-muted-foreground truncate max-w-[300px]">
              — {resultText.slice(0, 60)}{resultText.length > 60 ? '...' : ''}
            </span>
          )}
        </button>
        {expanded && (
          <div className="mt-1.5 space-y-1.5">
            {hasArgs && (
              <div className="rounded-md border border-border bg-muted/50 p-2">
                <div className="mb-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                  Arguments
                </div>
                <pre className="whitespace-pre-wrap break-words font-mono text-xs text-foreground max-h-[200px] overflow-y-auto">
                  {argsText}
                </pre>
              </div>
            )}
            {resultText && (
              <div className="rounded-md border border-border bg-muted/50 p-2 border-l-2 border-l-primary/30">
                <div className="mb-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                  Result
                </div>
                <pre className="whitespace-pre-wrap break-words font-mono text-xs text-foreground max-h-[300px] overflow-y-auto">
                  {resultText}
                </pre>
              </div>
            )}
          </div>
        )}
      </div>
    </article>
  )
}

function SimpleChatMessage({ message }: { message: DomainSessionMessage }) {
  switch (message.eventType) {
    case 'user':
      return <UserMessage message={message} />
    case 'assistant':
      return <AssistantMessage message={message} />
    default:
      return null
  }
}

// ---- Phase Status Indicator ----

const PHASE_STYLES: Record<string, string> = {
  Running: 'bg-emerald-100 text-emerald-800 border-emerald-300',
  Pending: 'bg-yellow-100 text-yellow-800 border-yellow-300',
  Creating: 'bg-blue-100 text-blue-800 border-blue-300',
  Stopping: 'bg-orange-100 text-orange-800 border-orange-300',
  Completed: 'bg-[#f2f2f2] text-[#4d4d4d] border-[#e0e0e0]',
  Failed: 'bg-[#ffe3d9] text-[#731f00] border-[#fbbea8]',
  Stopped: 'bg-[#f2f2f2] text-[#4d4d4d] border-[#e0e0e0]',
}

export function PhaseIndicator({ phase }: { phase: string }) {
  const style = PHASE_STYLES[phase] ?? PHASE_STYLES.Stopped
  return (
    <Badge
      variant="outline"
      className={cn('text-[11px] font-medium', style)}
    >
      {phase}
    </Badge>
  )
}

// ---- Chat Input ----

type ChatInputProps = {
  sessionId: string
  phase: string
  disabled: boolean
}

export function ChatInput({ sessionId, phase, disabled }: ChatInputProps) {
  const [input, setInput] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const sendMessage = useSendMessage(sessionId)
  const isRunning = phase === 'Running'
  const canSend = isRunning && !disabled && input.trim().length > 0 && !sendMessage.isPending

  const handleSend = useCallback(() => {
    const trimmed = input.trim()
    if (!trimmed || !isRunning || disabled || sendMessage.isPending) return
    sendMessage.mutate(trimmed, {
      onSuccess: () => {
        setInput('')
        textareaRef.current?.focus()
      },
    })
  }, [input, isRunning, sendMessage])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSend()
      }
    },
    [handleSend],
  )

  return (
    <div className="border-t bg-background px-4 py-3">
      {!isRunning && (
        <div className="flex items-center gap-2 mb-2">
          <span className="text-xs text-muted-foreground">
            Send is disabled while the session is not running.
          </span>
        </div>
      )}
      {sendMessage.isError && (
        <div className="mb-2">
          <span className="text-xs text-destructive">
            Failed to send message. Please try again.
          </span>
        </div>
      )}
      <div className="flex gap-2">
        <Textarea
          ref={textareaRef}
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={isRunning ? 'Send a message...' : 'Session is not running'}
          disabled={!isRunning || sendMessage.isPending}
          className="min-h-[40px] max-h-[120px] resize-none"
          aria-label="Chat message input"
          rows={1}
        />
        <Button
          onClick={handleSend}
          disabled={!canSend}
          size="icon"
          aria-label="Send message"
          className="shrink-0 self-end"
        >
          <Send className="h-4 w-4" />
        </Button>
      </div>
      {isRunning && (
        <p className="text-[10px] text-muted-foreground mt-1">
          Enter to send · Shift+Enter for new line
        </p>
      )}
    </div>
  )
}

// ---- Chat Items List (shared between tab and sidebar) ----

export function ChatItemsList({
  items,
  isLoading,
}: {
  items: ChatItem[]
  isLoading: boolean
}) {
  if (isLoading) {
    return (
      <div className="space-y-4 p-4">
        <div className="flex gap-3">
          <Skeleton className="h-7 w-7 rounded-full" />
          <div className="space-y-2 flex-1">
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-16 w-3/4 rounded-lg" />
          </div>
        </div>
        <div className="flex gap-3">
          <Skeleton className="h-7 w-7 rounded-full" />
          <div className="space-y-2 flex-1">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-24 w-full rounded-lg" />
          </div>
        </div>
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
        <Bot className="h-10 w-10 mb-3 opacity-40" aria-hidden="true" />
        <p className="text-sm">No conversation messages yet.</p>
        <p className="text-xs mt-1">
          Messages will appear here as the session runs.
        </p>
      </div>
    )
  }

  return (
    <div className="divide-y divide-transparent">
      {items.map(item => {
        if (item.kind === 'tool_call') {
          return <ToolCallBlock key={item.group.id} group={item.group} />
        }
        return <SimpleChatMessage key={item.message.id} message={item.message} />
      })}
    </div>
  )
}

/** Filter and group raw messages into chat items */
export function buildChatItems(messages: DomainSessionMessage[]): ChatItem[] {
  const enriched = enrichMessages(messages)
  const chatOnly = enriched.filter(m => CHAT_EVENT_TYPES.has(m.eventType))
  return groupChatItems(chatOnly)
}
