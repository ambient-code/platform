'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import { ScrollText, FileCode } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAgent } from '@/queries/use-agents'
import { getAgentLifecycle } from '../_components/lifecycle-badge'
import { AgentHeader } from './_components/agent-header'
import { AgentManifestTab } from './_components/agent-manifest-tab'
import { AgentSessionsTab } from './_components/agent-sessions-tab'

export default function AgentDetailPage() {
  const { projectId, agentId } = useParams<{ projectId: string; agentId: string }>()
  const [activeTab, setActiveTab] = useState('manifest')
  const { data: agent, isLoading, error } = useAgent(projectId, agentId)

  useEffect(() => {
    const tab = new URL(window.location.href).searchParams.get('tab')
    if (tab) setActiveTab(tab)
  }, [])

  const handleTabChange = (value: string) => {
    setActiveTab(value)
    const url = new URL(window.location.href)
    url.searchParams.set('tab', value)
    window.history.replaceState({}, '', url.toString())
  }

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load agent: {error.message}
      </p>
    )
  }

  if (isLoading || !agent) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  const lifecycle = getAgentLifecycle(agent.annotations)

  return (
    <div className="space-y-6">
      <AgentHeader agent={agent} lifecycle={lifecycle} />
      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList className="w-full *:flex-1">
          <TabsTrigger value="manifest">
            <FileCode className="size-4 mr-1.5" /> Manifest
          </TabsTrigger>
          <TabsTrigger value="sessions">
            <ScrollText className="size-4 mr-1.5" /> Sessions
          </TabsTrigger>
        </TabsList>
        <TabsContent value="manifest">
          <AgentManifestTab agent={agent} lifecycle={lifecycle} />
        </TabsContent>
        <TabsContent value="sessions">
          <AgentSessionsTab agentId={agentId} projectId={projectId} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
