'use client'

import {
  createContext,
  useContext,
  useState,
  useCallback,
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

export function ChatSidebarProvider({ children }: { children: ReactNode }) {
  const [openSessionId, setOpenSessionId] = useState<string | null>(null)

  const openSidebar = useCallback((sessionId: string) => {
    setOpenSessionId(sessionId)
  }, [])

  const closeSidebar = useCallback(() => {
    setOpenSessionId(null)
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
