'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useTheme } from 'next-themes'
import {
  Monitor,
  Bot,
  Calendar,
  AlertCircle,
  Settings,
  Key,
  Moon,
  Sun,
} from 'lucide-react'
import { ProjectSelector } from '@/components/project-selector'
import { Button } from '@/components/ui/button'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
} from '@/components/ui/sidebar'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

type AppSidebarProps = {
  projectId: string | null
}

const projectNavItems = [
  { label: 'Fleet', icon: Monitor, href: 'fleet', disabled: false },
  { label: 'Agents', icon: Bot, href: 'agents', disabled: true, tooltip: 'Coming soon' },
  { label: 'Schedules', icon: Calendar, href: 'schedules', disabled: true, tooltip: 'Coming soon' },
  { label: 'Issues', icon: AlertCircle, href: 'issues', disabled: true, tooltip: 'Coming soon' },
  { label: 'Settings', icon: Settings, href: 'settings', disabled: true, tooltip: 'Coming soon' },
] as const

const globalNavItems = [
  { label: 'Credentials', icon: Key, href: '/credentials', disabled: true, tooltip: 'Coming soon' },
] as const

export function AppSidebar({ projectId }: AppSidebarProps) {
  const pathname = usePathname()
  const { theme, setTheme } = useTheme()

  return (
    <Sidebar>
      <SidebarHeader>
        <ProjectSelector projectId={projectId} />
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Project</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {projectNavItems.map((item) => {
                const href = projectId ? `/${projectId}/${item.href}` : '#'
                const isActive = pathname === href
                const isDisabled = item.disabled || !projectId

                const menuButton = (
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

                if (isDisabled && 'tooltip' in item && item.tooltip) {
                  return (
                    <Tooltip key={item.label}>
                      <TooltipTrigger asChild>
                        {menuButton}
                      </TooltipTrigger>
                      <TooltipContent side="right">
                        {item.tooltip}
                      </TooltipContent>
                    </Tooltip>
                  )
                }

                return menuButton
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarSeparator />

        <SidebarGroup>
          <SidebarGroupLabel>Global</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {globalNavItems.map((item) => (
                <Tooltip key={item.label}>
                  <TooltipTrigger asChild>
                    <SidebarMenuItem>
                      <SidebarMenuButton disabled tooltip={item.label}>
                        <item.icon />
                        <span>{item.label}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  </TooltipTrigger>
                  <TooltipContent side="right">
                    {item.tooltip}
                  </TooltipContent>
                </Tooltip>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
          aria-label="Toggle theme"
        >
          <Sun className="size-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
          <Moon className="absolute size-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
        </Button>
      </SidebarFooter>
    </Sidebar>
  )
}
