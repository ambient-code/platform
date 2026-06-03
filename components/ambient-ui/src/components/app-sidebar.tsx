'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useTheme } from 'next-themes'
import {
  Monitor,
  Bot,
  Moon,
  Sun,
} from 'lucide-react'
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
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

type AppSidebarProps = {
  projectId: string | null
}

const projectNavItems = [
  { label: 'Sessions', icon: Monitor, href: 'sessions' },
] as const

export function AppSidebar({ projectId }: AppSidebarProps) {
  const pathname = usePathname()
  const { theme, setTheme } = useTheme()

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
        <SidebarGroup>
          <SidebarGroupLabel>Project</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {projectNavItems.map((item) => {
                const href = projectId ? `/${projectId}/${item.href}` : '#'
                const isActive = pathname === href || pathname.startsWith(href + '/')
                const isDisabled = !projectId

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
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
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
      </SidebarFooter>
    </Sidebar>
  )
}
