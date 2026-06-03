'use client'

import { useState, useEffect, useCallback } from 'react'
import { useParams } from 'next/navigation'
import { History, FileCode } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle,
} from '@/components/ui/resizable'
import { useAgent } from '@/queries/use-agents'
import { getAgentLifecycle } from '../_components/lifecycle-badge'
import { AgentHeader } from './_components/agent-header'
import { AgentManifestTab } from './_components/agent-manifest-tab'
import { AgentSessionsTab } from './_components/agent-sessions-tab'
import { TestSessionPane } from './_components/test-session-pane'
import type { TestHistoryEntry } from './_components/test-session-pane'

const MAX_HISTORY = 5

export default function AgentDetailPage() {
  const { projectId, agentId } = useParams<{ projectId: string; agentId: string }>()
  const [activeTab, setActiveTab] = useState('manifest')
  const { data: agent, isLoading, error } = useAgent(projectId, agentId)

  const [testSessionId, setTestSessionId] = useState<string | null>(null)
  const [testSessionName, setTestSessionName] = useState<string>('')
  const [testHistory, setTestHistory] = useState<TestHistoryEntry[]>([])

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

  const handleRunTest = useCallback((sessionId: string, name: string) => {
    // Push current test session into history (if one exists)
    if (testSessionId && testSessionName) {
      setTestHistory((prev) => {
        const entry: TestHistoryEntry = {
          id: testSessionId,
          name: testSessionName,
          phase: 'Stopped', // Previous session will be stopped
          createdAt: new Date().toISOString(),
        }
        return [entry, ...prev].slice(0, MAX_HISTORY)
      })
    }

    setTestSessionId(sessionId)
    setTestSessionName(name)
  }, [testSessionId, testSessionName])

  const handleCloseTest = useCallback(() => {
    setTestSessionId(null)
    setTestSessionName('')
  }, [])

  const handleSelectHistory = useCallback((entry: TestHistoryEntry) => {
    // Push current active session to history if there is one
    if (testSessionId && testSessionName) {
      setTestHistory((prev) => {
        const current: TestHistoryEntry = {
          id: testSessionId,
          name: testSessionName,
          phase: 'Stopped',
          createdAt: new Date().toISOString(),
        }
        // Remove the selected entry from history and add the current one
        const filtered = prev.filter((h) => h.id !== entry.id)
        return [current, ...filtered].slice(0, MAX_HISTORY)
      })
    } else {
      // Just remove the selected entry from history
      setTestHistory((prev) => prev.filter((h) => h.id !== entry.id))
    }

    setTestSessionId(entry.id)
    setTestSessionName(entry.name)
  }, [testSessionId, testSessionName])

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
  const hasTestSession = testSessionId !== null

  return (
    <ResizablePanelGroup orientation="horizontal" className={hasTestSession ? 'h-screen !overflow-hidden sticky top-0' : ''}>
      <ResizablePanel defaultSize={hasTestSession ? 55 : 100} minSize={30}>
        <div className={`space-y-6 pr-1 ${hasTestSession ? 'h-full overflow-y-auto' : ''}`}>
          <AgentHeader agent={agent} lifecycle={lifecycle} onRunTest={handleRunTest} />
          <Tabs value={activeTab} onValueChange={handleTabChange}>
            <TabsList className="w-full *:flex-1">
              <TabsTrigger value="manifest">
                <FileCode className="size-4 mr-1.5" /> Manifest
              </TabsTrigger>
              <TabsTrigger value="sessions">
                <History className="size-4 mr-1.5" /> Run History
              </TabsTrigger>
            </TabsList>
            <TabsContent value="manifest">
              <AgentManifestTab agent={agent} lifecycle={lifecycle} />
            </TabsContent>
            <TabsContent value="sessions">
              <AgentSessionsTab
                agentId={agentId}
                projectId={projectId}
                onSelectSession={handleRunTest}
              />
            </TabsContent>
          </Tabs>
        </div>
      </ResizablePanel>

      {hasTestSession && (
        <>
          <ResizableHandle withHandle />
          <ResizablePanel defaultSize={45} minSize={25}>
            <TestSessionPane
              sessionId={testSessionId}
              sessionName={testSessionName}
              projectId={projectId}
              agentId={agentId}
              agentName={agent.name}
              agentPrompt={agent.prompt}
              agentModel={agent.model}
              history={testHistory}
              onClose={handleCloseTest}
              onRunTest={handleRunTest}
              onSelectHistory={handleSelectHistory}
            />
          </ResizablePanel>
        </>
      )}
    </ResizablePanelGroup>
  )
}
