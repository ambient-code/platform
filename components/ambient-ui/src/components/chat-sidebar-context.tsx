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

type ChatSidebarState = {
  openSessionId: string | null
  isOpen: boolean
  openSidebar: (sessionId: string) => void
  closeSidebar: () => void
}

const ChatSidebarContext = createContext<ChatSidebarState | null>(null)

function readChatParam(): string | null {
  if (typeof window === 'undefined') return null
  return new URL(window.location.href).searchParams.get('chat')
}

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
  const [openSessionId, setOpenSessionId] = useState<string | null>(readChatParam)

  const openSidebar = useCallback((sessionId: string) => {
    setOpenSessionId(sessionId)
    updateChatParam(sessionId)
  }, [])

  const closeSidebar = useCallback(() => {
    setOpenSessionId(null)
    updateChatParam(null)
  }, [])

  useEffect(() => {
    const handlePopState = () => setOpenSessionId(readChatParam())
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
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
