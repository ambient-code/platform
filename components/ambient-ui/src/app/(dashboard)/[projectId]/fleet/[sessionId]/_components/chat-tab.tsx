'use client'

import { useState, useCallback, useMemo, useRef, useEffect } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { User, Bot, Wrench, Send, ChevronDown, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import type { DomainSession, DomainSessionMessage, SessionEventType } from '@/domain/types'
import { useSessionMessages } from '@/queries/use-session-messages'
import { useSendMessage } from '@/queries/use-send-message'
import { formatRelativeTime } from '@/lib/format-timestamp'
import { cn } from '@/lib/utils'
import { useLiveTail, LiveIndicator, JumpToLatestPill } from './live-tail-indicator'

const CHAT_EVENT_TYPES: ReadonlySet<SessionEventType> = new Set([
  'user',
  'assistant',
  'tool_use',
  'tool_result',
])

type ToolPayload = {
  name: string
  arguments: Record<string, unknown>
}

function tryParseToolPayload(payload: string): ToolPayload | null {
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

function tryParseToolResult(payload: string): ToolResultPayload | null {
  try {
    const parsed: unknown = JSON.parse(payload)
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
      return null
    }
    const obj = parsed as Record<string, unknown>
    let result = typeof obj.result === 'string' ? obj.result : payload
    // Strip wrapping quotes from result (e.g. "\"(Bash completed)\"" → "(Bash completed)")
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

function tryFormatJson(payload: string): string {
  try {
    const parsed: unknown = JSON.parse(payload)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return payload
  }
}

// ---- Message Bubble Components ----

function UserMessage({ message }: { message: DomainSessionMessage }) {
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
        <div className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-foreground">
          <pre className="whitespace-pre-wrap break-words font-sans">
            {message.payload}
          </pre>
        </div>
      </div>
    </article>
  )
}

function AssistantMessage({ message }: { message: DomainSessionMessage }) {
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

function ToolUseMessage({ message }: { message: DomainSessionMessage }) {
  const [expanded, setExpanded] = useState(false)
  const toolPayload = tryParseToolPayload(message.payload)
  const toolName = toolPayload?.name ?? 'Tool Call'
  const argsText = toolPayload
    ? JSON.stringify(toolPayload.arguments, null, 2)
    : tryFormatJson(message.payload)

  return (
    <article
      aria-label={`Tool call: ${toolName}, ${formatRelativeTime(message.createdAt)}`}
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
          aria-controls={`tool-args-${message.id}`}
        >
          {expanded ? (
            <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
          )}
          <Wrench className="h-3 w-3" aria-hidden="true" />
          <span className="font-mono text-xs font-medium text-foreground">{toolName}</span>
          <span className="text-xs text-muted-foreground">
            {formatRelativeTime(message.createdAt)}
          </span>
        </button>
        {expanded && Object.keys(toolPayload?.arguments ?? {}).length > 0 && (
          <div
            id={`tool-args-${message.id}`}
            className="mt-1.5 rounded-md border border-border bg-muted/50 p-2"
          >
            <div className="mb-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              Arguments
            </div>
            <pre className="whitespace-pre-wrap break-words font-mono text-xs text-foreground max-h-[300px] overflow-y-auto">
              {argsText}
            </pre>
          </div>
        )}
      </div>
    </article>
  )
}

function ToolResultMessage({ message }: { message: DomainSessionMessage }) {
  const [expanded, setExpanded] = useState(false)
  const parsed = tryParseToolResult(message.payload)
  const displayText = parsed ? parsed.result : tryFormatJson(message.payload)

  return (
    <article
      aria-label={`Tool result, ${formatRelativeTime(message.createdAt)}`}
      className="flex gap-3 px-4 py-1 ml-10"
    >
      <div className="min-w-0 flex-1">
        <button
          type="button"
          className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
          onClick={() => setExpanded(prev => !prev)}
          aria-expanded={expanded}
          aria-controls={`tool-result-${message.id}`}
        >
          {expanded ? (
            <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
          )}
          <span className="text-xs font-medium">Result</span>
        </button>
        {expanded && (
          <div
            id={`tool-result-${message.id}`}
            className="mt-1 rounded-md border border-border bg-muted/50 p-2 border-l-2 border-l-primary/30"
          >
            <pre className="whitespace-pre-wrap break-words font-mono text-xs text-foreground max-h-[300px] overflow-y-auto">
              {displayText}
            </pre>
          </div>
        )}
      </div>
    </article>
  )
}

function ChatMessage({ message }: { message: DomainSessionMessage }) {
  switch (message.eventType) {
    case 'user':
      return <UserMessage message={message} />
    case 'assistant':
      return <AssistantMessage message={message} />
    case 'tool_use':
      return <ToolUseMessage message={message} />
    case 'tool_result':
      return <ToolResultMessage message={message} />
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

function PhaseIndicator({ phase }: { phase: string }) {
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

function ChatInput({ sessionId, phase, disabled }: ChatInputProps) {
  const [input, setInput] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const sendMessage = useSendMessage(sessionId)
  const isRunning = phase === 'Running'
  const canSend = isRunning && !disabled && input.trim().length > 0 && !sendMessage.isPending

  const handleSend = useCallback(() => {
    const trimmed = input.trim()
    if (!trimmed || !isRunning || sendMessage.isPending) return
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
      <div className="flex items-center gap-2 mb-2">
        <PhaseIndicator phase={phase} />
        {!isRunning && (
          <span className="text-xs text-muted-foreground">
            Send is disabled while the session is not running.
          </span>
        )}
        {sendMessage.isError && (
          <span className="text-xs text-destructive">
            Failed to send message. Please try again.
          </span>
        )}
      </div>
      <div className="flex gap-2">
        <Textarea
          ref={textareaRef}
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={isRunning ? 'Send a message... (Enter to send, Shift+Enter for new line)' : 'Session is not running'}
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
    </div>
  )
}

// ---- Main Chat Tab ----

export function ChatTab({ session }: { session: DomainSession }) {
  const { data, isLoading, error } = useSessionMessages(session.id)

  const chatMessages = useMemo(() => {
    const messages = data?.items ?? []
    return messages
      .filter(m => CHAT_EVENT_TYPES.has(m.eventType))
      .filter(m => {
        if (m.eventType === 'assistant' && !m.payload.trim()) return false
        return true
      })
  }, [data])

  const { scrollRef, sentinelRef, isAtBottom, newEventCount, scrollToBottom } =
    useLiveTail(chatMessages.length)

  // Auto-scroll on initial load
  const hasScrolledOnLoad = useRef(false)
  useEffect(() => {
    if (!isLoading && chatMessages.length > 0 && !hasScrolledOnLoad.current) {
      hasScrolledOnLoad.current = true
      // Defer to let the DOM render
      requestAnimationFrame(() => {
        scrollToBottom()
      })
    }
  }, [isLoading, chatMessages.length, scrollToBottom])

  if (error) {
    return (
      <div className="pt-4">
        <p className="text-sm text-destructive">
          Failed to load messages: {error.message}
        </p>
      </div>
    )
  }

  return (
    <div className="pt-4">
      <Card className="flex flex-col overflow-hidden p-0">
        {/* Live indicator */}
        {isAtBottom && chatMessages.length > 0 && (
          <div className="absolute top-2 right-3 z-10">
            <LiveIndicator />
          </div>
        )}

        {/* Message area */}
        <CardContent className="flex-1 p-0">
          {isLoading ? (
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
          ) : chatMessages.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
              <Bot className="h-10 w-10 mb-3 opacity-40" aria-hidden="true" />
              <p className="text-sm">No conversation messages yet.</p>
              <p className="text-xs mt-1">
                Messages will appear here as the session runs.
              </p>
            </div>
          ) : (
            <div
              ref={scrollRef}
              className="max-h-[600px] overflow-y-auto relative"
              role="log"
              aria-label="Chat messages"
            >
              <div className="divide-y divide-transparent">
                {chatMessages.map(message => (
                  <ChatMessage key={message.id} message={message} />
                ))}
              </div>
              <div ref={sentinelRef} className="h-1" aria-hidden="true" />
              <JumpToLatestPill
                newEventCount={newEventCount}
                onClick={scrollToBottom}
              />
            </div>
          )}
        </CardContent>

        {/* Input area */}
        <ChatInput
          sessionId={session.id}
          phase={session.phase}
          disabled={isLoading}
        />
      </Card>
    </div>
  )
}
