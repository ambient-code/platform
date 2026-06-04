'use client'

import { useEffect } from 'react'
import { usePathname } from 'next/navigation'
import { AppSidebar } from '@/components/app-sidebar'
import { NavHeader } from '@/components/nav-header'
import { StatusBar } from '@/components/status-bar'
import { ChatSidebar } from '@/components/chat-sidebar'
import { ChatSidebarProvider } from '@/components/chat-sidebar-context'
import { CommandPalette } from '@/components/command-palette'
import { useProject } from '@/queries/use-projects'
import { useSession } from '@/queries/use-sessions'
import { useAgent } from '@/queries/use-agents'
import {
  SidebarInset,
  SidebarProvider,
} from '@/components/ui/sidebar'
import { useRecentVisits } from '@/hooks/use-recent-visits'

const GLOBAL_ROUTES = new Set(['credentials', 'settings'])

function extractNavContext(pathname: string) {
  const segments = pathname.split('/').filter(Boolean)
  const firstSegment = segments.length >= 1 ? segments[0] : null
  const isGlobalRoute = firstSegment !== null && GLOBAL_ROUTES.has(firstSegment)
  const projectId = isGlobalRoute ? null : firstSegment
  const pageName = isGlobalRoute
    ? capitalize(firstSegment)
    : segments.length >= 2 ? capitalize(segments[1]) : null
  const sessionId = segments.length >= 3 && segments[1] === 'sessions' ? segments[2] : null
  const agentId = segments.length >= 3 && segments[1] === 'agents' ? segments[2] : null
  return { projectId, pageName, sessionId, agentId }
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const pathname = usePathname()
  const { projectId, pageName, sessionId, agentId } = extractNavContext(pathname)
  const { data: project } = useProject(projectId ?? '')
  const { data: session } = useSession(sessionId ?? '', undefined)
  const { data: agent } = useAgent(projectId ?? '', agentId ?? '')
  const { recordVisit } = useRecentVisits()

  useEffect(() => {
    if (sessionId && session && projectId) {
      recordVisit({
        type: 'session',
        id: sessionId,
        projectId,
        label: session.name,
        sublabel: [session.phase, session.agentName].filter(Boolean).join(' · ') || null,
        href: pathname,
      })
    } else if (agentId && agent && projectId) {
      recordVisit({
        type: 'agent',
        id: agentId,
        projectId,
        label: agent.displayName ?? agent.name,
        sublabel: agent.displayName ? agent.name : null,
        href: pathname,
      })
    } else if (pageName === 'Credentials') {
      recordVisit({
        type: 'credential',
        id: 'credentials',
        projectId: null,
        label: 'Credentials',
        sublabel: null,
        href: pathname,
      })
    }
  }, [pathname, sessionId, session, agentId, agent, projectId, pageName, recordVisit])

  return (
    <ChatSidebarProvider>
      <SidebarProvider>
        <AppSidebar projectId={projectId} />
        <SidebarInset className="min-w-0 flex-1 overflow-x-clip">
          <NavHeader
            projectId={projectId}
            projectName={project?.name ?? null}
            pageName={pageName}
            sessionName={sessionId ? (session?.name ?? sessionId) : null}
            detailName={agentId ? (agent?.displayName ?? agent?.name ?? agentId) : null}
          />
          <div className="flex-1 p-6 pb-10 max-w-7xl">{children}</div>
          <StatusBar />
          <CommandPalette />
        </SidebarInset>
        <ChatSidebar />
      </SidebarProvider>
    </ChatSidebarProvider>
  )
}
