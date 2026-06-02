'use client'

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  useMemo,
  type ReactNode,
} from 'react'
import { useSearchParams, useRouter, usePathname } from 'next/navigation'

type ChatSidebarState = {
  openSessionId: string | null
  isOpen: boolean
  openSidebar: (sessionId: string) => void
  closeSidebar: () => void
}

const ChatSidebarContext = createContext<ChatSidebarState | null>(null)

function updateChatParam(sessionId: string | null) {
  const url = new URL(window.location.href)
  if (sessionId) {
    url.searchParams.set('chat', sessionId)
  } else {
    url.searchParams.delete('chat')
  }
  window.history.replaceState({}, '', url.toString())
}

export function ChatSidebarProvider({ children }: { children: ReactNode }) {
  const searchParams = useSearchParams()
  const initialChat = searchParams.get('chat')
  const [openSessionId, setOpenSessionId] = useState<string | null>(initialChat)

  const openSidebar = useCallback((sessionId: string) => {
    setOpenSessionId(sessionId)
    updateChatParam(sessionId)
  }, [])

  const closeSidebar = useCallback(() => {
    setOpenSessionId(null)
    updateChatParam(null)
  }, [])

  const value = useMemo<ChatSidebarState>(
    () => ({
      openSessionId,
      isOpen: openSessionId !== null,
      openSidebar,
      closeSidebar,
    }),
    [openSessionId, openSidebar, closeSidebar],
  )

  return (
    <ChatSidebarContext.Provider value={value}>
      {children}
    </ChatSidebarContext.Provider>
  )
}

export function useChatSidebar(): ChatSidebarState {
  const ctx = useContext(ChatSidebarContext)
  if (!ctx) {
    throw new Error('useChatSidebar must be used within a ChatSidebarProvider')
  }
  return ctx
}
