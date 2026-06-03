'use client'

import { useEffect, useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { Monitor, Bot } from 'lucide-react'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { useSessions } from '@/queries/use-sessions'
import { useAgents } from '@/queries/use-agents'

export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const router = useRouter()
  const params = useParams<{ projectId: string }>()
  const projectId = params.projectId ?? ''

  const { data: sessionsData } = useSessions(projectId)
  const { data: agentsData } = useAgents(projectId)

  const sessions = sessionsData?.items ?? []
  const agents = agentsData?.items ?? []

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setOpen((prev) => !prev)
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  function handleSelectSession(sessionId: string) {
    setOpen(false)
    router.push(`/${projectId}/sessions/${sessionId}`)
  }

  function handleSelectAgent(agentId: string) {
    setOpen(false)
    router.push(`/${projectId}/agents?selected=${agentId}`)
  }

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Search"
      description="Search across sessions and agents"
    >
      <CommandInput placeholder="Search sessions and agents..." />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        {sessions.length > 0 && (
          <CommandGroup heading="Sessions">
            {sessions.map((session) => (
              <CommandItem
                key={session.id}
                value={`session-${session.name}-${session.id}`}
                onSelect={() => handleSelectSession(session.id)}
              >
                <Monitor className="mr-2 size-4" />
                <div className="flex flex-col">
                  <span className="text-sm">{session.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {session.phase}
                    {session.agentName ? ` · ${session.agentName}` : ''}
                  </span>
                </div>
              </CommandItem>
            ))}
          </CommandGroup>
        )}

        {agents.length > 0 && (
          <CommandGroup heading="Agents">
            {agents.map((agent) => (
              <CommandItem
                key={agent.id}
                value={`agent-${agent.displayName ?? agent.name}-${agent.id}`}
                onSelect={() => handleSelectAgent(agent.id)}
              >
                <Bot className="mr-2 size-4" />
                <div className="flex flex-col">
                  <span className="text-sm">
                    {agent.displayName ?? agent.name}
                  </span>
                  {agent.displayName && (
                    <span className="text-xs text-muted-foreground">
                      {agent.name}
                    </span>
                  )}
                </div>
              </CommandItem>
            ))}
          </CommandGroup>
        )}
      </CommandList>
    </CommandDialog>
  )
}
