'use client'

import { createContext, useContext, useState, useCallback, useEffect, useMemo, type ReactNode } from 'react'

export const MAX_SESSIONS = 8

export type SidebarSession = {
  sessionId: string
  sessionName?: string
  mode: 'chat' | 'test'
  agentId?: string
  agentName?: string
  agentPrompt?: string | null
  agentModel?: string | null
}

type OpenTestOpts = {
  sessionId: string
  sessionName?: string
  agentId: string
  agentName: string
  agentPrompt: string | null
  agentModel: string | null
}

type ChatSidebarState = {
  sessions: SidebarSession[]
  activeSessionId: string | null
  isOpen: boolean
  isCollapsed: boolean
  canAddSession: () => boolean
  openSidebar: (sessionId: string, sessionName?: string) => void
  openTestSidebar: (opts: OpenTestOpts) => void
  collapseSidebar: () => void
  closeAllSessions: () => void
  closeSession: (sessionId: string) => void
  switchSession: (sessionId: string) => void
}

const ChatSidebarContext = createContext<ChatSidebarState | null>(null)

function readChatParam(): string | null {
  if (typeof window === 'undefined') return null
  return new URL(window.location.href).searchParams.get('chat')
}

function updateChatParam(sessionId: string | null) {
  const url = new URL(window.location.href)
  if (sessionId) url.searchParams.set('chat', sessionId)
  else url.searchParams.delete('chat')
  window.history.replaceState({}, '', url.toString())
}

export function ChatSidebarProvider({ children }: { children: ReactNode }) {
  const [sessions, setSessions] = useState<SidebarSession[]>(() => {
    const id = readChatParam()
    return id ? [{ sessionId: id, mode: 'chat' as const }] : []
  })
  const [activeSessionId, setActiveSessionId] = useState<string | null>(readChatParam)
  const [isCollapsed, setIsCollapsed] = useState(false)

  const canAddSession = useCallback(() => sessions.length < MAX_SESSIONS, [sessions.length])

  const addIfAbsent = useCallback((entry: SidebarSession) => {
    setSessions(prev => {
      if (prev.some(s => s.sessionId === entry.sessionId) || prev.length >= MAX_SESSIONS) return prev
      return [...prev, entry]
    })
  }, [])

  const openSidebar = useCallback((sessionId: string, sessionName?: string) => {
    addIfAbsent({ sessionId, sessionName, mode: 'chat' })
    setActiveSessionId(sessionId)
    setIsCollapsed(false)
    updateChatParam(sessionId)
  }, [addIfAbsent])

  const openTestSidebar = useCallback((opts: OpenTestOpts) => {
    addIfAbsent({
      sessionId: opts.sessionId, sessionName: opts.sessionName, mode: 'test',
      agentId: opts.agentId, agentName: opts.agentName,
      agentPrompt: opts.agentPrompt, agentModel: opts.agentModel,
    })
    setActiveSessionId(opts.sessionId)
    setIsCollapsed(false)
    updateChatParam(opts.sessionId)
  }, [addIfAbsent])

  const closeSession = useCallback((sessionId: string) => {
    setSessions(prev => prev.filter(s => s.sessionId !== sessionId))
    setActiveSessionId(prev => {
      if (prev !== sessionId) return prev
      const idx = sessions.findIndex(s => s.sessionId === sessionId)
      const remaining = sessions.filter(s => s.sessionId !== sessionId)
      if (remaining.length === 0) { updateChatParam(null); return null }
      const nextId = remaining[Math.min(idx, remaining.length - 1)].sessionId
      updateChatParam(nextId)
      return nextId
    })
  }, [sessions])

  const collapseSidebar = useCallback(() => {
    setIsCollapsed(true)
    updateChatParam(null)
  }, [])

  const closeAllSessions = useCallback(() => {
    setSessions([])
    setActiveSessionId(null)
    setIsCollapsed(false)
    updateChatParam(null)
  }, [])

  const switchSession = useCallback((sessionId: string) => {
    setActiveSessionId(sessionId)
    updateChatParam(sessionId)
  }, [])

  useEffect(() => {
    const handlePopState = () => {
      const id = readChatParam()
      setActiveSessionId(id)
      if (id) {
        addIfAbsent({ sessionId: id, mode: 'chat' })
        setIsCollapsed(false)
      }
    }
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [addIfAbsent])

  const isOpen = sessions.length > 0 && !isCollapsed
  const value = useMemo<ChatSidebarState>(
    () => ({ sessions, activeSessionId, isOpen, isCollapsed, canAddSession, openSidebar, openTestSidebar, collapseSidebar, closeAllSessions, closeSession, switchSession }),
    [sessions, activeSessionId, isOpen, isCollapsed, canAddSession, openSidebar, openTestSidebar, collapseSidebar, closeAllSessions, closeSession, switchSession],
  )

  return <ChatSidebarContext.Provider value={value}>{children}</ChatSidebarContext.Provider>
}

export function useChatSidebar(): ChatSidebarState {
  const ctx = useContext(ChatSidebarContext)
  if (!ctx) throw new Error('useChatSidebar must be used within a ChatSidebarProvider')
  return ctx
}
