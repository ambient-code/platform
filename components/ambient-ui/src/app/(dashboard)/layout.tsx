'use client'

import { usePathname } from 'next/navigation'
import { AppSidebar } from '@/components/app-sidebar'
import { NavHeader } from '@/components/nav-header'
import { StatusBar } from '@/components/status-bar'
import { useProject } from '@/queries/use-projects'
import {
  SidebarInset,
  SidebarProvider,
} from '@/components/ui/sidebar'

function extractNavContext(pathname: string) {
  const segments = pathname.split('/').filter(Boolean)
  const projectId = segments.length >= 1 ? segments[0] : null
  const pageName = segments.length >= 2 ? capitalize(segments[1]) : null
  return { projectId, pageName }
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
  const { projectId, pageName } = extractNavContext(pathname)
  const { data: project } = useProject(projectId ?? '')

  return (
    <SidebarProvider>
      <AppSidebar projectId={projectId} />
      <SidebarInset>
        <NavHeader
          projectId={projectId}
          projectName={project?.name ?? null}
          pageName={pageName}
        />
        <div className="flex-1 p-6 pb-10">{children}</div>
        <StatusBar />
      </SidebarInset>
    </SidebarProvider>
  )
}
