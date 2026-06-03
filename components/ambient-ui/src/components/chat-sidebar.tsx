'use client'

import { useRef, useEffect, useState, useCallback, useMemo } from 'react'
import { useRouter } from 'next/navigation'
import { X, PanelRightClose, PanelRightOpen, GripVertical, ChevronUp, ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { useChatSidebar } from '@/components/chat-sidebar-context'
import {
  ChatItemsList,
  ChatInput,
  buildChatItems,
  PhaseIndicator,
} from '@/components/chat-messages'
import { useSession } from '@/queries/use-sessions'
import { useSessionMessages } from '@/queries/use-session-messages'
import { useLiveTail, LiveIndicator } from '@/app/(dashboard)/[projectId]/sessions/[sessionId]/_components/live-tail-indicator'

const MIN_WIDTH = 320
const MAX_WIDTH = 800
const DEFAULT_WIDTH = 420
const COLLAPSED_WIDTH = 40

export function ChatSidebar() {
  const { openSessionId, isOpen, closeSidebar } = useChatSidebar()
  const router = useRouter()
  const [collapsed, setCollapsed] = useState(false)
  const [width, setWidth] = useState(DEFAULT_WIDTH)
  const [showScrollToTop, setShowScrollToTop] = useState(false)
  const isDragging = useRef(false)
  const startX = useRef(0)
  const startWidth = useRef(DEFAULT_WIDTH)

  const { data: session } = useSession(openSessionId ?? '', undefined)
  const { data: messagesData, isLoading: messagesLoading } = useSessionMessages(
    openSessionId ?? '',
  )

  const chatItems = useMemo(() => {
    return buildChatItems(messagesData?.items ?? [])
  }, [messagesData])

  const { scrollRef, sentinelRef, isAtBottom, newEventCount, scrollToBottom } =
    useLiveTail(chatItems.length)

  const [scrolledUp, setScrolledUp] = useState(false)

  const handleScroll = useCallback(() => {
    const container = scrollRef.current
    if (!container) return
    setShowScrollToTop(container.scrollTop > 300)
    const atBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 50
    setScrolledUp(!atBottom)
  }, [scrollRef])

  const scrollToTop = useCallback(() => {
    scrollRef.current?.scrollTo({ top: 0, behavior: 'smooth' })
  }, [scrollRef])

  // Auto-scroll on initial load
  const hasScrolledOnLoad = useRef(false)
  useEffect(() => {
    if (!messagesLoading && chatItems.length > 0 && !hasScrolledOnLoad.current) {
      hasScrolledOnLoad.current = true
      requestAnimationFrame(() => {
        scrollToBottom()
      })
    }
  }, [messagesLoading, chatItems.length, scrollToBottom])

  // Reset scroll tracking when session changes
  useEffect(() => {
    hasScrolledOnLoad.current = false
  }, [openSessionId])

  // Drag-to-resize handler with cleanup refs
  const dragListenersRef = useRef<{ move: (e: MouseEvent) => void; up: () => void } | null>(null)

  const cleanupDrag = useCallback(() => {
    if (dragListenersRef.current) {
      document.removeEventListener('mousemove', dragListenersRef.current.move)
      document.removeEventListener('mouseup', dragListenersRef.current.up)
      dragListenersRef.current = null
    }
    isDragging.current = false
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }, [])

  useEffect(() => {
    return cleanupDrag
  }, [cleanupDrag])

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      cleanupDrag()
      isDragging.current = true
      startX.current = e.clientX
      startWidth.current = width
      document.body.style.cursor = 'col-resize'
      document.body.style.userSelect = 'none'

      const handleMouseMove = (ev: MouseEvent) => {
        if (!isDragging.current) return
        const delta = startX.current - ev.clientX
        const newWidth = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, startWidth.current + delta))
        setWidth(newWidth)
      }

      const handleMouseUp = () => cleanupDrag()

      dragListenersRef.current = { move: handleMouseMove, up: handleMouseUp }
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)
    },
    [width, cleanupDrag],
  )

  // Handle Escape key to close
  useEffect(() => {
    if (!isOpen) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        closeSidebar()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, closeSidebar])

  if (!isOpen || !openSessionId) return null

  const sessionName = session?.name ?? 'Loading...'
  const sessionPhase = session?.phase ?? 'Pending'

  if (collapsed) {
    return (
      <aside
        className="flex-shrink-0 border-l bg-background flex flex-col items-center py-3 gap-2 h-screen sticky top-0"
        style={{ width: COLLAPSED_WIDTH }}
        aria-label="Chat sidebar (collapsed)"
      >
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={() => setCollapsed(false)}
          aria-label="Expand chat sidebar"
        >
          <PanelRightOpen className="h-4 w-4" />
        </Button>
        <div className="flex-1 flex items-center">
          <span
            className="text-xs font-medium text-muted-foreground [writing-mode:vertical-lr] rotate-180 truncate max-h-48"
            title={sessionName}
          >
            {sessionName}
          </span>
        </div>
      </aside>
    )
  }

  return (
    <aside
      className="flex-shrink-0 border-l bg-background flex flex-col relative h-screen sticky top-0 z-20 max-md:!w-screen max-md:fixed max-md:inset-0 max-md:border-l-0"
      style={{ width }}
      aria-label={`Chat sidebar for session ${sessionName}`}
      role="complementary"
    >
      {/* Drag handle (left edge) */}
      <div
        className="absolute inset-y-0 left-0 w-1 cursor-col-resize hover:bg-primary/20 active:bg-primary/30 z-10 flex items-center"
        onMouseDown={handleMouseDown}
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize chat sidebar"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'ArrowLeft') {
            setWidth(w => Math.min(MAX_WIDTH, w + 20))
          } else if (e.key === 'ArrowRight') {
            setWidth(w => Math.max(MIN_WIDTH, w - 20))
          }
        }}
      >
        <GripVertical className="h-4 w-4 text-muted-foreground/50 -ml-1.5 pointer-events-none" aria-hidden="true" />
      </div>

      {/* Header */}
      <div className="flex items-center gap-2 border-b px-3 py-2 min-h-[48px]">
        <button
          type="button"
          className="flex-1 min-w-0 text-left cursor-pointer hover:opacity-80 transition-opacity"
          onClick={() => {
            const projectId = session?.projectId
            if (projectId && openSessionId) {
              router.push(`/${projectId}/sessions/${openSessionId}`)
            }
          }}
          title="Go to session detail"
        >
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold truncate" title={sessionName}>
              {sessionName}
            </span>
            <PhaseIndicator phase={sessionPhase} />
          </div>
        </button>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => setCollapsed(true)}
            aria-label="Collapse chat sidebar"
          >
            <PanelRightClose className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={closeSidebar}
            aria-label="Close chat sidebar"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Message area */}
      <div className="flex-1 min-h-0 overflow-hidden relative">
        {isAtBottom && chatItems.length > 0 && (
          <div className="absolute top-2 right-3 z-10">
            <LiveIndicator />
          </div>
        )}
        <div
          ref={scrollRef}
          className="h-full overflow-y-auto"
          role="log"
          aria-label="Chat messages"
          onScroll={handleScroll}
        >
          <ChatItemsList items={chatItems} isLoading={messagesLoading} />
          <div ref={sentinelRef} className="h-1" aria-hidden="true" />
        </div>

        {/* Scroll buttons */}
        <TooltipProvider>
          <div className="absolute bottom-3 right-3 z-10 flex flex-col gap-1">
            <div className={`transition-all duration-200 ${showScrollToTop ? 'opacity-100 scale-100' : 'opacity-0 scale-75 pointer-events-none'}`}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-7 w-7 rounded-full shadow-md cursor-pointer"
                    onClick={scrollToTop}
                    aria-label="Scroll to top"
                  >
                    <ChevronUp className="h-4 w-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="left">Scroll to top</TooltipContent>
              </Tooltip>
            </div>
            <div className={`transition-all duration-200 ${scrolledUp && chatItems.length > 0 ? 'opacity-100 scale-100' : 'opacity-0 scale-75 pointer-events-none'}`}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-7 w-7 rounded-full shadow-md cursor-pointer"
                    onClick={scrollToBottom}
                    aria-label="Scroll to bottom"
                  >
                    <ChevronDown className="h-4 w-4" />
                    {newEventCount > 0 && (
                      <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground px-1">
                        {newEventCount}
                      </span>
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="left">Scroll to bottom</TooltipContent>
              </Tooltip>
            </div>
          </div>
        </TooltipProvider>
      </div>

      {/* Input area */}
      <ChatInput
        sessionId={openSessionId}
        phase={sessionPhase}
        disabled={messagesLoading}
      />
    </aside>
  )
}
