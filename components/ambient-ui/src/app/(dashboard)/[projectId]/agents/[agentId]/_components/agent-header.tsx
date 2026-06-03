'use client'

import { useState, useCallback } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { Play, Download, Trash2, MoreVertical } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import type { DomainAgent } from '@/domain/types'
import { LifecycleBadge, getAgentLifecycle } from '../../_components/lifecycle-badge'
import type { AgentLifecycle } from '../../_components/lifecycle-badge'
import { useDeleteAgent } from '@/queries/use-agents'
import { formatRelativeTime } from '@/lib/format-timestamp'

function agentToYaml(agent: DomainAgent): string {
  const lines: string[] = [
    'apiVersion: ambient-code.io/v1',
    'kind: Agent',
    'metadata:',
    `  name: ${agent.name}`,
  ]

  const annotationEntries = Object.entries(agent.annotations)
  if (annotationEntries.length > 0) {
    lines.push('  annotations:')
    for (const [key, value] of annotationEntries) {
      lines.push(`    ${key}: "${value}"`)
    }
  }

  lines.push('spec:')
  if (agent.displayName) lines.push(`  displayName: "${agent.displayName}"`)
  if (agent.description) lines.push(`  description: "${agent.description}"`)
  if (agent.model) lines.push(`  model: ${agent.model}`)
  if (agent.repoUrl) lines.push(`  repoUrl: ${agent.repoUrl}`)
  if (agent.prompt) {
    lines.push('  prompt: |')
    for (const promptLine of agent.prompt.split('\n')) {
      lines.push(`    ${promptLine}`)
    }
  }

  return lines.join('\n') + '\n'
}

export function AgentHeader({
  agent,
  lifecycle,
}: {
  agent: DomainAgent
  lifecycle: AgentLifecycle
}) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const router = useRouter()
  const { projectId } = useParams<{ projectId: string }>()
  const deleteAgent = useDeleteAgent()

  const handleConfirmDelete = useCallback(() => {
    deleteAgent.mutate({ projectId, agentId: agent.id }, {
      onSuccess: () => {
        setDeleteDialogOpen(false)
        router.push(`/${projectId}/agents`)
      },
      onError: () => setDeleteDialogOpen(false),
    })
  }, [deleteAgent, agent.id, router, projectId])

  const handleExportYaml = useCallback(() => {
    const yaml = agentToYaml(agent)
    const blob = new Blob([yaml], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `agent-${agent.name}.yaml`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }, [agent])

  const handleRunTestSession = useCallback(() => {
    router.push(`/${projectId}/sessions?create=true&agentId=${agent.id}`)
  }, [router, projectId, agent.id])

  return (
    <>
      <div className="sticky top-14 z-[5] bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 pb-4 -mx-1 px-1">
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <h1 className="text-lg font-semibold">
                {agent.displayName ?? agent.name}
              </h1>
              <LifecycleBadge lifecycle={lifecycle} />
            </div>

            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleRunTestSession}
                aria-label="Run test session"
              >
                <Play className="size-4" />
                Run Test Session
              </Button>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="size-8" aria-label="More actions">
                    <MoreVertical className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={handleExportYaml}>
                    <Download className="size-4 mr-2" />
                    Export YAML
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => setDeleteDialogOpen(true)}
                    disabled={deleteAgent.isPending}
                    className="text-destructive focus:text-destructive"
                  >
                    <Trash2 className="size-4 mr-2" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>

          <div className="flex items-center gap-6 text-sm text-muted-foreground">
            {agent.displayName && agent.name !== agent.displayName && (
              <MetaItem label="ID" value={agent.name} />
            )}
            {agent.model && <MetaItem label="Model" value={agent.model} />}
            {agent.ownerUserId && <MetaItem label="Owner" value={agent.ownerUserId} />}
            <MetaItem label="Updated" value={formatRelativeTime(agent.updatedAt)} />
          </div>
        </div>
      </div>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this agent?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. The agent definition will be permanently deleted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              Delete agent
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

function MetaItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="text-muted-foreground/70">{label}:</span>{' '}
      <span>{value}</span>
    </div>
  )
}
