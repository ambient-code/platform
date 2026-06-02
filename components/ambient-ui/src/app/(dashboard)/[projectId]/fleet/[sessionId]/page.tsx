'use client'

import { useParams } from 'next/navigation'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useSession } from '@/queries/use-sessions'
import { SessionHeader } from './_components/session-header'
import { PhaseTab } from './_components/phase-tab'
import { LogsTab } from './_components/logs-tab'

export default function SessionDetailPage() {
  const { sessionId } = useParams<{ projectId: string; sessionId: string }>()
  const { data: session, isLoading, error } = useSession(sessionId)

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
      <Tabs defaultValue="phase">
        <TabsList className="w-full *:flex-1">
          <TabsTrigger value="phase">Phase</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
          <TabsTrigger value="resources" disabled>Resources</TabsTrigger>
          <TabsTrigger value="details" disabled>Details</TabsTrigger>
          <TabsTrigger value="chat" disabled>Chat</TabsTrigger>
        </TabsList>
        <TabsContent value="phase">
          <PhaseTab session={session} />
        </TabsContent>
        <TabsContent value="logs">
          <LogsTab session={session} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
