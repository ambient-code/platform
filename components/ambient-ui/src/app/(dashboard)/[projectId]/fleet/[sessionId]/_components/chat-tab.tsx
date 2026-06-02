'use client'

import { useMemo, useRef, useEffect, useState, useCallback } from 'react'
import { ExternalLink, ArrowDownLeft, Bot, ChevronUp, ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import type { DomainSession } from '@/domain/types'
import { useSessionMessages } from '@/queries/use-session-messages'
import {
  ChatItemsList,
  ChatInput,
  buildChatItems,
} from '@/components/chat-messages'
import { useChatSidebar } from '@/components/chat-sidebar-context'
import { useLiveTail, LiveIndicator } from './live-tail-indicator'

// ---- Main Chat Tab ----

export function ChatTab({ session }: { session: DomainSession }) {
  const { data, isLoading, error } = useSessionMessages(session.id)
  const { openSessionId, openSidebar, closeSidebar } = useChatSidebar()
  const isInSidebar = openSessionId === session.id

  const chatItems = useMemo(() => {
    return buildChatItems(data?.items ?? [])
  }, [data])

  const chatItemCount = chatItems.length

  const [showScrollToTop, setShowScrollToTop] = useState(false)
  const [scrolledUp, setScrolledUp] = useState(false)

  const { scrollRef, sentinelRef, isAtBottom, newEventCount, scrollToBottom } =
    useLiveTail(chatItemCount)

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
    if (!isLoading && chatItemCount > 0 && !hasScrolledOnLoad.current) {
      hasScrolledOnLoad.current = true
      requestAnimationFrame(() => {
        scrollToBottom()
      })
    }
  }, [isLoading, chatItemCount, scrollToBottom])

  if (error) {
    return (
      <div className="pt-4">
        <p className="text-sm text-destructive">
          Failed to load messages: {error.message}
        </p>
      </div>
    )
  }

  // When the chat is open in the sidebar, show a placeholder with bring-back option
  if (isInSidebar) {
    return (
      <div className="pt-4">
        <Card className="flex flex-col items-center justify-center py-16 p-0">
          <CardContent className="flex flex-col items-center gap-4 text-center p-6">
            <Bot className="h-10 w-10 text-muted-foreground opacity-40" aria-hidden="true" />
            <div>
              <p className="text-sm font-medium text-foreground">
                Chat is open in the sidebar
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                You can continue chatting from the sidebar while navigating other pages.
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => closeSidebar()}
              aria-label="Bring chat back to this tab"
            >
              <ArrowDownLeft className="h-4 w-4 mr-1.5" />
              Bring back
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="pt-4">
      <Card className="flex flex-col overflow-hidden p-0">
        {/* Header with pop-out button */}
        <div className="flex items-center justify-between border-b px-4 py-2">
          <div className="flex items-center gap-2">
            {isAtBottom && chatItemCount > 0 && <LiveIndicator />}
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => openSidebar(session.id)}
            aria-label="Pop out chat to sidebar"
            className="text-xs text-muted-foreground hover:text-foreground"
          >
            <ExternalLink className="h-3.5 w-3.5 mr-1.5" />
            Pop out
          </Button>
        </div>

        {/* Message area */}
        <CardContent className="flex-1 p-0 relative">
          <div
            ref={scrollRef}
            className="max-h-[600px] overflow-y-auto"
            role="log"
            aria-label="Chat messages"
            onScroll={handleScroll}
          >
            <ChatItemsList items={chatItems} isLoading={isLoading} />
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
              <div className={`transition-all duration-200 ${scrolledUp && chatItemCount > 0 ? 'opacity-100 scale-100' : 'opacity-0 scale-75 pointer-events-none'}`}>
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
