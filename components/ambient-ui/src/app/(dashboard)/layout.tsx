'use client'

import { usePathname } from 'next/navigation'
import { AppSidebar } from '@/components/app-sidebar'
import { NavHeader } from '@/components/nav-header'
import { StatusBar } from '@/components/status-bar'
import { ChatSidebar } from '@/components/chat-sidebar'
import { ChatSidebarProvider } from '@/components/chat-sidebar-context'
import { useProject } from '@/queries/use-projects'
import { useSession } from '@/queries/use-sessions'
import {
  SidebarInset,
  SidebarProvider,
} from '@/components/ui/sidebar'

function extractNavContext(pathname: string) {
  const segments = pathname.split('/').filter(Boolean)
  const projectId = segments.length >= 1 ? segments[0] : null
  const pageName = segments.length >= 2 ? capitalize(segments[1]) : null
  const sessionId = segments.length >= 3 && segments[1] === 'sessions' ? segments[2] : null
  return { projectId, pageName, sessionId }
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
  const { projectId, pageName, sessionId } = extractNavContext(pathname)
  const { data: project } = useProject(projectId ?? '')
  const { data: session } = useSession(sessionId ?? '', undefined)

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
          />
          <div className="flex-1 p-6 pb-10">{children}</div>
          <StatusBar />
        </SidebarInset>
        <ChatSidebar />
      </SidebarProvider>
    </ChatSidebarProvider>
  )
}
