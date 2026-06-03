'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useSession } from '@/queries/use-sessions'
import {
  LayoutDashboard,
  ScrollText,
  FolderGit2,
  Settings,
  MessageSquare,
} from 'lucide-react'
import { SessionHeader } from './_components/session-header'
import { OverviewTab } from './_components/overview-tab'
import { LogsTab } from './_components/logs-tab'
import { ChatTab } from './_components/chat-tab'
import { ResourcesTab } from './_components/resources-tab'
import { ConfigTab } from './_components/config-tab'

export default function SessionDetailPage() {
  const { sessionId } = useParams<{ projectId: string; sessionId: string }>()
  const [activeTab, setActiveTab] = useState(() => {
    if (typeof window === 'undefined') return 'overview'
    return new URL(window.location.href).searchParams.get('tab') ?? 'overview'
  })
  const { data: session, isLoading, error } = useSession(sessionId)

  const handleTabChange = (value: string) => {
    setActiveTab(value)
    const url = new URL(window.location.href)
    url.searchParams.set('tab', value)
    window.history.replaceState({}, '', url.toString())
  }

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load session: {error.message}
      </p>
    )
  }

  if (isLoading || !session) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <SessionHeader session={session} />
      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList className="w-full *:flex-1">
          <TabsTrigger value="overview">
            <LayoutDashboard className="size-4 mr-1.5" /> Overview
          </TabsTrigger>
          <TabsTrigger value="logs">
            <ScrollText className="size-4 mr-1.5" /> Logs
          </TabsTrigger>
          <TabsTrigger value="resources">
            <FolderGit2 className="size-4 mr-1.5" /> Resources
          </TabsTrigger>
          <TabsTrigger value="config">
            <Settings className="size-4 mr-1.5" /> Config
          </TabsTrigger>
          <TabsTrigger value="chat">
            <MessageSquare className="size-4 mr-1.5" /> Chat
          </TabsTrigger>
        </TabsList>
        <TabsContent value="overview">
          <OverviewTab session={session} />
        </TabsContent>
        <TabsContent value="logs">
          <LogsTab session={session} />
        </TabsContent>
        <TabsContent value="resources">
          <ResourcesTab session={session} />
        </TabsContent>
        <TabsContent value="config">
          <ConfigTab session={session} />
        </TabsContent>
        <TabsContent value="chat">
          <ChatTab session={session} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
