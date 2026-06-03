'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useTheme } from 'next-themes'
import {
  LayoutDashboard,
  Monitor,
  Bot,
  Moon,
  Sun,
} from 'lucide-react'
import { useSessions } from '@/queries/use-sessions'
import { getAttentionItems } from '@/app/(dashboard)/[projectId]/_components/dashboard-helpers'
import { ProjectSelector } from '@/components/project-selector'
import { Button } from '@/components/ui/button'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

type AppSidebarProps = {
  projectId: string | null
}

type NavItem = { readonly label: string; readonly icon: typeof Monitor; readonly href: string }

const operateNavItems: readonly NavItem[] = [
  { label: 'Dashboard', icon: LayoutDashboard, href: '' },
  { label: 'Sessions', icon: Monitor, href: 'sessions' },
]

const buildNavItems: readonly NavItem[] = [
  { label: 'Agents', icon: Bot, href: 'agents' },
]

function NavGroup({
  label,
  items,
  projectId,
  pathname,
  badgeCounts,
}: {
  label: string
  items: readonly NavItem[]
  projectId: string | null
  pathname: string
  badgeCounts?: Record<string, number>
}) {
  const isDisabled = !projectId

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{label}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map((item) => {
            const href = projectId
              ? item.href
                ? `/${projectId}/${item.href}`
                : `/${projectId}`
              : '#'
            const isActive = item.href
              ? pathname === href || pathname.startsWith(href + '/')
              : pathname === href
            const badgeCount = badgeCounts?.[item.label] ?? 0

            return (
              <SidebarMenuItem key={item.label}>
                <SidebarMenuButton
                  asChild={!isDisabled}
                  isActive={isActive}
                  disabled={isDisabled}
                  tooltip={item.label}
                >
                  {isDisabled ? (
                    <>
                      <item.icon />
                      <span>{item.label}</span>
                    </>
                  ) : (
                    <Link href={href}>
                      <item.icon />
                      <span>{item.label}</span>
                    </Link>
                  )}
                </SidebarMenuButton>
                {badgeCount > 0 && (
                  <SidebarMenuBadge>{badgeCount}</SidebarMenuBadge>
                )}
              </SidebarMenuItem>
            )
          })}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}

export function AppSidebar({ projectId }: AppSidebarProps) {
  const pathname = usePathname()
  const { theme, setTheme } = useTheme()
  const { data: sessionsData } = useSessions(projectId ?? '')

  const operateBadges = (() => {
    if (!sessionsData?.items) return undefined
    const count = getAttentionItems(sessionsData.items).length
    return count > 0 ? { Dashboard: count } : undefined
  })()

  return (
    <Sidebar>
      <SidebarHeader>
        <div className="flex items-center gap-2 px-2 py-1.5">
          <Bot className="size-5 text-primary" />
          <span className="text-sm font-semibold tracking-tight">Ambient</span>
        </div>
        <ProjectSelector projectId={projectId} />
      </SidebarHeader>

      <SidebarContent>
        <NavGroup label="Operate" items={operateNavItems} projectId={projectId} pathname={pathname} badgeCounts={operateBadges} />
        <NavGroup label="Build" items={buildNavItems} projectId={projectId} pathname={pathname} />
      </SidebarContent>

      <SidebarFooter>
        <div className="flex items-center justify-between px-2 py-1">
          <span className="text-xs text-muted-foreground">Theme</span>
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            aria-label="Toggle theme"
          >
            <Sun className="size-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
            <Moon className="absolute size-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
          </Button>
        </div>
        {process.env.NEXT_PUBLIC_GIT_COMMIT && process.env.NEXT_PUBLIC_GIT_COMMIT !== 'unknown' && (
          <div className="px-2 pb-1">
            <span className="text-[0.65rem] text-muted-foreground/60">
              {process.env.NEXT_PUBLIC_GIT_COMMIT.slice(0, 8)}
            </span>
          </div>
        )}
      </SidebarFooter>
    </Sidebar>
  )
}
