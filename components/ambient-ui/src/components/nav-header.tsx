'use client'

import Link from 'next/link'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { Separator } from '@/components/ui/separator'

type NavHeaderProps = {
  projectId?: string | null
  projectName?: string | null
  pageName?: string | null
  sessionName?: string | null
}

export function NavHeader({ projectId, projectName, pageName, sessionName }: NavHeaderProps) {
  return (
    <header className="sticky top-0 z-20 flex h-14 items-center gap-2 border-b bg-background px-4">
      <SidebarTrigger />
      <Separator orientation="vertical" className="mx-1 h-5" />

      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/">
                <span className="font-semibold">Ambient</span>
              </Link>
            </BreadcrumbLink>
          </BreadcrumbItem>

          {projectId && (
            <>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                <BreadcrumbLink asChild>
                  <Link href={`/${projectId}/fleet`}>{projectName ?? projectId}</Link>
                </BreadcrumbLink>
              </BreadcrumbItem>
            </>
          )}

          {pageName && (
            <>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {sessionName ? (
                  <BreadcrumbLink asChild>
                    <Link href={`/${projectId}/fleet`}>{pageName}</Link>
                  </BreadcrumbLink>
                ) : (
                  <BreadcrumbPage>{pageName}</BreadcrumbPage>
                )}
              </BreadcrumbItem>
            </>
          )}

          {sessionName && (
            <>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                <BreadcrumbPage>{sessionName}</BreadcrumbPage>
              </BreadcrumbItem>
            </>
          )}
        </BreadcrumbList>
      </Breadcrumb>
    </header>
  )
}
