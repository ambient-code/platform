'use client'

import Link from 'next/link'
import { LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useCurrentUser } from '@/hooks/use-current-user'

type NavHeaderProps = {
  projectId?: string | null
  projectName?: string | null
  pageName?: string | null
  sessionName?: string | null
}

const BREADCRUMB_LABEL_MAP: Record<string, string> = {}

function displayLabel(raw: string): string {
  return BREADCRUMB_LABEL_MAP[raw] ?? raw
}

function UserMenu() {
  const { user, isLoading } = useCurrentUser()

  if (isLoading || !user) {
    return null
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 rounded-full text-xs font-medium"
          aria-label="User menu"
        >
          {user.initials}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className="font-normal">
          <div className="flex flex-col gap-1">
            <p className="text-sm font-medium leading-none">{user.name}</p>
            <p className="text-xs leading-none text-muted-foreground">{user.email}</p>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild>
          <a href="/api/auth/sso/logout" className="flex items-center gap-2 cursor-pointer">
            <LogOut className="h-4 w-4" />
            Sign out
          </a>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function NavHeader({ projectId, projectName, pageName, sessionName }: NavHeaderProps) {
  const mappedPageName = pageName ? displayLabel(pageName) : null

  return (
    <header className="sticky top-0 z-20 flex h-14 items-center gap-2 border-b bg-background px-4">
      <SidebarTrigger />
      <Separator orientation="vertical" className="mx-2 h-5" />

      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/">
                <span>Ambient</span>
              </Link>
            </BreadcrumbLink>
          </BreadcrumbItem>

          {projectId && (
            <>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                <BreadcrumbLink asChild>
                  <Link href={`/${projectId}/sessions`}>{projectName ?? projectId}</Link>
                </BreadcrumbLink>
              </BreadcrumbItem>
            </>
          )}

          {mappedPageName && (
            <>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {sessionName ? (
                  <BreadcrumbLink asChild>
                    <Link href={`/${projectId}/sessions`}>{mappedPageName}</Link>
                  </BreadcrumbLink>
                ) : (
                  <BreadcrumbPage>{mappedPageName}</BreadcrumbPage>
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

      <div className="ml-auto">
        <UserMenu />
      </div>
    </header>
  )
}
